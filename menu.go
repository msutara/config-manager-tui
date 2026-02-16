package tui

import tea "github.com/charmbracelet/bubbletea"

// MenuItem represents a single entry in a TUI menu.
type MenuItem struct {
	Title       string
	Description string
	Action      func() tea.Cmd
	IsQuit      bool // when true, selecting this item exits the TUI
}

// MainMenu returns the top-level menu items. In the future this will be
// populated dynamically from the core plugin registry.
func MainMenu() []MenuItem {
	return []MenuItem{
		{
			Title:       "System Info",
			Description: "View system information and status",
		},
		{
			Title:       "Plugins",
			Description: "Manage installed plugins",
		},
		{
			Title:       "Quit",
			Description: "Exit Config Manager",
			Action:      func() tea.Cmd { return tea.Quit },
			IsQuit:      true,
		},
	}
}
