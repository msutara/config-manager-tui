# Config Manager TUI

Terminal user interface for Config Manager — a raspi-config style interactive
menu built with Bubble Tea (Charmbracelet). Compiled into the core binary at
build time.

## Features

- **Raspi-config style menus** — arrow-key navigation, enter to select, q to quit
- **Dynamic plugin menus** — discovers plugins from the core plugin registry
- **Styled with Lip Gloss** — clean, consistent terminal rendering
- **Elm architecture** — Bubble Tea model with Init/Update/View
- **Single integration point** — export `New()` for the core binary to call

## Usage

This package is not a standalone binary. The core binary imports it:

```go
import tui "github.com/msutara/config-manager-tui"

model := tui.New()
p := tea.NewProgram(model)
p.Run()
```

## Documentation

- [Usage Guide](docs/USAGE.md) — TUI navigation and key bindings

## Specifications

- [SPEC.md](specs/SPEC.md) — TUI specification

## License

See [LICENSE](LICENSE) for details.