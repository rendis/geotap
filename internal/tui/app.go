package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rendis/map_scrapper/internal/tui/views"
)

type viewID int

const (
	viewHome viewID = iota
	viewSearch
	viewProgress
	viewExplorer
	viewFilePicker
	viewRecent
)

// App is the root bubbletea model.
type App struct {
	currentView viewID
	width       int
	height      int
	home        views.HomeModel
	search      views.SearchModel
	progress    views.ProgressModel
	explorer    views.ExplorerModel
	filePicker  views.FilePickerModel
	recent      views.RecentModel
}

func NewApp() App {
	return App{
		currentView: viewHome,
		home:        views.NewHomeModel(),
		search:      views.NewSearchModel(),
	}
}

func (a App) Init() tea.Cmd {
	return a.home.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" && a.currentView != viewProgress {
			return a, tea.Quit
		}
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
	case views.NavigateToSearch:
		a.currentView = viewSearch
		a.search = views.NewSearchModel()
		return a, a.search.Init()
	case views.NavigateToHome:
		a.currentView = viewHome
		return a, nil
	case views.NavigateToLoad:
		a.currentView = viewFilePicker
		a.filePicker = views.NewFilePickerModel()
		return a, a.filePicker.Init()
	case views.StartScanMsg:
		a.currentView = viewProgress
		a.progress = views.NewProgressModel(msg)
		return a, tea.Batch(a.progress.Init(), a.sizeCmd())
	case views.NavigateToExplorer:
		a.currentView = viewExplorer
		a.explorer = views.NewExplorerModel(msg.DBPath)
		SaveRecent(msg.DBPath)
		return a, tea.Batch(a.explorer.Init(), a.sizeCmd())
	case views.NavigateToRecent:
		a.currentView = viewRecent
		entries := LoadRecent()
		var recentEntries []views.RecentEntry
		for _, e := range entries {
			recentEntries = append(recentEntries, views.RecentEntry{
				Path:     e.Path,
				OpenedAt: e.OpenedAt,
			})
		}
		a.recent = views.NewRecentModel(recentEntries)
		return a, a.recent.Init()
	}

	var cmd tea.Cmd
	switch a.currentView {
	case viewHome:
		var m tea.Model
		m, cmd = a.home.Update(msg)
		a.home = m.(views.HomeModel)
	case viewSearch:
		var m tea.Model
		m, cmd = a.search.Update(msg)
		a.search = m.(views.SearchModel)
	case viewProgress:
		var m tea.Model
		m, cmd = a.progress.Update(msg)
		a.progress = m.(views.ProgressModel)
	case viewExplorer:
		var m tea.Model
		m, cmd = a.explorer.Update(msg)
		a.explorer = m.(views.ExplorerModel)
	case viewFilePicker:
		var m tea.Model
		m, cmd = a.filePicker.Update(msg)
		a.filePicker = m.(views.FilePickerModel)
	case viewRecent:
		var m tea.Model
		m, cmd = a.recent.Update(msg)
		a.recent = m.(views.RecentModel)
	}

	return a, cmd
}

func (a App) View() string {
	var content string
	switch a.currentView {
	case viewHome:
		content = a.home.View()
	case viewSearch:
		content = a.search.View()
	case viewProgress:
		content = a.progress.View()
	case viewExplorer:
		content = a.explorer.View()
	case viewFilePicker:
		content = a.filePicker.View()
	case viewRecent:
		content = a.recent.View()
	}

	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Top,
		content,
	)
}

// sizeCmd sends a WindowSizeMsg so newly created views get the current terminal size.
func (a App) sizeCmd() tea.Cmd {
	w, h := a.width, a.height
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: w, Height: h}
	}
}

// Run starts the TUI.
func Run() error {
	p := tea.NewProgram(NewApp(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
