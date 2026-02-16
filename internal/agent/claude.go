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
		return nil, fmt.Errorf("opening CLAUDE.md: %w", err)
	}

	cmd := exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "--print")
	cmd.Dir = a.projectDir
	cmd.Stdin = f

	ch, err := a.start(cmd)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Close file after process starts (stdin has been read)
	// Actually, keep it open — the process reads from it asynchronously
	// It will be cleaned up when the process exits
	return ch, nil
}
