package agent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// ProcessManager provides shared subprocess management for all agent types.
type ProcessManager struct {
	cmd       *exec.Cmd
	allOutput strings.Builder
	paused    atomic.Bool
	done      atomic.Bool
	mu        sync.Mutex
}

// start launches the command with the given stdin and returns a channel
// that streams output lines in real time.
//
// Uses OS-level pipes (not io.Pipe) for stdout/stderr. OS pipes have a
// kernel buffer (~64KB), so the subprocess can write freely without blocking.
// We also set env vars to hint CLIs to use unbuffered/line-buffered output.
func (pm *ProcessManager) start(cmd *exec.Cmd, stdin io.Reader) (<-chan string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.cmd = cmd
	pm.allOutput.Reset()
	pm.paused.Store(false)
	pm.done.Store(false)

	// Process group for pause/resume/kill of all children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set stdin
	cmd.Stdin = stdin

	// Env vars to encourage real-time output from common runtimes
	cmd.Env = append(os.Environ(),
		"TERM=dumb",
		"NO_COLOR=1",
		"COLUMNS=200",
		"LINES=50",
	)

	// Get pipes for stdout and stderr (OS-level, kernel-buffered)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting process: %w", err)
	}

	ch := make(chan string, 256)

	// Merge stdout and stderr into a single channel (matching 2>&1 behavior)
	var wg sync.WaitGroup
	wg.Add(2)

	readPipe := func(pipe io.ReadCloser) {
		defer wg.Done()
		scanner := bufio.NewScanner(pipe)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			pm.mu.Lock()
			pm.allOutput.WriteString(line)
			pm.allOutput.WriteByte('\n')
			pm.mu.Unlock()
			ch <- line
		}
	}

	go readPipe(stdoutPipe)
	go readPipe(stderrPipe)

	// Close channel when both pipes are drained and process exits
	go func() {
		wg.Wait()
		cmd.Wait()
		pm.done.Store(true)
		close(ch)
	}()

	return ch, nil
}

func (pm *ProcessManager) Wait() (string, error) {
	for !pm.done.Load() {
		time.Sleep(50 * time.Millisecond)
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.allOutput.String(), nil
}

func (pm *ProcessManager) Pause() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.cmd == nil || pm.cmd.Process == nil {
		return fmt.Errorf("no running process")
	}
	pm.paused.Store(true)
	return syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGSTOP)
}

func (pm *ProcessManager) Resume() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.cmd == nil || pm.cmd.Process == nil {
		return fmt.Errorf("no running process")
	}
	pm.paused.Store(false)
	return syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGCONT)
}

func (pm *ProcessManager) Kill() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.cmd == nil || pm.cmd.Process == nil {
		return nil
	}
	// Resume first in case it's paused
	syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGCONT)
	syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGTERM)
	go func() {
		time.Sleep(3 * time.Second)
		if !pm.done.Load() {
			pm.mu.Lock()
			if pm.cmd != nil && pm.cmd.Process != nil {
				syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGKILL)
			}
			pm.mu.Unlock()
		}
	}()
	return nil
}

func (pm *ProcessManager) IsPaused() bool {
	return pm.paused.Load()
}
