package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ppanda/chitragupta/internal/config"
	"github.com/ppanda/chitragupta/pkg/types"
	"github.com/ppanda/chitragupta/pkg/workspace"
)

// installWorkspaces handles workspace installation
func installWorkspaces(cfg *config.Config, rootPath string, skipSecurity bool) error {
	// Discover workspaces
	ws, err := workspace.Discover(rootPath)
	if err != nil {
		return err
	}

	if len(ws.Members) == 0 {
		// No workspaces, do regular install
		return installFromManifest(cfg, skipSecurity)
	}

	fmt.Printf("Found %d workspaces\n", len(ws.Members))
	for _, member := range ws.Members {
		relPath, _ := filepath.Rel(rootPath, member.Path)
		fmt.Printf("  - %s\n", relPath)
	}

	// Install shared dependencies to root
	if len(ws.AllDeps) > 0 {
		fmt.Printf("\nInstalling %d shared dependencies...\n", len(ws.AllDeps))

		// Use root config with merged deps
		if err := installWorkspaceSharedDeps(cfg, ws, skipSecurity); err != nil {
			return fmt.Errorf("failed to install shared deps: %w", err)
		}
	}

	// Install workspace-specific deps
	for _, member := range ws.Members {
		relPath, _ := filepath.Rel(rootPath, member.Path)

		// Check if member has any deps
		allMemberDeps := make([]string, 0)
		allMemberDeps = append(allMemberDeps, member.Config.Dependencies.Registry...)
		allMemberDeps = append(allMemberDeps, member.Config.Dependencies.Git...)
		allMemberDeps = append(allMemberDeps, member.Config.Dependencies.APM...)
		allMemberDeps = append(allMemberDeps, member.Config.Dependencies.OCI...)
		allMemberDeps = append(allMemberDeps, member.Config.Dependencies.HTTP...)

		if len(allMemberDeps) == 0 {
			continue
		}

		fmt.Printf("\nInstalling %d dependencies for %s...\n", len(allMemberDeps), relPath)

		// Install member manifest directly
		memberManifest := filepath.Join(member.Path, "chitragupta.yml")
		if err := installManifestDirect(cfg, memberManifest, skipSecurity); err != nil {
			return fmt.Errorf("failed to install deps for %s: %w", relPath, err)
		}
	}

	fmt.Println("\n✓ All workspaces installed")
	return nil
}

// installWorkspaceSharedDeps installs shared deps to root
func installWorkspaceSharedDeps(cfg *config.Config, ws *workspace.WorkspaceSet, skipSecurity bool) error {
	// Install root manifest deps directly (skip workspace detection)
	manifestPath := filepath.Join(ws.Root.Path, "chitragupta.yml")
	return installManifestDirect(cfg, manifestPath, skipSecurity)
}

// countSpecificDeps counts deps unique to this member
func countSpecificDeps(deps types.ConfigDependencies, shared map[string]string) int {
	count := 0

	// Check each dep type
	allDeps := make([]string, 0)
	allDeps = append(allDeps, deps.Registry...)
	allDeps = append(allDeps, deps.Git...)
	allDeps = append(allDeps, deps.APM...)
	allDeps = append(allDeps, deps.OCI...)
	allDeps = append(allDeps, deps.HTTP...)

	for _, dep := range allDeps {
		name := extractDepName(dep)
		if _, exists := shared[name]; !exists {
			count++
		}
	}

	return count
}

// extractDepName gets name from dep spec
func extractDepName(spec string) string {
	// Remove protocol
	spec = strings.TrimPrefix(spec, "https://")
	spec = strings.TrimPrefix(spec, "http://")

	// Split by @
	parts := strings.SplitN(spec, "@", 2)
	name := parts[0]

	// Get last path component
	pathParts := strings.Split(name, "/")
	name = pathParts[len(pathParts)-1]

	// Remove suffixes
	name = strings.TrimSuffix(name, ".git")
	if idx := strings.Index(name, "#"); idx >= 0 {
		name = name[:idx]
	}

	return name
}
