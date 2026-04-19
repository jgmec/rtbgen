package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// TimezoneClient resolves IANA timezone names from coordinates using a
// tzf-server instance. Results are cached by rounded coordinates (~1 km grid).
type TimezoneClient struct {
	baseURL string
	client  *http.Client
	cache   map[string]string // "%.2f,%.2f" → IANA timezone name
	mu      sync.Mutex
}

func NewTimezoneClient(baseURL string) *TimezoneClient {
	return &TimezoneClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
		cache:   make(map[string]string),
	}
}

// Timezone returns the IANA timezone name (e.g. "America/New_York") for the
// given coordinates. Returns an empty string when the client is nil or the
// lookup fails.
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

	tz := c.lookup(lat, lon)
	if tz == "" {
		return ""
	}

	c.mu.Lock()
	c.cache[key] = tz
	c.mu.Unlock()

	return tz
}

func (c *TimezoneClient) lookup(lat, lon float64) string {
	url := fmt.Sprintf("%s/api/v1/tz?latitude=%f&longitude=%f", c.baseURL, lat, lon)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "rtbgen/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.Timezone
}
