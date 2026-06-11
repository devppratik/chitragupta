package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ppanda/chitragupta/pkg/manifest"
	"github.com/ppanda/chitragupta/pkg/types"
	"gopkg.in/yaml.v3"
)

var validGitRefPattern = regexp.MustCompile(`^[a-zA-Z0-9/_.-]+$`)

// GitSource handles git-based package sources
type GitSource struct{}

// NewGitSource creates git source handler
func NewGitSource() *GitSource {
	return &GitSource{}
}

// Fetch clones git repo and extracts package
func (g *GitSource) Fetch(spec string, dest string) (*types.Package, error) {
	url, ref, subpath := parseGitSpec(spec)

	// Create temp directory for clone
	tempDir, err := os.MkdirTemp("", "chitra-git-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// Clone repository
	cmd := exec.Command("git", "clone", "--depth", "1")
	if ref != "" {
		// Validate ref to prevent command injection
		if !validGitRefPattern.MatchString(ref) {
			return nil, fmt.Errorf("invalid git ref (contains dangerous characters): %s", ref)
		}
		cmd.Args = append(cmd.Args, "--branch", ref)
	}
	cmd.Args = append(cmd.Args, url, tempDir)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git clone failed: %w", err)
	}

	// Resolve subpath
	sourcePath := tempDir
	if subpath != "" {
		sourcePath = filepath.Join(tempDir, subpath)
	}

	// Try to find manifest
	var m *types.Manifest
	manifestPath := filepath.Join(sourcePath, "manifest.yaml")
	if _, err := os.Stat(manifestPath); err == nil {
		m, err = manifest.Parse(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}
	} else {
		// Try chitragupta.yml
		manifestPath = filepath.Join(sourcePath, "chitragupta.yml")
		if _, err := os.Stat(manifestPath); err == nil {
			m, err = manifest.Parse(manifestPath)
			if err != nil {
				return nil, fmt.Errorf("failed to parse manifest: %w", err)
			}
		} else {
			// No manifest found - auto-detect structure
			m, err = AutoDetectManifest(sourcePath)
			if err != nil {
				return nil, fmt.Errorf("failed to auto-detect manifest: %w", err)
			}
		}
	}

	// Copy to destination
	if err := copyDir(sourcePath, dest); err != nil {
		return nil, err
	}

	// Save manifest to destination for lockfile hashing
	manifestBytes, err := yaml.Marshal(m)
	if err == nil {
		if writeErr := os.WriteFile(filepath.Join(dest, "manifest.yaml"), manifestBytes, 0644); writeErr != nil {
			// Log but don't fail - manifest copy is convenience, not critical
			fmt.Printf("Warning: failed to write manifest copy: %v\n", writeErr)
		}
	}

	pkg := &types.Package{
		Name:     m.Name,
		Version:  m.Version,
		Manifest: *m,
	}
	pkg.SetPath(dest)
	return pkg, nil
}

// Resolve gets package metadata without full clone
func (g *GitSource) Resolve(spec string) (*types.Package, error) {
	url, ref, _ := parseGitSpec(spec)

	// Validate ref to prevent command injection
	if ref != "" && !validGitRefPattern.MatchString(ref) {
		return nil, fmt.Errorf("invalid git ref (contains dangerous characters): %s", ref)
	}

	// Use git ls-remote to check if ref exists
	cmd := exec.Command("git", "ls-remote", url, ref)
	output, err := cmd.Output()
	if err != nil {
		// Fallback: try resolving HEAD
		cmd = exec.Command("git", "ls-remote", url, "HEAD")
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("git ls-remote failed: %w", err)
		}
	}

	// Extract commit SHA from output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no refs found for %s", spec)
	}

	// First line format: "<sha>\t<ref>"
	parts := strings.Fields(lines[0])
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid ls-remote output")
	}

	commitSHA := parts[0]

	return &types.Package{
		Name:    extractRepoName(url),
		Version: ref,
		Scope:   commitSHA[:7], // Use short SHA as scope/identifier
	}, nil
}

// parseGitSpec extracts URL, ref, and subpath from git spec
// Examples:
//
//	github.com/org/repo#v1.0.0
//	gitlab.com/org/repo#main
//	github.com/org/repo/subdir#v1.0.0
func parseGitSpec(spec string) (url, ref, subpath string) {
	// Split by #
	parts := strings.SplitN(spec, "#", 2)
	urlPart := parts[0]
	if len(parts) > 1 {
		ref = parts[1]
	} else {
		ref = "main" // default branch
	}

	// Handle subpath (org/repo/subdir)
	urlParts := strings.Split(urlPart, "/")
	if len(urlParts) > 2 {
		// Assume first 2 parts are host/org, 3rd is repo, rest is subpath
		if len(urlParts) > 3 {
			subpath = strings.Join(urlParts[3:], "/")
			urlPart = strings.Join(urlParts[:3], "/")
		}
	}

	// Construct full git URL
	if !strings.HasPrefix(urlPart, "https://") && !strings.HasPrefix(urlPart, "git@") {
		// Assume github.com/org/repo format
		url = "https://" + urlPart
	} else {
		url = urlPart
	}

	// Ensure .git suffix
	if !strings.HasSuffix(url, ".git") {
		url = url + ".git"
	}

	return url, ref, subpath
}

// extractRepoName gets repository name from URL
func extractRepoName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	name := parts[len(parts)-1]
	return strings.TrimSuffix(name, ".git")
}

// copyDir recursively copies directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
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

// copyFile copies a file
func copyFile(src, dst string, mode os.FileMode) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, mode)
}
