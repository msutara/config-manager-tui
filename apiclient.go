package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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

// NodeInfo represents the response from /api/v1/node.
type NodeInfo struct {
	Arch          string `json:"arch"`
	Hostname      string `json:"hostname"`
	Kernel        string `json:"kernel"`
	OS            string `json:"os"`
	UptimeSeconds int    `json:"uptime_seconds"`
}

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

// GetNode fetches system information.
func (c *APIClient) GetNode() (*NodeInfo, error) {
	var info NodeInfo
	if err := c.getJSON("/api/v1/node", &info); err != nil {
		return nil, err
	}
	return &info, nil
}

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

// UpdateConfig models the subset of /api/v1/plugins/update/config used by the TUI.
type UpdateConfig struct {
	SecurityAvailable bool `json:"security_available"`
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
		return fmt.Errorf("GET %s: status %d: %s", path, resp.StatusCode, string(body))
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
		return fmt.Errorf("POST %s: status %d: %s", path, resp.StatusCode, string(b))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
