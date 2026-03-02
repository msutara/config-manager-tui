package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds all visual styles used by the TUI. Create one with
// DefaultTheme(). YAML-based theme loading is planned for a future release.
type Theme struct {
	// Header is the style for the top title bar.
	Header lipgloss.Style
	// Footer is the style for footer help text.
	Footer lipgloss.Style
	// Selected is the style for the currently highlighted menu item.
	Selected lipgloss.Style
	// Normal is the style for non-selected menu items.
	Normal lipgloss.Style
	// Description is the style for item descriptions below titles.
	Description lipgloss.Style

	// Cursor is the string shown before the selected item (e.g. "▸").
	Cursor string
	// CursorStyle is the lipgloss style applied to the cursor glyph.
	CursorStyle lipgloss.Style
	// Separator is the repeating character for horizontal rules (e.g. "─").
	Separator string
	// SepWidth is the number of times Separator is repeated.
	SepWidth int

	// ConnBadgeText is the label shown when connected to a service.
	ConnBadgeText string
	// ConnBadgeStyle is the style for the connected badge.
	ConnBadgeStyle lipgloss.Style
	// StandBadgeText is the label shown in standalone mode.
	StandBadgeText string
	// StandBadgeStyle is the style for the standalone badge.
	StandBadgeStyle lipgloss.Style

	// ConfirmYes is the style for the [Y] Yes button in confirmation dialogs.
	ConfirmYes lipgloss.Style
	// ConfirmNo is the style for the [N] No button in confirmation dialogs.
	ConfirmNo lipgloss.Style

	// StatusBar is the style for the hostname/uptime bar in the footer.
	StatusBar lipgloss.Style

	// Spinner is the style for progress spinners (Phase 4, reserved).
	Spinner lipgloss.Style
}

// DefaultTheme returns the built-in colour scheme matching the original
// hardcoded styles.
func DefaultTheme() Theme {
	return Theme{
		Header:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		Footer:      lipgloss.NewStyle().Faint(true),
		Selected:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")),
		Normal:      lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
		Description: lipgloss.NewStyle().Faint(true),

		Cursor:      "▸ ",
		CursorStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
		Separator:   "─",
		SepWidth:    40,

		ConnBadgeText:   "● connected",
		ConnBadgeStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		StandBadgeText:  "● standalone",
		StandBadgeStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),

		ConfirmYes: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),
		ConfirmNo:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),

		StatusBar: lipgloss.NewStyle().Faint(true),

		Spinner: lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
	}
}
