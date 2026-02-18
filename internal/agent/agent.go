package agent

import "context"

// Agent represents an AI agent that can be started, paused, and stopped.
type Agent interface {
	// Start launches the agent subprocess. Output is streamed line-by-line
	// to the returned channel. The channel closes when the process exits.
	Start(ctx context.Context) (<-chan string, error)

	// Wait blocks until the subprocess exits and returns the full combined output.
	Wait() (string, error)

	// Pause sends SIGSTOP to the process group.
	Pause() error

	// Resume sends SIGCONT to the process group.
	Resume() error

	// Kill terminates the subprocess.
	Kill() error

	// IsPaused returns whether the process is currently stopped.
	IsPaused() bool

	// Name returns "amp" or "claude".
	Name() string
}

// New creates a new agent by name.
func New(name, ralphDir, projectDir, model string) Agent {
	switch name {
	case "claude":
		return &ClaudeAgent{
			ProcessManager: &ProcessManager{},
			ralphDir:       ralphDir,
			projectDir:     projectDir,
			model:          model,
		}
	case "amp":
		return &AmpAgent{
			ProcessManager: &ProcessManager{},
			ralphDir:       ralphDir,
			projectDir:     projectDir,
			model:          model,
		}
	default:
		return &ClaudeAgent{
			ProcessManager: &ProcessManager{},
			ralphDir:       ralphDir,
			projectDir:     projectDir,
			model:          model,
		}
	}
}
