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

	cmd := exec.CommandContext(ctx, "claude",
		"--dangerously-skip-permissions",
		"--print",
		"--output-format", "stream-json",
		"--verbose",
	)
	cmd.Dir = a.projectDir

	rawCh, err := a.start(cmd, f)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Parse the stream-json format into human-readable lines
	parsedCh := make(chan string, 256)
	go func() {
		defer close(parsedCh)
		for line := range rawCh {
			for _, parsed := range parseStreamJSON(line) {
				parsedCh <- parsed
			}
		}
	}()

	return parsedCh, nil
}
