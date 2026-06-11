package template

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Context holds auto-detected template variables
type Context struct {
	RepoName   string
	Language   string
	TeamName   string
	CIProvider string
	Framework  string
	Custom     map[string]string
}

// Detect auto-detects template context from repository
func Detect(repoPath string) (*Context, error) {
	ctx := &Context{
		Custom: make(map[string]string),
	}

	// Detect repo name from git or directory
	ctx.RepoName = detectRepoName(repoPath)

	// Detect primary language
	ctx.Language = detectLanguage(repoPath)

	// Detect team from CODEOWNERS
	ctx.TeamName = detectTeam(repoPath)

	// Detect CI provider
	ctx.CIProvider = detectCI(repoPath)

	// Detect framework
	ctx.Framework = detectFramework(repoPath)

	return ctx, nil
}

// ToMap converts context to var map
func (c *Context) ToMap() map[string]string {
	m := make(map[string]string)
	m["REPO_NAME"] = c.RepoName
	m["LANGUAGE"] = c.Language
	m["TEAM_NAME"] = c.TeamName
	m["CI_PROVIDER"] = c.CIProvider
	m["FRAMEWORK"] = c.Framework

	for k, v := range c.Custom {
		m[k] = v
	}

	return m
}

// detectRepoName gets repository name
func detectRepoName(path string) string {
	// Try git remote
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = path
	output, err := cmd.Output()
	if err == nil {
		url := strings.TrimSpace(string(output))
		// Extract repo name from URL
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			name = strings.TrimSuffix(name, ".git")
			if name != "" {
				return name
			}
		}
	}

	// Fallback to directory name
	return filepath.Base(path)
}

// detectLanguage identifies primary language
func detectLanguage(path string) string {
	// Count files by extension
	counts := make(map[string]int)

	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(p))
		if ext != "" {
			counts[ext]++
		}

		return nil
	})

	// Map extensions to languages
	langMap := map[string]string{
		".py":   "Python",
		".go":   "Go",
		".js":   "JavaScript",
		".ts":   "TypeScript",
		".java": "Java",
		".rb":   "Ruby",
		".rs":   "Rust",
		".php":  "PHP",
		".c":    "C",
		".cpp":  "C++",
	}

	// Find most common
	maxCount := 0
	primaryLang := ""

	for ext, count := range counts {
		if lang, ok := langMap[ext]; ok && count > maxCount {
			maxCount = count
			primaryLang = lang
		}
	}

	return primaryLang
}

// detectTeam extracts team from CODEOWNERS
func detectTeam(path string) string {
	codeownersPath := filepath.Join(path, ".github", "CODEOWNERS")
	if _, err := os.Stat(codeownersPath); os.IsNotExist(err) {
		codeownersPath = filepath.Join(path, "CODEOWNERS")
	}

	content, err := os.ReadFile(codeownersPath)
	if err != nil {
		return ""
	}

	// Parse first team mention (@org/team)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Find @org/team pattern
		if idx := strings.Index(line, "@"); idx >= 0 {
			team := strings.Fields(line[idx:])[0]
			team = strings.TrimPrefix(team, "@")
			return team
		}
	}

	return ""
}

// detectCI identifies CI provider
func detectCI(path string) string {
	ciChecks := map[string]string{
		".github/workflows":    "GitHub Actions",
		".gitlab-ci.yml":       "GitLab CI",
		".circleci/config.yml": "CircleCI",
		".travis.yml":          "Travis CI",
		"Jenkinsfile":          "Jenkins",
		".drone.yml":           "Drone CI",
	}

	for file, provider := range ciChecks {
		checkPath := filepath.Join(path, file)
		if _, err := os.Stat(checkPath); err == nil {
			return provider
		}
	}

	return "Unknown"
}

// detectFramework identifies framework from package files
func detectFramework(path string) string {
	// Check package.json for JS/TS frameworks
	packageJSON := filepath.Join(path, "package.json")
	if content, err := os.ReadFile(packageJSON); err == nil {
		contentStr := string(content)
		frameworks := map[string]string{
			"react":   "React",
			"vue":     "Vue",
			"angular": "Angular",
			"next":    "Next.js",
			"svelte":  "Svelte",
		}

		for keyword, framework := range frameworks {
			if strings.Contains(contentStr, keyword) {
				return framework
			}
		}
	}

	// Check requirements.txt for Python frameworks
	requirementsTxt := filepath.Join(path, "requirements.txt")
	if content, err := os.ReadFile(requirementsTxt); err == nil {
		contentStr := strings.ToLower(string(content))
		frameworks := map[string]string{
			"django":  "Django",
			"flask":   "Flask",
			"fastapi": "FastAPI",
		}

		for keyword, framework := range frameworks {
			if strings.Contains(contentStr, keyword) {
				return framework
			}
		}
	}

	// Check go.mod for Go frameworks
	goMod := filepath.Join(path, "go.mod")
	if content, err := os.ReadFile(goMod); err == nil {
		contentStr := string(content)
		frameworks := map[string]string{
			"gin-gonic/gin": "Gin",
			"gofiber/fiber": "Fiber",
			"labstack/echo": "Echo",
		}

		for keyword, framework := range frameworks {
			if strings.Contains(contentStr, keyword) {
				return framework
			}
		}
	}

	return ""
}
