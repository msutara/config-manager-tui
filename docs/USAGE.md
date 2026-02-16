# Config Manager TUI — Usage

## Navigation

The TUI presents a raspi-config style menu system. Use the following keys
to navigate:

| Key       | Action                        |
|-----------|-------------------------------|
| ↑ / k     | Move cursor up                |
| ↓ / j     | Move cursor down              |
| Enter     | Select the highlighted item   |
| q / ctrl+c | Quit the TUI                  |

## Main Menu

When the `cm` binary starts, the TUI displays a main menu with entries for:

- **System Info** — view node hostname, OS version, uptime, and resource usage.
- **Plugins** — (planned) plugin management entry intended to expose one submenu
  per registered plugin (e.g., Update Management, Network Config). Currently a
  placeholder that does not perform any action when selected.
- **Quit** — exit the TUI and stop the binary.

## Plugin Submenus

Plugin-specific submenus are a planned feature. Each plugin will provide its own
set of menu actions, and selecting a plugin from the main menu will open its
submenu once this feature is implemented.

## Running

The TUI is not a standalone binary. It runs as part of the Config Manager core:

```bash
# Build and run
go build -o cm ./cmd/cm
./cm
```

The TUI starts automatically as the main interface.
