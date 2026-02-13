package geo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type nominatimResult struct {
	BoundingBox []string `json:"boundingbox"` // [minLat, maxLat, minLng, maxLng]
	DisplayName string   `json:"display_name"`
}

// GeocodeRegion returns the bounding box for a region within a country
// using the OSM Nominatim API.
func GeocodeRegion(region, country string) (minLat, minLng, maxLat, maxLng float64, err error) {
	q := region
	if country != "" {
		q = region + ", " + country
	}

	u := "https://nominatim.openstreetmap.org/search?" + url.Values{
		"q":      {q},
		"format": {"json"},
		"limit":  {"1"},
	}.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "geotap/0.1 (geographic data scanner)")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("geocoding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, 0, 0, fmt.Errorf("geocoding returned status %d", resp.StatusCode)
	}

	var results []nominatimResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("decoding geocoding response: %w", err)
	}

	if len(results) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("region %q not found", q)
	}

	bb := results[0].BoundingBox
	if len(bb) < 4 {
		return 0, 0, 0, 0, fmt.Errorf("invalid bounding box from geocoder")
	}

	// Nominatim returns [minLat, maxLat, minLng, maxLng] as strings
	minLat, _ = strconv.ParseFloat(bb[0], 64)
	maxLat, _ = strconv.ParseFloat(bb[1], 64)
	minLng, _ = strconv.ParseFloat(bb[2], 64)
	maxLng, _ = strconv.ParseFloat(bb[3], 64)

	return minLat, minLng, maxLat, maxLng, nil
}
