package configprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"the_governor/constants"
	"the_governor/models"
	"the_governor/usecase"
	"the_governor/utils"

	"go.yaml.in/yaml/v2"
	k8syaml "sigs.k8s.io/yaml"
)

type ConfigProcessorWebhookUsecaseHandler struct {
}
type ConfigProcessorPluginUsecaseHandler struct {
	translatorUc usecase.GatewayTranslator
}

func NewConfigProcessorPluginUsecaseHandler(translatorUc usecase.GatewayTranslator) usecase.ConfigProcessorPluginUsecaseHandler {
	return &ConfigProcessorPluginUsecaseHandler{
		translatorUc: translatorUc,
	}
}

// func (c ConfigProcessorWebhookUsecaseHandler) ReadConfig(serviceDetails *models.RegisterServiceV2) error {
// 	//fetch config files by making API Call to endpoint
// 	log.Printf("Processing webhook for service: %s", serviceDetails.ServiceName)
// 	log.Printf("Service details: %+v", serviceDetails)
// 	var err error
// 	//create an array to keep track of processed routes and their status
// 	//translate the config files
// 	//print the config files
// 	//send final array file to the webhook endpoint of the service
// 	code, config, err := utils.MakeAPICall(http.MethodGet, serviceDetails.ConfigEndpoint, map[string]string{}, nil)

// 	if err != nil {
// 		log.Println("Error in Processing routing", err.Error(), serviceDetails.ServiceID)
// 		return err
// 	}
// 	if code != http.StatusOK {
// 		log.Println("endpoint returned %d", code)
// 		return fmt.Errorf("endpoint returned %d", code)
// 	}
// 	var routeConfig constants.Route
// 	err = json.Unmarshal(config, &routeConfig)
// 	if err != nil {
// 		log.Println("error Unmarshalling routing request, please follow README.md", err.Error())
// 	}
// 	printJson(routeConfig)
// 	if !reflect.DeepEqual(routeConfig, constants.Route{}) {
// 		return routeProcessor(routeConfig)
// 	} else {
// 		log.Println("Unable to process Config")
// 		return nil
// 	}
// }
// func printJson(data any) {
// 	jsonData, err := json.MarshalIndent(data, "", "  ")
// 	if err != nil {
// 		log.Printf("Error marshalling JSON: %v", err)
// 		return
// 	}
// 	log.Println("routing config is ", string(jsonData))
// }
// func routeProcessor(routeConfig constants.Route) error {
// 	k8sConfig, err := utils.GetK8sClient()
// 	if err != nil {
// 		log.Println("aborting")
// 		return fmt.Errorf("error creating K8s Client", err.Error())
// 	}
// 	gateWayType := os.Getenv("GATEWAY_PROVIDER")
// 	var routeTranslator usecase.RouteTranslator
// 	if gateWayType == "" {
// 		log.Println("selecting Base translator")
// 		routeTranslator = translator.NewBaseRouteTranslator(k8sConfig, "my-namespace")
// 	} else {
// 		log.Println("Implementation Not Created yet")
// 	}
// 	resp, err := routeTranslator.CreateHTTPRoute(context.Background(), routeConfig.Routes[0])
// 	if err != nil {
// 		log.Println("error creating http route", err.Error())
// 		return fmt.Errorf("error creating http route: %w", err)
// 	}
// 	log.Println("HTTP Route Created", resp)
// 	clientSet, err := gatewayclient.NewForConfig(k8sConfig)
// 	if err != nil {
// 		log.Println("Unable to create route")
// 		return fmt.Errorf("error creating gateway client: %w", err)
// 	}
// 	route, err := clientSet.GatewayV1().HTTPRoutes("my-namespace").Get(context.Background(), resp.Name, v1.GetOptions{})
// 	if err != nil {
// 		route, err = clientSet.GatewayV1().HTTPRoutes("my-namespace").Create(context.Background(), resp, v1.CreateOptions{})
// 	}
// 	resp.ResourceVersion = route.ResourceVersion
// 	route, err = clientSet.GatewayV1().HTTPRoutes("my-namespace").Update(context.Background(), resp, v1.UpdateOptions{})

// 	if err != nil {
// 		log.Println("error creating route", err.Error())
// 		return fmt.Errorf("error creating route: %w", err)
// 	}
// 	log.Println("Route Created Successfully", route)
// }

func (c ConfigProcessorPluginUsecaseHandler) ReadConfig(fileName string) error {
	data, err := os.ReadFile(fileName)
	if err != nil {
		log.Println("Unable to read file", err.Error())
		return fmt.Errorf("could not read config file %s: %w", fileName, err)
	}
	var config models.RegisterServiceV2
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("could not parse config file %s: %w", fileName, err)
	}
	code, resp, err := utils.MakeAPICall(http.MethodGet, config.ConfigEndpoint, map[string]string{}, nil)

	if err != nil {
		log.Println("Error in Processing routing", err.Error(), config.ServiceID)
		return err
	}
	if code != http.StatusOK {
		log.Printf("endpoint returned %d", code)
		return fmt.Errorf("endpoint returned %d", code)
	}
	var routeConfig constants.Route
	err = json.Unmarshal(resp, &routeConfig)
	if err != nil {
		log.Println("error Unmarshalling routing request, please follow README.md", err.Error())
	}
	ctx := context.Background()
	objects, err := c.translatorUc.TranslateAll(ctx, routeConfig.Routes)
	if err != nil {
		log.Printf("Error translating routes: %v", err)
		return err
	}
	for _, obj := range objects {
		PrintAsYaml(obj)
	}
	return nil
}

func PrintAsYaml(objects ...interface{}) error {
	for _, obj := range objects {
		// 1. Marshal the original K8s struct to JSON first
		// K8s types have JSON tags that make this very reliable
		jsonData, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("failed to marshal to json: %w", err)
		}

		// 2. Unmarshal into a generic Map
		var m map[string]interface{}
		if err := json.Unmarshal(jsonData, &m); err != nil {
			return fmt.Errorf("failed to unmarshal into map: %w", err)
		}

		// 3. PHYSICAL DELETION
		// This removes the "status" key and all its children from the map
		delete(m, "status")

		// 4. THE FIX: Marshal the MAP, not the original 'obj'
		// 'obj' still has the status; 'm' does not.
		yamlData, err := k8syaml.Marshal(m)
		if err != nil {
			return fmt.Errorf("failed to marshal sanitized map to yaml: %w", err)
		}

		// 5. Output for Argo CD
		fmt.Println("---")
		fmt.Println(string(yamlData))
	}
	return nil
}
