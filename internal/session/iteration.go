package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type IterationLog struct {
	ProjectDir string
	TaskID     string
	TaskTitle  string
	Agent      string
	StartedAt  time.Time
	file       *os.File
	hash       string
}

func NewIterationLog(projectDir, taskID, taskTitle, agent string) (*IterationLog, error) {
	dir := filepath.Join(projectDir, ".ralph-tui", "iterations")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	hashBytes := make([]byte, 4)
	rand.Read(hashBytes)
	hash := hex.EncodeToString(hashBytes)

	now := time.Now()
	timestamp := now.Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s_%s.log", hash, timestamp, taskID)

	f, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		return nil, err
	}

	log := &IterationLog{
		ProjectDir: projectDir,
		TaskID:     taskID,
		TaskTitle:  taskTitle,
		Agent:      agent,
		StartedAt:  now,
		file:       f,
		hash:       hash,
	}

	// Write metadata header
	var sb strings.Builder
	sb.WriteString("# Iteration Log\n\n")
	sb.WriteString("## Metadata\n\n")
	sb.WriteString(fmt.Sprintf("- **Task ID**: %s\n", taskID))
	sb.WriteString(fmt.Sprintf("- **Task Title**: %s\n", taskTitle))
	sb.WriteString(fmt.Sprintf("- **Started At**: %s\n", now.UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Agent**: %s\n", agent))
	sb.WriteString("--- RAW OUTPUT ---\n\n")
	f.WriteString(sb.String())

	return log, nil
}

func (l *IterationLog) WriteLine(line string) {
	if l.file != nil {
		l.file.WriteString(line + "\n")
	}
}

func (l *IterationLog) Close(completed bool, promiseDetected bool) error {
	if l.file == nil {
		return nil
	}
	duration := time.Since(l.StartedAt)

	status := "normal"
	if completed {
		status = "completed"
	}

	var sb strings.Builder
	sb.WriteString("\n--- END OUTPUT ---\n\n")
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Status**: %s\n", status))
	sb.WriteString(fmt.Sprintf("- **Task Completed**: %v\n", completed))
	sb.WriteString(fmt.Sprintf("- **Promise Detected**: %v\n", promiseDetected))
	sb.WriteString(fmt.Sprintf("- **Ended At**: %s\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("- **Duration**: %s\n", formatDuration(duration)))

	l.file.WriteString(sb.String())
	return l.file.Close()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}
