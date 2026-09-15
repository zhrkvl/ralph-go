package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Name identifies the agent in session state and iteration logs.
const Name = "claude"

// Options configures a Claude run.
type Options struct {
	RalphDir   string
	ProjectDir string
	Model      string
}

type Claude struct {
	*ProcessManager
	opts Options
}

func New(opts Options) *Claude {
	return &Claude{ProcessManager: &ProcessManager{}, opts: opts}
}

// buildArgs assembles the claude argv. Model is forwarded verbatim; claude
// warns and falls back to its default when the value is unrecognized.
func buildArgs(model string) []string {
	args := []string{
		"--dangerously-skip-permissions",
		"--print",
		"--output-format", "stream-json",
		"--verbose",
		"--include-partial-messages",
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	return args
}

func (a *Claude) Start(ctx context.Context) (<-chan string, error) {
	claudeMDPath := filepath.Join(a.opts.RalphDir, "CLAUDE.md")
	f, err := os.Open(claudeMDPath)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", claudeMDPath, err)
	}

	cmd := exec.CommandContext(ctx, "claude", buildArgs(a.opts.Model)...)
	cmd.Dir = a.opts.ProjectDir

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
