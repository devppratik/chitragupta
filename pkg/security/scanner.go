package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Issue represents a security issue
type Issue struct {
	Severity string // "critical", "high", "medium", "low"
	Type     string
	File     string
	Line     int
	Message  string
}

// Scanner performs security checks
type Scanner struct {
	issues []Issue
}

// NewScanner creates security scanner
func NewScanner() *Scanner {
	return &Scanner{
		issues: make([]Issue, 0),
	}
}

// Scan performs all security checks on package
func (s *Scanner) Scan(pkgPath string) ([]Issue, error) {
	s.issues = make([]Issue, 0)

	err := filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip non-text files
		if !isTextFile(path) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Run checks
		s.checkHiddenUnicode(path, string(content))
		s.checkSuspiciousPatterns(path, string(content))
		s.checkDangerousCommands(path, string(content))

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.issues, nil
}

// checkHiddenUnicode detects hidden Unicode characters
func (s *Scanner) checkHiddenUnicode(file, content string) {
	// List of dangerous Unicode codepoints
	dangerous := map[rune]string{
		0x200B: "Zero Width Space",
		0x200C: "Zero Width Non-Joiner",
		0x200D: "Zero Width Joiner",
		0x202A: "Left-to-Right Embedding",
		0x202B: "Right-to-Left Embedding",
		0x202C: "Pop Directional Formatting",
		0x202D: "Left-to-Right Override",
		0x202E: "Right-to-Left Override",
		0xFEFF: "Zero Width No-Break Space",
	}

	lines := strings.Split(content, "\n")
	for lineNum, line := range lines {
		for _, char := range line {
			if name, ok := dangerous[char]; ok {
				s.issues = append(s.issues, Issue{
					Severity: "critical",
					Type:     "hidden-unicode",
					File:     file,
					Line:     lineNum + 1,
					Message:  fmt.Sprintf("Hidden Unicode character detected: %s (U+%04X)", name, char),
				})
			}
		}
	}
}

// checkSuspiciousPatterns detects suspicious code patterns
func (s *Scanner) checkSuspiciousPatterns(file, content string) {
	patterns := []struct {
		pattern  string
		severity string
		message  string
	}{
		{"eval(", "high", "Use of eval() detected"},
		{"exec(", "high", "Use of exec() detected"},
		{"__import__", "medium", "Dynamic import detected"},
		{"subprocess.call", "medium", "Subprocess execution detected"},
		{"os.system", "high", "OS command execution detected"},
		{"base64.b64decode", "medium", "Base64 decoding detected (possible obfuscation)"},
	}

	lines := strings.Split(content, "\n")
	for lineNum, line := range lines {
		for _, p := range patterns {
			if strings.Contains(line, p.pattern) {
				s.issues = append(s.issues, Issue{
					Severity: p.severity,
					Type:     "suspicious-pattern",
					File:     file,
					Line:     lineNum + 1,
					Message:  p.message,
				})
			}
		}
	}
}

// checkDangerousCommands detects dangerous shell commands
func (s *Scanner) checkDangerousCommands(file, content string) {
	if !strings.HasSuffix(file, ".sh") && !strings.HasSuffix(file, ".bash") {
		return
	}

	dangerous := []struct {
		cmd      string
		severity string
		message  string
	}{
		{"rm -rf /", "critical", "Dangerous recursive delete detected"},
		{"curl | sh", "high", "Piping curl to shell detected"},
		{"wget | sh", "high", "Piping wget to shell detected"},
		{"chmod 777", "medium", "Overly permissive chmod detected"},
		{"sudo ", "medium", "Use of sudo detected"},
	}

	lines := strings.Split(content, "\n")
	for lineNum, line := range lines {
		for _, d := range dangerous {
			if strings.Contains(line, d.cmd) {
				s.issues = append(s.issues, Issue{
					Severity: d.severity,
					Type:     "dangerous-command",
					File:     file,
					Line:     lineNum + 1,
					Message:  d.message,
				})
			}
		}
	}
}

// isTextFile checks if file is likely text
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	textExts := []string{
		".md", ".txt", ".yaml", ".yml", ".json",
		".sh", ".bash", ".py", ".js", ".ts",
		".go", ".rs", ".java", ".rb", ".php",
	}

	for _, te := range textExts {
		if ext == te {
			return true
		}
	}

	return false
}

// HasCritical returns true if any critical issues found
func HasCritical(issues []Issue) bool {
	for _, i := range issues {
		if i.Severity == "critical" {
			return true
		}
	}
	return false
}

// FilterBySeverity returns issues matching severity
func FilterBySeverity(issues []Issue, severity string) []Issue {
	filtered := make([]Issue, 0)
	for _, i := range issues {
		if i.Severity == severity {
			filtered = append(filtered, i)
		}
	}
	return filtered
}
