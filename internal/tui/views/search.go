package views

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rendis/map_scrapper/internal/engine/geo"
	"github.com/rendis/map_scrapper/internal/tui/styles"
)

type searchMode int

const (
	modeCountry searchMode = iota
	modeCoords
)

// Field indices — fieldMode is a virtual field (not a textinput)
const (
	fieldMode = iota
	fieldQuery
	fieldCountry
	fieldRegion
	fieldLat
	fieldLng
	fieldRadius
	fieldZoom
	fieldConcurrency
	fieldOutput
	fieldCount
)

type SearchModel struct {
	inputs      []textinput.Model
	mode        searchMode
	focused     int
	err         string
	countries   []geo.CountryEntry
	suggestions []geo.CountryEntry
	suggIdx     int
}

type countriesLoadedMsg struct {
	entries []geo.CountryEntry
}

func NewSearchModel() SearchModel {
	inputs := make([]textinput.Model, fieldCount)

	inputs[fieldMode] = textinput.New() // placeholder, never used
	inputs[fieldQuery] = newInput("restaurants, cafes", "", 60)
	inputs[fieldCountry] = newInput("type to search country...", "", 30)
	inputs[fieldRegion] = newInput("optional: region, state, city...", "", 40)
	inputs[fieldLat] = newInput("40.4168", "", 15)
	inputs[fieldLng] = newInput("-3.7038", "", 15)
	inputs[fieldRadius] = newInput("10", "", 10)
	inputs[fieldZoom] = newInput("10", "", 5)
	inputs[fieldConcurrency] = newInput("50", "", 5)
	inputs[fieldOutput] = newInput("./projects", "", 50)

	return SearchModel{
		inputs:  inputs,
		mode:    modeCountry,
		focused: fieldMode,
		suggIdx: -1,
	}
}

func newInput(placeholder, value string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 100
	if width > 0 {
		ti.Width = width
	}
	if value != "" {
		ti.SetValue(value)
	}
	return ti
}

func (m SearchModel) Init() tea.Cmd {
	return func() tea.Msg {
		bs, err := geo.NewBoundaryStore()
		if err != nil {
			return countriesLoadedMsg{}
		}
		return countriesLoadedMsg{entries: bs.ListCountryEntries()}
	}
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case countriesLoadedMsg:
		m.countries = msg.entries
		return m, nil
	case tea.KeyMsg:
		key := msg.String()

		switch key {
		case "esc":
			return m, func() tea.Msg { return NavigateToHome{} }

		case "up":
			if m.focused == fieldCountry && len(m.suggestions) > 0 && m.suggIdx > 0 {
				m.suggIdx--
				return m, nil
			}
			m.err = ""
			return m, m.focusPrev()

		case "down":
			if m.focused == fieldCountry && len(m.suggestions) > 0 && m.suggIdx < len(m.suggestions)-1 {
				m.suggIdx++
				return m, nil
			}
			m.err = ""
			return m, m.focusNext()

		case "tab":
			m.err = ""
			if m.focused == fieldCountry && len(m.suggestions) > 0 {
				m.selectSuggestion()
			}
			return m, m.focusNext()

		case "shift+tab":
			m.err = ""
			return m, m.focusPrev()

		case "enter":
			if m.focused == fieldCountry && len(m.suggestions) > 0 {
				m.selectSuggestion()
				return m, m.focusNext()
			}
			if cmd := m.submit(); cmd != nil {
				return m, cmd
			}

		case "left":
			if m.focused == fieldMode {
				m.mode = modeCountry
				return m, nil
			}

		case "right":
			if m.focused == fieldMode {
				m.mode = modeCoords
				return m, nil
			}
		}
	}

	// Update focused textinput (skip mode field)
	var cmd tea.Cmd
	if m.focused != fieldMode && m.focused >= 0 && m.focused < fieldCount {
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	}

	// Update suggestions when typing in country field
	if m.focused == fieldCountry {
		m.updateSuggestions()
	}

	return m, cmd
}

func (m *SearchModel) selectSuggestion() {
	if m.suggIdx >= 0 && m.suggIdx < len(m.suggestions) {
		m.inputs[fieldCountry].SetValue(m.suggestions[m.suggIdx].Name)
		m.suggestions = nil
		m.suggIdx = -1
	}
}

func (m *SearchModel) updateSuggestions() {
	raw := strings.TrimSpace(m.inputs[fieldCountry].Value())
	if raw == "" {
		m.suggestions = nil
		m.suggIdx = -1
		return
	}

	q := normalize(raw)
	var matches []geo.CountryEntry
	for _, c := range m.countries {
		if strings.Contains(normalize(c.Name), q) ||
			(c.NameES != "" && strings.Contains(normalize(c.NameES), q)) ||
			strings.EqualFold(c.ISO2, raw) ||
			strings.EqualFold(c.ISO3, raw) {
			matches = append(matches, c)
			if len(matches) >= 5 {
				break
			}
		}
	}
	m.suggestions = matches
	if len(matches) > 0 {
		if m.suggIdx < 0 || m.suggIdx >= len(matches) {
			m.suggIdx = 0
		}
	} else {
		m.suggIdx = -1
	}
}

// resolveCountry checks if the input matches a known country and returns the canonical name.
func (m *SearchModel) resolveCountry(input string) (string, bool) {
	q := strings.ToLower(strings.TrimSpace(input))
	if q == "" {
		return "", false
	}
	for _, c := range m.countries {
		if strings.ToLower(c.Name) == q ||
			(c.NameES != "" && strings.ToLower(c.NameES) == q) ||
			strings.ToLower(c.ISO2) == q ||
			strings.ToLower(c.ISO3) == q {
			return c.Name, true
		}
	}
	return "", false
}

func (m *SearchModel) focusNext() tea.Cmd {
	if m.focused != fieldMode {
		m.inputs[m.focused].Blur()
	}
	m.focused++
	m.focused = m.skipField(m.focused, 1)
	if m.focused >= fieldCount {
		m.focused = fieldMode
	}
	if m.focused == fieldMode {
		return nil
	}
	m.inputs[m.focused].Focus()
	return textinput.Blink
}

func (m *SearchModel) focusPrev() tea.Cmd {
	if m.focused != fieldMode {
		m.inputs[m.focused].Blur()
	}
	m.focused--
	m.focused = m.skipField(m.focused, -1)
	if m.focused < 0 {
		m.focused = fieldOutput
		m.inputs[m.focused].Focus()
		return textinput.Blink
	}
	if m.focused == fieldMode {
		return nil
	}
	m.inputs[m.focused].Focus()
	return textinput.Blink
}

func (m *SearchModel) skipField(idx, dir int) int {
	for idx > fieldMode && idx < fieldCount {
		if m.mode == modeCountry && (idx == fieldLat || idx == fieldLng || idx == fieldRadius) {
			idx += dir
			continue
		}
		if m.mode == modeCoords && (idx == fieldCountry || idx == fieldRegion) {
			idx += dir
			continue
		}
		break
	}
	return idx
}

func (m *SearchModel) submit() tea.Cmd {
	query := strings.TrimSpace(m.inputs[fieldQuery].Value())
	if query == "" {
		m.err = "Query is required"
		return nil
	}
	output := strings.TrimSpace(m.inputs[fieldOutput].Value())
	if output == "" {
		m.err = "Output directory is required"
		return nil
	}

	var resolvedCountry string
	if m.mode == modeCountry {
		raw := strings.TrimSpace(m.inputs[fieldCountry].Value())
		if raw == "" {
			m.err = "Country is required"
			return nil
		}
		name, ok := m.resolveCountry(raw)
		if !ok {
			m.err = fmt.Sprintf("Unknown country %q — type to search", raw)
			return nil
		}
		resolvedCountry = name
	} else {
		if strings.TrimSpace(m.inputs[fieldLat].Value()) == "" ||
			strings.TrimSpace(m.inputs[fieldLng].Value()) == "" {
			m.err = "Lat and Lng are required"
			return nil
		}
		if strings.TrimSpace(m.inputs[fieldRadius].Value()) == "" {
			m.err = "Radius is required"
			return nil
		}
	}

	// Validate zoom
	zoomStr := strings.TrimSpace(m.inputs[fieldZoom].Value())
	if zoomStr != "" {
		z, err := strconv.Atoi(zoomStr)
		if err != nil {
			m.err = "Zoom must be a number (10-16)"
			return nil
		}
		if z < 10 || z > 16 {
			m.err = "Zoom must be between 10 and 16"
			return nil
		}
	}

	// Validate concurrency
	concStr := strings.TrimSpace(m.inputs[fieldConcurrency].Value())
	if concStr != "" {
		c, err := strconv.Atoi(concStr)
		if err != nil || c < 1 {
			m.err = "Concurrency must be a positive number"
			return nil
		}
	}

	return func() tea.Msg {
		return StartScanMsg{
			Query:       query,
			Mode:        m.mode,
			Country:     resolvedCountry,
			Region:      strings.TrimSpace(m.inputs[fieldRegion].Value()),
			Lat:         strings.TrimSpace(m.inputs[fieldLat].Value()),
			Lng:         strings.TrimSpace(m.inputs[fieldLng].Value()),
			Radius:      strings.TrimSpace(m.inputs[fieldRadius].Value()),
			Zoom:        zoomStr,
			Concurrency: concStr,
			Output:      output,
		}
	}
}

func (m SearchModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("New Search") + "\n\n")

	// Mode selector
	b.WriteString(m.renderMode())
	b.WriteString("\n")

	// Query
	b.WriteString(m.renderField("Query:", fieldQuery))

	// Mode-specific fields
	if m.mode == modeCountry {
		b.WriteString(m.renderField("Country:", fieldCountry))
		if m.focused == fieldCountry && len(m.suggestions) > 0 {
			b.WriteString(m.renderSuggestions())
		}
		b.WriteString(m.renderField("Region:", fieldRegion))
	} else {
		b.WriteString(m.renderField("Latitude:", fieldLat))
		b.WriteString(m.renderField("Longitude:", fieldLng))
		b.WriteString(m.renderField("Radius (km):", fieldRadius))
	}

	b.WriteString("\n")
	b.WriteString(m.renderField("Zoom:", fieldZoom))
	if m.focused == fieldZoom {
		hint := lipgloss.NewStyle().Foreground(styles.Muted).Italic(true).
			Render("  10-16 | lower=faster, fewer results | higher=slower, more coverage")
		b.WriteString(hint + "\n")
	}
	b.WriteString(m.renderField("Concurrency:", fieldConcurrency))
	b.WriteString(m.renderField("Output:", fieldOutput))

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(styles.ErrorText.Render("  " + m.err))
	}

	b.WriteString("\n\n")
	b.WriteString(styles.StatusBar.Render("enter start • tab next • esc back"))

	return styles.Border.Render(b.String())
}

func (m SearchModel) renderSuggestions() string {
	var sb strings.Builder
	active := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	inactive := lipgloss.NewStyle().Foreground(styles.Muted)

	for i, c := range m.suggestions {
		label := c.Name
		if c.ISO2 != "" && c.ISO2 != "-99" {
			label += " (" + c.ISO2 + ")"
		}
		if c.NameES != "" && c.NameES != c.Name {
			label += " · " + c.NameES
		}
		if i == m.suggIdx {
			sb.WriteString(active.Render("  > " + label))
		} else {
			sb.WriteString(inactive.Render("    " + label))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (m SearchModel) renderMode() string {
	label := styles.Label.Render("Mode:")

	active := lipgloss.NewStyle().Foreground(styles.Primary).Bold(true)
	inactive := lipgloss.NewStyle().Foreground(styles.Muted)

	var countryStr, coordsStr string
	if m.mode == modeCountry {
		countryStr = active.Render("< Country >")
		coordsStr = inactive.Render("Coordinates")
	} else {
		countryStr = inactive.Render("Country")
		coordsStr = active.Render("< Coordinates >")
	}

	line := fmt.Sprintf("%s  %s   %s", label, countryStr, coordsStr)

	if m.focused == fieldMode {
		indicator := lipgloss.NewStyle().Foreground(styles.Secondary).Render(" ←→")
		line += indicator
	}

	return line + "\n"
}

func (m SearchModel) renderField(label string, idx int) string {
	l := styles.Label.Render(label)
	v := m.inputs[idx].View()
	return fmt.Sprintf("%s %s\n", l, v)
}

// Messages
type NavigateToHome struct{}

type StartScanMsg struct {
	Query       string
	Mode        searchMode
	Country     string
	Region      string
	Lat         string
	Lng         string
	Radius      string
	Zoom        string
	Concurrency string
	Output      string
}
