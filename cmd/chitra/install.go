package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppanda/chitragupta/internal/config"
	configpkg "github.com/ppanda/chitragupta/pkg/config"
	"github.com/ppanda/chitragupta/pkg/downloader"
	"github.com/ppanda/chitragupta/pkg/graph"
	"github.com/ppanda/chitragupta/pkg/installer"
	"github.com/ppanda/chitragupta/pkg/lockfile"
	"github.com/ppanda/chitragupta/pkg/logger"
	"github.com/ppanda/chitragupta/pkg/security"
	"github.com/ppanda/chitragupta/pkg/sources"
	"github.com/ppanda/chitragupta/pkg/template"
	"github.com/ppanda/chitragupta/pkg/types"
	"github.com/ppanda/chitragupta/pkg/workspace"
)

// installFromManifest reads chitragupta.yml and installs all deps
func installFromManifest(cfg *config.Config, skipSecurity bool) error {
	// Find manifest
	manifestPath, err := configpkg.FindConfig()
	if err != nil {
		return fmt.Errorf("no chitragupta.yml found: %w", err)
	}

	logger.Info("Reading %s...", manifestPath)

	// Check if workspace root
	repoPath := filepath.Dir(manifestPath)
	ws, wsErr := workspace.Discover(repoPath)
	if wsErr == nil && len(ws.Members) > 0 {
		// Workspace mode
		return installWorkspaces(cfg, repoPath, skipSecurity)
	}

	// Direct install
	return installManifestDirect(cfg, manifestPath, skipSecurity)
}

// installManifestDirect installs from manifest path without workspace check
func installManifestDirect(cfg *config.Config, manifestPath string, skipSecurity bool) error {
	repoPath := filepath.Dir(manifestPath)

	// Parse manifest
	manifest, err := configpkg.Parse(manifestPath)
	if err != nil {
		return err
	}

	// Auto-detect template vars
	ctx, err := template.Detect(repoPath)
	if err != nil {
		logger.Warn("Template detection failed: %v", err)
		ctx = &template.Context{}
	}

	vars := ctx.ToMap()
	if vars["LANGUAGE"] != "" {
		logger.Info("Detected context: %s (%s)", vars["REPO_NAME"], vars["LANGUAGE"])
	} else {
		logger.Info("Detected context: %s", vars["REPO_NAME"])
	}

	// Initialize multi-source resolver
	resolver := sources.NewMultiSourceResolver(cfg.RegistryPath)

	// Collect all dependencies
	allDeps := make([]string, 0)
	allDeps = append(allDeps, manifest.Dependencies.Registry...)
	allDeps = append(allDeps, manifest.Dependencies.Git...)
	allDeps = append(allDeps, manifest.Dependencies.APM...)
	allDeps = append(allDeps, manifest.Dependencies.OCI...)
	allDeps = append(allDeps, manifest.Dependencies.HTTP...)

	if len(allDeps) == 0 {
		logger.Info("No dependencies to install")
		return nil
	}

	logger.Info("Resolving %d dependencies...", len(allDeps))
	logger.Debug("Dependencies: %v", allDeps)

	// Build dependency graph with transitive resolution
	g := graph.NewGraph()
	resolved := make(map[string]*types.Package)
	visited := make(map[string]bool)

	// Create virtual root node
	g.Root = g.AddNode("_root", "0.0.0", "virtual")

	// Recursively resolve dependencies
	var resolveDep func(dep string, parent *graph.Node) error
	resolveDep = func(dep string, parent *graph.Node) error {
		// Resolve package metadata
		pkg, err := resolver.Resolve(dep)
		if err != nil {
			return fmt.Errorf("failed to resolve %s: %w", dep, err)
		}

		// Check pkg != nil before accessing fields
		if pkg == nil {
			return fmt.Errorf("resolver returned nil package for %s", dep)
		}

		// Check if already resolved
		key := pkg.Name + "@" + pkg.Version
		if visited[key] {
			// Already resolved, just add edge if needed
			if existingNode, ok := g.Nodes[key]; ok && parent != nil {
				g.AddEdge(parent, existingNode)
			}
			return nil
		}

		visited[key] = true

		// Add node to graph
		node := g.AddNode(pkg.Name, pkg.Version, string(sources.DetectSource(dep)))

		// Connect parent to this package
		if parent != nil {
			g.AddEdge(parent, node)
		}

		resolved[pkg.Name] = pkg

		// Resolve transitive dependencies
		for depName, depVersion := range pkg.Manifest.Dependencies {
			depSpec := depName + "@" + depVersion
			if err := resolveDep(depSpec, node); err != nil {
				return err
			}
		}

		return nil
	}

	// Resolve all top-level dependencies
	for _, dep := range allDeps {
		if err := resolveDep(dep, g.Root); err != nil {
			return err
		}
	}

	// Topological sort for install order
	sortedNodes, err := g.TopologicalSort()
	if err != nil {
		return err
	}

	// Remove virtual root from sorted list
	sorted := make([]string, 0, len(sortedNodes)-1)
	for _, node := range sortedNodes {
		if node.Name != "_root" {
			sorted = append(sorted, node.Name)
		}
	}

	logger.Info("Installing %d packages...", len(sorted))
	logger.Debug("Install order: %v", sorted)

	// Download in parallel
	packages := make([]*types.Package, 0, len(sorted))
	for _, name := range sorted {
		if pkg, ok := resolved[name]; ok {
			packages = append(packages, pkg)
		}
	}

	dl := downloader.NewDownloader(filepath.Join(cfg.RegistryPath, "cache"), 10)

	// Build spec map from original deps
	specMap := make(map[string]string)
	for _, dep := range allDeps {
		pkg, _ := resolver.Resolve(dep)
		if pkg != nil {
			specMap[pkg.Name] = dep
		}
	}

	err = dl.Download(packages, func(pkg *types.Package) error {
		// Get original spec
		spec := pkg.Name + "@" + pkg.Version
		if origSpec, ok := specMap[pkg.Name]; ok {
			spec = origSpec
		}

		// Fetch to cache (downloader handles this)
		cachePath := filepath.Join(cfg.RegistryPath, "cache", pkg.Name, pkg.Version)

		// Only fetch if not cached
		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			fetchedPkg, err := resolver.Fetch(spec, cachePath)
			if err != nil {
				return err
			}
			pkg.Manifest = fetchedPkg.Manifest
		}

		pkg.SetPath(cachePath)
		return nil
	})

	if err != nil {
		return err
	}

	// Security scan
	if !skipSecurity {
		logger.Info("Running security scans...")
		scanner := security.NewScanner()

		for _, pkg := range packages {
			logger.Debug("Scanning %s for security issues", pkg.Name)
			issues, err := scanner.Scan(pkg.GetPath())
			if err != nil {
				logger.Warn("Security scan failed for %s: %v", pkg.Name, err)
				continue
			}

			if len(issues) > 0 {
				logger.Warn("Security issues in %s:", pkg.Name)
				for _, issue := range issues {
					logger.Warn("  [%s] %s:%d - %s", issue.Severity, filepath.Base(issue.File), issue.Line, issue.Message)
				}

				if security.HasCritical(issues) {
					return fmt.Errorf("critical security issues found in %s, aborting install", pkg.Name)
				}
			}
		}
	}

	// Install packages
	inst := installer.New(cfg.GlobalDir, ".claude")

	for _, pkg := range packages {
		logger.Info("Installing %s@%s...", pkg.Name, pkg.Version)

		// Install global targets
		if len(pkg.Manifest.Install.Global) > 0 {
			logger.Debug("Installing %d global targets for %s", len(pkg.Manifest.Install.Global), pkg.Name)
			if err := inst.Install(pkg.GetPath(), &pkg.Manifest, "global", vars); err != nil {
				return fmt.Errorf("failed to install %s globally: %w", pkg.Name, err)
			}
		}

		// Install repo targets
		if len(pkg.Manifest.Install.Repo) > 0 {
			logger.Debug("Installing %d repo targets for %s", len(pkg.Manifest.Install.Repo), pkg.Name)
			if err := inst.Install(pkg.GetPath(), &pkg.Manifest, "repo", vars); err != nil {
				return fmt.Errorf("failed to install %s to repo: %w", pkg.Name, err)
			}
		}

		logger.Success("Installed %s@%s", pkg.Name, pkg.Version)
	}

	// Generate lockfile
	lock, err := lockfile.Generate(resolved)
	if err != nil {
		return fmt.Errorf("failed to generate lockfile: %w", err)
	}

	lockPath := filepath.Join(repoPath, "chitragupta.lock")
	if err := lockfile.Write(lockPath, lock); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	logger.Success("Successfully installed %d packages", len(packages))
	logger.Success("Lockfile saved to %s", lockPath)

	return nil
}
