package main

import (
	"fmt"

	"github.com/ppanda/chitragupta/pkg/compiler"
	"github.com/spf13/cobra"
)

func compileCmd() *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile primitives to target-specific format",
		Long: `Compile installed primitives to target AI client format.

Targets:
  copilot - GitHub Copilot (.github/copilot-instructions.md)
  claude  - Claude Code (.claude/)
  cursor  - Cursor (.cursorrules)
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompile(target)
		},
	}

	cmd.Flags().StringVarP(&target, "target", "t", "claude", "Target format (copilot, claude, cursor)")

	return cmd
}

func runCompile(targetStr string) error {
	var target compiler.Target

	switch targetStr {
	case "copilot":
		target = compiler.TargetCopilot
	case "claude":
		target = compiler.TargetClaude
	case "cursor":
		target = compiler.TargetCursor
	default:
		return fmt.Errorf("unknown target: %s (must be: copilot, claude, cursor)", targetStr)
	}

	fmt.Printf("Compiling for %s...\n", target)

	c := compiler.New(target)

	// Compile from current .claude directory
	if err := c.Compile(".claude", "."); err != nil {
		return fmt.Errorf("compilation failed: %w", err)
	}

	switch target {
	case compiler.TargetCopilot:
		fmt.Println("✓ Generated .github/copilot-instructions.md")
	case compiler.TargetClaude:
		fmt.Println("✓ Generated .claude/")
	case compiler.TargetCursor:
		fmt.Println("✓ Generated .cursorrules")
	}

	return nil
}
