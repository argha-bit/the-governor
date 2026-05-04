package translator

import (
	"context"
	"testing"

	"the_governor/constants"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestGlooEdgeTranslator_TranslateAll(t *testing.T) {
	translator := NewGlooEdgeTranslator("default")

	routes := []constants.RouteDefinition{{
		RouteName:   "route-1",
		Enabled:     true,
		Description: "test route",
		Hostnames:   []string{"example.com"},
		Path:        "/",
		PathType:    "Exact",
		Methods:     []string{"GET"},
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
	if len(objects) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(objects))
	}

	vs, ok := objects[0].(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("expected first object to be *unstructured.Unstructured, got %T", objects[0])
	}
	if vs.GetKind() != "VirtualService" {
		t.Fatalf("expected VirtualService, got %s", vs.GetKind())
	}
	annotations, found, err := unstructured.NestedStringMap(vs.Object, "metadata", "annotations")
	if err != nil || !found {
		t.Fatalf("expected metadata.annotations on virtual service")
	}
	if annotations[applyStrategyAnnotation] != applyStrategyCreateOnly {
		t.Fatalf("expected annotation %s=%s, got %s", applyStrategyAnnotation, applyStrategyCreateOnly, annotations[applyStrategyAnnotation])
	}

	upstream, ok := objects[1].(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("expected second object to be *unstructured.Unstructured, got %T", objects[1])
	}
	if upstream.GetKind() != "Upstream" {
		t.Fatalf("expected Upstream, got %s", upstream.GetKind())
	}

	rt, ok := objects[2].(*unstructured.Unstructured)
	if !ok {
		t.Fatalf("expected third object to be *unstructured.Unstructured, got %T", objects[2])
	}
	if rt.GetKind() != "RouteTable" {
		t.Fatalf("expected RouteTable, got %s", rt.GetKind())
	}

	labels, found, err := unstructured.NestedStringMap(rt.Object, "metadata", "labels")
	if err != nil || !found {
		t.Fatalf("expected metadata.labels on route table")
	}
	if labels[domainLabelKey] != "example-com" {
		t.Fatalf("expected label %s=example-com, got %s", domainLabelKey, labels[domainLabelKey])
	}
}

func TestGlooEdgeTranslator_buildUpstream_UnsupportedBackend(t *testing.T) {
	translator := &GlooEdgeTranslator{namespace: "default"}

	tests := []struct {
		name    string
		backend constants.BackendRef
		wantErr bool
	}{
		{
			name: "unsupported backend type",
			backend: constants.BackendRef{
				Type: "unsupported",
			},
			wantErr: true,
		},
		{
			name: "external backend builds successfully",
			backend: constants.BackendRef{
				Type: "external",
				Host: "example.com",
				Port: 80,
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := translator.buildUpstream(tc.backend)
			if (err != nil) != tc.wantErr {
				t.Fatalf("buildUpstream() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestGlooEdgeTranslator_buildRouteOptions(t *testing.T) {
	translator := &GlooEdgeTranslator{namespace: "default"}

	tests := []struct {
		name    string
		route   constants.RouteDefinition
		wantKey string
	}{
		{
			name: "header manipulation options",
			route: constants.RouteDefinition{
				Extensions: &constants.RouteExtensions{
					RequestHeaders: &constants.HeaderModification{
						Add: []constants.Header{{Name: "x-test", Value: "value"}},
					},
					ResponseHeaders: &constants.HeaderModification{
						Remove: []string{"x-remove"},
					},
				},
			},
			wantKey: "headerManipulation",
		},
		{
			name: "timeout and retries options",
			route: constants.RouteDefinition{
				Extensions: &constants.RouteExtensions{
					Timeout: "5s",
					Retries: &constants.RetryConfig{Attempts: 3, RetryOn: []string{"5xx"}},
				},
			},
			wantKey: "timeout",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			options := translator.buildRouteOptions(tc.route)
			if options == nil {
				t.Fatalf("buildRouteOptions() returned nil for %s", tc.name)
			}
			if _, ok := options[tc.wantKey]; !ok {
				t.Fatalf("expected options to contain %s, got %v", tc.wantKey, options)
			}
		})
	}
}
