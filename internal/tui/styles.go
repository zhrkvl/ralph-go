package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Minimal, dense styling — lazygit-inspired
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")) // bright blue

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")) // gray

	accentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // green

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")) // yellow

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")) // red

	statusRunning = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	statusPaused = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

	statusCompleted = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	statusFailed = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Bold(true)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("4"))

	storyCompletedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10"))

	storyOpenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	keyHintKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	keyHintDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))

	progressBarFilled = lipgloss.NewStyle().
				Foreground(lipgloss.Color("10"))

	progressBarEmpty = lipgloss.NewStyle().
				Foreground(lipgloss.Color("8"))
)

func separator(width int) string {
	s := ""
	for i := 0; i < width; i++ {
		s += "─"
	}
	return separatorStyle.Render(s)
}
