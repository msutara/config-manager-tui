# Config Manager TUI — Specification

## 1. Purpose

The Config Manager TUI provides a raspi-config style terminal user interface
for the Config Manager system. It is the primary interactive interface when
running the `cm` binary.

The TUI is a separate Go module imported by the core binary at build time.

---

## 2. Responsibilities

- **Menu rendering:**
  - Discover registered plugins from the core plugin registry.
  - Render dynamic menus based on plugin metadata (name, description, actions).
  - Provide a top-level menu with one entry per plugin plus system info.

- **User interaction:**
  - Arrow keys for navigation, Enter to select, q to quit.
  - Plugin-specific submenus for triggering actions.

- **Action triggers:**
  - Invoke plugin operations when the user selects a menu action.
  - Display operation results and status in the TUI.

- **Integration:**
  - Export a public `New()` function returning the concrete `Model` type (which implements `tea.Model`).
  - Core's `main.go` creates a `tea.Program` with this model and calls `Run()`.

---

## 3. Non-responsibilities

The TUI does **not**:

- Implement plugin logic (that lives in plugin repos).
- Handle HTTP/API concerns (that lives in the core).
- Manage configuration loading (core handles that).
- Run as a standalone binary (it is compiled into the core).

---

## 4. Technology

- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) — Elm-architecture TUI framework.
- **Lip Gloss** (`github.com/charmbracelet/lipgloss`) — Styling and layout.
- **Go 1.22+** — module compatible with the core binary.

---

## 5. Architecture

- **`tui.go`** — Main Bubble Tea model (`Model` struct, `New()`, `Init()`, `Update()`, `View()`).
- **`menu.go`** — Menu data structures (`MenuItem`) and menu builders.
- **`views.go`** — View rendering functions (header, footer, main menu, plugin views).

The core binary imports this package and runs it as the main loop:

```go
import (
	tea "github.com/charmbracelet/bubbletea"
	tui "github.com/msutara/config-manager-tui"
)

model := tui.New()
p := tea.NewProgram(model)
p.Run()
```

---

## 6. Key bindings

| Key       | Action                        |
|-----------|-------------------------------|
| ↑ / k     | Move cursor up                |
| ↓ / j     | Move cursor down              |
| Enter     | Select menu item              |
| q / ctrl+c | Quit the TUI                  |

---

## 7. Menu structure

```
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

---

## 8. Future extensions

- Confirmation dialogs for destructive actions.
- Progress indicators for long-running operations.
- Log viewer within the TUI.
- Theme/color customization via config.
