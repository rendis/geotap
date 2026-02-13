package geo

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/planar"

	"github.com/rendis/map_scrapper/internal/model"
)

// FilterLandSectors removes sectors that fall in the ocean (outside the given polygon).
func FilterLandSectors(sectors []model.Sector, landPoly orb.MultiPolygon) []model.Sector {
	var land []model.Sector
	for _, s := range sectors {
		point := orb.Point{s.Lng, s.Lat} // orb.Point is [lng, lat]
		if planar.MultiPolygonContains(landPoly, point) {
			land = append(land, s)
		}
	}
	return land
}
