package views

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paulmach/orb"
	"github.com/rendis/map_scrapper/internal/engine/geo"
	"github.com/rendis/map_scrapper/internal/engine/scraper"
	"github.com/rendis/map_scrapper/internal/engine/storage"
	"github.com/rendis/map_scrapper/internal/model"
	"github.com/rendis/map_scrapper/internal/tui/styles"
)

// sharedState holds data shared between the scraper goroutine and TUI.
// Lives behind a pointer so it survives bubbletea's value copies.
type sharedState struct {
	mu         sync.Mutex
	stats      *scraper.Stats
	cancel     context.CancelFunc
	numSectors int
}

// ProgressModel manages the scraping progress view.
type ProgressModel struct {
	params    model.SearchParams
	progress  progress.Model
	startTime time.Time
	done        bool
	confirmQuit bool
	err         error
	dbPath      string
	logPath   string
	width     int
	height    int
	shared    *sharedState
}

// Messages
type progressTickMsg time.Time

type scrapeCompleteMsg struct {
	Err error
}

func NewProgressModel(msg StartScanMsg) ProgressModel {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
	)

	m := ProgressModel{
		progress:  p,
		startTime: time.Now(),
		shared:    &sharedState{},
	}

	// Parse params
	m.params.Queries = strings.Split(msg.Query, ",")
	for i := range m.params.Queries {
		m.params.Queries[i] = strings.TrimSpace(m.params.Queries[i])
	}

	if msg.Mode == modeCountry {
		m.params.Country = msg.Country
		m.params.Region = msg.Region
	} else {
		m.params.Lat, _ = strconv.ParseFloat(msg.Lat, 64)
		m.params.Lng, _ = strconv.ParseFloat(msg.Lng, 64)
		m.params.Radius, _ = strconv.ParseFloat(msg.Radius, 64)
	}

	m.params.Zoom, _ = strconv.Atoi(msg.Zoom)
	if m.params.Zoom == 0 {
		if m.params.IsCoordMode() {
			m.params.Zoom = 13
		} else {
			m.params.Zoom = 10
		}
	}
	m.params.Concurrency, _ = strconv.Atoi(msg.Concurrency)
	if m.params.Concurrency <= 0 {
		m.params.Concurrency = 50
	}
	m.params.MaxPages = 1
	m.params.Lang = "en"

	// Setup output paths
	ts := time.Now().Format("20060102_150405")
	baseName := fmt.Sprintf("geotap_%s", ts)
	outDir := msg.Output
	os.MkdirAll(outDir, 0755)
	m.dbPath = filepath.Join(outDir, baseName+".db")
	m.logPath = filepath.Join(outDir, baseName+".log")
	m.params.DBPath = m.dbPath

	return m
}

func (m ProgressModel) Init() tea.Cmd {
	return tea.Batch(
		m.startScraping(),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return progressTickMsg(t)
	})
}

func (m ProgressModel) startScraping() tea.Cmd {
	shared := m.shared
	params := m.params
	dbPath := m.dbPath
	logPath := m.logPath

	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())

		// Generate sectors
		var sectors []model.Sector

		// Load country polygon for geo-filtering
		var poly orb.MultiPolygon
		var err error
		if params.Country != "" {
			var bs *geo.BoundaryStore
			bs, err = geo.NewBoundaryStore()
			if err != nil {
				cancel()
				return scrapeCompleteMsg{Err: err}
			}
			poly, err = bs.GetCountryPolygon(params.Country)
			if err != nil {
				cancel()
				return scrapeCompleteMsg{Err: err}
			}
		}

		if params.IsCoordMode() {
			sectors = geo.GenerateRadiusGrid(params.Lat, params.Lng, params.Radius, params.Zoom)
			if poly != nil {
				sectors = geo.FilterLandSectors(sectors, poly)
			}
		} else {
			var minLat, minLng, maxLat, maxLng float64
			if params.Region != "" {
				minLat, minLng, maxLat, maxLng, err = geo.GeocodeRegion(params.Region, params.Country)
				if err != nil {
					cancel()
					return scrapeCompleteMsg{Err: fmt.Errorf("geocoding region %q: %w", params.Region, err)}
				}
			} else {
				var bs *geo.BoundaryStore
				bs, err = geo.NewBoundaryStore()
				if err != nil {
					cancel()
					return scrapeCompleteMsg{Err: err}
				}
				minLat, minLng, maxLat, maxLng, err = bs.GetCountryBounds(params.Country)
				if err != nil {
					cancel()
					return scrapeCompleteMsg{Err: err}
				}
			}
			allSectors := geo.GenerateGrid(minLat, minLng, maxLat, maxLng, params.Zoom)
			if poly != nil {
				sectors = geo.FilterLandSectors(allSectors, poly)
			} else {
				sectors = allSectors
			}
		}

		// Open storage
		store, err := storage.NewStore(dbPath)
		if err != nil {
			cancel()
			return scrapeCompleteMsg{Err: err}
		}

		// Setup logger
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			store.Close()
			cancel()
			return scrapeCompleteMsg{Err: err}
		}
		logger := log.New(logFile, "", log.LstdFlags)

		numSectors := len(sectors) * len(params.Queries)
		stats := &scraper.Stats{SectorsTotal: numSectors}

		// Store into shared state (survives bubbletea value copies)
		shared.mu.Lock()
		shared.stats = stats
		shared.cancel = cancel
		shared.numSectors = numSectors
		shared.mu.Unlock()

		_, runErr := scraper.Run(ctx, sectors, params, store, logger, &scraper.RunOptions{
			SuppressStderr: true,
			Stats:          stats,
			GeoFilter:      poly,
		})

		logFile.Close()
		store.Close()

		return scrapeCompleteMsg{Err: runErr}
	}
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if cancel := m.shared.getCancel(); cancel != nil {
				cancel()
			}
			return m, tea.Quit
		case "esc":
			if m.done {
				return m, func() tea.Msg {
					return NavigateToExplorer{DBPath: m.dbPath}
				}
			}
			if m.confirmQuit {
				// Second esc: cancel and go home
				if cancel := m.shared.getCancel(); cancel != nil {
					cancel()
				}
				return m, func() tea.Msg { return NavigateToHome{} }
			}
			// First esc: show confirmation
			m.confirmQuit = true
			return m, nil
		case "enter":
			if m.done {
				return m, func() tea.Msg {
					return NavigateToExplorer{DBPath: m.dbPath}
				}
			}
			if m.confirmQuit {
				m.confirmQuit = false
				return m, nil
			}
		}
		// Any other key cancels the confirmation
		if m.confirmQuit {
			m.confirmQuit = false
		}
	case progressTickMsg:
		if m.done {
			return m, nil
		}
		return m, tickCmd()
	case scrapeCompleteMsg:
		m.done = true
		m.err = msg.Err
		return m, nil
	}

	var cmd tea.Cmd
	var pModel tea.Model
	pModel, cmd = m.progress.Update(msg)
	m.progress = pModel.(progress.Model)
	return m, cmd
}

func (m ProgressModel) View() string {
	var b strings.Builder

	query := strings.Join(m.params.Queries, ", ")
	location := m.params.Country
	if location == "" {
		location = fmt.Sprintf("%.4f, %.4f", m.params.Lat, m.params.Lng)
	}
	b.WriteString(styles.Title.Render(fmt.Sprintf("Scraping: %q in %s", query, location)))
	b.WriteString("\n\n")

	// Stats
	statsContent := m.renderStats()
	statsBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Muted).
		Padding(0, 1).
		Width(30).
		Render(statsContent)
	b.WriteString(statsBox)
	b.WriteString("\n\n")

	// Progress bar
	stats := m.shared.getStats()
	var pct float64
	if stats != nil && stats.SectorsTotal > 0 {
		pct = float64(stats.SectorsDone.Load()) / float64(stats.SectorsTotal)
	}
	b.WriteString(m.progress.ViewAs(pct))
	b.WriteString("\n\n")

	// Status
	if m.done {
		if m.err != nil && m.err != context.Canceled {
			b.WriteString(styles.ErrorText.Render(fmt.Sprintf("Error: %v", m.err)))
		} else {
			total := int64(0)
			if stats != nil {
				total = stats.BusinessesStored.Load()
			}
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Success).Bold(true).
				Render(fmt.Sprintf("Complete! %d businesses stored", total)))
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Foreground(styles.Muted).
				Render(fmt.Sprintf("Database: %s", m.dbPath)))
		}
		b.WriteString("\n\n")
		b.WriteString(styles.StatusBar.Render("enter explore results • esc back"))
	} else if m.confirmQuit {
		b.WriteString(styles.ErrorText.Render("Press ESC again to stop the scan and go back"))
		b.WriteString("\n")
		b.WriteString(styles.StatusBar.Render("esc confirm stop • any key continue"))
	} else {
		b.WriteString(styles.StatusBar.Render("esc cancel • ctrl+c quit"))
	}

	return b.String()
}

func (m ProgressModel) renderStats() string {
	var sb strings.Builder
	elapsed := time.Since(m.startTime).Truncate(time.Second)

	var sectorsDone int64
	var sectorsTotal int64
	var found, stored, errors, rateLimits int64

	stats := m.shared.getStats()
	if stats != nil {
		sectorsDone = stats.SectorsDone.Load()
		sectorsTotal = int64(stats.SectorsTotal)
		found = stats.BusinessesFound.Load()
		stored = stats.BusinessesStored.Load()
		errors = stats.Errors.Load()
		rateLimits = stats.RateLimits.Load()
	}

	statLabel := lipgloss.NewStyle().Foreground(styles.Muted).Width(12)
	statVal := lipgloss.NewStyle().Foreground(styles.Text).Bold(true)

	row := func(label string, value string) {
		sb.WriteString(statLabel.Render(label))
		sb.WriteString(statVal.Render(value))
		sb.WriteString("\n")
	}

	row("Sectors:", fmt.Sprintf("%d/%d", sectorsDone, sectorsTotal))
	row("Found:", fmt.Sprintf("%d", found))
	row("Stored:", fmt.Sprintf("%d", stored))

	errStyle := statVal
	if errors > 0 {
		errStyle = lipgloss.NewStyle().Foreground(styles.Error).Bold(true)
	}
	sb.WriteString(statLabel.Render("Errors:"))
	sb.WriteString(errStyle.Render(fmt.Sprintf("%d", errors)))
	sb.WriteString("\n")

	if rateLimits > 0 {
		rlStyle := lipgloss.NewStyle().Foreground(styles.Warning).Bold(true)
		sb.WriteString(statLabel.Render("Rate Lim:"))
		sb.WriteString(rlStyle.Render(fmt.Sprintf("%d", rateLimits)))
		sb.WriteString("\n")
	}

	row("Elapsed:", elapsed.String())

	// ETA
	if sectorsDone > 0 && sectorsTotal > 0 && !m.done {
		rate := float64(sectorsDone) / elapsed.Seconds()
		remaining := float64(sectorsTotal-sectorsDone) / rate
		eta := time.Duration(remaining * float64(time.Second)).Truncate(time.Second)
		row("ETA:", "~"+eta.String())
	}

	return sb.String()
}

func (s *sharedState) getCancel() context.CancelFunc {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel
}

func (s *sharedState) getStats() *scraper.Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}


// NavigateToExplorer signals transition to explorer view.
type NavigateToExplorer struct {
	DBPath string
}
