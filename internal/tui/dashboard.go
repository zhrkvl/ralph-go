package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/zhrkvl/ralph-go/internal/prd"
)

func renderDashboard(m *Model) string {
	var b strings.Builder
	w := m.width

	// Line 1: Ralph | tool | iteration | status | branch
	statusStr := renderStatus(m)
	left := fmt.Sprintf("%s %s %s %s %s",
		titleStyle.Render("Ralph"),
		dimStyle.Render("|"),
		m.agentName,
		dimStyle.Render("|"),
		fmt.Sprintf("Iteration %d/%d", m.iteration, m.maxIterations),
	)
	right := ""
	if m.prd != nil {
		right = dimStyle.Render(m.prd.BranchName)
	}
	mid := fmt.Sprintf(" %s %s ", dimStyle.Render("|"), statusStr)

	// Compose header line
	headerLeft := left + mid
	gap := w - lipglossWidth(headerLeft) - lipglossWidth(right)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(headerLeft + strings.Repeat(" ", gap) + right)
	b.WriteString("\n")

	// Line 2: Project — N/M stories + progress bar
	if m.prd != nil {
		completed := m.prd.CompletedCount()
		total := m.prd.TotalCount()
		pct := 0
		if total > 0 {
			pct = completed * 100 / total
		}
		bar := renderProgressBar(completed, total, w-40)
		line := fmt.Sprintf("%s %s %d/%d stories %s %d%%",
			m.prd.Name,
			dimStyle.Render("—"),
			completed, total,
			bar,
			pct,
		)
		b.WriteString(line)
	}
	b.WriteString("\n")

	// Separator
	b.WriteString(separator(w))
	b.WriteString("\n")

	// Current story
	if m.prd != nil {
		if cs := m.prd.CurrentStory(); cs != nil {
			b.WriteString(fmt.Sprintf("%s %s %s (P%d)",
				accentStyle.Render("▶"),
				accentStyle.Render(cs.ID),
				cs.Title,
				cs.Priority,
			))
		} else {
			b.WriteString(accentStyle.Render("All stories completed!"))
		}
	}
	b.WriteString("\n")

	// Separator
	b.WriteString(separator(w))
	b.WriteString("\n")

	// Agent output viewport (fill remaining space)
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Bottom separator
	b.WriteString(separator(w))

	return b.String()
}

func renderStatus(m *Model) string {
	if m.sessionStatus == "completed" {
		return statusCompleted.Render("Completed")
	}
	if m.sessionStatus == "failed" {
		return statusFailed.Render("Failed")
	}
	if m.agentPaused {
		return statusPaused.Render("Paused")
	}
	if m.agentRunning {
		return statusRunning.Render("Running")
	}
	return dimStyle.Render("Idle")
}

func renderProgressBar(completed, total, width int) string {
	if width < 5 {
		width = 5
	}
	if total == 0 {
		return ""
	}
	filled := width * completed / total
	empty := width - filled

	bar := progressBarFilled.Render(strings.Repeat("█", filled)) +
		progressBarEmpty.Render(strings.Repeat("░", empty))
	return bar
}

func initViewport(width, height int) viewport.Model {
	// Reserve lines for header (2), separators (3), current story (1), status bar (1)
	vpHeight := height - 8
	if vpHeight < 3 {
		vpHeight = 3
	}
	vp := viewport.New(width, vpHeight)
	vp.SetContent("")
	return vp
}

func updateViewportContent(vp *viewport.Model, lines []string, _ *prd.PRD) {
	content := strings.Join(lines, "\n")
	vp.SetContent(content)
	vp.GotoBottom()
}

func lipglossWidth(s string) int {
	// Strip ANSI escape codes to get actual width
	return len(stripAnsi(s))
}

func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
