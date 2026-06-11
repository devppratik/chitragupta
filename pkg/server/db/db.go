package db

import (
	"fmt"

	"github.com/ppanda/chitragupta/pkg/server/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Config holds database configuration
type Config struct {
	Type string // "sqlite" or "postgres"
	DSN  string // connection string
}

// Connect establishes database connection
func Connect(cfg Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.Type {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN)
	case "postgres":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate schemas
	if err := db.AutoMigrate(&models.Package{}, &models.Dependency{}); err != nil {
		return nil, err
	}

	return db, nil
}
