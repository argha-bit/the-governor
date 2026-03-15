package usecase

import (
	"context"
	"the_governor/constants"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type RouteTranslator interface {
	// Core Gateway API methods (all translators must implement)
	CreateHTTPRoute(ctx context.Context, route constants.RouteDefinition) (*gatewayv1.HTTPRoute, error)
	CreateBackendRef(ctx context.Context, backend constants.BackendRef) (gatewayv1.BackendRef, error)
	// CreateService(ctx context.Context, backend constants.BackendRef) error

	// Optional vendor-specific features
	SupportsHealthChecks() bool
	SupportsHeaderTransformation() bool
	SupportsRateLimiting() bool

	// Apply extensions if supported
	ApplyExtensions(ctx context.Context, route constants.RouteDefinition, httpRoute *gatewayv1.HTTPRoute) error
}
