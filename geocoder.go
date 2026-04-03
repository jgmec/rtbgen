package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ReverseGeocoder enriches Geo objects with city/country/region metadata using
// the Nominatim reverse geocoding API. Results are cached by rounded coordinates
// (~1 km grid cells). Requests are rate-limited to 1 per second per Nominatim's
// usage policy.
type ReverseGeocoder struct {
	baseURL  string
	client   *http.Client
	cache    map[string]*geoMeta
	mu       sync.Mutex
	lastCall time.Time
}

type geoMeta struct {
	City    string
	Country string
	Region  string
	Zip     string
}

type nominatimResponse struct {
	Address struct {
		City        string `json:"city"`
		Town        string `json:"town"`
		Village     string `json:"village"`
		State       string `json:"state"`
		Country     string `json:"country"`
		CountryCode string `json:"country_code"`
		Postcode    string `json:"postcode"`
	} `json:"address"`
}

func NewReverseGeocoder(baseURL string) *ReverseGeocoder {
	return &ReverseGeocoder{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
		cache:   make(map[string]*geoMeta),
	}
}

// Enrich fills in City, Country, Region, and Zip on geo using a cached Nominatim
// lookup. If the lookup fails, geo is returned unchanged.
func (g *ReverseGeocoder) Enrich(geo *Geo) *Geo {
	if g == nil || geo == nil {
		return geo
	}

	key := fmt.Sprintf("%.2f,%.2f", geo.Lat, geo.Lon)

	g.mu.Lock()
	if meta, ok := g.cache[key]; ok {
		g.mu.Unlock()
		return applyMeta(geo, meta)
	}

	// Rate-limit: ensure at least 1 second between requests.
	if wait := time.Second - time.Since(g.lastCall); wait > 0 {
		g.mu.Unlock()
		time.Sleep(wait)
		g.mu.Lock()
	}
	g.lastCall = time.Now()
	g.mu.Unlock()

	meta := g.lookup(geo.Lat, geo.Lon)

	g.mu.Lock()
	g.cache[key] = meta
	g.mu.Unlock()

	return applyMeta(geo, meta)
}

func (g *ReverseGeocoder) lookup(lat, lon float64) *geoMeta {
	url := fmt.Sprintf("%s/reverse?lat=%f&lon=%f&format=json", g.baseURL, lat, lon)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return &geoMeta{}
	}
	req.Header.Set("User-Agent", "rtbgen/1.0")
	req.Header.Set("Accept-Language", "en")

	resp, err := g.client.Do(req)
	if err != nil {
		return &geoMeta{}
	}
	defer resp.Body.Close()

	var nr nominatimResponse
	if err := json.NewDecoder(resp.Body).Decode(&nr); err != nil {
		return &geoMeta{}
	}

	city := nr.Address.City
	if city == "" {
		city = nr.Address.Town
	}
	if city == "" {
		city = nr.Address.Village
	}

	country := nr.Address.CountryCode
	if country != "" {
		// Uppercase to match ISO 3166-1 alpha-2 convention used elsewhere.
		for i, c := range country {
			if c >= 'a' && c <= 'z' {
				country = country[:i] + string(c-32) + country[i+1:]
			}
		}
	}

	return &geoMeta{
		City:    city,
		Country: country,
		Region:  nr.Address.State,
		Zip:     nr.Address.Postcode,
	}
}

func applyMeta(geo *Geo, meta *geoMeta) *Geo {
	if meta == nil {
		return geo
	}
	out := *geo
	if meta.City != "" {
		out.City = meta.City
	}
	if meta.Country != "" {
		out.Country = meta.Country
	}
	if meta.Region != "" {
		out.Region = meta.Region
	}
	if meta.Zip != "" {
		out.Zip = meta.Zip
	}
	return &out
}
