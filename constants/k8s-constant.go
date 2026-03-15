package constants

const (
	GATEWAY_NETWORKING_API_VERSION = "gateway.networking.k8s.io/v1"
	K8S_VERSION                    = "v1"
	K8S_SERVICE_KIND               = "Service"
)

var K8S_SERVICE_TYPES = map[string]string{
	"ClusterIP":    "ClusterIP",
	"NodePort":     "NodePort",
	"LoadBalancer": "LoadBalancer",
	"ExternalName": "ExternalName",
}
