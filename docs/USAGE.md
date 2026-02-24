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

Once integrated into the `cm` binary (Phase 2), the TUI will display a main
menu with entries for:

- **System Info** — (planned) view node hostname, OS version, uptime, and
  resource usage.
- **Plugins** — (planned) one submenu per registered plugin (e.g., Update
  Management, Network Config). Currently a static placeholder.
- **Quit** — exit the TUI and stop the binary.

## 4. Plugin Submenus

Plugin-specific submenus are a planned feature. Each plugin will provide its own
set of menu actions, and selecting a plugin from the main menu will open its
submenu once this feature is implemented.

## 5. Running

The TUI is not a standalone binary. It will run as part of the Config Manager
core once integration is wired (Phase 2). Tests are planned for a future
iteration (tracked as `tui-tests`).
