package sources

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppanda/chitragupta/pkg/registry"
	"github.com/ppanda/chitragupta/pkg/sources/git"
	"github.com/ppanda/chitragupta/pkg/sources/http"
	"github.com/ppanda/chitragupta/pkg/sources/oci"
	"github.com/ppanda/chitragupta/pkg/types"
)

// MultiSourceResolver handles all source types
type MultiSourceResolver struct {
	registry   *registry.LocalRegistry
	gitSource  *git.GitSource
	ociSource  *oci.OCISource
	httpSource *http.HTTPSource
}

// NewMultiSourceResolver creates resolver
func NewMultiSourceResolver(registryPath string) *MultiSourceResolver {
	return &MultiSourceResolver{
		registry:   registry.NewLocal(registryPath),
		gitSource:  git.NewGitSource(),
		ociSource:  oci.NewOCISource(),
		httpSource: http.NewHTTPSource(),
	}
}

// Fetch downloads package from any source
func (m *MultiSourceResolver) Fetch(spec string, dest string) (*types.Package, error) {
	sourceType := DetectSource(spec)

	switch sourceType {
	case SourceRegistry:
		return m.fetchFromRegistry(spec, dest)
	case SourceGit, SourceAPM:
		return m.gitSource.Fetch(spec, dest)
	case SourceOCI:
		return m.ociSource.Fetch(spec, dest)
	case SourceHTTP:
		return m.httpSource.Fetch(spec, dest)
	default:
		return nil, fmt.Errorf("unknown source type for spec: %s", spec)
	}
}

// Resolve gets package metadata
func (m *MultiSourceResolver) Resolve(spec string) (*types.Package, error) {
	sourceType := DetectSource(spec)

	switch sourceType {
	case SourceRegistry:
		return m.resolveFromRegistry(spec)
	case SourceGit, SourceAPM:
		return m.gitSource.Resolve(spec)
	case SourceOCI:
		return m.ociSource.Resolve(spec)
	case SourceHTTP:
		return m.httpSource.Resolve(spec)
	default:
		return nil, fmt.Errorf("unknown source type for spec: %s", spec)
	}
}

// fetchFromRegistry gets package from local registry
func (m *MultiSourceResolver) fetchFromRegistry(spec, dest string) (*types.Package, error) {
	name, version := parseRegistrySpec(spec)

	pkgPath, manifest, err := m.registry.Get(name, version)
	if err != nil {
		return nil, err
	}

	// Copy files based on manifest.Files field
	if err := copyPackageFiles(pkgPath, dest, manifest.Files); err != nil {
		return nil, err
	}

	pkg := &types.Package{
		Name:     manifest.Name,
		Version:  manifest.Version,
		Manifest: *manifest,
	}
	pkg.SetPath(dest)
	return pkg, nil
}

// resolveFromRegistry gets metadata from registry
func (m *MultiSourceResolver) resolveFromRegistry(spec string) (*types.Package, error) {
	name, version := parseRegistrySpec(spec)

	_, manifest, err := m.registry.Get(name, version)
	if err != nil {
		return nil, err
	}

	return &types.Package{
		Name:     manifest.Name,
		Version:  manifest.Version,
		Manifest: *manifest,
	}, nil
}

// parseRegistrySpec extracts name and version from pkg@version
func parseRegistrySpec(spec string) (name, version string) {
	// Format: package@version or package
	for i := len(spec) - 1; i >= 0; i-- {
		if spec[i] == '@' {
			// Validate name is not empty (@ cannot be at position 0)
			if i == 0 {
				break
			}
			return spec[:i], spec[i+1:]
		}
	}
	return spec, "latest"
}

// copyDir recursively copies directory
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

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, content, info.Mode())
	})
}

// copyPackageFiles copies files based on manifest files field
func copyPackageFiles(src, dst string, files []string) error {
	// If no files specified, copy everything
	if len(files) == 0 {
		return copyDir(src, dst)
	}

	// Copy only specified files/patterns
	for _, pattern := range files {
		matches, err := filepath.Glob(filepath.Join(src, pattern))
		if err != nil {
			return err
		}

		for _, match := range matches {
			relPath, err := filepath.Rel(src, match)
			if err != nil {
				return err
			}

			dstPath := filepath.Join(dst, relPath)
			info, err := os.Stat(match)
			if err != nil {
				return err
			}

			if info.IsDir() {
				// Copy directory recursively
				if err := copyDir(match, dstPath); err != nil {
					return err
				}
			} else {
				// Copy single file
				if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
					return err
				}
				content, err := os.ReadFile(match)
				if err != nil {
					return err
				}
				if err := os.WriteFile(dstPath, content, info.Mode()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
