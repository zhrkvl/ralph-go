package tui

import "fmt"

func renderStatusBar(view View, agentRunning bool, width int) string {
	var hints []string

	hints = append(hints, keyHint("Tab", viewName(nextView(view))))

	if agentRunning {
		hints = append(hints, keyHint("p", "pause"))
		hints = append(hints, keyHint("s", "skip"))
	}

	switch view {
	case viewStories, viewHistory:
		hints = append(hints, keyHint("Enter", "select"))
	case viewStoryDetail, viewHistoryDetail:
		hints = append(hints, keyHint("Esc", "back"))
	}

	hints = append(hints, keyHint("t", "timestamps"))
	hints = append(hints, keyHint("↑↓", "scroll"))
	hints = append(hints, keyHint("q", "quit"))

	line := ""
	for i, h := range hints {
		if i > 0 {
			line += "  "
		}
		line += h
	}

	// Pad to width
	for len(line) < width {
		line += " "
	}
	return line
}

func keyHint(k, desc string) string {
	return fmt.Sprintf("%s:%s", keyHintKeyStyle.Render(k), keyHintDescStyle.Render(desc))
}

func viewName(v View) string {
	switch v {
	case viewDashboard:
		return "dash"
	case viewStories:
		return "stories"
	case viewHistory:
		return "history"
	default:
		return "dash"
	}
}

func nextView(v View) View {
	switch v {
	case viewDashboard:
		return viewStories
	case viewStories:
		return viewHistory
	case viewHistory:
		return viewDashboard
	default:
		return viewDashboard
	}
}
