package handlers

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/ppanda/chitragupta/pkg/pathutil"
	"github.com/ppanda/chitragupta/pkg/server/models"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

var validPackageNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)

type PackageHandler struct {
	db          *gorm.DB
	storagePath string
}

func NewPackageHandler(db *gorm.DB, storagePath string) *PackageHandler {
	return &PackageHandler{
		db:          db,
		storagePath: storagePath,
	}
}

// Publish handles package upload
func (h *PackageHandler) Publish(c *fiber.Ctx) error {
	file, err := c.FormFile("package")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "package file required",
		})
	}

	// Check file size (100MB limit)
	const maxSize = 100 * 1024 * 1024
	if file.Size > maxSize {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "package too large (max 100MB)",
		})
	}

	// Save uploaded tarball temporarily
	// Sanitize filename to prevent path traversal
	safeFilename := filepath.Base(file.Filename)
	tempPath := filepath.Join(os.TempDir(), safeFilename)
	if err := c.SaveFile(file, tempPath); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save file",
		})
	}
	defer os.Remove(tempPath)

	// Extract and parse manifest
	manifestData, err := h.extractManifest(tempPath)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("invalid package: %v", err),
		})
	}

	var m struct {
		Name         string            `yaml:"name"`
		Version      string            `yaml:"version"`
		Description  string            `yaml:"description"`
		Author       string            `yaml:"author"`
		License      string            `yaml:"license"`
		Homepage     string            `yaml:"homepage"`
		Dependencies map[string]string `yaml:"dependencies"`
	}

	if err := yaml.Unmarshal(manifestData, &m); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid manifest",
		})
	}

	// Validate package name and version to prevent path traversal
	if !validPackageNamePattern.MatchString(m.Name) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid package name (must be alphanumeric with -, _, .)",
		})
	}
	if !validPackageNamePattern.MatchString(m.Version) {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid version (must be alphanumeric with -, _, .)",
		})
	}

	// Check if package version already exists
	var existing models.Package
	if err := h.db.Where("name = ? AND version = ?", m.Name, m.Version).First(&existing).Error; err == nil {
		return c.Status(http.StatusConflict).JSON(fiber.Map{
			"error": "package version already exists",
		})
	}

	// Store package file (validate no path traversal)
	storagePath, err := pathutil.SafeJoin(h.storagePath, m.Name, m.Version, "package.tar.gz")
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("invalid path: %v", err),
		})
	}
	if err := os.MkdirAll(filepath.Dir(storagePath), 0755); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create storage directory",
		})
	}

	if err := copyFile(tempPath, storagePath); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to store package",
		})
	}

	// Create database record
	pkg := models.Package{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Author:      m.Author,
		License:     m.License,
		Homepage:    m.Homepage,
		StoragePath: storagePath,
		Downloads:   0,
	}

	if err := h.db.Create(&pkg).Error; err != nil {
		os.Remove(storagePath)
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to save package metadata",
		})
	}

	// Save dependencies
	for depName, depVersion := range m.Dependencies {
		dep := models.Dependency{
			PackageID:         pkg.ID,
			DependencyName:    depName,
			VersionConstraint: depVersion,
		}
		if err := h.db.Create(&dep).Error; err != nil {
			// Rollback: delete package and storage file
			h.db.Delete(&pkg)
			os.Remove(storagePath)
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to save dependencies: %v", err),
			})
		}
	}

	return c.Status(http.StatusCreated).JSON(pkg)
}

// Get retrieves package metadata
func (h *PackageHandler) Get(c *fiber.Ctx) error {
	name := c.Params("name")
	version := c.Params("version", "latest")

	var pkg models.Package

	if version == "latest" {
		if err := h.db.Where("name = ?", name).Order("created_at DESC").First(&pkg).Error; err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "package not found",
			})
		}
	} else {
		if err := h.db.Where("name = ? AND version = ?", name, version).First(&pkg).Error; err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "package not found",
			})
		}
	}

	return c.JSON(pkg)
}

// Download serves package tarball
func (h *PackageHandler) Download(c *fiber.Ctx) error {
	name := c.Params("name")
	version := c.Params("version", "latest")

	var pkg models.Package

	if version == "latest" {
		if err := h.db.Where("name = ?", name).Order("created_at DESC").First(&pkg).Error; err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "package not found",
			})
		}
	} else {
		if err := h.db.Where("name = ? AND version = ?", name, version).First(&pkg).Error; err != nil {
			return c.Status(http.StatusNotFound).JSON(fiber.Map{
				"error": "package not found",
			})
		}
	}

	// Increment download counter
	if err := h.db.Model(&pkg).UpdateColumn("downloads", gorm.Expr("downloads + 1")).Error; err != nil {
		// Log error but don't fail the download
		// The download counter is not critical to the download operation
		fmt.Printf("Warning: failed to update download counter for %s@%s: %v\n", pkg.Name, pkg.Version, err)
	}

	return c.SendFile(pkg.StoragePath)
}

// List returns all packages
func (h *PackageHandler) List(c *fiber.Ctx) error {
	var packages []models.Package

	if err := h.db.Order("created_at DESC").Find(&packages).Error; err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch packages",
		})
	}

	return c.JSON(packages)
}

// Search finds packages by query
func (h *PackageHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q", "")

	var packages []models.Package

	if query == "" {
		h.db.Order("created_at DESC").Find(&packages)
	} else {
		// Escape SQL LIKE wildcards to prevent injection
		escapedQuery := strings.ReplaceAll(query, "%", "\\%")
		escapedQuery = strings.ReplaceAll(escapedQuery, "_", "\\_")

		h.db.Where("name LIKE ? OR description LIKE ?",
			"%"+escapedQuery+"%", "%"+escapedQuery+"%").Order("created_at DESC").Find(&packages)
	}

	return c.JSON(packages)
}

// extractManifest extracts manifest.yaml from tarball
func (h *PackageHandler) extractManifest(tarballPath string) ([]byte, error) {
	f, err := os.Open(tarballPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(header.Name, "manifest.yaml") {
			// Limit manifest size to 10MB
			const maxManifestSize = 10 * 1024 * 1024
			limited := io.LimitReader(tr, maxManifestSize)
			data, err := io.ReadAll(limited)
			if err != nil {
				return nil, err
			}
			if int64(len(data)) == maxManifestSize {
				return nil, fmt.Errorf("manifest.yaml too large (max 10MB)")
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("manifest.yaml not found in package")
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
