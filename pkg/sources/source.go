package sources

import (
	"github.com/ppanda/chitragupta/pkg/types"
)

// Source defines interface for package sources
type Source interface {
	// Fetch downloads package to destination directory
	Fetch(spec string, dest string) (*types.Package, error)

	// Resolve gets package metadata without downloading
	Resolve(spec string) (*types.Package, error)
}

// SourceType identifies package source
type SourceType string

const (
	SourceRegistry SourceType = "registry"
	SourceGit      SourceType = "git"
	SourceOCI      SourceType = "oci"
	SourceHTTP     SourceType = "http"
	SourceAPM      SourceType = "apm" // APM-compatible git
)

// DetectSource identifies source type from spec
func DetectSource(spec string) SourceType {
	// registry: package@version or package
	// git: github.com/org/repo#ref, gitlab.com/..., etc
	// oci: ghcr.io/org/pkg:tag, docker.io/..., etc
	// http: https://... or http://...

	if len(spec) >= 8 && spec[:8] == "https://" || len(spec) >= 7 && spec[:7] == "http://" {
		// Could be git or http
		if contains(spec, "github.com") || contains(spec, "gitlab.com") ||
			contains(spec, "bitbucket.org") || contains(spec, "dev.azure.com") {
			return SourceGit
		}
		return SourceHTTP
	}

	if contains(spec, ".io/") || contains(spec, ".azurecr.io/") {
		return SourceOCI
	}

	if contains(spec, "/") && (contains(spec, "#") || contains(spec, "@")) {
		// org/repo#ref or github.com/org/repo
		return SourceGit
	}

	// Default to registry
	return SourceRegistry
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
