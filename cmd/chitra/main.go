package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppanda/chitragupta/internal/config"
	"github.com/ppanda/chitragupta/pkg/logger"
	"github.com/ppanda/chitragupta/pkg/workspace"
	"github.com/spf13/cobra"
)

var (
	Version   = "dev"     // Injected at build time via -ldflags
	Commit    = "unknown" // Git commit hash
	Date      = "unknown" // Build date
	verboseMode bool
	debugMode   bool
)

func main() {
	cfg, err := config.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.EnsureDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directories: %v\n", err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:   "cg",
		Short: "Package manager for AI skills, prompts, agents, and tools",
		Long:  `Chitragupta (cg) - Universal package manager for AI development assets across multiple repositories and AI platforms.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Set log level based on flags
			if debugMode {
				logger.SetLevel(logger.LevelDebug)
			} else if verboseMode {
				logger.SetLevel(logger.LevelInfo)
			}
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&verboseMode, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "Debug output with detailed logs")

	rootCmd.AddCommand(installCmd(cfg))
	rootCmd.AddCommand(publishCmd(cfg))
	rootCmd.AddCommand(listCmd(cfg))
	rootCmd.AddCommand(searchCmd(cfg))
	rootCmd.AddCommand(verifyCmd())
	rootCmd.AddCommand(compileCmd())
	rootCmd.AddCommand(workspaceCmd())
	rootCmd.AddCommand(aiCommands())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func installCmd(cfg *config.Config) *cobra.Command {
	var global bool
	var vars []string
	var skipSecurity bool

	cmd := &cobra.Command{
		Use:   "install [package[@version]]",
		Short: "Install packages from manifest or specific package",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				// Install from manifest
				return installFromManifest(cfg, skipSecurity)
			}
			// Install specific package
			return runInstall(cfg, args[0], global, vars)
		},
	}

	cmd.Flags().BoolVarP(&global, "global", "g", false, "Install globally to ~/.claude/")
	cmd.Flags().StringSlice("var", []string{}, "Template variables (key=value)")
	cmd.Flags().BoolVar(&skipSecurity, "skip-security", false, "Skip security scanning")
	return cmd
}

func publishCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "publish <directory>",
		Short: "Publish a package to registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPublish(cfg, args[0])
		},
	}
}

func listCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List packages in registry",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cfg)
		},
	}
}

func searchCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search for packages",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			return runSearch(cfg, query)
		},
	}
}

func workspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Workspace commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all workspaces",
		RunE: func(cmd *cobra.Command, args []string) error {
			ws, err := workspace.Discover(".")
			if err != nil {
				return err
			}

			if len(ws.Members) == 0 {
				fmt.Println("No workspaces configured")
				return nil
			}

			fmt.Printf("Found %d workspaces:\n", len(ws.Members))
			for _, member := range ws.Members {
				relPath, _ := filepath.Rel(".", member.Path)
				fmt.Printf("  - %s (%s)\n", relPath, member.Config.Name)
			}

			return nil
		},
	})

	return cmd
}

func versionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("cg version %s\n", Version)
			if verboseMode || debugMode {
				fmt.Printf("  Commit: %s\n", Commit)
				fmt.Printf("  Built:  %s\n", Date)
			}
		},
	}

	return cmd
}
