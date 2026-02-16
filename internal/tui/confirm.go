package tui

import "fmt"

func renderConfirmQuit(width int) string {
	msg := "Agent is running. Quit and kill it? (y/n)"
	padding := ""
	if width > len(msg)+4 {
		pad := (width - len(msg)) / 2
		for i := 0; i < pad; i++ {
			padding += " "
		}
	}
	return fmt.Sprintf("\n\n%s%s\n", padding, warnStyle.Render(msg))
}
