package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ppanda/chitragupta/internal/config"
	"github.com/ppanda/chitragupta/pkg/installer"
	"github.com/ppanda/chitragupta/pkg/registry"
	"github.com/ppanda/chitragupta/pkg/resolver"
)

func runInstall(cfg *config.Config, packageSpec string, global bool, varsFlag []string) error {
	// Parse package spec (name@version)
	parts := strings.Split(packageSpec, "@")
	name := parts[0]
	version := "latest"
	if len(parts) > 1 {
		version = parts[1]
	}

	// Parse template variables
	vars := make(map[string]string)
	for _, v := range varsFlag {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) == 2 {
			vars[kv[0]] = kv[1]
		}
	}

	// Auto-detect some vars if not provided
	if _, ok := vars["REPO_NAME"]; !ok {
		if cwd, err := os.Getwd(); err == nil {
			vars["REPO_NAME"] = filepath.Base(cwd)
		}
	}

	// Setup registry and resolver
	reg := registry.NewLocal(cfg.RegistryPath)
	res := resolver.New(reg)

	// Resolve dependencies
	fmt.Printf("Resolving dependencies for %s@%s...\n", name, version)
	packages, err := res.Resolve(name, version)
	if err != nil {
		return fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	// Determine installation scope
	scope := "repo"
	repoDir := ".claude"
	if global {
		scope = "global"
		repoDir = cfg.GlobalDir
	}

	// Install packages in dependency order
	inst := installer.New(cfg.GlobalDir, repoDir)

	for _, pkg := range packages {
		fmt.Printf("Installing %s@%s...\n", pkg.Name, pkg.Version)

		if err := inst.Install(pkg.GetPath(), &pkg.Manifest, scope, vars); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkg.Name, err)
		}

		fmt.Printf("✓ Installed %s@%s\n", pkg.Name, pkg.Version)
	}

	fmt.Printf("\n✓ Successfully installed %s@%s and %d dependencies\n", name, version, len(packages)-1)
	return nil
}

func runPublish(cfg *config.Config, pkgDir string) error {
	// Ensure absolute path
	absPath, err := filepath.Abs(pkgDir)
	if err != nil {
		return err
	}

	// Check manifest exists
	manifestPath := filepath.Join(absPath, "manifest.yaml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest.yaml not found in %s", pkgDir)
	}

	reg := registry.NewLocal(cfg.RegistryPath)

	fmt.Printf("Publishing package from %s...\n", pkgDir)
	if err := reg.Publish(absPath); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	fmt.Printf("✓ Package published successfully\n")
	return nil
}

func runList(cfg *config.Config) error {
	reg := registry.NewLocal(cfg.RegistryPath)

	packages, err := reg.List()
	if err != nil {
		return err
	}

	if len(packages) == 0 {
		fmt.Println("No packages found in registry")
		return nil
	}

	fmt.Printf("Found %d package(s):\n\n", len(packages))
	for _, pkg := range packages {
		fmt.Printf("%s@%s\n", pkg.Name, pkg.Version)
		fmt.Printf("  %s\n", pkg.Manifest.Description)
		if len(pkg.Manifest.Dependencies) > 0 {
			fmt.Printf("  Dependencies: %v\n", pkg.Manifest.Dependencies)
		}
		fmt.Println()
	}

	return nil
}

func runSearch(cfg *config.Config, query string) error {
	reg := registry.NewLocal(cfg.RegistryPath)

	packages, err := reg.Search(query)
	if err != nil {
		return err
	}

	if len(packages) == 0 {
		fmt.Printf("No packages found matching '%s'\n", query)
		return nil
	}

	fmt.Printf("Found %d package(s):\n\n", len(packages))
	for _, pkg := range packages {
		fmt.Printf("%s@%s\n", pkg.Name, pkg.Version)
		fmt.Printf("  %s\n", pkg.Manifest.Description)
		fmt.Println()
	}

	return nil
}
