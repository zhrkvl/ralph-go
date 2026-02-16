package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zhrkvl/ralph-go/internal/session"
)

func renderHistory(m *Model) string {
	var b strings.Builder
	w := m.width

	b.WriteString(titleStyle.Render("Session History"))
	b.WriteString("\n")
	b.WriteString(separator(w))
	b.WriteString("\n")

	if len(m.archives) == 0 {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("  No archived sessions found"))
		b.WriteString("\n")
		return b.String()
	}

	// Column headers
	b.WriteString(dimStyle.Render(fmt.Sprintf("  %-12s %-30s %s", "Date", "Branch", "Files")))
	b.WriteString("\n")

	visibleHeight := m.height - 6
	startIdx := m.historyCursor - visibleHeight/2
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + visibleHeight
	if endIdx > len(m.archives) {
		endIdx = len(m.archives)
	}

	for i := startIdx; i < endIdx; i++ {
		a := m.archives[i]
		var files []string
		if a.HasPRD {
			files = append(files, "prd.json")
		}
		if a.HasProgress {
			files = append(files, "progress.txt")
		}

		line := fmt.Sprintf("  %-12s %-30s %s", a.Date, a.BranchName, strings.Join(files, ", "))

		if i == m.historyCursor {
			line = selectedStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

func renderHistoryDetail(m *Model) string {
	if m.historyCursor >= len(m.archives) {
		return "No archive selected"
	}

	a := m.archives[m.historyCursor]
	var b strings.Builder
	w := m.width

	b.WriteString(fmt.Sprintf("%s %s %s %s",
		titleStyle.Render("Archive"),
		dimStyle.Render("|"),
		a.Date,
		a.BranchName,
	))
	b.WriteString("\n")
	b.WriteString(separator(w))
	b.WriteString("\n\n")

	// Show progress.txt content
	if a.HasProgress {
		content, err := os.ReadFile(filepath.Join(a.Path, "progress.txt"))
		if err == nil {
			b.WriteString(string(content))
		} else {
			b.WriteString(errorStyle.Render(fmt.Sprintf("Error reading progress.txt: %v", err)))
		}
	} else {
		b.WriteString(dimStyle.Render("No progress.txt in this archive"))
	}

	return b.String()
}

func loadArchives(ralphDir string) []session.ArchiveEntry {
	archives, err := session.ListArchives(ralphDir)
	if err != nil {
		return nil
	}
	return archives
}

func clampHistoryCursor(m *Model) {
	max := len(m.archives) - 1
	if max < 0 {
		max = 0
	}
	if m.historyCursor > max {
		m.historyCursor = max
	}
	if m.historyCursor < 0 {
		m.historyCursor = 0
	}
}
