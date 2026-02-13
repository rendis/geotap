package scraper

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/rendis/geotap/internal/model"
)

// ParseMapResponse parses a tbm=map JSON response into businesses.
// Returns the list of businesses and whether there may be more results (for pagination).
func ParseMapResponse(body []byte, query string) ([]model.Business, bool) {
	// Strip anti-XSS prefix )]}'\n
	if idx := bytes.IndexByte(body, '\n'); idx >= 0 && idx < 10 {
		body = body[idx+1:]
	}

	var raw []any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, false
	}

	// Business items are at root[0][1][1..N][14]
	items := safeSlice(safeGet(raw, 0, 1))
	if len(items) == 0 {
		return nil, false
	}

	var businesses []model.Business
	// Skip index 0 (search metadata), iterate actual results
	for i := 1; i < len(items); i++ {
		biz := safeSlice(safeGet(items, i, 14))
		if len(biz) == 0 {
			continue
		}

		name := safeString(safeGet(biz, 11))
		if name == "" {
			continue
		}

		// All categories joined
		var categories string
		if cats := safeSlice(safeGet(biz, 13)); len(cats) > 0 {
			var catStrs []string
			for _, c := range cats {
				if s := safeString(c); s != "" {
					catStrs = append(catStrs, s)
				}
			}
			categories = strings.Join(catStrs, ", ")
		}

		// Open hours as JSON
		var openHours string
		hours := safeGet(biz, 203, 0)
		if hours == nil {
			hours = safeGet(biz, 34, 1)
		}
		if hours != nil {
			if hb, err := json.Marshal(hours); err == nil {
				openHours = string(hb)
			}
		}

		b := model.Business{
			Name:        name,
			Query:       query,
			Rating:      safeFloat(safeGet(biz, 4, 7)),
			ReviewCount: int(safeFloat(safeGet(biz, 4, 8))),
			Category:    safeString(safeGet(biz, 13, 0)),
			Address:     safeString(safeGet(biz, 18)),
			PriceRange:  safeString(safeGet(biz, 4, 2)),
			Lat:         safeFloat(safeGet(biz, 9, 2)),
			Lng:         safeFloat(safeGet(biz, 9, 3)),
			CID:         safeString(safeGet(biz, 10)),
			Website:     safeString(safeGet(biz, 7, 0)),
			Phone:       safeString(safeGet(biz, 178, 0, 0)),
			GoogleURL:   buildGoogleURL(safeString(safeGet(biz, 78))),
			Description: safeString(safeGet(biz, 32, 1, 1)),
			PlaceID:     safeString(safeGet(biz, 78)),
			OpenHours:   openHours,
			Thumbnail:   safeString(safeGet(biz, 157)),
			Categories:  categories,
			City:        safeString(safeGet(biz, 183, 1, 3)),
			PostalCode:  safeString(safeGet(biz, 183, 1, 4)),
			CountryCode: safeString(safeGet(biz, 183, 1, 6)),
		}

		businesses = append(businesses, b)
	}

	hasMore := len(businesses) >= pageSize
	return businesses, hasMore
}

// buildGoogleURL constructs a Google Maps URL from a Place ID.
func buildGoogleURL(placeID string) string {
	if placeID == "" {
		return ""
	}
	return "https://www.google.com/maps/place/?q=place_id:" + placeID
}

// safeGet navigates nested []any arrays by index path without panicking.
func safeGet(data any, path ...int) any {
	current := data
	for _, idx := range path {
		slice, ok := current.([]any)
		if !ok || idx < 0 || idx >= len(slice) {
			return nil
		}
		current = slice[idx]
	}
	return current
}

// safeSlice converts any to []any, returns nil if not a slice.
func safeSlice(data any) []any {
	if data == nil {
		return nil
	}
	slice, ok := data.([]any)
	if !ok {
		return nil
	}
	return slice
}

// safeString extracts a string from any. Handles string and json.Number.
func safeString(data any) string {
	if data == nil {
		return ""
	}
	switch v := data.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

// safeFloat extracts a float64 from any. Handles float64, json.Number, and numeric strings.
func safeFloat(data any) float64 {
	if data == nil {
		return 0
	}
	switch v := data.(type) {
	case float64:
		return v
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	}
	return 0
}
