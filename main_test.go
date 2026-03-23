package main

import (
	"encoding/json"
	"testing"
)

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
