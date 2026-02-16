package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zhrkvl/ralph-go/internal/config"
	"github.com/zhrkvl/ralph-go/internal/prd"
	"github.com/zhrkvl/ralph-go/internal/session"
	"github.com/zhrkvl/ralph-go/internal/tui"
)

var (
	toolFlag      string
	maxIterFlag   int
	ralphDirFlag  string
	projectDirFlag string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ralph",
		Short: "Ralph — autonomous AI agent loop with TUI",
		Long:  "Ralph orchestrates an AI agent (Claude or Amp) to work through user stories in a PRD.",
		RunE:  run,
	}

	rootCmd.Flags().StringVar(&toolFlag, "tool", "", "agent tool to use: amp or claude (default from config or amp)")
	rootCmd.Flags().IntVar(&maxIterFlag, "max-iterations", 0, "maximum iterations (default from config or 10)")
	rootCmd.Flags().StringVar(&ralphDirFlag, "ralph-dir", "", "directory containing prd.json and CLAUDE.md")
	rootCmd.Flags().StringVar(&projectDirFlag, "project-dir", "", "working directory for agent (default: CWD)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Resolve project dir
	projectDir := projectDirFlag
	if projectDir == "" {
		var err error
		projectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("getting CWD: %w", err)
		}
	}
	projectDir, _ = filepath.Abs(projectDir)

	// Load config from project dir
	cfg, err := config.Load(projectDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Resolve ralph dir
	ralphDir := resolveRalphDir(ralphDirFlag, projectDir)
	if ralphDir == "" {
		return fmt.Errorf("cannot find ralph directory (no prd.json found). Use --ralph-dir to specify")
	}
	ralphDir, _ = filepath.Abs(ralphDir)

	// Determine tool and max iterations (CLI flags override config)
	agentName := cfg.Agent
	if toolFlag != "" {
		agentName = toolFlag
	}
	if agentName != "amp" && agentName != "claude" {
		return fmt.Errorf("invalid tool '%s'. Must be 'amp' or 'claude'", agentName)
	}

	maxIter := cfg.MaxIterations
	if maxIterFlag > 0 {
		maxIter = maxIterFlag
	}
	if maxIter <= 0 {
		maxIter = 10
	}

	// Load PRD
	prdPath := filepath.Join(ralphDir, "prd.json")
	p, err := prd.Load(prdPath)
	if err != nil {
		return fmt.Errorf("loading PRD: %w", err)
	}

	// Branch change detection and archival
	archived, err := session.CheckAndArchive(ralphDir, p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: archival check failed: %v\n", err)
	}
	if archived {
		fmt.Fprintf(os.Stderr, "Archived previous run\n")
	}

	// Update branch tracking
	session.UpdateLastBranch(ralphDir, p.BranchName)

	// Initialize progress file
	session.InitProgressFile(ralphDir)

	// Create session
	sess := session.NewSession(projectDir, prdPath, agentName, maxIter, p)
	sess.Save(projectDir)
	sess.SaveMeta(projectDir)

	// Launch TUI
	return tui.Run(tui.Options{
		PRD:           p,
		PRDPath:       prdPath,
		RalphDir:      ralphDir,
		ProjectDir:    projectDir,
		AgentName:     agentName,
		MaxIterations: maxIter,
		Session:       sess,
	})
}

// resolveRalphDir finds the ralph directory containing prd.json.
func resolveRalphDir(explicit, projectDir string) string {
	if explicit != "" {
		return explicit
	}

	// Check RALPH_DIR env var
	if envDir := os.Getenv("RALPH_DIR"); envDir != "" {
		if hasPRD(envDir) {
			return envDir
		}
	}

	// Check scripts/ralph/ relative to project dir
	candidate := filepath.Join(projectDir, "scripts", "ralph")
	if hasPRD(candidate) {
		return candidate
	}

	// Check CWD itself
	if hasPRD(projectDir) {
		return projectDir
	}

	return ""
}

func hasPRD(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "prd.json"))
	return err == nil
}
