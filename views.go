package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	footerStyle   = lipgloss.NewStyle().Faint(true)
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	descStyle     = lipgloss.NewStyle().Faint(true)
	cursorGlyph   = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("▸ ")
	blankGlyph    = "  "
)

// renderHeader returns the styled header block for the TUI.
func renderHeader() string {
	title := headerStyle.Render("Config Manager")
	separator := lipgloss.NewStyle().Faint(true).Render(strings.Repeat("─", 40))
	return "\n  " + title + "\n  " + separator + "\n\n"
}

// renderFooter returns the styled footer with key hints.
func renderFooter() string {
	return "\n  " + footerStyle.Render("↑/↓: navigate • enter: select • q: quit") + "\n"
}

// renderSubFooter returns a footer with back-navigation hints.
func renderSubFooter() string {
	return "\n  " + footerStyle.Render("↑/↓: navigate • enter: select • esc/q/backspace: back") + "\n"
}

// renderMainMenu renders the list of menu items with a cursor indicator.
func renderMainMenu(items []MenuItem, cursor int) string {
	var b strings.Builder
	for i, item := range items {
		if i == cursor {
			title := selectedStyle.Render(item.Title)
			desc := descStyle.Render(item.Description)
			fmt.Fprintf(&b, "  %s%s  %s\n", cursorGlyph, title, desc) //nolint:errcheck // writes to strings.Builder
		} else {
			title := normalStyle.Render(item.Title)
			desc := descStyle.Render(item.Description)
			fmt.Fprintf(&b, "  %s%s  %s\n", blankGlyph, title, desc) //nolint:errcheck // writes to strings.Builder
		}
	}
	return b.String()
}

// renderPluginView renders a plugin-specific submenu body (without footer).
func renderPluginView(pluginName string, items []MenuItem, cursor int) string {
	var b strings.Builder
	name := headerStyle.Render(pluginName)
	fmt.Fprintf(&b, "\n  %s\n\n", name) //nolint:errcheck // writes to strings.Builder
	for i, item := range items {
		if i == cursor {
			title := selectedStyle.Render(item.Title)
			desc := descStyle.Render(item.Description)
			fmt.Fprintf(&b, "  %s%s  %s\n", cursorGlyph, title, desc) //nolint:errcheck // writes to strings.Builder
		} else {
			title := normalStyle.Render(item.Title)
			desc := descStyle.Render(item.Description)
			fmt.Fprintf(&b, "  %s%s  %s\n", blankGlyph, title, desc) //nolint:errcheck // writes to strings.Builder
		}
	}
	return b.String()
}
