package usecase

import (
	"context"
	"the_governor/constants"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type RouteTranslator interface {
	// Core Gateway API methods (all translators must implement)
	TranslateHTTPRoute(ctx context.Context, route constants.RouteDefinition) (*gatewayv1.HTTPRoute, []metav1.Object, error)
	TranslateBackendRef(ctx context.Context, backend constants.BackendRef) (gatewayv1.BackendRef, []metav1.Object, error)
	// CreateService(ctx context.Context, backend constants.BackendRef) error

	// Optional vendor-specific features
	SupportsHealthChecks() bool
	SupportsHeaderTransformation() bool
	SupportsRateLimiting() bool

	// Apply extensions if supported
	ApplyExtensions(ctx context.Context, route constants.RouteDefinition, httpRoute *gatewayv1.HTTPRoute) error
}

// GatewayTranslator is the pluggable replacement for RouteTranslator.
// Translate returns all Kubernetes objects needed for a route (HTTPRoute,
// backend Services, RouteOptions, Upstreams, VirtualServices — whatever the
// target gateway requires). Callers apply the slice without knowing the type.
type GatewayTranslator interface {
	Translate(ctx context.Context, route constants.RouteDefinition) ([]client.Object, error)

	SupportsHealthChecks() bool
	SupportsHeaderTransformation() bool
	SupportsRateLimiting() bool
}
