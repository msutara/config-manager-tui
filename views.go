package tui

import "fmt"

// renderHeader returns the styled header block for the TUI.
func renderHeader() string {
	return "\n  Config Manager\n\n"
}

// renderFooter returns the styled footer with key hints.
func renderFooter() string {
	return "\n  ↑/↓: navigate • enter: select • q: quit\n"
}

// renderMainMenu renders the list of menu items with a cursor indicator.
func renderMainMenu(items []MenuItem, cursor int) string {
	s := ""
	for i, item := range items {
		indicator := "  "
		if i == cursor {
			indicator = "> "
		}
		s += fmt.Sprintf("  %s%s — %s\n", indicator, item.Title, item.Description)
	}
	return s
}

// renderPluginView renders a plugin-specific submenu. This is a stub that will
// be expanded when plugin integration is implemented.
func renderPluginView(pluginName string, items []MenuItem, cursor int) string {
	s := fmt.Sprintf("\n  %s\n\n", pluginName)
	for i, item := range items {
		indicator := "  "
		if i == cursor {
			indicator = "> "
		}
		s += fmt.Sprintf("  %s%s — %s\n", indicator, item.Title, item.Description)
	}
	s += renderFooter()
	return s
}
