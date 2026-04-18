package geocoder

import (
	"log"

	"github.com/sams96/rgeo"
	"github.com/twpayne/go-geom"
)

// Geocoder wraps the rgeo library for offline reverse geocoding.
type Geocoder struct {
	r *rgeo.Rgeo
}

// NewGeocoder creates a new Geocoder with embedded geographic datasets.
// Returns nil if initialization fails (metadata extraction will still work without geo).
func NewGeocoder() *Geocoder {
	r, err := rgeo.New(rgeo.Provinces10, rgeo.Cities10)
	if err != nil {
		log.Printf("WARNING: Failed to initialize geocoder: %v (geolocation will be disabled)", err)
		return nil
	}
	return &Geocoder{r: r}
}

// ReverseGeocode converts GPS coordinates to country and city/province names.
// Returns empty strings if the location cannot be determined.
func (g *Geocoder) ReverseGeocode(lat, lng float64) (country, city string) {
	if g == nil || g.r == nil {
		return "", ""
	}

	// rgeo uses GeoJSON convention: [longitude, latitude]
	loc, err := g.r.ReverseGeocode(geom.Coord{lng, lat})
	if err != nil {
		return "", ""
	}

	country = loc.CountryLong
	city = loc.City
	if city == "" {
		city = loc.Province
	}

	return country, city
}
