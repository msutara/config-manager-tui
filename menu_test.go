package tui

import "testing"

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
