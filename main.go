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
	"the_governor/server"
	"the_governor/usecase"
	cf "the_governor/usecase/configprocessor"
	"the_governor/usecase/translator"
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
	// Load configuration from config.json
	configData, err := os.ReadFile("/code/config.json")
	if err != nil {
		log.Println("Error reading config file: %v\n", err)
		return
	}
	//set env  variable for configs by looping and parsing JSON data
	var config map[string]interface{}
	if err := json.Unmarshal(configData, &config); err != nil {
		log.Println("Error parsing config file: %v\n", err)
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
	gateWayType := os.Getenv("GATEWAY_PROVIDER")
	var routeTranslator usecase.RouteTranslator
	if gateWayType == "" {
		log.Println("selecting Base translator")
		routeTranslator = translator.NewBaseRouteTranslator("my-namespace")
	} else {
		log.Println("Implementation Not Created yet")
	}
	configProcessor := cf.NewConfigProcessorPluginUsecaseHandler(routeTranslator)

	err := configProcessor.ReadConfig("governor-config.yaml")
	if err != nil {
		log.Println("Error processing config: %v", err)
	}
	log.Println("Processing Completing exiting now")
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
	default:
		log.Println("NO VALID ENGINE MODE FOUND! AVAILABLE ENGINE MODES ARE: ARGO_PLUGIN,WEB_SERVER")
	}
}
