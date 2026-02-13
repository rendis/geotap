package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Primary   = lipgloss.Color("#7C3AED") // violet
	Secondary = lipgloss.Color("#06B6D4") // cyan
	Success   = lipgloss.Color("#22C55E") // green
	Warning   = lipgloss.Color("#F59E0B") // amber
	Error     = lipgloss.Color("#EF4444") // red
	Muted     = lipgloss.Color("#6B7280") // gray
	Text      = lipgloss.Color("#E5E7EB") // light gray
	BgDark    = lipgloss.Color("#111827") // dark bg

	// Component styles
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		MarginBottom(1)

	Subtitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Secondary)

	Label = lipgloss.NewStyle().
		Foreground(Muted).
		Width(14)

	Value = lipgloss.NewStyle().
		Foreground(Text)

	ActiveItem = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)

	InactiveItem = lipgloss.NewStyle().
			Foreground(Muted)

	StatusBar = lipgloss.NewStyle().
			Foreground(Muted).
			MarginTop(1)

	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Muted).
		Padding(1, 2)

	FocusedBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	ErrorText = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)
)
