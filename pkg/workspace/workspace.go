package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ppanda/chitragupta/pkg/config"
	"github.com/ppanda/chitragupta/pkg/types"
)

// Workspace represents a workspace member
type Workspace struct {
	Path   string
	Config *types.Config
	IsRoot bool
}

// WorkspaceSet holds all workspaces
type WorkspaceSet struct {
	Root    *Workspace
	Members []*Workspace
	AllDeps map[string]string // merged dependencies
}

// Discover finds all workspaces from root
func Discover(rootPath string) (*WorkspaceSet, error) {
	// Parse root config
	rootConfigPath := filepath.Join(rootPath, "chitragupta.yml")
	rootConfig, err := config.Parse(rootConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root config: %w", err)
	}

	ws := &WorkspaceSet{
		Root: &Workspace{
			Path:   rootPath,
			Config: rootConfig,
			IsRoot: true,
		},
		Members: make([]*Workspace, 0),
		AllDeps: make(map[string]string),
	}

	// No workspaces defined
	if len(rootConfig.Workspaces) == 0 {
		return ws, nil
	}

	// Discover workspace members
	for _, pattern := range rootConfig.Workspaces {
		matches, err := expandPattern(rootPath, pattern)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			member, err := loadWorkspace(match)
			if err != nil {
				fmt.Printf("Warning: failed to load workspace %s: %v\n", match, err)
				continue
			}

			ws.Members = append(ws.Members, member)
		}
	}

	// Merge dependencies
	ws.mergeDependencies()

	return ws, nil
}

// loadWorkspace loads workspace config
func loadWorkspace(path string) (*Workspace, error) {
	configPath := filepath.Join(path, "chitragupta.yml")

	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no chitragupta.yml in %s", path)
	}

	cfg, err := config.Parse(configPath)
	if err != nil {
		return nil, err
	}

	return &Workspace{
		Path:   path,
		Config: cfg,
		IsRoot: false,
	}, nil
}

// expandPattern expands glob pattern
func expandPattern(root, pattern string) ([]string, error) {
	fullPattern := filepath.Join(root, pattern)

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		matches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}

		// Filter for directories
		dirs := make([]string, 0)
		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.IsDir() {
				dirs = append(dirs, match)
			}
		}

		return dirs, nil
	}

	// Single path
	return []string{fullPattern}, nil
}

// mergeDependencies combines deps from root and all members
func (ws *WorkspaceSet) mergeDependencies() {
	// Add root dependencies
	ws.addDeps(ws.Root.Config.Dependencies)

	// Add member dependencies
	for _, member := range ws.Members {
		ws.addDeps(member.Config.Dependencies)
	}
}

// addDeps adds dependencies to merged set
func (ws *WorkspaceSet) addDeps(deps types.ConfigDependencies) {
	// Registry deps
	for _, dep := range deps.Registry {
		name, version := parseSpec(dep)
		if existingVersion, exists := ws.AllDeps[name]; exists && existingVersion != version {
			fmt.Printf("Warning: dependency version conflict for %s: %s vs %s\n", name, existingVersion, version)
		}
		ws.AllDeps[name] = version
	}

	// Git deps
	for _, dep := range deps.Git {
		name := extractName(dep)
		ws.AllDeps[name] = dep
	}

	// APM deps
	for _, dep := range deps.APM {
		name := extractName(dep)
		ws.AllDeps[name] = dep
	}

	// OCI deps
	for _, dep := range deps.OCI {
		name := extractName(dep)
		ws.AllDeps[name] = dep
	}

	// HTTP deps
	for _, dep := range deps.HTTP {
		name := extractName(dep)
		ws.AllDeps[name] = dep
	}
}

// parseSpec extracts name and version
func parseSpec(spec string) (name, version string) {
	parts := strings.SplitN(spec, "@", 2)
	name = parts[0]
	version = "latest"
	if len(parts) > 1 {
		version = parts[1]
	}
	return name, version
}

// extractName gets package name from any spec
func extractName(spec string) string {
	// Remove protocol
	spec = strings.TrimPrefix(spec, "https://")
	spec = strings.TrimPrefix(spec, "http://")

	// Split by /
	parts := strings.Split(spec, "/")
	if len(parts) == 0 {
		return spec
	}

	// Get last part
	name := parts[len(parts)-1]

	// Remove git/tag suffixes
	name = strings.TrimSuffix(name, ".git")
	if idx := strings.Index(name, "#"); idx >= 0 {
		name = name[:idx]
	}
	if idx := strings.Index(name, ":"); idx >= 0 {
		name = name[:idx]
	}

	return name
}

// All returns all workspaces including root
func (ws *WorkspaceSet) All() []*Workspace {
	all := make([]*Workspace, 0, len(ws.Members)+1)
	all = append(all, ws.Root)
	all = append(all, ws.Members...)
	return all
}
