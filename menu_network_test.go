package tui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestActionNetworkMenu_ItemCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("actionNetworkMenu should not make API calls; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkMenu(api)
	cmd := cmdFn()
	msg := cmd()

	sm, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	if sm.title != "Network Manager" {
		t.Errorf("got title %q, want %q", sm.title, "Network Manager")
	}

	// Verify key items exist by title rather than an exact count,
	// so the test is stable when menu items are added or reordered.
	titleMap := make(map[string]bool)
	for _, item := range sm.items {
		titleMap[item.Title] = true
	}
	for _, expected := range []string{
		"List Interfaces", "Network Status", "DNS Settings",
		"Set Static IP", "Set DNS Servers", "Delete Static IP",
		"Rollback Interface", "Rollback DNS", "Back",
	} {
		if !titleMap[expected] {
			t.Errorf("missing expected menu item %q", expected)
		}
	}

	// Verify Rollback DNS has NeedsConfirm.
	for _, item := range sm.items {
		if item.Title == "Rollback DNS" && !item.NeedsConfirm {
			t.Error("Rollback DNS should have NeedsConfirm=true")
		}
	}

	// Verify the separator exists and has nil Action (non-selectable).
	foundSeparator := false
	for _, item := range sm.items {
		if strings.Contains(item.Title, "────") {
			foundSeparator = true
			if item.Action != nil {
				t.Error("separator item should have nil Action (non-selectable)")
			}
		}
	}
	if !foundSeparator {
		t.Error("expected a separator item in the network menu")
	}
}

func TestFormatNetworkWriteResult_WithMessage(t *testing.T) {
	res := &NetworkWriteResult{
		Message: "Static IP configured successfully",
		Changes: []string{"set address to 192.168.1.10/24", "set gateway to 192.168.1.1"},
	}
	got := formatNetworkWriteResult("Set IP for eth0", res)
	if !strings.Contains(got, "Set IP for eth0") {
		t.Errorf("result should contain operation name, got: %s", got)
	}
	if !strings.Contains(got, "Static IP configured successfully") {
		t.Errorf("result should contain message, got: %s", got)
	}
	if !strings.Contains(got, "set address to 192.168.1.10/24") {
		t.Errorf("result should contain first change, got: %s", got)
	}
	if !strings.Contains(got, "set gateway to 192.168.1.1") {
		t.Errorf("result should contain second change, got: %s", got)
	}
	if !strings.Contains(got, "Changes applied:") {
		t.Errorf("result should contain 'Changes applied:' header, got: %s", got)
	}
}

func TestFormatNetworkWriteResult_NoChanges(t *testing.T) {
	res := &NetworkWriteResult{Message: "No changes needed"}
	got := formatNetworkWriteResult("DNS rollback", res)
	if !strings.Contains(got, "DNS rollback") {
		t.Errorf("result should contain operation name, got: %s", got)
	}
	if strings.Contains(got, "Changes applied:") {
		t.Errorf("result should not contain 'Changes applied:' when no changes, got: %s", got)
	}
}

func TestFormatNetworkWriteResult_EmptyMessage(t *testing.T) {
	res := &NetworkWriteResult{
		Changes: []string{"removed static config"},
	}
	got := formatNetworkWriteResult("Delete IP", res)
	if !strings.Contains(got, "Delete IP") {
		t.Errorf("result should contain operation name, got: %s", got)
	}
	if !strings.Contains(got, "removed static config") {
		t.Errorf("result should contain change, got: %s", got)
	}
}

func TestFormatNetworkWriteResult_SanitizesOutput(t *testing.T) {
	res := &NetworkWriteResult{
		Message: "done\x1b[31m injected",
		Changes: []string{"change\x00hidden"},
	}
	got := formatNetworkWriteResult("op\x1b[0m name", res)
	if strings.Contains(got, "\x1b") {
		t.Errorf("result should not contain escape sequences, got: %s", got)
	}
	if strings.Contains(got, "\x00") {
		t.Errorf("result should not contain null bytes, got: %s", got)
	}
}

func TestActionNetworkSetStaticIP_ShowsInterfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
				{Name: "wlan0", MAC: "11:22:33:44:55:66", IP: "10.0.0.1", State: "up"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkSetStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	sm, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	if sm.title != "Set Static IP — Select Interface" {
		t.Errorf("got title %q", sm.title)
	}
	// 2 interfaces + Back = 3
	if len(sm.items) != 3 {
		t.Errorf("got %d items, want 3", len(sm.items))
	}
}

func TestActionNetworkDeleteStaticIP_ShowsInterfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkDeleteStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	sm, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	if !sm.items[0].NeedsConfirm {
		t.Error("interface item should have NeedsConfirm=true")
	}
}

func TestActionNetworkRollbackInterface_ShowsInterfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackInterface(api)
	cmd := cmdFn()
	msg := cmd()

	sm, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	if sm.title != "Rollback Interface — Select Interface" {
		t.Errorf("got title %q", sm.title)
	}
	if !sm.items[0].NeedsConfirm {
		t.Error("interface item should have NeedsConfirm=true")
	}
}

func TestActionNetworkRollbackDNS_Success(t *testing.T) {
	expected := newNetworkWriteResult("dns restored", []string{"nameservers reverted"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/dns/rollback" && r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(expected)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackDNS(api)
	cmd := cmdFn()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if !result.refreshMenu {
		t.Error("expected refreshMenu=true")
	}
	if !strings.Contains(result.detail, "dns restored") {
		t.Errorf("detail should contain message, got: %s", result.detail)
	}
}

func TestActionNetworkSetDNS_ReturnsEditInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/dns" {
			json.NewEncoder(w).Encode(DNSConfig{
				Nameservers: []string{"8.8.8.8", "8.8.4.4"},
				Search:      []string{"local"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkSetDNS(api)
	cmd := cmdFn()
	msg := cmd()

	input, ok := msg.(editInputMsg)
	if !ok {
		t.Fatalf("expected editInputMsg, got %T", msg)
	}
	if input.key != "network.dns" {
		t.Errorf("got key %q, want %q", input.key, "network.dns")
	}
	if input.currentVal != "8.8.8.8, 8.8.4.4" {
		t.Errorf("got currentVal %q, want %q", input.currentVal, "8.8.8.8, 8.8.4.4")
	}
}

func TestActionNetworkSetStaticIP_NoInterfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]NetworkInterface{})
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkSetStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected error for no interfaces")
	}
	if !strings.Contains(result.err.Error(), "no network interfaces found") {
		t.Errorf("error %q should mention no interfaces", result.err.Error())
	}
}

// --- TUI-TEST-3: SetStaticIP action closure ---

func TestActionNetworkSetStaticIP_ActionReturnsEditInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkSetStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	sm, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	// First item should be the interface (not Back).
	if len(sm.items) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(sm.items))
	}
	item := sm.items[0]
	if item.Action == nil {
		t.Fatal("interface item should have an Action")
	}

	// Invoke the action closure.
	innerCmd := item.Action()
	innerMsg := innerCmd()

	input, ok := innerMsg.(editInputMsg)
	if !ok {
		t.Fatalf("expected editInputMsg, got %T", innerMsg)
	}
	if input.key != "network.static_ip.eth0" {
		t.Errorf("key = %q, want %q", input.key, "network.static_ip.eth0")
	}
	if !strings.Contains(input.prompt, "eth0") {
		t.Errorf("prompt %q should mention eth0", input.prompt)
	}
	if input.currentVal != "192.168.1.5" {
		t.Errorf("currentVal = %q, want %q", input.currentVal, "192.168.1.5")
	}
}

// --- TUI-TEST-4: Delete action closure ---

func TestActionNetworkDeleteStaticIP_ActionCallsAPI(t *testing.T) {
	expected := NetworkWriteResult{Message: "reverted to DHCP", Changes: []string{"removed static"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		if r.URL.Path == "/api/v1/plugins/network/interfaces/eth0" && r.Method == http.MethodDelete {
			json.NewEncoder(w).Encode(expected)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkDeleteStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	item := sm.items[0]
	innerCmd := item.Action()
	innerMsg := innerCmd()

	result, ok := innerMsg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", innerMsg)
	}
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if !strings.Contains(result.detail, "reverted to DHCP") {
		t.Errorf("detail %q should contain message", result.detail)
	}
	if !result.refreshMenu {
		t.Error("expected refreshMenu=true")
	}
}

// --- TUI-TEST-5: Rollback action closure ---

func TestActionNetworkRollbackInterface_ActionCallsAPI(t *testing.T) {
	expected := NetworkWriteResult{Message: "rolled back", Changes: []string{"restored config"}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		if r.URL.Path == "/api/v1/plugins/network/interfaces/eth0/rollback" && r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(expected)
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackInterface(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	item := sm.items[0]
	innerCmd := item.Action()
	innerMsg := innerCmd()

	result, ok := innerMsg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", innerMsg)
	}
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if !strings.Contains(result.detail, "rolled back") {
		t.Errorf("detail %q should contain message", result.detail)
	}
	if !result.refreshMenu {
		t.Error("expected refreshMenu=true")
	}
}

// --- Multi-interface closure isolation tests ---

func TestActionNetworkSetStaticIP_MultiInterface_ClosureIsolation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]NetworkInterface{
			{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", State: "up"},
			{Name: "wlan0", MAC: "11:22:33:44:55:66", IP: "10.0.0.2", State: "up"},
		})
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	msg := actionNetworkSetStaticIP(api)()()
	sm := msg.(subMenuMsg)

	// 2 interfaces + Back
	if len(sm.items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(sm.items))
	}

	// Each closure must reference its own interface.
	for i, want := range []struct {
		title string
		key   string
		ip    string
	}{
		{"eth0", inputKeyNetworkStaticIPPrefix + "eth0", "10.0.0.1"},
		{"wlan0", inputKeyNetworkStaticIPPrefix + "wlan0", "10.0.0.2"},
	} {
		input := sm.items[i].Action()().(editInputMsg)
		if input.key != want.key {
			t.Errorf("item[%d] key = %q, want %q", i, input.key, want.key)
		}
		if !strings.Contains(input.prompt, want.title) {
			t.Errorf("item[%d] prompt %q should mention %s", i, input.prompt, want.title)
		}
		if input.currentVal != want.ip {
			t.Errorf("item[%d] currentVal = %q, want %q", i, input.currentVal, want.ip)
		}
	}
}

func TestActionNetworkDeleteStaticIP_MultiInterface_ClosureIsolation(t *testing.T) {
	var deletedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", State: "up"},
				{Name: "br0", MAC: "11:22:33:44:55:66", IP: "10.0.0.2", State: "up"},
			})
			return
		}
		deletedPath = r.URL.Path
		json.NewEncoder(w).Encode(NetworkWriteResult{Message: "ok"})
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	msg := actionNetworkDeleteStaticIP(api)()()
	sm := msg.(subMenuMsg)

	if len(sm.items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(sm.items))
	}

	// Invoke second item (br0) — must call DELETE for br0, not eth0.
	result := sm.items[1].Action()().(apiResultMsg)
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if !strings.Contains(deletedPath, "/br0") {
		t.Errorf("expected DELETE for br0, got path %q", deletedPath)
	}
}

func TestActionNetworkRollbackInterface_MultiInterface_ClosureIsolation(t *testing.T) {
	var rolledBackPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", State: "up"},
				{Name: "wlan1", MAC: "11:22:33:44:55:66", IP: "10.0.0.2", State: "up"},
			})
			return
		}
		rolledBackPath = r.URL.Path
		json.NewEncoder(w).Encode(NetworkWriteResult{Message: "ok"})
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	msg := actionNetworkRollbackInterface(api)()()
	sm := msg.(subMenuMsg)

	if len(sm.items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(sm.items))
	}

	// Invoke second item (wlan1) — must call rollback for wlan1, not eth0.
	result := sm.items[1].Action()().(apiResultMsg)
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if !strings.Contains(rolledBackPath, "/wlan1") {
		t.Errorf("expected rollback for wlan1, got path %q", rolledBackPath)
	}
}

// --- TUI-TEST-6: API error tests ---

func TestActionNetworkDeleteStaticIP_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "disk full"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkDeleteStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	item := sm.items[0]
	innerCmd := item.Action()
	innerMsg := innerCmd()

	result, ok := innerMsg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", innerMsg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 500 response")
	}
	if !strings.Contains(result.err.Error(), "disk full") {
		t.Errorf("error %q should contain 'disk full'", result.err.Error())
	}
}

func TestActionNetworkRollbackInterface_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "rollback not available"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackInterface(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	item := sm.items[0]
	innerCmd := item.Action()
	innerMsg := innerCmd()

	result, ok := innerMsg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", innerMsg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 500 response")
	}
	if !strings.Contains(result.err.Error(), "rollback not available") {
		t.Errorf("error %q should contain 'rollback not available'", result.err.Error())
	}
}

// --- DNS "none" fix validation ---

func TestActionNetworkSetDNS_EmptyNameservers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/dns" {
			json.NewEncoder(w).Encode(DNSConfig{
				Nameservers: []string{},
				Search:      []string{},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkSetDNS(api)
	cmd := cmdFn()
	msg := cmd()

	input, ok := msg.(editInputMsg)
	if !ok {
		t.Fatalf("expected editInputMsg, got %T", msg)
	}
	if input.currentVal != "" {
		t.Errorf("currentVal = %q, want empty string (not \"none\")", input.currentVal)
	}
}

// --- Sanitized ifName validation ---

func TestActionNetworkSetStaticIP_FiltersInvalidNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]NetworkInterface{
			{Name: "eth0\x1b[31m", MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", State: "up"},
			{Name: "eth1", MAC: "aa:bb:cc:dd:ee:00", IP: "10.0.0.2", State: "up"},
		})
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkSetStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	// Only eth1 + Back should remain; the ANSI-injected name is filtered.
	validItems := 0
	for _, item := range sm.items {
		if item.Title == "Back" {
			continue
		}
		validItems++
		if strings.Contains(item.Title, "\x1b") {
			t.Errorf("filtered interface should not appear: %q", item.Title)
		}
	}
	if validItems != 1 {
		t.Errorf("expected 1 valid interface, got %d", validItems)
	}
}

func TestActionNetworkDeleteStaticIP_FiltersInvalidNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]NetworkInterface{
			{Name: "eth0\x1b[31m", MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", State: "up"},
			{Name: "eth1", MAC: "aa:bb:cc:dd:ee:00", IP: "10.0.0.2", State: "up"},
		})
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkDeleteStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	validItems := 0
	for _, item := range sm.items {
		if item.Title == "Back" {
			continue
		}
		validItems++
		if strings.Contains(item.ConfirmMsg, "\x1b") {
			t.Errorf("ConfirmMsg should not contain escape sequences: %q", item.ConfirmMsg)
		}
	}
	if validItems != 1 {
		t.Errorf("expected 1 valid interface, got %d", validItems)
	}
}

func TestActionNetworkRollbackInterface_FiltersInvalidNames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]NetworkInterface{
			{Name: "eth0\x1b[31m", MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", State: "up"},
			{Name: "eth1", MAC: "aa:bb:cc:dd:ee:00", IP: "10.0.0.2", State: "up"},
		})
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackInterface(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	validItems := 0
	for _, item := range sm.items {
		if item.Title == "Back" {
			continue
		}
		validItems++
		if strings.Contains(item.ConfirmMsg, "\x1b") {
			t.Errorf("ConfirmMsg should not contain escape sequences: %q", item.ConfirmMsg)
		}
	}
	if validItems != 1 {
		t.Errorf("expected 1 valid interface, got %d", validItems)
	}
}

// --- R2-TUI-TEST-2: RollbackDNS API error path ---

func TestActionNetworkRollbackDNS_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/dns/rollback" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "rollback not possible"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackDNS(api)
	cmd := cmdFn()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 500 response")
	}
	if !strings.Contains(result.err.Error(), "rollback not possible") {
		t.Errorf("error %q should contain 'rollback not possible'", result.err.Error())
	}
}

// --- R2-TUI-TEST-3: Delete/Rollback with empty interfaces list ---

func TestActionNetworkDeleteStaticIP_NoInterfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" {
			json.NewEncoder(w).Encode([]NetworkInterface{})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkDeleteStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected error for no interfaces")
	}
	if !strings.Contains(result.err.Error(), "no network interfaces found") {
		t.Errorf("error %q should mention no interfaces", result.err.Error())
	}
}

func TestActionNetworkRollbackInterface_NoInterfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" {
			json.NewEncoder(w).Encode([]NetworkInterface{})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackInterface(api)
	cmd := cmdFn()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected error for no interfaces")
	}
	if !strings.Contains(result.err.Error(), "no network interfaces found") {
		t.Errorf("error %q should mention no interfaces", result.err.Error())
	}
}

// --- R2-TUI-TEST-4: SetDNS fetch error ---

func TestActionNetworkSetDNS_FetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/dns" {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "dns service unavailable"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkSetDNS(api)
	cmd := cmdFn()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 500 response")
	}
	if !strings.Contains(result.err.Error(), "dns service unavailable") {
		t.Errorf("error %q should contain 'dns service unavailable'", result.err.Error())
	}
}

// --- Policy denial (403) handler-level tests ---

func TestActionNetworkDeleteStaticIP_PolicyDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "interface 'eth0' is not allowed for write operations"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkDeleteStaticIP(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	item := sm.items[0]
	innerCmd := item.Action()
	innerMsg := innerCmd()

	result, ok := innerMsg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", innerMsg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 403 policy denial")
	}
	if !strings.Contains(result.err.Error(), "protected by write policy") {
		t.Errorf("error %q should contain 'protected by write policy'", result.err.Error())
	}
}

func TestActionNetworkRollbackInterface_PolicyDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/interfaces" && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]NetworkInterface{
				{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IP: "192.168.1.5", State: "up"},
			})
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "interface 'eth0' is not allowed for write operations"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackInterface(api)
	cmd := cmdFn()
	msg := cmd()

	sm := msg.(subMenuMsg)
	item := sm.items[0]
	innerCmd := item.Action()
	innerMsg := innerCmd()

	result, ok := innerMsg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", innerMsg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 403 policy denial")
	}
	if !strings.Contains(result.err.Error(), "protected by write policy") {
		t.Errorf("error %q should contain 'protected by write policy'", result.err.Error())
	}
}

func TestActionNetworkRollbackDNS_PolicyDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/plugins/network/dns/rollback" && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"message": "interface 'eth0' is not allowed for write operations"},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	cmdFn := actionNetworkRollbackDNS(api)
	cmd := cmdFn()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 403 policy denial")
	}
	if !strings.Contains(result.err.Error(), "protected by write policy") {
		t.Errorf("error %q should contain 'protected by write policy'", result.err.Error())
	}
}
