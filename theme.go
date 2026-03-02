package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

// Theme holds all visual styles used by the TUI. Create one with
// DefaultTheme(), or parse from YAML with ThemeFromYAML().
type Theme struct {
	// Header is the style for the top title bar.
	Header lipgloss.Style
	// Footer is the style for footer help text.
	Footer lipgloss.Style
	// Selected is the style for the currently highlighted menu item.
	Selected lipgloss.Style
	// Normal is the style for non-selected menu items.
	Normal lipgloss.Style
	// Description is the style for item descriptions below titles.
	Description lipgloss.Style

	// Cursor is the string shown before the selected item (e.g. "▸").
	Cursor string
	// CursorStyle is the lipgloss style applied to the cursor glyph.
	CursorStyle lipgloss.Style
	// Separator is the repeating character for horizontal rules (e.g. "─").
	Separator string
	// SepWidth is the number of times Separator is repeated.
	SepWidth int

	// ConnBadgeText is the label shown when connected to a service.
	ConnBadgeText string
	// ConnBadgeStyle is the style for the connected badge.
	ConnBadgeStyle lipgloss.Style
	// StandBadgeText is the label shown in standalone mode.
	StandBadgeText string
	// StandBadgeStyle is the style for the standalone badge.
	StandBadgeStyle lipgloss.Style

	// ConfirmYes is the style for the [Y] Yes button in confirmation dialogs.
	ConfirmYes lipgloss.Style
	// ConfirmNo is the style for the [N] No button in confirmation dialogs.
	ConfirmNo lipgloss.Style

	// StatusBar is the style for the hostname/uptime bar in the footer.
	StatusBar lipgloss.Style

	// Spinner is the style for progress spinners.
	Spinner lipgloss.Style
}

// DefaultTheme returns the built-in colour scheme matching the original
// hardcoded styles.
func DefaultTheme() Theme {
	return Theme{
		Header:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		Footer:      lipgloss.NewStyle().Faint(true),
		Selected:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")),
		Normal:      lipgloss.NewStyle().Foreground(lipgloss.Color("7")),
		Description: lipgloss.NewStyle().Faint(true),

		Cursor:      "▸ ",
		CursorStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
		Separator:   "─",
		SepWidth:    40,

		ConnBadgeText:   "● connected",
		ConnBadgeStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		StandBadgeText:  "● standalone",
		StandBadgeStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("11")),

		ConfirmYes: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),
		ConfirmNo:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),

		StatusBar: lipgloss.NewStyle().Faint(true),

		Spinner: lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
	}
}

// SetTheme replaces the model's active theme. Call before Run() to apply a
// custom or built-in theme.
func (m *Model) SetTheme(t Theme) {
	m.theme = t
}

// styleDef is the YAML-serialisable representation of a lipgloss.Style.
// All fields are pointers so we can detect which ones the user specified;
// unset fields inherit from the base (default) theme.
type styleDef struct {
	Fg    *string `yaml:"fg,omitempty"`
	Bg    *string `yaml:"bg,omitempty"`
	Bold  *bool   `yaml:"bold,omitempty"`
	Faint *bool   `yaml:"faint,omitempty"`
}

// themeYAML is the top-level YAML document shape for a theme file.
type themeYAML struct {
	Colors *themeColors `yaml:"colors,omitempty"`
	Glyphs *themeGlyphs `yaml:"glyphs,omitempty"`
}

// themeColors maps each styleable element to its colour/attribute definition.
type themeColors struct {
	Header      *styleDef `yaml:"header,omitempty"`
	Footer      *styleDef `yaml:"footer,omitempty"`
	Selected    *styleDef `yaml:"selected,omitempty"`
	Normal      *styleDef `yaml:"normal,omitempty"`
	Description *styleDef `yaml:"description,omitempty"`
	Cursor      *styleDef `yaml:"cursor,omitempty"`
	Connected   *styleDef `yaml:"connected,omitempty"`
	Standalone  *styleDef `yaml:"standalone,omitempty"`
	ConfirmYes  *styleDef `yaml:"confirm_yes,omitempty"`
	ConfirmNo   *styleDef `yaml:"confirm_no,omitempty"`
	StatusBar   *styleDef `yaml:"status_bar,omitempty"`
	Spinner     *styleDef `yaml:"spinner,omitempty"`
}

// themeGlyphs holds customisable text/glyphs for the TUI chrome.
type themeGlyphs struct {
	Cursor         *string `yaml:"cursor,omitempty"`
	Separator      *string `yaml:"separator,omitempty"`
	SepWidth       *int    `yaml:"separator_width,omitempty"`
	ConnBadgeText  *string `yaml:"connected_badge,omitempty"`
	StandBadgeText *string `yaml:"standalone_badge,omitempty"`
}

// applyStyleDef merges a styleDef into a base lipgloss.Style. Only non-nil
// fields are applied; the rest keep their base values.
func applyStyleDef(base lipgloss.Style, def *styleDef) lipgloss.Style {
	if def == nil {
		return base
	}
	s := base
	if def.Fg != nil {
		s = s.Foreground(lipgloss.Color(*def.Fg))
	}
	if def.Bg != nil {
		s = s.Background(lipgloss.Color(*def.Bg))
	}
	if def.Bold != nil {
		s = s.Bold(*def.Bold)
	}
	if def.Faint != nil {
		s = s.Faint(*def.Faint)
	}
	return s
}

// ThemeFromYAML parses YAML bytes into a Theme, using DefaultTheme() as
// the base. Only fields present in the YAML are overridden; all others keep
// their default values. Returns an error if the YAML is malformed.
func ThemeFromYAML(data []byte) (Theme, error) {
	var raw themeYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Theme{}, fmt.Errorf("invalid theme YAML: %w", err)
	}

	t := DefaultTheme()

	if c := raw.Colors; c != nil {
		t.Header = applyStyleDef(t.Header, c.Header)
		t.Footer = applyStyleDef(t.Footer, c.Footer)
		t.Selected = applyStyleDef(t.Selected, c.Selected)
		t.Normal = applyStyleDef(t.Normal, c.Normal)
		t.Description = applyStyleDef(t.Description, c.Description)
		t.CursorStyle = applyStyleDef(t.CursorStyle, c.Cursor)
		t.ConnBadgeStyle = applyStyleDef(t.ConnBadgeStyle, c.Connected)
		t.StandBadgeStyle = applyStyleDef(t.StandBadgeStyle, c.Standalone)
		t.ConfirmYes = applyStyleDef(t.ConfirmYes, c.ConfirmYes)
		t.ConfirmNo = applyStyleDef(t.ConfirmNo, c.ConfirmNo)
		t.StatusBar = applyStyleDef(t.StatusBar, c.StatusBar)
		t.Spinner = applyStyleDef(t.Spinner, c.Spinner)
	}

	if g := raw.Glyphs; g != nil {
		if g.Cursor != nil {
			t.Cursor = *g.Cursor
		}
		if g.Separator != nil {
			t.Separator = *g.Separator
		}
		if g.SepWidth != nil {
			if *g.SepWidth < 0 || *g.SepWidth > 500 {
				return Theme{}, fmt.Errorf("invalid theme: separator_width must be 0–500, got %d", *g.SepWidth)
			}
			t.SepWidth = *g.SepWidth
		}
		if g.ConnBadgeText != nil {
			t.ConnBadgeText = *g.ConnBadgeText
		}
		if g.StandBadgeText != nil {
			t.StandBadgeText = *g.StandBadgeText
		}
	}

	return t, nil
}

// builtinThemes maps theme names to their YAML definitions. All built-in
// themes are compiled into the binary — no filesystem access needed.
var builtinThemes = map[string]string{
	"default": `# Default theme — matches the original hardcoded colours.
colors:
  header:      {fg: "12", bold: true}
  footer:      {faint: true}
  selected:    {fg: "14", bold: true}
  normal:      {fg: "7"}
  description: {faint: true}
  cursor:      {fg: "14"}
  connected:   {fg: "10"}
  standalone:  {fg: "11"}
  confirm_yes: {fg: "10", bold: true}
  confirm_no:  {fg: "9", bold: true}
  status_bar:  {faint: true}
  spinner:     {fg: "14"}
`,

	"high-contrast": `# High-contrast theme for maximum readability / accessibility.
colors:
  header:      {fg: "15", bg: "0", bold: true}
  footer:      {fg: "15"}
  selected:    {fg: "0", bg: "15", bold: true}
  normal:      {fg: "15"}
  description: {fg: "248"}
  cursor:      {fg: "15"}
  connected:   {fg: "46", bold: true}
  standalone:  {fg: "226", bold: true}
  confirm_yes: {fg: "46", bold: true}
  confirm_no:  {fg: "196", bold: true}
  status_bar:  {fg: "248"}
  spinner:     {fg: "15"}
`,

	"nord": `# Nord colour palette — muted arctic tones.
colors:
  header:      {fg: "111", bold: true}
  footer:      {fg: "60", faint: true}
  selected:    {fg: "152", bold: true}
  normal:      {fg: "252"}
  description: {fg: "60", faint: true}
  cursor:      {fg: "152"}
  connected:   {fg: "108"}
  standalone:  {fg: "222"}
  confirm_yes: {fg: "108", bold: true}
  confirm_no:  {fg: "174", bold: true}
  status_bar:  {fg: "60", faint: true}
  spinner:     {fg: "111"}
`,

	"solarized-dark": `# Solarized Dark palette by Ethan Schoonover.
colors:
  header:      {fg: "33", bold: true}
  footer:      {fg: "240", faint: true}
  selected:    {fg: "37", bold: true}
  normal:      {fg: "244"}
  description: {fg: "240", faint: true}
  cursor:      {fg: "37"}
  connected:   {fg: "64"}
  standalone:  {fg: "136"}
  confirm_yes: {fg: "64", bold: true}
  confirm_no:  {fg: "160", bold: true}
  status_bar:  {fg: "240", faint: true}
  spinner:     {fg: "33"}
`,
}

// BuiltinTheme returns a Theme for the given built-in name. The second
// return value is false if the name is not recognised.
func BuiltinTheme(name string) (Theme, bool) {
	raw, ok := builtinThemes[strings.ToLower(name)]
	if !ok {
		return Theme{}, false
	}
	t, err := ThemeFromYAML([]byte(raw))
	if err != nil {
		// Built-in YAML is compiled in and must always parse.
		panic(fmt.Sprintf("built-in theme %q has invalid YAML: %v", name, err))
	}
	return t, true
}

// BuiltinThemeNames returns the sorted list of available built-in theme names.
func BuiltinThemeNames() []string {
	names := make([]string, 0, len(builtinThemes))
	for n := range builtinThemes {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
