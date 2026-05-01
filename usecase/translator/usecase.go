package translator

import (
	"context"
	"fmt"
	"log"
	"the_governor/constants"
	"the_governor/usecase"
	"the_governor/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type BaseTranslator struct {
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

func (b *BaseTranslator) TranslateHTTPRoute(ctx context.Context, route constants.RouteDefinition) (*gatewayv1.HTTPRoute, []metav1.Object, error) {
	var backendRefs []gatewayv1.HTTPBackendRef
	var backendObjects []metav1.Object
	for _, backend := range route.Backend {
		ref, extraObjects, err := b.TranslateBackendRef(ctx, backend)
		if err != nil {
			return nil, backendObjects, err
		}
		backendObjects = append(backendObjects, extraObjects...)
		backendRefs = append(backendRefs, gatewayv1.HTTPBackendRef{BackendRef: ref})
	}
	log.Println(len(backendObjects), "is the number of extra backend objects created during translation")
	httpRoute := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "gateway.networking.k8s.io/v1",
			Kind:       "HTTPRoute",
		},
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
					Matches:     buildMatches(route),
					Filters:     buildHeaderFilters(route.Extensions),
					BackendRefs: backendRefs,
				},
			},
		},
	}

	return httpRoute, backendObjects, nil
}

func (b *BaseTranslator) TranslateBackendRef(ctx context.Context, backend constants.BackendRef) (gatewayv1.BackendRef, []metav1.Object, error) {
	var extraObjects []metav1.Object
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
		}, nil, nil
	case "external":
		//for external we need to create the Service and extract the BackendRef
		service, err := utils.BuildExternalK8sService(backend, "my-namespace")
		if err != nil {
			return gatewayv1.BackendRef{}, nil, err
		}
		extraObjects = append(extraObjects, service)
		log.Println("Extra Object length", len(extraObjects))
		return gatewayv1.BackendRef{
			BackendObjectReference: gatewayv1.BackendObjectReference{
				Group:     ptr.To(gatewayv1.Group("")),
				Kind:      ptr.To(gatewayv1.Kind("Service")),
				Name:      gatewayv1.ObjectName(service.Name),
				Namespace: ptr.To(gatewayv1.Namespace(service.Namespace)),
				Port:      ptr.To(gatewayv1.PortNumber(service.Spec.Ports[0].Port)),
			},
		}, extraObjects, nil
	}
	return gatewayv1.BackendRef{}, nil, fmt.Errorf("unknown backend type")
}

func (b *BaseTranslator) ApplyExtensions(ctx context.Context, route constants.RouteDefinition, httpRoute *gatewayv1.HTTPRoute) error {
	// Default: do nothing
	return nil
}

func buildHeaderFilters(ext *constants.RouteExtensions) []gatewayv1.HTTPRouteFilter {
	//handle nil case
	if ext == nil {
		return nil
	}
	var filters []gatewayv1.HTTPRouteFilter
	//handle Request case if not nil
	if ext.RequestHeaders != nil {
		filters = append(filters, gatewayv1.HTTPRouteFilter{
			Type: gatewayv1.HTTPRouteFilterRequestHeaderModifier,
			RequestHeaderModifier: &gatewayv1.HTTPHeaderFilter{
				Add:    toHTTPHeaders(ext.RequestHeaders.Add),
				Remove: ext.RequestHeaders.Remove,
			},
		})
	}
	//handle Response case if not nil
	if ext.ResponseHeaders != nil {
		filters = append(filters, gatewayv1.HTTPRouteFilter{
			Type: gatewayv1.HTTPRouteFilterResponseHeaderModifier,
			ResponseHeaderModifier: &gatewayv1.HTTPHeaderFilter{
				Add:    toHTTPHeaders(ext.ResponseHeaders.Add),
				Remove: ext.ResponseHeaders.Remove,
			},
		})
	}
	return filters
}

func toHTTPHeaders(headers []constants.Header) []gatewayv1.HTTPHeader {
	if len(headers) == 0 {
		return nil
	}
	result := make([]gatewayv1.HTTPHeader, len(headers))
	for i, h := range headers {
		result[i] = gatewayv1.HTTPHeader{
			Name:  gatewayv1.HTTPHeaderName(h.Name),
			Value: h.Value,
		}
	}
	return result
}

func buildMatches(route constants.RouteDefinition) []gatewayv1.HTTPRouteMatch {
	pathMatch := &gatewayv1.HTTPPathMatch{
		Type:  ptr.To(convertPathType(route.PathType)),
		Value: ptr.To(route.Path),
	}
	if len(route.Methods) == 0 {
		return []gatewayv1.HTTPRouteMatch{{Path: pathMatch}}
	}
	matches := make([]gatewayv1.HTTPRouteMatch, len(route.Methods))
	for i, m := range route.Methods {
		matches[i] = gatewayv1.HTTPRouteMatch{
			Path:   pathMatch,
			Method: ptr.To(gatewayv1.HTTPMethod(m)),
		}
	}
	return matches
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

func NewBaseRouteTranslator(namespace string) usecase.RouteTranslator {
	return &BaseTranslator{
		namespace: namespace,
	}
}

// BaseGatewayTranslator implements the pluggable GatewayTranslator interface.
// It reuses BaseTranslator internally so all existing translation logic is preserved.
type BaseGatewayTranslator struct {
	base *BaseTranslator
}

func NewBaseGatewayTranslator(namespace string) usecase.GatewayTranslator {
	return &BaseGatewayTranslator{
		base: &BaseTranslator{namespace: namespace},
	}
}

func (g *BaseGatewayTranslator) SupportsHealthChecks() bool       { return false }
func (g *BaseGatewayTranslator) SupportsHeaderTransformation() bool { return true }
func (g *BaseGatewayTranslator) SupportsRateLimiting() bool        { return false }

func (g *BaseGatewayTranslator) Translate(ctx context.Context, route constants.RouteDefinition) ([]client.Object, error) {
	httpRoute, backendObjects, err := g.base.TranslateHTTPRoute(ctx, route)
	if err != nil {
		return nil, err
	}
	objects := make([]client.Object, 0, 1+len(backendObjects))
	objects = append(objects, httpRoute)
	for _, obj := range backendObjects {
		objects = append(objects, obj.(client.Object))
	}
	return objects, nil
}
