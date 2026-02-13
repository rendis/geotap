package components

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rendis/map_scrapper/internal/tui/styles"
)

// Point represents a geographic point to plot.
type Point struct {
	Lat float64
	Lng float64
}

// MapView renders a scatter plot of geographic points using Braille characters.
type MapView struct {
	width    int
	height   int
	points   []Point
	border   []Point // country border polygon
	selected int     // index of selected point, -1 if none
	// Viewport bounds
	minLat, maxLat float64
	minLng, maxLng float64
	// Base bounds (for zoom reference)
	basMinLat, basMaxLat float64
	basMinLng, basMaxLng float64
	autoFit              bool
	zoomLevel            float64 // 1.0 = no zoom, >1 = zoomed in
	panLat, panLng       float64 // pan offset in degrees
}

func NewMapView(width, height int) MapView {
	return MapView{
		width:     width,
		height:    height,
		selected:  -1,
		autoFit:   true,
		zoomLevel: 1.0,
	}
}

func (m *MapView) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *MapView) SetBorder(points []Point) {
	m.border = points
}

func (m *MapView) SetPoints(points []Point) {
	m.points = points
	if m.autoFit {
		m.fitBounds()
	}
}

func (m *MapView) AddPoints(points []Point) {
	m.points = append(m.points, points...)
	// Don't refit on add — keep stable viewport
}

func (m *MapView) SetSelected(idx int) {
	m.selected = idx
}

func (m *MapView) SetBounds(minLat, minLng, maxLat, maxLng float64) {
	m.basMinLat = minLat
	m.basMaxLat = maxLat
	m.basMinLng = minLng
	m.basMaxLng = maxLng
	m.autoFit = false
	m.applyZoom()
}

func (m *MapView) ZoomIn() {
	m.zoomLevel *= 1.5
	if m.zoomLevel > 20 {
		m.zoomLevel = 20
	}
	m.applyZoom()
}

func (m *MapView) ZoomOut() {
	m.zoomLevel /= 1.5
	if m.zoomLevel < 0.5 {
		m.zoomLevel = 0.5
	}
	m.applyZoom()
}

func (m *MapView) ZoomReset() {
	m.zoomLevel = 1.0
	m.panLat = 0
	m.panLng = 0
	m.applyZoom()
}

func (m *MapView) Pan(dLat, dLng float64) {
	latRange := m.basMaxLat - m.basMinLat
	lngRange := m.basMaxLng - m.basMinLng
	m.panLat += dLat * latRange * 0.1 / m.zoomLevel
	m.panLng += dLng * lngRange * 0.1 / m.zoomLevel
	m.applyZoom()
}

func (m *MapView) applyZoom() {
	centerLat := (m.basMinLat+m.basMaxLat)/2 + m.panLat
	centerLng := (m.basMinLng+m.basMaxLng)/2 + m.panLng
	halfLat := (m.basMaxLat - m.basMinLat) / 2 / m.zoomLevel
	halfLng := (m.basMaxLng - m.basMinLng) / 2 / m.zoomLevel
	m.minLat = centerLat - halfLat
	m.maxLat = centerLat + halfLat
	m.minLng = centerLng - halfLng
	m.maxLng = centerLng + halfLng
}

func (m *MapView) fitBounds() {
	if len(m.border) > 0 {
		m.basMinLat = m.border[0].Lat
		m.basMaxLat = m.border[0].Lat
		m.basMinLng = m.border[0].Lng
		m.basMaxLng = m.border[0].Lng
		for _, p := range m.border {
			m.basMinLat = math.Min(m.basMinLat, p.Lat)
			m.basMaxLat = math.Max(m.basMaxLat, p.Lat)
			m.basMinLng = math.Min(m.basMinLng, p.Lng)
			m.basMaxLng = math.Max(m.basMaxLng, p.Lng)
		}
	} else if len(m.points) > 0 {
		m.basMinLat = m.points[0].Lat
		m.basMaxLat = m.points[0].Lat
		m.basMinLng = m.points[0].Lng
		m.basMaxLng = m.points[0].Lng
		for _, p := range m.points {
			m.basMinLat = math.Min(m.basMinLat, p.Lat)
			m.basMaxLat = math.Max(m.basMaxLat, p.Lat)
			m.basMinLng = math.Min(m.basMinLng, p.Lng)
			m.basMaxLng = math.Max(m.basMaxLng, p.Lng)
		}
	}
	// Add padding
	latPad := (m.basMaxLat - m.basMinLat) * 0.05
	lngPad := (m.basMaxLng - m.basMinLng) * 0.05
	if latPad == 0 {
		latPad = 0.01
	}
	if lngPad == 0 {
		lngPad = 0.01
	}
	m.basMinLat -= latPad
	m.basMaxLat += latPad
	m.basMinLng -= lngPad
	m.basMaxLng += lngPad
	m.applyZoom()
}

// Braille character encoding:
// Each braille char is a 2x4 dot grid.
// Dot positions:  0 3
//
//	1 4
//	2 5
//	6 7
//
// Unicode: 0x2800 + sum of raised dot bits
var brailleDots = [8]rune{0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80}

func (m MapView) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	// Each braille char represents 2 columns x 4 rows of dots
	cols := m.width
	rows := m.height
	dotW := cols * 2
	dotH := rows * 4

	latRange := m.maxLat - m.minLat
	lngRange := m.maxLng - m.minLng
	if latRange == 0 || lngRange == 0 {
		return strings.Repeat(strings.Repeat(" ", cols)+"\n", rows)
	}

	// Aspect ratio correction: 1° lng is shorter than 1° lat at higher latitudes.
	// A terminal char is ~2x taller than wide; braille dots are 2 wide x 4 tall per char,
	// so each dot is roughly square on screen.
	avgLat := (m.minLat + m.maxLat) / 2
	cosLat := math.Cos(avgLat * math.Pi / 180)
	geoW := lngRange * cosLat // geographic width in equatorial-degree units
	geoH := latRange          // geographic height

	// Fit into dot grid preserving aspect ratio
	geoAspect := geoW / geoH
	dotAspect := float64(dotW) / float64(dotH)

	effectiveW, effectiveH := dotW, dotH
	offsetX, offsetY := 0, 0
	if geoAspect < dotAspect {
		// Taller than wide (e.g. Chile): use full height, reduce width
		effectiveW = int(float64(dotH) * geoAspect)
		if effectiveW < 4 {
			effectiveW = 4
		}
		offsetX = (dotW - effectiveW) / 2
	} else {
		// Wider than tall: use full width, reduce height
		effectiveH = int(float64(dotW) / geoAspect)
		if effectiveH < 4 {
			effectiveH = 4
		}
		offsetY = (dotH - effectiveH) / 2
	}

	// Create dot grids
	borderGrid := make([][]bool, dotH)
	pointGrid := make([][]bool, dotH)
	for i := range borderGrid {
		borderGrid[i] = make([]bool, dotW)
		pointGrid[i] = make([]bool, dotW)
	}

	// Helper: convert lat/lng to dot coordinates with aspect correction
	toDot := func(lat, lng float64) (int, int) {
		x := offsetX + int((lng-m.minLng)/lngRange*float64(effectiveW-1))
		y := offsetY + int((m.maxLat-lat)/latRange*float64(effectiveH-1))
		return x, y
	}

	// Draw border as connected line segments (Bresenham)
	for i := 0; i < len(m.border); i++ {
		x0, y0 := toDot(m.border[i].Lat, m.border[i].Lng)
		next := (i + 1) % len(m.border)
		x1, y1 := toDot(m.border[next].Lat, m.border[next].Lng)

		// Skip lines that wrap around the whole polygon (between rings)
		dx := abs(x1 - x0)
		dy := abs(y1 - y0)
		if dx > dotW/2 || dy > dotH/2 {
			continue
		}

		drawLine(borderGrid, x0, y0, x1, y1, dotW, dotH)
	}

	// Plot business points
	for _, p := range m.points {
		x, y := toDot(p.Lat, p.Lng)
		if x >= 0 && x < dotW && y >= 0 && y < dotH {
			pointGrid[y][x] = true
		}
	}

	// Render braille characters
	borderStyle := lipgloss.NewStyle().Foreground(styles.Secondary)
	pointStyle := lipgloss.NewStyle().Foreground(styles.Success)

	var sb strings.Builder
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			var borderVal rune = 0x2800
			var pointVal rune = 0x2800

			dotPositions := [8][2]int{
				{0, 0}, {1, 0}, {2, 0}, {0, 1},
				{1, 1}, {2, 1}, {3, 0}, {3, 1},
			}

			for dot := 0; dot < 8; dot++ {
				dy := row*4 + dotPositions[dot][0]
				dx := col*2 + dotPositions[dot][1]
				if dy < dotH && dx < dotW {
					if borderGrid[dy][dx] {
						borderVal |= brailleDots[dot]
					}
					if pointGrid[dy][dx] {
						pointVal |= brailleDots[dot]
					}
				}
			}

			if pointVal != 0x2800 {
				sb.WriteString(pointStyle.Render(string(pointVal)))
			} else if borderVal != 0x2800 {
				sb.WriteString(borderStyle.Render(string(borderVal)))
			} else {
				sb.WriteRune(' ')
			}
		}
		if row < rows-1 {
			sb.WriteRune('\n')
		}
	}

	return sb.String()
}

// drawLine draws a line between two points using Bresenham's algorithm.
func drawLine(grid [][]bool, x0, y0, x1, y1, maxW, maxH int) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}
	err := dx + dy

	for {
		if x0 >= 0 && x0 < maxW && y0 >= 0 && y0 < maxH {
			grid[y0][x0] = true
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
