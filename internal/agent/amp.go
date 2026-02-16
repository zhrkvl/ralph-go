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
}

func (a *AmpAgent) Name() string { return "amp" }

func (a *AmpAgent) Start(ctx context.Context) (<-chan string, error) {
	promptPath := filepath.Join(a.ralphDir, "prompt.md")
	promptContent, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("reading prompt.md: %w", err)
	}

	cmd := exec.CommandContext(ctx, "amp", "--dangerously-allow-all")
	cmd.Dir = a.projectDir
	cmd.Stdin = bytes.NewReader(promptContent)

	return a.start(cmd)
}
