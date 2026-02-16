// Package tui provides a raspi-config style terminal user interface for
// Config Manager. It is built with Bubble Tea and styled with Lip Gloss.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Model is the main Bubble Tea model for the Config Manager TUI.
type Model struct {
	menuItems []MenuItem
	cursor    int
	quitting  bool
}

// New returns an initialised TUI model with the default main menu.
func New() Model {
	return Model{
		menuItems: MainMenu(),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It handles keyboard input for menu navigation.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.menuItems) == 0 || m.cursor < 0 || m.cursor >= len(m.menuItems) {
				break
			}
			item := m.menuItems[m.cursor]
			if item.Action != nil {
				m.quitting = item.IsQuit
				return m, item.Action()
			}
		}
	}

	return m, nil
}

// View implements tea.Model. It renders the full TUI screen.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	s := renderHeader()
	s += renderMainMenu(m.menuItems, m.cursor)
	s += renderFooter()

	return s
}
