package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"the_governor/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	rest "k8s.io/client-go/rest"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

func GetK8sClient() (*rest.Config, error) {
	deploymentKind := os.Getenv("DEPLOYMENT_KIND")
	var config *rest.Config
	var err error

	switch deploymentKind {
	case "IN_CLUSTER":
		config, err = rest.InClusterConfig()
		if err != nil {
			//Handle for fallback
			log.Println("Error creating in-cluster config:", err.Error())
			config, err = createOutofClusterConfig()
			if err != nil {
				return config, err
			}
		}
		// Create in-cluster Kubernetes client
	case "OUT_OF_CLUSTER":
		// Create out-of-cluster Kubernetes client
		config, err = createOutofClusterConfig()
		if err != nil {
			return config, err
		}
	default:
		// Handle invalid deployment kind
		log.Default().Println("Invalid DEPLOYMENT_KIND environment variable. Must be 'IN_CLUSTER' or 'OUT_OF_CLUSTER'")
		return config, fmt.Errorf("invalid DEPLOYMENT_KIND environment variable")
	}
	return config, err
}
func createOutofClusterConfig() (*rest.Config, error) {
	var config *rest.Config
	var err error
	kubeconfigPath := os.Getenv("KUBECONFIG_PATH")
	if kubeconfigPath == "" {
		log.Println("KUBECONFIG_PATH environment variable is not set")
		return nil, fmt.Errorf("failed to create kubernetes config %w", err.Error())
	}
	config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Println("Error creating out-of-cluster config:", err.Error())
		return nil, fmt.Errorf("failed to create kubernetes config %w", err.Error())
	}
	return config, nil
}

func CreateExternalK8sService(routeBackend constants.BackendRef, namespace string, client *rest.Config) (*corev1.Service, error) {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-external-service", routeBackend.ServiceName),
			Namespace: namespace,
			Labels: map[string]string{
				"app": fmt.Sprintf("%s-external-service", routeBackend.ServiceName),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     int32(routeBackend.Port),
				},
			},
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: routeBackend.Host,
		},
	}
	clientSet, err := kubernetes.NewForConfig(client)
	if err != nil {
		log.Println("Error creating Kubernetes clientset:", err.Error())
		return &corev1.Service{}, fmt.Errorf("failed to create kubernetes clientset %w", err.Error())
	}
	result, err := clientSet.CoreV1().Services(namespace).Create(
		context.Background(),
		service,
		metav1.CreateOptions{},
	)
	if err != nil {
		log.Println("Error creating Kubernetes service:", err.Error())
		return &corev1.Service{}, fmt.Errorf("failed to create kubernetes service %w", err.Error())
	}
	log.Printf("Created service %s in namespace %s\n", result.Name, result.Namespace)
	return result, nil
}
