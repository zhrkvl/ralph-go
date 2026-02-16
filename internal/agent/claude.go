package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type ClaudeAgent struct {
	*ProcessManager
	ralphDir   string
	projectDir string
}

func (a *ClaudeAgent) Name() string { return "claude" }

func (a *ClaudeAgent) Start(ctx context.Context) (<-chan string, error) {
	claudeMDPath := filepath.Join(a.ralphDir, "CLAUDE.md")
	f, err := os.Open(claudeMDPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", claudeMDPath, err)
	}

	cmd := exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "--print")
	cmd.Dir = a.projectDir

	return a.start(cmd, f)
}
