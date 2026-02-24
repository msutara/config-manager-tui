package tui

import "testing"

func TestMainMenu(t *testing.T) {
	items := MainMenu()
	if len(items) < 3 {
		t.Fatalf("MainMenu() returned %d items, want at least 3", len(items))
	}

	// First item should be System Info
	if items[0].Title != "System Info" {
		t.Errorf("first item: got %q, want %q", items[0].Title, "System Info")
	}
	if items[0].Description == "" {
		t.Error("System Info should have a description")
	}

	// Last item should be Quit
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

func TestMenuItemDescriptions(t *testing.T) {
	items := MainMenu()
	for _, item := range items {
		if item.Title == "" {
			t.Error("menu item has empty Title")
		}
		if item.Description == "" {
			t.Errorf("menu item %q has empty Description", item.Title)
		}
	}
}
