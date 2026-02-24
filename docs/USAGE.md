# Config Manager TUI — Usage

## 1. Overview

The Config Manager TUI provides a raspi-config style terminal menu system for
managing headless Debian-based nodes. It is the primary interactive interface
when running the `cm` binary.

## 2. Navigation

Use the following keys to navigate:

| Key        | Action                       |
|------------|------------------------------|
| ↑ / k      | Move cursor up               |
| ↓ / j      | Move cursor down             |
| Enter      | Select the highlighted item  |
| q / ctrl+c | Quit the TUI                 |

## 3. Main Menu

When the `cm` binary starts, the TUI displays a main menu with entries for:

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

The TUI is not a standalone binary. It runs as part of the Config Manager core.
Build and run from the
[config-manager-core](https://github.com/msutara/config-manager-core) repo:

```bash
cd config-manager-core
go build -o cm ./cmd/cm
./cm
```

Once TUI integration is wired (Phase 2), it will start automatically as the
main interface.
