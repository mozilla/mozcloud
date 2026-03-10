package helmutil_test

import (
	"reflect"
	"testing"

	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/helmutil"
)

// --- DeepMerge ---

func TestDeepMerge(t *testing.T) {
	t.Run("src key overwrites dst scalar", func(t *testing.T) {
		dst := map[string]interface{}{"a": 1}
		helmutil.DeepMerge(dst, map[string]interface{}{"a": 2})
		if dst["a"] != 2 {
			t.Errorf("got %v, want 2", dst["a"])
		}
	})

	t.Run("src key added when not in dst", func(t *testing.T) {
		dst := map[string]interface{}{"a": 1}
		helmutil.DeepMerge(dst, map[string]interface{}{"b": 2})
		if dst["b"] != 2 {
			t.Errorf("got %v, want 2", dst["b"])
		}
		if dst["a"] != 1 {
			t.Errorf("a was overwritten, got %v", dst["a"])
		}
	})

	t.Run("nested maps merged recursively", func(t *testing.T) {
		dst := map[string]interface{}{
			"ingress": map[string]interface{}{"host": "old.example.com", "port": 80},
		}
		helmutil.DeepMerge(dst, map[string]interface{}{
			"ingress": map[string]interface{}{"host": "new.example.com"},
		})
		ingress := dst["ingress"].(map[string]interface{})
		if ingress["host"] != "new.example.com" {
			t.Errorf("host: got %v, want new.example.com", ingress["host"])
		}
		if ingress["port"] != 80 {
			t.Errorf("port was removed, got %v", ingress["port"])
		}
	})

	t.Run("src map overwrites dst scalar at same key", func(t *testing.T) {
		dst := map[string]interface{}{"x": "scalar"}
		helmutil.DeepMerge(dst, map[string]interface{}{"x": map[string]interface{}{"nested": true}})
		if _, ok := dst["x"].(map[string]interface{}); !ok {
			t.Errorf("expected map at x, got %T", dst["x"])
		}
	})

	t.Run("empty src leaves dst unchanged", func(t *testing.T) {
		dst := map[string]interface{}{"a": 1}
		orig := map[string]interface{}{"a": 1}
		helmutil.DeepMerge(dst, map[string]interface{}{})
		if !reflect.DeepEqual(dst, orig) {
			t.Errorf("dst changed: got %v", dst)
		}
	})

	t.Run("empty dst gets all src keys", func(t *testing.T) {
		dst := map[string]interface{}{}
		src := map[string]interface{}{"a": 1, "b": "two"}
		helmutil.DeepMerge(dst, src)
		if !reflect.DeepEqual(dst, src) {
			t.Errorf("got %v, want %v", dst, src)
		}
	})
}

// --- ParseResources ---

func TestParseResources(t *testing.T) {
	t.Run("single resource", func(t *testing.T) {
		manifests := `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
`
		got := helmutil.ParseResources(manifests)
		want := []string{"apps/v1/Deployment/my-app"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("multiple resources", func(t *testing.T) {
		manifests := `---
apiVersion: v1
kind: Service
metadata:
  name: my-svc
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
`
		got := helmutil.ParseResources(manifests)
		want := []string{"v1/Service/my-svc", "apps/v1/Deployment/my-app"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("only first name per resource block captured", func(t *testing.T) {
		// A resource with a nested name field (e.g. in spec) should not produce a duplicate.
		manifests := `---
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
data:
  name: should-be-ignored
`
		got := helmutil.ParseResources(manifests)
		want := []string{"v1/ConfigMap/my-config"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("empty manifest returns nil", func(t *testing.T) {
		got := helmutil.ParseResources("")
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})

	t.Run("incomplete block without name skipped", func(t *testing.T) {
		manifests := `---
apiVersion: v1
kind: Namespace
`
		got := helmutil.ParseResources(manifests)
		if len(got) != 0 {
			t.Errorf("expected empty, got %v", got)
		}
	})
}

// --- SummarizeDiff ---

func TestSummarizeDiff(t *testing.T) {
	t.Run("counts additions and removals", func(t *testing.T) {
		diff := `--- a/file
+++ b/file
+added line
+another added
-removed line`
		got := helmutil.SummarizeDiff(diff)
		if got != "2 additions, 1 removals" {
			t.Errorf("got %q, want %q", got, "2 additions, 1 removals")
		}
	})

	t.Run("excludes --- and +++ header lines", func(t *testing.T) {
		diff := `--- original
+++ modified
+only addition`
		got := helmutil.SummarizeDiff(diff)
		if got != "1 additions, 0 removals" {
			t.Errorf("got %q, want %q", got, "1 additions, 0 removals")
		}
	})

	t.Run("empty diff", func(t *testing.T) {
		got := helmutil.SummarizeDiff("")
		if got != "0 additions, 0 removals" {
			t.Errorf("got %q, want %q", got, "0 additions, 0 removals")
		}
	})
}
