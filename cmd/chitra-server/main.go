package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/ppanda/chitragupta/pkg/logger"
	"github.com/ppanda/chitragupta/pkg/server/db"
	"github.com/ppanda/chitragupta/pkg/server/handlers"
)

func main() {
	// Configuration from environment
	dbType := getEnv("DB_TYPE", "sqlite")
	dbDSN := getEnv("DB_DSN", "chitragupta.db")
	storagePath := getEnv("STORAGE_PATH", "./storage/packages")
	port := getEnv("PORT", "8080")
	logLevel := getEnv("LOG_LEVEL", "info")

	// Set log level
	logger.SetLevel(logger.ParseLevel(logLevel))

	// Validate storage path is absolute
	if !filepath.IsAbs(storagePath) {
		absPath, err := filepath.Abs(storagePath)
		if err != nil {
			logger.Fatal("Failed to resolve storage path to absolute: %v", err)
		}
		storagePath = absPath
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		logger.Fatal("Failed to create storage directory: %v", err)
	}

	// Verify storage path is accessible
	if stat, err := os.Stat(storagePath); err != nil {
		logger.Fatal("Storage path is not accessible: %v", err)
	} else if !stat.IsDir() {
		logger.Fatal("Storage path is not a directory: %s", storagePath)
	}

	// Connect to database
	logger.Debug("Connecting to database: %s (%s)", dbType, dbDSN)
	database, err := db.Connect(db.Config{
		Type: dbType,
		DSN:  dbDSN,
	})
	if err != nil {
		logger.Fatal("Failed to connect to database: %v", err)
	}

	// Setup graceful shutdown for DB connection
	setupGracefulShutdown(database)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Chitra Registry",
	})

	// Middleware
	if logLevel == "debug" {
		app.Use(fiberlogger.New())
	}
	app.Use(cors.New())

	// Initialize handlers
	pkgHandler := handlers.NewPackageHandler(database, storagePath)

	// Routes
	api := app.Group("/api/v1")

	// Package routes
	api.Post("/packages", pkgHandler.Publish)
	api.Get("/packages", pkgHandler.List)
	api.Get("/packages/search", pkgHandler.Search)
	api.Get("/packages/:name", pkgHandler.Get)
	api.Get("/packages/:name/:version", pkgHandler.Get)
	api.Get("/packages/:name/:version/download", pkgHandler.Download)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"name":   "chitra-registry",
		})
	})

	// Start server
	addr := fmt.Sprintf(":%s", port)
	logger.Info("Starting Chitra Registry on %s", addr)
	logger.Info("Database: %s (%s)", dbType, dbDSN)
	logger.Info("Storage: %s", storagePath)

	if err := app.Listen(addr); err != nil {
		logger.Fatal("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupGracefulShutdown(database interface{}) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logger.Info("Shutting down gracefully...")

		// Close database connection
		if db, ok := database.(interface{ DB() (*sql.DB, error) }); ok {
			if sqlDB, err := db.DB(); err == nil {
				sqlDB.Close()
				logger.Info("Database connection closed")
			}
		}

		os.Exit(0)
	}()
}
