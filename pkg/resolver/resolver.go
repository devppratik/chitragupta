package resolver

import (
	"fmt"
	"strings"

	"github.com/ppanda/chitragupta/pkg/types"
)

// Resolver handles dependency resolution
type Resolver struct {
	registry Registry
}

// Registry interface for package lookup
type Registry interface {
	Get(name, version string) (string, *types.Manifest, error)
}

// New creates a new resolver
func New(registry Registry) *Resolver {
	return &Resolver{registry: registry}
}

// Resolve returns dependency tree in installation order
func (r *Resolver) Resolve(name, version string) ([]*types.Package, error) {
	resolved := make(map[string]*types.Package)
	var order []*types.Package

	if err := r.resolveDeps(name, version, resolved, &order); err != nil {
		return nil, err
	}

	return order, nil
}

// resolveDeps recursively resolves dependencies
func (r *Resolver) resolveDeps(name, version string, resolved map[string]*types.Package, order *[]*types.Package) error {
	key := fmt.Sprintf("%s@%s", name, version)

	// Already resolved
	if _, ok := resolved[key]; ok {
		return nil
	}

	// Get package from registry
	pkgPath, manifest, err := r.registry.Get(name, version)
	if err != nil {
		return err
	}

	pkg := &types.Package{
		Name:     manifest.Name,
		Version:  manifest.Version,
		Manifest: *manifest,
	}
	pkg.SetPath(pkgPath)

	// Resolve dependencies first
	for depName, depVersion := range manifest.Dependencies {
		// Parse version constraint (basic support for now)
		actualVersion := parseVersionConstraint(depVersion)

		if err := r.resolveDeps(depName, actualVersion, resolved, order); err != nil {
			return fmt.Errorf("failed to resolve dependency %s: %w", depName, err)
		}
	}

	// Add to resolved and order
	resolved[key] = pkg
	*order = append(*order, pkg)

	return nil
}

// parseVersionConstraint extracts version from constraint
func parseVersionConstraint(constraint string) string {
	// Use semver package for proper parsing
	// For now, keep simple trimming for backward compat
	constraint = strings.TrimLeft(constraint, "^~><=")
	return constraint
}
