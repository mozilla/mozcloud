package pathsafe_test

import (
	"path/filepath"
	"testing"

	"github.com/mozilla/mozcloud/tools/mozcloud-mcp/internal/pathsafe"
)

func TestCheck(t *testing.T) {
	t.Run("target within root", func(t *testing.T) {
		err := pathsafe.Check("/charts/myapp/subdir", []string{"/charts/myapp"})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("target equals root", func(t *testing.T) {
		err := pathsafe.Check("/charts/myapp", []string{"/charts/myapp"})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("target outside root", func(t *testing.T) {
		err := pathsafe.Check("/charts/other", []string{"/charts/myapp"})
		if err == nil {
			t.Error("expected error for path outside root, got nil")
		}
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		err := pathsafe.Check("/charts/myapp/../other", []string{"/charts/myapp"})
		if err == nil {
			t.Error("expected error for traversal path, got nil")
		}
	})

	t.Run("prefix match does not allow sibling dirs", func(t *testing.T) {
		// /charts/myapp-evil should NOT match root /charts/myapp
		err := pathsafe.Check("/charts/myapp-evil", []string{"/charts/myapp"})
		if err == nil {
			t.Error("expected error for sibling dir with shared prefix, got nil")
		}
	})

	t.Run("empty roots defaults to target as root", func(t *testing.T) {
		err := pathsafe.Check("/charts/myapp", []string{})
		if err != nil {
			t.Errorf("expected nil with empty roots (self-root), got %v", err)
		}
	})

	t.Run("multiple roots — match second", func(t *testing.T) {
		err := pathsafe.Check("/tmp/work", []string{"/charts/myapp", "/tmp"})
		if err != nil {
			t.Errorf("expected nil when target matches second root, got %v", err)
		}
	})

	t.Run("relative paths resolved", func(t *testing.T) {
		abs, _ := filepath.Abs(".")
		err := pathsafe.Check(".", []string{abs})
		if err != nil {
			t.Errorf("expected nil for relative path resolving to root, got %v", err)
		}
	})
}
