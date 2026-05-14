package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

func lookupIPGeo(mmdb *geoip2.Reader, ipStr string) *Geo {
	if mmdb != nil {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			if record, err := mmdb.City(ip); err == nil &&
				(record.Location.Latitude != 0 || record.Location.Longitude != 0) {
				geo := &Geo{
					Lat:     record.Location.Latitude,
					Lon:     record.Location.Longitude,
					Country: record.Country.IsoCode,
					City:    record.City.Names["en"],
					Zip:     record.Postal.Code,
					Type:    2,
				}
				if len(record.Subdivisions) > 0 {
					geo.Region = record.Subdivisions[0].IsoCode
				}
				return geo
			}
		}
	}
	return generateGeo()
}

// ReverseGeocoder enriches Geo objects with city/country/region metadata using
// the Nominatim reverse geocoding API. Results are cached by rounded coordinates
// (~1 km grid cells). minDelay controls the minimum gap between consecutive HTTP
// calls; use 1s for the public Nominatim instance, 0 for self-hosted.
type ReverseGeocoder struct {
	baseURL  string
	minDelay time.Duration
	client   *http.Client
	cache    map[string]*geoMeta
	cacheMu  sync.RWMutex // guards cache reads and writes
	rateMu   sync.Mutex   // held across sleep + HTTP call; serialises requests
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

func NewReverseGeocoder(baseURL string, minDelay time.Duration) *ReverseGeocoder {
	return &ReverseGeocoder{
		baseURL:  baseURL,
		minDelay: minDelay,
		client:   &http.Client{Timeout: 5 * time.Second},
		cache:    make(map[string]*geoMeta),
	}
}

// Enrich fills in City, Country, Region, and Zip on geo using a cached Nominatim
// lookup. If the lookup fails, geo is returned unchanged.
func (g *ReverseGeocoder) Enrich(geo *Geo) *Geo {
	if g == nil || geo == nil {
		return geo
	}

	key := fmt.Sprintf("%.2f,%.2f", geo.Lat, geo.Lon)

	// Fast path: concurrent cache reads.
	g.cacheMu.RLock()
	if meta, ok := g.cache[key]; ok {
		g.cacheMu.RUnlock()
		return applyMeta(geo, meta)
	}
	g.cacheMu.RUnlock()

	// Slow path: one goroutine at a time through the rate limiter.
	// Holding rateMu across the sleep and the HTTP call ensures at most one
	// in-flight Nominatim request at any moment.
	g.rateMu.Lock()
	// Double-check: the previous holder may have fetched this key.
	g.cacheMu.RLock()
	if meta, ok := g.cache[key]; ok {
		g.cacheMu.RUnlock()
		g.rateMu.Unlock()
		return applyMeta(geo, meta)
	}
	g.cacheMu.RUnlock()

	if g.minDelay > 0 {
		if wait := g.minDelay - time.Since(g.lastCall); wait > 0 {
			time.Sleep(wait)
		}
	}
	g.lastCall = time.Now()
	meta := g.lookup(geo.Lat, geo.Lon)
	g.rateMu.Unlock()

	g.cacheMu.Lock()
	g.cache[key] = meta
	g.cacheMu.Unlock()

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
