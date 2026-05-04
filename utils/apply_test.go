package utils

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplyObject(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}

	ctx := context.Background()
	baseService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: 8080}},
		},
	}

	tests := []struct {
		name        string
		setup       func(c client.Client) *corev1.Service
		wantErr     bool
		wantCreated bool
		wantUpdated bool
	}{
		{
			name: "create new service",
			setup: func(c client.Client) *corev1.Service {
				return baseService.DeepCopy()
			},
			wantErr:     false,
			wantCreated: true,
		},
		{
			name: "update existing service",
			setup: func(c client.Client) *corev1.Service {
				existing := baseService.DeepCopy()
				existing.Labels = map[string]string{"initial": "true"}
				if err := c.Create(ctx, existing); err != nil {
					t.Fatalf("setup create existing service failed: %v", err)
				}
				updated := existing.DeepCopy()
				updated.Labels = map[string]string{"updated": "true"}
				return updated
			},
			wantErr:     false,
			wantUpdated: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			obj := tc.setup(fakeClient)
			err := ApplyObject(ctx, fakeClient, obj)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ApplyObject() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantCreated {
				stored := &corev1.Service{}
				if err := fakeClient.Get(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, stored); err != nil {
					t.Fatalf("expected created service, got error: %v", err)
				}
			}
			if tc.wantUpdated {
				stored := &corev1.Service{}
				if err := fakeClient.Get(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, stored); err != nil {
					t.Fatalf("expected updated service, got error: %v", err)
				}
				if stored.Labels["updated"] != "true" {
					t.Fatalf("expected service label updated=true, got %v", stored.Labels)
				}
			}
		})
	}
}
