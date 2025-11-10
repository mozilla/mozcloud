package helm

import (
	"strings"
	"testing"
)

func TestIsHelmChart(t *testing.T) {
	testCases := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "Valid Helm chart from examples",
			path: "../../examples/helm/helloworld",
			want: true,
		},
		{
			name: "Kustomize directory (should be false)",
			path: "../../examples/kustomize/helloworld",
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
			got := IsHelmChart(tc.path)
			if got != tc.want {
				t.Errorf("IsHelmChart(%q) = %v; want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestRenderChart(t *testing.T) {
	// Using our example helm chart
	chartPath := "../../examples/helm/helloworld"
	releaseName := "test-release"

	t.Run("Render with default values", func(t *testing.T) {
		valuesFiles := []string{}
		debug := false // Test the silent path
		update := false

		output, err := RenderChart(chartPath, releaseName, valuesFiles, debug, update)
		if err != nil {
			t.Fatalf("RenderChart failed: %v", err)
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

		if !strings.Contains(output, "name: test-release-") {
			t.Errorf("Output missing expected release name 'test-release-'. Got:\n%s", output)
		}
	})

	t.Run("Render with override values", func(t *testing.T) {
		// Using dev values file
		valuesFile := "../../examples/helm/helloworld/values-dev.yaml"

		valuesFiles := []string{valuesFile}
		debug := false // Test the silent path
		update := false

		output, err := RenderChart(chartPath, releaseName, valuesFiles, debug, update)
		if err != nil {
			t.Fatalf("RenderChart failed: %v", err)
		}

		// Checking for the .Values.image.tag change
		if !strings.Contains(output, "nginx:dev") {
			t.Errorf("Output missing expected nginx:dev. Got:\n%s", output)
		}

		if output == "" {
			t.Errorf("Rendered output was empty")
		}
	})

	t.Run("Render with override values and chart dependencies", func(t *testing.T) {
		// Using dev values file
		valuesFile := "../../examples/helm/helloworld/values-dev.yaml"

		valuesFiles := []string{valuesFile}
		debug := false // Test the silent path
		update := true

		output, err := RenderChart(chartPath, releaseName, valuesFiles, debug, update)
		if err != nil {
			t.Fatalf("RenderChart failed: %v", err)
		}

		// Checking for the .Values.image.tag change
		if !strings.Contains(output, "nginx:dev") {
			t.Errorf("Output missing expected nginx:dev. Got:\n%s", output)
		}

		// Checking for the dep configMap change
		if !strings.Contains(output, "test-release-dep") {
			t.Errorf("Output missing expected test-release-dep. Got:\n%s", output)
		}

		if output == "" {
			t.Errorf("Rendered output was empty")
		}
	})
}
