package translator

import (
	"context"
	"testing"

	"the_governor/constants"

	corev1 "k8s.io/api/core/v1"
)

func TestTranslateBackendRefKube(t *testing.T) {
	translator := &BaseTranslator{}
	backend := constants.BackendRef{
		Type:        "kube",
		ServiceName: "my-service",
		Namespace:   "default",
		Port:        8080,
	}

	ref, extra, err := translator.TranslateBackendRef(context.Background(), backend)
	if err != nil {
		t.Fatalf("TranslateBackendRef() error = %v", err)
	}
	if len(extra) != 0 {
		t.Fatalf("expected no extra objects, got %d", len(extra))
	}
	if ref.BackendObjectReference.Name != "my-service" {
		t.Fatalf("expected service name my-service, got %s", ref.BackendObjectReference.Name)
	}
	if ref.BackendObjectReference.Namespace == nil || *ref.BackendObjectReference.Namespace != "default" {
		t.Fatalf("expected namespace default, got %v", ref.BackendObjectReference.Namespace)
	}
	if ref.BackendObjectReference.Port == nil || *ref.BackendObjectReference.Port != 8080 {
		t.Fatalf("expected port 8080, got %v", ref.BackendObjectReference.Port)
	}
}

func TestTranslateBackendRefExternal(t *testing.T) {
	translator := &BaseTranslator{}
	backend := constants.BackendRef{
		Type:        "external",
		Host:        "external.example.com",
		ServiceName: "external-svc",
		Namespace:   "external-ns",
		Port:        9090,
	}

	ref, extra, err := translator.TranslateBackendRef(context.Background(), backend)
	if err != nil {
		t.Fatalf("TranslateBackendRef() error = %v", err)
	}
	if len(extra) != 1 {
		t.Fatalf("expected one extra object, got %d", len(extra))
	}
	service, ok := extra[0].(*corev1.Service)
	if !ok {
		t.Fatalf("expected extra object to be *corev1.Service, got %T", extra[0])
	}
	if service.Name != "external-svc-external-service" {
		t.Fatalf("expected service name external-svc-external-service, got %s", service.Name)
	}
	if ref.BackendObjectReference.Name != "external-svc-external-service" {
		t.Fatalf("expected backend name external-svc-external-service, got %s", ref.BackendObjectReference.Name)
	}
	if ref.BackendObjectReference.Port == nil || *ref.BackendObjectReference.Port != 9090 {
		t.Fatalf("expected backend port 9090, got %v", ref.BackendObjectReference.Port)
	}
}

func TestTranslateHTTPRouteWithMethods(t *testing.T) {
	translator := &BaseTranslator{}
	route := constants.RouteDefinition{
		RouteName:   "route-1",
		Enabled:     true,
		Description: "test route",
		Hostnames:   []string{"example.com"},
		Path:        "/api",
		PathType:    "Exact",
		Methods:     []string{"GET", "POST"},
		Backend: []constants.BackendRef{{
			Type:        "kube",
			ServiceName: "backend-service",
			Namespace:   "default",
			Port:        8080,
		}},
	}

	httpRoute, extra, err := translator.TranslateHTTPRoute(context.Background(), route)
	if err != nil {
		t.Fatalf("TranslateHTTPRoute() error = %v", err)
	}
	if len(extra) != 0 {
		t.Fatalf("expected no extra objects, got %d", len(extra))
	}
	if httpRoute.Name != "route-1" {
		t.Fatalf("expected httpRoute name route-1, got %s", httpRoute.Name)
	}
	if len(httpRoute.Spec.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(httpRoute.Spec.Rules))
	}
	if len(httpRoute.Spec.Rules[0].Matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(httpRoute.Spec.Rules[0].Matches))
	}
	if httpRoute.Spec.Rules[0].Matches[0].Method == nil {
		t.Fatal("expected method pointer for first match")
	}
}

func TestBaseGatewayTranslatorTranslateAll(t *testing.T) {
	translator := NewBaseGatewayTranslator("default")
	routes := []constants.RouteDefinition{{
		RouteName:   "route-all",
		Enabled:     true,
		Description: "route all",
		Hostnames:   []string{"example.com"},
		Path:        "/",
		PathType:    "Exact",
		Backend: []constants.BackendRef{{
			Type:        "kube",
			ServiceName: "backend-service",
			Namespace:   "default",
			Port:        8080,
		}},
	}}

	objects, err := translator.TranslateAll(context.Background(), routes)
	if err != nil {
		t.Fatalf("TranslateAll() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
}
