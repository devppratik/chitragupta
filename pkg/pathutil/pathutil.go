package pathutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateExtractPath ensures target path doesn't escape base directory
func ValidateExtractPath(base, target string) error {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("failed to resolve base path: %w", err)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Ensure target is within base directory
	if !strings.HasPrefix(absTarget, absBase+string(filepath.Separator)) &&
		absTarget != absBase {
		return fmt.Errorf("path traversal detected: %s escapes %s", target, base)
	}

	return nil
}

// SafeJoin joins paths and validates no traversal
func SafeJoin(base string, elem ...string) (string, error) {
	target := filepath.Join(base, filepath.Join(elem...))
	if err := ValidateExtractPath(base, target); err != nil {
		return "", err
	}
	return target, nil
}
