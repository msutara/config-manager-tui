# Config Manager TUI — Specification

## 1. Purpose

The Config Manager TUI provides a raspi-config style terminal user interface
for the Config Manager system. It is the primary interactive interface when
running the `cm` binary.

The TUI is a separate Go module imported by the core binary at build time.

## 2. Responsibilities

- **Menu rendering** — render menus with one entry per plugin plus system info.
  Dynamic discovery from the core plugin registry is planned (Phase 2);
  currently menus are static.
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

The TUI is imported by the core binary and run as the main loop:

- Export a public `New()` function returning the concrete `Model` type (which
  implements `tea.Model`).
- Core's `main.go` creates a `tea.Program` with this model and calls `Run()`.

```go
import (
  tea "github.com/charmbracelet/bubbletea"
  tui "github.com/msutara/config-manager-tui"
)

model := tui.New()
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
├── Plugin: Update Management
│   ├── Check for Updates
│   ├── Apply Updates
│   └── Back
├── Plugin: Network Config
│   ├── Show Interfaces
│   ├── Edit Config
│   └── Back
└── Quit
```

Menus are currently static. Dynamic generation from registered plugins is
planned for Phase 2.

## 8. Future Extensions

- Confirmation dialogs for destructive actions.
- Progress indicators for long-running operations.
- Log viewer within the TUI.
- Theme/color customization via config.
