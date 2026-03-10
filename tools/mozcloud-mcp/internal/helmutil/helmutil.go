// Package helmutil provides pure helper functions for working with Helm
// values maps and rendered Kubernetes manifests.
package helmutil

import (
	"fmt"
	"strings"
)

// DeepMerge recursively merges src into dst. For keys present in both, if
// both values are maps they are merged recursively; otherwise src wins.
func DeepMerge(dst, src map[string]interface{}) {
	for k, srcVal := range src {
		if dstVal, exists := dst[k]; exists {
			dstMap, dstIsMap := dstVal.(map[string]interface{})
			srcMap, srcIsMap := srcVal.(map[string]interface{})
			if dstIsMap && srcIsMap {
				DeepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[k] = srcVal
	}
}

// ParseResources parses a multi-document YAML manifest string and returns a
// slice of "apiVersion/Kind/name" strings, one per Kubernetes resource.
func ParseResources(manifests string) []string {
	var resources []string
	var apiVersion, kind string

	for _, line := range strings.Split(manifests, "\n") {
		line = strings.TrimSpace(line)
		if line == "---" {
			apiVersion, kind = "", ""
			continue
		}
		if strings.HasPrefix(line, "apiVersion:") {
			apiVersion = strings.TrimSpace(strings.TrimPrefix(line, "apiVersion:"))
		} else if strings.HasPrefix(line, "kind:") {
			kind = strings.TrimSpace(strings.TrimPrefix(line, "kind:"))
		} else if strings.HasPrefix(line, "name:") && apiVersion != "" && kind != "" {
			name := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			resources = append(resources, fmt.Sprintf("%s/%s/%s", apiVersion, kind, name))
			apiVersion, kind = "", ""
		}
	}
	return resources
}

// SummarizeDiff counts added and removed lines in a unified diff string and
// returns a human-readable summary.
func SummarizeDiff(diff string) string {
	var adds, removes int
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			adds++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removes++
		}
	}
	return fmt.Sprintf("%d additions, %d removals", adds, removes)
}
