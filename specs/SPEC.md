# Config Manager TUI — Specification

## 1. Purpose

The Config Manager TUI provides a raspi-config style terminal user interface
for the Config Manager system. Once wired into the core binary (Phase 2), it
will serve as the primary interactive interface.

The TUI is a separate Go module that will be imported by the core binary at
build time (Phase 2).

## 2. Responsibilities

- **Menu rendering** — render menus with one entry per plugin plus system info.
  The main menu is built dynamically from `[]PluginInfo` passed to `New()`.
- **User interaction** — arrow keys for navigation, Enter to select, q to
  quit. Plugin-specific submenus for triggering actions.
- **Action triggering** — invoke plugin operations when the user selects a
  menu action.
- **Result display** — show operation results and status in the TUI.

## 3. Non-responsibilities

The TUI does **not**:

- Serve a REST API (that lives in the core).
- Implement plugin logic (that lives in plugin repos).
- Make direct system calls (plugins handle that).
- Manage configuration loading (core handles that).

## 4. Integration

The TUI will be imported by the core binary and run as the main loop
(Phase 2). The integration pattern:

- Export a public `New(plugins []PluginInfo)` function returning the concrete
  `Model` type (which implements `tea.Model`).
- Core's `main.go` converts its plugin registry to `[]tui.PluginInfo` and
  passes it to `New()`.
- Core creates a `tea.Program` with this model and calls `Run()`.

```go
import (
  tea "github.com/charmbracelet/bubbletea"
  tui "github.com/msutara/config-manager-tui"
)

plugins := []tui.PluginInfo{
  {Name: "Update Management", Description: "OS updates"},
  {Name: "Network Config", Description: "Network interfaces"},
}
model := tui.New(plugins)
p := tea.NewProgram(model)
p.Run()
```

## 5. Technology

- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) — Elm-architecture
  TUI framework.
- **Lip Gloss** (`github.com/charmbracelet/lipgloss`) — Styling and layout.
- **Go 1.22+** — module compatible with the core binary.

## 6. Key Bindings

| Key        | Action                       |
|------------|------------------------------|
| ↑ / k      | Move cursor up               |
| ↓ / j      | Move cursor down             |
| Enter      | Select menu item             |
| q / ctrl+c | Quit the TUI                 |

## 7. Menu Structure

```text
Config Manager
├── System Info
├── Update Management
│   ├── Check for Updates
│   ├── Apply Updates
│   └── Back
├── Network Config
│   ├── Show Interfaces
│   ├── Edit Config
│   └── Back
└── Quit
```

The main menu is built dynamically from `[]PluginInfo` passed to `New()`.
Plugin-specific submenus (e.g., "Check for Updates") are a future extension.

## 8. Visual Style

- Header displays "Config Manager" in bold blue with a separator line.
- Selected menu item uses a `▸` cursor glyph and bold cyan text.
- Unselected items use muted white text.
- Descriptions are rendered in faint style beside each title.
- Footer shows key hints in faint text.

## 9. Future Extensions

- Confirmation dialogs for destructive actions.
- Progress indicators for long-running operations.
- Log viewer within the TUI.
- Theme/color customization via config.
