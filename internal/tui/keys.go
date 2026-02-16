package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit         key.Binding
	Tab          key.Binding
	Pause        key.Binding
	Skip         key.Binding
	Timestamps   key.Binding
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Esc          key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "views"),
	),
	Pause: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "pause"),
	),
	Skip: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "skip"),
	),
	Timestamps: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "timestamps"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑", "scroll up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓", "scroll down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "page up"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("pgdn", "page down"),
	),
	HalfPageUp: key.NewBinding(
		key.WithKeys("ctrl+u"),
		key.WithHelp("C-u", "half page up"),
	),
	HalfPageDown: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("C-d", "half page down"),
	),
}
