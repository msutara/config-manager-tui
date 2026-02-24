package tui

import tea "github.com/charmbracelet/bubbletea"

// PluginInfo describes a registered plugin for menu rendering. The core binary
// populates this from its plugin registry — the TUI has no direct dependency
// on the core plugin package.
type PluginInfo struct {
	Name        string
	Description string
}

// MenuItem represents a single entry in a TUI menu.
type MenuItem struct {
	Title       string
	Description string
	Action      func() tea.Cmd
	IsQuit      bool // when true, selecting this item exits the TUI
}

// MainMenu returns the top-level menu items. When plugins is non-empty, one
// entry is added per plugin between "System Info" and "Quit".
func MainMenu(plugins []PluginInfo) []MenuItem {
	items := []MenuItem{
		{
			Title:       "System Info",
			Description: "View system information and status",
		},
	}

	for _, p := range plugins {
		items = append(items, MenuItem{
			Title:       p.Name,
			Description: p.Description,
		})
	}

	items = append(items, MenuItem{
		Title:       "Quit",
		Description: "Exit Config Manager",
		Action:      func() tea.Cmd { return tea.Quit },
		IsQuit:      true,
	})

	return items
}
