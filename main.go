package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zhrkvl/ralph-go/internal/agent"
	"github.com/zhrkvl/ralph-go/internal/config"
	"github.com/zhrkvl/ralph-go/internal/prd"
	"github.com/zhrkvl/ralph-go/internal/session"
	"github.com/zhrkvl/ralph-go/internal/tui"
)

var (
	modelFlag         string
	effortFlag        string
	maxIterFlag       int
	ralphDirFlag      string
	projectDirFlag    string
	installClaudeFlag bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ralph",
		Short: "Ralph — autonomous AI agent loop with TUI",
		Long:  "Ralph orchestrates Claude Code to work through user stories in a PRD.",
		RunE:  run,
	}

	rootCmd.Flags().StringVar(&modelFlag, "model", "", "model forwarded to claude --model (alias such as opus or sonnet, or a full model name)")
	rootCmd.Flags().StringVar(&effortFlag, "effort", "", "effort level forwarded to claude --effort (low, medium, high, xhigh, max)")
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

	// Max iterations: CLI flag overrides config
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
	sess := session.NewSession(projectDir, prdPath, agent.Name, maxIter, p)
	sess.Save(projectDir)
	sess.SaveMeta(projectDir)

	// Launch TUI
	return tui.Run(tui.Options{
		PRD:           p,
		PRDPath:       prdPath,
		RalphDir:      ralphDir,
		ProjectDir:    projectDir,
		Model:         modelFlag,
		Effort:        effortFlag,
		MaxIterations: maxIter,
		Session:       sess,
	})
}

// installClaude clones github.com/snarktank/ralph and copies CLAUDE.md and
// ralph.sh into ./scripts/ralph in the current working directory.
func installClaude() error {
	const (
		repoURL = "https://github.com/snarktank/ralph"
		branch  = "main"
		destRel = "scripts/ralph"
	)

	// Files to copy from the repo root (leading slash required by git sparse-checkout non-cone mode).
	files := []string{"CLAUDE.md", "ralph.sh"}
	sparseFiles := []string{"/CLAUDE.md", "/ralph.sh"}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting CWD: %w", err)
	}

	dest := filepath.Join(cwd, destRel)
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("destination %s already exists", dest)
	}

	tmpDir, err := os.MkdirTemp("", "ralph-install-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	gitRun := func(args ...string) error {
		c := exec.Command(args[0], args[1:]...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	}

	fmt.Printf("Cloning %s (branch %s) ...\n", repoURL, branch)
	if err := gitRun("git", "clone", "--depth=1", "--filter=blob:none", "--sparse",
		"--branch", branch, repoURL, tmpDir); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}
	// Sparse-checkout the individual files from the repo root.
	if err := gitRun("git", "-C", tmpDir, "sparse-checkout", "set", "--no-cone"); err != nil {
		return fmt.Errorf("git sparse-checkout set: %w", err)
	}
	if err := gitRun(append([]string{"git", "-C", tmpDir, "sparse-checkout", "add"}, sparseFiles...)...); err != nil {
		return fmt.Errorf("git sparse-checkout add: %w", err)
	}

	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("creating dest dir: %w", err)
	}

	for _, f := range files {
		src := filepath.Join(tmpDir, f)
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("reading %s: %w", f, err)
		}
		info, err := os.Stat(src)
		if err != nil {
			return fmt.Errorf("stat %s: %w", f, err)
		}
		if err := os.WriteFile(filepath.Join(dest, f), data, info.Mode()); err != nil {
			return fmt.Errorf("writing %s: %w", f, err)
		}
	}

	fmt.Printf("Installed into %s\n", dest)
	return nil
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
