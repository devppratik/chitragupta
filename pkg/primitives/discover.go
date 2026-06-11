package primitives

import (
	"os"
	"path/filepath"
	"strings"
)

// Discover finds all primitives in a package directory
func Discover(pkgPath, source string) (*PrimitiveSet, error) {
	ps := &PrimitiveSet{}

	// Standard APM structure: .apm/{skills,prompts,instructions,agents,hooks}
	apmDir := filepath.Join(pkgPath, ".apm")
	if _, err := os.Stat(apmDir); err == nil {
		if err := discoverInDir(apmDir, source, ps); err != nil {
			return nil, err
		}
	}

	// Also check root level for backward compatibility
	if err := discoverInDir(pkgPath, source, ps); err != nil {
		return nil, err
	}

	return ps, nil
}

// discoverInDir scans directory for primitives
func discoverInDir(dir, source string, ps *PrimitiveSet) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Detect primitive type by file name/path
		pType, name := detectPrimitive(path, dir)
		if pType == "" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		ps.Add(Primitive{
			Type:     pType,
			Name:     name,
			FilePath: path,
			Content:  string(content),
			Source:   source,
		})

		return nil
	})
}

// detectPrimitive identifies primitive type from file path
func detectPrimitive(path, baseDir string) (PrimitiveType, string) {
	relPath, _ := filepath.Rel(baseDir, path)
	fileName := filepath.Base(path)

	// Skills: SKILL.md or .apm/skills/*.md
	if fileName == "SKILL.md" || strings.Contains(relPath, "skills/") && strings.HasSuffix(fileName, ".md") {
		return TypeSkill, strings.TrimSuffix(fileName, ".md")
	}

	// Prompts: *.prompt.md
	if strings.HasSuffix(fileName, ".prompt.md") {
		return TypePrompt, strings.TrimSuffix(fileName, ".prompt.md")
	}

	// Instructions: *.instructions.md
	if strings.HasSuffix(fileName, ".instructions.md") {
		return TypeInstruction, strings.TrimSuffix(fileName, ".instructions.md")
	}

	// Agents: *.agent.md
	if strings.HasSuffix(fileName, ".agent.md") {
		return TypeAgent, strings.TrimSuffix(fileName, ".agent.md")
	}

	// Hooks: .apm/hooks/* (shell scripts)
	if strings.Contains(relPath, "hooks/") && (strings.HasSuffix(fileName, ".sh") || strings.HasSuffix(fileName, ".bash")) {
		return TypeHook, strings.TrimSuffix(fileName, filepath.Ext(fileName))
	}

	// MCP: mcp-server.json or .apm/mcp/*.json
	if strings.Contains(relPath, "mcp/") && strings.HasSuffix(fileName, ".json") {
		return TypeMCP, strings.TrimSuffix(fileName, ".json")
	}

	// Plugin: plugin.json
	if fileName == "plugin.json" {
		return TypePlugin, "plugin"
	}

	return "", ""
}
