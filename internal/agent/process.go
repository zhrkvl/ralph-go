package agent

import (
	"bufio"
	"fmt"
	"io"
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

// start launches the command and returns a channel that streams output lines.
func (pm *ProcessManager) start(cmd *exec.Cmd) (<-chan string, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.cmd = cmd
	pm.allOutput.Reset()
	pm.paused.Store(false)
	pm.done.Store(false)

	// Use process group so we can pause/kill all children
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Merge stdout and stderr (matching 2>&1 behavior)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting process: %w", err)
	}

	ch := make(chan string, 100)

	// Read output lines
	go func() {
		defer close(ch)
		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			pm.mu.Lock()
			pm.allOutput.WriteString(line)
			pm.allOutput.WriteByte('\n')
			pm.mu.Unlock()
			ch <- line
		}
	}()

	// Close pipe writer when process exits
	go func() {
		cmd.Wait()
		pw.Close()
		pm.done.Store(true)
	}()

	return ch, nil
}

func (pm *ProcessManager) Wait() (string, error) {
	// Wait for process to finish — poll since cmd.Wait() is called in goroutine
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
	// Signal the whole process group
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
	// Then terminate
	syscall.Kill(-pm.cmd.Process.Pid, syscall.SIGTERM)
	// Give it a moment, then force kill
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
