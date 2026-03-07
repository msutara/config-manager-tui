package tui

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// networkTestHandler records request details and returns a configurable response.
type networkTestHandler struct {
	gotMethod  string
	gotPath    string
	gotQuery   string
	gotHeaders http.Header
	gotBody    string
	respStatus int
	respBody   any
}

func (h *networkTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.gotMethod = r.Method
	h.gotPath = r.URL.Path
	h.gotQuery = r.URL.RawQuery
	h.gotHeaders = r.Header.Clone()
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		h.gotBody = string(b)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(h.respStatus)
	json.NewEncoder(w).Encode(h.respBody) //nolint:errcheck
}

func newNetworkWriteResult(msg string, changes []string) NetworkWriteResult {
	return NetworkWriteResult{
		Message: msg,
		Changes: changes,
	}
}

// ---------- SetStaticIP ----------

func TestSetStaticIP_Success(t *testing.T) {
	expected := newNetworkWriteResult("configured", []string{"set address"})
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: expected}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.SetStaticIP("eth0", StaticIPConfig{Address: "192.168.1.10/24", Gateway: "192.168.1.1"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "configured" {
		t.Errorf("got message %q, want %q", res.Message, "configured")
	}
	if h.gotMethod != http.MethodPut {
		t.Errorf("got method %q, want PUT", h.gotMethod)
	}
	if h.gotPath != "/api/v1/plugins/network/interfaces/eth0" {
		t.Errorf("got path %q, want /api/v1/plugins/network/interfaces/eth0", h.gotPath)
	}
	if h.gotHeaders.Get("X-Confirm") != "true" {
		t.Error("missing X-Confirm: true header")
	}
	if h.gotHeaders.Get("Content-Type") != "application/json" {
		t.Error("missing Content-Type: application/json header")
	}
	// Verify body contains the config.
	var body StaticIPConfig
	if err := json.Unmarshal([]byte(h.gotBody), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body.Address != "192.168.1.10/24" {
		t.Errorf("got address %q, want %q", body.Address, "192.168.1.10/24")
	}
}

func TestSetStaticIP_InvalidName(t *testing.T) {
	client := NewAPIClient("http://localhost:1")
	_, err := client.SetStaticIP("../etc/passwd", StaticIPConfig{Address: "1.2.3.4/24"}, false)
	if err == nil {
		t.Fatal("expected error for invalid interface name")
	}
	if !strings.Contains(err.Error(), "invalid interface name") {
		t.Errorf("error %q should contain 'invalid interface name'", err.Error())
	}
}

func TestSetStaticIP_DryRun(t *testing.T) {
	expected := newNetworkWriteResult("dry run ok", nil)
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: expected}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.SetStaticIP("eth0", StaticIPConfig{Address: "10.0.0.1/8"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.gotQuery != "dry_run=true" {
		t.Errorf("got query %q, want %q", h.gotQuery, "dry_run=true")
	}
}

func TestSetStaticIP_APIError(t *testing.T) {
	h := &networkTestHandler{
		respStatus: http.StatusBadRequest,
		respBody:   map[string]any{"error": map[string]any{"message": "invalid CIDR"}},
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.SetStaticIP("eth0", StaticIPConfig{Address: "bad"}, false)
	if err == nil {
		t.Fatal("expected error for API failure")
	}
	if !strings.Contains(err.Error(), "invalid CIDR") {
		t.Errorf("error %q should contain 'invalid CIDR'", err.Error())
	}
}

// ---------- SetDNS ----------

func TestSetDNS_Success(t *testing.T) {
	expected := newNetworkWriteResult("dns set", []string{"updated nameservers"})
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: expected}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.SetDNS(DNSWriteConfig{Nameservers: []string{"8.8.8.8", "1.1.1.1"}}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "dns set" {
		t.Errorf("got message %q, want %q", res.Message, "dns set")
	}
	if h.gotPath != "/api/v1/plugins/network/dns" {
		t.Errorf("got path %q, want /api/v1/plugins/network/dns", h.gotPath)
	}
	if h.gotHeaders.Get("X-Confirm") != "true" {
		t.Error("missing X-Confirm: true header")
	}
}

func TestSetDNS_DryRun(t *testing.T) {
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: newNetworkWriteResult("ok", nil)}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.SetDNS(DNSWriteConfig{Nameservers: []string{"8.8.8.8"}}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.gotQuery != "dry_run=true" {
		t.Errorf("got query %q, want %q", h.gotQuery, "dry_run=true")
	}
}

func TestSetDNS_APIError(t *testing.T) {
	h := &networkTestHandler{
		respStatus: http.StatusInternalServerError,
		respBody:   map[string]any{"error": map[string]any{"message": "write failed"}},
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.SetDNS(DNSWriteConfig{Nameservers: []string{"bad"}}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Errorf("error %q should contain 'write failed'", err.Error())
	}
}

// ---------- DeleteStaticIP ----------

func TestDeleteStaticIP_Success(t *testing.T) {
	expected := newNetworkWriteResult("reverted to DHCP", []string{"removed static config"})
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: expected}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.DeleteStaticIP("wlan0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "reverted to DHCP" {
		t.Errorf("got message %q, want %q", res.Message, "reverted to DHCP")
	}
	if h.gotMethod != http.MethodDelete {
		t.Errorf("got method %q, want DELETE", h.gotMethod)
	}
	if h.gotPath != "/api/v1/plugins/network/interfaces/wlan0" {
		t.Errorf("got path %q", h.gotPath)
	}
	if h.gotHeaders.Get("X-Confirm") != "true" {
		t.Error("missing X-Confirm: true header")
	}
}

func TestDeleteStaticIP_InvalidName(t *testing.T) {
	client := NewAPIClient("http://localhost:1")
	_, err := client.DeleteStaticIP("", false)
	if err == nil {
		t.Fatal("expected error for empty interface name")
	}
}

func TestDeleteStaticIP_DryRun(t *testing.T) {
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: newNetworkWriteResult("ok", nil)}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.DeleteStaticIP("eth0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.gotQuery != "dry_run=true" {
		t.Errorf("got query %q, want %q", h.gotQuery, "dry_run=true")
	}
}

func TestDeleteStaticIP_APIError(t *testing.T) {
	h := &networkTestHandler{
		respStatus: http.StatusNotFound,
		respBody:   map[string]any{"error": map[string]any{"message": "interface not found"}},
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.DeleteStaticIP("eth99", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "interface not found") {
		t.Errorf("error %q should contain 'interface not found'", err.Error())
	}
}

// ---------- RollbackInterface ----------

func TestRollbackInterface_Success(t *testing.T) {
	expected := newNetworkWriteResult("rolled back", []string{"restored previous config"})
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: expected}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.RollbackInterface("eth0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "rolled back" {
		t.Errorf("got message %q, want %q", res.Message, "rolled back")
	}
	if h.gotMethod != http.MethodPost {
		t.Errorf("got method %q, want POST", h.gotMethod)
	}
	if h.gotPath != "/api/v1/plugins/network/interfaces/eth0/rollback" {
		t.Errorf("got path %q", h.gotPath)
	}
	if h.gotHeaders.Get("X-Confirm") != "true" {
		t.Error("missing X-Confirm: true header")
	}
}

func TestRollbackInterface_InvalidName(t *testing.T) {
	client := NewAPIClient("http://localhost:1")
	_, err := client.RollbackInterface("bad name!", false)
	if err == nil {
		t.Fatal("expected error for invalid interface name")
	}
}

func TestRollbackInterface_DryRun(t *testing.T) {
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: newNetworkWriteResult("ok", nil)}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.RollbackInterface("eth0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.gotQuery != "dry_run=true" {
		t.Errorf("got query %q, want %q", h.gotQuery, "dry_run=true")
	}
}

func TestRollbackInterface_APIError(t *testing.T) {
	h := &networkTestHandler{
		respStatus: http.StatusConflict,
		respBody:   map[string]any{"error": map[string]any{"message": "no previous config"}},
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.RollbackInterface("eth0", false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no previous config") {
		t.Errorf("error %q should contain 'no previous config'", err.Error())
	}
}

// ---------- RollbackDNS ----------

func TestRollbackDNS_Success(t *testing.T) {
	expected := newNetworkWriteResult("dns rolled back", []string{"restored nameservers"})
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: expected}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.RollbackDNS(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "dns rolled back" {
		t.Errorf("got message %q, want %q", res.Message, "dns rolled back")
	}
	if h.gotMethod != http.MethodPost {
		t.Errorf("got method %q, want POST", h.gotMethod)
	}
	if h.gotPath != "/api/v1/plugins/network/dns/rollback" {
		t.Errorf("got path %q", h.gotPath)
	}
	if h.gotHeaders.Get("X-Confirm") != "true" {
		t.Error("missing X-Confirm: true header")
	}
}

func TestRollbackDNS_DryRun(t *testing.T) {
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: newNetworkWriteResult("ok", nil)}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.RollbackDNS(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.gotQuery != "dry_run=true" {
		t.Errorf("got query %q, want %q", h.gotQuery, "dry_run=true")
	}
}

func TestRollbackDNS_APIError(t *testing.T) {
	h := &networkTestHandler{
		respStatus: http.StatusInternalServerError,
		respBody:   map[string]any{"error": map[string]any{"message": "rollback failed"}},
	}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.RollbackDNS(false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "rollback failed") {
		t.Errorf("error %q should contain 'rollback failed'", err.Error())
	}
}

func TestRollbackDNS_Accepted(t *testing.T) {
	h := &networkTestHandler{respStatus: http.StatusAccepted, respBody: newNetworkWriteResult("accepted", nil)}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.RollbackDNS(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "accepted" {
		t.Errorf("got message %q, want %q", res.Message, "accepted")
	}
}

// ---------- Auth header ----------

func TestNetworkWriteOps_AuthHeader(t *testing.T) {
	h := &networkTestHandler{respStatus: http.StatusOK, respBody: newNetworkWriteResult("ok", nil)}
	srv := httptest.NewServer(h)
	defer srv.Close()

	client := NewAPIClientWithToken(srv.URL, "test-token-123")
	_, err := client.SetStaticIP("eth0", StaticIPConfig{Address: "10.0.0.1/24"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := h.gotHeaders.Get("Authorization"); got != "Bearer test-token-123" {
		t.Errorf("got Authorization %q, want %q", got, "Bearer test-token-123")
	}
}

// ---------- validIfaceName ----------

func TestValidIfaceName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"simple", "eth0", true},
		{"wireless", "wlan0", true},
		{"bridge", "br-lan", true},
		{"dots", "enp0s3.100", true},
		{"underscore", "docker_gwbridge", true},
		{"empty", "", false},
		{"starts-with-dot", ".hidden", false},
		{"starts-with-dash", "-bad", false},
		{"traversal", "../etc", false},
		{"space", "eth 0", false},
		{"slash", "net/iface", false},
		{"colon_alias", "eth0:1", true},
		{"null_byte", "eth0\x00bad", false},
		{"unicode", "ethö", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validIfaceName.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("validIfaceName(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}

func TestWithDryRun(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		dryRun bool
		want   string
	}{
		{"no_dry_run", "/api/v1/foo", false, "/api/v1/foo"},
		{"dry_run", "/api/v1/foo", true, "/api/v1/foo?dry_run=true"},
		{"existing_query", "/api/v1/foo?mode=preview", true, "/api/v1/foo?mode=preview&dry_run=true"},
		{"no_dry_run_with_query", "/api/v1/foo?x=1", false, "/api/v1/foo?x=1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withDryRun(tt.path, tt.dryRun)
			if got != tt.want {
				t.Errorf("withDryRun(%q, %v) = %q, want %q", tt.path, tt.dryRun, got, tt.want)
			}
		})
	}
}
