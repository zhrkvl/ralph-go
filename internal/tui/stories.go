package tui

import (
	"fmt"
	"strings"
)

func renderStories(m *Model) string {
	if m.prd == nil {
		return "No PRD loaded"
	}

	var b strings.Builder
	w := m.width

	// Header
	header := fmt.Sprintf("%s (%d total)",
		titleStyle.Render("Stories"),
		m.prd.TotalCount(),
	)
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(separator(w))
	b.WriteString("\n")

	stories := m.prd.UserStories
	visibleHeight := m.height - 5 // header, separator, status bar, padding

	// Calculate scroll window
	startIdx := m.storyCursor - visibleHeight/2
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visibleHeight
	if endIdx > len(stories) {
		endIdx = len(stories)
		startIdx = endIdx - visibleHeight
		if startIdx < 0 {
			startIdx = 0
		}
	}

	for i := startIdx; i < endIdx; i++ {
		s := stories[i]
		icon := storyOpenStyle.Render("○")
		if s.Passes {
			icon = storyCompletedStyle.Render("✓")
		}

		line := fmt.Sprintf(" %s %-7s %s", icon, s.ID, s.Title)

		if i == m.storyCursor {
			// Highlight selected line
			line = selectedStyle.Render(line)
		}

		// Truncate to width
		if lipglossWidth(line) > w {
			line = line[:w]
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(stories) > visibleHeight {
		b.WriteString(dimStyle.Render(fmt.Sprintf(" %d/%d", m.storyCursor+1, len(stories))))
		b.WriteString("\n")
	}

	return b.String()
}

func renderStoryDetail(m *Model) string {
	if m.prd == nil || m.storyCursor >= len(m.prd.UserStories) {
		return "No story selected"
	}

	s := m.prd.UserStories[m.storyCursor]
	var b strings.Builder
	w := m.width

	// Header
	icon := storyOpenStyle.Render("○")
	if s.Passes {
		icon = storyCompletedStyle.Render("✓")
	}
	b.WriteString(fmt.Sprintf("%s %s %s (P%d)", icon, titleStyle.Render(s.ID), s.Title, s.Priority))
	b.WriteString("\n")
	b.WriteString(separator(w))
	b.WriteString("\n\n")

	// Description
	if s.Description != "" {
		b.WriteString(dimStyle.Render("Description:"))
		b.WriteString("\n")
		b.WriteString(s.Description)
		b.WriteString("\n\n")
	}

	// Acceptance criteria
	if len(s.AcceptanceCriteria) > 0 {
		b.WriteString(dimStyle.Render("Acceptance Criteria:"))
		b.WriteString("\n")
		for _, ac := range s.AcceptanceCriteria {
			b.WriteString(fmt.Sprintf("  • %s\n", ac))
		}
		b.WriteString("\n")
	}

	// Notes
	if s.Notes != "" {
		b.WriteString(dimStyle.Render("Notes:"))
		b.WriteString("\n")
		b.WriteString(s.Notes)
		b.WriteString("\n\n")
	}

	// Iterations
	if s.Iterations > 0 {
		b.WriteString(fmt.Sprintf("%s %d\n", dimStyle.Render("Iterations:"), s.Iterations))
	}

	return b.String()
}

func storyDetailForViewport(m *Model) string {
	return renderStoryDetail(m)
}

func clampStoryCursor(m *Model) {
	if m.prd == nil {
		m.storyCursor = 0
		return
	}
	max := len(m.prd.UserStories) - 1
	if max < 0 {
		max = 0
	}
	if m.storyCursor > max {
		m.storyCursor = max
	}
	if m.storyCursor < 0 {
		m.storyCursor = 0
	}
}
