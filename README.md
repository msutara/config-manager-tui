# config-manager-tui

Terminal user interface for
[Config Manager](https://github.com/msutara/config-manager-core). Provides a
raspi-config style interactive menu built with Bubble Tea for headless
Debian-based nodes.

## Features

- Raspi-config style menu navigation (arrow keys, Enter, q to quit)
- Dynamic plugin menu discovery from the core plugin registry
- System info display (hostname, OS, kernel, arch, uptime)
- Plugin-specific submenus with back-navigation (esc/q/backspace)
- Confirmation dialogs for destructive actions (updates, POST endpoints)
- Status bar showing hostname and uptime in the footer
- Theme system with centralised colours, glyphs, and badges (`DefaultTheme`)

## Documentation

- [Usage Guide](docs/USAGE.md) — navigation and key bindings
- [Specification](specs/SPEC.md) — TUI specification and menu structure

## Development

```bash
# lint
golangci-lint run

# test
go test ./...
```

CI runs automatically on push/PR to `main` via GitHub Actions
(`.github/workflows/ci.yml`).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

The TUI applies multiple layers of defense when handling untrusted data from
the Config Manager API and plugin registry:

- **Input sanitization** — All API response text is passed through
  `sanitizeText` (or `sanitizeBody` for multi-line output) before terminal
  rendering. These helpers strip C0 control characters (U+0000–U+001F,
  U+007F), Unicode C1 control codes (U+0080–U+009F), and ANSI escape
  sequences, preventing terminal injection attacks.
- **Path validation** — Every API path built from user or plugin data is
  validated by `validateAPIPath`, which URL-decodes the path first and then
  checks for directory traversal sequences (including percent-encoded
  variants such as `%2e%2e`). `cleanPluginPath` further canonicalises
  plugin endpoint paths and verifies they stay under the expected route
  prefix.
- **Plugin name validation** — Plugin identifiers are matched against
  `validPluginName` (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`), rejecting any
  names that could be used to construct malicious API paths.
- **Route prefix validation** — Plugin route prefixes received from the
  registry are decoded and checked for traversal sequences and control
  characters as defense-in-depth against a compromised registry.

For vulnerability reporting see [SECURITY.md](SECURITY.md).

## License

See [LICENSE](LICENSE) for details.
