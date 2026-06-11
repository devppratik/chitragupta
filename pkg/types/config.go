package types

// Config represents chitragupta.yml manifest
type Config struct {
	Name         string                 `yaml:"name"`
	Version      string                 `yaml:"version"`
	Description  string                 `yaml:"description,omitempty"`
	Author       string                 `yaml:"author,omitempty"`
	License      string                 `yaml:"license,omitempty"`
	Homepage     string                 `yaml:"homepage,omitempty"`
	Dependencies ConfigDependencies     `yaml:"dependencies,omitempty"`
	DevDeps      ConfigDependencies     `yaml:"devDependencies,omitempty"`
	Workspaces   []string               `yaml:"workspaces,omitempty"`
	Extends      string                 `yaml:"extends,omitempty"`
	Metadata     map[string]interface{} `yaml:"metadata,omitempty"`
}

// ConfigDependencies defines multi-source dependencies
type ConfigDependencies struct {
	Registry []string        `yaml:"registry,omitempty"` // pkg@version
	Git      []string        `yaml:"git,omitempty"`      // github.com/org/repo#ref
	OCI      []string        `yaml:"oci,omitempty"`      // ghcr.io/org/pkg:tag
	HTTP     []string        `yaml:"http,omitempty"`     // https://url/pkg.tar.gz
	APM      []string        `yaml:"apm,omitempty"`      // APM-compatible git sources
	MCP      []MCPDependency `yaml:"mcp,omitempty"`      // MCP servers
}

// MCPDependency represents an MCP server dependency
type MCPDependency struct {
	Name      string                 `yaml:"name"`
	Transport string                 `yaml:"transport"`
	Config    map[string]interface{} `yaml:"config,omitempty"`
}

// Lockfile represents chitragupta.lock
type Lockfile struct {
	Version      int                  `yaml:"version"` // lockfile format version
	Generated    string               `yaml:"generated"`
	Dependencies map[string]LockEntry `yaml:"dependencies"`
}

// LockEntry represents a locked dependency
type LockEntry struct {
	Name         string   `yaml:"name"`
	Version      string   `yaml:"version"`
	Source       string   `yaml:"source"`                 // registry, git, oci, http
	SourceURL    string   `yaml:"source_url"`             // full URL/path
	Resolved     string   `yaml:"resolved"`               // exact version/commit
	Integrity    string   `yaml:"integrity"`              // sha256 hash
	Dependencies []string `yaml:"dependencies,omitempty"` // transitive deps
}
