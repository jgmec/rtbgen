package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func nominatimHandler(city, town, village, state, countryCode, postcode string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := nominatimResponse{}
		resp.Address.City = city
		resp.Address.Town = town
		resp.Address.Village = village
		resp.Address.State = state
		resp.Address.CountryCode = countryCode
		resp.Address.Postcode = postcode
		json.NewEncoder(w).Encode(resp)
	}
}

func TestReverseGeocoder_Enrich_NilGeocoder(t *testing.T) {
	var g *ReverseGeocoder
	geo := &Geo{Lat: 51.5, Lon: -0.1}
	got := g.Enrich(geo)
	if got != geo {
		t.Error("nil geocoder should return geo unchanged")
	}
}

func TestReverseGeocoder_Enrich_NilGeo(t *testing.T) {
	g := NewReverseGeocoder("http://localhost")
	if got := g.Enrich(nil); got != nil {
		t.Error("nil geo should return nil")
	}
}

func TestReverseGeocoder_Enrich_City(t *testing.T) {
	srv := httptest.NewServer(nominatimHandler("London", "", "", "England", "gb", "SW1A"))
	defer srv.Close()

	g := NewReverseGeocoder(srv.URL)
	geo := &Geo{Lat: 51.50, Lon: -0.12}
	out := g.Enrich(geo)

	if out.City != "London" {
		t.Errorf("city: got %q, want %q", out.City, "London")
	}
	if out.Country != "GB" {
		t.Errorf("country: got %q, want %q", out.Country, "GB")
	}
	if out.Region != "England" {
		t.Errorf("region: got %q, want %q", out.Region, "England")
	}
	if out.Zip != "SW1A" {
		t.Errorf("zip: got %q, want %q", out.Zip, "SW1A")
	}
	// Original geo must not be mutated.
	if geo.City != "" {
		t.Error("original geo was mutated")
	}
}

func TestReverseGeocoder_Enrich_FallbackTownThenVillage(t *testing.T) {
	// No city — should fall back to town.
	srv := httptest.NewServer(nominatimHandler("", "Smalltown", "", "County", "us", "12345"))
	defer srv.Close()
	g := NewReverseGeocoder(srv.URL)
	out := g.Enrich(&Geo{Lat: 40.00, Lon: -75.00})
	if out.City != "Smalltown" {
		t.Errorf("expected town fallback, got %q", out.City)
	}

	// No city or town — should fall back to village.
	srv2 := httptest.NewServer(nominatimHandler("", "", "Hamlet", "State", "de", "10115"))
	defer srv2.Close()
	g2 := NewReverseGeocoder(srv2.URL)
	out2 := g2.Enrich(&Geo{Lat: 52.52, Lon: 13.40})
	if out2.City != "Hamlet" {
		t.Errorf("expected village fallback, got %q", out2.City)
	}
}

func TestReverseGeocoder_Enrich_Cache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		nominatimHandler("Paris", "", "", "Île-de-France", "fr", "75001")(w, r)
	}))
	defer srv.Close()

	g := NewReverseGeocoder(srv.URL)
	geo := &Geo{Lat: 48.86, Lon: 2.35}

	g.Enrich(geo)
	g.Enrich(geo) // same rounded key — must hit cache
	g.Enrich(&Geo{Lat: 48.861, Lon: 2.352}) // rounds to same key — cache hit

	if n := calls.Load(); n != 1 {
		t.Errorf("expected 1 HTTP call, got %d", n)
	}
}

func TestReverseGeocoder_Enrich_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	g := NewReverseGeocoder(srv.URL)
	geo := &Geo{Lat: 10.00, Lon: 20.00, City: "original"}
	out := g.Enrich(geo)
	// Invalid JSON body after 500 — city should remain empty (not "original") because
	// applyMeta only overwrites non-empty fields; the failed lookup returns empty geoMeta.
	if out.City != "original" {
		t.Errorf("expected original city preserved, got %q", out.City)
	}
}

func TestReverseGeocoder_Enrich_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	g := NewReverseGeocoder(srv.URL)
	geo := &Geo{Lat: 11.00, Lon: 22.00}
	out := g.Enrich(geo)
	if out == nil {
		t.Fatal("expected non-nil geo")
	}
	if out.City != "" {
		t.Errorf("expected empty city after bad JSON, got %q", out.City)
	}
}

func TestReverseGeocoder_Enrich_RateLimit(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		nominatimHandler("Berlin", "", "", "Berlin", "de", "10115")(w, r)
	}))
	defer srv.Close()

	g := NewReverseGeocoder(srv.URL)
	// Force lastCall to now so the next Enrich must wait ~1s.
	g.lastCall = time.Now()

	start := time.Now()
	g.Enrich(&Geo{Lat: 52.52, Lon: 13.41}) // different key — cache miss, must wait
	elapsed := time.Since(start)

	if elapsed < 900*time.Millisecond {
		t.Errorf("rate limit not enforced: elapsed %v, want >= 1s", elapsed)
	}
}

func TestApplyMeta_NilMeta(t *testing.T) {
	geo := &Geo{Lat: 1, Lon: 2, City: "X"}
	out := applyMeta(geo, nil)
	if out != geo {
		t.Error("nil meta should return original geo pointer")
	}
}

func TestApplyMeta_PartialMeta(t *testing.T) {
	geo := &Geo{City: "OldCity", Country: "OldCountry", Zip: "OldZip"}
	meta := &geoMeta{City: "NewCity"} // only city set
	out := applyMeta(geo, meta)
	if out.City != "NewCity" {
		t.Errorf("city: got %q", out.City)
	}
	if out.Country != "OldCountry" {
		t.Errorf("country should be unchanged, got %q", out.Country)
	}
	if out.Zip != "OldZip" {
		t.Errorf("zip should be unchanged, got %q", out.Zip)
	}
}
