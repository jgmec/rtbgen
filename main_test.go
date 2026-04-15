package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintJSON_Pretty(t *testing.T) {
	prettyPrint = true
	out := captureStdout(func() { printJSON(map[string]string{"key": "val"}) })
	if !json.Valid([]byte(out)) {
		t.Errorf("output is not valid JSON: %s", out)
	}
	// Pretty output contains newlines beyond a single-line compact form.
	compact := fmt.Sprintf(`{"key":"val"}`) + "\n"
	if out == compact {
		t.Error("expected indented output in pretty mode")
	}
}

func TestPrintJSON_Compact(t *testing.T) {
	prettyPrint = false
	out := captureStdout(func() { printJSON(map[string]string{"key": "val"}) })
	prettyPrint = true
	if !json.Valid([]byte(out)) {
		t.Errorf("output is not valid JSON: %s", out)
	}
}

func TestShowExamples_NoError(t *testing.T) {
	prettyPrint = true
	out := captureStdout(showExamples)
	if len(out) == 0 {
		t.Error("expected non-empty output from showExamples")
	}
}

func TestParseBoundingBox_Valid(t *testing.T) {
	bb, err := parseBoundingBox("40.9,-73.7,40.5,-74.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bb.MaxLat != 40.9 || bb.MaxLon != -73.7 || bb.MinLat != 40.5 || bb.MinLon != -74.1 {
		t.Errorf("unexpected bbox: %+v", bb)
	}
}

func TestParseBoundingBox_WrongFieldCount(t *testing.T) {
	if _, err := parseBoundingBox("40.9,-73.7,40.5"); err == nil {
		t.Error("expected error for wrong field count")
	}
}

func TestParseBoundingBox_InvalidFloat(t *testing.T) {
	if _, err := parseBoundingBox("40.9,-73.7,40.5,notafloat"); err == nil {
		t.Error("expected error for invalid float")
	}
}

func TestGenerateRandomBidRequest(t *testing.T) {
	tests := []struct {
		name        string
		requestType string
		impType     string
	}{
		{"Site Banner", "site", "banner"},
		{"App Banner", "app", "banner"},
		{"Site Video", "site", "video"},
		{"App Video", "app", "video"},
		{"Random Banner", "random", "banner"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := GenerateRandomBidRequest(tt.requestType, tt.impType)

			// Validate basic required fields
			if req.ID == "" {
				t.Error("BidRequest ID should not be empty")
			}
			if len(req.Imp) == 0 {
				t.Error("BidRequest should have at least one impression")
			}
			if req.AT == 0 {
				t.Error("Auction type (AT) should be set")
			}

			// Validate request type
			if tt.requestType == "site" && req.Site == nil {
				t.Error("Site should not be nil for site requests")
			}
			if tt.requestType == "app" && req.App == nil {
				t.Error("App should not be nil for app requests")
			}

			// Validate impression type
			for _, imp := range req.Imp {
				if imp.ID == "" {
					t.Error("Impression ID should not be empty")
				}

				if tt.impType == "banner" && imp.Banner == nil {
					t.Error("Banner should not be nil for banner impressions")
				}
				if tt.impType == "video" && imp.Video == nil {
					t.Error("Video should not be nil for video impressions")
				}
			}

			// Validate device
			if req.Device == nil {
				t.Error("Device should not be nil")
			} else {
				if req.Device.IP == "" {
					t.Error("Device IP should not be empty")
				}
			}

			// Validate user
			if req.User == nil {
				t.Error("User should not be nil")
			}
		})
	}
}

func TestBidRequestJSONMarshaling(t *testing.T) {
	req := GenerateRandomBidRequest("site", "banner")

	// Test JSON marshaling
	jsonData, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal BidRequest to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled BidRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON to BidRequest: %v", err)
	}

	// Validate key fields
	if unmarshaled.ID != req.ID {
		t.Errorf("Expected ID %s, got %s", req.ID, unmarshaled.ID)
	}
	if len(unmarshaled.Imp) != len(req.Imp) {
		t.Errorf("Expected %d impressions, got %d", len(req.Imp), len(unmarshaled.Imp))
	}
}

func TestGenerateBanner(t *testing.T) {
	banner := generateBanner()

	if banner == nil {
		t.Fatal("Banner should not be nil")
	}
	if banner.W == nil {
		t.Error("Banner width should not be nil")
	}
	if banner.H == nil {
		t.Error("Banner height should not be nil")
	}
	if len(banner.Format) == 0 {
		t.Error("Banner should have at least one format")
	}
}

func TestGenerateVideo(t *testing.T) {
	video := generateVideo()

	if video == nil {
		t.Fatal("Video should not be nil")
	}
	if len(video.MIMEs) == 0 {
		t.Error("Video should have at least one MIME type")
	}
	if video.MinDuration <= 0 {
		t.Error("Video min duration should be positive")
	}
	if video.MaxDuration <= 0 {
		t.Error("Video max duration should be positive")
	}
}

func TestGenerateDevice(t *testing.T) {
	device := generateDevice(DefaultConfig)

	if device == nil {
		t.Fatal("Device should not be nil")
	}
	if device.IP == "" {
		t.Error("Device IP should not be empty")
	}
	if device.UA == "" {
		t.Error("Device UA should not be empty")
	}
	if device.Geo == nil {
		t.Error("Device geo should not be nil")
	}
}

func TestGenerateGeo(t *testing.T) {
	geo := generateGeo()

	if geo == nil {
		t.Fatal("Geo should not be nil")
	}
	if geo.Country == "" {
		t.Error("Geo country should not be empty")
	}
	if geo.Lat < -90 || geo.Lat > 90 {
		t.Errorf("Geo latitude %f is out of valid range", geo.Lat)
	}
	if geo.Lon < -180 || geo.Lon > 180 {
		t.Errorf("Geo longitude %f is out of valid range", geo.Lon)
	}
}

func BenchmarkGenerateRandomBidRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateRandomBidRequest("random", "banner")
	}
}

func BenchmarkGenerateBanner(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateBanner()
	}
}

func BenchmarkGenerateDevice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateDevice(DefaultConfig)
	}
}
