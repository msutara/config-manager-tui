# Config Manager TUI — Usage

## 1. Overview

The Config Manager TUI provides a raspi-config style terminal menu system for
managing headless Debian-based nodes. Once wired into the core binary
(Phase 2), it will serve as the primary interactive interface.

## 2. Navigation

Use the following keys to navigate:

| Key        | Action                       |
|------------|------------------------------|
| ↑ / k      | Move cursor up               |
| ↓ / j      | Move cursor down             |
| Enter      | Select the highlighted item  |
| q / ctrl+c | Quit the TUI                 |

## 3. Main Menu

The main menu is built dynamically from the plugins registered in the core
binary. It always includes:

- **System Info** — (planned) view node hostname, OS version, uptime, and
  resource usage.
- **One entry per plugin** — e.g., "Update Management", "Network Config".
  These appear in the order provided by the core.
- **Quit** — exit the TUI and stop the binary.

## 4. Plugin Submenus

Plugin-specific submenus are a planned feature. Each plugin will provide its own
set of menu actions (e.g., "Check for Updates", "Show Interfaces"), and
selecting a plugin from the main menu will open its submenu once this is
implemented.

## 5. Running

The TUI is not a standalone binary. It will run as part of the Config Manager
core once integration is wired (Phase 2). Run the test suite with:

```bash
go test -cover ./...
```
