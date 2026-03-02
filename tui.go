// Package tui provides a raspi-config style terminal user interface for
// Config Manager. It is built with Bubble Tea and styled with Lip Gloss.
package tui

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// screen identifies which TUI screen is active.
type screen int

const (
	screenMain    screen = iota // top-level menu
	screenSub                   // plugin sub-menu
	screenDetail                // read-only detail view (press any key to go back)
	screenInput                 // text input for editing a config value
	screenConfirm               // confirmation dialog (y/n)
)

// ConnectionMode indicates how the TUI is connected to the API.
type ConnectionMode int

const (
	// ModeStandalone means the TUI started its own embedded API server.
	ModeStandalone ConnectionMode = iota
	// ModeConnected means the TUI is connected to an external running service.
	ModeConnected
)

// Model is the main Bubble Tea model for the Config Manager TUI.
type Model struct {
	api       *APIClient
	plugins   []PluginInfo
	menuItems []MenuItem
	cursor    int
	quitting  bool
	connMode  ConnectionMode
	theme     Theme

	screen       screen
	screenTitle  string     // title for sub-menu / detail view
	detail       string     // rendered content for detail screen
	parentItems  []MenuItem // saved main menu for returning from sub-menu
	parentCursor int        // saved cursor position for returning from sub-menu
	statusMsg    string     // transient status message
	loading      bool       // true while an async command is in flight

	// Input screen state (screenInput).
	inputBuffer string // current text being edited
	inputPrompt string // label shown above the input field
	inputKey    string // config key being edited (e.g. "schedule")
	inputPlugin string // plugin name for the PUT call

	// Confirmation screen state (screenConfirm).
	confirmTitle  string         // e.g. "Run Full Update?"
	confirmMsg    string         // e.g. "This will update all packages..."
	confirmAction func() tea.Cmd // action to execute on confirmation

	// Status bar data (fetched once on startup).
	hostname  string
	uptimeStr string // human-readable uptime, e.g. "3d 4h"
}

// New returns an initialised TUI model with default API URL.
// Prefer NewWithAPI when the caller knows the configured host/port.
func New(plugins []PluginInfo) Model {
	return NewWithAPI(plugins, "http://localhost:7788")
}

// NewWithAPI returns an initialised TUI model using the given API base URL.
func NewWithAPI(plugins []PluginInfo, apiBaseURL string) Model {
	return NewWithAuth(plugins, apiBaseURL, "")
}

// NewWithAuth returns an initialised TUI model that sends a Bearer token
// with every API request. Pass empty token to disable auth.
func NewWithAuth(plugins []PluginInfo, apiBaseURL, token string) Model {
	m := Model{
		api:      NewAPIClientWithToken(apiBaseURL, token),
		plugins:  plugins,
		screen:   screenMain,
		connMode: ModeStandalone,
		theme:    DefaultTheme(),
	}
	m.menuItems = m.buildMainMenu()
	return m
}

// SetConnectionMode sets the TUI's connection mode indicator.
func (m *Model) SetConnectionMode(mode ConnectionMode) {
	m.connMode = mode
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.api == nil {
		return nil
	}
	return func() tea.Msg {
		info, err := m.api.GetNode()
		if err != nil {
			return nodeInfoMsg{}
		}
		return nodeInfoMsg{hostname: info.Hostname, uptime: info.UptimeSeconds}
	}
}

// nodeInfoMsg carries hostname and uptime fetched at startup.
type nodeInfoMsg struct {
	hostname string
	uptime   int
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

// editInputMsg tells Update to switch to the text input screen.
type editInputMsg struct {
	prompt     string // label shown above the input field
	key        string // config key being edited
	plugin     string // plugin name for the PUT call
	currentVal string // pre-filled value
}

// settingsResultMsg carries the result of a config update.
type settingsResultMsg struct {
	detail string
	err    error
}

// Update implements tea.Model. It handles keyboard input for menu navigation.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case nodeInfoMsg:
		m.hostname = sanitizeText(msg.hostname)
		m.uptimeStr = formatUptime(msg.uptime)
		return m, nil

	case apiResultMsg:
		if msg.err != nil {
			m.detail = fmt.Sprintf("Error: %s\n\nPress any key to go back.", sanitizeText(msg.err.Error()))
		} else {
			m.detail = msg.detail + "\n\nPress any key to go back."
		}
		m.screen = screenDetail
		m.statusMsg = ""
		m.loading = false
		return m, nil

	case settingsResultMsg:
		if msg.err != nil {
			m.detail = fmt.Sprintf("Error: %s\n\nPress any key to go back.", sanitizeText(msg.err.Error()))
		} else {
			m.detail = msg.detail + "\n\nPress any key to go back."
		}
		m.screen = screenDetail
		m.statusMsg = ""
		m.loading = false
		return m, nil

	case editInputMsg:
		m.loading = false
		m.screen = screenInput
		m.inputPrompt = msg.prompt
		m.inputKey = msg.key
		m.inputPlugin = msg.plugin
		m.inputBuffer = msg.currentVal
		m.statusMsg = ""
		return m, nil

	case subMenuMsg:
		m.loading = false
		if msg.title == "" {
			// Empty subMenuMsg = "Back" action.
			m.goBack()
			return m, nil
		}
		m.parentItems = m.menuItems
		m.parentCursor = m.cursor
		m.menuItems = msg.items
		m.cursor = 0
		m.screen = screenSub
		m.screenTitle = msg.title
		m.statusMsg = ""
		return m, nil

	case tea.KeyMsg:
		// ctrl+c always quits, regardless of screen.
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}

		// Input screen handles its own keys.
		if m.screen == screenInput {
			return m.handleInputKey(msg)
		}

		// Confirmation dialog handles y/n.
		if m.screen == screenConfirm {
			return m.handleConfirmKey(msg)
		}

		// In detail view, any other key goes back.
		if m.screen == screenDetail {
			m.goBack()
			return m, nil
		}

		switch msg.String() {
		case "q":
			if m.screen == screenSub {
				if m.loading {
					break // don't navigate away while API call in flight
				}
				m.goBack()
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit

		case "esc", "backspace":
			if m.screen == screenSub {
				if m.loading {
					break
				}
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
				if item.NeedsConfirm {
					m.confirmTitle = item.Title + "?"
					m.confirmMsg = item.ConfirmMsg
					m.confirmAction = item.Action
					m.screen = screenConfirm
					return m, nil
				}
				m.quitting = item.IsQuit
				m.loading = true
				m.statusMsg = "Loading…"
				return m, item.Action()
			}
		}
	}

	return m, nil
}

// handleInputKey processes key events while the text input screen is active.
func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	switch msg.Type {
	case tea.KeyEsc:
		m.goBack()
		return m, nil
	case tea.KeyEnter:
		value := m.inputBuffer
		key := m.inputKey
		pluginName := m.inputPlugin
		api := m.api
		m.loading = true
		m.statusMsg = "Saving…"
		return m, func() tea.Msg {
			res, err := api.UpdatePluginSetting(pluginName, key, value)
			if err != nil {
				return settingsResultMsg{err: err}
			}
			detail := formatSettingsResult(key, value, res)
			return settingsResultMsg{detail: detail}
		}
	case tea.KeyBackspace:
		if len(m.inputBuffer) > 0 {
			_, size := utf8.DecodeLastRuneInString(m.inputBuffer)
			m.inputBuffer = m.inputBuffer[:len(m.inputBuffer)-size]
		}
		return m, nil
	case tea.KeySpace:
		m.inputBuffer += " "
		return m, nil
	default:
		if msg.Type == tea.KeyRunes {
			m.inputBuffer += string(msg.Runes)
		}
		return m, nil
	}
}

// handleConfirmKey processes key events in the confirmation dialog.
func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		action := m.confirmAction
		m.screen = screenSub
		if m.parentItems == nil {
			m.screen = screenMain
		}
		m.confirmTitle = ""
		m.confirmMsg = ""
		m.confirmAction = nil
		if action == nil {
			return m, nil
		}
		m.loading = true
		m.statusMsg = "Loading…"
		return m, action()
	case "n", "N", "esc", "q":
		m.screen = screenSub
		if m.parentItems == nil {
			m.screen = screenMain
		}
		m.confirmTitle = ""
		m.confirmMsg = ""
		m.confirmAction = nil
		return m, nil
	}
	return m, nil
}

// goBack returns to the previous screen.
func (m *Model) goBack() {
	switch m.screen {
	case screenSub:
		m.menuItems = m.parentItems
		m.parentItems = nil
		m.cursor = m.parentCursor
		m.parentCursor = 0
		m.screen = screenMain
		m.screenTitle = ""
	case screenDetail:
		if m.parentItems != nil {
			m.screen = screenSub
		} else {
			m.screen = screenMain
		}
	case screenInput:
		if m.parentItems != nil {
			m.screen = screenSub
		} else {
			m.screen = screenMain
		}
		m.inputBuffer = ""
		m.inputPrompt = ""
		m.inputKey = ""
		m.inputPlugin = ""
	}
	m.detail = ""
	m.statusMsg = ""
	m.loading = false
}

// View implements tea.Model. It renders the full TUI screen.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	switch m.screen {
	case screenDetail:
		return m.viewDetail()
	case screenInput:
		return m.viewInput()
	case screenConfirm:
		return m.viewConfirm()
	case screenSub:
		return m.viewSubMenu()
	default:
		return m.viewMainMenu()
	}
}

func (m Model) viewMainMenu() string {
	var b strings.Builder
	b.WriteString(renderHeader(m.theme))                          //nolint:errcheck // writes to strings.Builder
	b.WriteString(renderMainMenu(m.menuItems, m.cursor, m.theme)) //nolint:errcheck // writes to strings.Builder
	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n") //nolint:errcheck // writes to strings.Builder
	}
	b.WriteString(renderFooter(m.connMode, m.hostname, m.uptimeStr, m.theme)) //nolint:errcheck // writes to strings.Builder
	return b.String()
}

func (m Model) viewSubMenu() string {
	var b strings.Builder
	b.WriteString(renderHeader(m.theme))                                           //nolint:errcheck // writes to strings.Builder
	b.WriteString(renderPluginView(m.screenTitle, m.menuItems, m.cursor, m.theme)) //nolint:errcheck // writes to strings.Builder
	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n") //nolint:errcheck // writes to strings.Builder
	}
	b.WriteString(renderSubFooter(m.connMode, m.hostname, m.uptimeStr, m.theme)) //nolint:errcheck // writes to strings.Builder
	return b.String()
}

func (m Model) viewDetail() string {
	var b strings.Builder
	b.WriteString(renderHeader(m.theme)) //nolint:errcheck // writes to strings.Builder
	if m.screenTitle != "" {
		b.WriteString("  " + m.theme.Header.Render(m.screenTitle) + "\n\n") //nolint:errcheck // writes to strings.Builder
	}
	for _, line := range strings.Split(m.detail, "\n") {
		b.WriteString("  " + line + "\n") //nolint:errcheck // writes to strings.Builder
	}
	return b.String()
}

func (m Model) viewInput() string {
	var b strings.Builder
	b.WriteString(renderHeader(m.theme)) //nolint:errcheck // writes to strings.Builder
	if m.screenTitle != "" {
		b.WriteString("  " + m.theme.Header.Render(m.screenTitle) + "\n\n") //nolint:errcheck // writes to strings.Builder
	}
	b.WriteString("  " + m.inputPrompt + "\n\n")                  //nolint:errcheck // writes to strings.Builder
	b.WriteString("  > " + sanitizeText(m.inputBuffer) + "█\n\n") //nolint:errcheck // writes to strings.Builder
	b.WriteString("  Enter: save  Esc: cancel\n")                 //nolint:errcheck // writes to strings.Builder
	if m.statusMsg != "" {
		b.WriteString("\n  " + m.statusMsg + "\n") //nolint:errcheck // writes to strings.Builder
	}
	return b.String()
}

func (m Model) viewConfirm() string {
	var b strings.Builder
	b.WriteString(renderHeader(m.theme))                                 //nolint:errcheck // writes to strings.Builder
	b.WriteString("  " + m.theme.Header.Render(m.confirmTitle) + "\n\n") //nolint:errcheck // writes to strings.Builder
	if m.confirmMsg != "" {
		b.WriteString("  " + m.confirmMsg + "\n\n") //nolint:errcheck // writes to strings.Builder
	}
	yes := m.theme.ConfirmYes.Render("[Y] Yes")
	no := m.theme.ConfirmNo.Render("[N] No")
	b.WriteString("  " + yes + "    " + no + "\n") //nolint:errcheck // writes to strings.Builder
	return b.String()
}

// sanitizeValue converts an arbitrary config value to a sanitized string.
// Composite types (slices, maps) are formatted then sanitized to prevent
// terminal escape injection from nested string elements.
func sanitizeValue(v any) string {
	if s, ok := v.(string); ok {
		return sanitizeText(s)
	}
	return sanitizeText(fmt.Sprintf("%v", v))
}

// formatSettingsResult builds a human-readable detail string from a config update.
func formatSettingsResult(key, value string, res *PluginSettingsUpdateResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Updated %q to %q\n", sanitizeText(key), sanitizeText(value)) //nolint:errcheck // writes to strings.Builder
	if res.Warning != "" {
		fmt.Fprintf(&b, "\nWarning: %s\n", sanitizeText(res.Warning)) //nolint:errcheck // writes to strings.Builder
	}
	b.WriteString("\nCurrent settings:\n") //nolint:errcheck // writes to strings.Builder
	keys := make([]string, 0, len(res.Config))
	for k := range res.Config {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "  %-20s %s\n", sanitizeText(k)+":", sanitizeValue(res.Config[k])) //nolint:errcheck // writes to strings.Builder
	}
	return b.String()
}
