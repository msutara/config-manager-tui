# Config Manager TUI — Usage

## 1. Overview

The Config Manager TUI provides a raspi-config style terminal menu system for
managing headless Debian-based nodes. It runs as part of the core binary and
serves as the primary interactive interface.

## 2. Navigation

Use the following keys to navigate:

| Key            | Action                              |
| -------------- | ----------------------------------- |
| ↑ / k          | Move cursor up                      |
| ↓ / j          | Move cursor down                    |
| Enter          | Select the highlighted item         |
| esc/q/backspace | Go back (in sub-menus/detail views) |
| q / ctrl+c     | Quit the TUI (from main menu)       |

## 3. Main Menu

The main menu is built dynamically from the plugins registered in the core
binary. It always includes:

- **System Info** — view node hostname, OS version, kernel, architecture, and
  uptime.
- **One entry per plugin** — e.g., "Update Manager", "Network Manager".
  These appear in the order provided by the core.
- **Quit** — exit the TUI and stop the binary.

## 4. Plugin Submenus

Selecting a plugin from the main menu opens its submenu. Each plugin provides
its own set of menu actions:

- **Update Manager** — Check Status, Full Update, Security Update, View Logs
- **Network Manager** — List Interfaces, Network Status, DNS Settings

Each submenu includes a "Back" item to return to the main menu. You can also
press esc, q, or backspace to go back.

## 5. Running

The TUI is not a standalone binary. It runs as part of the Config Manager
core binary. Run the test suite with:

```bash
go test -cover ./...
```
