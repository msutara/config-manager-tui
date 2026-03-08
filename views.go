package tui

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

// renderHeader returns the styled header block for the TUI.
func renderHeader(th Theme) string {
	title := th.Header.Render("Config Manager")
	separator := th.Footer.Render(strings.Repeat(th.Separator, th.SepWidth))
	return "\n  " + title + "\n  " + separator + "\n\n"
}

// renderFooter returns the styled footer with key hints and connection mode badge.
func renderFooter(mode ConnectionMode, hostname, uptime string, th Theme) string {
	badge := modeBadge(mode, th)
	hints := th.Footer.Render("↑/↓: navigate • enter: select • q: quit")
	status := renderStatusBar(hostname, uptime, th)
	return "\n  " + hints + "  " + status + badge + "\n"
}

// renderSubFooter returns a footer with back-navigation hints and connection mode badge.
func renderSubFooter(mode ConnectionMode, hostname, uptime string, th Theme) string {
	badge := modeBadge(mode, th)
	hints := th.Footer.Render("↑/↓: navigate • enter: select • esc/q/backspace: back")
	status := renderStatusBar(hostname, uptime, th)
	return "\n  " + hints + "  " + status + badge + "\n"
}

// renderInputFooter returns a footer for the input screen with save/cancel hints and connection badge.
func renderInputFooter(mode ConnectionMode, hostname, uptime string, th Theme) string {
	badge := modeBadge(mode, th)
	hints := th.Footer.Render("enter: save • esc: cancel")
	status := renderStatusBar(hostname, uptime, th)
	return "\n  " + hints + "  " + status + badge + "\n"
}

// renderStatusBar returns a formatted hostname and uptime string for the footer.
func renderStatusBar(hostname, uptime string, th Theme) string {
	if hostname == "" {
		return ""
	}
	s := hostname
	if uptime != "" {
		s += " | up " + uptime
	}
	return th.StatusBar.Render(s) + "  "
}

func modeBadge(mode ConnectionMode, th Theme) string {
	if mode == ModeConnected {
		return th.ConnBadgeStyle.Render(th.ConnBadgeText)
	}
	return th.StandBadgeStyle.Render(th.StandBadgeText)
}

// formatJobHistory renders a table-like view of job execution history.
func formatJobHistory(jobID string, runs []JobRun) string {
	if len(runs) == 0 {
		return fmt.Sprintf("Job History: %s\n\nNo executions recorded.", sanitizeText(jobID))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Job History: %s\n\n", sanitizeText(jobID))                       //nolint:errcheck // writes to strings.Builder
	fmt.Fprintf(&b, "%-4s  %-20s  %-10s  %s\n", "  ", "Started", "Duration", "Error") //nolint:errcheck // writes to strings.Builder
	b.WriteString(strings.Repeat("─", 60) + "\n")                                     //nolint:errcheck // writes to strings.Builder

	for _, r := range runs {
		icon := "•"
		switch r.Status {
		case "completed":
			icon = "✓"
		case "failed":
			icon = "✗"
		case "running":
			icon = "⟳"
		}

		started := sanitizeText(r.StartedAt)
		if utf8.RuneCountInString(started) > 19 {
			started = string([]rune(started)[:19])
		}

		duration := sanitizeText(r.Duration)
		if duration == "" {
			duration = "-"
		}

		errMsg := "-"
		if r.Error != "" {
			errMsg = sanitizeText(r.Error)
			if utf8.RuneCountInString(errMsg) > 30 {
				errMsg = string([]rune(errMsg)[:30]) + "…"
			}
		}

		fmt.Fprintf(&b, "%-4s  %-20s  %-10s  %s\n", icon, started, duration, errMsg) //nolint:errcheck // writes to strings.Builder
	}

	return b.String()
}

// renderMainMenu renders the list of menu items with a cursor indicator.
func renderMainMenu(items []MenuItem, cursor int, th Theme) string {
	var b strings.Builder
	cursorGlyph := th.CursorStyle.Render(th.Cursor)
	blankGlyph := strings.Repeat(" ", lipgloss.Width(cursorGlyph))
	for i, item := range items {
		if i == cursor {
			title := th.Selected.Render(item.Title)
			desc := th.Description.Render(item.Description)
			fmt.Fprintf(&b, "  %s%s  %s\n", cursorGlyph, title, desc) //nolint:errcheck // writes to strings.Builder
		} else {
			title := th.Normal.Render(item.Title)
			desc := th.Description.Render(item.Description)
			fmt.Fprintf(&b, "  %s%s  %s\n", blankGlyph, title, desc) //nolint:errcheck // writes to strings.Builder
		}
	}
	return b.String()
}

// renderPluginView renders a plugin-specific submenu body (without footer).
func renderPluginView(pluginName string, items []MenuItem, cursor int, th Theme) string {
	var b strings.Builder
	name := th.Header.Render(pluginName)
	cursorGlyph := th.CursorStyle.Render(th.Cursor)
	blankGlyph := strings.Repeat(" ", lipgloss.Width(cursorGlyph))
	fmt.Fprintf(&b, "\n  %s\n\n", name) //nolint:errcheck // writes to strings.Builder
	for i, item := range items {
		if i == cursor {
			title := th.Selected.Render(item.Title)
			desc := th.Description.Render(item.Description)
			fmt.Fprintf(&b, "  %s%s  %s\n", cursorGlyph, title, desc) //nolint:errcheck // writes to strings.Builder
		} else {
			title := th.Normal.Render(item.Title)
			desc := th.Description.Render(item.Description)
			fmt.Fprintf(&b, "  %s%s  %s\n", blankGlyph, title, desc) //nolint:errcheck // writes to strings.Builder
		}
	}
	return b.String()
}
