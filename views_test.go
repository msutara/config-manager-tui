package tui

import (
	"strings"
	"testing"
)

func TestRenderHeader(t *testing.T) {
	h := renderHeader()
	if h == "" {
		t.Fatal("renderHeader() should not return empty string")
	}
	if !strings.Contains(h, "Config Manager") {
		t.Error("header should contain 'Config Manager'")
	}
}

func TestRenderFooter(t *testing.T) {
	f := renderFooter(ModeStandalone)
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
	f := renderFooter(ModeConnected)
	if !strings.Contains(f, "connected") {
		t.Error("connected footer should contain 'connected' badge")
	}
	if strings.Contains(f, "standalone") {
		t.Error("connected footer should not contain 'standalone'")
	}
}

func TestRenderMainMenu(t *testing.T) {
	items := MainMenu(nil)
	result := renderMainMenu(items, 0)

	// Should contain all menu item titles
	for _, item := range items {
		if !strings.Contains(result, item.Title) {
			t.Errorf("menu should contain %q", item.Title)
		}
	}
}

func TestRenderMainMenuCursor(t *testing.T) {
	items := MainMenu(nil)

	// Cursor at 0 — first item should have indicator
	result := renderMainMenu(items, 0)
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
	result = renderMainMenu(items, 1)
	lines = strings.Split(strings.TrimSpace(result), "\n")
	if !strings.Contains(lines[1], "▸") {
		t.Error("second line should have cursor indicator when cursor=1")
	}
	if strings.Contains(lines[0], "▸") {
		t.Error("first line should not have cursor indicator when cursor=1")
	}
}

func TestRenderMainMenuEmpty(t *testing.T) {
	result := renderMainMenu([]MenuItem{}, 0)
	if result != "" {
		t.Errorf("empty menu should render empty string, got %q", result)
	}
}

func TestRenderPluginView(t *testing.T) {
	items := []MenuItem{
		{Title: "Action One", Description: "First action"},
		{Title: "Action Two", Description: "Second action"},
	}
	result := renderPluginView("Test Plugin", items, 0)

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
	f := renderSubFooter(ModeStandalone)
	if f == "" {
		t.Fatal("renderSubFooter() should not return empty string")
	}
	if !strings.Contains(f, "back") {
		t.Error("sub-footer should mention back navigation")
	}
	if !strings.Contains(f, "backspace") {
		t.Error("sub-footer should mention backspace key")
	}
}

func TestRenderSubFooterConnected(t *testing.T) {
	f := renderSubFooter(ModeConnected)
	if !strings.Contains(f, "connected") {
		t.Error("connected sub-footer should contain 'connected' badge")
	}
}
