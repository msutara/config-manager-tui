package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
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

// UpdateConfig models the response from /api/v1/plugins/update/config.
type UpdateConfig struct {
	Schedule          string `json:"schedule"`
	AutoSecurity      *bool  `json:"auto_security"`
	SecuritySource    string `json:"security_source"`
	SecurityAvailable *bool  `json:"security_available"`
}

// PluginSettings models the response from GET /api/v1/plugins/{name}/settings.
type PluginSettings struct {
	Config map[string]any `json:"config"`
}

// PluginSettingsUpdateResult models the response from PUT /api/v1/plugins/{name}/settings.
type PluginSettingsUpdateResult struct {
	Config  map[string]any `json:"config"`
	Warning string         `json:"warning,omitempty"`
}

// --- Job run types (core scheduler) ---

// JobRun holds a job execution record returned by GET /api/v1/jobs/{id}/runs/latest
// and GET /api/v1/jobs/{id}/runs (paginated list).
type JobRun struct {
	JobID     string  `json:"job_id"`
	Status    string  `json:"status"` // "running", "completed", "failed"
	StartedAt string  `json:"started_at"`
	EndedAt   *string `json:"ended_at,omitempty"`
	Error     string  `json:"error,omitempty"`
	Duration  string  `json:"duration,omitempty"`
}

// TriggerJobResult represents the response from POST /api/v1/jobs/trigger.
type TriggerJobResult struct {
	Status string `json:"status"`
	JobID  string `json:"job_id"`
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

// StaticIPConfig is the request body for PUT /api/v1/plugins/network/interfaces/{name}.
type StaticIPConfig struct {
	Address string `json:"address"`
	Gateway string `json:"gateway,omitempty"`
	Netmask string `json:"netmask,omitempty"`
}

// DNSWriteConfig is the request body for PUT /api/v1/plugins/network/dns.
type DNSWriteConfig struct {
	Nameservers []string `json:"nameservers"`
	Search      []string `json:"search,omitempty"`
}

// NetworkWriteResult is the response from network write operations.
type NetworkWriteResult struct {
	Valid    bool           `json:"valid,omitempty"`
	Changes  []string       `json:"changes,omitempty"`
	Current  map[string]any `json:"current,omitempty"`
	Proposed map[string]any `json:"proposed,omitempty"`
	Message  string         `json:"message,omitempty"`
}

// --- Path validation ---

// validateAPIPath checks that an API path is safe before use in HTTP requests.
// It rejects empty paths, paths without a leading slash, and paths that contain
// directory traversal sequences (including percent-encoded variants like %2e%2e).
func validateAPIPath(p string) error {
	if p == "" {
		return fmt.Errorf("empty API path")
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("API path must start with /")
	}
	// Decode percent-encoding before traversal check so that sequences like
	// /%2e%2e/secret are caught.
	decoded, err := url.PathUnescape(p)
	if err != nil {
		return fmt.Errorf("invalid API path encoding: %w", err)
	}
	// Reject traversal attempts by checking for ".." path segments.
	for _, seg := range strings.Split(decoded, "/") {
		if seg == ".." {
			return fmt.Errorf("API path contains traversal")
		}
	}
	return nil
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

// APIError represents an error response from the API, preserving the HTTP
// status code alongside the human-readable message for programmatic handling.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string { return e.Message }

// isPolicyDenied reports whether err is a 403 Forbidden response from the API,
// indicating the operation was rejected by an interface write policy.
// Checks both status code and message to distinguish from other 403 causes
// (e.g., future auth denials).
func isPolicyDenied(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		return false
	}
	return strings.Contains(apiErr.Message, "not allowed for write operations")
}

// apiErrorEnvelope matches the standard error JSON from the core API:
//
//	{"error":{"code":"...","message":"..."}}
type apiErrorEnvelope struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// friendlyAPIError extracts a human-readable message from a raw JSON error
// body.  If the body is a well-formed error envelope with a message, only the
// message is returned.  Otherwise it falls back to the full body (truncated).
// The returned error is always an *APIError so callers can inspect StatusCode.
func friendlyAPIError(method, path string, status int, body []byte) error {
	var env apiErrorEnvelope
	if json.Unmarshal(body, &env) == nil && env.Error.Message != "" {
		return &APIError{StatusCode: status, Message: env.Error.Message}
	}
	return &APIError{
		StatusCode: status,
		Message:    fmt.Sprintf("%s %s: status %d: %s", method, path, status, truncateBody(body)),
	}
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
	if err := validateAPIPath(apiPath); err != nil {
		return "", err
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
		return "", friendlyAPIError("GET", apiPath, resp.StatusCode, body)
	}
	return string(body), nil
}

// PostRaw sends a POST to an arbitrary endpoint and returns the status message.
func (c *APIClient) PostRaw(apiPath string) (string, error) {
	if err := validateAPIPath(apiPath); err != nil {
		return "", err
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
		return "", friendlyAPIError("POST", apiPath, resp.StatusCode, body)
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

// validPluginName matches only safe plugin identifiers (lowercase alphanum + hyphens).
var validPluginName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// GetPluginSettings fetches a plugin's configurable settings via the core
// settings endpoint (GET /api/v1/plugins/{name}/settings).
func (c *APIClient) GetPluginSettings(name string) (*PluginSettings, error) {
	if !validPluginName.MatchString(name) {
		return nil, fmt.Errorf("invalid plugin name: %q", name)
	}
	var ps PluginSettings
	if err := c.getJSON("/api/v1/plugins/"+name+"/settings", &ps); err != nil {
		return nil, err
	}
	return &ps, nil
}

// UpdatePluginSetting changes a single setting key via the core settings
// endpoint (PUT /api/v1/plugins/{name}/settings).
func (c *APIClient) UpdatePluginSetting(name, key string, value any) (*PluginSettingsUpdateResult, error) {
	if !validPluginName.MatchString(name) {
		return nil, fmt.Errorf("invalid plugin name: %q", name)
	}
	payload, err := json.Marshal(struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}{Key: key, Value: value})
	if err != nil {
		return nil, err
	}
	var r PluginSettingsUpdateResult
	if err := c.putJSON("/api/v1/plugins/"+name+"/settings", string(payload), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// GetUpdateLogs fetches the last update run status.
func (c *APIClient) GetUpdateLogs() (*RunStatus, error) {
	var rs RunStatus
	if err := c.getJSON("/api/v1/plugins/update/logs", &rs); err != nil {
		return nil, err
	}
	return &rs, nil
}

// --- Job tracking methods ---

// validJobID matches dot-separated job identifiers (e.g. "update.full", "update-daily.full").
var validJobID = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`)

// TriggerJob fires a job by ID via the core scheduler endpoint.
func (c *APIClient) TriggerJob(jobID string) (*TriggerJobResult, error) {
	if !validPluginName.MatchString(jobID) && !validJobID.MatchString(jobID) {
		return nil, fmt.Errorf("invalid job ID: %q", jobID)
	}
	payload, err := json.Marshal(struct {
		JobID string `json:"job_id"`
	}{JobID: jobID})
	if err != nil {
		return nil, err
	}
	var r TriggerJobResult
	if err := c.postJSON("/api/v1/jobs/trigger", string(payload), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ListJobRuns fetches paginated job execution history (newest-first).
// Job IDs are either dot-separated (e.g. "update.full") matching validJobID,
// or single-word identifiers (e.g. "cleanup") matching validPluginName.
func (c *APIClient) ListJobRuns(jobID string, limit, offset int) ([]JobRun, error) {
	if !validPluginName.MatchString(jobID) && !validJobID.MatchString(jobID) {
		return nil, fmt.Errorf("invalid job ID: %q", jobID)
	}
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100, got %d", limit)
	}
	if offset < 0 {
		return nil, fmt.Errorf("offset must be non-negative, got %d", offset)
	}
	path := fmt.Sprintf("/api/v1/jobs/%s/runs?limit=%d&offset=%d", jobID, limit, offset)
	var runs []JobRun
	if err := c.getJSON(path, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

// GetJobRunLatest fetches the most recent execution record for a job.
// Job IDs are either dot-separated (e.g. "update.full") matching validJobID,
// or single-word identifiers (e.g. "cleanup") matching validPluginName.
func (c *APIClient) GetJobRunLatest(jobID string) (*JobRun, error) {
	if !validPluginName.MatchString(jobID) && !validJobID.MatchString(jobID) {
		return nil, fmt.Errorf("invalid job ID: %q", jobID)
	}
	var run JobRun
	if err := c.getJSON("/api/v1/jobs/"+jobID+"/runs/latest", &run); err != nil {
		return nil, err
	}
	return &run, nil
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

// validIfaceName matches safe interface names (alphanumeric plus hyphens, dots, underscores, and colons for alias interfaces).
// Linux limits interface names to 15 characters (IFNAMSIZ-1).
var validIfaceName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,14}$`)

// withDryRun appends the dry_run=true query parameter when dryRun is true.
func withDryRun(path string, dryRun bool) string {
	if dryRun {
		if strings.Contains(path, "?") {
			return path + "&dry_run=true"
		}
		return path + "?dry_run=true"
	}
	return path
}

// SetStaticIP configures a static IP on the given interface.
func (c *APIClient) SetStaticIP(name string, cfg StaticIPConfig, dryRun bool) (*NetworkWriteResult, error) {
	if !validIfaceName.MatchString(name) {
		return nil, fmt.Errorf("invalid interface name: %q", name)
	}
	payload, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	path := withDryRun("/api/v1/plugins/network/interfaces/"+name, dryRun)
	var r NetworkWriteResult
	if err := c.putConfirmJSON(path, string(payload), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// SetDNS configures DNS servers.
func (c *APIClient) SetDNS(cfg DNSWriteConfig, dryRun bool) (*NetworkWriteResult, error) {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	path := withDryRun("/api/v1/plugins/network/dns", dryRun)
	var r NetworkWriteResult
	if err := c.putConfirmJSON(path, string(payload), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteStaticIP removes static IP config, reverting to DHCP.
func (c *APIClient) DeleteStaticIP(name string, dryRun bool) (*NetworkWriteResult, error) {
	if !validIfaceName.MatchString(name) {
		return nil, fmt.Errorf("invalid interface name: %q", name)
	}
	path := withDryRun("/api/v1/plugins/network/interfaces/"+name, dryRun)
	var r NetworkWriteResult
	if err := c.deleteConfirmJSON(path, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// RollbackInterface restores previous interface configuration.
func (c *APIClient) RollbackInterface(name string, dryRun bool) (*NetworkWriteResult, error) {
	if !validIfaceName.MatchString(name) {
		return nil, fmt.Errorf("invalid interface name: %q", name)
	}
	path := withDryRun("/api/v1/plugins/network/interfaces/"+name+"/rollback", dryRun)
	var r NetworkWriteResult
	if err := c.postConfirmJSON(path, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// RollbackDNS restores previous DNS configuration.
func (c *APIClient) RollbackDNS(dryRun bool) (*NetworkWriteResult, error) {
	path := withDryRun("/api/v1/plugins/network/dns/rollback", dryRun)
	var r NetworkWriteResult
	if err := c.postConfirmJSON(path, &r); err != nil {
		return nil, err
	}
	return &r, nil
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
		return friendlyAPIError("GET", path, resp.StatusCode, body)
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
		return friendlyAPIError("POST", path, resp.StatusCode, b)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func (c *APIClient) putJSON(path, body string, out interface{}) error {
	req, err := http.NewRequest(http.MethodPut, c.baseURL+path, strings.NewReader(body))
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

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort error body
		return friendlyAPIError("PUT", path, resp.StatusCode, b)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// doConfirmJSON sends an HTTP request with X-Confirm: true header and optional
// JSON body. It is the shared implementation for putConfirmJSON,
// deleteConfirmJSON, and postConfirmJSON.
func (c *APIClient) doConfirmJSON(method, path string, body string, out interface{}) error {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Confirm", "true")
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
		return friendlyAPIError(method, path, resp.StatusCode, b)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// putConfirmJSON sends a PUT with X-Confirm: true header and JSON body.
func (c *APIClient) putConfirmJSON(path, body string, out interface{}) error {
	return c.doConfirmJSON(http.MethodPut, path, body, out)
}

// deleteConfirmJSON sends a DELETE with X-Confirm: true header.
func (c *APIClient) deleteConfirmJSON(path string, out interface{}) error {
	return c.doConfirmJSON(http.MethodDelete, path, "", out)
}

// postConfirmJSON sends a POST with X-Confirm: true header and no body.
func (c *APIClient) postConfirmJSON(path string, out interface{}) error {
	return c.doConfirmJSON(http.MethodPost, path, "", out)
}
