package main

import (
	"os"
	"testing"
	"time"
)

func TestGenerateAudio(t *testing.T) {
	audio := generateAudio()
	if audio == nil {
		t.Fatal("Audio should not be nil")
	}
	if len(audio.MIMEs) == 0 {
		t.Error("Audio should have at least one MIME type")
	}
	if audio.MinDuration <= 0 {
		t.Error("Audio min duration should be positive")
	}
	if audio.MaxDuration <= 0 {
		t.Error("Audio max duration should be positive")
	}
}

func TestGenerateNative(t *testing.T) {
	native := generateNative()
	if native == nil {
		t.Fatal("Native should not be nil")
	}
	if native.Request == "" {
		t.Error("Native request should not be empty")
	}
	if native.Ver == "" {
		t.Error("Native version should not be empty")
	}
}

func TestGenerateBatch(t *testing.T) {
	batch := GenerateBatch(5, "random", "banner")
	if len(batch) != 5 {
		t.Errorf("got %d requests, want 5", len(batch))
	}
	for i, req := range batch {
		if req.ID == "" {
			t.Errorf("request %d: ID should not be empty", i)
		}
	}
}

func TestRandomTimestamp(t *testing.T) {
	start := time.Now().Add(-5 * time.Minute)
	end := time.Now()

	for range 20 {
		ts := randomTimestamp(start, end)
		if ts.Before(start) || ts.After(end) {
			t.Errorf("timestamp %v outside [%v, %v]", ts, start, end)
		}
	}
}

func TestRandomTimestamp_ZeroDelta(t *testing.T) {
	now := time.Now()
	ts := randomTimestamp(now, now)
	if !ts.Equal(now) {
		t.Errorf("expected %v, got %v", now, ts)
	}
}

func TestGenerateGeoInBBox(t *testing.T) {
	bbox := &BoundingBox{MaxLat: 51.0, MaxLon: 0.5, MinLat: 50.0, MinLon: -0.5}

	for range 20 {
		geo := generateGeoInBBox(bbox)
		if geo == nil {
			t.Fatal("geo should not be nil")
		}
		if geo.Lat < bbox.MinLat || geo.Lat > bbox.MaxLat {
			t.Errorf("lat %f outside [%f, %f]", geo.Lat, bbox.MinLat, bbox.MaxLat)
		}
		if geo.Lon < bbox.MinLon || geo.Lon > bbox.MaxLon {
			t.Errorf("lon %f outside [%f, %f]", geo.Lon, bbox.MinLon, bbox.MaxLon)
		}
	}
}

func TestGenerateDeviceWithBBox(t *testing.T) {
	bbox := &BoundingBox{MaxLat: 40.9, MaxLon: -73.7, MinLat: 40.5, MinLon: -74.1}
	config := DefaultConfig
	config.BoundingBox = bbox

	for range 10 {
		device := generateDevice(config)
		if device.Geo == nil {
			t.Fatal("device geo should not be nil")
		}
		if device.Geo.Lat < bbox.MinLat || device.Geo.Lat > bbox.MaxLat {
			t.Errorf("lat %f outside bbox", device.Geo.Lat)
		}
		if device.Geo.Lon < bbox.MinLon || device.Geo.Lon > bbox.MaxLon {
			t.Errorf("lon %f outside bbox", device.Geo.Lon)
		}
	}
}

func TestGenerateRequestForTask_IP(t *testing.T) {
	task := &Task{
		ID:            "task-1",
		CorrelationID: "corr-1",
		CriteriaType:  CriteriaIP,
		IPAddress:     "10.0.0.1",
	}
	now := time.Now()
	req := generateRequestForTask(task, now.Add(-5*time.Minute), now)

	if req.Device.IP != "10.0.0.1" {
		t.Errorf("got IP %q, want %q", req.Device.IP, "10.0.0.1")
	}
	ext, ok := req.Ext.(map[string]any)
	if !ok {
		t.Fatal("ext should be a map")
	}
	if ext["task_id"] != "task-1" {
		t.Errorf("ext.task_id: got %v, want task-1", ext["task_id"])
	}
	if ext["correlation_id"] != "corr-1" {
		t.Errorf("ext.correlation_id: got %v, want corr-1", ext["correlation_id"])
	}
	if _, ok := ext["timestamp"]; !ok {
		t.Error("ext.timestamp should be set")
	}
}

func TestGenerateRequestForTask_IFA(t *testing.T) {
	task := &Task{
		CriteriaType: CriteriaIFA,
		IFA:          "ifa-abc-123",
	}
	now := time.Now()
	req := generateRequestForTask(task, now.Add(-5*time.Minute), now)

	if req.Device.IFA != "ifa-abc-123" {
		t.Errorf("got IFA %q, want %q", req.Device.IFA, "ifa-abc-123")
	}
}

func TestGenerateRequestForTask_BBox(t *testing.T) {
	bbox := &BoundingBox{MaxLat: 51.0, MaxLon: 0.5, MinLat: 50.0, MinLon: -0.5}
	task := &Task{
		CriteriaType: CriteriaBBox,
		BoundingBox:  bbox,
	}
	now := time.Now()
	for range 10 {
		req := generateRequestForTask(task, now.Add(-5*time.Minute), now)
		geo := req.Device.Geo
		if geo.Lat < bbox.MinLat || geo.Lat > bbox.MaxLat {
			t.Errorf("lat %f outside bbox", geo.Lat)
		}
	}
}

func TestGenerateImpression_AudioNative(t *testing.T) {
	audio := generateImpression("audio", DefaultConfig)
	if audio.Audio == nil {
		t.Error("audio impression should have Audio set")
	}

	native := generateImpression("native", DefaultConfig)
	if native.Native == nil {
		t.Error("native impression should have Native set")
	}
}

func TestParseBoundingBox(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		maxLat  float64
	}{
		{"40.9,-73.7,40.5,-74.1", false, 40.9},
		{"51.0,0.5,50.0,-0.5", false, 51.0},
		{"bad,input", true, 0},
		{"1,2,3", true, 0},
		{"a,b,c,d", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bbox, err := parseBoundingBox(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if bbox.MaxLat != tt.maxLat {
				t.Errorf("MaxLat: got %f, want %f", bbox.MaxLat, tt.maxLat)
			}
		})
	}
}

func TestGenerateUA(t *testing.T) {
	tests := []struct{ os, make string }{
		{"iOS", "Apple"},
		{"Android", "Samsung"},
		{"Windows", ""},
		{"MacOS", ""},
		{"ChromeOS", ""},
	}
	for _, tt := range tests {
		t.Run(tt.os, func(t *testing.T) {
			ua := generateUA(tt.os, tt.make)
			if ua == "" {
				t.Error("UA should not be empty")
			}
		})
	}
}

func TestGenerateImpression_Random(t *testing.T) {
	gotBanner, gotVideo := false, false
	for range 100 {
		imp := generateImpression("random", DefaultConfig)
		if imp.Banner != nil {
			gotBanner = true
		}
		if imp.Video != nil {
			gotVideo = true
		}
		if gotBanner && gotVideo {
			break
		}
	}
	if !gotBanner {
		t.Error("expected at least one banner from random impression type")
	}
	if !gotVideo {
		t.Error("expected at least one video from random impression type")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	sc := NewScheduler(newTestStore(t), t.TempDir(), 100*time.Millisecond)
	done := make(chan struct{})
	go func() {
		sc.Start()
		close(done)
	}()
	sc.Stop()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("scheduler did not stop within 1s")
	}
}

func TestScheduler_RunNoActiveTasks(t *testing.T) {
	outDir := t.TempDir()
	sc := NewScheduler(newTestStore(t), outDir, 5*time.Minute)
	sc.run(time.Now())
	entries, _ := os.ReadDir(outDir)
	if len(entries) != 0 {
		t.Errorf("expected no output files, got %d", len(entries))
	}
}

func TestScheduler_GenerateForTask_OutDirError(t *testing.T) {
	// Use a file as the output dir so MkdirAll fails.
	blockingFile := t.TempDir() + "/file"
	os.WriteFile(blockingFile, []byte("x"), 0644)
	sc := NewScheduler(newTestStore(t), blockingFile+"/subdir", 5*time.Minute)
	err := sc.generateForTask(&Task{ID: randomID(), Count: 1, CriteriaType: CriteriaIP, IPAddress: "1.2.3.4"}, time.Now())
	if err == nil {
		t.Error("expected error when outDir cannot be created")
	}
}

func TestScheduler_GenerateForTask_FileCreateError(t *testing.T) {
	outDir := t.TempDir()
	os.Chmod(outDir, 0444)
	defer os.Chmod(outDir, 0755)
	sc := NewScheduler(newTestStore(t), outDir, 5*time.Minute)
	err := sc.generateForTask(&Task{ID: randomID(), Count: 1, CriteriaType: CriteriaIP, IPAddress: "1.2.3.4"}, time.Now())
	if err == nil {
		t.Error("expected error when output file cannot be created")
	}
}

func TestScheduler_GenerateForTask(t *testing.T) {
	store := newTestStore(t)
	outDir := t.TempDir()
	sc := NewScheduler(store, outDir, 5*time.Minute)

	now := time.Now()
	task := &Task{
		ID:            randomID(),
		CorrelationID: "corr",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIP,
		IPAddress:     "1.2.3.4",
		Count:         3,
	}

	if err := sc.generateForTask(task, now); err != nil {
		t.Fatalf("generateForTask: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d files, want 1", len(entries))
	}
}

func TestScheduler_Run(t *testing.T) {
	store := newTestStore(t)
	outDir := t.TempDir()
	sc := NewScheduler(store, outDir, 5*time.Minute)

	now := time.Now()
	active := &Task{
		ID:            randomID(),
		CorrelationID: "corr",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIP,
		IPAddress:     "1.2.3.4",
		Count:         2,
	}
	store.Add(active)

	sc.run(now)

	entries, _ := os.ReadDir(outDir)
	if len(entries) != 1 {
		t.Errorf("got %d output files, want 1", len(entries))
	}
}
