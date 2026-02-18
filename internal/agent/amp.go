package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type AmpAgent struct {
	*ProcessManager
	ralphDir   string
	projectDir string
	model      string
}

func (a *AmpAgent) Name() string { return "amp" }

func (a *AmpAgent) Start(ctx context.Context) (<-chan string, error) {
	promptPath := filepath.Join(a.ralphDir, "prompt.md")
	promptContent, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", promptPath, err)
	}

	args := []string{"--dangerously-allow-all"}
	if a.model != "" {
		args = append(args, "--model", a.model)
	}
	cmd := exec.CommandContext(ctx, "amp", args...)
	cmd.Dir = a.projectDir

	return a.start(cmd, bytes.NewReader(promptContent))
}
