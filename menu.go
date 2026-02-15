package tui

// MenuItem represents a single entry in a TUI menu.
type MenuItem struct {
	Title       string
	Description string
	Action      func()
}

// MainMenu returns the top-level menu items. In the future this will be
// populated dynamically from the core plugin registry.
func MainMenu() []MenuItem {
	return []MenuItem{
		{
			Title:       "System Info",
			Description: "View system information and status",
		},
		{
			Title:       "Plugins",
			Description: "Manage installed plugins",
		},
		{
			Title:       "Quit",
			Description: "Exit Config Manager",
		},
	}
}
