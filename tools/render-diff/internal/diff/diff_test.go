package diff

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGetRepoRoot(t *testing.T) {
	path, err := GetRepoRoot()
	if err != nil {
		t.Fatalf("GetRepoRoot() failed: %v", err)
	}

	if path == "" {
		t.Errorf("Expected a non-empty path, got empty string")
	}

	if !filepath.IsAbs(path) {
		t.Errorf("Expected an absolute path, got: %s", path)
	}
}

// TestRenderManifests uses the chart and kustomization in our
// examples directory
func TestRenderManifests(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		debug       bool
		values      []string
		wantContent string
		wantErr     bool
	}{
		{
			name:        "Renders Helm chart",
			path:        "../../examples/helm/helloworld",
			debug:       false,
			values:      nil,
			wantContent: "kind: ConfigMap",
			wantErr:     false,
		},
		{
			name:        "Renders Helm chart with values",
			path:        "../../examples/helm/helloworld",
			debug:       false,
			values:      []string{"../../examples/helm/helloworld/values-dev.yaml"},
			wantContent: "nginx:dev",
			wantErr:     false,
		},
		{
			name:        "Renders Kustomize project",
			path:        "../../examples/kustomize/helloworld",
			debug:       false,
			values:      nil,
			wantContent: "kind: ConfigMap",
			wantErr:     false,
		},
		{
			name:    "Returns error for invalid path",
			path:    "../../examples/not-a-real-path",
			debug:   false,
			values:  nil,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := RenderManifests(tc.path, tc.values, tc.debug)

			if (err != nil) != tc.wantErr {
				t.Fatalf("RenderManifests() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr && !strings.Contains(output, tc.wantContent) {
				t.Errorf("RenderManifests() output did not contain %q. Got:\n%s", tc.wantContent, output)
			}
		})
	}
}

func TestCreateDiff(t *testing.T) {
	testCases := []struct {
		name     string
		a        string
		b        string
		fromName string
		toName   string
		want     string
	}{
		{
			name:     "Simple change",
			a:        "line 1\nline 2\nline 3",
			b:        "line 1\nline two\nline 3",
			fromName: "a.txt",
			toName:   "b.txt",
			want:     "--- a.txt\n+++ b.txt\n@@ -1,3 +1,3 @@\n line 1\n-line 2\n+line two\n line 3\n\\ No newline at end of file\n",
		},
		{
			name:     "No changes",
			a:        "line 1\nline 2",
			b:        "line 1\nline 2",
			fromName: "a.txt",
			toName:   "b.txt",
			want:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := CreateDiff(tc.a, tc.b, tc.fromName, tc.toName)
			if strings.TrimSpace(got) != strings.TrimSpace(tc.want) {
				t.Errorf("CreateDiff() =\n%q\nWant:\n%q", got, tc.want)
			}
		})
	}
}
