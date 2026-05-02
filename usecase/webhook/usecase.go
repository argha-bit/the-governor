package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"the_governor/constants"
	"the_governor/models"
	"the_governor/repository/historyrepository"
	"the_governor/repository/servicerepository"
	"the_governor/request"
	"the_governor/usecase"
	"the_governor/usecase/githubutility"
	"the_governor/usecase/translator"
	"the_governor/utils"
	"time"

	"github.com/go-sql-driver/mysql"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type WebhookUsecase struct {
	registerServiceRepo servicerepository.ServiceRepository
	historyServiceRepo  historyrepository.HistoryRepository
	ghAppClient         usecase.GithubAppClient
}

func NewWebhookUsecaseHandler(repo servicerepository.ServiceRepository, historyRepo historyrepository.HistoryRepository, ghAppClient usecase.GithubAppClient) usecase.WebhookUsecaseHandler {
	return &WebhookUsecase{
		registerServiceRepo: repo,
		historyServiceRepo:  historyRepo,
		ghAppClient:         ghAppClient,
	}
}

func (u *WebhookUsecase) HandleWebhook(request *request.WebhookRequest) error {
	ctx := context.Background()
	// Handle the webhook request and perform necessary actions based on the request type and payload
	serviceDetails, err := u.registerServiceRepo.GetByOwnerRepo(ctx, request.Owner, request.Repository)
	if err != nil {
		log.Println("Error fetching service details", err.Error())
		return err
	}
	if serviceDetails == nil {
		log.Printf("No registered service found for %s/%s", request.Owner, request.Repository)
		return fmt.Errorf("no registered service found for %s/%s", request.Owner, request.Repository)
	}
	now := time.Now()
	history := &models.ConfigFetchHistory{
		ServiceID:    serviceDetails.ID,
		Owner:        serviceDetails.Owner,
		Repository:   serviceDetails.Repository,
		CommitSHA:    request.CommitSHA,
		Branch:       request.Branch,
		FilesFetched: models.JSONFetchedFiles{},
		FetchedAt:    &now,
		Status:       "pending",
	}
	if err := u.historyServiceRepo.Create(ctx, history); err != nil {
		log.Println("Error creating history record", err.Error())
		return fmt.Errorf("error creating history record: %w", err)
	}
	//create client
	var fetchedFiles models.JSONFetchedFiles
	for _, filePath := range serviceDetails.ConfigPaths {
		file, err := githubutility.NewGitHubUtility(u.ghAppClient, serviceDetails.InstallationID).ReadFileContent(ctx, serviceDetails.Owner, serviceDetails.Repository, filePath)
		if err != nil {
			log.Printf("[Async] WARNING: Failed to fetch %s: %v", filePath, err)
			// Continue with other files, don't fail entire fetch
			fetchedFiles = append(fetchedFiles, models.FetchedFile{
				Path:       filePath,
				Content:    "",
				ConfigType: "error",
			})
			continue
		}
		fetchedFiles = append(fetchedFiles, models.FetchedFile{
			Path:       filePath,
			Content:    string(file),
			ConfigType: "configType",
		})
	}
	if err := u.historyServiceRepo.UpdateFilesFetched(ctx, history.ID, fetchedFiles); err != nil {
		log.Printf("[Async] ERROR: Failed to update files_fetched: %v", err)
	}
	if err := u.historyServiceRepo.UpdateStatus(ctx, history.ID, "success", ""); err != nil {
		log.Printf("[Async] ERROR: Failed to update status: %v", err)
	}

	for _, files := range fetchedFiles {
		log.Printf("Fetched file: %s, content length: %d", files.Path, files.Content)
	}
	return nil
}

func (u *WebhookUsecase) HandleWebhookV2(request *request.WebhookRequestV2) error {
	//Verify if Service is registered
	ctx := context.Background()
	serviceDetails, err := u.registerServiceRepo.GetByID(ctx, request.ServiceID)
	if err != nil {
		log.Println("Error fetching service details", err.Error())
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			log.Printf("MySQL error code: %d, message: %s", mysqlErr.Number, mysqlErr.Message)
			if mysqlErr.Number == 1062 {
				return fmt.Errorf("service with ID %s not found", request.ServiceID)
			}
			return fmt.Errorf("failed to fetch service for %s", request.ServiceID)
		}
	}
	if serviceDetails == nil {
		log.Printf("No registered service found for ID %s", request.ServiceID)
		return fmt.Errorf("no registered service found for ID %s", request.ServiceID)
	}
	go processWebhookAsync(serviceDetails)
	return nil

}
func processWebhookAsync(serviceDetails *models.RegisterServiceV2) {
	log.Printf("Processing webhook for service: %s", serviceDetails.ServiceName)
	ctx := context.Background()

	code, config, err := utils.MakeAPICall(http.MethodGet, serviceDetails.ConfigEndpoint, map[string]string{}, nil)
	if err != nil {
		log.Println("Error in Processing routing", err.Error(), serviceDetails.ServiceID)
		return
	}
	if code != http.StatusOK {
		log.Printf("endpoint returned %d", code)
		return
	}

	var routeConfig constants.Route
	if err := json.Unmarshal(config, &routeConfig); err != nil {
		log.Println("error Unmarshalling routing request, please follow README.md", err.Error())
		return
	}
	printJson(routeConfig)

	k8sConfig, err := utils.GetK8sClient()
	if err != nil {
		log.Println("aborting: could not get k8s config")
		return
	}

	//creating an empty scheme registry for the runtime
	scheme := runtime.NewScheme()
	//register standard types to scheme
	_ = clientgoscheme.AddToScheme(scheme)
	//install gateway api to scheme
	_ = gatewayv1.Install(scheme)

	k8sClient, err := client.New(k8sConfig, client.Options{Scheme: scheme})
	if err != nil {
		log.Println("aborting: could not create k8s client", err)
		return
	}

	gatewayTranslator := translator.NewTranslatorFromEnv("my-namespace")

	objects, err := gatewayTranslator.TranslateAll(ctx, routeConfig.Routes)
	if err != nil {
		log.Printf("ERROR translating routes: %v", err)
		return
	}
	for _, obj := range objects {
		if err := utils.ApplyObject(ctx, k8sClient, obj); err != nil {
			log.Printf("ERROR applying %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
		}
	}
}
func printJson(data any) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		return
	}
	log.Println("routing config is ", string(jsonData))
}
