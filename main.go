package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"the_governor/controller/governorroutecontroller"
	"the_governor/server"
	"the_governor/usecase"
	cf "the_governor/usecase/configprocessor"
	"the_governor/usecase/translator"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	governorv1alpha1 "the_governor/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
)

func readContent(ghUtility usecase.GithubUtility, ctx context.Context, wg *sync.WaitGroup, owner, repoName, path string) {

	if isRepoAvailableForOwner(ghUtility, ctx, owner, repoName) {
		content, err := ghUtility.ReadFileContent(ctx, owner, repoName, path)
		if err != nil {
			fmt.Printf("Error reading file content: %v\n", err)
			return
		}
		fmt.Printf("File content:\n%s\n", content)
	}
	wg.Done()
}

func isRepoAvailableForOwner(ghUtility usecase.GithubUtility, ctx context.Context, owner, repoName string) bool {
	_, err := ghUtility.GetRepository(ctx, owner, repoName)
	if err != nil {
		fmt.Printf("Repository %s/%s not found or error: %v\n", owner, repoName, err)
		return false
	}
	return true
}

func init() {
	// Load configuration from config.json for docker images read from /code/config.json
	configData, err := os.ReadFile("/code/config.json")
	if err != nil {
		log.Printf("Error reading config file: %v\n", err)
		return
	}
	//set env  variable for configs by looping and parsing JSON data
	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		log.Printf("Error parsing config file: %v\n", err)
		return
	}
	for key, value := range config {
		switch v := value.(type) {
		case string:
			os.Setenv(key, v)
		case float64:
			os.Setenv(key, strconv.Itoa(int(v)))
		case int:
			os.Setenv(key, strconv.Itoa(v))
		}
	}

}

func startPluginHandler() {
	configProcessor := cf.NewConfigProcessorPluginUsecaseHandler(translator.NewTranslatorFromEnv("my-namespace"))

	err := configProcessor.ReadConfig("governor-config.yaml")
	if err != nil {
		log.Printf("Error processing config: %v", err)
	}
	log.Println("Processing Completing exiting now")
}
func startOperator() {
	// 1. Setup scheme
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = governorv1alpha1.AddToScheme(scheme)
	_ = gatewayv1.Install(scheme)

	// 2. Create manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Fatalf("Unable to start manager: %v", err)
	}

	// 3. Select translator
	gatewayTranslator := translator.NewTranslatorFromEnv("my-namespace")

	// 4. Register controller with manager
	if err = governorroutecontroller.NewGovernorRouteController(
		mgr.GetClient(),
		mgr.GetScheme(),
		gatewayTranslator,
	).SetupWithManager(mgr); err != nil {
		log.Fatalf("Unable to create GovernorRoute controller: %v", err)
	}

	// 5. Start manager — blocks until process is killed
	log.Println("Governor Operator started, watching for GovernorRoute resources")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalf("Manager exited with error: %v", err)
	}
}

func main() {
	var engineMode, pluginServiceId, serviceMode string
	flag.StringVar(&engineMode, "engine", "", "Governor Engine Mode")
	flag.StringVar(&pluginServiceId, "service_id", "", "Plugin Service Id")
	flag.StringVar(&serviceMode, "service", "", "service mode for ARGO_PLUGIN")
	log.Println("=== The Governor - Decentralized Gateway Configuration Manager ===")
	flag.Parse()
	//TODO: Flag Based Start mode
	switch engineMode {
	case "ARGO_PLUGIN":
		switch serviceMode {
		case "generator":
			log.Println("starting route generator")
			startPluginHandler()
		case "plugin-server":
			log.Println("starting up plugin-server")
			server.StartServer(engineMode)
		default:
			log.Println("Service is required to trigger ", engineMode)
			return
		}
	case "WEB_SERVER":
		server.StartServer(engineMode)
	case "OPERATOR":
		startOperator()
	default:
		log.Println("NO VALID ENGINE MODE FOUND! AVAILABLE ENGINE MODES ARE: ARGO_PLUGIN,WEB_SERVER")
	}
}
