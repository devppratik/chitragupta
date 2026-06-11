package models

import (
	"time"

	"gorm.io/gorm"
)

// Package represents a package in the registry
type Package struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	Name        string         `gorm:"uniqueIndex:idx_name_version;not null" json:"name"`
	Version     string         `gorm:"uniqueIndex:idx_name_version;not null" json:"version"`
	Description string         `json:"description"`
	Author      string         `json:"author"`
	License     string         `json:"license"`
	Homepage    string         `json:"homepage"`
	StoragePath string         `json:"-"` // S3 key or file path
	Downloads   int64          `json:"downloads"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Dependency represents a package dependency
type Dependency struct {
	ID                uint    `gorm:"primaryKey"`
	PackageID         uint    `gorm:"index;not null"`
	DependencyName    string  `gorm:"not null"`
	VersionConstraint string  `gorm:"not null"`
	Package           Package `gorm:"foreignKey:PackageID"`
}
