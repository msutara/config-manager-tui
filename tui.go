// Package tui provides a raspi-config style terminal user interface for
// Config Manager. It is built with Bubble Tea and styled with Lip Gloss.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// screen identifies which TUI screen is active.
type screen int

const (
	screenMain   screen = iota // top-level menu
	screenSub                  // plugin sub-menu
	screenDetail               // read-only detail view (press any key to go back)
)

// Model is the main Bubble Tea model for the Config Manager TUI.
type Model struct {
	api       *APIClient
	plugins   []PluginInfo
	menuItems []MenuItem
	cursor    int
	quitting  bool

	screen      screen
	screenTitle string     // title for sub-menu / detail view
	detail      string     // rendered content for detail screen
	parentItems []MenuItem // saved main menu for returning from sub-menu
	statusMsg   string     // transient status message
	loading     bool       // true while an async command is in flight
}

// New returns an initialised TUI model with default API URL.
// Prefer NewWithAPI when the caller knows the configured host/port.
func New(plugins []PluginInfo) Model {
	return NewWithAPI(plugins, "http://localhost:7788")
}

// NewWithAPI returns an initialised TUI model using the given API base URL.
func NewWithAPI(plugins []PluginInfo, apiBaseURL string) Model {
	m := Model{
		api:     NewAPIClient(apiBaseURL),
		plugins: plugins,
		screen:  screenMain,
	}
	m.menuItems = m.buildMainMenu()
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// apiResultMsg carries the result of an async API call back to Update.
type apiResultMsg struct {
	detail string
	err    error
}

// subMenuMsg tells Update to switch to a sub-menu.
type subMenuMsg struct {
	title string
	items []MenuItem
}

// Update implements tea.Model. It handles keyboard input for menu navigation.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case apiResultMsg:
		if msg.err != nil {
			m.detail = fmt.Sprintf("Error: %v\n\nPress any key to go back.", msg.err)
		} else {
			m.detail = msg.detail + "\n\nPress any key to go back."
		}
		m.screen = screenDetail
		m.statusMsg = ""
		m.loading = false
		return m, nil

	case subMenuMsg:
		m.loading = false
		if msg.title == "" {
			// Empty subMenuMsg = "Back" action.
			m.goBack()
			return m, nil
		}
		m.parentItems = m.menuItems
		m.menuItems = msg.items
		m.cursor = 0
		m.screen = screenSub
		m.screenTitle = msg.title
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		// In detail view, any key goes back.
		if m.screen == screenDetail {
			m.goBack()
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			if m.screen == screenSub {
				m.goBack()
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "esc", "backspace":
			if m.screen == screenSub {
				m.goBack()
				return m, nil
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		case "enter":
			if m.loading {
				break // prevent double-dispatch while a command is in flight
			}
			if len(m.menuItems) == 0 || m.cursor < 0 || m.cursor >= len(m.menuItems) {
				break
			}
			item := m.menuItems[m.cursor]
			if item.Action != nil {
				m.quitting = item.IsQuit
				m.loading = true
				m.statusMsg = "Loading…"
				return m, item.Action()
			}
		}
	}

	return m, nil
}

// goBack returns to the previous screen.
func (m *Model) goBack() {
	switch m.screen {
	case screenSub:
		m.menuItems = m.parentItems
		m.parentItems = nil
		m.cursor = 0
		m.screen = screenMain
		m.screenTitle = ""
	case screenDetail:
		if m.parentItems != nil {
			m.screen = screenSub
		} else {
			m.screen = screenMain
		}
	}
	m.detail = ""
	m.statusMsg = ""
}

// View implements tea.Model. It renders the full TUI screen.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	switch m.screen {
	case screenDetail:
		return m.viewDetail()
	case screenSub:
		return m.viewSubMenu()
	default:
		return m.viewMainMenu()
	}
}

func (m Model) viewMainMenu() string {
	var b strings.Builder
	b.WriteString(renderHeader())                        //nolint:errcheck // writes to strings.Builder
	b.WriteString(renderMainMenu(m.menuItems, m.cursor)) //nolint:errcheck // writes to strings.Builder
	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n") //nolint:errcheck // writes to strings.Builder
	}
	b.WriteString(renderFooter()) //nolint:errcheck // writes to strings.Builder
	return b.String()
}

func (m Model) viewSubMenu() string {
	var b strings.Builder
	b.WriteString(renderHeader())                                         //nolint:errcheck // writes to strings.Builder
	b.WriteString(renderPluginView(m.screenTitle, m.menuItems, m.cursor)) //nolint:errcheck // writes to strings.Builder
	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n") //nolint:errcheck // writes to strings.Builder
	}
	return b.String()
}

func (m Model) viewDetail() string {
	var b strings.Builder
	b.WriteString(renderHeader()) //nolint:errcheck // writes to strings.Builder
	if m.screenTitle != "" {
		b.WriteString("  " + headerStyle.Render(m.screenTitle) + "\n\n") //nolint:errcheck // writes to strings.Builder
	}
	for _, line := range strings.Split(m.detail, "\n") {
		b.WriteString("  " + line + "\n") //nolint:errcheck // writes to strings.Builder
	}
	return b.String()
}
