# Theming Guide

Config Manager TUI supports custom colour themes via YAML.

## Quick Start

Use a built-in theme by setting `theme:` in your `config.yaml`:

```yaml
theme: nord
```

Available built-in themes: `default`, `high-contrast`, `nord`, `solarized-dark`.

Or point to a custom YAML file:

```yaml
theme: /home/pi/.config/cm/my-theme.yaml
```

## Writing a Custom Theme

A theme file has two optional sections — `colors` and `glyphs`. Only include
the fields you want to change; everything else inherits from the default theme.

### Minimal Example

```yaml
# Just change the header colour.
colors:
  header: {fg: "196", bold: true}
```

### Full Reference

```yaml
colors:
  header:      {fg: "12", bold: true}       # Title bar
  footer:      {faint: true}                # Footer help text
  selected:    {fg: "14", bold: true}       # Highlighted menu item
  normal:      {fg: "7"}                    # Non-selected items
  description: {faint: true}                # Item descriptions
  cursor:      {fg: "14"}                   # Cursor glyph (▸)
  connected:   {fg: "10"}                   # Connected badge
  standalone:  {fg: "11"}                   # Standalone badge
  confirm_yes: {fg: "10", bold: true}       # [Y] button
  confirm_no:  {fg: "9", bold: true}        # [N] button
  status_bar:  {faint: true}                # Hostname/uptime bar
  spinner:     {fg: "14"}                   # Progress spinner

glyphs:
  cursor: "▸ "                              # Prefix for selected item
  separator: "─"                            # Horizontal rule character
  separator_width: 40                       # Rule width in characters (0–500, error if out of range)
  connected_badge: "● connected"            # Label in connected mode
  standalone_badge: "● standalone"          # Label in standalone mode
```

`separator_width` must be between 0 and 500 (inclusive). Values outside this
range cause a parse error.

### Style Properties

Each colour entry supports:

| Property | Type | Description |
| --- | --- | --- |
| `fg` | string | Foreground colour — ANSI 256 (`"14"`) or hex (`"#5e81ac"`) |
| `bg` | string | Background colour — same format |
| `bold` | bool | Bold text |
| `faint` | bool | Dimmed text |

## Built-in Themes

### default

The original hardcoded colour scheme. Bright blue header, cyan selection,
green/yellow badges. Works well on most terminal emulators.

### high-contrast

Designed for maximum readability and accessibility. White on black with bright
green/red confirmation buttons. Good for users with vision impairments or
low-contrast displays.

### nord

Muted arctic tones from the [Nord](https://www.nordtheme.com/) colour palette.
Soft blues and greens that are easy on the eyes for extended use.

### solarized-dark

Based on [Solarized Dark](https://ethanschoonover.com/solarized/) by Ethan
Schoonover. Precise colour relationships designed for readability on dark
backgrounds.

## Colour Reference

Colour values can be:

- **ANSI 256** — a number from `"0"` to `"255"` (e.g., `"14"` for cyan)
- **Hex** — a `#rrggbb` string (e.g., `"#5e81ac"` for Nord blue)

For an ANSI 256 colour chart, see
<https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit>.

## Programmatic Usage

```go
import "github.com/msutara/config-manager-tui"

// Built-in theme by name.
theme, ok := tui.BuiltinTheme("nord")

// Custom theme from YAML bytes.
theme, err := tui.ThemeFromYAML(yamlBytes)

// Apply before Run().
model := tui.New(plugins)
model.SetTheme(theme)
```
