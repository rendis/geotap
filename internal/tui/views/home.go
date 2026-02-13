package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rendis/geotap/internal/tui/styles"
)

type menuItem struct {
	key   string
	label string
	desc  string
}

type HomeModel struct {
	items    []menuItem
	cursor   int
	quitting bool
}

func NewHomeModel() HomeModel {
	return HomeModel{
		items: []menuItem{
			{key: "n", label: "New Search", desc: "Start a new geographic scan"},
			{key: "l", label: "Load Project", desc: "Open an existing .db file"},
			{key: "r", label: "Recent Projects", desc: "View recent scan results"},
			{key: "q", label: "Quit", desc: "Exit geotap"},
		},
	}
}

func (m HomeModel) Init() tea.Cmd {
	return nil
}

func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			return m, m.handleSelect()
		case "n":
			m.cursor = 0
			return m, m.handleSelect()
		case "l":
			m.cursor = 1
			return m, m.handleSelect()
		case "r":
			m.cursor = 2
			return m, m.handleSelect()
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m HomeModel) handleSelect() tea.Cmd {
	switch m.cursor {
	case 0: // New Search
		return func() tea.Msg {
			return NavigateToSearch{}
		}
	case 1: // Load Project
		return func() tea.Msg {
			return NavigateToLoad{}
		}
	case 2: // Recent
		return func() tea.Msg {
			return NavigateToRecent{}
		}
	case 3: // Quit
		return tea.Quit
	}
	return nil
}

func (m HomeModel) View() string {
	var b strings.Builder

	// Logo
	logo := lipgloss.NewStyle().
		Foreground(styles.Primary).
		Bold(true).
		Render("  geotap")

	version := lipgloss.NewStyle().
		Foreground(styles.Muted).
		Render(" v0.1.0")

	tagline := lipgloss.NewStyle().
		Foreground(styles.Secondary).
		Italic(true).
		Render("  Geographic data scanner")

	b.WriteString(logo + version + "\n")
	b.WriteString(tagline + "\n\n")

	// Menu items
	for i, item := range m.items {
		cursor := "  "
		style := styles.InactiveItem
		if i == m.cursor {
			cursor = "> "
			style = styles.ActiveItem
		}

		key := lipgloss.NewStyle().
			Foreground(styles.Secondary).
			Bold(true).
			Render(fmt.Sprintf("[%s]", item.key))

		label := style.Render(item.label)
		desc := lipgloss.NewStyle().
			Foreground(styles.Muted).
			Render(" - " + item.desc)

		b.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, key, label, desc))
	}

	b.WriteString("\n")
	b.WriteString(styles.StatusBar.Render("↑↓ navigate • enter select • q quit"))

	return styles.Border.Render(b.String())
}

// Navigation messages
type NavigateToSearch struct{}
type NavigateToLoad struct{}
