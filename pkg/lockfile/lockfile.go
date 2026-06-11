package lockfile

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ppanda/chitragupta/pkg/types"
	"gopkg.in/yaml.v3"
)

const LockfileVersion = 1

// Generate creates lockfile from resolved dependencies
func Generate(deps map[string]*types.Package) (*types.Lockfile, error) {
	lock := &types.Lockfile{
		Version:      LockfileVersion,
		Generated:    time.Now().UTC().Format(time.RFC3339),
		Dependencies: make(map[string]types.LockEntry),
	}

	for name, pkg := range deps {
		// Calculate integrity hash
		pkgPath := pkg.GetPath()
		integrity, err := calculateHash(pkgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to hash %s: %w", name, err)
		}

		lock.Dependencies[name] = types.LockEntry{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Source:    pkg.Scope, // reuse scope field for source type
			SourceURL: pkgPath,
			Resolved:  pkg.Version,
			Integrity: integrity,
		}
	}

	return lock, nil
}

// Write saves lockfile to disk
func Write(path string, lock *types.Lockfile) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Read loads lockfile from disk
func Read(path string) (*types.Lockfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lock types.Lockfile
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, err
	}

	if lock.Version != LockfileVersion {
		return nil, fmt.Errorf("unsupported lockfile version: %d", lock.Version)
	}

	return &lock, nil
}

// Verify checks if installed packages match lockfile
func Verify(lock *types.Lockfile, installedPath string) error {
	for name, entry := range lock.Dependencies {
		pkgPath := fmt.Sprintf("%s/%s", installedPath, name)

		// Check if package exists
		if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
			return fmt.Errorf("package %s missing", name)
		}

		// Verify integrity
		hash, err := calculateHash(pkgPath)
		if err != nil {
			return fmt.Errorf("failed to verify %s: %w", name, err)
		}

		if hash != entry.Integrity {
			return fmt.Errorf("integrity mismatch for %s (expected %s, got %s)", name, entry.Integrity, hash)
		}
	}

	return nil
}

// calculateHash computes SHA256 hash of package directory
func calculateHash(path string) (string, error) {
	h := sha256.New()

	// Walk directory tree deterministically
	err := hashDirectory(path, path, h)
	if err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// hashDirectory recursively hashes directory contents in sorted order
func hashDirectory(root, path string, h io.Writer) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	// Sort entries for deterministic ordering
	// Already sorted by ReadDir, but explicit for clarity
	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		relPath, err := filepath.Rel(root, fullPath)
		if err != nil {
			return err
		}

		// Hash relative path
		io.WriteString(h, relPath)

		if entry.IsDir() {
			// Recurse into directory
			if err := hashDirectory(root, fullPath, h); err != nil {
				return err
			}
		} else {
			// Hash file contents + mode
			info, err := entry.Info()
			if err != nil {
				return err
			}

			// Hash file mode
			fmt.Fprintf(h, "%o", info.Mode())

			// Hash file contents - close immediately to avoid FD exhaustion
			if err := func() error {
				f, err := os.Open(fullPath)
				if err != nil {
					return err
				}
				defer f.Close()

				if _, err := io.Copy(h, f); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return err
			}
		}
	}

	return nil
}
