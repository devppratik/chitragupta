package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ppanda/chitragupta/pkg/manifest"
	"github.com/ppanda/chitragupta/pkg/semver"
	"github.com/ppanda/chitragupta/pkg/types"
)

// LocalRegistry implements filesystem-based package registry
type LocalRegistry struct {
	path string
}

// NewLocal creates a new local registry
func NewLocal(path string) *LocalRegistry {
	return &LocalRegistry{path: path}
}

// Publish adds a package to the registry
func (r *LocalRegistry) Publish(pkgDir string) error {
	manifestPath := filepath.Join(pkgDir, "manifest.yaml")
	m, err := manifest.Parse(manifestPath)
	if err != nil {
		return err
	}

	// Create package directory in registry
	pkgPath := filepath.Join(r.path, "packages", m.Name, m.Version)
	if err := os.MkdirAll(pkgPath, 0755); err != nil {
		return err
	}

	// Copy entire package directory
	return copyDir(pkgDir, pkgPath)
}

// Get retrieves a package by name and version
func (r *LocalRegistry) Get(name, version string) (string, *types.Manifest, error) {
	if version == "" || version == "latest" {
		version = r.getLatestVersion(name)
		if version == "" {
			return "", nil, fmt.Errorf("package not found: %s", name)
		}
	}

	pkgPath := filepath.Join(r.path, "packages", name, version)
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("package not found: %s@%s", name, version)
	}

	manifestPath := filepath.Join(pkgPath, "manifest.yaml")
	m, err := manifest.Parse(manifestPath)
	if err != nil {
		return "", nil, err
	}

	return pkgPath, m, nil
}

// List returns all packages in the registry
func (r *LocalRegistry) List() ([]types.Package, error) {
	packagesPath := filepath.Join(r.path, "packages")
	if _, err := os.Stat(packagesPath); os.IsNotExist(err) {
		return []types.Package{}, nil
	}

	var packages []types.Package

	// Walk through packages directory
	entries, err := os.ReadDir(packagesPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		versions, err := r.getVersions(name)
		if err != nil {
			continue
		}

		for _, version := range versions {
			pkgPath := filepath.Join(packagesPath, name, version)
			manifestPath := filepath.Join(pkgPath, "manifest.yaml")

			m, err := manifest.Parse(manifestPath)
			if err != nil {
				continue
			}

			pkg := types.Package{
				Name:     m.Name,
				Version:  m.Version,
				Manifest: *m,
			}
			pkg.SetPath(pkgPath)
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

// Search finds packages matching a query
func (r *LocalRegistry) Search(query string) ([]types.Package, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}

	if query == "" {
		return all, nil
	}

	var results []types.Package
	query = strings.ToLower(query)

	for _, pkg := range all {
		if strings.Contains(strings.ToLower(pkg.Name), query) ||
			strings.Contains(strings.ToLower(pkg.Manifest.Description), query) {
			results = append(results, pkg)
		}
	}

	return results, nil
}

// getLatestVersion returns the latest version for a package
func (r *LocalRegistry) getLatestVersion(name string) string {
	versions, err := r.getVersions(name)
	if err != nil || len(versions) == 0 {
		return ""
	}

	// Parse all versions
	var parsedVersions []struct {
		raw string
		ver *semver.Version
	}

	for _, v := range versions {
		sv, err := semver.Parse(v)
		if err != nil {
			// Skip invalid semver
			continue
		}
		parsedVersions = append(parsedVersions, struct {
			raw string
			ver *semver.Version
		}{v, sv})
	}

	if len(parsedVersions) == 0 {
		// No valid semver found, return last alphabetically
		return versions[len(versions)-1]
	}

	// Find max version
	latest := parsedVersions[0]
	for _, pv := range parsedVersions[1:] {
		if pv.ver.Compare(latest.ver) > 0 {
			latest = pv
		}
	}

	return latest.raw
}

// getVersions returns all versions for a package
func (r *LocalRegistry) getVersions(name string) ([]string, error) {
	pkgPath := filepath.Join(r.path, "packages", name)
	entries, err := os.ReadDir(pkgPath)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath, info.Mode())
	})
}

// copyFile copies a file with permissions
func copyFile(src, dst string, mode os.FileMode) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, mode)
}
