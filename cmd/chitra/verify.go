package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppanda/chitragupta/pkg/config"
	"github.com/ppanda/chitragupta/pkg/lockfile"
	"github.com/spf13/cobra"
)

func verifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify manifest, lockfile, and installed packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify()
		},
	}
}

func runVerify() error {
	fmt.Println("🔍 Verifying configuration...")

	// Find manifest
	manifestPath, err := config.FindConfig()
	if err != nil {
		return fmt.Errorf("no chitragupta.yml found: %w", err)
	}

	fmt.Printf("✓ Found manifest: %s\n", manifestPath)

	// Parse manifest
	manifest, err := config.Parse(manifestPath)
	if err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	fmt.Printf("✓ Manifest valid: %s v%s\n", manifest.Name, manifest.Version)

	// Check lockfile
	lockPath := filepath.Join(filepath.Dir(manifestPath), "chitragupta.lock")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		fmt.Println("⚠️  No lockfile found. Run `chitra install` to generate.")
		return nil
	}

	lock, err := lockfile.Read(lockPath)
	if err != nil {
		return fmt.Errorf("invalid lockfile: %w", err)
	}

	fmt.Printf("✓ Lockfile valid (%d dependencies)\n", len(lock.Dependencies))

	// Verify packages in cache
	for name, entry := range lock.Dependencies {
		if _, err := os.Stat(entry.SourceURL); os.IsNotExist(err) {
			return fmt.Errorf("package %s missing (expected at %s)", name, entry.SourceURL)
		}
	}

	fmt.Println("✓ All packages present in cache")
	fmt.Println("\n✅ All checks passed")

	return nil
}
