package utils

import (
	"os"
	"testing"

	"the_governor/constants"
)

func TestGetK8sClient_InvalidDeploymentKind(t *testing.T) {
	old := os.Getenv("DEPLOYMENT_KIND")
	defer os.Setenv("DEPLOYMENT_KIND", old)
	_ = os.Setenv("DEPLOYMENT_KIND", "INVALID_KIND")

	_, err := GetK8sClient()
	if err == nil {
		t.Fatal("expected error for invalid DEPLOYMENT_KIND")
	}
}

func TestCreateOutofClusterConfig_NoKubeconfig(t *testing.T) {
	old := os.Getenv("KUBECONFIG_PATH")
	defer os.Setenv("KUBECONFIG_PATH", old)
	_ = os.Unsetenv("KUBECONFIG_PATH")

	_, err := createOutofClusterConfig()
	if err == nil {
		t.Fatal("expected error when KUBECONFIG_PATH is unset")
	}
}

func TestBuildExternalK8sService(t *testing.T) {
	tests := []struct {
		name         string
		routeBackend constants.BackendRef
		namespace    string
		wantName     string
		wantHost     string
		wantPort     int32
	}{
		{
			name: "build external service",
			routeBackend: constants.BackendRef{
				Host:        "example.com",
				Port:        8080,
				ServiceName: "my-service",
			},
			namespace: "test-ns",
			wantName:  "my-service-external-service",
			wantHost:  "example.com",
			wantPort:  8080,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, err := BuildExternalK8sService(tc.routeBackend, tc.namespace)
			if err != nil {
				t.Fatalf("BuildExternalK8sService() error = %v", err)
			}
			if svc.Name != tc.wantName {
				t.Fatalf("expected service name %s, got %s", tc.wantName, svc.Name)
			}
			if svc.Namespace != tc.namespace {
				t.Fatalf("expected namespace %s, got %s", tc.namespace, svc.Namespace)
			}
			if svc.Spec.ExternalName != tc.wantHost {
				t.Fatalf("expected external name %s, got %s", tc.wantHost, svc.Spec.ExternalName)
			}
			if len(svc.Spec.Ports) != 1 || svc.Spec.Ports[0].Port != tc.wantPort {
				t.Fatalf("expected port %d, got %v", tc.wantPort, svc.Spec.Ports)
			}
		})
	}
}
