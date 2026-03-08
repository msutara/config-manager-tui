# Config Manager TUI — Usage

## 1. Overview

The Config Manager TUI provides a raspi-config style terminal menu system for
managing headless Debian-based nodes. It runs as part of the core binary and
serves as the primary interactive interface.

## 2. Navigation

Use the following keys to navigate:

| Key             | Action                                             |
| --------------- | -------------------------------------------------- |
| ↑ / k           | Move cursor up                                     |
| ↓ / j           | Move cursor down                                   |
| Enter           | Select the highlighted item                        |
| esc/q/backspace | Go back (in sub-menus); any key goes back (detail) |
| q               | Quit the TUI (from main menu)                      |
| ctrl+c          | Quit the TUI (from any screen)                     |

### Input Screen

When editing a text value (e.g. cron schedule), an input screen appears:

| Key       | Action                    |
| --------- | ------------------------- |
| Type text | Appends to the input      |
| Backspace | Delete last character     |
| Enter     | Save the value            |
| Esc       | Cancel and return to menu |

### Confirmation Dialog

Destructive actions (Full Update, Security Update, generic POST endpoints)
show a confirmation dialog before executing. The dialog displays a title and
an explanation of the action.

| Key   | Action                        |
| ----- | ----------------------------- |
| y / Y | Confirm and execute the action |
| n / N | Cancel and return to menu      |
| Esc   | Cancel and return to menu      |
| q     | Cancel and return to menu      |

> **Note:** The Enter key is intentionally excluded from the confirmation
> dialog to prevent accidental double-tap execution. You must press `y` to
> confirm.

## 3. Status Bar

The footer displays a status bar with the node hostname and uptime (fetched
once on startup from `GET /api/v1/node`). If the API is unreachable the
status bar is omitted. The status bar and connection badge appear on the main
menu, sub-menu, and input screens.

## 4. Main Menu

The main menu is built dynamically from the plugins registered in the core
binary. It always includes:

- **System Info** — view node hostname, OS version, kernel, architecture, and
  uptime.
- **One entry per plugin** — e.g., "Update Manager", "Network Manager".
  These appear in the order provided by the core.
- **Quit** — exit the TUI and stop the binary.

## 5. Plugin Submenus

Selecting a plugin from the main menu opens its submenu. Each plugin provides
its own set of menu actions:

- **Update Manager** — Check Status, Full Update, Security Update (when
  available), View Logs, View Settings, Edit Schedule, Toggle Auto-Security,
  Change Security Source
- **Network Manager** — List Interfaces, Network Status, DNS Settings, Set
  Static IP, Set DNS Servers, Delete Static IP, Rollback Interface, Rollback DNS

Each submenu includes a "Back" item to return to the main menu. You can also
press esc, q, or backspace to go back.

### Nested Submenus

Some actions open a second-level submenu. For example, **Set Static IP**,
**Delete Static IP**, and **Rollback Interface** first show an interface picker
listing all available network interfaces. Selecting an interface proceeds to
the input or confirmation screen. Pressing Back from a nested submenu returns
to the parent plugin submenu (not the main menu).

### Network Write Operations

The Network Manager submenu includes write actions that modify network
configuration on the device:

| Action | Description |
| --- | --- |
| Set Static IP | Select an interface, then enter a CIDR address (e.g. `192.168.1.10/24`) |
| Set DNS Servers | Edit the comma-separated list of DNS nameservers |
| Delete Static IP | Select an interface to remove its static IP assignment |
| Rollback Interface | Select an interface to revert to its previous configuration |
| Rollback DNS | Revert DNS settings to the previous configuration |

All write operations display a confirmation dialog before executing. After a
successful write, the menu refreshes automatically to reflect the updated
state.

## 6. Troubleshooting

### "protected by write policy" Messages

If a network write operation displays a message like:

> interface 'eth0' is protected by write policy — check interface_policy config

this means the Config Manager API rejected the operation with a 403 Forbidden
status because the target interface (or DNS) is restricted by the
`interface_policy` configuration in the network plugin.

**To resolve:**

1. Open the Config Manager core configuration file.
2. Locate the `interface_policy` section under the network plugin settings.
3. Verify the target interface is listed under `writable` or remove it from the
   `read_only` list, depending on the policy model in use.
4. Restart the Config Manager core for changes to take effect.
5. Retry the operation in the TUI.

This behaviour is by design — it prevents accidental modification of
interfaces that are managed by other systems or marked as critical
infrastructure.

## 7. Running

The TUI is not a standalone binary. It runs as part of the Config Manager
core binary. Run the test suite with:

```bash
go test -cover ./...
```
