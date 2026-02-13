package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rendis/geotap/internal/tui/styles"
)

type RecentEntry struct {
	Path     string
	OpenedAt time.Time
}

type RecentModel struct {
	entries []RecentEntry
	cursor  int
}

func NewRecentModel(entries []RecentEntry) RecentModel {
	return RecentModel{entries: entries}
}

func (m RecentModel) Init() tea.Cmd {
	return nil
}

func (m RecentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.entries) {
				return m, func() tea.Msg {
					return NavigateToExplorer{DBPath: m.entries[m.cursor].Path}
				}
			}
		case "esc":
			return m, func() tea.Msg { return NavigateToHome{} }
		}
	}
	return m, nil
}

func (m RecentModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Recent Projects"))
	b.WriteString("\n\n")

	if len(m.entries) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Muted).Italic(true).
			Render("No recent projects"))
		b.WriteString("\n\n")
		b.WriteString(styles.StatusBar.Render("esc back"))
		return styles.Border.Render(b.String())
	}

	for i, entry := range m.entries {
		cursor := "  "
		style := styles.InactiveItem
		if i == m.cursor {
			cursor = "> "
			style = styles.ActiveItem
		}

		name := filepath.Base(entry.Path)
		dir := filepath.Dir(entry.Path)

		exists := true
		if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
			exists = false
		}

		nameStr := style.Render(name)
		if !exists {
			nameStr = lipgloss.NewStyle().Foreground(styles.Error).Strikethrough(true).Render(name)
		}

		ago := timeAgo(entry.OpenedAt)
		dirStr := lipgloss.NewStyle().Foreground(styles.Muted).Render(
			fmt.Sprintf("  %s  %s", dir, ago))

		b.WriteString(fmt.Sprintf("%s%s\n%s\n", cursor, nameStr, dirStr))
	}

	b.WriteString("\n")
	b.WriteString(styles.StatusBar.Render("enter open • esc back"))

	return styles.Border.Render(b.String())
}

func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// NavigateToRecent signals navigation to recent projects view.
type NavigateToRecent struct{}
