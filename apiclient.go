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
	client  *http.Client
}

// NewAPIClient returns a client pointing at the given base URL.
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Second},
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

// UpdateStatus represents the response from /api/v1/plugins/update/status.
type UpdateStatus struct {
	LastRun   string `json:"last_run"`
	Status    string `json:"status"`
	Pending   int    `json:"pending"`
	Security  int    `json:"security"`
	AutoTimer string `json:"auto_timer"`
}

// UpdateRunResult represents the response from POST /api/v1/plugins/update/run.
type UpdateRunResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	JobID   string `json:"job_id"`
}

// UpdateLogEntry represents a single log entry from /api/v1/plugins/update/logs.
type UpdateLogEntry struct {
	Timestamp string `json:"timestamp"`
	Action    string `json:"action"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// NetworkInterface represents a network interface from /api/v1/plugins/network/interfaces.
type NetworkInterface struct {
	Name    string `json:"name"`
	State   string `json:"state"`
	Type    string `json:"type"`
	Address string `json:"address"`
	Gateway string `json:"gateway"`
	Method  string `json:"method"`
}

// DNSConfig represents DNS settings from /api/v1/plugins/network/dns.
type DNSConfig struct {
	Servers []string `json:"servers"`
	Search  []string `json:"search"`
}

// NetworkStatus represents /api/v1/plugins/network/status.
type NetworkStatus struct {
	Hostname     string `json:"hostname"`
	DefaultGW    string `json:"default_gateway"`
	DNSServers   string `json:"dns_servers"`
	Connectivity string `json:"connectivity"`
}

// GetNode fetches system information.
func (c *APIClient) GetNode() (*NodeInfo, error) {
	var info NodeInfo
	if err := c.getJSON("/api/v1/node", &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetUpdateStatus fetches the update plugin status.
func (c *APIClient) GetUpdateStatus() (*UpdateStatus, error) {
	var s UpdateStatus
	if err := c.getJSON("/api/v1/plugins/update/status", &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// RunUpdate triggers an update run.
func (c *APIClient) RunUpdate(mode string) (*UpdateRunResult, error) {
	payload, err := json.Marshal(struct {
		Mode string `json:"mode"`
	}{Mode: mode})
	if err != nil {
		return nil, err
	}
	var r UpdateRunResult
	if err := c.postJSON("/api/v1/plugins/update/run", string(payload), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetUpdateLogs fetches recent update logs.
func (c *APIClient) GetUpdateLogs() ([]UpdateLogEntry, error) {
	var logs []UpdateLogEntry
	if err := c.getJSON("/api/v1/plugins/update/logs", &logs); err != nil {
		return nil, err
	}
	return logs, nil
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
	resp, err := c.client.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort error body
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *APIClient) postJSON(path, body string, out interface{}) error {
	resp, err := c.client.Post(
		c.baseURL+path,
		"application/json",
		strings.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort error body
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(b))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}
