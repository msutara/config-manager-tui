package tui

import (
	"strings"
	"testing"
)

func TestRenderHeader(t *testing.T) {
	th := DefaultTheme()
	h := renderHeader(th)
	if h == "" {
		t.Fatal("renderHeader() should not return empty string")
	}
	if !strings.Contains(h, "Config Manager") {
		t.Error("header should contain 'Config Manager'")
	}
}

func TestRenderFooter(t *testing.T) {
	th := DefaultTheme()
	f := renderFooter(ModeStandalone, "", "", th)
	if f == "" {
		t.Fatal("renderFooter() should not return empty string")
	}
	if !strings.Contains(f, "quit") {
		t.Error("footer should mention quit key")
	}
	if !strings.Contains(f, "standalone") {
		t.Error("standalone footer should contain 'standalone' badge")
	}
}

func TestRenderFooterConnected(t *testing.T) {
	th := DefaultTheme()
	f := renderFooter(ModeConnected, "", "", th)
	if !strings.Contains(f, "connected") {
		t.Error("connected footer should contain 'connected' badge")
	}
	if strings.Contains(f, "standalone") {
		t.Error("connected footer should not contain 'standalone'")
	}
}

func TestRenderMainMenu(t *testing.T) {
	th := DefaultTheme()
	items := MainMenu(nil)
	result := renderMainMenu(items, 0, th)

	// Should contain all menu item titles
	for _, item := range items {
		if !strings.Contains(result, item.Title) {
			t.Errorf("menu should contain %q", item.Title)
		}
	}
}

func TestRenderMainMenuCursor(t *testing.T) {
	th := DefaultTheme()
	items := MainMenu(nil)

	// Cursor at 0 — first item should have indicator
	result := renderMainMenu(items, 0, th)
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) < len(items) {
		t.Fatalf("expected %d lines, got %d", len(items), len(lines))
	}
	if !strings.Contains(lines[0], "▸") {
		t.Error("first line should have cursor indicator '▸' when cursor=0")
	}
	if strings.Contains(lines[1], "▸") {
		t.Error("second line should not have cursor indicator when cursor=0")
	}

	// Cursor at 1 — second item should have indicator
	result = renderMainMenu(items, 1, th)
	lines = strings.Split(strings.TrimSpace(result), "\n")
	if !strings.Contains(lines[1], "▸") {
		t.Error("second line should have cursor indicator when cursor=1")
	}
	if strings.Contains(lines[0], "▸") {
		t.Error("first line should not have cursor indicator when cursor=1")
	}
}

func TestRenderMainMenuEmpty(t *testing.T) {
	th := DefaultTheme()
	result := renderMainMenu([]MenuItem{}, 0, th)
	if result != "" {
		t.Errorf("empty menu should render empty string, got %q", result)
	}
}

func TestRenderPluginView(t *testing.T) {
	th := DefaultTheme()
	items := []MenuItem{
		{Title: "Action One", Description: "First action"},
		{Title: "Action Two", Description: "Second action"},
	}
	result := renderPluginView("Test Plugin", items, 0, th)

	if !strings.Contains(result, "Test Plugin") {
		t.Error("plugin view should contain plugin name")
	}
	if !strings.Contains(result, "Action One") {
		t.Error("plugin view should contain first action")
	}
	if !strings.Contains(result, "Action Two") {
		t.Error("plugin view should contain second action")
	}
	if !strings.Contains(result, "▸") {
		t.Error("plugin view should contain cursor indicator")
	}
}

func TestRenderSubFooter(t *testing.T) {
	th := DefaultTheme()
	f := renderSubFooter(ModeStandalone, "", "", th)
	if f == "" {
		t.Fatal("renderSubFooter() should not return empty string")
	}
	if !strings.Contains(f, "back") {
		t.Error("sub-footer should mention back navigation")
	}
	if !strings.Contains(f, "backspace") {
		t.Error("sub-footer should mention backspace key")
	}
	if !strings.Contains(f, "standalone") {
		t.Error("standalone sub-footer should contain 'standalone' badge")
	}
}

func TestRenderSubFooterConnected(t *testing.T) {
	th := DefaultTheme()
	f := renderSubFooter(ModeConnected, "", "", th)
	if !strings.Contains(f, "connected") {
		t.Error("connected sub-footer should contain 'connected' badge")
	}
}

func TestRenderSubFooter_StatusBar(t *testing.T) {
	th := DefaultTheme()
	f := renderSubFooter(ModeStandalone, "myhost", "2h", th)
	if !strings.Contains(f, "myhost") {
		t.Error("sub-footer should contain hostname")
	}
	if !strings.Contains(f, "up 2h") {
		t.Error("sub-footer should contain uptime string")
	}
}

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th.Cursor == "" {
		t.Error("cursor glyph should not be empty")
	}
	if th.Separator == "" {
		t.Error("separator should not be empty")
	}
	if th.SepWidth == 0 {
		t.Error("separator width should not be zero")
	}
	if th.ConnBadgeText == "" {
		t.Error("connected badge text should not be empty")
	}
	if th.StandBadgeText == "" {
		t.Error("standalone badge text should not be empty")
	}
}

func TestRenderStatusBar_WithHostInfo(t *testing.T) {
	th := DefaultTheme()
	s := renderStatusBar("myhost", "3d 4h", th)
	if !strings.Contains(s, "myhost") {
		t.Error("status bar should contain hostname")
	}
	if !strings.Contains(s, "up 3d 4h") {
		t.Error("status bar should contain uptime")
	}
}

func TestRenderStatusBar_HostnameOnly(t *testing.T) {
	th := DefaultTheme()
	s := renderStatusBar("myhost", "", th)
	if !strings.Contains(s, "myhost") {
		t.Error("status bar should contain hostname")
	}
	if strings.Contains(s, "up") {
		t.Error("status bar should not contain 'up' when uptime is empty")
	}
}

func TestRenderStatusBar_Empty(t *testing.T) {
	th := DefaultTheme()
	s := renderStatusBar("", "", th)
	if s != "" {
		t.Error("status bar should be empty when hostname is empty")
	}
}

func TestRenderFooter_WithStatusBar(t *testing.T) {
	th := DefaultTheme()
	f := renderFooter(ModeStandalone, "myhost", "2h", th)
	if !strings.Contains(f, "myhost") {
		t.Error("footer should contain hostname from status bar")
	}
	if !strings.Contains(f, "up 2h") {
		t.Error("footer should contain uptime from status bar")
	}
}

func TestRenderInputFooter(t *testing.T) {
	th := DefaultTheme()
	f := renderInputFooter(ModeConnected, "myhost", "3d 4h", th)
	if !strings.Contains(f, "enter: save") {
		t.Error("input footer should contain save hint")
	}
	if !strings.Contains(f, "esc: cancel") {
		t.Error("input footer should contain cancel hint")
	}
	if !strings.Contains(f, "connected") {
		t.Error("input footer should contain connection badge")
	}
	if !strings.Contains(f, "myhost") {
		t.Error("input footer should contain hostname")
	}
}

func TestRenderInputFooter_Standalone(t *testing.T) {
	th := DefaultTheme()
	f := renderInputFooter(ModeStandalone, "", "", th)
	if !strings.Contains(f, "enter: save") {
		t.Error("input footer should contain save hint")
	}
	if !strings.Contains(f, "standalone") {
		t.Error("input footer should contain standalone badge")
	}
}

// ---------- formatJobHistory tests ----------

func TestFormatJobHistory(t *testing.T) {
	end := "2026-03-02T04:00:10Z"
	runs := []JobRun{
		{
			JobID:     "update.full",
			Status:    "completed",
			StartedAt: "2026-03-02T04:00:00Z",
			EndedAt:   &end,
			Duration:  "10s",
		},
		{
			JobID:     "update.full",
			Status:    "failed",
			StartedAt: "2026-03-01T04:00:00Z",
			Duration:  "5s",
			Error:     "package conflict",
		},
		{
			JobID:     "update.full",
			Status:    "running",
			StartedAt: "2026-03-03T04:00:00Z",
		},
	}

	result := formatJobHistory("update.full", runs)

	if !strings.Contains(result, "Job History: update.full") {
		t.Error("should contain title")
	}
	if !strings.Contains(result, "✓") {
		t.Error("completed run should show ✓")
	}
	if !strings.Contains(result, "✗") {
		t.Error("failed run should show ✗")
	}
	if !strings.Contains(result, "⟳") {
		t.Error("running run should show ⟳")
	}
	if !strings.Contains(result, "package conflict") {
		t.Error("failed run should show error message")
	}
	if !strings.Contains(result, "10s") {
		t.Error("should show duration for completed run")
	}
	if !strings.Contains(result, "2026-03-02T04:00:00") {
		t.Error("output should contain truncated timestamp")
	}
	if strings.Contains(result, "2026-03-02T04:00:00Z") {
		t.Error("timestamp should be truncated — trailing Z should be removed")
	}
}

func TestFormatJobHistory_Empty(t *testing.T) {
	result := formatJobHistory("update.full", nil)
	if !strings.Contains(result, "No executions recorded") {
		t.Errorf("empty history should say no executions, got %q", result)
	}
	if !strings.Contains(result, "update.full") {
		t.Error("empty history should still show job ID")
	}
}

func TestFormatJobHistory_LongError(t *testing.T) {
	runs := []JobRun{
		{
			JobID:     "update.full",
			Status:    "failed",
			StartedAt: "2026-03-01T04:00:00Z",
			Duration:  "5s",
			Error:     "this is a very long error message that should be truncated to fit in the table display",
		},
	}

	result := formatJobHistory("update.full", runs)
	if !strings.Contains(result, "…") {
		t.Error("long error should be truncated with ellipsis")
	}
	// The original error is longer than 30 runes.
	// Assert the tail of the original error is NOT in the output.
	if strings.Contains(result, "table display") {
		t.Error("long error should be truncated — tail should not appear")
	}
}

func TestFormatJobHistory_UnknownStatus(t *testing.T) {
	runs := []JobRun{
		{JobID: "test.job", Status: "cancelled", StartedAt: "2026-03-02T04:00:00Z", Duration: "5s"},
	}
	result := formatJobHistory("test.job", runs)
	if !strings.Contains(result, "•") {
		t.Error("unknown status should show bullet icon •")
	}
}

func TestFormatJobHistory_Sanitization(t *testing.T) {
	runs := []JobRun{
		{
			JobID:     "test.job",
			Status:    "failed",
			StartedAt: "2026-03-02\x1b[31mBAD",
			Error:     "fail\x07with\x1bbells",
			Duration:  "5s",
		},
	}
	result := formatJobHistory("inject\x1b[0m.job", runs)
	if strings.ContainsAny(result, "\x1b\x07") {
		t.Error("output should not contain control characters after sanitization")
	}
	if !strings.Contains(result, "inject") {
		t.Error("sanitized job ID should still contain safe characters")
	}
}

func TestFormatJobHistory_MissingDuration(t *testing.T) {
	runs := []JobRun{
		{
			JobID:     "update.full",
			Status:    "running",
			StartedAt: "2026-03-03T04:00:00Z",
		},
	}

	result := formatJobHistory("update.full", runs)
	if !strings.Contains(result, "-") {
		t.Error("missing duration should show dash")
	}
}
