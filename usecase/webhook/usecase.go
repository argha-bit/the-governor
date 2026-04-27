package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
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
	//fetch config files by making API Call to endpoint
	log.Printf("Processing webhook for service: %s", serviceDetails.ServiceName)
	log.Printf("Service details: %+v", serviceDetails)
	var err error
	//create an array to keep track of processed routes and their status
	//translate the config files
	//print the config files
	//send final array file to the webhook endpoint of the service
	code, config, err := utils.MakeAPICall(http.MethodGet, serviceDetails.ConfigEndpoint, map[string]string{}, nil)

	if err != nil {
		log.Println("Error in Processing routing", err.Error(), serviceDetails.ServiceID)
		return
	}
	if code != http.StatusOK {
		log.Println("endpoint returned %d", code)
		return
	}
	var routeConfig constants.Route
	err = json.Unmarshal(config, &routeConfig)
	if err != nil {
		log.Println("error Unmarshalling routing request, please follow README.md", err.Error())
	}
	printJson(routeConfig)
	k8sConfig, err := utils.GetK8sClient()
	if err != nil {
		log.Println("aborting")
		return
	}
	gateWayType := os.Getenv("GATEWAY_PROVIDER")
	var routeTranslator usecase.RouteTranslator
	if gateWayType == "" {
		log.Println("selecting Base translator")
		routeTranslator = translator.NewBaseRouteTranslator("my-namespace")
	} else {
		log.Println("Implementation Not Created yet")
	}
	clientSet, err := gatewayclient.NewForConfig(k8sConfig)
	if err != nil {
		log.Println("Unable to create route")
		return
	}
	for _, routeDefn := range routeConfig.Routes {
		resp, backendObjects, err := routeTranslator.TranslateHTTPRoute(context.Background(), routeDefn)
		if err != nil {
			log.Printf("ERROR translating route %s: %v", routeDefn.RouteName, err)
			continue
		}
		for _, backendRef := range backendObjects {
			log.Printf("Creating backend object: %s/%s", backendRef.GetNamespace(), backendRef.GetName())
			//create the backend object in Kubernetes
			utils.CreateExternalK8sService(backendRef.(*corev1.Service), backendRef.GetNamespace(), k8sConfig)
		}
		//handle backend objects if any
		route, err := clientSet.GatewayV1().HTTPRoutes("my-namespace").Get(context.Background(), resp.Name, v1.GetOptions{})
		if err != nil {
			route, err = clientSet.GatewayV1().HTTPRoutes("my-namespace").Create(context.Background(), resp, v1.CreateOptions{})
		}
		log.Println("%+v", route)
		resp.ResourceVersion = route.ResourceVersion
		route, err = clientSet.GatewayV1().HTTPRoutes("my-namespace").Update(context.Background(), resp, v1.UpdateOptions{})

		if err != nil {
			log.Printf("ERROR translating route %s: %v", route.Name, err)
			continue
		}
		log.Println("Route Created Successfully", route.Name)
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
