package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// PluginInfo describes a registered plugin for menu rendering. The core binary
// populates this from its plugin registry — the TUI has no direct dependency
// on the core plugin package.
type PluginInfo struct {
	Name        string
	Description string
}

// MenuItem represents a single entry in a TUI menu.
type MenuItem struct {
	Title       string
	Description string
	Action      func() tea.Cmd
	IsQuit      bool // when true, selecting this item exits the TUI
}

// buildMainMenu returns the top-level menu items with live actions.
func (m *Model) buildMainMenu() []MenuItem {
	api := m.api // capture API client, not *Model
	items := []MenuItem{
		{
			Title:       "System Info",
			Description: "View system information and status",
			Action:      actionSystemInfo(api),
		},
	}

	for _, p := range m.plugins {
		p := p // capture loop variable
		switch p.Name {
		case "update":
			items = append(items, MenuItem{
				Title:       "Update Manager",
				Description: p.Description,
				Action:      actionUpdateMenu(api),
			})
		case "network":
			items = append(items, MenuItem{
				Title:       "Network Manager",
				Description: p.Description,
				Action:      actionNetworkMenu(api),
			})
		default:
			items = append(items, MenuItem{
				Title:       p.Name,
				Description: p.Description,
			})
		}
	}

	items = append(items, MenuItem{
		Title:       "Quit",
		Description: "Exit Config Manager",
		Action:      func() tea.Cmd { return tea.Quit },
		IsQuit:      true,
	})

	return items
}

// --- System Info ---

func actionSystemInfo(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			info, err := api.GetNode()
			if err != nil {
				return apiResultMsg{err: err}
			}
			uptime := formatUptime(info.UptimeSeconds)
			detail := fmt.Sprintf(
				"Hostname:  %s\nOS:        %s\nKernel:    %s\nArch:      %s\nUptime:    %s",
				info.Hostname, info.OS, info.Kernel, info.Arch, uptime,
			)
			return apiResultMsg{detail: detail}
		}
	}
}

// --- Update Plugin Sub-Menu ---

func actionUpdateMenu(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			return subMenuMsg{
				title: "Update Manager",
				items: []MenuItem{
					{Title: "Check Status", Description: "View update status", Action: actionUpdateStatus(api)},
					{Title: "Full Update", Description: "Run full system update", Action: actionUpdateRunFull(api)},
					{Title: "Security Update", Description: "Apply security patches only", Action: actionUpdateRunSecurity(api)},
					{Title: "View Logs", Description: "Recent update activity", Action: actionUpdateLogs(api)},
					{Title: "Back", Description: "Return to main menu", Action: func() tea.Cmd {
						return func() tea.Msg { return subMenuMsg{} }
					}},
				},
			}
		}
	}
}

func actionUpdateStatus(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			s, err := api.GetUpdateStatus()
			if err != nil {
				return apiResultMsg{err: err}
			}
			detail := fmt.Sprintf(
				"Status:     %s\nLast Run:   %s\nPending:    %d packages\nSecurity:   %d patches\nAuto Timer: %s",
				s.Status, s.LastRun, s.Pending, s.Security, s.AutoTimer,
			)
			return apiResultMsg{detail: detail}
		}
	}
}

func actionUpdateRunFull(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			r, err := api.RunUpdate("full")
			if err != nil {
				return apiResultMsg{err: err}
			}
			detail := fmt.Sprintf("Status:  %s\nMessage: %s\nJob ID:  %s", r.Status, r.Message, r.JobID)
			return apiResultMsg{detail: detail}
		}
	}
}

func actionUpdateRunSecurity(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			r, err := api.RunUpdate("security")
			if err != nil {
				return apiResultMsg{err: err}
			}
			detail := fmt.Sprintf("Status:  %s\nMessage: %s\nJob ID:  %s", r.Status, r.Message, r.JobID)
			return apiResultMsg{detail: detail}
		}
	}
}

func actionUpdateLogs(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			logs, err := api.GetUpdateLogs()
			if err != nil {
				return apiResultMsg{err: err}
			}
			if len(logs) == 0 {
				return apiResultMsg{detail: "No update logs available."}
			}
			var b strings.Builder
			for _, l := range logs {
				fmt.Fprintf(&b, "[%s] %s — %s: %s\n", l.Timestamp, l.Action, l.Status, l.Message) //nolint:errcheck // writes to strings.Builder
			}
			return apiResultMsg{detail: b.String()}
		}
	}
}

// --- Network Plugin Sub-Menu ---

func actionNetworkMenu(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			return subMenuMsg{
				title: "Network Manager",
				items: []MenuItem{
					{Title: "List Interfaces", Description: "Show network interfaces", Action: actionNetworkInterfaces(api)},
					{Title: "Network Status", Description: "Overall connectivity status", Action: actionNetworkStatus(api)},
					{Title: "DNS Settings", Description: "View DNS configuration", Action: actionNetworkDNS(api)},
					{Title: "Back", Description: "Return to main menu", Action: func() tea.Cmd {
						return func() tea.Msg { return subMenuMsg{} }
					}},
				},
			}
		}
	}
}

func actionNetworkInterfaces(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			ifaces, err := api.GetNetworkInterfaces()
			if err != nil {
				return apiResultMsg{err: err}
			}
			if len(ifaces) == 0 {
				return apiResultMsg{detail: "No network interfaces found."}
			}
			var b strings.Builder
			for _, iface := range ifaces {
				fmt.Fprintf(&b, "%-12s  %-6s  %-8s  %-16s  gw: %s  (%s)\n",
					iface.Name, iface.State, iface.Type, iface.Address, iface.Gateway, iface.Method) //nolint:errcheck // writes to strings.Builder
			}
			return apiResultMsg{detail: b.String()}
		}
	}
}

func actionNetworkStatus(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			s, err := api.GetNetworkStatus()
			if err != nil {
				return apiResultMsg{err: err}
			}
			detail := fmt.Sprintf(
				"Hostname:       %s\nDefault GW:     %s\nDNS Servers:    %s\nConnectivity:   %s",
				s.Hostname, s.DefaultGW, s.DNSServers, s.Connectivity,
			)
			return apiResultMsg{detail: detail}
		}
	}
}

func actionNetworkDNS(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			dns, err := api.GetDNS()
			if err != nil {
				return apiResultMsg{err: err}
			}
			servers := "none"
			if len(dns.Servers) > 0 {
				servers = strings.Join(dns.Servers, ", ")
			}
			search := "none"
			if len(dns.Search) > 0 {
				search = strings.Join(dns.Search, ", ")
			}
			detail := fmt.Sprintf("DNS Servers:  %s\nSearch:       %s", servers, search)
			return apiResultMsg{detail: detail}
		}
	}
}

// --- Helpers ---

func formatUptime(seconds int) string {
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// MainMenu is kept for backward compatibility but delegates to buildMainMenu.
// Callers that don't need API access can still use it — actions will be nil.
func MainMenu(plugins []PluginInfo) []MenuItem {
	items := []MenuItem{
		{
			Title:       "System Info",
			Description: "View system information and status",
		},
	}

	for _, p := range plugins {
		items = append(items, MenuItem{
			Title:       p.Name,
			Description: p.Description,
		})
	}

	items = append(items, MenuItem{
		Title:       "Quit",
		Description: "Exit Config Manager",
		Action:      func() tea.Cmd { return tea.Quit },
		IsQuit:      true,
	})

	return items
}
