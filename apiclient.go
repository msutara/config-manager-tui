package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
)

// APIClient calls the local CM REST API.
type APIClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewAPIClient returns a client pointing at the given base URL.
func NewAPIClient(baseURL string) *APIClient {
	return NewAPIClientWithToken(baseURL, "")
}

// NewAPIClientWithToken returns a client that sends a Bearer token with
// every request (except unauthenticated endpoints handled server-side).
func NewAPIClientWithToken(baseURL, token string) *APIClient {
	return &APIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// --- Generic types (plugin registry) ---

// PluginRegistryEntry describes a plugin as returned by GET /api/v1/plugins.
type PluginRegistryEntry struct {
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Description string           `json:"description"`
	RoutePrefix string           `json:"route_prefix"`
	Endpoints   []PluginEndpoint `json:"endpoints"`
}

// PluginEndpoint describes a single HTTP endpoint exposed by a plugin.
type PluginEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

// --- Core types ---

// NodeInfo represents the response from /api/v1/node.
type NodeInfo struct {
	Arch          string `json:"arch"`
	Hostname      string `json:"hostname"`
	Kernel        string `json:"kernel"`
	OS            string `json:"os"`
	UptimeSeconds int    `json:"uptime_seconds"`
}

// --- Update plugin types ---

// PendingUpdate represents a single pending update from /api/v1/plugins/update/status.
type PendingUpdate struct {
	Package        string `json:"package"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
	Security       bool   `json:"security"`
}

// UpdateRunResult represents the response from POST /api/v1/plugins/update/run.
type UpdateRunResult struct {
	Status string `json:"status"`
	Type   string `json:"type"`
}

// RunStatus represents the response from /api/v1/plugins/update/logs.
type RunStatus struct {
	Type      string `json:"type"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at,omitempty"`
	Duration  string `json:"duration"`
	Packages  int    `json:"packages"`
	Log       string `json:"log"`
}

// UpdateConfig models the subset of /api/v1/plugins/update/config used by the TUI.
type UpdateConfig struct {
	SecurityAvailable *bool `json:"security_available"`
}

// --- Network plugin types ---

// NetworkInterface represents a network interface from /api/v1/plugins/network/interfaces.
type NetworkInterface struct {
	Name  string `json:"name"`
	MAC   string `json:"mac"`
	IP    string `json:"ip"`
	State string `json:"state"`
}

// DNSConfig represents DNS settings from /api/v1/plugins/network/dns.
type DNSConfig struct {
	Nameservers []string `json:"nameservers"`
	Search      []string `json:"search"`
}

// NetworkStatus represents /api/v1/plugins/network/status.
type NetworkStatus struct {
	DefaultGateway    string `json:"default_gateway"`
	DNSReachable      bool   `json:"dns_reachable"`
	InternetReachable bool   `json:"internet_reachable"`
}

// --- Generic plugin methods ---

// truncateBody sanitizes and truncates a response body for inclusion in error
// messages, preventing terminal injection and oversized output.
func truncateBody(b []byte) string {
	const maxLen = 200
	s := string(b)
	runes := make([]rune, 0, maxLen)
	for _, r := range s {
		if unicode.IsControl(r) {
			continue // strip all control characters (ASCII C0 + Unicode C1)
		}
		runes = append(runes, r)
		if len(runes) == maxLen {
			return string(runes) + "..."
		}
	}
	return string(runes)
}

// GetPlugins fetches the plugin registry.
func (c *APIClient) GetPlugins() ([]PluginRegistryEntry, error) {
	var plugins []PluginRegistryEntry
	if err := c.getJSON("/api/v1/plugins", &plugins); err != nil {
		return nil, err
	}
	return plugins, nil
}

// GetRaw fetches an arbitrary endpoint and returns its raw body string.
func (c *APIClient) GetRaw(apiPath string) (string, error) {
	if !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}
	req, err := http.NewRequest(http.MethodGet, c.baseURL+apiPath, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", fmt.Errorf("read body %s: %w", apiPath, readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d: %s", apiPath, resp.StatusCode, truncateBody(body))
	}
	return string(body), nil
}

// PostRaw sends a POST to an arbitrary endpoint and returns the status message.
func (c *APIClient) PostRaw(apiPath string) (string, error) {
	if !strings.HasPrefix(apiPath, "/") {
		apiPath = "/" + apiPath
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+apiPath, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", fmt.Errorf("read body %s: %w", apiPath, readErr)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("POST %s: status %d: %s", apiPath, resp.StatusCode, truncateBody(body))
	}
	return string(body), nil
}

// --- Core methods ---

// GetNode fetches system information.
func (c *APIClient) GetNode() (*NodeInfo, error) {
	var info NodeInfo
	if err := c.getJSON("/api/v1/node", &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// --- Update plugin methods ---

// GetUpdateStatus fetches pending updates.
func (c *APIClient) GetUpdateStatus() ([]PendingUpdate, error) {
	var updates []PendingUpdate
	if err := c.getJSON("/api/v1/plugins/update/status", &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// RunUpdate triggers an update run.
func (c *APIClient) RunUpdate(mode string) (*UpdateRunResult, error) {
	payload, err := json.Marshal(struct {
		Type string `json:"type"`
	}{Type: mode})
	if err != nil {
		return nil, err
	}
	var r UpdateRunResult
	if err := c.postJSON("/api/v1/plugins/update/run", string(payload), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetUpdateConfig fetches the update plugin configuration.
func (c *APIClient) GetUpdateConfig() (*UpdateConfig, error) {
	var cfg UpdateConfig
	if err := c.getJSON("/api/v1/plugins/update/config", &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetUpdateLogs fetches the last update run status.
func (c *APIClient) GetUpdateLogs() (*RunStatus, error) {
	var rs RunStatus
	if err := c.getJSON("/api/v1/plugins/update/logs", &rs); err != nil {
		return nil, err
	}
	return &rs, nil
}

// --- Network plugin methods ---

// GetNetworkInterfaces lists all network interfaces.
func (c *APIClient) GetNetworkInterfaces() ([]NetworkInterface, error) {
	var ifaces []NetworkInterface
	if err := c.getJSON("/api/v1/plugins/network/interfaces", &ifaces); err != nil {
		return nil, err
	}
	return ifaces, nil
}

// GetDNS fetches DNS configuration.
func (c *APIClient) GetDNS() (*DNSConfig, error) {
	var dns DNSConfig
	if err := c.getJSON("/api/v1/plugins/network/dns", &dns); err != nil {
		return nil, err
	}
	return &dns, nil
}

// GetNetworkStatus fetches overall network status.
func (c *APIClient) GetNetworkStatus() (*NetworkStatus, error) {
	var s NetworkStatus
	if err := c.getJSON("/api/v1/plugins/network/status", &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *APIClient) getJSON(path string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort error body
		return fmt.Errorf("GET %s: status %d: %s", path, resp.StatusCode, truncateBody(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func (c *APIClient) postJSON(path, body string, out interface{}) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort error body
		return fmt.Errorf("POST %s: status %d: %s", path, resp.StatusCode, truncateBody(b))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
