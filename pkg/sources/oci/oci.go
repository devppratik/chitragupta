package oci

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ppanda/chitragupta/pkg/manifest"
	"github.com/ppanda/chitragupta/pkg/pathutil"
	"github.com/ppanda/chitragupta/pkg/types"
)

// OCISource handles OCI registry sources (Docker, GHCR, ECR)
type OCISource struct{}

// NewOCISource creates OCI source handler
func NewOCISource() *OCISource {
	return &OCISource{}
}

// Fetch pulls OCI image and extracts package
func (o *OCISource) Fetch(spec string, dest string) (*types.Package, error) {
	image, tag := parseOCISpec(spec)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "chitra-oci-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	// Pull image using docker/podman
	tarPath := filepath.Join(tempDir, "image.tar")
	if err := pullImage(image, tag, tarPath); err != nil {
		return nil, err
	}

	// Extract tarball
	extractDir := filepath.Join(tempDir, "extracted")
	if err := extractTar(tarPath, extractDir); err != nil {
		return nil, err
	}

	// Find manifest in extracted content
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
func (o *OCISource) Resolve(spec string) (*types.Package, error) {
	image, tag := parseOCISpec(spec)

	return &types.Package{
		Name:    extractImageName(image),
		Version: tag,
	}, nil
}

// parseOCISpec extracts image and tag
// Examples:
//
//	ghcr.io/org/pkg:v1.0.0
//	docker.io/org/pkg:latest
func parseOCISpec(spec string) (image, tag string) {
	parts := strings.SplitN(spec, ":", 2)
	image = parts[0]
	tag = "latest"
	if len(parts) > 1 {
		tag = parts[1]
	}
	return image, tag
}

// pullImage pulls OCI image using available runtime
func pullImage(image, tag string, output string) error {
	fullImage := fmt.Sprintf("%s:%s", image, tag)

	// Try docker first, fallback to podman
	runtimes := []string{"docker", "podman"}

	for _, runtime := range runtimes {
		if _, err := exec.LookPath(runtime); err != nil {
			continue
		}

		// Pull image
		pullCmd := exec.Command(runtime, "pull", fullImage)
		pullCmd.Stderr = os.Stderr
		if err := pullCmd.Run(); err != nil {
			continue
		}

		// Save to tarball
		saveCmd := exec.Command(runtime, "save", "-o", output, fullImage)
		saveCmd.Stderr = os.Stderr
		if err := saveCmd.Run(); err != nil {
			continue
		}

		return nil
	}

	return fmt.Errorf("no OCI runtime found (docker or podman required)")
}

// extractTar extracts tar.gz file
func extractTar(tarPath, dest string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	tr := tar.NewReader(f)

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

// findManifest searches for manifest.yaml or chitragupta.yml
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
		return "", fmt.Errorf("manifest not found in OCI image")
	}

	return manifestPath, nil
}

// extractImageName gets package name from image URL
func extractImageName(image string) string {
	parts := strings.Split(image, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
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
