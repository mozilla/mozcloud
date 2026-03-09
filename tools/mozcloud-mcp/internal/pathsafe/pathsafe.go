// Package pathsafe validates filesystem paths for tools that write to disk.
// Only paths within configured allowed roots are permitted. This prevents
// accidental writes outside the user's chart directory.
package pathsafe

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Check returns nil if target is within one of the allowed roots (after
// resolving both paths to their absolute, cleaned forms). If roots is empty
// the target itself is used as the sole allowed root, which means any
// subdirectory of target is permitted but nothing else.
func Check(target string, roots []string) error {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("cannot resolve target path %q: %w", target, err)
	}

	// Default: allow only the target directory itself (and its children).
	if len(roots) == 0 {
		roots = []string{absTarget}
	}

	for _, root := range roots {
		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		// Ensure the comparison uses a trailing separator so that
		// "/foo/bar" does not match "/foo/barbaz".
		if absTarget == absRoot || strings.HasPrefix(absTarget, absRoot+string(filepath.Separator)) {
			return nil
		}
	}

	return fmt.Errorf("path %q is outside the allowed write roots %v", target, roots)
}

// CheckDestination is a convenience wrapper for tools that receive an explicit
// destination argument. It resolves destination and checks it against roots.
func CheckDestination(destination string, roots []string) error {
	return Check(destination, roots)
}
