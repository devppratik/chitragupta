package manifest

import (
	"fmt"
	"os"

	"github.com/ppanda/chitragupta/pkg/types"
	"gopkg.in/yaml.v3"
)

// Parse reads and validates a manifest.yaml file
func Parse(path string) (*types.Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var m types.Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if err := validate(&m); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &m, nil
}

// validate checks required fields
func validate(m *types.Manifest) error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	if m.Description == "" {
		return fmt.Errorf("description is required")
	}
	return nil
}
