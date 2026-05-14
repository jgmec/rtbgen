package main

import (
	"fmt"
	"sync"

	"github.com/ringsaturn/tzf"
)

// TimezoneClient resolves IANA timezone names from coordinates using embedded
// tzf boundary data. Results are cached by rounded coordinates (~1 km grid).
type TimezoneClient struct {
	finder tzf.F
	cache  map[string]string
	mu     sync.Mutex
}

// NewTimezoneClient loads the embedded tzf timezone data. Returns an error if
// the bundled data cannot be parsed.
func NewTimezoneClient() (*TimezoneClient, error) {
	f, err := tzf.NewFullFinder()
	if err != nil {
		return nil, err
	}
	return &TimezoneClient{
		finder: f,
		cache:  make(map[string]string),
	}, nil
}

// Timezone returns the IANA timezone name (e.g. "America/New_York") for the
// given coordinates. Returns an empty string when the client is nil or no
// timezone covers the location.
func (c *TimezoneClient) Timezone(lat, lon float64) string {
	if c == nil {
		return ""
	}

	key := fmt.Sprintf("%.2f,%.2f", lat, lon)

	c.mu.Lock()
	if tz, ok := c.cache[key]; ok {
		c.mu.Unlock()
		return tz
	}
	c.mu.Unlock()

	// tzf takes (lng, lat) — longitude first
	tz := c.finder.GetTimezoneName(lon, lat)
	if tz == "" {
		return ""
	}

	c.mu.Lock()
	c.cache[key] = tz
	c.mu.Unlock()

	return tz
}
