// Package helmclient provides a shared Helm environment (settings + action config)
// used by multiple tool groups to avoid duplicating boilerplate.
package helmclient

import (
	"fmt"
	"log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

// Settings returns a new Helm CLI environment with default values.
// Callers may mutate the returned object before passing it to NewActionConfig.
func Settings() *cli.EnvSettings {
	return cli.New()
}

// NewActionConfig constructs a Helm action.Configuration for the given
// namespace and Helm settings. It uses an in-memory (no-op) Kubernetes
// client so it is safe to call without a live cluster.
func NewActionConfig(ns string, settings *cli.EnvSettings) (*action.Configuration, error) {
	cfg := new(action.Configuration)
	logger := func(format string, v ...interface{}) {
		log.Printf("[helm] "+format, v...)
	}
	// Use "memory" driver so no Kubernetes cluster is required for rendering.
	if err := cfg.Init(settings.RESTClientGetter(), ns, "memory", logger); err != nil {
		return nil, fmt.Errorf("helm action config init: %w", err)
	}
	return cfg, nil
}

// DebugLog is a no-op debug logger suitable for passing to Helm SDK calls
// that expect a printf-style function. It writes to stderr.
func DebugLog(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, "[helm-debug] "+format+"\n", v...)
}
