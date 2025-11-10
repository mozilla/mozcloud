package kustomize

import (
	"strings"
	"testing"
)

func TestIsKustomize(t *testing.T) {
	testCases := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "Valid Kustomize directory",
			path: "../../examples/kustomize/helloworld",
			want: true,
		},
		{
			name: "Helm directory (should be false)",
			path: "../../examples/helm/helloworld",
			want: false,
		},
		{
			name: "Non-existent directory",
			path: "testdata/does-not-exist",
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsKustomize(tc.path)
			if got != tc.want {
				t.Errorf("IsKustomize(%q) = %v; want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestRenderKustomization(t *testing.T) {
	t.Run("Renders a valid kustomization", func(t *testing.T) {
		path := "../../examples/kustomize/helloworld"

		output, err := RenderKustomization(path)
		if err != nil {
			t.Fatalf("RenderKustomization failed: %v", err)
		}

		if !strings.Contains(output, "kind: ConfigMap") {
			t.Errorf("Output missing expected content 'kind: ConfigMap'. Got:\n%s", output)
		}

		if !strings.Contains(output, "kind: Deployment") {
			t.Errorf("Output missing expected content 'kind: Deployment'. Got:\n%s", output)
		}

		if !strings.Contains(output, "kind: Service") {
			t.Errorf("Output missing expected content 'kind: Service'. Got:\n%s", output)
		}

		if !strings.Contains(output, "app: hello") {
			t.Errorf("Output missing expected 'app: hello' label. Got:\n%s", output)
		}
	})

	t.Run("Fails on an invalid path", func(t *testing.T) {
		// This is a Helm chart, not kustomization
		path := "../../examples/helm/helloworld"

		_, err := RenderKustomization(path)
		if err == nil {
			t.Errorf("RenderKustomization did not fail for an invalid path, expected error")
		}
	})

	t.Run("Fails on a non-existent path", func(t *testing.T) {
		path := "testdata/does-not-exist"

		_, err := RenderKustomization(path)
		if err == nil {
			t.Errorf("RenderKustomization did not fail for a non-existent path, expected error")
		}
	})
}
