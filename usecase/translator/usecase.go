package translator

import (
	"context"
	"fmt"
	"the_governor/constants"
	"the_governor/usecase"
	"the_governor/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type BaseTranslator struct {
	clientset *rest.Config
	namespace string
}

func (b *BaseTranslator) SupportsHealthChecks() bool {
	return false
}
func (b *BaseTranslator) SupportsHeaderTransformation() bool {
	return true
}
func (b *BaseTranslator) SupportsRateLimiting() bool {
	return false
}

func (b *BaseTranslator) CreateHTTPRoute(ctx context.Context, route constants.RouteDefinition) (*gatewayv1.HTTPRoute, error) {
	var backendRefs []gatewayv1.HTTPBackendRef
	for _, backend := range route.Backend {
		ref, err := b.CreateBackendRef(ctx, backend)
		if err != nil {
			return nil, err
		}
		backendRefs = append(backendRefs, gatewayv1.HTTPBackendRef{BackendRef: ref})
	}
	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      route.RouteName,
			Namespace: "my-namespace",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{
						Name:      gatewayv1.ObjectName("test-gateway"),
						Namespace: ptr.To(gatewayv1.Namespace("my-namespace")),
					},
				},
			},
			Hostnames: convertToHostnames(route.Hostnames),
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  ptr.To(convertPathType(route.PathType)),
								Value: ptr.To(route.Path),
							},
						},
					},
					BackendRefs: backendRefs,
				},
			},
		},
	}

	return httpRoute, nil
}

func (b *BaseTranslator) CreateBackendRef(ctx context.Context, backend constants.BackendRef) (gatewayv1.BackendRef, error) {
	switch backend.Type {
	case "kube":
		var weight int
		if backend.Weight == nil {
			weight = 100
		} else {
			weight = *backend.Weight
		}
		return gatewayv1.BackendRef{
			BackendObjectReference: gatewayv1.BackendObjectReference{
				Group:     ptr.To(gatewayv1.Group("")),
				Kind:      ptr.To(gatewayv1.Kind("Service")),
				Name:      gatewayv1.ObjectName(backend.ServiceName),
				Namespace: ptr.To(gatewayv1.Namespace(backend.Namespace)),
				Port:      ptr.To(gatewayv1.PortNumber(backend.Port)),
			},
			Weight: ptr.To(int32(weight)),
		}, nil
	case "external":
		//for external we need to create the Service and extract the BackendRef
		service, err := utils.CreateExternalK8sService(backend, "my-namespace", b.clientset)
		if err != nil {
			return gatewayv1.BackendRef{}, err
		}
		return gatewayv1.BackendRef{
			BackendObjectReference: gatewayv1.BackendObjectReference{
				Group:     ptr.To(gatewayv1.Group("")),
				Kind:      ptr.To(gatewayv1.Kind("Service")),
				Name:      gatewayv1.ObjectName(service.Name),
				Namespace: ptr.To(gatewayv1.Namespace(service.Namespace)),
				Port:      ptr.To(gatewayv1.PortNumber(service.Spec.Ports[0].Port)),
			},
		}, nil
	}
	return gatewayv1.BackendRef{}, fmt.Errorf("unknown backend type")
}

func (b *BaseTranslator) ApplyExtensions(ctx context.Context, route constants.RouteDefinition, httpRoute *gatewayv1.HTTPRoute) error {
	// Default: do nothing
	return nil
}

func convertPathType(pathType string) gatewayv1.PathMatchType {
	switch pathType {
	case "PathPrefix":
		return gatewayv1.PathMatchPathPrefix
	case "Exact":
		return gatewayv1.PathMatchExact
	case "RegularExpression":
		return gatewayv1.PathMatchRegularExpression
	default:
		return gatewayv1.PathMatchPathPrefix
	}
}

func convertToHostnames(hostnames []string) []gatewayv1.Hostname {
	result := make([]gatewayv1.Hostname, len(hostnames))
	for i, h := range hostnames {
		result[i] = gatewayv1.Hostname(h)
	}
	return result
}

// func (g *GlooTranslator) CreateHTTPRoute(ctx context.Context, route constants.RouteDefinition) (*gatewayv1.HTTPRoute, error) {}

func NewBaseRouteTranslator(clientset *rest.Config, namespace string) usecase.RouteTranslator {
	return &BaseTranslator{
		clientset: clientset,
		namespace: namespace,
	}
}
