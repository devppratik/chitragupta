package types

import "sync"

// Manifest defines package metadata and installation rules
type Manifest struct {
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	Description  string            `yaml:"description"`
	Author       string            `yaml:"author,omitempty"`
	License      string            `yaml:"license,omitempty"`
	Homepage     string            `yaml:"homepage,omitempty"`
	Files        []string          `yaml:"files,omitempty"` // Files to include in package (defaults to all)
	Dependencies map[string]string `yaml:"dependencies,omitempty"`
	Install      InstallRules      `yaml:"install"`
}

// InstallRules defines where files are installed
type InstallRules struct {
	Global []InstallTarget `yaml:"global,omitempty"`
	Repo   []InstallTarget `yaml:"repo,omitempty"`
}

// InstallTarget specifies source, destination, and template rendering
type InstallTarget struct {
	Src      string   `yaml:"src"`
	Dest     string   `yaml:"dest"`
	Template bool     `yaml:"template,omitempty"`
	Vars     []string `yaml:"vars,omitempty"`
}

// Package represents an installed package
type Package struct {
	mu        sync.RWMutex
	Name      string
	Version   string
	Scope     string // "global" or "repo"
	path      string
	Manifest  Manifest
	Installed bool
}

// GetPath safely reads Path field
func (p *Package) GetPath() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.path
}

// SetPath safely writes Path field
func (p *Package) SetPath(path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.path = path
}

// Path is deprecated - use GetPath/SetPath for thread-safe access
// Kept for backwards compatibility but may cause data races
func (p *Package) Path() string {
	return p.GetPath()
}
