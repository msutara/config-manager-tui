package tui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTitleCase(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "update", "Update"},
		{"hyphenated", "my-plugin", "My Plugin"},
		{"multi", "a-b-c", "A B C"},
		{"empty", "", ""},
		{"single char", "x", "X"},
		{"double hyphen", "a--b", "A B"},
		{"trailing hyphen", "trailing-", "Trailing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := titleCase(tt.in)
			if got != tt.want {
				t.Errorf("titleCase(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"clean", "Hello World", "Hello World"},
		{"null byte", "abc\x00def", "abcdef"},
		{"newline", "line1\nline2", "line1line2"},
		{"ansi escape", "bad\x1b[31mred\x1b[0m", "bad[31mred[0m"},
		{"tab", "col1\tcol2", "col1col2"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeText(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeBody(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"preserves newlines", "line1\nline2", "line1\nline2"},
		{"preserves tabs", "col1\tcol2", "col1\tcol2"},
		{"strips null", "abc\x00def", "abcdef"},
		{"strips ansi", "\x1b[31mred\x1b[0m", "[31mred[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeBody(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeBody(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCleanPluginPath(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		path   string
		want   string
	}{
		{"normal", "/api/v1/plugins/firewall", "/rules", "/api/v1/plugins/firewall/rules"},
		{"no leading slash", "/api/v1/plugins/firewall", "rules", "/api/v1/plugins/firewall/rules"},
		{"trailing slash prefix", "/api/v1/plugins/firewall/", "/rules", "/api/v1/plugins/firewall/rules"},
		{"traversal literal", "/api/v1/plugins/firewall", "/../../../etc/passwd", ""},
		{"traversal encoded", "/api/v1/plugins/firewall", "/%2e%2e/%2e%2e/secret", ""},
		{"double encoded", "/api/v1/plugins/firewall", "/%252e%252e/secret", ""},
		{"invalid escape", "/api/v1/plugins/firewall", "/%zz", ""},
		{"empty prefix", "", "/rules", ""},
		{"root path", "/api/v1/plugins/firewall", "/", "/api/v1/plugins/firewall"},
		{"dot-in-segment", "/api/v1/plugins/firewall", "/..status", "/api/v1/plugins/firewall/..status"},
		{"null byte", "/api/v1/plugins/firewall", "/x%00y", ""},
		{"newline", "/api/v1/plugins/firewall", "/x%0ay", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanPluginPath(tt.prefix, tt.path)
			if got != tt.want {
				t.Errorf("cleanPluginPath(%q, %q) = %q, want %q", tt.prefix, tt.path, got, tt.want)
			}
		})
	}
}

func TestMainMenuNoPlugins(t *testing.T) {
	items := MainMenu(nil)
	if len(items) != 2 {
		t.Fatalf("MainMenu(nil) returned %d items, want 2 (System Info + Quit)", len(items))
	}

	if items[0].Title != "System Info" {
		t.Errorf("first item: got %q, want %q", items[0].Title, "System Info")
	}

	last := items[len(items)-1]
	if last.Title != "Quit" {
		t.Errorf("last item: got %q, want %q", last.Title, "Quit")
	}
	if !last.IsQuit {
		t.Error("Quit item should have IsQuit=true")
	}
	if last.Action == nil {
		t.Error("Quit item should have a non-nil Action")
	}
}

func TestMainMenuEmptySlice(t *testing.T) {
	items := MainMenu([]PluginInfo{})
	if len(items) != 2 {
		t.Fatalf("MainMenu(empty) returned %d items, want 2", len(items))
	}
	if items[0].Title != "System Info" {
		t.Errorf("first: got %q, want %q", items[0].Title, "System Info")
	}
	if items[1].Title != "Quit" {
		t.Errorf("last: got %q, want %q", items[1].Title, "Quit")
	}
}

func TestMainMenuWithPlugins(t *testing.T) {
	plugins := []PluginInfo{
		{Name: "Update Management", Description: "OS and package updates"},
		{Name: "Network Config", Description: "Network interface management"},
	}
	items := MainMenu(plugins)

	// System Info + 2 plugins + Quit = 4
	if len(items) != 4 {
		t.Fatalf("MainMenu(2 plugins) returned %d items, want 4", len(items))
	}

	// First is System Info
	if items[0].Title != "System Info" {
		t.Errorf("first item: got %q, want %q", items[0].Title, "System Info")
	}

	// Middle items are plugins in order
	if items[1].Title != "Update Management" {
		t.Errorf("second item: got %q, want %q", items[1].Title, "Update Management")
	}
	if items[1].Description != "OS and package updates" {
		t.Errorf("second item desc: got %q", items[1].Description)
	}
	if items[2].Title != "Network Config" {
		t.Errorf("third item: got %q, want %q", items[2].Title, "Network Config")
	}

	// Last is Quit
	if items[3].Title != "Quit" {
		t.Errorf("last item: got %q, want %q", items[3].Title, "Quit")
	}
}

func TestMenuItemDescriptions(t *testing.T) {
	plugins := []PluginInfo{
		{Name: "Test Plugin", Description: "A test plugin"},
	}
	items := MainMenu(plugins)
	for _, item := range items {
		if item.Title == "" {
			t.Error("menu item has empty Title")
		}
		if item.Description == "" {
			t.Errorf("menu item %q has empty Description", item.Title)
		}
	}
}

func TestBuildMainMenuGenericPluginHasAction(t *testing.T) {
	plugins := []PluginInfo{
		{
			Name: "firewall", Description: "Firewall management",
			RoutePrefix: "/api/v1/plugins/firewall",
			Endpoints: []PluginEndpoint{
				{Method: "GET", Path: "/rules", Description: "Active rules"},
				{Method: "POST", Path: "/reload", Description: "Reload rules"},
			},
		},
	}
	m := New(plugins)

	var found bool
	for _, item := range m.menuItems {
		if item.Title == "Firewall" {
			found = true
			if item.Action == nil {
				t.Error("generic plugin should have Action wired")
			}
		}
	}
	if !found {
		t.Error("generic plugin should appear in menu as title-cased name")
	}
}

func TestBuildMainMenuGenericPluginNoEndpoints(t *testing.T) {
	plugins := []PluginInfo{
		{Name: "empty", Description: "No endpoints"},
	}
	m := New(plugins)

	var found bool
	for _, item := range m.menuItems {
		if item.Title == "Empty" {
			found = true
			if item.Action == nil {
				t.Error("generic plugin with no endpoints should still have Action (shows Back only)")
			}
		}
	}
	if !found {
		t.Error("empty plugin should appear in menu")
	}
}

func TestActionGenericPlugin_SubMenuContent(t *testing.T) {
	api := NewAPIClient("http://localhost:1") // not called during sub-menu build
	p := PluginInfo{
		Name: "firewall", Description: "Firewall management",
		RoutePrefix: "/api/v1/plugins/firewall",
		Endpoints: []PluginEndpoint{
			{Method: "GET", Path: "/rules", Description: "Active rules"},
			{Method: "POST", Path: "/reload", Description: "Reload rules"},
		},
	}

	action := actionGenericPlugin(api, p)
	cmd := action()
	msg := cmd()

	sub, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	if sub.title != "Firewall" {
		t.Errorf("title = %q, want %q", sub.title, "Firewall")
	}
	// Expect: 1 GET + 1 POST + Back = 3 items
	if len(sub.items) != 3 {
		t.Fatalf("items: got %d, want 3", len(sub.items))
	}
	if sub.items[0].Title != "Active rules" {
		t.Errorf("first item title = %q, want %q", sub.items[0].Title, "Active rules")
	}
	if sub.items[0].Description != "GET /rules" {
		t.Errorf("first item desc = %q, want %q", sub.items[0].Description, "GET /rules")
	}
	if sub.items[1].Title != "Reload rules" {
		t.Errorf("second item title = %q, want %q", sub.items[1].Title, "Reload rules")
	}
	if sub.items[2].Title != "Back" {
		t.Errorf("last item should be Back, got %q", sub.items[2].Title)
	}
}

func TestActionGenericPlugin_NoEndpoints(t *testing.T) {
	api := NewAPIClient("http://localhost:1")
	p := PluginInfo{Name: "empty", Description: "No endpoints"}

	action := actionGenericPlugin(api, p)
	cmd := action()
	msg := cmd()

	sub, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	if len(sub.items) != 1 {
		t.Fatalf("items: got %d, want 1 (Back only)", len(sub.items))
	}
	if sub.items[0].Title != "Back" {
		t.Errorf("only item should be Back, got %q", sub.items[0].Title)
	}
}

func TestActionGenericGet_PrettyPrintsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"a":1,"b":"two"}`))
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	action := actionGenericGet(api, "/test")
	cmd := action()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if !strings.Contains(result.detail, "  \"a\": 1") {
		t.Errorf("expected pretty-printed JSON, got %q", result.detail)
	}
}

func TestActionGenericGet_NonJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("plain text response"))
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	action := actionGenericGet(api, "/test")
	cmd := action()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.detail != "plain text response" {
		t.Errorf("expected raw text, got %q", result.detail)
	}
}

func TestActionGenericGet_Error(t *testing.T) {
	api := NewAPIClient("http://localhost:1") // nothing listening
	action := actionGenericGet(api, "/fail")
	cmd := action()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected error for unreachable API")
	}
}

func TestActionGenericPost_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	api := NewAPIClient(srv.URL)
	action := actionGenericPost(api, "/test", "Reload rules")
	cmd := action()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if !strings.Contains(result.detail, "Reload rules completed successfully") {
		t.Errorf("expected success message, got %q", result.detail)
	}
}

func TestActionGenericPost_Error(t *testing.T) {
	api := NewAPIClient("http://localhost:1") // nothing listening
	action := actionGenericPost(api, "/fail", "Test action")
	cmd := action()
	msg := cmd()

	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected error for unreachable API")
	}
}

func TestActionGenericPlugin_SkipsInvalidPaths(t *testing.T) {
	api := NewAPIClient("http://localhost:1")
	p := PluginInfo{
		Name: "bad", Description: "Bad paths",
		RoutePrefix: "/api/v1/plugins/bad",
		Endpoints: []PluginEndpoint{
			{Method: "GET", Path: "/%2e%2e/secret", Description: "Traversal GET"},
			{Method: "POST", Path: "/%2e%2e/admin", Description: "Traversal POST"},
			{Method: "GET", Path: "/valid", Description: "Valid GET"},
		},
	}

	action := actionGenericPlugin(api, p)
	cmd := action()
	msg := cmd()

	sub, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	// Only valid GET + Back = 2 items (traversal endpoints skipped).
	if len(sub.items) != 2 {
		t.Fatalf("items: got %d, want 2 (valid GET + Back)", len(sub.items))
	}
	if sub.items[0].Title != "Valid GET" {
		t.Errorf("first item = %q, want %q", sub.items[0].Title, "Valid GET")
	}
	if sub.items[1].Title != "Back" {
		t.Errorf("last item = %q, want %q", sub.items[1].Title, "Back")
	}
}

func TestActionGenericPlugin_BackAction(t *testing.T) {
	api := NewAPIClient("http://localhost:1")
	p := PluginInfo{Name: "test", Description: "Test"}

	action := actionGenericPlugin(api, p)
	cmd := action()
	msg := cmd()

	sub, ok := msg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg, got %T", msg)
	}
	// Execute the Back action.
	backCmd := sub.items[0].Action() // only item is Back
	backMsg := backCmd()

	backSub, ok := backMsg.(subMenuMsg)
	if !ok {
		t.Fatalf("expected subMenuMsg from Back, got %T", backMsg)
	}
	if backSub.title != "" {
		t.Errorf("Back should return empty title, got %q", backSub.title)
	}
}
