package utils

import (
	"context"
	"log"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ApplyStrategyAnnotation = "governor.io/apply-strategy"
	ApplyStrategyCreateOnly = "create-only"
)

// ApplyObject creates or updates obj on the cluster.
// If the object carries annotation governor.io/apply-strategy=create-only it is
// created only when it does not yet exist — existing resources are left untouched
// (first-writer-wins semantics used for Gloo VirtualService domain ownership).
func ApplyObject(ctx context.Context, k8sClient client.Client, obj client.Object) error {
	strategy := obj.GetAnnotations()[ApplyStrategyAnnotation]

	existing := obj.DeepCopyObject().(client.Object)
	err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), existing)

	if k8serrors.IsNotFound(err) {
		log.Printf("Creating %s/%s", obj.GetNamespace(), obj.GetName())
		return k8sClient.Create(ctx, obj)
	}
	if err != nil {
		return err
	}

	// Object exists
	if strategy == ApplyStrategyCreateOnly {
		log.Printf("Skipping %s/%s — already owned (create-only)", obj.GetNamespace(), obj.GetName())
		return nil
	}

	log.Printf("Updating %s/%s", obj.GetNamespace(), obj.GetName())
	obj.SetResourceVersion(existing.GetResourceVersion())
	return k8sClient.Update(ctx, obj)
}
