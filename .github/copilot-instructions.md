# Copilot Instructions

## Project Overview

config-manager-tui is the terminal user interface for the Config Manager
system. It provides a raspi-config style interactive menu built with Bubble Tea
(Charmbracelet). This package is compiled into the core binary at build time
and serves as the primary user-facing interface.

Target platforms: Raspbian Bookworm (ARM64), Debian Bullseye slim.

## Architecture

- **tui.go** — main Bubble Tea model: `Model` struct with
  `New(plugins []PluginInfo)`, `NewWithAPI(plugins, apiBaseURL)`,
  `Init()`, `Update()`, `View()`
- **menu.go** — `PluginInfo` struct, `MenuItem` struct, action builders,
  and `MainMenu(plugins []PluginInfo)` backward-compat builder
- **apiclient.go** — HTTP client for the local CM REST API
- **views.go** — view rendering functions: header, footer, main menu, plugin
  views

## Integration

The core binary imports this package and runs it:

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

## Conventions

- Use Bubble Tea's Elm architecture: `Init()`, `Update()`, `View()`
- Use Lip Gloss for all styling — no raw ANSI escape codes
- Exported `New(plugins []PluginInfo)` and `NewWithAPI(plugins, apiBaseURL)`
  are the public constructors. `New` defaults to `localhost:7788`.
- Menu items are built dynamically from `[]PluginInfo` passed by the core
- Use `log/slog` for all structured logging
- Specs live in `specs/`, user docs in `docs/`
- Filenames use UPPERCASE (e.g., `SPEC.md`, `USAGE.md`); use UPPERCASE-KEBAB-CASE for multi-word names (e.g., `PLUGIN-INTERFACE.md`)

## Specifications

- [specs/SPEC.md](../specs/SPEC.md) — TUI specification and menu structure
- [docs/USAGE.md](../docs/USAGE.md) — navigation and key bindings

## Validation

- All Go code must pass `golangci-lint run`
- All tests must pass: `go test ./...`
- CI runs markdownlint + lint + test via `.github/workflows/ci.yml`
- Never push directly to main — always use feature branches and PRs
