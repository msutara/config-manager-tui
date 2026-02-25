# config-manager-tui

Terminal user interface for
[Config Manager](https://github.com/msutara/config-manager-core). Provides a
raspi-config style interactive menu built with Bubble Tea for headless
Debian-based nodes.

## Features

- Raspi-config style menu navigation (arrow keys, Enter, q to quit)
- Dynamic plugin menu discovery from the core plugin registry (planned)
- System info display (planned)
- Plugin-specific submenus (planned)

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

See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## License

See [LICENSE](LICENSE) for details.
