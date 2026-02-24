# Copilot Instructions

## Project Overview

config-manager-tui is the terminal user interface for the Config Manager
system. It provides a raspi-config style interactive menu built with Bubble Tea
(Charmbracelet). This package will be compiled into the core binary at build
time and serve as the primary user-facing interface (Phase 2).

Target platforms: Raspbian Bookworm (ARM64), Debian Bullseye slim.

## Architecture

- **tui.go** — main Bubble Tea model: `Model` struct with `New()`, `Init()`,
  `Update()`, `View()`
- **menu.go** — menu data structures (`MenuItem`) and menu builder functions
- **views.go** — view rendering functions: header, footer, main menu, plugin
  views

## Integration

The core binary will import this package and run it (Phase 2):

```go
import (
  tea "github.com/charmbracelet/bubbletea"
  tui "github.com/msutara/config-manager-tui"
)

model := tui.New()
p := tea.NewProgram(model)
p.Run()
```

## Conventions

- Use Bubble Tea's Elm architecture: `Init()`, `Update()`, `View()`
- Use Lip Gloss for all styling — no raw ANSI escape codes
- Exported `New()` function is the only public constructor
- Menu items are built dynamically from the core plugin registry (planned)
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
