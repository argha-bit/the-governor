package configprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"the_governor/constants"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type fakeTranslator struct {
	objects []client.Object
	err     error
}

func (f fakeTranslator) TranslateAll(ctx context.Context, routes []constants.RouteDefinition) ([]client.Object, error) {
	return f.objects, f.err
}

func (f fakeTranslator) SupportsHealthChecks() bool         { return false }
func (f fakeTranslator) SupportsHeaderTransformation() bool { return false }
func (f fakeTranslator) SupportsRateLimiting() bool         { return false }

func TestReadConfig(t *testing.T) {
	validRoute := constants.Route{
		Version: "v1",
		ServiceMetadata: constants.RouteMetaData{
			ServiceName: "test-service",
			TeamName:    "test-team",
			Namespace:   "default",
			Owner:       "owner",
		},
		Routes: []constants.RouteDefinition{{
			RouteName:   "my-route",
			Enabled:     true,
			Description: "a route",
			Hostnames:   []string{"example.com"},
			Path:        "/",
			PathType:    "Exact",
			Backend: []constants.BackendRef{{
				Type:        "external",
				Host:        "example.com",
				Port:        80,
				ServiceName: "my-service",
				Namespace:   "default",
			}},
		}},
	}
	encoded, err := json.Marshal(validRoute)
	if err != nil {
		t.Fatalf("failed to marshal route: %v", err)
	}

	serverOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(encoded)
	}))
	defer serverOK.Close()

	serverError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	}))
	defer serverError.Close()

	tests := []struct {
		name        string
		fileContent string
		translator  fakeTranslator
		wantErr     bool
		wantMessage string
	}{
		{
			name:        "missing config file",
			fileContent: "",
			translator:  fakeTranslator{},
			wantErr:     true,
			wantMessage: "could not read config file",
		},
		{
			name:        "invalid yaml file",
			fileContent: "serviceId: [not-a-string",
			translator:  fakeTranslator{},
			wantErr:     true,
			wantMessage: "could not parse config file",
		},
		{
			name:        "endpoint returns non-200",
			fileContent: fmt.Sprintf("serviceId: id\nserviceName: name\nteamName: team\nnamespace: default\ncontactEmail: x@x.com\nconfigEndpoint: %s\nwebhookUrl: http://example.com", serverError.URL),
			translator:  fakeTranslator{},
			wantErr:     true,
			wantMessage: "endpoint returned 500",
		},
		{
			name:        "translator failure",
			fileContent: fmt.Sprintf("serviceId: id\nserviceName: name\nteamName: team\nnamespace: default\ncontactEmail: x@x.com\nconfigEndpoint: %s\nwebhookUrl: http://example.com", serverOK.URL),
			translator:  fakeTranslator{err: errors.New("translate failed")},
			wantErr:     true,
			wantMessage: "translate failed",
		},
		{
			name:        "success path",
			fileContent: fmt.Sprintf("serviceId: id\nserviceName: name\nteamName: team\nnamespace: default\ncontactEmail: x@x.com\nconfigEndpoint: %s\nwebhookUrl: http://example.com", serverOK.URL),
			translator:  fakeTranslator{objects: []client.Object{&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "my-service", Namespace: "default"}}}},
			wantErr:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filePath := ""
			if tc.name != "missing config file" {
				f, err := os.CreateTemp("", "config-*.yaml")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				defer os.Remove(f.Name())
				if _, err := f.WriteString(tc.fileContent); err != nil {
					t.Fatalf("failed to write temp file: %v", err)
				}
				_ = f.Close()
				filePath = f.Name()
			}

			handler := NewConfigProcessorPluginUsecaseHandler(tc.translator)
			err := handler.ReadConfig(filePath)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ReadConfig() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil && tc.wantMessage != "" && !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("expected error to contain %q, got %v", tc.wantMessage, err)
			}
		})
	}
}
