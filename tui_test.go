package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	m := New()
	if len(m.menuItems) == 0 {
		t.Fatal("New() should return a model with menu items")
	}
	if m.cursor != 0 {
		t.Errorf("cursor: got %d, want 0", m.cursor)
	}
	if m.quitting {
		t.Error("quitting should be false on init")
	}
}

func TestInit(t *testing.T) {
	m := New()
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestUpdateKeyDown(t *testing.T) {
	m := New()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

	updated, cmd := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 1 {
		t.Errorf("cursor after 'j': got %d, want 1", model.cursor)
	}
	if cmd != nil {
		t.Error("cmd should be nil for navigation")
	}
}

func TestUpdateKeyUp(t *testing.T) {
	m := New()
	m.cursor = 2
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}

	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 1 {
		t.Errorf("cursor after 'k': got %d, want 1", model.cursor)
	}
}

func TestUpdateKeyUpAtTop(t *testing.T) {
	m := New()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}

	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 0 {
		t.Errorf("cursor should stay at 0 when at top, got %d", model.cursor)
	}
}

func TestUpdateKeyDownAtBottom(t *testing.T) {
	m := New()
	m.cursor = len(m.menuItems) - 1
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != len(m.menuItems)-1 {
		t.Errorf("cursor should stay at bottom, got %d", model.cursor)
	}
}

func TestUpdateArrowKeys(t *testing.T) {
	m := New()

	// Down arrow
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(Model)
	if model.cursor != 1 {
		t.Errorf("cursor after down: got %d, want 1", model.cursor)
	}

	// Up arrow
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(Model)
	if model.cursor != 0 {
		t.Errorf("cursor after up: got %d, want 0", model.cursor)
	}
}

func TestUpdateQuit(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"q key", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}},
		{"ctrl+c", tea.KeyMsg{Type: tea.KeyCtrlC}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New()
			updated, cmd := m.Update(tt.msg)
			model := updated.(Model)
			if !model.quitting {
				t.Error("quitting should be true")
			}
			if cmd == nil {
				t.Error("cmd should not be nil (should be tea.Quit)")
			}
		})
	}
}

func TestUpdateEnterOnQuit(t *testing.T) {
	m := New()
	// Navigate to Quit (last item)
	m.cursor = len(m.menuItems) - 1
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if !model.quitting {
		t.Error("selecting Quit should set quitting=true")
	}
	if cmd == nil {
		t.Error("selecting Quit should return a cmd")
	}
}

func TestUpdateEnterNoAction(t *testing.T) {
	m := New()
	// First item (System Info) has no action
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.quitting {
		t.Error("selecting item without action should not quit")
	}
	if cmd != nil {
		t.Error("selecting item without action should return nil cmd")
	}
}

func TestUpdateEnterEmptyMenu(t *testing.T) {
	m := Model{menuItems: []MenuItem{}, cursor: 0}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.quitting {
		t.Error("enter on empty menu should not quit")
	}
	if cmd != nil {
		t.Error("enter on empty menu should return nil cmd")
	}
}

func TestUpdateUnknownKey(t *testing.T) {
	m := New()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model := updated.(Model)
	if model.cursor != 0 {
		t.Errorf("unknown key should not move cursor, got %d", model.cursor)
	}
	if cmd != nil {
		t.Error("unknown key should return nil cmd")
	}
}

func TestUpdateNonKeyMsg(t *testing.T) {
	m := New()
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model := updated.(Model)
	if model.cursor != 0 {
		t.Error("non-key msg should not change cursor")
	}
	if cmd != nil {
		t.Error("non-key msg should return nil cmd")
	}
}

func TestViewNormal(t *testing.T) {
	m := New()
	v := m.View()
	if v == "" {
		t.Fatal("View() should not return empty string")
	}
	if !containsStr(v, "Config Manager") {
		t.Error("View should contain header")
	}
	if !containsStr(v, "System Info") {
		t.Error("View should contain menu items")
	}
	if !containsStr(v, "quit") {
		t.Error("View should contain footer key hints")
	}
}

func TestViewQuitting(t *testing.T) {
	m := New()
	m.quitting = true
	v := m.View()
	if v != "Goodbye!\n" {
		t.Errorf("quitting view: got %q, want %q", v, "Goodbye!\n")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
