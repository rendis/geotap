package geo

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

//go:embed geodata/ne_110m_countries.geojson
var countriesFS embed.FS

type BoundaryStore struct {
	features map[string]*geojson.Feature // key: lowercase country name or ISO code
}

func NewBoundaryStore() (*BoundaryStore, error) {
	data, err := countriesFS.ReadFile("geodata/ne_110m_countries.geojson")
	if err != nil {
		return nil, fmt.Errorf("reading embedded geojson: %w", err)
	}

	fc := &geojson.FeatureCollection{}
	if err := json.Unmarshal(data, fc); err != nil {
		return nil, fmt.Errorf("parsing geojson: %w", err)
	}

	store := &BoundaryStore{
		features: make(map[string]*geojson.Feature),
	}

	for _, f := range fc.Features {
		// Index by multiple keys
		if name, ok := f.Properties["NAME"].(string); ok {
			store.features[strings.ToLower(name)] = f
		}
		if iso2, ok := f.Properties["ISO_A2"].(string); ok {
			store.features[strings.ToLower(iso2)] = f
		}
		if iso3, ok := f.Properties["ISO_A3"].(string); ok {
			store.features[strings.ToLower(iso3)] = f
		}
		// Also index by ADMIN name
		if admin, ok := f.Properties["ADMIN"].(string); ok {
			store.features[strings.ToLower(admin)] = f
		}
		// Index by Spanish name
		if nameES, ok := f.Properties["NAME_ES"].(string); ok {
			store.features[strings.ToLower(nameES)] = f
		}
	}

	return store, nil
}

// GetCountryPolygon returns the MultiPolygon for a country by name or ISO code.
func (bs *BoundaryStore) GetCountryPolygon(country string) (orb.MultiPolygon, error) {
	f, ok := bs.features[strings.ToLower(country)]
	if !ok {
		return nil, fmt.Errorf("country %q not found in boundaries", country)
	}

	switch g := f.Geometry.(type) {
	case orb.MultiPolygon:
		return g, nil
	case orb.Polygon:
		return orb.MultiPolygon{g}, nil
	default:
		return nil, fmt.Errorf("unexpected geometry type %T for %q", g, country)
	}
}

// GetCountryBounds returns the bounding box for a country.
func (bs *BoundaryStore) GetCountryBounds(country string) (minLat, minLng, maxLat, maxLng float64, err error) {
	f, ok := bs.features[strings.ToLower(country)]
	if !ok {
		return 0, 0, 0, 0, fmt.Errorf("country %q not found", country)
	}

	bound := f.Geometry.Bound()
	return bound.Min.Lat(), bound.Min.Lon(), bound.Max.Lat(), bound.Max.Lon(), nil
}

// CountryEntry holds display info for a country.
type CountryEntry struct {
	Name   string // English name (canonical)
	NameES string // Spanish name
	ISO2   string // 2-letter ISO code
	ISO3   string // 3-letter ISO code
}

// ListCountryEntries returns all countries sorted by name.
func (bs *BoundaryStore) ListCountryEntries() []CountryEntry {
	seen := make(map[string]bool)
	var entries []CountryEntry
	for _, f := range bs.features {
		name, _ := f.Properties["NAME"].(string)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		nameES, _ := f.Properties["NAME_ES"].(string)
		iso2, _ := f.Properties["ISO_A2"].(string)
		iso3, _ := f.Properties["ISO_A3"].(string)
		entries = append(entries, CountryEntry{Name: name, NameES: nameES, ISO2: iso2, ISO3: iso3})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}

// ValidateCountry checks if input matches any indexed key and returns the canonical name.
func (bs *BoundaryStore) ValidateCountry(input string) (string, bool) {
	f, ok := bs.features[strings.ToLower(strings.TrimSpace(input))]
	if !ok {
		return "", false
	}
	name, _ := f.Properties["NAME"].(string)
	return name, true
}

// ListCountries returns all available country names.
func (bs *BoundaryStore) ListCountries() []string {
	seen := make(map[string]bool)
	var names []string
	for _, f := range bs.features {
		if name, ok := f.Properties["NAME"].(string); ok && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}
