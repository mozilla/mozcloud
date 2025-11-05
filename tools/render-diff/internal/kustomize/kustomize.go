// Package kustomize provides functions to build kustomizations,
// and check if the provided path contains kustomizations
package kustomize

import (
	"fmt"

	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

// RenderKustomization runs 'kustomize build' on a given path and
// returns the rendered manifests.
func RenderKustomization(kustomizePath string) (string, error) {
	opts := krusty.MakeDefaultOptions()
	opts.PluginConfig.HelmConfig.Enabled = false

	k := krusty.MakeKustomizer(opts)

	fSys := filesys.MakeFsOnDisk()

	// Run the kustomize build
	// This is the equivalent of `kustomize build <kustomizePath>`
	resMap, err := k.Run(fSys, kustomizePath)
	if err != nil {
		return "", fmt.Errorf("failed to run kustomize build: %w", err)
	}

	// Encode the resulting resources into a single YAML byte slice
	yamlBytes, err := resMap.AsYaml()
	if err != nil {
		return "", fmt.Errorf("failed to marshal kustomize resources to YAML: %w", err)
	}

	// Return as a string, ready for diffing
	return string(yamlBytes), nil
}

func IsKustomize(path string) bool {
	opts := krusty.MakeDefaultOptions()
	opts.PluginConfig.HelmConfig.Enabled = false

	fSys := filesys.MakeFsOnDisk()
	k := krusty.MakeKustomizer(opts)

	_, err := k.Run(fSys, path)
	return err == nil
}
