package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zhrkvl/ralph-go/internal/config"
	"github.com/zhrkvl/ralph-go/internal/prd"
	"github.com/zhrkvl/ralph-go/internal/session"
	"github.com/zhrkvl/ralph-go/internal/tui"
)

var (
	toolFlag           string
	modelFlag          string
	maxIterFlag        int
	ralphDirFlag       string
	projectDirFlag     string
	installClaudeFlag  bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ralph",
		Short: "Ralph — autonomous AI agent loop with TUI",
		Long:  "Ralph orchestrates an AI agent (Claude or Amp) to work through user stories in a PRD.",
		RunE:  run,
	}

	rootCmd.Flags().StringVar(&toolFlag, "tool", "", "agent tool to use: amp or claude (default from config or amp)")
	rootCmd.Flags().StringVar(&modelFlag, "model", "", "model to use (passed as --model to the agent)")
	rootCmd.Flags().IntVar(&maxIterFlag, "max-iterations", 0, "maximum iterations (default from config or 10)")
	rootCmd.Flags().StringVar(&ralphDirFlag, "ralph-dir", "", "directory containing prd.json and CLAUDE.md")
	rootCmd.Flags().StringVar(&projectDirFlag, "project-dir", "", "working directory for agent (default: CWD)")
	rootCmd.Flags().BoolVar(&installClaudeFlag, "install-claude", false, "download scripts/ralph (CLAUDE.md, ralph.sh) from github.com/snarktank/ralph into CWD")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if installClaudeFlag {
		return installClaude()
	}

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

	// Warn if multiple prd.json files exist in the project tree.
	if dupes := findPRDFiles(projectDir); len(dupes) > 1 {
		fmt.Fprintf(os.Stderr, "Warning: multiple prd.json files found in %s:\n", projectDir)
		for _, p := range dupes {
			fmt.Fprintf(os.Stderr, "  %s (%s)\n", p, prdSummary(p))
		}
		fmt.Fprintf(os.Stderr, "Use --ralph-dir to specify which one to use.\n")
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
		Model:         modelFlag,
		MaxIterations: maxIter,
		Session:       sess,
	})
}

// installClaude sparse-clones scripts/ralph from github.com/snarktank/ralph
// into ./scripts/ralph in the current working directory.
func installClaude() error {
	const (
		repoURL   = "https://github.com/snarktank/ralph"
		subPath   = "scripts/ralph"
		branch    = "main"
	)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting CWD: %w", err)
	}

	dest := filepath.Join(cwd, subPath)
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("destination %s already exists", dest)
	}

	tmpDir, err := os.MkdirTemp("", "ralph-install-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	run := func(args ...string) error {
		c := exec.Command(args[0], args[1:]...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	fmt.Printf("Cloning %s (branch %s) ...\n", repoURL, branch)
	if err := run("git", "clone", "--depth=1", "--filter=blob:none", "--sparse",
		"--branch", branch, repoURL, tmpDir); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	if err := run("git", "-C", tmpDir, "sparse-checkout", "set", subPath); err != nil {
		return fmt.Errorf("git sparse-checkout: %w", err)
	}

	src := filepath.Join(tmpDir, subPath)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating scripts dir: %w", err)
	}
	if err := os.Rename(src, dest); err != nil {
		// Rename across devices fails; fall back to copy.
		if copyErr := copyDir(src, dest); copyErr != nil {
			return fmt.Errorf("copying files: %w", copyErr)
		}
	}

	fmt.Printf("Installed %s\n", dest)
	return nil
}

// copyDir recursively copies src directory to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
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

// prdSummary reads a prd.json and returns a human-readable task count string.
func prdSummary(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "unreadable"
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return "empty file"
	}
	var p prd.PRD
	if err := json.Unmarshal(data, &p); err != nil {
		return "invalid JSON"
	}
	if p.UserStories == nil {
		return "no userStories field"
	}
	total := p.TotalCount()
	active := p.RemainingCount()
	return fmt.Sprintf("%d tasks active, %d tasks total", active, total)
}

// findPRDFiles returns all prd.json paths found under root, skipping common
// noise directories (.git, node_modules, vendor).
func findPRDFiles(root string) []string {
	skipDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
	}
	var found []string
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() && skipDirs[d.Name()] {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == "prd.json" {
			found = append(found, path)
		}
		return nil
	})
	return found
}
