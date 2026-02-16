package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/zhrkvl/ralph-go/internal/prd"
)

type Session struct {
	Version          int           `json:"version"`
	SessionID        string        `json:"sessionId"`
	Status           string        `json:"status"` // running, completed, failed, interrupted
	StartedAt        time.Time     `json:"startedAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	CurrentIteration int           `json:"currentIteration"`
	MaxIterations    int           `json:"maxIterations"`
	TasksCompleted   int           `json:"tasksCompleted"`
	IsPaused         bool          `json:"isPaused"`
	AgentPlugin      string        `json:"agentPlugin"`
	TrackerState     *TrackerState `json:"trackerState"`
	Iterations       []any         `json:"iterations"`
	CWD              string        `json:"cwd"`
	ActiveTaskIDs    []string      `json:"activeTaskIds"`
}

type TrackerState struct {
	Plugin    string      `json:"plugin"`
	PRDPath   string      `json:"prdPath"`
	TotalTasks int        `json:"totalTasks"`
	Tasks     []TaskState `json:"tasks"`
}

type TaskState struct {
	ID                 string `json:"id"`
	Title              string `json:"title"`
	Status             string `json:"status"` // completed, open
	CompletedInSession bool   `json:"completedInSession"`
}

type SessionMeta struct {
	ID               string     `json:"id"`
	Status           string     `json:"status"`
	StartedAt        time.Time  `json:"startedAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	AgentPlugin      string     `json:"agentPlugin"`
	TrackerPlugin    string     `json:"trackerPlugin"`
	PRDPath          string     `json:"prdPath"`
	CurrentIteration int        `json:"currentIteration"`
	MaxIterations    int        `json:"maxIterations"`
	TotalTasks       int        `json:"totalTasks"`
	TasksCompleted   int        `json:"tasksCompleted"`
	CWD              string     `json:"cwd"`
	EndedAt          *time.Time `json:"endedAt,omitempty"`
}

func NewSession(projectDir string, prdPath string, agentName string, maxIter int, p *prd.PRD) *Session {
	now := time.Now().UTC()
	id := uuid.New().String()

	tasks := make([]TaskState, len(p.UserStories))
	for i, s := range p.UserStories {
		status := "open"
		if s.Passes {
			status = "completed"
		}
		tasks[i] = TaskState{
			ID:                 s.ID,
			Title:              s.Title,
			Status:             status,
			CompletedInSession: false,
		}
	}

	var activeIDs []string
	if cs := p.CurrentStory(); cs != nil {
		activeIDs = []string{cs.ID}
	}

	return &Session{
		Version:          1,
		SessionID:        id,
		Status:           "running",
		StartedAt:        now,
		UpdatedAt:        now,
		CurrentIteration: 0,
		MaxIterations:    maxIter,
		TasksCompleted:   p.CompletedCount(),
		IsPaused:         false,
		AgentPlugin:      agentName,
		TrackerState: &TrackerState{
			Plugin:     "json",
			PRDPath:    prdPath,
			TotalTasks: p.TotalCount(),
			Tasks:      tasks,
		},
		Iterations:    []any{},
		CWD:           projectDir,
		ActiveTaskIDs: activeIDs,
	}
}

func (s *Session) Save(projectDir string) error {
	dir := filepath.Join(projectDir, ".ralph-tui")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return writeJSON(filepath.Join(dir, "session.json"), s)
}

func (s *Session) SaveMeta(projectDir string) error {
	dir := filepath.Join(projectDir, ".ralph-tui")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	meta := SessionMeta{
		ID:               s.SessionID,
		Status:           s.Status,
		StartedAt:        s.StartedAt,
		UpdatedAt:        s.UpdatedAt,
		AgentPlugin:      s.AgentPlugin,
		TrackerPlugin:    "json",
		PRDPath:          s.TrackerState.PRDPath,
		CurrentIteration: s.CurrentIteration,
		MaxIterations:    s.MaxIterations,
		TotalTasks:       s.TrackerState.TotalTasks,
		TasksCompleted:   s.TasksCompleted,
		CWD:              s.CWD,
	}
	if s.Status == "completed" || s.Status == "failed" || s.Status == "interrupted" {
		now := time.Now().UTC()
		meta.EndedAt = &now
	}
	return writeJSON(filepath.Join(dir, "session-meta.json"), &meta)
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
