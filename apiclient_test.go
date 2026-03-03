package tui

import (
	"encoding/json"
	"fmt"
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
		{"strips C1 controls", "abc\u0085\u008A\u009Fdef", "abcdef"},
		{"truncates long", strings.Repeat("x", 300), strings.Repeat("x", 200) + "..."},
		{"multibyte no truncation", strings.Repeat("é", 101), strings.Repeat("é", 101)},
		{"multibyte truncates", strings.Repeat("é", 250), strings.Repeat("é", 200) + "..."},
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
			"auto_security":      true,
			"security_available": true,
			"schedule":           "0 3 * * *",
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
	if cfg.AutoSecurity == nil || !*cfg.AutoSecurity {
		t.Error("expected AutoSecurity=true")
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
			"auto_security":      true,
			"security_available": false,
			"schedule":           "0 3 * * *",
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
			"auto_security": true,
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

// ---------- Plugin settings API tests ----------

func TestAPIClientGetPluginSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/plugins/update/settings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"config": map[string]any{
				"schedule":        "0 3 * * *",
				"auto_security":   true,
				"security_source": "detected",
			},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	ps, err := client.GetPluginSettings("update")
	if err != nil {
		t.Fatalf("GetPluginSettings: %v", err)
	}
	if ps.Config["schedule"] != "0 3 * * *" {
		t.Errorf("schedule = %v, want 0 3 * * *", ps.Config["schedule"])
	}
	if ps.Config["auto_security"] != true {
		t.Errorf("auto_security = %v, want true", ps.Config["auto_security"])
	}
}

func TestAPIClientGetPluginSettings_NotConfigurable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "not configurable"})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.GetPluginSettings("update")
	if err == nil {
		t.Fatal("expected error for 501 response")
	}
	if !strings.Contains(err.Error(), "501") {
		t.Errorf("error should mention 501: %v", err)
	}
}

func TestAPIClientUpdatePluginSetting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/plugins/update/settings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s", r.Method)
		}
		var body struct {
			Key   string `json:"key"`
			Value any    `json:"value"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Key != "schedule" {
			t.Errorf("key = %q, want schedule", body.Key)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"config": map[string]any{
				"schedule":        "0 4 * * *",
				"auto_security":   true,
				"security_source": "detected",
			},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.UpdatePluginSetting("update", "schedule", "0 4 * * *")
	if err != nil {
		t.Fatalf("UpdatePluginSetting: %v", err)
	}
	if res.Config["schedule"] != "0 4 * * *" {
		t.Errorf("schedule = %v, want 0 4 * * *", res.Config["schedule"])
	}
}

func TestAPIClientUpdatePluginSetting_WithWarning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"config":  map[string]any{"schedule": "0 4 * * *"},
			"warning": "scheduler restart required",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.UpdatePluginSetting("update", "schedule", "0 4 * * *")
	if err != nil {
		t.Fatalf("UpdatePluginSetting: %v", err)
	}
	if res.Warning != "scheduler restart required" {
		t.Errorf("warning = %q, want 'scheduler restart required'", res.Warning)
	}
}

// ---------- Job tracking API tests ----------

func TestAPIClientTriggerJob(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/jobs/trigger" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		var body struct {
			JobID string `json:"job_id"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.JobID != "update.full" {
			t.Errorf("job_id = %q, want update.full", body.JobID)
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(TriggerJobResult{Status: "accepted", JobID: "update.full"})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	res, err := client.TriggerJob("update.full")
	if err != nil {
		t.Fatalf("TriggerJob: %v", err)
	}
	if res.Status != "accepted" {
		t.Errorf("status = %q, want accepted", res.Status)
	}
	if res.JobID != "update.full" {
		t.Errorf("job_id = %q, want update.full", res.JobID)
	}
}

func TestAPIClientTriggerJob_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, `{"error":{"code":"job_not_found"}}`)
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.TriggerJob("missing.job")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestAPIClientTriggerJob_InvalidJobID(t *testing.T) {
	client := NewAPIClient("http://localhost:0")
	_, err := client.TriggerJob("../etc/passwd")
	if err == nil {
		t.Fatal("expected validation error for invalid job ID")
	}
	if !strings.Contains(err.Error(), "invalid job ID") {
		t.Errorf("expected 'invalid job ID' error, got: %v", err)
	}
}

func TestAPIClientGetJobRunLatest_Running(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/jobs/update.full/runs/latest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(JobRun{
			JobID:     "update.full",
			Status:    "running",
			StartedAt: "2026-03-02T04:00:00Z",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	run, err := client.GetJobRunLatest("update.full")
	if err != nil {
		t.Fatalf("GetJobRunLatest: %v", err)
	}
	if run.Status != "running" {
		t.Errorf("status = %q, want running", run.Status)
	}
}

func TestAPIClientGetJobRunLatest_Completed(t *testing.T) {
	end := "2026-03-02T04:00:10Z"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(JobRun{
			JobID:     "update.full",
			Status:    "completed",
			StartedAt: "2026-03-02T04:00:00Z",
			EndedAt:   &end,
			Duration:  "10s",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	run, err := client.GetJobRunLatest("update.full")
	if err != nil {
		t.Fatalf("GetJobRunLatest: %v", err)
	}
	if run.Status != "completed" {
		t.Errorf("status = %q, want completed", run.Status)
	}
	if run.Duration != "10s" {
		t.Errorf("duration = %q, want 10s", run.Duration)
	}
}

func TestAPIClientGetJobRunLatest_InvalidJobID(t *testing.T) {
	client := NewAPIClient("http://localhost:1")
	_, err := client.GetJobRunLatest("../etc/passwd")
	if err == nil {
		t.Fatal("expected error for invalid job ID")
	}
	if !strings.Contains(err.Error(), "invalid job ID") {
		t.Errorf("error should mention invalid job ID: %v", err)
	}
}

func TestAPIClientGetJobRunLatest_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"internal"}`)
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.GetJobRunLatest("update.full")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status 500: %v", err)
	}
}

func TestAPIClientUpdatePluginSetting_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid cron expression"})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.UpdatePluginSetting("update", "schedule", "not-a-cron")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("error should mention 400: %v", err)
	}
}

func TestAPIClientUpdatePluginSettingAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q, want Bearer secret-token", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"config": map[string]any{},
		})
	}))
	defer srv.Close()

	client := NewAPIClientWithToken(srv.URL, "secret-token")
	_, err := client.UpdatePluginSetting("update", "schedule", "0 4 * * *")
	if err != nil {
		t.Fatalf("UpdatePluginSetting with auth: %v", err)
	}
}

func TestAPIClientUpdatePluginSetting_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "{broken")
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	_, err := client.UpdatePluginSetting("update", "schedule", "0 4 * * *")
	if err == nil {
		t.Fatal("expected error for malformed JSON response")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("error should mention decode: %v", err)
	}
}

func TestAPIClientGetPluginSettings_InvalidName(t *testing.T) {
	api := NewAPIClient("http://localhost")
	for _, bad := range []string{"../etc", "up/date", "foo bar", "", "-start", "end-"} {
		_, err := api.GetPluginSettings(bad)
		if err == nil {
			t.Errorf("expected error for invalid plugin name %q", bad)
		}
		if !strings.Contains(err.Error(), "invalid plugin name") {
			t.Errorf("error should mention invalid plugin name for %q: %v", bad, err)
		}
	}
}

func TestAPIClientUpdatePluginSetting_InvalidName(t *testing.T) {
	api := NewAPIClient("http://localhost")
	_, err := api.UpdatePluginSetting("../traversal", "key", "val")
	if err == nil {
		t.Fatal("expected error for invalid plugin name")
	}
	if !strings.Contains(err.Error(), "invalid plugin name") {
		t.Errorf("error should mention invalid plugin name: %v", err)
	}
}
