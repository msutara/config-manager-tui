package tui

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

func TestNewWithAuth(t *testing.T) {
	m := NewWithAuth(nil, "http://localhost:9999", "my-token")
	if m.api == nil {
		t.Fatal("api client should not be nil")
	}
	if m.api.token != "my-token" {
		t.Errorf("token: got %q, want %q", m.api.token, "my-token")
	}
	if m.api.baseURL != "http://localhost:9999" {
		t.Errorf("baseURL: got %q, want %q", m.api.baseURL, "http://localhost:9999")
	}
}

func TestInit(t *testing.T) {
	m := New(nil)
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() should return a command to fetch node info")
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

func TestNestedSubMenuBack(t *testing.T) {
	m := New(nil)
	mainCount := len(m.menuItems)

	// Enter first sub-menu (Network Manager).
	updated, _ := m.Update(subMenuMsg{
		title: "Network Manager",
		items: []MenuItem{{Title: "Set Static IP"}, {Title: "Back"}},
	})
	model := updated.(Model)
	if model.screen != screenSub {
		t.Fatalf("screen after first sub: got %d, want %d", model.screen, screenSub)
	}

	// Enter second-level sub-menu (interface picker).
	updated, _ = model.Update(subMenuMsg{
		title: "Set Static IP — Select Interface",
		items: []MenuItem{{Title: "eth0"}, {Title: "Back"}},
	})
	model = updated.(Model)
	if model.screen != screenSub {
		t.Fatalf("screen after nested sub: got %d, want %d", model.screen, screenSub)
	}
	if model.screenTitle != "Set Static IP — Select Interface" {
		t.Fatalf("title after nested sub: got %q", model.screenTitle)
	}

	// Press esc — should return to Network Manager (screenSub), not screenMain.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.screen != screenSub {
		t.Errorf("screen after back from nested: got %d, want screenSub (%d)", model.screen, screenSub)
	}
	if model.screenTitle != "Network Manager" {
		t.Errorf("title after back from nested: got %q, want %q", model.screenTitle, "Network Manager")
	}
	if len(model.menuItems) != 2 {
		t.Errorf("items after back from nested: got %d, want 2", len(model.menuItems))
	}

	// Press esc again — should return to main menu.
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.screen != screenMain {
		t.Errorf("screen after second back: got %d, want screenMain (%d)", model.screen, screenMain)
	}
	if len(model.menuItems) != mainCount {
		t.Errorf("items after second back: got %d, want %d", len(model.menuItems), mainCount)
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

func TestCtrlCQuitsFromDetailScreen(t *testing.T) {
	m := New(nil)
	updated, _ := m.Update(apiResultMsg{detail: "Test detail"})
	model := updated.(Model)
	if model.screen != screenDetail {
		t.Fatalf("screen: got %d, want screenDetail", model.screen)
	}

	// ctrl+c should quit, not just go back.
	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	model = updated.(Model)
	if !model.quitting {
		t.Error("ctrl+c from detail should quit")
	}
	if cmd == nil {
		t.Error("ctrl+c should return tea.Quit cmd")
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
	if m.connMode != ModeStandalone {
		t.Errorf("default connMode: got %d, want ModeStandalone", m.connMode)
	}
}

func TestSetConnectionMode(t *testing.T) {
	m := New(nil)
	if m.connMode != ModeStandalone {
		t.Fatalf("default: got %d, want ModeStandalone", m.connMode)
	}
	m.SetConnectionMode(ModeConnected)
	if m.connMode != ModeConnected {
		t.Errorf("after set: got %d, want ModeConnected", m.connMode)
	}
}

func TestViewShowsModeBadge(t *testing.T) {
	m := New(nil)
	v := m.View()
	if !containsStr(v, "standalone") {
		t.Error("standalone view should contain 'standalone' badge")
	}

	m.SetConnectionMode(ModeConnected)
	v = m.View()
	if !containsStr(v, "connected") {
		t.Error("connected view should contain 'connected' badge")
	}
}

func TestSubMenuViewShowsModeBadge(t *testing.T) {
	m := New(nil)
	m.SetConnectionMode(ModeConnected)
	// Enter sub-menu.
	updated, _ := m.Update(subMenuMsg{title: "Test", items: []MenuItem{{Title: "X"}}})
	model := updated.(Model)
	v := model.View()
	if !containsStr(v, "connected") {
		t.Error("sub-menu view should show 'connected' badge")
	}
}

func TestConnModePersistsAcrossNavigation(t *testing.T) {
	m := New(nil)
	m.SetConnectionMode(ModeConnected)

	// main → sub
	updated, _ := m.Update(subMenuMsg{title: "Test", items: []MenuItem{{Title: "Act"}}})
	model := updated.(Model)
	if model.connMode != ModeConnected {
		t.Error("connMode should persist after entering sub-menu")
	}

	// sub → detail
	updated, _ = model.Update(apiResultMsg{detail: "data"})
	model = updated.(Model)
	if model.connMode != ModeConnected {
		t.Error("connMode should persist after entering detail")
	}

	// detail → sub (any key)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model = updated.(Model)
	if model.connMode != ModeConnected {
		t.Error("connMode should persist after returning to sub")
	}
	if !containsStr(model.View(), "connected") {
		t.Error("sub-menu view should show 'connected' badge after returning from detail")
	}

	// sub → main (esc)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(Model)
	if model.connMode != ModeConnected {
		t.Error("connMode should persist after returning to main")
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{"under_1m", 59, "0m"},
		{"1h", 3600, "1h 0m"},
		{"1h_1m", 3661, "1h 1m"},
		{"1d_1h_1m", 90061, "1d 1h 1m"},
		{"max_int32", math.MaxInt32, "24855d 3h 14m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatUptime(tt.seconds)
			if got != tt.want {
				t.Errorf("formatUptime(%d): got %q, want %q", tt.seconds, got, tt.want)
			}
		})
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
	m.menuStack = []menuState{{items: m.menuItems, cursor: 0}}
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
	m.menuStack = []menuState{{items: m.menuItems, cursor: 0}}
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
	m.menuStack = []menuState{{items: m.menuItems, cursor: 0}}
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
		if item.Title == "Custom Plugin" {
			found = true
			if item.Action == nil {
				t.Error("unknown plugin should have generic Action wired")
			}
		}
	}
	if !found {
		t.Error("unknown plugin should appear in menu with title-cased name")
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

// ---------- Input screen tests ----------

func TestEditInputMsgSwitchesToInputScreen(t *testing.T) {
	m := New(nil)
	msg := editInputMsg{
		prompt:     "Enter schedule:",
		key:        "schedule",
		plugin:     "update",
		currentVal: "0 3 * * *",
	}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if m2.screen != screenInput {
		t.Errorf("screen = %v, want screenInput", m2.screen)
	}
	if m2.inputBuffer != "0 3 * * *" {
		t.Errorf("inputBuffer = %q, want '0 3 * * *'", m2.inputBuffer)
	}
	if m2.inputKey != "schedule" {
		t.Errorf("inputKey = %q, want 'schedule'", m2.inputKey)
	}
	if m2.inputPlugin != "update" {
		t.Errorf("inputPlugin = %q, want 'update'", m2.inputPlugin)
	}
}

func TestInputScreenRendersPromptAndBuffer(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputPrompt = "Enter cron:"
	m.inputBuffer = "0 4 * * *"
	view := m.View()
	if !strings.Contains(view, "Enter cron:") {
		t.Error("view should contain input prompt")
	}
	if !strings.Contains(view, "0 4 * * *") {
		t.Error("view should contain input buffer")
	}
	if !strings.Contains(view, "enter: save") {
		t.Error("view should contain key hints")
	}
}

func TestInputScreenTypingAppendsRunes(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = "0 3"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" * * *")})
	m2 := updated.(Model)
	if m2.inputBuffer != "0 3 * * *" {
		t.Errorf("inputBuffer = %q, want '0 3 * * *'", m2.inputBuffer)
	}
}

func TestInputScreenBackspaceDeletesLast(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = "abc"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m2 := updated.(Model)
	if m2.inputBuffer != "ab" {
		t.Errorf("inputBuffer = %q, want 'ab'", m2.inputBuffer)
	}
}

func TestInputScreenBackspaceOnEmpty(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = ""

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m2 := updated.(Model)
	if m2.inputBuffer != "" {
		t.Errorf("inputBuffer = %q, want empty", m2.inputBuffer)
	}
}

func TestInputScreenEscGoesBack(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.menuStack = []menuState{{items: []MenuItem{{Title: "test"}}, cursor: 0}}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := updated.(Model)
	if m2.screen != screenSub {
		t.Errorf("screen = %v, want screenSub after Esc", m2.screen)
	}
	if m2.inputBuffer != "" {
		t.Error("inputBuffer should be cleared after Esc")
	}
}

func TestInputScreenEscGoesToMainWhenNoParent(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.menuStack = nil

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := updated.(Model)
	if m2.screen != screenMain {
		t.Errorf("screen = %v, want screenMain", m2.screen)
	}
}

func TestInputScreenEnterReturnsCmd(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = "0 5 * * *"
	m.inputKey = "schedule"
	m.inputPlugin = "update"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter on input screen")
	}
}

func TestInputScreenEnterSanitizesValue(t *testing.T) {
	var captured struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Errorf("unmarshal body: %v", err)
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"ok","config":{"security_source":"always"}}`)
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "always\x00\x07"
	m.inputKey = "security_source"
	m.inputPlugin = "update"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil cmd from Enter")
	}
	// Execute the command to trigger the API call.
	cmd()

	if captured.Value != "always" {
		t.Errorf("API received unsanitized value: got %q, want %q", captured.Value, "always")
	}
	if strings.ContainsAny(captured.Value, "\x00\x07") {
		t.Error("API value should not contain control characters")
	}
}

func TestInputScreenEnterRejectsBadCron(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = "0 2 * * * MON"
	m.inputKey = "schedule"
	m.inputPlugin = "update"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if cmd != nil {
		t.Error("bad cron should not produce a cmd (no API call)")
	}
	if !strings.Contains(m2.statusMsg, "expected 5 fields") {
		t.Errorf("statusMsg should mention 5 fields, got: %q", m2.statusMsg)
	}
	if m2.loading {
		t.Error("should not enter loading state for invalid cron")
	}
}

func TestInputScreenEnterAllowsNonScheduleKeys(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = "any value here"
	m.inputKey = "security_source"
	m.inputPlugin = "update"

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("non-schedule keys should not be cron-validated")
	}
}

func TestInputScreenEnterAcceptsCronShortcuts(t *testing.T) {
	for _, shortcut := range []string{"@daily", "@weekly", "@monthly", "@hourly", "@yearly", "@annually", "@midnight"} {
		m := New(nil)
		m.screen = screenInput
		m.inputBuffer = shortcut
		m.inputKey = "schedule"
		m.inputPlugin = "update"

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if cmd == nil {
			t.Errorf("@-shortcut %q should be accepted, not rejected", shortcut)
		}
	}
}

func TestInputScreenCtrlCQuits(t *testing.T) {
	m := New(nil)
	m.screen = screenInput

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m2 := updated.(Model)
	if !m2.quitting {
		t.Error("ctrl+c should quit from input screen")
	}
	if cmd == nil {
		t.Error("expected quit cmd")
	}
}

func TestInputScreenSpaceAppendsSpace(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = "0"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	m2 := updated.(Model)
	if m2.inputBuffer != "0 " {
		t.Errorf("inputBuffer = %q, want %q", m2.inputBuffer, "0 ")
	}
}

func TestInputScreenLoadingGuardIgnoresKeys(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = "test"
	m.loading = true

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m2 := updated.(Model)
	if m2.inputBuffer != "test" {
		t.Errorf("inputBuffer = %q, want %q (loading should block input)", m2.inputBuffer, "test")
	}
	if cmd != nil {
		t.Error("loading should produce nil cmd")
	}
}

func TestApiResultMsg_SettingsShowsDetail(t *testing.T) {
	m := New(nil)
	msg := apiResultMsg{
		refreshMenu: true,
		detail:      "Updated schedule to 0 4 * * *",
	}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if m2.screen != screenDetail {
		t.Errorf("screen = %v, want screenDetail", m2.screen)
	}
	if !strings.Contains(m2.detail, "Updated schedule") {
		t.Error("detail should contain settings result")
	}
}

func TestApiResultMsg_SettingsShowsError(t *testing.T) {
	m := New(nil)
	msg := apiResultMsg{
		err: fmt.Errorf("invalid cron expression"),
	}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if m2.screen != screenDetail {
		t.Errorf("screen = %v, want screenDetail", m2.screen)
	}
	if !strings.Contains(m2.detail, "invalid cron") {
		t.Error("detail should contain error message")
	}
}

func TestApiResultMsg_SettingsSanitizesError(t *testing.T) {
	m := New(nil)
	msg := apiResultMsg{
		err: fmt.Errorf("bad value: \x1b[31mred\x1b[0m"),
	}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if strings.Contains(m2.detail, "\x1b") {
		t.Error("detail should not contain ANSI escape sequences")
	}
	if !strings.Contains(m2.detail, "bad value") {
		t.Error("detail should contain the error text")
	}
}

func TestApiResultMsg_Settings_SetsNeedsMenuRefresh(t *testing.T) {
	m := New(nil)
	msg := apiResultMsg{refreshMenu: true, detail: "Updated auto_security to ON"}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if !m2.needsMenuRefresh {
		t.Error("needsMenuRefresh should be true after successful settings change")
	}
}

func TestApiResultMsg_Settings_ErrorDoesNotSetNeedsMenuRefresh(t *testing.T) {
	m := New(nil)
	msg := apiResultMsg{refreshMenu: true, err: fmt.Errorf("fail")}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if m2.needsMenuRefresh {
		t.Error("needsMenuRefresh should be false after settings error")
	}
}

func TestApiResultMsg_NoRefreshDoesNotSetNeedsMenuRefresh(t *testing.T) {
	m := New(nil)
	msg := apiResultMsg{detail: "Some API result"}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if m2.needsMenuRefresh {
		t.Error("needsMenuRefresh should remain false when refreshMenu is not set")
	}
}

func TestMenuRefreshMsg_UpdatesItemsInPlace(t *testing.T) {
	m := New(nil)
	m.screen = screenSub
	m.menuStack = []menuState{{items: []MenuItem{{Title: "Main"}}, cursor: 0}}
	m.menuItems = []MenuItem{{Title: "Old1"}, {Title: "Old2"}}
	m.cursor = 1

	newItems := []MenuItem{{Title: "New1"}, {Title: "New2"}, {Title: "New3"}}
	updated, _ := m.Update(menuRefreshMsg{items: newItems})
	m2 := updated.(Model)

	if len(m2.menuItems) != 3 {
		t.Fatalf("menuItems length = %d, want 3", len(m2.menuItems))
	}
	if m2.menuItems[0].Title != "New1" {
		t.Errorf("first item = %q, want New1", m2.menuItems[0].Title)
	}
	// menuStack should be unchanged (not overwritten).
	if len(m2.menuStack) != 1 || m2.menuStack[0].items[0].Title != "Main" {
		t.Error("menuStack should remain unchanged after refresh")
	}
	// cursor should stay at 1 (within bounds).
	if m2.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m2.cursor)
	}
}

func TestMenuRefreshMsg_ClampsCursor(t *testing.T) {
	m := New(nil)
	m.screen = screenSub
	m.menuItems = []MenuItem{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	m.cursor = 2

	// Refresh with fewer items.
	updated, _ := m.Update(menuRefreshMsg{items: []MenuItem{{Title: "X"}}})
	m2 := updated.(Model)
	if m2.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", m2.cursor)
	}
}

func TestRefreshSkippedWhenGoBackReturnsToMain(t *testing.T) {
	m := New(nil)
	// Detail screen without menuStack → goBack returns to screenMain.
	m.screen = screenDetail
	m.needsMenuRefresh = true
	m.menuStack = nil

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m2 := updated.(Model)
	if m2.screen != screenMain {
		t.Errorf("screen = %v, want screenMain", m2.screen)
	}
	if cmd != nil {
		t.Error("cmd should be nil — no refresh when returning to main")
	}
	if m2.needsMenuRefresh {
		t.Error("needsMenuRefresh should be cleared even when refresh is skipped")
	}
}

func TestRefreshTriggeredWhenDismissingSettingsDetail(t *testing.T) {
	m := New(nil)
	m.screen = screenDetail
	m.needsMenuRefresh = true
	m.menuStack = []menuState{{items: []MenuItem{{Title: "Parent"}}, cursor: 0}}
	m.menuItems = []MenuItem{{Title: "Old"}}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m2 := updated.(Model)

	if m2.screen == screenMain {
		t.Errorf("screen = %v, want screenSub", m2.screen)
	}
	if m2.needsMenuRefresh {
		t.Error("needsMenuRefresh should be cleared after scheduling refresh")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd to perform menu refresh")
	}
	if !m2.loading {
		t.Error("loading should be true while menu refresh is in progress")
	}
	if m2.statusMsg == "" {
		t.Error("statusMsg should be set while menu refresh is in progress")
	}

	// Execute the refresh command and apply the resulting menuRefreshMsg.
	msg := cmd()
	refresh, ok := msg.(menuRefreshMsg)
	if !ok {
		t.Fatalf("expected menuRefreshMsg, got %T", msg)
	}

	updated3, _ := m2.Update(refresh)
	m3 := updated3.(Model)
	if m3.loading {
		t.Error("loading should be false after menu refresh is applied")
	}
	if len(m3.menuItems) == 0 {
		t.Error("menuItems should be populated after menu refresh")
	}
}

func TestAPIResultMsgSanitizesError(t *testing.T) {
	m := New(nil)
	msg := apiResultMsg{
		err: fmt.Errorf("fail: \x1b[1mbold\x1b[0m"),
	}
	updated, _ := m.Update(msg)
	m2 := updated.(Model)
	if strings.Contains(m2.detail, "\x1b") {
		t.Error("detail should not contain ANSI escape sequences")
	}
}

// ---------- Settings action tests ----------

func TestBoolOnOff(t *testing.T) {
	if boolOnOff(true) != "ON" {
		t.Error("boolOnOff(true) should be ON")
	}
	if boolOnOff(false) != "OFF" {
		t.Error("boolOnOff(false) should be OFF")
	}
}

func TestFormatSettingsResult(t *testing.T) {
	res := &PluginSettingsUpdateResult{
		Config: map[string]any{
			"schedule":      "0 4 * * *",
			"auto_security": true,
		},
		Warning: "test warning",
	}
	detail := formatSettingsResult("schedule", "0 4 * * *", res)
	if !strings.Contains(detail, `"schedule"`) {
		t.Error("should contain key name")
	}
	if !strings.Contains(detail, "test warning") {
		t.Error("should contain warning")
	}
	if !strings.Contains(detail, "Current settings") {
		t.Error("should contain settings header")
	}
}

func TestFormatSettingsResultNoWarning(t *testing.T) {
	res := &PluginSettingsUpdateResult{
		Config: map[string]any{"schedule": "0 4 * * *"},
	}
	detail := formatSettingsResult("schedule", "0 4 * * *", res)
	if strings.Contains(detail, "Warning") {
		t.Error("should not contain warning when empty")
	}
}

func TestConfirmDialog_YesExecutes(t *testing.T) {
	executed := false
	m := New(nil)
	m.screen = screenConfirm
	m.confirmTitle = "Run Full Update?"
	m.confirmMsg = "This will update all packages."
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			executed = true
			return apiResultMsg{detail: "done"}
		}
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	um := updated.(Model)
	if um.screen == screenConfirm {
		t.Error("should leave confirm screen after y")
	}
	if !um.loading {
		t.Error("should be loading after confirm")
	}
	if cmd == nil {
		t.Fatal("cmd should not be nil after confirm")
	}
	// Execute the command chain to verify it runs the action.
	msg := cmd()
	if !executed {
		t.Error("confirm action should have executed")
	}
	if res, ok := msg.(apiResultMsg); !ok {
		t.Errorf("msg type: got %T, want apiResultMsg", msg)
	} else if res.detail != "done" {
		t.Errorf("msg detail: got %q, want %q", res.detail, "done")
	}
}

func TestConfirmDialog_NoReturns(t *testing.T) {
	executed := false
	m := New(nil)
	m.screen = screenConfirm
	m.confirmTitle = "Delete?"
	m.confirmMsg = "Are you sure?"
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			executed = true
			return apiResultMsg{detail: "should not run"}
		}
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	um := updated.(Model)
	if um.screen != screenMain {
		t.Errorf("screen: got %d, want screenMain", um.screen)
	}
	if um.loading {
		t.Error("should not be loading after cancel")
	}
	if cmd != nil {
		t.Error("cmd should be nil after cancel")
	}
	if um.confirmTitle != "" {
		t.Error("confirmTitle should be cleared")
	}
	if executed {
		t.Error("action should not have executed on cancel")
	}
}

func TestConfirmDialog_EscCancels(t *testing.T) {
	executed := false
	m := New(nil)
	m.screen = screenConfirm
	m.confirmTitle = "Delete?"
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			executed = true
			return apiResultMsg{detail: "should not run"}
		}
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	um := updated.(Model)
	if um.screen != screenMain {
		t.Errorf("screen: got %d, want screenMain", um.screen)
	}
	if cmd != nil {
		t.Error("cmd should be nil after esc cancel")
	}
	if executed {
		t.Error("action should not have executed on esc cancel")
	}
}

func TestConfirmDialog_Render(t *testing.T) {
	m := New(nil)
	m.screen = screenConfirm
	m.confirmTitle = "Run Full Update?"
	m.confirmMsg = "This will update all packages."

	view := m.View()
	if !strings.Contains(view, "Run Full Update?") {
		t.Error("confirm view should contain title")
	}
	if !strings.Contains(view, "This will update all packages.") {
		t.Error("confirm view should contain message")
	}
	if !strings.Contains(view, "Yes") {
		t.Error("confirm view should contain Yes option")
	}
	if !strings.Contains(view, "No") {
		t.Error("confirm view should contain No option")
	}
}

func TestMenuItem_NeedsConfirm_EnterShowsConfirm(t *testing.T) {
	m := New(nil)
	m.menuItems = []MenuItem{
		{
			Title:        "Dangerous Action",
			Description:  "Does something dangerous",
			NeedsConfirm: true,
			ConfirmMsg:   "Are you sure?",
			Action: func() tea.Cmd {
				return func() tea.Msg { return apiResultMsg{detail: "done"} }
			},
		},
	}
	m.cursor = 0
	m.screen = screenMain

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	if um.screen != screenConfirm {
		t.Errorf("screen: got %d, want screenConfirm (%d)", um.screen, screenConfirm)
	}
	if cmd != nil {
		t.Error("should not execute action yet — should show confirm first")
	}
	if um.confirmTitle != "Dangerous Action?" {
		t.Errorf("confirmTitle: got %q, want %q", um.confirmTitle, "Dangerous Action?")
	}
}

func TestNodeInfoMsg_SetsHostnameUptime(t *testing.T) {
	m := New(nil)
	updated, _ := m.Update(nodeInfoMsg{hostname: "pi-node", uptime: 5*86400 + 2*3600})
	um := updated.(Model)
	if um.hostname != "pi-node" {
		t.Errorf("hostname: got %q, want %q", um.hostname, "pi-node")
	}
	if um.uptimeStr == "" {
		t.Error("uptimeStr should not be empty")
	}
	if !strings.Contains(um.uptimeStr, "5d") {
		t.Errorf("uptimeStr should contain '5d', got %q", um.uptimeStr)
	}
}

func TestConfirmDialog_EnterDoesNotConfirm(t *testing.T) {
	m := New(nil)
	m.screen = screenConfirm
	m.confirmTitle = "Delete?"
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg { return apiResultMsg{detail: "should not run"} }
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	// Enter is NOT a confirmation key (prevents double-tap bypass).
	if um.screen != screenConfirm {
		t.Error("enter should NOT confirm — should stay on confirm screen")
	}
	if cmd != nil {
		t.Error("cmd should be nil — enter must not trigger action")
	}
}

func TestConfirmDialog_QCancels(t *testing.T) {
	executed := false
	m := New(nil)
	m.screen = screenConfirm
	m.confirmTitle = "Delete?"
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			executed = true
			return apiResultMsg{detail: "should not run"}
		}
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	um := updated.(Model)
	if um.screen != screenMain {
		t.Errorf("screen: got %d, want screenMain", um.screen)
	}
	if cmd != nil {
		t.Error("cmd should be nil after cancel")
	}
	if executed {
		t.Error("action should not have executed on q cancel")
	}
}

func TestConfirmDialog_NilAction(t *testing.T) {
	m := New(nil)
	m.screen = screenConfirm
	m.confirmTitle = "Delete?"
	m.confirmAction = nil // nil action should not panic

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	um := updated.(Model)
	if um.screen != screenMain {
		t.Errorf("screen: got %d, want screenMain", um.screen)
	}
	if cmd != nil {
		t.Error("nil action should return nil cmd")
	}
	if um.loading {
		t.Error("should not be loading when action is nil")
	}
	if um.confirmTitle != "" {
		t.Error("confirmTitle should be cleared")
	}
}

func TestNodeInfoMsg_SanitizesHostname(t *testing.T) {
	m := New(nil)
	updated, _ := m.Update(nodeInfoMsg{hostname: "evil\x1b[31mhost", uptime: 100})
	um := updated.(Model)
	if strings.Contains(um.hostname, "\x1b") {
		t.Error("hostname should be sanitized — contains escape sequence")
	}
}

func TestConfirmFlow_EndToEnd_Accept(t *testing.T) {
	executed := false
	m := New(nil)
	m.menuItems = []MenuItem{
		{
			Title:        "Run Update",
			Description:  "Run full system update",
			NeedsConfirm: true,
			ConfirmMsg:   "This will update all packages.",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					executed = true
					return apiResultMsg{detail: "done"}
				}
			},
		},
	}
	m.cursor = 0
	m.screen = screenMain

	// Step 1: press Enter on NeedsConfirm item
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	if um.screen != screenConfirm {
		t.Fatalf("step 1: screen should be screenConfirm, got %d", um.screen)
	}
	if cmd != nil {
		t.Fatal("step 1: cmd should be nil — action not yet executed")
	}
	if um.confirmTitle != "Run Update?" {
		t.Errorf("step 1: confirmTitle: got %q", um.confirmTitle)
	}

	// Step 2: press 'y' to confirm
	updated2, cmd2 := um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	um2 := updated2.(Model)
	if um2.screen != screenMain {
		t.Errorf("step 2: screen: got %d, want screenMain", um2.screen)
	}
	if !um2.loading {
		t.Error("step 2: should be loading after confirm")
	}
	if um2.confirmTitle != "" {
		t.Error("step 2: confirmTitle should be cleared")
	}
	if um2.confirmAction != nil {
		t.Error("step 2: confirmAction should be cleared")
	}
	if cmd2 == nil {
		t.Fatal("step 2: cmd should not be nil — action should execute")
	}
	// Run the cmd to verify action fires.
	msg := cmd2()
	if !executed {
		t.Error("step 2: action cmd should have been executed")
	}
	if res, ok := msg.(apiResultMsg); !ok {
		t.Errorf("step 2: msg type: got %T, want apiResultMsg", msg)
	} else if res.detail != "done" {
		t.Errorf("step 2: msg detail: got %q, want %q", res.detail, "done")
	}
}

func TestConfirmFlow_EndToEnd_Cancel(t *testing.T) {
	m := New(nil)
	m.menuItems = []MenuItem{
		{
			Title:        "Run Update",
			Description:  "Run full system update",
			NeedsConfirm: true,
			ConfirmMsg:   "This will update all packages.",
			Action: func() tea.Cmd {
				return func() tea.Msg {
					return apiResultMsg{detail: "should not run"}
				}
			},
		},
	}
	m.cursor = 0
	m.screen = screenMain

	// Step 1: press Enter on NeedsConfirm item
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	um := updated.(Model)
	if um.screen != screenConfirm {
		t.Fatalf("step 1: screen should be screenConfirm, got %d", um.screen)
	}

	// Step 2: press 'n' to cancel
	updated2, cmd2 := um.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	um2 := updated2.(Model)
	if um2.screen != screenMain {
		t.Errorf("step 2: screen: got %d, want screenMain", um2.screen)
	}
	if um2.loading {
		t.Error("step 2: should not be loading after cancel")
	}
	if cmd2 != nil {
		t.Error("step 2: cmd should be nil — action should not execute")
	}
	if um2.confirmAction != nil {
		t.Error("step 2: confirmAction should be cleared")
	}
	if um2.confirmTitle != "" {
		t.Error("step 2: confirmTitle should be cleared")
	}
}

func TestConfirmFlow_Submenu_Accept(t *testing.T) {
	executed := false
	m := New(nil)
	// Simulate being in a plugin submenu: menuStack has parent state.
	m.menuStack = []menuState{{items: []MenuItem{{Title: "Back"}}, cursor: 0}}
	m.screen = screenConfirm
	m.confirmTitle = "Run Update?"
	m.confirmMsg = "This will update all packages."
	m.confirmAction = func() tea.Cmd {
		return func() tea.Msg {
			executed = true
			return apiResultMsg{detail: "done"}
		}
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	um := updated.(Model)
	if um.screen != screenSub {
		t.Errorf("should return to screenSub, got %d", um.screen)
	}
	if !um.loading {
		t.Error("should be loading after confirm")
	}
	if um.confirmTitle != "" {
		t.Error("confirmTitle should be cleared")
	}
	if um.confirmAction != nil {
		t.Error("confirmAction should be cleared")
	}
	if cmd == nil {
		t.Fatal("cmd should not be nil — action should execute")
	}
	msg := cmd()
	if !executed {
		t.Error("confirmation action command should have executed")
	}
	if res, ok := msg.(apiResultMsg); !ok {
		t.Errorf("cmd() should return apiResultMsg, got %T", msg)
	} else if res.detail != "done" {
		t.Errorf("apiResultMsg.detail = %q, want %q", res.detail, "done")
	}
}

func TestConfirmFlow_Submenu_Cancel(t *testing.T) {
	m := New(nil)
	m.menuStack = []menuState{{items: []MenuItem{{Title: "Back"}}, cursor: 0}}
	m.screen = screenConfirm
	m.confirmTitle = "Delete?"
	m.confirmAction = func() tea.Cmd { return nil }

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	um := updated.(Model)
	if um.screen != screenSub {
		t.Errorf("should return to screenSub, got %d", um.screen)
	}
	if um.loading {
		t.Error("should not be loading after cancel")
	}
	if cmd != nil {
		t.Error("cmd should be nil after cancel")
	}
	if um.confirmAction != nil {
		t.Error("confirmAction should be cleared")
	}
}

// ---------- Progress screen (Phase 4) ----------

func TestJobAcceptedMsg_TransitionsToProgress(t *testing.T) {
	m := New(nil)
	m.menuStack = []menuState{{items: []MenuItem{{Title: "Back"}}, cursor: 0}}
	m.screen = screenSub
	m.loading = true

	updated, cmd := m.Update(jobAcceptedMsg{jobID: "update.full", title: "Full Update"})
	um := updated.(Model)
	if um.screen != screenProgress {
		t.Fatalf("screen: got %d, want screenProgress", um.screen)
	}
	if um.progressJobID != "update.full" {
		t.Errorf("progressJobID: got %q, want update.full", um.progressJobID)
	}
	if um.progressTitle != "Full Update" {
		t.Errorf("progressTitle: got %q, want Full Update", um.progressTitle)
	}
	if um.loading {
		t.Error("loading should be false after transitioning to progress")
	}
	if cmd == nil {
		t.Fatal("cmd should not be nil — tick should start")
	}
	if um.progressSession != 1 {
		t.Errorf("progressSession: got %d, want 1 (incremented from 0)", um.progressSession)
	}
}

func TestTickMsg_AdvancesSpinner(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"
	m.progressTicks = 0

	// Tick 1: should advance and NOT poll (odd tick).
	updated, cmd := m.Update(tickMsg{})
	um := updated.(Model)
	if um.progressTicks != 1 {
		t.Errorf("ticks: got %d, want 1", um.progressTicks)
	}
	if cmd == nil {
		t.Fatal("cmd should not be nil — next tick expected")
	}
}

func TestTickMsg_PollsOnEvenTick(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTicks = 1 // next tick will be 2 (even) → poll

	updated, cmd := m.Update(tickMsg{})
	um := updated.(Model)
	if um.progressTicks != 2 {
		t.Errorf("ticks: got %d, want 2", um.progressTicks)
	}
	if cmd == nil {
		t.Fatal("cmd should not be nil — tick+poll batch expected")
	}
	if !um.pollInFlight {
		t.Error("pollInFlight should be true after dispatching poll")
	}
}

func TestTickMsg_IgnoredOutsideProgress(t *testing.T) {
	m := New(nil)
	m.screen = screenMain

	updated, cmd := m.Update(tickMsg{})
	um := updated.(Model)
	if um.screen != screenMain {
		t.Errorf("screen should remain screenMain")
	}
	if cmd != nil {
		t.Error("cmd should be nil — tick not applicable outside progress")
	}
}

func TestTickMsg_SkipsPollWhenInFlight(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTicks = 1 // next tick will be 2 (even) → would poll
	m.pollInFlight = true

	updated, cmd := m.Update(tickMsg{})
	um := updated.(Model)
	if um.progressTicks != 2 {
		t.Errorf("ticks: got %d, want 2", um.progressTicks)
	}
	// Should only return tickCmd (no poll batch) since poll is in flight.
	if cmd == nil {
		t.Fatal("cmd should not be nil — tick should still continue")
	}
}

func TestJobPollMsg_ClearsPollInFlight(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.pollInFlight = true

	updated, _ := m.Update(jobPollMsg{
		jobID: "update.full",
		run:   &JobRun{Status: "running"},
	})
	um := updated.(Model)
	if um.pollInFlight {
		t.Error("pollInFlight should be cleared after receiving poll result")
	}
}

func TestJobPollMsg_Completed(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"

	updated, cmd := m.Update(jobPollMsg{
		jobID: "update.full",
		run:   &JobRun{Status: "completed", Duration: "8s"},
	})
	um := updated.(Model)
	if um.screen != screenDetail {
		t.Fatalf("screen: got %d, want screenDetail", um.screen)
	}
	if !strings.Contains(um.detail, "completed") {
		t.Errorf("detail should contain 'completed': %q", um.detail)
	}
	if !strings.Contains(um.detail, "8s") {
		t.Errorf("detail should contain duration '8s': %q", um.detail)
	}
	if cmd != nil {
		t.Error("cmd should be nil — polling stops on completion")
	}
}

func TestJobPollMsg_Failed(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"

	updated, cmd := m.Update(jobPollMsg{
		jobID: "update.full",
		run:   &JobRun{Status: "failed", Error: "apt-get exited 100"},
	})
	um := updated.(Model)
	if um.screen != screenDetail {
		t.Fatalf("screen: got %d, want screenDetail", um.screen)
	}
	if !strings.Contains(um.detail, "failed") {
		t.Errorf("detail should contain 'failed': %q", um.detail)
	}
	if !strings.Contains(um.detail, "apt-get exited 100") {
		t.Errorf("detail should contain error message: %q", um.detail)
	}
	if cmd != nil {
		t.Error("cmd should be nil — polling stops on failure")
	}
}

func TestJobPollMsg_FailedGenericError(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"

	updated, _ := m.Update(jobPollMsg{
		jobID: "update.full",
		run:   &JobRun{Status: "failed"},
	})
	um := updated.(Model)
	if !strings.Contains(um.detail, "see server logs") {
		t.Errorf("detail should contain generic error: %q", um.detail)
	}
}

func TestJobPollMsg_Running(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"

	updated, cmd := m.Update(jobPollMsg{
		jobID: "update.full",
		run:   &JobRun{Status: "running"},
	})
	um := updated.(Model)
	if um.screen != screenProgress {
		t.Errorf("screen should remain screenProgress, got %d", um.screen)
	}
	if cmd != nil {
		t.Error("cmd should be nil — polling continues via next tick")
	}
}

func TestJobPollMsg_TransientError(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"

	updated, cmd := m.Update(jobPollMsg{jobID: "update.full", err: fmt.Errorf("connection refused")})
	um := updated.(Model)
	if um.screen != screenProgress {
		t.Errorf("screen should remain screenProgress on transient error")
	}
	if cmd != nil {
		t.Error("cmd should be nil — keep polling via next tick")
	}
}

func TestJobPollMsg_IgnoredOutsideProgress(t *testing.T) {
	m := New(nil)
	m.screen = screenMain

	updated, _ := m.Update(jobPollMsg{run: &JobRun{Status: "completed"}})
	um := updated.(Model)
	if um.screen != screenMain {
		t.Errorf("screen should remain screenMain")
	}
}

func TestProgressScreen_EscDismisses(t *testing.T) {
	m := New(nil)
	m.menuStack = []menuState{{items: []MenuItem{{Title: "Back"}}, cursor: 0}}
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"
	m.pollInFlight = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	um := updated.(Model)
	if um.screen != screenSub {
		t.Errorf("screen: got %d, want screenSub", um.screen)
	}
	if um.progressJobID != "" {
		t.Error("progressJobID should be cleared after dismiss")
	}
	if um.pollInFlight {
		t.Error("pollInFlight should be cleared after dismiss")
	}
}

func TestProgressScreen_QDismisses(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	um := updated.(Model)
	if um.screen != screenMain {
		t.Errorf("screen: got %d, want screenMain (no menuStack)", um.screen)
	}
}

func TestProgressScreen_View(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"
	m.progressTicks = 3

	view := m.View()
	if !strings.Contains(view, "Full Update") {
		t.Errorf("view should contain progress title: %q", view)
	}
	if !strings.Contains(view, "Elapsed:") {
		t.Errorf("view should contain elapsed time: %q", view)
	}
	if !strings.Contains(view, "Esc/q: cancel") {
		t.Errorf("view should contain dismiss hint: %q", view)
	}
}

func TestJobPollMsg_Completed_NoDuration(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"

	updated, _ := m.Update(jobPollMsg{
		jobID: "update.full",
		run:   &JobRun{Status: "completed"}, // no Duration field
	})
	um := updated.(Model)
	if um.screen != screenDetail {
		t.Fatalf("screen: got %d, want screenDetail", um.screen)
	}
	if !strings.Contains(um.detail, "completed") {
		t.Errorf("detail should contain 'completed': %q", um.detail)
	}
	// Fallback elapsed time should be present (not the API's Duration field).
	if um.detail == "" {
		t.Error("detail should not be empty")
	}
}

func TestJobPollMsg_StalePollDiscarded(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.security" // current job

	// Stale poll from a previously dismissed job.
	updated, cmd := m.Update(jobPollMsg{
		jobID: "update.full", // wrong job ID
		run:   &JobRun{Status: "completed", Duration: "5s"},
	})
	um := updated.(Model)
	if um.screen != screenProgress {
		t.Errorf("screen should remain screenProgress when poll is stale, got %d", um.screen)
	}
	if cmd != nil {
		t.Error("cmd should be nil for stale poll")
	}
}

func TestJobPollMsg_SameJobID_StaleSession(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressSession = 2 // current session

	// Poll from a previous session with the same jobID but old session counter.
	updated, cmd := m.Update(jobPollMsg{
		jobID:   "update.full",
		session: 1, // stale session
		run:     &JobRun{Status: "completed", Duration: "5s"},
	})
	um := updated.(Model)
	if um.screen != screenProgress {
		t.Errorf("screen should remain screenProgress for stale session, got %d", um.screen)
	}
	if cmd != nil {
		t.Error("cmd should be nil for stale session poll")
	}
}

func TestJobPollMsg_PersistentErrorSurfacesAfterThreshold(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"

	// First maxPollErrors-1 errors should stay on progress screen.
	for i := 0; i < maxPollErrors-1; i++ {
		updated, _ := m.Update(jobPollMsg{jobID: "update.full", err: fmt.Errorf("connection refused")})
		m = updated.(Model)
		if m.screen != screenProgress {
			t.Fatalf("error %d/%d should stay on progress, got screen %d", i+1, maxPollErrors-1, m.screen)
		}
		if m.pollErrors != i+1 {
			t.Fatalf("pollErrors should be %d, got %d", i+1, m.pollErrors)
		}
	}

	// The maxPollErrors-th error should transition to detail with error message.
	updated, cmd := m.Update(jobPollMsg{jobID: "update.full", err: fmt.Errorf("connection refused")})
	um := updated.(Model)
	if um.screen != screenDetail {
		t.Errorf("should transition to detail after %d errors, got screen %d", maxPollErrors, um.screen)
	}
	if !strings.Contains(um.detail, "Full Update") {
		t.Error("detail should contain the job title")
	}
	if !strings.Contains(um.detail, "connection refused") {
		t.Error("detail should contain the error text")
	}
	if cmd != nil {
		t.Error("cmd should be nil after surfacing error")
	}
}

func TestJobPollMsg_ErrorCounterResetsOnSuccess(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.pollErrors = maxPollErrors - 1 // one more error would surface

	// A successful poll resets the counter.
	updated, _ := m.Update(jobPollMsg{jobID: "update.full", run: &JobRun{Status: "running"}})
	um := updated.(Model)
	if um.pollErrors != 0 {
		t.Errorf("pollErrors should reset to 0 on success, got %d", um.pollErrors)
	}
	if um.screen != screenProgress {
		t.Error("should remain on progress for running status")
	}
}

func TestGoBack_ClearsProgressStart(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressStart = time.Now()
	m.progressJobID = "update.full"
	m.progressTitle = "Full Update"
	m.pollErrors = 3

	m.goBack()

	if !m.progressStart.IsZero() {
		t.Error("progressStart should be zero after goBack")
	}
	if m.pollErrors != 0 {
		t.Error("pollErrors should be 0 after goBack")
	}
}

func TestTickMsg_StaleSessionDiscarded(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressSession = 3

	// Tick from an old session should be discarded — no spinner advance, no cmd.
	updated, cmd := m.Update(tickMsg{session: 1})
	um := updated.(Model)
	if um.progressTicks != 0 {
		t.Errorf("ticks should not advance for stale session tick, got %d", um.progressTicks)
	}
	if cmd != nil {
		t.Error("cmd should be nil for stale session tick")
	}
}

func TestTickMsg_MatchingNonZeroSessionAccepted(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	m.progressSession = 5
	m.progressTicks = 0

	// Tick with matching session should advance spinner and schedule next tick.
	updated, cmd := m.Update(tickMsg{session: 5})
	um := updated.(Model)
	if um.progressTicks != 1 {
		t.Errorf("ticks: got %d, want 1", um.progressTicks)
	}
	if cmd == nil {
		t.Fatal("cmd should not be nil — next tick expected for matching session")
	}
}

func TestSanitizeValue_BoolTrue(t *testing.T) {
	got := sanitizeValue(true)
	if got != "ON" {
		t.Errorf("sanitizeValue(true): got %q, want %q", got, "ON")
	}
}

func TestSanitizeValue_BoolFalse(t *testing.T) {
	got := sanitizeValue(false)
	if got != "OFF" {
		t.Errorf("sanitizeValue(false): got %q, want %q", got, "OFF")
	}
}

func TestSanitizeValue_String(t *testing.T) {
	got := sanitizeValue("hello")
	if got != "hello" {
		t.Errorf("sanitizeValue(string): got %q, want %q", got, "hello")
	}
}

func TestSanitizeValue_Number(t *testing.T) {
	got := sanitizeValue(42)
	if got != "42" {
		t.Errorf("sanitizeValue(42): got %q, want %q", got, "42")
	}
}

func TestProgressScreen_EscKeyType(t *testing.T) {
	m := New(nil)
	m.screen = screenProgress
	m.progressJobID = "update.full"
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ := m.Update(msg)
	um := updated.(Model)
	if um.screen == screenProgress {
		t.Error("Esc (KeyType) should dismiss progress screen")
	}
}

// --- TUI-TEST-1: handleInputKey with network static IP keys ---

func TestHandleInputKey_NetworkStaticIP_EmptyAddress(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = ""
	m.inputKey = inputKeyNetworkStaticIPPrefix + "eth0"
	m.inputPlugin = "network"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if cmd != nil {
		t.Error("empty address should not produce a cmd")
	}
	if m2.statusMsg != "Address cannot be empty" {
		t.Errorf("statusMsg = %q, want %q", m2.statusMsg, "Address cannot be empty")
	}
}

func TestHandleInputKey_NetworkStaticIP_NoCIDR(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = "192.168.1.10"
	m.inputKey = inputKeyNetworkStaticIPPrefix + "eth0"
	m.inputPlugin = "network"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if cmd != nil {
		t.Error("address without CIDR should not produce a cmd")
	}
	if !strings.Contains(m2.statusMsg, "CIDR format") {
		t.Errorf("statusMsg %q should mention CIDR format", m2.statusMsg)
	}
}

func TestHandleInputKey_NetworkStaticIP_Valid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(NetworkWriteResult{Message: "ok", Changes: []string{"set address"}})
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "192.168.1.10/24"
	m.inputKey = inputKeyNetworkStaticIPPrefix + "eth0"
	m.inputPlugin = "network"

	// Enter should transition to confirmation screen, not execute immediately.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.screen != screenConfirm {
		t.Fatalf("expected screenConfirm, got %d", m2.screen)
	}
	if cmd != nil {
		t.Error("confirmation screen should not produce a cmd")
	}
	if m2.confirmAction == nil {
		t.Fatal("confirmAction should be set")
	}
	if !strings.Contains(m2.confirmMsg, "192.168.1.10/24") || !strings.Contains(m2.confirmMsg, "eth0") {
		t.Errorf("confirmMsg should mention address and interface, got: %s", m2.confirmMsg)
	}

	// Simulate pressing "y" to confirm.
	updated2, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m3 := updated2.(Model)
	if !m3.loading {
		t.Error("after confirm, loading should be true")
	}
	if cmd2 == nil {
		t.Fatal("after confirm, should return a non-nil cmd")
	}

	// Execute the closure and assert the apiResultMsg.
	msg := cmd2()
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
	if !strings.Contains(result.detail, "Static IP set for eth0") {
		t.Errorf("detail should mention eth0, got: %s", result.detail)
	}
}

func TestHandleInputKey_NetworkStaticIP_Cancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("API should not be called on cancel; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "192.168.1.10/24"
	m.inputKey = inputKeyNetworkStaticIPPrefix + "eth0"
	m.inputPlugin = "network"

	// Enter should transition to confirmation screen.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.screen != screenConfirm {
		t.Fatalf("expected screenConfirm, got %d", m2.screen)
	}
	if cmd != nil {
		t.Error("Enter should transition to confirm, not return a cmd")
	}

	// Simulate pressing "n" to cancel.
	updated2, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m3 := updated2.(Model)
	if m3.screen == screenConfirm {
		t.Error("after cancel, screen should not remain on screenConfirm")
	}
	if m3.loading {
		t.Error("after cancel, loading should be false")
	}
	if cmd2 != nil {
		t.Error("after cancel, cmd should be nil")
	}
	if m3.confirmAction != nil {
		t.Error("after cancel, confirmAction should be cleared")
	}
}

func TestHandleInputKey_NetworkStaticIP_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "device busy"},
		})
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "192.168.1.10/24"
	m.inputKey = inputKeyNetworkStaticIPPrefix + "eth0"
	m.inputPlugin = "network"

	// Enter should transition to confirmation screen.
	updated, enterCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.screen != screenConfirm {
		t.Fatalf("expected screenConfirm, got %d", m2.screen)
	}
	if enterCmd != nil {
		t.Error("Enter should transition to confirm, not return a cmd")
	}
	if m2.confirmAction == nil {
		t.Fatal("confirmAction should be set")
	}

	// Simulate pressing "y" to confirm.
	_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 500 response")
	}
	if !strings.Contains(result.err.Error(), "device busy") {
		t.Errorf("error %q should contain 'device busy'", result.err.Error())
	}
}

// --- TUI-TEST-2: handleInputKey with network DNS keys ---

func TestHandleInputKey_NetworkDNS_Empty(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = ""
	m.inputKey = inputKeyNetworkDNS
	m.inputPlugin = "network"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if cmd != nil {
		t.Error("empty DNS should not produce a cmd")
	}
	if m2.statusMsg != "At least one DNS server required" {
		t.Errorf("statusMsg = %q, want %q", m2.statusMsg, "At least one DNS server required")
	}
}

func TestHandleInputKey_NetworkDNS_OnlyCommas(t *testing.T) {
	m := New(nil)
	m.screen = screenInput
	m.inputBuffer = ",,,"
	m.inputKey = inputKeyNetworkDNS
	m.inputPlugin = "network"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if cmd != nil {
		t.Error("commas-only DNS should not produce a cmd")
	}
	if m2.statusMsg != "At least one DNS server required" {
		t.Errorf("statusMsg = %q, want %q", m2.statusMsg, "At least one DNS server required")
	}
}

func TestHandleInputKey_NetworkDNS_Valid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(NetworkWriteResult{Message: "ok", Changes: []string{"set nameservers"}})
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "8.8.8.8, 1.1.1.1"
	m.inputKey = inputKeyNetworkDNS
	m.inputPlugin = "network"

	// Enter should transition to confirmation screen, not execute immediately.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.screen != screenConfirm {
		t.Fatalf("expected screenConfirm, got %d", m2.screen)
	}
	if cmd != nil {
		t.Error("confirmation screen should not produce a cmd")
	}
	if m2.confirmAction == nil {
		t.Fatal("confirmAction should be set")
	}
	if !strings.Contains(m2.confirmMsg, "8.8.8.8") {
		t.Errorf("confirmMsg should mention DNS servers, got: %s", m2.confirmMsg)
	}

	// Simulate pressing "y" to confirm.
	updated2, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m3 := updated2.(Model)
	if !m3.loading {
		t.Error("after confirm, loading should be true")
	}
	if cmd2 == nil {
		t.Fatal("after confirm, should return a non-nil cmd")
	}

	// Execute the closure and assert the apiResultMsg.
	msg := cmd2()
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
	if !strings.Contains(result.detail, "DNS servers updated") {
		t.Errorf("detail should mention DNS update, got: %s", result.detail)
	}
}

func TestHandleInputKey_NetworkDNS_Cancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("API should not be called on cancel; got %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "8.8.8.8"
	m.inputKey = inputKeyNetworkDNS
	m.inputPlugin = "network"

	// Enter should transition to confirmation screen.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.screen != screenConfirm {
		t.Fatalf("expected screenConfirm, got %d", m2.screen)
	}
	if cmd != nil {
		t.Error("Enter should transition to confirm, not return a cmd")
	}

	// Simulate pressing "n" to cancel.
	updated2, cmd2 := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m3 := updated2.(Model)
	if m3.screen == screenConfirm {
		t.Error("after cancel, screen should not remain on screenConfirm")
	}
	if m3.loading {
		t.Error("after cancel, loading should be false")
	}
	if cmd2 != nil {
		t.Error("after cancel, cmd should be nil")
	}
	if m3.confirmAction != nil {
		t.Error("after cancel, confirmAction should be cleared")
	}
}

func TestHandleInputKey_NetworkDNS_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "write failed"},
		})
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "8.8.8.8"
	m.inputKey = inputKeyNetworkDNS
	m.inputPlugin = "network"

	// Enter should transition to confirmation screen.
	updated, enterCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.screen != screenConfirm {
		t.Fatalf("expected screenConfirm, got %d", m2.screen)
	}
	if enterCmd != nil {
		t.Error("Enter should transition to confirm, not return a cmd")
	}
	if m2.confirmAction == nil {
		t.Fatal("confirmAction should be set")
	}

	// Simulate pressing "y" to confirm.
	_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	result, ok := msg.(apiResultMsg)
	if !ok {
		t.Fatalf("expected apiResultMsg, got %T", msg)
	}
	if result.err == nil {
		t.Fatal("expected non-nil error for 500 response")
	}
	if !strings.Contains(result.err.Error(), "write failed") {
		t.Errorf("error %q should contain 'write failed'", result.err.Error())
	}
}

// --- Policy denial (403) handler-level tests ---

func TestHandleInputKey_NetworkStaticIP_PolicyDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "interface 'eth0' is not allowed for write operations"},
		})
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "192.168.1.10/24"
	m.inputKey = inputKeyNetworkStaticIPPrefix + "eth0"
	m.inputPlugin = "network"

	// Enter should transition to confirmation screen.
	updated, enterCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.screen != screenConfirm {
		t.Fatalf("expected screenConfirm, got %d", m2.screen)
	}
	if enterCmd != nil {
		t.Error("Enter should transition to confirm, not return a cmd")
	}
	if m2.confirmAction == nil {
		t.Fatal("confirmAction should be set")
	}

	// Simulate pressing "y" to confirm.
	_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

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

func TestHandleInputKey_NetworkDNS_PolicyDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "interface 'eth0' is not allowed for write operations"},
		})
	}))
	defer srv.Close()

	m := New(nil)
	m.api = NewAPIClient(srv.URL)
	m.screen = screenInput
	m.inputBuffer = "8.8.8.8"
	m.inputKey = inputKeyNetworkDNS
	m.inputPlugin = "network"

	// Enter should transition to confirmation screen.
	updated, enterCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := updated.(Model)
	if m2.screen != screenConfirm {
		t.Fatalf("expected screenConfirm, got %d", m2.screen)
	}
	if enterCmd != nil {
		t.Error("Enter should transition to confirm, not return a cmd")
	}
	if m2.confirmAction == nil {
		t.Fatal("confirmAction should be set")
	}

	// Simulate pressing "y" to confirm.
	_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

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

// ---------------------------------------------------------------------------
// TEST-2: Generic plugin needsMenuRefresh path
// ---------------------------------------------------------------------------

func TestGenericPluginMenuRefresh(t *testing.T) {
	m := New(nil)
	m.screen = screenDetail
	m.screenTitle = "Firewall Manager"
	m.needsMenuRefresh = true
	m.menuStack = []menuState{{items: []MenuItem{{Title: "Parent"}}, cursor: 0, title: "Firewall Manager"}}
	m.menuItems = []MenuItem{{Title: "Rule 1"}, {Title: "Rule 2"}}

	// Dismiss the detail screen — should trigger a refresh.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m2 := updated.(Model)

	if !m2.loading {
		t.Fatal("loading should be true while menu refresh is in progress")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd to perform menu refresh")
	}

	// Execute the refresh command and apply the result.
	refreshMsg := cmd()
	refresh, ok := refreshMsg.(menuRefreshMsg)
	if !ok {
		t.Fatalf("expected menuRefreshMsg, got %T", refreshMsg)
	}

	updated2, _ := m2.Update(refresh)
	m3 := updated2.(Model)

	// The screen title must remain "Firewall Manager" — it must NOT
	// silently fall back to "Update Manager".
	if m3.screenTitle == "Update Manager" {
		t.Error("screenTitle silently fell back to 'Update Manager' for a generic plugin")
	}
	if m3.screenTitle != "Firewall Manager" {
		t.Errorf("screenTitle = %q, want %q", m3.screenTitle, "Firewall Manager")
	}
}

// ---------------------------------------------------------------------------
// TEST-5: Init() error path — verify no panic
// ---------------------------------------------------------------------------

func TestInit_UnreachableAPI(t *testing.T) {
	m := NewWithAuth(nil, closedTestServer(), "")
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a non-nil Cmd even when API is unreachable")
	}
	// Execute the returned command to exercise the error path and ensure no panic.
	msg := cmd()
	if _, ok := msg.(nodeInfoMsg); !ok {
		t.Errorf("Init() Cmd returned %T, want nodeInfoMsg", msg)
	}
}

// ---------------------------------------------------------------------------
// TEST-7: Max input length cap
// ---------------------------------------------------------------------------

func TestHandleInputKey_MaxInputLen(t *testing.T) {
	m := NewWithAuth(nil, "http://localhost", "")
	m.screen = screenInput
	m.inputBuffer = strings.Repeat("a", 512) // at max

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	model := updated.(Model)
	if utf8.RuneCountInString(model.inputBuffer) != 512 {
		t.Errorf("buffer grew beyond maxInputLen: got %d runes", utf8.RuneCountInString(model.inputBuffer))
	}
	if model.statusMsg != "Input limit reached" {
		t.Errorf("statusMsg = %q, want %q", model.statusMsg, "Input limit reached")
	}
}

// ---------------------------------------------------------------------------
// TEST-8: Nil-action separator not selectable
// ---------------------------------------------------------------------------

func TestHandleMenuKey_SeparatorNotSelectable(t *testing.T) {
	m := NewWithAuth(nil, "http://localhost", "")
	m.screen = screenSub
	m.menuItems = []MenuItem{
		{Title: "──── Actions ────", Action: nil},
	}
	m.cursor = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if model.statusMsg != "Not a selectable item" {
		t.Errorf("statusMsg = %q, want %q", model.statusMsg, "Not a selectable item")
	}
}

// ---------------------------------------------------------------------------
// TEST-9: DNS validation rejects invalid address
// ---------------------------------------------------------------------------

func TestHandleInputKey_NetworkDNS_InvalidAddress(t *testing.T) {
	m := NewWithAuth(nil, "http://localhost", "")
	m.screen = screenInput
	m.inputBuffer = "8.8.8.8, notanip"
	m.inputKey = inputKeyNetworkDNS
	m.inputPlugin = "network"

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model := updated.(Model)
	if !strings.Contains(model.statusMsg, "invalid DNS") {
		t.Errorf("statusMsg = %q, want to contain 'invalid DNS'", model.statusMsg)
	}
}
