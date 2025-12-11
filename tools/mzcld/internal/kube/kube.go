package kube

import (
	"fmt"

	appsv1alpha1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// We are creating a new kubernetes client
// Requires an active kubectl context
func NewClient() (client.Client, *runtime.Scheme, error) {
	scheme := runtime.NewScheme()
	if err := appsv1alpha1.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("add argo application scheme: %w", err)
	}

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("get kube config: %w", err)
	}

	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, nil, fmt.Errorf("create client: %w", err)
	}

	return c, scheme, nil
}
