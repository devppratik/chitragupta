package http

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ppanda/chitragupta/pkg/manifest"
	"github.com/ppanda/chitragupta/pkg/pathutil"
	"github.com/ppanda/chitragupta/pkg/types"
)

// HTTPSource handles HTTP tarball sources
type HTTPSource struct {
	client *http.Client
}

// NewHTTPSource creates HTTP source handler
func NewHTTPSource() *HTTPSource {
	return &HTTPSource{
		client: &http.Client{},
	}
}

// Fetch downloads HTTP tarball and extracts package
func (h *HTTPSource) Fetch(spec string, dest string) (*types.Package, error) {
	url, expectedHash := parseHTTPSpec(spec)

	// Download tarball
	tempDir, err := os.MkdirTemp("", "chitra-http-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	tarPath := filepath.Join(tempDir, "package.tar.gz")
	actualHash, err := h.download(url, tarPath)
	if err != nil {
		return nil, err
	}

	// Verify integrity if hash provided
	if expectedHash != "" && actualHash != expectedHash {
		return nil, fmt.Errorf("integrity check failed: expected %s, got %s", expectedHash, actualHash)
	}

	// Extract tarball
	extractDir := filepath.Join(tempDir, "extracted")
	if err := extractTarGz(tarPath, extractDir); err != nil {
		return nil, err
	}

	// Find manifest
	manifestPath, err := findManifest(extractDir)
	if err != nil {
		return nil, err
	}

	m, err := manifest.Parse(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Copy to destination
	packageDir := filepath.Dir(manifestPath)
	if err := copyDir(packageDir, dest); err != nil {
		return nil, err
	}

	pkg := &types.Package{
		Name:     m.Name,
		Version:  m.Version,
		Manifest: *m,
	}
	pkg.SetPath(dest)
	return pkg, nil
}

// Resolve gets package metadata
func (h *HTTPSource) Resolve(spec string) (*types.Package, error) {
	url, _ := parseHTTPSpec(spec)

	return &types.Package{
		Name:    extractNameFromURL(url),
		Version: "latest",
	}, nil
}

// download fetches URL and returns SHA256 hash
func (h *HTTPSource) download(url, output string) (string, error) {
	resp, err := h.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	outFile, err := os.Create(output)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	// Calculate hash while downloading
	hash := sha256.New()
	writer := io.MultiWriter(outFile, hash)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// parseHTTPSpec extracts URL and optional integrity hash
// Format: https://example.com/pkg.tar.gz#sha256:abc123
func parseHTTPSpec(spec string) (url, hash string) {
	parts := strings.SplitN(spec, "#", 2)
	url = parts[0]
	if len(parts) > 1 {
		hash = parts[1]
		// Remove sha256: prefix if present
		hash = strings.TrimPrefix(hash, "sha256:")
	}
	return url, hash
}

// extractTarGz extracts .tar.gz file
func extractTarGz(tarPath, dest string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target, err := pathutil.SafeJoin(dest, header.Name)
		if err != nil {
			return fmt.Errorf("path traversal detected: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// findManifest searches for manifest file
func findManifest(root string) (string, error) {
	var manifestPath string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (info.Name() == "manifest.yaml" || info.Name() == "chitragupta.yml") {
			manifestPath = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	if manifestPath == "" {
		return "", fmt.Errorf("manifest not found in tarball")
	}

	return manifestPath, nil
}

// extractNameFromURL gets package name from URL
func extractNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return "unknown"
	}
	name := parts[len(parts)-1]
	// Remove .tar.gz extension
	name = strings.TrimSuffix(name, ".tar.gz")
	name = strings.TrimSuffix(name, ".tgz")
	return name
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
