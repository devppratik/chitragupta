package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ppanda/chitragupta/pkg/types"
)

// Installer handles package installation
type Installer struct {
	globalDir string
	repoDir   string
}

// New creates a new installer
func New(globalDir, repoDir string) *Installer {
	return &Installer{
		globalDir: globalDir,
		repoDir:   repoDir,
	}
}

// Install installs a package to the specified scope
func (i *Installer) Install(pkgPath string, manifest *types.Manifest, scope string, vars map[string]string) error {
	if pkgPath == "" {
		return fmt.Errorf("pkgPath cannot be empty")
	}
	if manifest == nil {
		return fmt.Errorf("manifest cannot be nil")
	}

	var targets []types.InstallTarget
	var baseDir string

	switch scope {
	case "global":
		targets = manifest.Install.Global
		baseDir = i.globalDir
	case "repo":
		targets = manifest.Install.Repo
		baseDir = i.repoDir
	default:
		return fmt.Errorf("invalid scope: %s", scope)
	}

	for _, target := range targets {
		if err := i.installTarget(pkgPath, baseDir, target, vars); err != nil {
			return fmt.Errorf("failed to install %s: %w", target.Src, err)
		}
	}

	return nil
}

// installTarget installs a single target
func (i *Installer) installTarget(pkgPath, baseDir string, target types.InstallTarget, vars map[string]string) error {
	srcPattern := filepath.Join(pkgPath, target.Src)

	// Handle glob patterns
	matches, err := filepath.Glob(srcPattern)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return fmt.Errorf("no files match pattern: %s", target.Src)
	}

	for _, srcPath := range matches {
		info, err := os.Stat(srcPath)
		if err != nil {
			return err
		}

		// For single file, use basename only
		// For glob patterns, preserve relative structure
		var destPath string
		if filepath.Base(target.Src) == "*" || strings.Contains(target.Src, "*") {
			// Glob pattern - compute relative path from actual matched file
			// Remove glob wildcards from pattern to get base directory
			basePattern := target.Src
			for strings.Contains(basePattern, "*") {
				basePattern = filepath.Dir(basePattern)
			}
			basePath := filepath.Join(pkgPath, basePattern)

			relPath, err := filepath.Rel(basePath, srcPath)
			if err != nil {
				return err
			}
			destPath = filepath.Join(baseDir, target.Dest, relPath)
		} else {
			// Single file/dir - use basename
			destPath = filepath.Join(baseDir, target.Dest)
		}

		// Handle directories recursively
		if info.IsDir() {
			if err := i.copyDir(srcPath, destPath, target.Template, vars); err != nil {
				return err
			}
			continue
		}

		// Create destination directory
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Copy file
		if target.Template {
			if err := i.copyTemplate(srcPath, destPath, vars); err != nil {
				return err
			}
		} else {
			if err := i.copyFile(srcPath, destPath); err != nil {
				return err
			}
		}

		// Preserve executable permissions
		if err := i.preservePerms(srcPath, destPath); err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies a regular file
func (i *Installer) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := srcFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dstFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// copyTemplate copies and renders a template file
func (i *Installer) copyTemplate(src, dst string, vars map[string]string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	rendered := string(content)
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		rendered = strings.ReplaceAll(rendered, placeholder, value)
	}

	return os.WriteFile(dst, []byte(rendered), 0644)
}

// preservePerms copies executable permissions from src to dst
func (i *Installer) preservePerms(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies directory
func (i *Installer) copyDir(src, dst string, template bool, vars map[string]string) error {
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

		// Create parent dir
		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}

		// Copy file
		if template {
			if err := i.copyTemplate(path, dstPath, vars); err != nil {
				return err
			}
		} else {
			if err := i.copyFile(path, dstPath); err != nil {
				return err
			}
		}

		// Preserve permissions
		return os.Chmod(dstPath, info.Mode())
	})
}
