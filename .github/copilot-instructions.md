# Copilot Instructions

## Project Overview

Config Manager TUI is the terminal user interface for the Config Manager system. It provides a raspi-config style interactive menu built with Bubble Tea (Charmbracelet). This package is compiled into the core binary at build time and serves as the primary user-facing interface.

## Architecture

- **`tui.go`** — Main Bubble Tea model: `Model` struct with `New()`, `Init()`, `Update()`, `View()`
- **`menu.go`** — Menu data structures (`MenuItem`) and menu builder functions
- **`views.go`** — View rendering functions: header, footer, main menu, plugin views

The core binary imports this package and runs it:

```go
import tui "github.com/msutara/config-manager-tui"

model := tui.New()
p := tea.NewProgram(model)
p.Run()
```

## Conventions

- Use Bubble Tea's Elm architecture: `Init()`, `Update()`, `View()`
- Use Lip Gloss for all styling — no raw ANSI escape codes
- Exported `New()` function is the only public constructor
- Menu items are built dynamically from the core plugin registry
- All Go code must pass `golangci-lint run`
- Agent-readable specifications live in `specs/` (UPPERCASE filenames, e.g. `SPEC.md`)
- User-facing documentation lives in `docs/` (UPPERCASE filenames, e.g. `USAGE.md`)
- Never push directly to main — always use feature branches and PRs

## Validation

- All Go code must pass `golangci-lint run`
- All tests must pass: `go test ./...`
- CI runs lint + test via `.github/workflows/ci.yml`
- Never push directly to main — always use feature branches and PRs
