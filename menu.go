package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// PluginInfo describes a registered plugin for menu rendering. The core binary
// populates this from its plugin registry — the TUI has no direct dependency
// on the core plugin package.
type PluginInfo struct {
	Name        string
	Description string
	RoutePrefix string
	Endpoints   []PluginEndpoint
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
		switch p.Name {
		case "update":
			items = append(items, MenuItem{
				Title:       "Update Manager",
				Description: sanitizeText(p.Description),
				Action:      actionUpdateMenu(api),
			})
		case "network":
			items = append(items, MenuItem{
				Title:       "Network Manager",
				Description: sanitizeText(p.Description),
				Action:      actionNetworkMenu(api),
			})
		default:
			safeName := sanitizeText(p.Name)
			items = append(items, MenuItem{
				Title:       titleCase(safeName),
				Description: sanitizeText(p.Description),
				Action:      actionGenericPlugin(api, p),
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
				sanitizeText(info.Hostname), sanitizeText(info.OS),
				sanitizeText(info.Kernel), sanitizeText(info.Arch), uptime,
			)
			return apiResultMsg{detail: detail}
		}
	}
}

// --- Generic Plugin Sub-Menu ---

// titleCase converts a hyphen-separated name like "my-plugin" to "My Plugin".
func titleCase(s string) string {
	if s == "" {
		return s
	}
	parts := strings.Split(s, "-")
	var out []string
	for _, p := range parts {
		if p == "" {
			continue
		}
		r, size := utf8.DecodeRuneInString(p)
		out = append(out, string(unicode.ToUpper(r))+p[size:])
	}
	return strings.Join(out, " ")
}

// sanitizeText strips ASCII control characters (including ANSI escape sequences)
// from untrusted text before rendering in the terminal.
func sanitizeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if !unicode.IsControl(r) {
			_, _ = b.WriteRune(r) //nolint:errcheck // strings.Builder.WriteRune never fails
		}
	}
	return b.String()
}

// sanitizeBody strips control characters but preserves newlines and tabs
// for readable display of multi-line API response bodies.
func sanitizeBody(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			_, _ = b.WriteRune(r) //nolint:errcheck // strings.Builder.WriteRune never fails
		}
	}
	return b.String()
}

// cleanPluginPath builds a safe API path from a route prefix and endpoint path,
// rejecting path traversal (including percent-encoded sequences) and verifying
// the result stays under the expected prefix. routePrefix is trusted — it is
// set by the plugin registry (server-controlled, not user input).
func cleanPluginPath(routePrefix, epPath string) string {
	prefix := strings.TrimRight(routePrefix, "/")
	if prefix == "" {
		return ""
	}

	// Decode percent-encoding before validation to catch %2e%2e etc.
	decoded, err := url.PathUnescape(epPath)
	if err != nil {
		return ""
	}
	// Reject any remaining percent signs (double-encoding attempt).
	if strings.Contains(decoded, "%") {
		return ""
	}
	// Reject control characters (NUL, newlines, C1, etc.).
	for _, r := range decoded {
		if unicode.IsControl(r) {
			return ""
		}
	}
	if !strings.HasPrefix(decoded, "/") {
		decoded = "/" + decoded
	}

	// Canonicalize and verify no traversal escapes the prefix.
	full := path.Clean(prefix + decoded)
	if !strings.HasPrefix(full, prefix+"/") && full != prefix {
		return ""
	}
	return full
}

func actionGenericPlugin(api *APIClient, p PluginInfo) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			title := titleCase(sanitizeText(p.Name))
			var items []MenuItem

			for _, ep := range p.Endpoints {
				desc := sanitizeText(ep.Description)
				safePath := sanitizeText(ep.Path)
				switch strings.ToUpper(ep.Method) {
				case "GET":
					apiPath := cleanPluginPath(p.RoutePrefix, ep.Path)
					if apiPath == "" {
						continue
					}
					items = append(items, MenuItem{
						Title:       desc,
						Description: fmt.Sprintf("GET %s", safePath),
						Action:      actionGenericGet(api, apiPath),
					})
				case "POST":
					apiPath := cleanPluginPath(p.RoutePrefix, ep.Path)
					if apiPath == "" {
						continue
					}
					items = append(items, MenuItem{
						Title:       desc,
						Description: fmt.Sprintf("POST %s", safePath),
						Action:      actionGenericPost(api, apiPath, desc),
					})
				}
			}

			items = append(items, MenuItem{
				Title: "Back", Description: "Return to main menu",
				Action: func() tea.Cmd {
					return func() tea.Msg { return subMenuMsg{} }
				},
			})

			return subMenuMsg{title: title, items: items}
		}
	}
}

func actionGenericGet(api *APIClient, apiPath string) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			body, err := api.GetRaw(apiPath)
			if err != nil {
				return apiResultMsg{err: err}
			}
			// Try to pretty-print JSON; fall back to raw.
			var buf bytes.Buffer
			if json.Indent(&buf, []byte(body), "", "  ") == nil {
				body = buf.String()
			}
			return apiResultMsg{detail: sanitizeBody(body)}
		}
	}
}

func actionGenericPost(api *APIClient, apiPath, description string) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			_, err := api.PostRaw(apiPath)
			if err != nil {
				return apiResultMsg{err: err}
			}
			return apiResultMsg{detail: fmt.Sprintf("%s completed successfully.", description)}
		}
	}
}

// --- Update Plugin Sub-Menu ---

func actionUpdateMenu(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			items := []MenuItem{
				{Title: "Check Status", Description: "View update status", Action: actionUpdateStatus(api)},
				{Title: "Full Update", Description: "Run full system update", Action: actionUpdateRunFull(api)},
			}

			// Fetch config for both the security-available check and the
			// settings items.  Transient errors default to showing everything.
			var cfg *UpdateConfig
			showSecurity := true
			if c, err := api.GetUpdateConfig(); err == nil {
				cfg = c
				if c.SecurityAvailable != nil {
					showSecurity = *c.SecurityAvailable
				}
			}

			if showSecurity {
				items = append(items, MenuItem{
					Title: "Security Update", Description: "Apply security patches only",
					Action: actionUpdateRunSecurity(api),
				})
			}

			items = append(items,
				MenuItem{Title: "View Logs", Description: "Recent update activity", Action: actionUpdateLogs(api)},
			)

			// --- Settings section ---
			items = append(items, MenuItem{
				Title: "View Settings", Description: "Current update configuration",
				Action: actionUpdateViewSettings(api),
			})

			scheduleDisplay := "(unavailable)"
			autoSecDisplay := "(unknown)"
			secSourceDisplay := "(unknown)"
			if cfg != nil {
				if cfg.Schedule != "" {
					scheduleDisplay = cfg.Schedule
				}
				if cfg.AutoSecurity != nil {
					autoSecDisplay = boolOnOff(*cfg.AutoSecurity)
				}
				if cfg.SecuritySource != "" {
					secSourceDisplay = cfg.SecuritySource
				}
			}

			items = append(items,
				MenuItem{
					Title:       "Edit Schedule",
					Description: fmt.Sprintf("Current: %s", sanitizeText(scheduleDisplay)),
					Action:      actionEditSchedule(api),
				},
				MenuItem{
					Title:       "Toggle Auto-Security",
					Description: fmt.Sprintf("Currently: %s", sanitizeText(autoSecDisplay)),
					Action:      actionToggleAutoSecurity(api),
				},
				MenuItem{
					Title:       "Change Security Source",
					Description: fmt.Sprintf("Currently: %s", sanitizeText(secSourceDisplay)),
					Action:      actionCycleSecuritySource(api),
				},
			)

			items = append(items, MenuItem{Title: "Back", Description: "Return to main menu", Action: func() tea.Cmd {
				return func() tea.Msg { return subMenuMsg{} }
			}})

			return subMenuMsg{
				title: "Update Manager",
				items: items,
			}
		}
	}
}

func actionUpdateStatus(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			updates, err := api.GetUpdateStatus()
			if err != nil {
				return apiResultMsg{err: err}
			}
			if len(updates) == 0 {
				return apiResultMsg{detail: "No pending updates."}
			}
			var b strings.Builder
			secCount := 0
			for _, u := range updates {
				flag := " "
				if u.Security {
					flag = "!"
					secCount++
				}
				fmt.Fprintf(&b, "%s %-30s  %s → %s\n", flag, sanitizeText(u.Package), sanitizeText(u.CurrentVersion), sanitizeText(u.NewVersion)) //nolint:errcheck // writes to strings.Builder
			}
			header := fmt.Sprintf("Pending: %d packages (%d security)\n\n", len(updates), secCount)
			return apiResultMsg{detail: header + b.String()}
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
			detail := fmt.Sprintf("Status: %s\nType:   %s", sanitizeText(r.Status), sanitizeText(r.Type))
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
			detail := fmt.Sprintf("Status: %s\nType:   %s", sanitizeText(r.Status), sanitizeText(r.Type))
			return apiResultMsg{detail: detail}
		}
	}
}

func actionUpdateLogs(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			rs, err := api.GetUpdateLogs()
			if err != nil {
				return apiResultMsg{err: err}
			}
			if rs.Status == "none" {
				return apiResultMsg{detail: "No update runs recorded yet."}
			}
			detail := fmt.Sprintf(
				"Type:     %s\nStatus:   %s\nStarted:  %s\nDuration: %s\nPackages: %d",
				sanitizeText(rs.Type), sanitizeText(rs.Status),
				sanitizeText(rs.StartedAt), sanitizeText(rs.Duration), rs.Packages,
			)
			if rs.Log != "" {
				detail += "\n\nLog:\n" + sanitizeBody(rs.Log)
			}
			return apiResultMsg{detail: detail}
		}
	}
}

// --- Update Settings Actions ---

func actionUpdateViewSettings(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			ps, err := api.GetPluginSettings("update")
			if err != nil {
				return apiResultMsg{err: err}
			}
			var b strings.Builder
			b.WriteString("Update Plugin Settings\n\n") //nolint:errcheck // writes to strings.Builder
			keys := make([]string, 0, len(ps.Config))
			for k := range ps.Config {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(&b, "  %-20s %s\n", sanitizeText(k)+":", sanitizeValue(ps.Config[k])) //nolint:errcheck // writes to strings.Builder
			}
			return apiResultMsg{detail: b.String()}
		}
	}
}

func actionEditSchedule(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			// Fetch current schedule to avoid stale prefill.
			current := ""
			ps, err := api.GetPluginSettings("update")
			if err == nil {
				if v, ok := ps.Config["schedule"].(string); ok {
					current = v
				}
			}
			return editInputMsg{
				prompt:     "Enter new cron schedule (e.g. 0 3 * * *):",
				key:        "schedule",
				plugin:     "update",
				currentVal: current,
			}
		}
	}
}

func actionToggleAutoSecurity(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			// Fetch current value to avoid stale-closure toggling.
			ps, err := api.GetPluginSettings("update")
			if err != nil {
				return settingsResultMsg{err: err}
			}
			current := true
			if v, ok := ps.Config["auto_security"].(bool); ok {
				current = v
			}
			newVal := !current
			res, err := api.UpdatePluginSetting("update", "auto_security", newVal)
			if err != nil {
				return settingsResultMsg{err: err}
			}
			detail := formatSettingsResult("auto_security", sanitizeValue(newVal), res)
			return settingsResultMsg{detail: detail}
		}
	}
}

func actionCycleSecuritySource(api *APIClient) func() tea.Cmd {
	return func() tea.Cmd {
		return func() tea.Msg {
			// Fetch current value to avoid stale-closure cycling.
			ps, err := api.GetPluginSettings("update")
			if err != nil {
				return settingsResultMsg{err: err}
			}
			current := "available"
			if v, ok := ps.Config["security_source"].(string); ok {
				current = v
			}
			newVal := "always"
			if current == "always" {
				newVal = "available"
			}
			res, err := api.UpdatePluginSetting("update", "security_source", newVal)
			if err != nil {
				return settingsResultMsg{err: err}
			}
			detail := formatSettingsResult("security_source", newVal, res)
			return settingsResultMsg{detail: detail}
		}
	}
}

func boolOnOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
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
				fmt.Fprintf(&b, "%-12s  %-6s  %-17s  %s\n",
					sanitizeText(iface.Name), sanitizeText(iface.State),
					sanitizeText(iface.MAC), sanitizeText(iface.IP)) //nolint:errcheck // writes to strings.Builder
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
				"Default GW:        %s\nDNS Reachable:     %v\nInternet Reachable: %v",
				sanitizeText(s.DefaultGateway), s.DNSReachable, s.InternetReachable,
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
			if len(dns.Nameservers) > 0 {
				sanitized := make([]string, len(dns.Nameservers))
				for i, ns := range dns.Nameservers {
					sanitized[i] = sanitizeText(ns)
				}
				servers = strings.Join(sanitized, ", ")
			}
			search := "none"
			if len(dns.Search) > 0 {
				sanitized := make([]string, len(dns.Search))
				for i, s := range dns.Search {
					sanitized[i] = sanitizeText(s)
				}
				search = strings.Join(sanitized, ", ")
			}
			detail := fmt.Sprintf("Nameservers:  %s\nSearch:       %s", servers, search)
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

// MainMenu is a legacy static menu builder kept for backward compatibility.
// It returns menu items without plugin actions wired; only Quit has an action.
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
