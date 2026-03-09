package tui

import (
	"sort"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// DefaultTheme
// ---------------------------------------------------------------------------

func TestDefaultTheme_NonZero(t *testing.T) {
	th := DefaultTheme()
	if th.Cursor == "" {
		t.Error("DefaultTheme().Cursor must not be empty")
	}
	if th.Separator == "" {
		t.Error("DefaultTheme().Separator must not be empty")
	}
	if th.SepWidth <= 0 {
		t.Errorf("DefaultTheme().SepWidth: got %d, want > 0", th.SepWidth)
	}
	if th.ConnBadgeText == "" {
		t.Error("DefaultTheme().ConnBadgeText must not be empty")
	}
	if th.StandBadgeText == "" {
		t.Error("DefaultTheme().StandBadgeText must not be empty")
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — full override
// ---------------------------------------------------------------------------

func TestThemeFromYAML_FullOverride(t *testing.T) {
	yml := `
colors:
  header:      {fg: "196", bold: true}
  footer:      {fg: "240", faint: true}
  selected:    {fg: "46", bold: true}
  normal:      {fg: "255"}
  description: {fg: "248", faint: true}
  cursor:      {fg: "46"}
  connected:   {fg: "34"}
  standalone:  {fg: "214"}
  confirm_yes: {fg: "34", bold: true}
  confirm_no:  {fg: "196", bold: true}
  status_bar:  {fg: "240"}
  spinner:     {fg: "46"}
glyphs:
  cursor: "> "
  separator: "="
  separator_width: 60
  connected_badge: "[OK]"
  standalone_badge: "[LOCAL]"
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	if th.Cursor != "> " {
		t.Errorf("Cursor: got %q, want %q", th.Cursor, "> ")
	}
	if th.Separator != "=" {
		t.Errorf("Separator: got %q, want %q", th.Separator, "=")
	}
	if th.SepWidth != 60 {
		t.Errorf("SepWidth: got %d, want 60", th.SepWidth)
	}
	if th.ConnBadgeText != "[OK]" {
		t.Errorf("ConnBadgeText: got %q, want %q", th.ConnBadgeText, "[OK]")
	}
	if th.StandBadgeText != "[LOCAL]" {
		t.Errorf("StandBadgeText: got %q, want %q", th.StandBadgeText, "[LOCAL]")
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — partial override (defaults preserved)
// ---------------------------------------------------------------------------

func TestThemeFromYAML_PartialOverride(t *testing.T) {
	yml := `
colors:
  header: {fg: "196"}
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}

	def := DefaultTheme()

	// header fg should be "196", not the default "12"
	got := th.Header.GetForeground()
	defFG := def.Header.GetForeground()
	if got == defFG {
		t.Errorf("expected header fg to differ from default; both are %v", got)
	}

	// cursor glyph should stay default
	if th.Cursor != def.Cursor {
		t.Errorf("Cursor: got %q, want default %q", th.Cursor, def.Cursor)
	}
	if th.SepWidth != def.SepWidth {
		t.Errorf("SepWidth: got %d, want default %d", th.SepWidth, def.SepWidth)
	}
	if th.ConnBadgeText != def.ConnBadgeText {
		t.Errorf("ConnBadgeText: got %q, want default %q", th.ConnBadgeText, def.ConnBadgeText)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — empty YAML returns default
// ---------------------------------------------------------------------------

func TestThemeFromYAML_EmptyReturnsDefault(t *testing.T) {
	th, err := ThemeFromYAML([]byte(""))
	if err != nil {
		t.Fatalf("ThemeFromYAML(empty): %v", err)
	}
	def := DefaultTheme()
	if th.Cursor != def.Cursor {
		t.Errorf("Cursor: got %q, want %q", th.Cursor, def.Cursor)
	}
	if th.SepWidth != def.SepWidth {
		t.Errorf("SepWidth: got %d, want %d", th.SepWidth, def.SepWidth)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — comment-only YAML returns default
// ---------------------------------------------------------------------------

func TestThemeFromYAML_CommentsOnly(t *testing.T) {
	yml := "# just a comment\n# another comment\n"
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	def := DefaultTheme()
	if th.Cursor != def.Cursor {
		t.Errorf("Cursor: got %q, want %q", th.Cursor, def.Cursor)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — invalid YAML returns error
// ---------------------------------------------------------------------------

func TestThemeFromYAML_InvalidYAML(t *testing.T) {
	_, err := ThemeFromYAML([]byte("{{{{not yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "invalid theme YAML") {
		t.Errorf("error message: got %q, want to contain 'invalid theme YAML'", err.Error())
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — only glyphs
// ---------------------------------------------------------------------------

func TestThemeFromYAML_GlyphsOnly(t *testing.T) {
	yml := `
glyphs:
  cursor: "→ "
  separator: "-"
  separator_width: 30
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	if th.Cursor != "→ " {
		t.Errorf("Cursor: got %q, want %q", th.Cursor, "→ ")
	}
	if th.Separator != "-" {
		t.Errorf("Separator: got %q, want %q", th.Separator, "-")
	}
	if th.SepWidth != 30 {
		t.Errorf("SepWidth: got %d, want 30", th.SepWidth)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — only colours, no glyphs
// ---------------------------------------------------------------------------

func TestThemeFromYAML_ColoursOnly(t *testing.T) {
	yml := `
colors:
  spinner: {fg: "196"}
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	def := DefaultTheme()
	// spinner fg should differ from default "14"
	if th.Spinner.GetForeground() == def.Spinner.GetForeground() {
		t.Error("spinner fg should differ from default after override")
	}
	// glyph defaults preserved
	if th.Cursor != def.Cursor {
		t.Errorf("Cursor: got %q, want default %q", th.Cursor, def.Cursor)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — background colour
// ---------------------------------------------------------------------------

func TestThemeFromYAML_BackgroundColour(t *testing.T) {
	yml := `
colors:
  header: {fg: "15", bg: "0"}
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	if th.Header.GetForeground() == DefaultTheme().Header.GetForeground() {
		t.Error("header fg should differ after override to 15")
	}
	bg := th.Header.GetBackground()
	if bg == nil {
		t.Error("header bg should be set after override")
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — bold/faint toggles
// ---------------------------------------------------------------------------

func TestThemeFromYAML_BoldFaintToggles(t *testing.T) {
	yml := `
colors:
  header: {bold: false}
  footer: {faint: false, fg: "15"}
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	// header had bold=true, now false
	if th.Header.GetBold() {
		t.Error("header should not be bold after bold:false override")
	}
	// footer had faint=true, now false
	if th.Footer.GetFaint() {
		t.Error("footer should not be faint after faint:false override")
	}
	// footer fg should be set to "15"
	if th.Footer.GetForeground() == DefaultTheme().Footer.GetForeground() {
		t.Error("footer fg should differ after override to 15")
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — unknown fields are ignored
// ---------------------------------------------------------------------------

func TestThemeFromYAML_UnknownFieldsIgnored(t *testing.T) {
	yml := `
colors:
  header: {fg: "12"}
  unknown_element: {fg: "99"}
extra_section:
  foo: bar
`
	_, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("expected unknown fields to be ignored, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — hex colour strings
// ---------------------------------------------------------------------------

func TestThemeFromYAML_HexColours(t *testing.T) {
	yml := `
colors:
  header: {fg: "#ff5733"}
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	if th.Header.GetForeground() == DefaultTheme().Header.GetForeground() {
		t.Error("header fg with hex colour should differ from default ANSI")
	}
}

// ---------------------------------------------------------------------------
// BuiltinTheme — all built-in themes parse
// ---------------------------------------------------------------------------

func TestBuiltinTheme_AllParse(t *testing.T) {
	names := BuiltinThemeNames()
	if len(names) == 0 {
		t.Fatal("BuiltinThemeNames() returned no names")
	}
	for _, name := range names {
		th, ok := BuiltinTheme(name)
		if !ok {
			t.Errorf("BuiltinTheme(%q) returned ok=false", name)
			continue
		}
		if th.Cursor == "" {
			t.Errorf("BuiltinTheme(%q).Cursor is empty", name)
		}
	}
}

// ---------------------------------------------------------------------------
// BuiltinTheme — known names
// ---------------------------------------------------------------------------

func TestBuiltinTheme_KnownNames(t *testing.T) {
	expected := []string{"default", "high-contrast", "nord", "solarized-dark"}
	names := BuiltinThemeNames()
	sort.Strings(expected)
	sort.Strings(names)
	if len(names) != len(expected) {
		t.Fatalf("BuiltinThemeNames: got %v, want %v", names, expected)
	}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("BuiltinThemeNames[%d]: got %q, want %q", i, n, expected[i])
		}
	}
}

// ---------------------------------------------------------------------------
// BuiltinTheme — unknown name
// ---------------------------------------------------------------------------

func TestBuiltinTheme_UnknownName(t *testing.T) {
	_, ok := BuiltinTheme("does-not-exist")
	if ok {
		t.Error("BuiltinTheme(unknown) should return false")
	}
}

// ---------------------------------------------------------------------------
// BuiltinTheme — case insensitive
// ---------------------------------------------------------------------------

func TestBuiltinTheme_CaseInsensitive(t *testing.T) {
	_, ok := BuiltinTheme("Nord")
	if !ok {
		t.Error("BuiltinTheme should be case-insensitive")
	}
	_, ok = BuiltinTheme("HIGH-CONTRAST")
	if !ok {
		t.Error("BuiltinTheme should be case-insensitive")
	}
}

// ---------------------------------------------------------------------------
// BuiltinTheme("default") matches DefaultTheme()
// ---------------------------------------------------------------------------

func TestBuiltinTheme_DefaultMatchesDefaultTheme(t *testing.T) {
	builtin, ok := BuiltinTheme("default")
	if !ok {
		t.Fatal("BuiltinTheme(default) not found")
	}
	def := DefaultTheme()

	// Compare rendered output for key styles
	tests := []struct {
		name    string
		builtin string
		def     string
	}{
		{"Header", builtin.Header.Render("x"), def.Header.Render("x")},
		{"Selected", builtin.Selected.Render("x"), def.Selected.Render("x")},
		{"Normal", builtin.Normal.Render("x"), def.Normal.Render("x")},
		{"ConfirmYes", builtin.ConfirmYes.Render("x"), def.ConfirmYes.Render("x")},
		{"Spinner", builtin.Spinner.Render("x"), def.Spinner.Render("x")},
	}
	for _, tc := range tests {
		if tc.builtin != tc.def {
			t.Errorf("%s: builtin %q != default %q", tc.name, tc.builtin, tc.def)
		}
	}

	// Glyphs must match
	if builtin.Cursor != def.Cursor {
		t.Errorf("Cursor: %q != %q", builtin.Cursor, def.Cursor)
	}
	if builtin.SepWidth != def.SepWidth {
		t.Errorf("SepWidth: %d != %d", builtin.SepWidth, def.SepWidth)
	}
}

// ---------------------------------------------------------------------------
// SetTheme — model uses replaced theme
// ---------------------------------------------------------------------------

func TestSetTheme_AppliedToModel(t *testing.T) {
	m := New(nil)
	custom := DefaultTheme()
	custom.Cursor = ">> "
	custom.SepWidth = 99

	m.SetTheme(custom)

	if m.theme.Cursor != ">> " {
		t.Errorf("after SetTheme, Cursor: got %q, want %q", m.theme.Cursor, ">> ")
	}
	if m.theme.SepWidth != 99 {
		t.Errorf("after SetTheme, SepWidth: got %d, want 99", m.theme.SepWidth)
	}
}

// ---------------------------------------------------------------------------
// applyStyleDef — nil def returns base unchanged
// ---------------------------------------------------------------------------

func TestApplyStyleDef_NilReturnsBase(t *testing.T) {
	base := DefaultTheme().Header
	result := applyStyleDef(base, nil)
	if result.Render("x") != base.Render("x") {
		t.Error("applyStyleDef(base, nil) should return base unchanged")
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — zero-width separator
// ---------------------------------------------------------------------------

func TestThemeFromYAML_ZeroSepWidth(t *testing.T) {
	yml := `
glyphs:
  separator_width: 0
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	if th.SepWidth != 0 {
		t.Errorf("SepWidth: got %d, want 0", th.SepWidth)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — negative separator_width returns error
// ---------------------------------------------------------------------------

func TestThemeFromYAML_NegativeSepWidth(t *testing.T) {
	yml := `
glyphs:
  separator_width: -1
`
	_, err := ThemeFromYAML([]byte(yml))
	if err == nil {
		t.Fatal("expected error for negative separator_width, got nil")
	}
	if !strings.Contains(err.Error(), "separator_width") {
		t.Errorf("error should mention separator_width: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — empty string glyphs
// ---------------------------------------------------------------------------

func TestThemeFromYAML_EmptyGlyphs(t *testing.T) {
	yml := `
glyphs:
  cursor: ""
  connected_badge: ""
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: %v", err)
	}
	if th.Cursor != "" {
		t.Errorf("Cursor: got %q, want empty", th.Cursor)
	}
	if th.ConnBadgeText != "" {
		t.Errorf("ConnBadgeText: got %q, want empty", th.ConnBadgeText)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — separator_width at maximum boundary (500)
// ---------------------------------------------------------------------------

func TestThemeFromYAML_MaxSepWidth(t *testing.T) {
	yml := `
glyphs:
  separator_width: 500
`
	th, err := ThemeFromYAML([]byte(yml))
	if err != nil {
		t.Fatalf("ThemeFromYAML: separator_width 500 should be accepted: %v", err)
	}
	if th.SepWidth != 500 {
		t.Errorf("SepWidth: got %d, want 500", th.SepWidth)
	}
}

// ---------------------------------------------------------------------------
// ThemeFromYAML — separator_width above maximum returns error
// ---------------------------------------------------------------------------

func TestThemeFromYAML_OverMaxSepWidth(t *testing.T) {
	yml := `
glyphs:
  separator_width: 501
`
	_, err := ThemeFromYAML([]byte(yml))
	if err == nil {
		t.Fatal("expected error for separator_width > 500, got nil")
	}
	if !strings.Contains(err.Error(), "separator_width") {
		t.Errorf("error should mention separator_width: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TEST-3: ThemeFromYAML with untrusted glyph content (ANSI sequences)
// ---------------------------------------------------------------------------

func TestThemeFromYAML_SanitizesGlyphs(t *testing.T) {
	// Use Unicode BiDi override chars (Bidi_Control property) — YAML allows these,
	// and sanitizeText should strip them after the SEC-1/SEC-4 fix.
	yml := []byte("glyphs:\n  cursor: \">\u202E\"\n  connected_badge: \"OK\u202D\"\n  standalone_badge: \"OFF\u2066\"")
	theme, err := ThemeFromYAML(yml)
	if err != nil {
		t.Fatalf("ThemeFromYAML failed: %v", err)
	}
	if strings.Contains(theme.Cursor, "\u202E") {
		t.Error("cursor glyph should not contain BiDi override characters")
	}
	if strings.Contains(theme.ConnBadgeText, "\u202D") {
		t.Error("connected_badge glyph should not contain BiDi override characters")
	}
	if strings.Contains(theme.StandBadgeText, "\u2066") {
		t.Error("standalone_badge glyph should not contain BiDi override characters")
	}
}

// ---------------------------------------------------------------------------
// TEST-7: BuiltinTheme false/unknown case
// ---------------------------------------------------------------------------

func TestBuiltinTheme_Unknown(t *testing.T) {
	_, ok := BuiltinTheme("nonexistent")
	if ok {
		t.Error("BuiltinTheme should return false for unknown theme name")
	}
}
