# Config Manager TUI — Specification

## 1. Purpose

The Config Manager TUI provides a raspi-config style terminal user interface
for the Config Manager system. It runs as part of the core binary and serves
as the primary interactive interface.

The TUI is a separate Go module imported by the core binary at build time.

## 2. Responsibilities

- **Menu rendering** — render menus with one entry per plugin plus system info.
  The main menu is built dynamically from `[]PluginInfo` passed to
  `New()` or `NewWithAPI()`.
- **User interaction** — arrow keys for navigation, Enter to select, q to
  quit. Plugin-specific submenus for triggering actions.
- **Confirmation dialogs** — destructive actions (system updates, POST
  endpoints) require explicit `y`/`Y` confirmation before execution. The
  Enter key is excluded from the confirm dialog to prevent accidental
  double-tap bypass.
- **Action triggering** — invoke plugin operations when the user selects a
  menu action.
- **Result display** — show operation results and status in the TUI.
- **Status bar** — the footer displays the node hostname and uptime,
  fetched once at startup via `GET /api/v1/node`. Gracefully omitted when
  the API is unreachable.
- **Theme system** — primary colours, glyphs, and badge text are defined in a
  `Theme` struct. `DefaultTheme()` provides the built-in look. Render
  functions generally accept `Theme` as a parameter (no global style variables
  for core elements, with minor inline styles permitted for simple separators
  or spacing). Custom themes are loaded from YAML via `ThemeFromYAML([]byte)`.
  Four built-in themes ship with the binary: `default`, `high-contrast`,
  `nord`, and `solarized-dark`. Partial overrides are supported — only the
  fields present in the YAML are changed; everything else inherits from
  `DefaultTheme()`. Use `SetTheme()` on the model before `Run()` to apply.
- **Config editing** — edit plugin settings via the core settings API
  (`PUT /api/v1/plugins/{name}/settings`). Supports text input (schedule),
  toggles (auto-security), and cycling enum values (security source).
  All editing actions fetch the current value from the API before acting
  to avoid stale state. If the fetch fails or the value is missing/invalid,
  an error is shown instead of proceeding with a potentially incorrect
  mutation. Menu descriptions show "(unavailable)" or "(unknown)" when
  the config endpoint is unreachable.

## 3. Non-responsibilities

The TUI does **not**:

- Serve a REST API (that lives in the core).
- Implement plugin logic (that lives in plugin repos).
- Make direct system calls (plugins handle that).
- Manage configuration loading (core handles that).

## 4. Integration

The TUI is imported by the core binary and runs as the main interactive loop.
The integration pattern:

- Export `New(plugins []PluginInfo)` and `NewWithAPI(plugins []PluginInfo, apiBaseURL string)`
  returning the concrete `Model` type (which implements `tea.Model`).
- Core's `main.go` converts its plugin registry to `[]tui.PluginInfo` and
  passes it to `NewWithAPI()` with the configured API base URL.
- Core creates a `tea.Program` with this model and calls `Run()`.

```go
import (
  tea "github.com/charmbracelet/bubbletea"
  tui "github.com/msutara/config-manager-tui"
)

plugins := []tui.PluginInfo{
  {Name: "update", Description: "OS and package update management"},
  {Name: "network", Description: "Network interface configuration"},
}
model := tui.NewWithAPI(plugins, "http://localhost:7788")
p := tea.NewProgram(model)
p.Run()
```

## 5. Technology

- **Bubble Tea** (`github.com/charmbracelet/bubbletea`) — Elm-architecture
  TUI framework.
- **Lip Gloss** (`github.com/charmbracelet/lipgloss`) — Styling and layout.
- **Go 1.22+** — module compatible with the core binary.

## 6. Key Bindings

| Key             | Action                                             |
| --------------- | -------------------------------------------------- |
| ↑ / k           | Move cursor up                                     |
| ↓ / j           | Move cursor down                                   |
| Enter           | Select menu item                                   |
| esc/q/backspace | Go back (in sub-menus); any key goes back (detail) |
| q               | Quit the TUI (from main menu)                      |
| ctrl+c          | Quit the TUI (from any screen)                     |

### Input Screen Keys

| Key       | Action                    |
| --------- | ------------------------- |
| Type text | Appends to the input      |
| Backspace | Delete last character     |
| Enter     | Save the value            |
| Esc       | Cancel and return to menu |

## 7. Menu Structure

```text
Config Manager
├── System Info
├── Update Manager
│   ├── Check Status
│   ├── Full Update
│   ├── Security Update  (only when distro has a separate security source)
│   ├── View Logs
│   ├── View Settings
│   ├── Edit Schedule          (opens text input screen)
│   ├── Toggle Auto-Security   (immediate toggle ON/OFF)
│   ├── Change Security Source  (cycles available ↔ always)
│   └── Back
├── Network Manager
│   ├── List Interfaces
│   ├── Network Status
│   ├── DNS Settings
│   └── Back
└── Quit
```

The main menu is built dynamically from `[]PluginInfo` passed to
`New()` or `NewWithAPI()`.
Plugin-specific submenus are navigated via Enter and exited with
esc/q/backspace.

## 8. Visual Style

- Header displays "Config Manager" in bold blue with a separator line.
- Selected menu item uses a `▸` cursor glyph and bold cyan text.
- Unselected items use muted white text.
- Descriptions are rendered in faint style beside each title.
- Footer shows key hints in faint text.

## 9. Screens

The TUI has six screen types:

| Screen           | Purpose                                              |
| ---------------- | ---------------------------------------------------- |
| `screenMain`     | Top-level menu (System Info, plugins, Quit)          |
| `screenSub`      | Plugin sub-menu (actions, settings, Back)            |
| `screenDetail`   | Read-only result display (press any key to go back)  |
| `screenInput`    | Text input for editing a config value                |
| `screenConfirm`  | Y/N confirmation dialog for destructive actions      |
| `screenProgress` | Job progress indicator with spinner and polling      |

### Progress Screen (`screenProgress`)

When an update action is confirmed (e.g., "Run Full Update"), the TUI
transitions to a progress screen instead of showing a raw API response.
Other confirmed POST actions (e.g., generic plugin endpoints) still show
a standard detail response.

1. The action calls `TriggerJob(jobID)` via `POST /api/v1/jobs/trigger`.
2. On success, a `jobAcceptedMsg` transitions to `screenProgress`.
3. A braille spinner (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) animates at 1 s intervals.
4. Every 2 s the TUI polls `GET /api/v1/jobs/{id}/runs/latest` for status.
5. **Completed** — screen transitions to `screenDetail` with a success
   summary including duration.
6. **Failed** — screen transitions to `screenDetail` with error details.
7. **Esc / q** — dismisses the progress screen; the job continues running
   on the server in the background.
8. Transient poll errors (network blips) are silently retried — the next
   tick retries automatically. After `maxPollErrors` consecutive failures
   (currently 5), the progress screen transitions to an error detail view.
9. Stale poll results from a previously dismissed job are discarded using
   both job ID and a per-progress-screen session counter (`progressSession`),
   so that responses from an earlier session are ignored even if the same
   job ID is re-triggered. Tick messages also carry the session counter to
   prevent duplicate tick loops when sessions overlap.

```text
  ⠹ Full Update

  Elapsed: 12s

  Esc/q: cancel
```

## 10. Theme System

The TUI ships four built-in themes and supports custom YAML themes.

### Built-in Themes

| Name | Description |
| --- | --- |
| `default` | Bright blue/cyan — matches the original hardcoded colours |
| `high-contrast` | Maximum readability — white-on-black, bold badges |
| `nord` | Muted arctic tones from the Nord colour palette |
| `solarized-dark` | Ethan Schoonover's Solarized Dark palette |

### YAML Format

Themes are defined in YAML with two optional sections — `colors` and `glyphs`.
All fields are optional; omitted fields inherit from `DefaultTheme()`.

```yaml
colors:
  header:      {fg: "111", bold: true}
  footer:      {fg: "60", faint: true}
  selected:    {fg: "152", bold: true}
  normal:      {fg: "252"}
  description: {fg: "60", faint: true}
  cursor:      {fg: "152"}
  connected:   {fg: "108"}
  standalone:  {fg: "222"}
  confirm_yes: {fg: "108", bold: true}
  confirm_no:  {fg: "174", bold: true}
  status_bar:  {fg: "60", faint: true}
  spinner:     {fg: "111"}

glyphs:
  cursor: "▸ "
  separator: "─"
  separator_width: 40
  connected_badge: "● connected"
  standalone_badge: "● standalone"
```

Colour values are strings — ANSI 256 numbers (`"14"`) or hex (`"#ff5733"`).

### API

- `ThemeFromYAML(data []byte) (Theme, error)` — parse YAML into a Theme.
- `BuiltinTheme(name string) (Theme, bool)` — look up a built-in by name (case-insensitive).
- `BuiltinThemeNames() []string` — sorted list of built-in names.
- `Model.SetTheme(t Theme)` — replace the active theme before `Run()`.

The core binary reads the user's config, resolves the theme (built-in name or
custom YAML file path), and calls `SetTheme()` before launching the TUI.

## 11. Future Extensions

- Log viewer within the TUI.
