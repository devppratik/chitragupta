package git

import (
	"os"
	"path/filepath"

	"github.com/ppanda/chitragupta/pkg/primitives"
	"github.com/ppanda/chitragupta/pkg/types"
)

// AutoDetectManifest creates virtual manifest from repo structure
func AutoDetectManifest(repoPath string) (*types.Manifest, error) {
	// Check for APM structure (.apm/ directory)
	apmDir := filepath.Join(repoPath, ".apm")
	if _, err := os.Stat(apmDir); err == nil {
		return detectAPMStructure(repoPath)
	}

	// Check for raw primitives (skills/, prompts/, etc)
	return detectRawStructure(repoPath)
}

// detectAPMStructure reads APM-format repos
func detectAPMStructure(repoPath string) (*types.Manifest, error) {
	// Discover primitives in .apm/
	ps, err := primitives.Discover(filepath.Join(repoPath, ".apm"), "local")
	if err != nil {
		return nil, err
	}

	// Extract name from directory
	name := filepath.Base(repoPath)

	manifest := &types.Manifest{
		Name:        name,
		Version:     "0.0.0", // Auto-detected version
		Description: "Auto-detected from APM structure",
		Install: types.InstallRules{
			Global: []types.InstallTarget{},
			Repo:   []types.InstallTarget{},
		},
	}

	// Map primitives to install targets
	if len(ps.Skills) > 0 {
		manifest.Install.Global = append(manifest.Install.Global, types.InstallTarget{
			Src:  ".apm/skills/*",
			Dest: "skills/",
		})
	}

	if len(ps.Prompts) > 0 {
		manifest.Install.Global = append(manifest.Install.Global, types.InstallTarget{
			Src:  ".apm/prompts/*",
			Dest: "prompts/",
		})
	}

	if len(ps.Instructions) > 0 {
		manifest.Install.Global = append(manifest.Install.Global, types.InstallTarget{
			Src:  ".apm/instructions/*",
			Dest: "instructions/",
		})
	}

	if len(ps.Agents) > 0 {
		manifest.Install.Global = append(manifest.Install.Global, types.InstallTarget{
			Src:  ".apm/agents/*",
			Dest: "agents/",
		})
	}

	if len(ps.Hooks) > 0 {
		manifest.Install.Repo = append(manifest.Install.Repo, types.InstallTarget{
			Src:  ".apm/hooks/*",
			Dest: "hooks/",
		})
	}

	return manifest, nil
}

// detectRawStructure reads raw file repos
func detectRawStructure(repoPath string) (*types.Manifest, error) {
	name := filepath.Base(repoPath)

	manifest := &types.Manifest{
		Name:        name,
		Version:     "0.0.0",
		Description: "Auto-detected from raw structure",
		Install: types.InstallRules{
			Global: []types.InstallTarget{},
			Repo:   []types.InstallTarget{},
		},
	}

	// Check for common directories
	dirs := []struct {
		path   string
		dest   string
		global bool
	}{
		{"skills", "skills/", true},
		{"prompts", "prompts/", true},
		{"instructions", "instructions/", true},
		{"agents", "agents/", true},
		{"hooks", "hooks/", false},
		{"tools", "tools/", true},
	}

	for _, d := range dirs {
		dirPath := filepath.Join(repoPath, d.path)
		if _, err := os.Stat(dirPath); err == nil {
			target := types.InstallTarget{
				Src:  d.path + "/*",
				Dest: d.dest,
			}

			if d.global {
				manifest.Install.Global = append(manifest.Install.Global, target)
			} else {
				manifest.Install.Repo = append(manifest.Install.Repo, target)
			}
		}
	}

	// If no structure found, install entire repo as-is
	if len(manifest.Install.Global) == 0 && len(manifest.Install.Repo) == 0 {
		manifest.Install.Repo = append(manifest.Install.Repo, types.InstallTarget{
			Src:  "*",
			Dest: name + "/",
		})
	}

	return manifest, nil
}
