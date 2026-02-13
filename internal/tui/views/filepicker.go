package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rendis/geotap/internal/tui/styles"
)

type FilePickerModel struct {
	dir    string
	files  []os.DirEntry
	cursor int
	err    error
}

func NewFilePickerModel() FilePickerModel {
	cwd, _ := os.Getwd()
	m := FilePickerModel{dir: cwd}
	m.loadDir()
	return m
}

func (m *FilePickerModel) loadDir() {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		m.err = err
		return
	}

	m.files = nil
	// Add parent dir
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() || strings.HasSuffix(name, ".db") {
			m.files = append(m.files, e)
		}
	}
	m.cursor = 0
}

func (m FilePickerModel) Init() tea.Cmd {
	return nil
}

func (m FilePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.files) {
				entry := m.files[m.cursor]
				fullPath := filepath.Join(m.dir, entry.Name())
				if entry.IsDir() {
					m.dir = fullPath
					m.loadDir()
					return m, nil
				}
				// Selected a .db file
				return m, func() tea.Msg {
					return NavigateToExplorer{DBPath: fullPath}
				}
			}
		case "backspace":
			parent := filepath.Dir(m.dir)
			if parent != m.dir {
				m.dir = parent
				m.loadDir()
			}
		case "esc":
			return m, func() tea.Msg { return NavigateToHome{} }
		}
	}
	return m, nil
}

func (m FilePickerModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Load Project"))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(styles.Muted).Render(m.dir))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err)))
		return styles.Border.Render(b.String())
	}

	if len(m.files) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Muted).Italic(true).
			Render("No .db files or directories found"))
	}

	// Show max 15 items
	start := 0
	if m.cursor > 12 {
		start = m.cursor - 12
	}
	end := start + 15
	if end > len(m.files) {
		end = len(m.files)
	}

	for i := start; i < end; i++ {
		entry := m.files[i]
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		icon := "  "
		style := styles.InactiveItem
		if entry.IsDir() {
			icon = "📁 "
		} else {
			icon = "💾 "
		}

		if i == m.cursor {
			style = styles.ActiveItem
		}

		b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, icon, style.Render(entry.Name())))
	}

	b.WriteString("\n")
	b.WriteString(styles.StatusBar.Render("enter open • backspace parent dir • esc back"))

	return styles.Border.Render(b.String())
}
