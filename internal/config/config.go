package config

import (
	"os"
	"path/filepath"
)

// Config holds CLI configuration
type Config struct {
	RegistryPath string
	GlobalDir    string
	ConfigPath   string
}

// Default returns default configuration
func Default() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	chitraDir := filepath.Join(home, ".chitra")

	return &Config{
		RegistryPath: filepath.Join(chitraDir, "registry"),
		GlobalDir:    filepath.Join(home, ".claude"),
		ConfigPath:   filepath.Join(chitraDir, "config.yaml"),
	}, nil
}

// EnsureDirs creates necessary directories
func (c *Config) EnsureDirs() error {
	dirs := []string{
		c.RegistryPath,
		filepath.Join(c.RegistryPath, "packages"),
		c.GlobalDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
