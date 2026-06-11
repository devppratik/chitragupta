package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppanda/chitragupta/pkg/types"
	"gopkg.in/yaml.v3"
)

// Parse reads and validates chitragupta.yml
func Parse(path string) (*types.Config, error) {
	return parseWithVisited(path, make(map[string]bool))
}

func parseWithVisited(path string, visited map[string]bool) (*types.Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	if visited[absPath] {
		return nil, fmt.Errorf("circular dependency detected: %s", absPath)
	}
	visited[absPath] = true

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Handle extends (workspace child configs)
	if cfg.Extends != "" {
		parent, err := parseWithVisited(filepath.Join(filepath.Dir(absPath), cfg.Extends), visited)
		if err != nil {
			return nil, fmt.Errorf("failed to load parent config: %w", err)
		}
		cfg = merge(parent, &cfg)
	}

	return &cfg, nil
}

// validate checks required fields
func validate(cfg *types.Config) error {
	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}
	if cfg.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}

// merge combines parent and child configs (child overrides parent)
func merge(parent, child *types.Config) types.Config {
	result := *parent

	// Override scalar fields if child provides them
	if child.Name != "" {
		result.Name = child.Name
	}
	if child.Version != "" {
		result.Version = child.Version
	}
	if child.Description != "" {
		result.Description = child.Description
	}
	if child.Author != "" {
		result.Author = child.Author
	}
	if child.License != "" {
		result.License = child.License
	}
	if child.Homepage != "" {
		result.Homepage = child.Homepage
	}

	// Merge dependencies (child additions)
	result.Dependencies.Registry = append(result.Dependencies.Registry, child.Dependencies.Registry...)
	result.Dependencies.Git = append(result.Dependencies.Git, child.Dependencies.Git...)
	result.Dependencies.OCI = append(result.Dependencies.OCI, child.Dependencies.OCI...)
	result.Dependencies.HTTP = append(result.Dependencies.HTTP, child.Dependencies.HTTP...)
	result.Dependencies.APM = append(result.Dependencies.APM, child.Dependencies.APM...)
	result.Dependencies.MCP = append(result.Dependencies.MCP, child.Dependencies.MCP...)

	return result
}

// FindConfig searches for chitragupta.yml in current and parent directories
func FindConfig() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	current := cwd
	for {
		candidate := filepath.Join(current, "chitragupta.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		// Try parent directory
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", fmt.Errorf("chitragupta.yml not found")
}
