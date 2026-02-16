package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zhrkvl/ralph-go/internal/agent"
	"github.com/zhrkvl/ralph-go/internal/prd"
	"github.com/zhrkvl/ralph-go/internal/session"
)

type View int

const (
	viewDashboard View = iota
	viewStories
	viewHistory
	viewStoryDetail
	viewHistoryDetail
	viewConfirmQuit
)

const maxOutputLines = 500

// Messages

type agentOutputMsg struct{ line string }

type agentStartedMsg struct {
	outputCh <-chan string
	agent    agent.Agent
	iterLog  *session.IterationLog
}

type agentDoneMsg struct {
	completed bool
	errMsg    string // non-empty if agent failed to start
}

type prdReloadMsg struct{ p *prd.PRD }
type tickMsg time.Time
type iterationSleepDoneMsg struct{}

type Model struct {
	// View state
	activeView   View
	previousView View
	width        int
	height       int
	ready        bool // true after first WindowSizeMsg

	// Data
	prd        *prd.PRD
	prdPath    string
	ralphDir   string
	projectDir string
	agentName  string
	archives   []session.ArchiveEntry

	// Agent loop
	currentAgent   agent.Agent
	agentRunning   bool
	agentPaused    bool
	iteration      int
	maxIterations  int
	outputLines    []string
	sessionStatus  string // running, completed, failed, interrupted
	outputCh       <-chan string
	cancelAgent    context.CancelFunc

	// Viewport for agent output
	viewport       viewport.Model
	showTimestamps bool

	// List cursors
	storyCursor   int
	historyCursor int

	// Detail viewport (for story/history detail views)
	detailViewport viewport.Model

	// Session tracking
	sess    *session.Session
	iterLog *session.IterationLog

	quitting bool
}

type Options struct {
	PRD           *prd.PRD
	PRDPath       string
	RalphDir      string
	ProjectDir    string
	AgentName     string
	MaxIterations int
	Session       *session.Session
}

func NewModel(opts Options) Model {
	return Model{
		activeView:    viewDashboard,
		prd:           opts.PRD,
		prdPath:       opts.PRDPath,
		ralphDir:      opts.RalphDir,
		projectDir:    opts.ProjectDir,
		agentName:     opts.AgentName,
		maxIterations: opts.MaxIterations,
		iteration:     0,
		sessionStatus: "running",
		outputLines:   make([]string, 0, maxOutputLines),
		archives:       loadArchives(opts.RalphDir),
		sess:           opts.Session,
		viewport:       viewport.New(80, 20),
		detailViewport: viewport.New(80, 20),
		showTimestamps: true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.startAgentCmd(),
		tickCmd(),
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport = initViewport(m.width, m.height)
		m.detailViewport = viewport.New(m.width, m.height-5)
		updateViewportContent(&m.viewport, m.outputLines, m.showTimestamps)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case agentStartedMsg:
		m.outputCh = msg.outputCh
		m.currentAgent = msg.agent
		m.iterLog = msg.iterLog
		m.agentRunning = true
		m.agentPaused = false
		m.appendOutput(fmt.Sprintf(
			"%s  %s %d / %d",
			dimStyle.Render(strings.Repeat("═", 50)),
			titleStyle.Render("Iteration"),
			m.iteration, m.maxIterations,
		))
		return m, waitForOutput(m.outputCh)

	case agentOutputMsg:
		m.appendOutput(msg.line)
		if m.iterLog != nil {
			m.iterLog.WriteLine(msg.line)
		}
		// Check for completion signal in this line
		if strings.Contains(msg.line, "<promise>COMPLETE</promise>") {
			// Don't wait for more output — mark completed
			// The channel will close naturally
		}
		return m, waitForOutput(m.outputCh)

	case agentDoneMsg:
		m.agentRunning = false
		m.agentPaused = false
		m.currentAgent = nil

		// Show error if agent failed to start
		if msg.errMsg != "" {
			m.appendOutput(errorStyle.Render("Agent error: " + msg.errMsg))
		}

		// Check accumulated output for completion signal
		completed := msg.completed
		if !completed {
			for _, line := range m.outputLines {
				if strings.Contains(line, "<promise>COMPLETE</promise>") {
					completed = true
					break
				}
			}
		}

		// Close iteration log
		if m.iterLog != nil {
			m.iterLog.Close(completed, completed)
			m.iterLog = nil
		}

		if completed {
			m.sessionStatus = "completed"
			m.appendOutput("")
			m.appendOutput(accentStyle.Render("All tasks completed!"))
			m.saveState()
			return m, nil
		}

		if m.iteration >= m.maxIterations {
			m.sessionStatus = "failed"
			m.appendOutput("")
			m.appendOutput(errorStyle.Render(fmt.Sprintf(
				"Max iterations (%d) reached without completion.", m.maxIterations)))
			m.saveState()
			return m, nil
		}

		// Sleep 2s then start next iteration (matching ralph.sh)
		m.appendOutput(dimStyle.Render("Iteration complete. Next in 2s..."))
		return m, tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
			return iterationSleepDoneMsg{}
		})

	case iterationSleepDoneMsg:
		return m, m.startAgentCmd()

	case prdReloadMsg:
		if msg.p != nil {
			m.prd = msg.p
			if m.sess != nil {
				m.sess.TasksCompleted = msg.p.CompletedCount()
			}
		}
		return m, nil

	case tickMsg:
		var cmds []tea.Cmd
		cmds = append(cmds, tickCmd(), reloadPRD(m.prdPath))
		if m.sess != nil {
			m.sess.UpdatedAt = time.Now().UTC()
			m.sess.CurrentIteration = m.iteration
			m.sess.Status = m.sessionStatus
			m.sess.IsPaused = m.agentPaused
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Confirm quit dialog
	if m.activeView == viewConfirmQuit {
		switch msg.String() {
		case "y", "Y":
			m.killAgent()
			m.quitting = true
			m.sessionStatus = "interrupted"
			m.saveState()
			return m, tea.Quit
		case "n", "N", "esc":
			m.activeView = m.previousView
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.Quit):
		if m.agentRunning {
			m.previousView = m.activeView
			m.activeView = viewConfirmQuit
			return m, nil
		}
		m.saveState()
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, keys.Tab):
		m.activeView = nextView(m.activeView)
		return m, nil

	case key.Matches(msg, keys.Pause):
		if m.agentRunning {
			m.togglePause()
		}
		return m, nil

	case key.Matches(msg, keys.Skip):
		if m.agentRunning {
			m.killAgent()
			m.appendOutput(warnStyle.Render("Skipping iteration..."))
		}
		return m, nil

	case key.Matches(msg, keys.Timestamps):
		m.showTimestamps = !m.showTimestamps
		updateViewportContent(&m.viewport, m.outputLines, m.showTimestamps)
		return m, nil

	case key.Matches(msg, keys.Esc):
		switch m.activeView {
		case viewStoryDetail:
			m.activeView = viewStories
		case viewHistoryDetail:
			m.activeView = viewHistory
		}
		return m, nil

	case key.Matches(msg, keys.Enter):
		switch m.activeView {
		case viewStories:
			m.activeView = viewStoryDetail
			m.detailViewport.SetContent(storyDetailForViewport(m))
			m.detailViewport.GotoTop()
		case viewHistory:
			if len(m.archives) > 0 {
				m.activeView = viewHistoryDetail
				m.detailViewport.SetContent(renderHistoryDetail(m))
				m.detailViewport.GotoTop()
			}
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		m.handleScroll(-1)
	case key.Matches(msg, keys.Down):
		m.handleScroll(1)
	case key.Matches(msg, keys.PageUp):
		m.handlePageScroll(-1)
	case key.Matches(msg, keys.PageDown):
		m.handlePageScroll(1)
	case key.Matches(msg, keys.HalfPageUp):
		m.handleHalfPageScroll(-1)
	case key.Matches(msg, keys.HalfPageDown):
		m.handleHalfPageScroll(1)
	}

	return m, nil
}

func (m *Model) handleScroll(dir int) {
	switch m.activeView {
	case viewDashboard:
		if dir < 0 {
			m.viewport.LineUp(1)
		} else {
			m.viewport.LineDown(1)
		}
	case viewStories:
		m.storyCursor += dir
		clampStoryCursor(m)
	case viewHistory:
		m.historyCursor += dir
		clampHistoryCursor(m)
	case viewStoryDetail, viewHistoryDetail:
		if dir < 0 {
			m.detailViewport.LineUp(1)
		} else {
			m.detailViewport.LineDown(1)
		}
	}
}

func (m *Model) handlePageScroll(dir int) {
	switch m.activeView {
	case viewDashboard:
		if dir < 0 {
			m.viewport.ViewUp()
		} else {
			m.viewport.ViewDown()
		}
	case viewStoryDetail, viewHistoryDetail:
		if dir < 0 {
			m.detailViewport.ViewUp()
		} else {
			m.detailViewport.ViewDown()
		}
	}
}

func (m *Model) handleHalfPageScroll(dir int) {
	switch m.activeView {
	case viewDashboard:
		if dir < 0 {
			m.viewport.HalfViewUp()
		} else {
			m.viewport.HalfViewDown()
		}
	case viewStoryDetail, viewHistoryDetail:
		if dir < 0 {
			m.detailViewport.HalfViewUp()
		} else {
			m.detailViewport.HalfViewDown()
		}
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var content string
	switch m.activeView {
	case viewDashboard:
		content = renderDashboard(&m)
	case viewStories:
		content = renderStories(&m)
	case viewStoryDetail:
		content = m.detailViewport.View()
	case viewHistory:
		content = renderHistory(&m)
	case viewHistoryDetail:
		content = m.detailViewport.View()
	case viewConfirmQuit:
		content = renderDashboard(&m) + renderConfirmQuit(m.width)
	}

	statusBar := renderStatusBar(m.activeView, m.agentRunning, m.width)
	return content + "\n" + statusBar
}

func (m *Model) appendOutput(line string) {
	m.outputLines = append(m.outputLines, line)
	if len(m.outputLines) > maxOutputLines {
		m.outputLines = m.outputLines[len(m.outputLines)-maxOutputLines:]
	}
	updateViewportContent(&m.viewport, m.outputLines, m.showTimestamps)
}

func (m *Model) togglePause() {
	if m.currentAgent == nil {
		return
	}
	if m.agentPaused {
		if err := m.currentAgent.Resume(); err == nil {
			m.agentPaused = false
			m.appendOutput(accentStyle.Render("▶ Agent resumed"))
		}
	} else {
		if err := m.currentAgent.Pause(); err == nil {
			m.agentPaused = true
			m.appendOutput(warnStyle.Render("⏸ Agent paused"))
		}
	}
}

func (m *Model) killAgent() {
	if m.cancelAgent != nil {
		m.cancelAgent()
		m.cancelAgent = nil
	}
	if m.currentAgent != nil {
		m.currentAgent.Kill()
	}
}

func (m *Model) saveState() {
	if m.sess == nil {
		return
	}
	m.sess.Status = m.sessionStatus
	m.sess.UpdatedAt = time.Now().UTC()
	m.sess.CurrentIteration = m.iteration
	m.sess.Save(m.projectDir)
	m.sess.SaveMeta(m.projectDir)
}

// startAgentCmd returns a Cmd that starts the next agent iteration.
// It increments the iteration counter and launches the subprocess.
// The result is an agentStartedMsg containing the output channel.
func (m *Model) startAgentCmd() tea.Cmd {
	m.iteration++
	iter := m.iteration
	agentName := m.agentName
	ralphDir := m.ralphDir
	projectDir := m.projectDir

	taskID := "unknown"
	taskTitle := "unknown"
	if m.prd != nil {
		if cs := m.prd.CurrentStory(); cs != nil {
			taskID = cs.ID
			taskTitle = cs.Title
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelAgent = cancel

	return func() tea.Msg {
		iterLog, _ := session.NewIterationLog(projectDir, taskID, taskTitle, agentName)

		a := agent.New(agentName, ralphDir, projectDir)
		ch, err := a.Start(ctx)
		if err != nil {
			if iterLog != nil {
				iterLog.Close(false, false)
			}
			return agentDoneMsg{completed: false, errMsg: err.Error()}
		}

		_ = iter
		return agentStartedMsg{
			outputCh: ch,
			agent:    a,
			iterLog:  iterLog,
		}
	}
}

func waitForOutput(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return agentDoneMsg{}
		}
		line, ok := <-ch
		if !ok {
			return agentDoneMsg{completed: false}
		}
		return agentOutputMsg{line: line}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func reloadPRD(path string) tea.Cmd {
	return func() tea.Msg {
		p, err := prd.Load(path)
		if err != nil {
			return prdReloadMsg{p: nil}
		}
		return prdReloadMsg{p: p}
	}
}

// Run starts the TUI program.
func Run(opts Options) error {
	m := NewModel(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
