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
		"--include-partial-messages",
	)
	cmd.Dir = a.projectDir

	rawCh, err := a.start(cmd, f)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Parse stream-json into human-readable lines with stateful delta accumulation
	parsedCh := make(chan string, 256)
	go func() {
		defer close(parsedCh)
		parser := newStreamParser()
		for line := range rawCh {
			for _, parsed := range parser.parseLine(line) {
				parsedCh <- parsed
			}
		}
		// Flush any remaining partial text
		for _, flushed := range parser.flush() {
			parsedCh <- flushed
		}
	}()

	return parsedCh, nil
}
