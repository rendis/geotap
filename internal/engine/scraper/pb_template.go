package scraper

import (
	"fmt"
	"math"
)

const (
	viewportW = 1024
	viewportH = 768
	pageSize  = 20
)

// BuildPB constructs the pb= protobuf URL parameter for tbm=map requests.
// The template is derived from gosom/google-maps-scraper's proven format.
func BuildPB(lat, lng float64, zoom, offset int) string {
	alt := altitude(lat, zoom)
	return fmt.Sprintf(
		"!4m12!1m3!1d%.4f!2d%.7f!3d%.7f!2m3!1f0!2f0!3f0!3m2!1i%d!2i%d!4f13.1"+
			"!7i%d!8i%d!10b1"+
			"!12m22!1m3!18b1!30b1!34e1!2m3!5m1!6e2!20e3!4b0!10b1!12b1!13b1!16b1!17m1!3e1!20m3!5e2!6b1!14b1!46m1!1b0!96b1"+
			"!19m4!2m3!1i360!2i120!4i8",
		alt, lng, lat,
		viewportW, viewportH,
		pageSize, offset,
	)
}

// altitude converts zoom level to meters for the !1d field.
// Formula: alt = (2 * pi * R * viewportH) / (512 * 2^zoom)
func altitude(lat float64, zoom int) float64 {
	const earthRadius = 6371010.0
	latRad := lat * math.Pi / 180
	return (2 * math.Pi * earthRadius * float64(viewportH) * math.Cos(latRad)) / (512 * math.Pow(2, float64(zoom)))
}
