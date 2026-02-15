package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	footerStyle = lipgloss.NewStyle().Faint(true)
)

// renderHeader returns the styled header block for the TUI.
func renderHeader() string {
	return "\n  " + headerStyle.Render("Config Manager") + "\n\n"
}

// renderFooter returns the styled footer with key hints.
func renderFooter() string {
	return "\n  " + footerStyle.Render("↑/↓: navigate • enter: select • q: quit") + "\n"
}

// renderMainMenu renders the list of menu items with a cursor indicator.
func renderMainMenu(items []MenuItem, cursor int) string {
	var b strings.Builder
	for i, item := range items {
		indicator := "  "
		if i == cursor {
			indicator = "> "
		}
		fmt.Fprintf(&b, "  %s%s — %s\n", indicator, item.Title, item.Description)
	}
	return b.String()
}

// renderPluginView renders a plugin-specific submenu. This is a stub that will
// be expanded when plugin integration is implemented.
//
//nolint:unused // stub — will be called when plugin submenus are wired
func renderPluginView(pluginName string, items []MenuItem, cursor int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n  %s\n\n", pluginName)
	for i, item := range items {
		indicator := "  "
		if i == cursor {
			indicator = "> "
		}
		fmt.Fprintf(&b, "  %s%s — %s\n", indicator, item.Title, item.Description)
	}
	b.WriteString(renderFooter())
	return b.String()
}
