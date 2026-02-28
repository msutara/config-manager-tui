package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------- Generic helpers ----------

func TestTruncateBody(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"short", "error detail", "error detail"},
		{"strips control chars", "err\x00or\x1b[31m", "error[31m"},
		{"strips del", "abc\x7fdef", "abcdef"},
		{"truncates long", strings.Repeat("x", 300), strings.Repeat("x", 200) + "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateBody([]byte(tt.in))
			if got != tt.want {
				t.Errorf("truncateBody(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// ---------- Generic plugin API tests ----------

func TestAPIClientGetPlugins(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/plugins" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]PluginRegistryEntry{
			{
				Name: "firewall", Version: "0.1.0",
				Description: "Firewall management",
				RoutePrefix: "/api/v1/plugins/firewall",
				Endpoints: []PluginEndpoint{
					{Method: "GET", Path: "/rules", Description: "Active rules"},
					{Method: "POST", Path: "/reload", Description: "Reload rules"},
				},
			},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	plugins, err := client.GetPlugins()
	if err != nil {
		t.Fatalf("GetPlugins: %v", err)
	}
	if len(plugins) != 1 {
		t.Fatalf("plugins: got %d, want 1", len(plugins))
	}
	if plugins[0].Name != "firewall" {
		t.Errorf("name: got %q, want %q", plugins[0].Name, "firewall")
	}
	if len(plugins[0].Endpoints) != 2 {
		t.Fatalf("endpoints: got %d, want 2", len(plugins[0].Endpoints))
	}
	if plugins[0].Endpoints[0].Method != "GET" {
		t.Errorf("first endpoint method: got %q, want %q", plugins[0].Endpoints[0].Method, "GET")
	}
}

func TestAPIClientGetPluginsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.GetPlugins()
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestAPIClientGetRawWithToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client := NewAPIClientWithToken(srv.URL, "raw-secret")
	_, err := client.GetRaw("/test")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if gotAuth != "Bearer raw-secret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer raw-secret")
	}
}

func TestAPIClientPostRawWithToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	client := NewAPIClientWithToken(srv.URL, "post-raw-secret")
	_, err := client.PostRaw("/test")
	if err != nil {
		t.Fatalf("PostRaw: %v", err)
	}
	if gotAuth != "Bearer post-raw-secret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer post-raw-secret")
	}
}

func TestAPIClientGetRaw(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/plugins/firewall/rules" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"rules":["allow 22/tcp"]}`))
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	body, err := client.GetRaw("/api/v1/plugins/firewall/rules")
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if !strings.Contains(body, "allow 22/tcp") {
		t.Errorf("body should contain rule data, got %q", body)
	}
}

func TestAPIClientGetRawError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.GetRaw("/fail")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention 500: %v", err)
	}
}

func TestAPIClientPostRaw(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	body, err := client.PostRaw("/api/v1/plugins/firewall/reload")
	if err != nil {
		t.Fatalf("PostRaw: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method: got %q, want POST", gotMethod)
	}
	if !strings.Contains(body, "ok") {
		t.Errorf("body should contain status, got %q", body)
	}
}

func TestAPIClientPostRawError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("broken"))
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.PostRaw("/fail")
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention 500: %v", err)
	}
}

func TestAPIClientPostRaw204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.PostRaw("/ok")
	if err != nil {
		t.Fatalf("204 should not be error: %v", err)
	}
}

// ---------- Core API tests ----------

func TestAPIClientGetNode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/node" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(NodeInfo{
			Hostname:      "testhost",
			OS:            "Debian 12",
			Kernel:        "6.1.0",
			Arch:          "arm",
			UptimeSeconds: 3661,
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	info, err := client.GetNode()
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if info.Hostname != "testhost" {
		t.Errorf("hostname: got %q, want %q", info.Hostname, "testhost")
	}
	if info.UptimeSeconds != 3661 {
		t.Errorf("uptime: got %d, want 3661", info.UptimeSeconds)
	}
}

func TestAPIClientGetNodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.GetNode()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestAPIClientGetNodeConnectionRefused(t *testing.T) {
	client := NewAPIClient("http://127.0.0.1:1") // nothing listening
	_, err := client.GetNode()
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

// ---------- Update plugin API tests ----------

func TestAPIClientGetUpdateStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/plugins/update/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]PendingUpdate{
			{Package: "curl", CurrentVersion: "7.88.1", NewVersion: "7.88.2", Security: true},
			{Package: "vim", CurrentVersion: "9.0.1", NewVersion: "9.0.2", Security: false},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	updates, err := client.GetUpdateStatus()
	if err != nil {
		t.Fatalf("GetUpdateStatus: %v", err)
	}
	if len(updates) != 2 {
		t.Fatalf("updates: got %d, want 2", len(updates))
	}
	if !updates[0].Security {
		t.Error("first update should be security")
	}
}

func TestAPIClientRunUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		var req struct {
			Type string `json:"type"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Type != "full" {
			t.Errorf("type: got %q, want %q", req.Type, "full")
		}
		json.NewEncoder(w).Encode(UpdateRunResult{
			Status: "completed",
			Type:   "full",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	r, err := client.RunUpdate("full")
	if err != nil {
		t.Fatalf("RunUpdate: %v", err)
	}
	if r.Type != "full" {
		t.Errorf("type: got %q, want %q", r.Type, "full")
	}
}

func TestAPIClientRunUpdateAccepted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(UpdateRunResult{
			Status: "completed",
			Type:   "security",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	r, err := client.RunUpdate("security")
	if err != nil {
		t.Fatalf("RunUpdate with 202: %v", err)
	}
	if r.Type != "security" {
		t.Errorf("type: got %q, want %q", r.Type, "security")
	}
}

// ---------- Network plugin API tests ----------

func TestAPIClientGetNetworkInterfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]NetworkInterface{
			{Name: "eth0", State: "up", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.10"},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	ifaces, err := client.GetNetworkInterfaces()
	if err != nil {
		t.Fatalf("GetNetworkInterfaces: %v", err)
	}
	if len(ifaces) != 1 {
		t.Fatalf("interfaces: got %d, want 1", len(ifaces))
	}
	if ifaces[0].Name != "eth0" {
		t.Errorf("name: got %q, want %q", ifaces[0].Name, "eth0")
	}
}

func TestAPIClientGetDNS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(DNSConfig{
			Nameservers: []string{"8.8.8.8", "1.1.1.1"},
			Search:      []string{"local"},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	dns, err := client.GetDNS()
	if err != nil {
		t.Fatalf("GetDNS: %v", err)
	}
	if len(dns.Nameservers) != 2 {
		t.Errorf("nameservers: got %d, want 2", len(dns.Nameservers))
	}
}

func TestAPIClientGetNetworkStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(NetworkStatus{
			DefaultGateway:    "192.168.1.1",
			DNSReachable:      true,
			InternetReachable: true,
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	s, err := client.GetNetworkStatus()
	if err != nil {
		t.Fatalf("GetNetworkStatus: %v", err)
	}
	if !s.DNSReachable {
		t.Error("dns_reachable should be true")
	}
}

func TestAPIClientGetUpdateLogs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(RunStatus{
			Type:     "full",
			Status:   "completed",
			Duration: "2m30s",
			Packages: 5,
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	rs, err := client.GetUpdateLogs()
	if err != nil {
		t.Fatalf("GetUpdateLogs: %v", err)
	}
	if rs.Packages != 5 {
		t.Errorf("packages: got %d, want 5", rs.Packages)
	}
}

// ---------- Client behavior tests ----------

func TestAPIClientTrailingSlashNormalized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/node" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(NodeInfo{Hostname: "test"})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL + "/")
	info, err := client.GetNode()
	if err != nil {
		t.Fatalf("GetNode with trailing slash: %v", err)
	}
	if info.Hostname != "test" {
		t.Errorf("hostname: got %q, want %q", info.Hostname, "test")
	}
}

func TestAPIClientMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{broken`)) //nolint:errcheck // test helper
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.GetNode()
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestAPIClientWithTokenSendsBearer(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(NodeInfo{Hostname: "auth-test"})
	}))
	defer srv.Close()

	client := NewAPIClientWithToken(srv.URL, "test-secret")
	_, err := client.GetNode()
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if gotAuth != "Bearer test-secret" {
		t.Errorf("Authorization: got %q, want %q", gotAuth, "Bearer test-secret")
	}
}

func TestAPIClientWithoutTokenOmitsHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(NodeInfo{Hostname: "no-auth"})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.GetNode()
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if gotAuth != "" {
		t.Errorf("Authorization: got %q, want empty", gotAuth)
	}
}

func TestAPIClientPostWithToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(UpdateRunResult{Status: "ok", Type: "full"})
	}))
	defer srv.Close()

	client := NewAPIClientWithToken(srv.URL, "post-secret")
	_, err := client.RunUpdate("full")
	if err != nil {
		t.Fatalf("RunUpdate: %v", err)
	}
	if gotAuth != "Bearer post-secret" {
		t.Errorf("Authorization: got %q, want %q", gotAuth, "Bearer post-secret")
	}
}

func TestAPIClientGetUpdateConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/plugins/update/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"auto_security_updates": true,
			"security_available":    true,
			"schedule":              "0 3 * * *",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	cfg, err := client.GetUpdateConfig()
	if err != nil {
		t.Fatalf("GetUpdateConfig: %v", err)
	}
	if cfg.SecurityAvailable == nil || !*cfg.SecurityAvailable {
		t.Error("expected SecurityAvailable=true")
	}
}

func TestAPIClientGetUpdateConfig_Unavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/plugins/update/config" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"auto_security_updates": true,
			"security_available":    false,
			"schedule":              "0 3 * * *",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	cfg, err := client.GetUpdateConfig()
	if err != nil {
		t.Fatalf("GetUpdateConfig: %v", err)
	}
	if cfg.SecurityAvailable == nil || *cfg.SecurityAvailable {
		t.Error("expected SecurityAvailable=false")
	}
}

func TestAPIClientGetUpdateConfig_MissingField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Older server or empty response — field absent.
		json.NewEncoder(w).Encode(map[string]any{
			"auto_security_updates": true,
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	cfg, err := client.GetUpdateConfig()
	if err != nil {
		t.Fatalf("GetUpdateConfig: %v", err)
	}
	if cfg.SecurityAvailable != nil {
		t.Errorf("expected SecurityAvailable=nil for missing field, got %v", *cfg.SecurityAvailable)
	}
}
