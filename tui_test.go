package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	m := New(nil)
	if len(m.menuItems) == 0 {
		t.Fatal("New(nil) should return a model with menu items")
	}
	if m.cursor != 0 {
		t.Errorf("cursor: got %d, want 0", m.cursor)
	}
	if m.quitting {
		t.Error("quitting should be false on init")
	}
}

func TestNewWithPlugins(t *testing.T) {
	plugins := []PluginInfo{
		{Name: "update", Description: "Updates"},
		{Name: "network", Description: "Networking"},
	}
	m := New(plugins)
	// System Info + Update Manager + Network Manager + Quit = 4
	if len(m.menuItems) != 4 {
		t.Fatalf("New(2 plugins) menu items: got %d, want 4", len(m.menuItems))
	}
	if m.menuItems[1].Title != "Update Manager" {
		t.Errorf("first plugin: got %q, want %q", m.menuItems[1].Title, "Update Manager")
	}
}

func TestInit(t *testing.T) {
	m := New(nil)
	cmd := m.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestUpdateKeyDown(t *testing.T) {
	m := New(nil)
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
	m := New(nil)
	m.cursor = len(m.menuItems) - 1
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}

	updated, _ := m.Update(msg)
	model := updated.(Model)
	want := len(m.menuItems) - 2
	if model.cursor != want {
		t.Errorf("cursor after 'k': got %d, want %d", model.cursor, want)
	}
}

func TestUpdateKeyUpAtTop(t *testing.T) {
	m := New(nil)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}

	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != 0 {
		t.Errorf("cursor should stay at 0 when at top, got %d", model.cursor)
	}
}

func TestUpdateKeyDownAtBottom(t *testing.T) {
	m := New(nil)
	m.cursor = len(m.menuItems) - 1
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}

	updated, _ := m.Update(msg)
	model := updated.(Model)
	if model.cursor != len(m.menuItems)-1 {
		t.Errorf("cursor should stay at bottom, got %d", model.cursor)
	}
}

func TestUpdateArrowKeys(t *testing.T) {
	m := New(nil)

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
			m := New(nil)
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
	m := New(nil)
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
	// Create a model with an explicitly actionless item.
	m := Model{
		menuItems: []MenuItem{{Title: "Stub", Description: "no action"}},
		screen:    screenMain,
	}
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.quitting {
		t.Error("selecting item without action should not quit")
	}
	if cmd != nil {
		t.Error("selecting item without action should return nil cmd")
	}
}

func TestUpdateEnterSystemInfo(t *testing.T) {
	m := New(nil)
	// System Info is first item and now has an action.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.quitting {
		t.Error("selecting System Info should not quit")
	}
	if cmd == nil {
		t.Error("selecting System Info should return a cmd (API call)")
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
	m := New(nil)
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
	m := New(nil)
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
	m := New(nil)
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
	m := New(nil)
	m.quitting = true
	v := m.View()
	if v != "Goodbye!\n" {
		t.Errorf("quitting view: got %q, want %q", v, "Goodbye!\n")
	}
}

func TestSubMenuNavigation(t *testing.T) {
	m := New(nil)
	// Simulate receiving a subMenuMsg (as if a plugin menu was selected).
	subItems := []MenuItem{
		{Title: "Action 1", Description: "test action"},
		{Title: "Back", Description: "go back"},
	}
	updated, _ := m.Update(subMenuMsg{title: "Test Plugin", items: subItems})
	model := updated.(Model)
	if model.screen != screenSub {
		t.Errorf("screen: got %d, want %d (screenSub)", model.screen, screenSub)
	}
	if model.screenTitle != "Test Plugin" {
		t.Errorf("title: got %q, want %q", model.screenTitle, "Test Plugin")
	}
	if len(model.menuItems) != 2 {
		t.Fatalf("sub-menu items: got %d, want 2", len(model.menuItems))
	}
}

func TestSubMenuBack(t *testing.T) {
	m := New(nil)
	mainCount := len(m.menuItems)

	// Enter sub-menu.
	updated, _ := m.Update(subMenuMsg{
		title: "Test",
		items: []MenuItem{{Title: "X"}},
	})
	model := updated.(Model)

	// Press esc to go back.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.screen != screenMain {
		t.Errorf("screen after esc: got %d, want %d", model.screen, screenMain)
	}
	if len(model.menuItems) != mainCount {
		t.Errorf("menu items after back: got %d, want %d", len(model.menuItems), mainCount)
	}
}

func TestSubMenuQGoesBack(t *testing.T) {
	m := New(nil)
	// Enter sub-menu.
	updated, _ := m.Update(subMenuMsg{title: "X", items: []MenuItem{{Title: "Y"}}})
	model := updated.(Model)
	// In sub-menu, 'q' goes back (not quit).
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	model = updated.(Model)
	if model.quitting {
		t.Error("q in sub-menu should go back, not quit")
	}
	if model.screen != screenMain {
		t.Errorf("screen: got %d, want screenMain", model.screen)
	}
	if cmd != nil {
		t.Error("going back should not return a cmd")
	}
}

func TestDetailViewAnyKeyGoesBack(t *testing.T) {
	m := New(nil)
	// Simulate API result → detail screen.
	updated, _ := m.Update(apiResultMsg{detail: "Test detail"})
	model := updated.(Model)
	if model.screen != screenDetail {
		t.Fatalf("screen: got %d, want screenDetail", model.screen)
	}

	// Any key should go back to main.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(Model)
	if model.screen != screenMain {
		t.Errorf("screen after keypress: got %d, want screenMain", model.screen)
	}
}

func TestDetailViewShowsContent(t *testing.T) {
	m := New(nil)
	updated, _ := m.Update(apiResultMsg{detail: "Hello World"})
	model := updated.(Model)
	v := model.View()
	if !containsStr(v, "Hello World") {
		t.Error("detail view should contain the detail text")
	}
	if !containsStr(v, "Press any key") {
		t.Error("detail view should contain back hint")
	}
}

func TestDetailViewError(t *testing.T) {
	m := New(nil)
	updated, _ := m.Update(apiResultMsg{err: errTest})
	model := updated.(Model)
	v := model.View()
	if !containsStr(v, "Error:") {
		t.Error("error detail should show Error prefix")
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestEmptySubMenuMsgGoesBack(t *testing.T) {
	m := New(nil)
	// Enter sub-menu first.
	updated, _ := m.Update(subMenuMsg{title: "X", items: []MenuItem{{Title: "Y"}}})
	model := updated.(Model)
	// Empty subMenuMsg = back.
	updated, _ = model.Update(subMenuMsg{})
	model = updated.(Model)
	if model.screen != screenMain {
		t.Errorf("empty subMenuMsg should go back to main, got screen %d", model.screen)
	}
}

func TestNewWithAPI(t *testing.T) {
	m := NewWithAPI(nil, "http://example.com:9999")
	if m.api == nil {
		t.Fatal("api client should not be nil")
	}
	if m.api.baseURL != "http://example.com:9999" {
		t.Errorf("baseURL: got %q, want %q", m.api.baseURL, "http://example.com:9999")
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{59, "0m"},
		{3600, "1h 0m"},
		{3661, "1h 1m"},
		{90061, "1d 1h 1m"},
	}
	for _, tt := range tests {
		got := formatUptime(tt.seconds)
		if got != tt.want {
			t.Errorf("formatUptime(%d): got %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestDetailToSubNavigation(t *testing.T) {
	m := New(nil)
	// main → sub
	updated, _ := m.Update(subMenuMsg{title: "Test", items: []MenuItem{{Title: "Action"}}})
	model := updated.(Model)
	// sub → detail (via API result)
	updated, _ = model.Update(apiResultMsg{detail: "Result data"})
	model = updated.(Model)
	if model.screen != screenDetail {
		t.Fatalf("screen: got %d, want screenDetail", model.screen)
	}
	// detail → back to sub
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(Model)
	if model.screen != screenSub {
		t.Errorf("screen after back from detail: got %d, want screenSub", model.screen)
	}
}

func TestLoadingGuardPreventsDoubleDispatch(t *testing.T) {
	m := New(nil)
	// First enter dispatches a command.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if cmd == nil {
		t.Fatal("first enter should return a cmd")
	}
	if !model.loading {
		t.Error("loading should be true after dispatch")
	}
	// Second enter while loading should be ignored.
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(Model)
	if cmd != nil {
		t.Error("enter while loading should return nil cmd")
	}
}

func TestLoadingClearedOnAPIResult(t *testing.T) {
	m := New(nil)
	m.loading = true
	updated, _ := m.Update(apiResultMsg{detail: "done"})
	model := updated.(Model)
	if model.loading {
		t.Error("loading should be false after API result")
	}
}

func TestLoadingClearedOnSubMenuMsg(t *testing.T) {
	m := New(nil)
	m.loading = true
	updated, _ := m.Update(subMenuMsg{title: "X", items: []MenuItem{{Title: "Y"}}})
	model := updated.(Model)
	if model.loading {
		t.Error("loading should be false after sub menu msg")
	}
}

func TestLoadingClearedOnGoBack(t *testing.T) {
	m := New(nil)
	m.screen = screenSub
	m.parentItems = m.menuItems
	m.menuItems = []MenuItem{{Title: "X"}}

	// goBack itself clears loading (when called directly).
	m.loading = true
	m.goBack()
	if m.loading {
		t.Error("goBack should clear loading")
	}
}

func TestBackBlockedWhileLoading(t *testing.T) {
	m := New(nil)
	m.screen = screenSub
	m.parentItems = m.menuItems
	m.menuItems = []MenuItem{{Title: "X"}}
	m.loading = true

	// esc while loading should be ignored.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model := updated.(Model)
	if model.screen != screenSub {
		t.Error("esc while loading should stay in sub-menu")
	}

	// q while loading should also be ignored (q = back in sub-menu).
	model.loading = true
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	model = updated.(Model)
	if model.screen != screenSub {
		t.Error("q while loading should stay in sub-menu")
	}
}

func TestCtrlCAlwaysQuits(t *testing.T) {
	m := New(nil)
	m.screen = screenSub
	m.parentItems = m.menuItems
	m.menuItems = []MenuItem{{Title: "X"}}
	m.loading = true

	// ctrl+c should always quit, even while loading.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model := updated.(Model)
	if !model.quitting {
		t.Error("ctrl+c should quit even when loading")
	}
	if cmd == nil {
		t.Error("ctrl+c should return a cmd (tea.Quit)")
	}
}

func TestParentCursorRestored(t *testing.T) {
	m := New([]PluginInfo{{Name: "update", Description: "updates"}, {Name: "network", Description: "net"}})
	// Move cursor to second plugin item (index 2: System Info=0, Update=1, Network=2).
	m.cursor = 2
	// Enter sub-menu.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if cmd == nil {
		t.Fatal("expected cmd from enter")
	}
	// Simulate subMenuMsg.
	msg := cmd()
	updated, _ = model.Update(msg)
	model = updated.(Model)
	if model.screen != screenSub {
		t.Fatalf("expected screenSub, got %d", model.screen)
	}
	// Go back.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.cursor != 2 {
		t.Errorf("parent cursor: got %d, want 2", model.cursor)
	}
}

func TestFormatUptimeZero(t *testing.T) {
	result := formatUptime(0)
	if result != "0m" {
		t.Errorf("formatUptime(0): got %q, want %q", result, "0m")
	}
}

func TestFormatUptimeNegative(t *testing.T) {
	// Negative values are not expected in practice but should not panic.
	result := formatUptime(-100)
	if result == "" {
		t.Error("formatUptime(-100) should return a non-empty string")
	}
}

func TestBuildMainMenuUnknownPlugin(t *testing.T) {
	m := New([]PluginInfo{{Name: "custom-plugin", Description: "A custom one"}})
	found := false
	for _, item := range m.menuItems {
		if item.Title == "custom-plugin" {
			found = true
			if item.Action != nil {
				t.Error("unknown plugin should have nil Action")
			}
		}
	}
	if !found {
		t.Error("unknown plugin should appear in menu by name")
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
