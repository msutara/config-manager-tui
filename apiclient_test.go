package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestAPIClientGetUpdateStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/plugins/update/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(UpdateStatus{
			Status:  "idle",
			Pending: 5,
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	s, err := client.GetUpdateStatus()
	if err != nil {
		t.Fatalf("GetUpdateStatus: %v", err)
	}
	if s.Pending != 5 {
		t.Errorf("pending: got %d, want 5", s.Pending)
	}
}

func TestAPIClientRunUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		json.NewEncoder(w).Encode(UpdateRunResult{
			Status:  "started",
			Message: "update initiated",
			JobID:   "job-123",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	r, err := client.RunUpdate("full")
	if err != nil {
		t.Fatalf("RunUpdate: %v", err)
	}
	if r.JobID != "job-123" {
		t.Errorf("job_id: got %q, want %q", r.JobID, "job-123")
	}
}

func TestAPIClientRunUpdateAccepted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(UpdateRunResult{
			Status:  "queued",
			Message: "update queued",
			JobID:   "job-456",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	r, err := client.RunUpdate("security")
	if err != nil {
		t.Fatalf("RunUpdate with 202: %v", err)
	}
	if r.JobID != "job-456" {
		t.Errorf("job_id: got %q, want %q", r.JobID, "job-456")
	}
}

func TestAPIClientGetNetworkInterfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]NetworkInterface{
			{Name: "eth0", State: "up", Address: "192.168.1.10"},
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
			Servers: []string{"8.8.8.8", "1.1.1.1"},
			Search:  []string{"local"},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	dns, err := client.GetDNS()
	if err != nil {
		t.Fatalf("GetDNS: %v", err)
	}
	if len(dns.Servers) != 2 {
		t.Errorf("servers: got %d, want 2", len(dns.Servers))
	}
}

func TestAPIClientGetNetworkStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(NetworkStatus{
			Hostname:     "testhost",
			DefaultGW:    "192.168.1.1",
			Connectivity: "full",
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	s, err := client.GetNetworkStatus()
	if err != nil {
		t.Fatalf("GetNetworkStatus: %v", err)
	}
	if s.Connectivity != "full" {
		t.Errorf("connectivity: got %q, want %q", s.Connectivity, "full")
	}
}

func TestAPIClientGetUpdateLogs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode([]UpdateLogEntry{
			{Timestamp: "2026-01-01T00:00:00Z", Action: "full", Status: "success", Message: "ok"},
		})
	}))
	defer srv.Close()

	client := NewAPIClient(srv.URL)
	logs, err := client.GetUpdateLogs()
	if err != nil {
		t.Fatalf("GetUpdateLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs: got %d, want 1", len(logs))
	}
}

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
