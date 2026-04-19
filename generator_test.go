package main

import (
	"bytes"
	"encoding/json"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// readJSONLEntries returns only .jsonl directory entries, sorted by name.
func readJSONLEntries(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	all, _ := os.ReadDir(dir)
	var out []os.DirEntry
	for _, e := range all {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			out = append(out, e)
		}
	}
	return out
}

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
		CorrelationID: "task-1",
		CriteriaType:  CriteriaIP,
		IPAddress:     "10.0.0.1",
	}
	now := time.Now()
	req := generateRequestForTask(task, randomTimestamp(now.Add(-5*time.Minute), now), nil, "")

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
	if ext["correlation_id"] != "task-1" {
		t.Errorf("ext.correlation_id: got %v, want task-1", ext["correlation_id"])
	}
	if _, ok := ext["ts"]; !ok {
		t.Error("ext.timestamp should be set")
	}
}

func TestGenerateGeoNear(t *testing.T) {
	baseLat, baseLon := 51.5074, -0.1278 // London
	const radiusKm = 1.0
	const kmPerDegree = 111.0

	for range 50 {
		geo := generateGeoNear(baseLat, baseLon, radiusKm)
		dLat := (geo.Lat - baseLat) * kmPerDegree
		dLon := (geo.Lon - baseLon) * kmPerDegree * math.Cos(baseLat*math.Pi/180)
		dist := math.Sqrt(dLat*dLat + dLon*dLon)
		if dist > radiusKm {
			t.Errorf("point is %.3f km from base, want <= %.1f km", dist, radiusKm)
		}
	}
}

func TestGenerateRequestForTask_IFA_SameLocation(t *testing.T) {
	baseGeo := generateGeo()
	task := &Task{
		CorrelationID: "ifa-task",
		CriteriaType:  CriteriaIFA,
		IFA:           "ifa-abc-123",
	}
	now := time.Now()
	const radiusKm = 1.0
	const kmPerDegree = 111.0

	for range 20 {
		req := generateRequestForTask(task, randomTimestamp(now.Add(-5*time.Minute), now), baseGeo, "")
		geo := req.Device.Geo
		dLat := (geo.Lat - baseGeo.Lat) * kmPerDegree
		dLon := (geo.Lon - baseGeo.Lon) * kmPerDegree * math.Cos(baseGeo.Lat*math.Pi/180)
		dist := math.Sqrt(dLat*dLat + dLon*dLon)
		if dist > radiusKm {
			t.Errorf("IFA request geo is %.3f km from base, want <= %.1f km", dist, radiusKm)
		}
	}
}

func geoDistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const kmPerDegree = 111.0
	avgLat := (lat1 + lat2) / 2
	dLat := (lat2 - lat1) * kmPerDegree
	dLon := (lon2 - lon1) * kmPerDegree * math.Cos(avgLat*math.Pi/180)
	return math.Sqrt(dLat*dLat + dLon*dLon)
}

func TestIFA_ConsecutiveLocationsWithin2km(t *testing.T) {
	baseGeo := generateGeo()
	task := &Task{
		CorrelationID: "ifa-task",
		CriteriaType:  CriteriaIFA,
		IFA:           "38400000-8cf0-11bd-b23e-10b96e40000d",
	}
	now := time.Now()
	const count = 50
	const radiusKm = 1.0
	const kmPerDegree = 111.0

	// Bounding box of the 1 km radius circle around the base point.
	latDelta := radiusKm / kmPerDegree
	lonDelta := radiusKm / (kmPerDegree * math.Cos(baseGeo.Lat*math.Pi/180))
	bbox := BoundingBox{
		MinLat: baseGeo.Lat - latDelta,
		MaxLat: baseGeo.Lat + latDelta,
		MinLon: baseGeo.Lon - lonDelta,
		MaxLon: baseGeo.Lon + lonDelta,
	}

	reqs := make([]*BidRequest, count)
	for i := range count {
		reqs[i] = generateRequestForTask(task, randomTimestamp(now.Add(-5*time.Minute), now), baseGeo, "")
	}

	for i, req := range reqs {
		geo := req.Device.Geo

		// All locations within bounding box of 1 km radius.
		if geo.Lat < bbox.MinLat || geo.Lat > bbox.MaxLat {
			t.Errorf("request %d: lat %.6f outside bbox [%.6f, %.6f]", i, geo.Lat, bbox.MinLat, bbox.MaxLat)
		}
		if geo.Lon < bbox.MinLon || geo.Lon > bbox.MaxLon {
			t.Errorf("request %d: lon %.6f outside bbox [%.6f, %.6f]", i, geo.Lon, bbox.MinLon, bbox.MaxLon)
		}

		// Consecutive locations within 2 km.
		if i == 0 {
			continue
		}
		prev := reqs[i-1].Device.Geo
		dist := geoDistanceKm(prev.Lat, prev.Lon, geo.Lat, geo.Lon)
		if dist > 2.0 {
			t.Errorf("requests %d and %d are %.3f km apart, want <= 2 km", i-1, i, dist)
		}
	}
}

func TestIFA_ConsistentBaseGeoAcrossSchedulerRuns(t *testing.T) {
	store := newTestStore(t)
	outDir := t.TempDir()
	srv := NewServer(store, nil)
	sc := NewScheduler(store, outDir, 5*time.Minute, nil, nil, nil, nil)

	now := time.Now()
	task := &Task{
		CorrelationID: "ifa-persist",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIFA,
		IFA:           "38400000-8cf0-11bd-b23e-10b96e40000d",
		Count:         10,
		LastGeo:       srv.resolveInitialGeo(CriteriaIFA, ""),
	}
	store.Add(task)

	sc.run(now)

	// After tick 1: LastGeo must be persisted.
	afterTick1, _ := store.Get(task.CorrelationID)
	if afterTick1.LastGeo == nil {
		t.Fatal("LastGeo should be persisted after first scheduler run")
	}
	lastGeoTick1 := afterTick1.LastGeo

	sc.run(now.Add(5 * time.Minute))

	// After tick 2: LastGeo should be updated.
	afterTick2, _ := store.Get(task.CorrelationID)
	if afterTick2.LastGeo == nil {
		t.Fatal("LastGeo should be persisted after second scheduler run")
	}

	entries := readJSONLEntries(t, outDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 output files (one per run), got %d", len(entries))
	}

	// Collect all requests from both ticks.
	// The first request of tick 2 must start within 2 km of tick 1's last position.
	// All consecutive pairs (within a tick and across ticks) must be within 2 km.
	var allReqs []*BidRequest
	for _, entry := range entries {
		data, _ := os.ReadFile(outDir + "/" + entry.Name())
		for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
			var req BidRequest
			if err := json.Unmarshal(line, &req); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			allReqs = append(allReqs, &req)
		}
	}

	// First request of tick 2 must be within 2 km of tick 1's persisted LastGeo.
	if len(allReqs) > 10 {
		dist := geoDistanceKm(lastGeoTick1.Lat, lastGeoTick1.Lon,
			allReqs[10].Device.Geo.Lat, allReqs[10].Device.Geo.Lon)
		if dist > 2.0 {
			t.Errorf("first request of tick 2 is %.3f km from tick-1 LastGeo, want <= 2 km", dist)
		}
	}

	for i := 1; i < len(allReqs); i++ {
		prev := allReqs[i-1].Device.Geo
		cur := allReqs[i].Device.Geo
		if dist := geoDistanceKm(prev.Lat, prev.Lon, cur.Lat, cur.Lon); dist > 2.0 {
			t.Errorf("requests %d and %d are %.3f km apart, want <= 2 km", i-1, i, dist)
		}
	}
}

func TestGenerateRequestForTask_IFA(t *testing.T) {
	task := &Task{
		CriteriaType: CriteriaIFA,
		IFA:          "ifa-abc-123",
	}
	now := time.Now()
	req := generateRequestForTask(task, randomTimestamp(now.Add(-5*time.Minute), now), nil, "")

	if req.Device.IFA != "ifa-abc-123" {
		t.Errorf("got IFA %q, want %q", req.Device.IFA, "ifa-abc-123")
	}
}

func TestGenerateRequestForTask_BBox(t *testing.T) {
	task := &Task{
		CriteriaType: CriteriaBBox,
		Geometry:     testPolygon(-0.5, 50.0, 0.5, 51.0),
	}
	now := time.Now()
	for range 10 {
		req := generateRequestForTask(task, randomTimestamp(now.Add(-5*time.Minute), now), nil, "")
		geo := req.Device.Geo
		if geo.Lat < 50.0 || geo.Lat > 51.0 {
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
	sc := NewScheduler(newTestStore(t), t.TempDir(), 5*time.Minute, nil, nil, nil, nil)
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
	sc := NewScheduler(newTestStore(t), outDir, 5*time.Minute, nil, nil, nil, nil)
	sc.run(time.Now())
	entries := readJSONLEntries(t, outDir)
	if len(entries) != 0 {
		t.Errorf("expected no output files, got %d", len(entries))
	}
}

func TestScheduler_GenerateForTask_OutDirError(t *testing.T) {
	// Use a file as the output dir so MkdirAll fails.
	blockingFile := t.TempDir() + "/file"
	os.WriteFile(blockingFile, []byte("x"), 0644)
	sc := NewScheduler(newTestStore(t), blockingFile, 5*time.Minute, nil, nil, nil, nil)
	_, err := sc.generateForTask(&Task{CorrelationID: randomID(), Count: 1, CriteriaType: CriteriaIP, IPAddress: "1.2.3.4"}, time.Now())
	if err == nil {
		t.Error("expected error when outDir cannot be created")
	}
}

func TestScheduler_GenerateForTask_FileCreateError(t *testing.T) {
	outDir := t.TempDir()
	os.Chmod(outDir, 0444)
	defer os.Chmod(outDir, 0755)
	sc := NewScheduler(newTestStore(t), outDir, 5*time.Minute, nil, nil, nil, nil)
	_, err := sc.generateForTask(&Task{CorrelationID: randomID(), Count: 1, CriteriaType: CriteriaIP, IPAddress: "1.2.3.4"}, time.Now())
	if err == nil {
		t.Error("expected error when output file cannot be created")
	}
}

func TestScheduler_GenerateForTask(t *testing.T) {
	store := newTestStore(t)
	outDir := t.TempDir()
	sc := NewScheduler(store, outDir, 5*time.Minute, nil, nil, nil, nil)

	now := time.Now()
	task := &Task{
		CorrelationID: "corr",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIP,
		IPAddress:     "1.2.3.4",
		Count:         3,
	}

	if _, err := sc.generateForTask(task, now); err != nil {
		t.Fatalf("generateForTask: %v", err)
	}

	entries := readJSONLEntries(t, outDir)
	if len(entries) != 1 {
		t.Fatalf("got %d files, want 1", len(entries))
	}
}

func TestScheduler_Run(t *testing.T) {
	store := newTestStore(t)
	outDir := t.TempDir()
	sc := NewScheduler(store, outDir, 5*time.Minute, nil, nil, nil, nil)

	now := time.Now()
	active := &Task{
		CorrelationID: "corr",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIP,
		IPAddress:     "1.2.3.4",
		Count:         2,
	}
	store.Add(active)

	sc.run(now)

	entries := readJSONLEntries(t, outDir)
	if len(entries) != 1 {
		t.Errorf("got %d output files, want 1", len(entries))
	}
}

// openTestMMDB opens the local GeoLite2-City.mmdb for integration tests.
// Skips the test if the file is not present.
func openTestMMDB(t *testing.T) *geoip2.Reader {
	t.Helper()
	const mmdbPath = "data/GeoLite2-City.mmdb"
	if _, err := os.Stat(mmdbPath); os.IsNotExist(err) {
		t.Skip("GeoLite2-City.mmdb not found; skipping MMDB integration test")
	}
	db, err := geoip2.Open(mmdbPath)
	if err != nil {
		t.Fatalf("open mmdb: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestLookupIPGeo_NilMMDB(t *testing.T) {
	srv := NewServer(newTestStore(t), nil)
	geo := srv.lookupIPGeo("8.8.8.8")
	if geo == nil {
		t.Fatal("expected non-nil geo when mmdb is nil")
	}
}

func TestLookupIPGeo_InvalidIP(t *testing.T) {
	srv := NewServer(newTestStore(t), nil)
	geo := srv.lookupIPGeo("not-an-ip")
	if geo == nil {
		t.Fatal("expected non-nil fallback geo for invalid IP")
	}
}

func TestLookupIPGeo_WithMMDB_PublicIP(t *testing.T) {
	db := openTestMMDB(t)
	srv := NewServer(newTestStore(t), db)
	geo := srv.lookupIPGeo("8.8.8.8")
	if geo == nil {
		t.Fatal("expected non-nil geo for 8.8.8.8")
	}
	if geo.Lat == 0 && geo.Lon == 0 {
		t.Error("expected non-zero lat/lon from MMDB lookup")
	}
	if geo.Country == "" {
		t.Error("expected non-empty country from MMDB lookup")
	}
}

func TestLookupIPGeo_WithMMDB_PrivateIP(t *testing.T) {
	db := openTestMMDB(t)
	srv := NewServer(newTestStore(t), db)
	geo := srv.lookupIPGeo("192.168.1.1")
	if geo == nil {
		t.Fatal("expected non-nil fallback geo for private IP")
	}
}

func TestResolveInitialGeo_BBoxReturnsNil(t *testing.T) {
	srv := NewServer(newTestStore(t), nil)
	geo := srv.resolveInitialGeo(CriteriaBBox, "")
	if geo != nil {
		t.Errorf("expected nil geo for bbox task, got %+v", geo)
	}
}

func TestGenerateRandomBidRequestWithConfig_TestMode(t *testing.T) {
	config := DefaultConfig
	config.TestMode = true
	req := GenerateRandomBidRequestWithConfig("site", "banner", config)
	if req.Test != 1 {
		t.Errorf("expected Test=1 in test mode, got %d", req.Test)
	}
}

// TestIP_InitialGeoFromMMDB verifies that when a task uses criteria_type=ip and an
// MMDB is configured, generated locations for the first tick are within 1 km of
// the MMDB coordinates for the given IP.
func TestIP_InitialGeoFromMMDB(t *testing.T) {
	db := openTestMMDB(t)
	store := newTestStore(t)
	outDir := t.TempDir()
	srv := NewServer(store, db)
	sc := NewScheduler(store, outDir, 5*time.Minute, db, nil, nil, nil)

	// Look up expected coordinates directly from the MMDB.
	record, err := db.City(net.ParseIP("8.8.8.8"))
	if err != nil || (record.Location.Latitude == 0 && record.Location.Longitude == 0) {
		t.Skip("8.8.8.8 not found in MMDB")
	}
	expectedLat := record.Location.Latitude
	expectedLon := record.Location.Longitude

	now := time.Now()
	task := &Task{
		CorrelationID: "ip-mmdb-test",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIP,
		IPAddress:     "8.8.8.8",
		Count:         10,
		LastGeo:       srv.resolveInitialGeo(CriteriaIP, "8.8.8.8"),
	}
	store.Add(task)

	sc.run(now)

	// All generated locations must be within 1 km of the MMDB coordinates.
	entries := readJSONLEntries(t, outDir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 output file, got %d", len(entries))
	}
	data, _ := os.ReadFile(outDir + "/" + entries[0].Name())
	for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
		var req BidRequest
		if err := json.Unmarshal(line, &req); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		dist := geoDistanceKm(expectedLat, expectedLon, req.Device.Geo.Lat, req.Device.Geo.Lon)
		if dist > 1.0 {
			t.Errorf("location (%.4f, %.4f) is %.3f km from MMDB point (%.4f, %.4f), want <= 1 km",
				req.Device.Geo.Lat, req.Device.Geo.Lon, dist, expectedLat, expectedLon)
		}
	}
}

// TestIP_ConsecutiveLocationsWithin2km verifies that for an IP task with persistent devices:
// - Total requests per tick == count.
// - Each IFA appears at most count/10 times per tick.
// - Consecutive appearances of the same IFA within a tick are within 1 km of each other.
// - For IFAs that appear in both ticks, the last tick-1 position is within 1 km of the first tick-2 position.
// - All tick-1 first-appearance positions are within 1 km of the IP anchor.
func TestIP_ConsecutiveLocationsWithin2km(t *testing.T) {
	store := newTestStore(t)
	outDir := t.TempDir()
	srv := NewServer(store, nil)
	sc := NewScheduler(store, outDir, 5*time.Minute, nil, nil, nil, nil)

	const count = 20
	now := time.Now()
	anchor := srv.resolveInitialGeo(CriteriaIP, "8.8.8.8")
	task := &Task{
		CorrelationID: "ip-proximity",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIP,
		IPAddress:     "8.8.8.8",
		Count:         count,
		LastGeo:       anchor,
	}
	store.Add(task)

	sc.run(now)
	sc.run(now.Add(5 * time.Minute))

	entries := readJSONLEntries(t, outDir)
	if len(entries) != 2 {
		t.Fatalf("expected 2 output files, got %d", len(entries))
	}

	// readOrdered returns an ordered slice of (IFA, Geo) pairs from a JSONL file.
	type ifaGeo struct {
		ifa string
		geo *Geo
	}
	readOrdered := func(filename string) []ifaGeo {
		var result []ifaGeo
		data, _ := os.ReadFile(filename)
		for _, line := range bytes.Split(bytes.TrimSpace(data), []byte("\n")) {
			var req BidRequest
			if err := json.Unmarshal(line, &req); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			result = append(result, ifaGeo{req.Device.IFA, req.Device.Geo})
		}
		return result
	}

	tick1 := readOrdered(outDir + "/" + entries[0].Name())
	tick2 := readOrdered(outDir + "/" + entries[1].Name())

	maxPerDevice := count / 10
	if maxPerDevice < 1 {
		maxPerDevice = 1
	}

	checkTick := func(tickName string, records []ifaGeo) map[string][]ifaGeo {
		if len(records) != count {
			t.Errorf("%s: expected %d records, got %d", tickName, count, len(records))
		}
		byIFA := make(map[string][]ifaGeo)
		for _, r := range records {
			byIFA[r.ifa] = append(byIFA[r.ifa], r)
		}
		for ifa, appearances := range byIFA {
			if len(appearances) > maxPerDevice {
				t.Errorf("%s: IFA %s appears %d times, want <= %d", tickName, ifa, len(appearances), maxPerDevice)
			}
			// Consecutive appearances within tick must be within 1 km.
			for i := 1; i < len(appearances); i++ {
				d := geoDistanceKm(appearances[i-1].geo.Lat, appearances[i-1].geo.Lon,
					appearances[i].geo.Lat, appearances[i].geo.Lon)
				if d > 1.0 {
					t.Errorf("%s: IFA %s consecutive appearances %d→%d distance %.3f km, want <= 1 km",
						tickName, ifa, i-1, i, d)
				}
			}
		}
		return byIFA
	}

	byIFA1 := checkTick("tick1", tick1)
	byIFA2 := checkTick("tick2", tick2)

	// All tick-1 first appearances must be within 1 km of the anchor.
	const radiusKm = 1.0
	for ifa, appearances := range byIFA1 {
		first := appearances[0].geo
		if d := geoDistanceKm(first.Lat, first.Lon, anchor.Lat, anchor.Lon); d > radiusKm {
			t.Errorf("tick1: IFA %s first appearance %.3f km from anchor, want <= 1 km", ifa, d)
		}
	}

	// For IFAs appearing in both ticks: last tick-1 position → first tick-2 position must be within 1 km.
	for ifa, app1 := range byIFA1 {
		app2, ok := byIFA2[ifa]
		if !ok {
			continue // device may be idle in tick 2 — allowed
		}
		last1 := app1[len(app1)-1].geo
		first2 := app2[0].geo
		if d := geoDistanceKm(last1.Lat, last1.Lon, first2.Lat, first2.Lon); d > 1.0 {
			t.Errorf("IFA %s: tick1-last→tick2-first distance %.3f km, want <= 1 km", ifa, d)
		}
	}
}

// ---- createZip tests ----

func TestCreateZip_Success(t *testing.T) {
	dir := t.TempDir()
	f1 := dir + "/a.jsonl"
	f2 := dir + "/b.jsonl"
	os.WriteFile(f1, []byte(`{"id":"1"}`), 0644)
	os.WriteFile(f2, []byte(`{"id":"2"}`), 0644)

	zipPath := dir + "/out.zip"
	if err := createZip(zipPath, f1, f2); err != nil {
		t.Fatalf("createZip: %v", err)
	}
	info, err := os.Stat(zipPath)
	if err != nil {
		t.Fatalf("zip not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("zip file is empty")
	}
}

func TestCreateZip_MissingSourceFile(t *testing.T) {
	dir := t.TempDir()
	zipPath := dir + "/out.zip"
	err := createZip(zipPath, "/nonexistent/file.jsonl")
	if err == nil {
		t.Error("expected error for missing source file")
	}
	// Partial zip should be cleaned up.
	if _, statErr := os.Stat(zipPath); statErr == nil {
		t.Error("partial zip should have been deleted on error")
	}
}

func TestCreateZip_UnwritableDestination(t *testing.T) {
	dir := t.TempDir()
	f := dir + "/data.jsonl"
	os.WriteFile(f, []byte("x"), 0644)
	err := createZip("/nonexistent/dir/out.zip", f)
	if err == nil {
		t.Error("expected error for unwritable destination")
	}
}

// ---- uploadAndClean / grouping tests ----

// TestUploadAndClean_SFTPFails_FilesKeptLocally verifies that local files are
// preserved when the SFTP upload fails.
func TestUploadAndClean_SFTPFails_FilesKeptLocally(t *testing.T) {
	dir := t.TempDir()
	jsonlPath := dir + "/task_x_1.jsonl"
	zipPath := dir + "/out.zip"
	os.WriteFile(jsonlPath, []byte("{\"id\":\"1\"}\n"), 0644)
	os.WriteFile(zipPath, []byte("zip"), 0644)

	// Port 1 will always refuse connections.
	badSFTP := &SFTPConfig{Host: "127.0.0.1", Port: 1, User: "u", Password: "p"}
	sc := NewScheduler(newTestStore(t), dir, 5*time.Minute, nil, nil, nil, nil)
	sc.uploadAndClean(zipPath, []string{jsonlPath}, badSFTP)

	if _, err := os.Stat(zipPath); err != nil {
		t.Error("zip should be kept after failed upload")
	}
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Error("jsonl should be kept after failed upload")
	}
}

// TestRun_NoSFTP_ZipKeptLocally verifies that when no SFTP is configured the
// zip is created locally and not deleted.
func TestRun_NoSFTP_ZipKeptLocally(t *testing.T) {
	store := newTestStore(t)
	outDir := t.TempDir()
	sc := NewScheduler(store, outDir, 5*time.Minute, nil, nil, nil, nil)

	now := time.Now()
	store.Add(&Task{
		CorrelationID: "local-task",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIFA,
		IFA:           "ifa-local",
		Count:         2,
		LastGeo:       &Geo{Lat: 51.5, Lon: -0.1},
	})
	sc.run(now)

	entries, _ := os.ReadDir(outDir)
	var zips, jsonls int
	for _, e := range entries {
		switch {
		case strings.HasSuffix(e.Name(), ".zip"):
			zips++
		case strings.HasSuffix(e.Name(), ".jsonl"):
			jsonls++
		}
	}
	if zips != 1 {
		t.Errorf("expected 1 zip, got %d", zips)
	}
	if jsonls != 1 {
		t.Errorf("expected 1 jsonl, got %d", jsonls)
	}
}

// TestRun_SFTPGrouping verifies that tasks are grouped by SFTP config:
// tasks sharing the same SFTP produce one zip, tasks with a different SFTP
// produce a separate zip, and tasks with no SFTP produce a local-only zip.
func TestRun_SFTPGrouping(t *testing.T) {
	store := newTestStore(t)
	outDir := t.TempDir()
	badSFTP := &SFTPConfig{Host: "127.0.0.1", Port: 1, User: "u", Password: "p"}
	// No global default — tasks will each carry their own SFTP or none.
	sc := NewScheduler(store, outDir, 5*time.Minute, nil, nil, nil, nil)

	now := time.Now()
	sftpA := &SFTPConfig{Host: "sftp-a.example.com", Port: 22, User: "u", Password: "p", Dir: "/"}
	sftpB := &SFTPConfig{Host: "sftp-b.example.com", Port: 22, User: "u", Password: "p", Dir: "/"}
	_ = badSFTP

	addIFATask := func(id, ifa string, sftp *SFTPConfig) {
		store.Add(&Task{
			CorrelationID: id,
			StartTime:     now.Add(-time.Hour),
			EndTime:       now.Add(time.Hour),
			CriteriaType:  CriteriaIFA,
			IFA:           ifa,
			Count:         1,
			LastGeo:       &Geo{Lat: 51.5, Lon: -0.1},
			SFTP:          sftp,
		})
	}

	// Two tasks on sftp-a → one zip for sftp-a.
	addIFATask("task-a1", "ifa-a1", sftpA)
	addIFATask("task-a2", "ifa-a2", sftpA)
	// One task on sftp-b → separate zip.
	addIFATask("task-b1", "ifa-b1", sftpB)
	// One task with no SFTP → local zip.
	addIFATask("task-local", "ifa-local", nil)

	sc.run(now)

	// Count zip files produced (SFTP uploads will fail since hosts are fake,
	// so all zips stay locally — we can count them).
	entries, _ := os.ReadDir(outDir)
	var zips []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".zip") {
			zips = append(zips, e.Name())
		}
	}
	// Expect 3 zips: one for sftp-a (2 tasks), one for sftp-b (1 task), one local.
	if len(zips) != 3 {
		t.Errorf("expected 3 zip files (one per SFTP group + local), got %d: %v", len(zips), zips)
	}
}

// ---- TimezoneClient tests ----

func TestTimezoneClient_NilReceiver(t *testing.T) {
	var c *TimezoneClient
	if got := c.Timezone(0, 0); got != "" {
		t.Errorf("expected empty string from nil receiver, got %q", got)
	}
}

func TestTimezoneClient_SuccessAndCache(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]string{"timezone": "America/New_York"})
	}))
	defer srv.Close()

	c := NewTimezoneClient(srv.URL)
	tz1 := c.Timezone(40.71, -74.01)
	tz2 := c.Timezone(40.71, -74.01) // same rounded coords → cache hit

	if tz1 != "America/New_York" {
		t.Errorf("expected America/New_York, got %q", tz1)
	}
	if tz2 != tz1 {
		t.Errorf("expected cached result %q, got %q", tz1, tz2)
	}
	if calls != 1 {
		t.Errorf("expected 1 HTTP call (second should hit cache), got %d", calls)
	}
}

func TestTimezoneClient_DifferentCoordsNoCacheHit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]string{"timezone": "Europe/London"})
	}))
	defer srv.Close()

	c := NewTimezoneClient(srv.URL)
	c.Timezone(51.50, -0.10)
	c.Timezone(51.60, -0.20) // different rounded coords → new request
	if calls != 2 {
		t.Errorf("expected 2 HTTP calls for different coords, got %d", calls)
	}
}

func TestTimezoneClient_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewTimezoneClient(srv.URL)
	if got := c.Timezone(40.71, -74.01); got != "" {
		t.Errorf("expected empty string on server error, got %q", got)
	}
}

func TestTimezoneClient_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewTimezoneClient(srv.URL)
	if got := c.Timezone(40.71, -74.01); got != "" {
		t.Errorf("expected empty string on invalid JSON, got %q", got)
	}
}

func TestTimezoneClient_UnreachableServer(t *testing.T) {
	c := NewTimezoneClient("http://127.0.0.1:1")
	if got := c.Timezone(40.71, -74.01); got != "" {
		t.Errorf("expected empty string for unreachable server, got %q", got)
	}
}

// ---- enrichGeo tests ----

func TestEnrichGeo_NilGeo(t *testing.T) {
	sc := NewScheduler(newTestStore(t), t.TempDir(), 5*time.Minute, nil, nil, nil, nil)
	if got := sc.enrichGeo(nil, time.Now()); got != nil {
		t.Errorf("expected nil for nil geo input, got %+v", got)
	}
}

func TestEnrichGeo_NoTzClient(t *testing.T) {
	sc := NewScheduler(newTestStore(t), t.TempDir(), 5*time.Minute, nil, nil, nil, nil)
	geo := &Geo{Lat: 51.5, Lon: -0.1}
	got := sc.enrichGeo(geo, time.Now())
	if got == nil {
		t.Fatal("expected non-nil geo")
	}
	if got.UTCOffset != 0 {
		t.Errorf("expected UTCOffset=0 with no tzClient, got %d", got.UTCOffset)
	}
}

func TestEnrichGeo_TzClientSetsUTCOffset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"timezone": "America/New_York"})
	}))
	defer srv.Close()

	sc := NewScheduler(newTestStore(t), t.TempDir(), 5*time.Minute, nil, nil, NewTimezoneClient(srv.URL), nil)
	geo := &Geo{Lat: 40.7, Lon: -74.0}
	ts := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC) // January → EST (UTC-5)

	got := sc.enrichGeo(geo, ts)
	if got == nil {
		t.Fatal("expected non-nil geo")
	}
	loc, _ := time.LoadLocation("America/New_York")
	_, expectedSecs := ts.In(loc).Zone()
	if got.UTCOffset != expectedSecs/60 {
		t.Errorf("expected UTCOffset=%d, got %d", expectedSecs/60, got.UTCOffset)
	}
}

func TestEnrichGeo_TzClientEmptyTz(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"timezone": ""})
	}))
	defer srv.Close()

	sc := NewScheduler(newTestStore(t), t.TempDir(), 5*time.Minute, nil, nil, NewTimezoneClient(srv.URL), nil)
	geo := &Geo{Lat: 40.7, Lon: -74.0}
	got := sc.enrichGeo(geo, time.Now())
	if got == nil {
		t.Fatal("expected non-nil geo")
	}
	if got.UTCOffset != 0 {
		t.Errorf("expected UTCOffset=0 for empty timezone, got %d", got.UTCOffset)
	}
}

func TestEnrichGeo_TzClientInvalidTz(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"timezone": "INVALID_TIMEZONE"})
	}))
	defer srv.Close()

	sc := NewScheduler(newTestStore(t), t.TempDir(), 5*time.Minute, nil, nil, NewTimezoneClient(srv.URL), nil)
	geo := &Geo{Lat: 40.7, Lon: -74.0}
	got := sc.enrichGeo(geo, time.Now())
	if got == nil {
		t.Fatal("expected non-nil geo")
	}
	if got.UTCOffset != 0 {
		t.Errorf("expected UTCOffset=0 for invalid timezone name, got %d", got.UTCOffset)
	}
}

func TestEnrichGeo_OriginalGeoUnmodified(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"timezone": "America/New_York"})
	}))
	defer srv.Close()

	sc := NewScheduler(newTestStore(t), t.TempDir(), 5*time.Minute, nil, nil, NewTimezoneClient(srv.URL), nil)
	original := &Geo{Lat: 40.7, Lon: -74.0, UTCOffset: 0}
	got := sc.enrichGeo(original, time.Now())

	if got == original {
		t.Error("enrichGeo should return a new Geo, not mutate the original")
	}
	if original.UTCOffset != 0 {
		t.Error("enrichGeo must not modify the original Geo")
	}
}

// TestExtTS_MatchesDeviceGeoTimezone verifies that ext.ts is formatted in the
// same timezone as device.geo.utcoffset for both ifa and ip/bbox task types.
func TestExtTS_MatchesDeviceGeoTimezone(t *testing.T) {
	// tzf-server mock: New York → UTC-5 in January (EST)
	tzSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"timezone": "America/New_York"})
	}))
	defer tzSrv.Close()

	loc, _ := time.LoadLocation("America/New_York")

	for _, criteriaType := range []CriteriaType{CriteriaIFA, CriteriaIP} {
		t.Run(string(criteriaType), func(t *testing.T) {
			store := newTestStore(t)
			outDir := t.TempDir()
			tzClient := NewTimezoneClient(tzSrv.URL)
			sc := NewScheduler(store, outDir, 5*time.Minute, nil, nil, tzClient, nil)

			anchor := &Geo{Lat: 40.7128, Lon: -74.0060}
			now := time.Now()
			task := &Task{
				CorrelationID: "tz-test-" + string(criteriaType),
				StartTime:     now.Add(-time.Hour),
				EndTime:       now.Add(time.Hour),
				CriteriaType:  criteriaType,
				IPAddress:     "8.8.8.8",
				IFA:           "38400000-8cf0-11bd-b23e-10b96e40000d",
				Count:         3,
				LastGeo:       anchor,
			}
			store.Add(task)

			filename, err := sc.generateForTask(task, now)
			if err != nil {
				t.Fatalf("generateForTask: %v", err)
			}

			data, _ := os.ReadFile(filename)
			for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
				var req struct {
					Device struct {
						Geo struct {
							UTCOffset int `json:"utcoffset"`
						} `json:"geo"`
					} `json:"device"`
					Ext map[string]any `json:"ext"`
				}
				if err := json.Unmarshal([]byte(line), &req); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}

				tsStr, ok := req.Ext["ts"].(string)
				if !ok {
					t.Fatal("ext.ts should be a string")
				}
				ts, err := time.Parse(tsFormat, tsStr)
				if err != nil {
					t.Fatalf("parse ext.ts %q: %v", tsStr, err)
				}

				// UTC offset in ext.ts must match device.geo.utcoffset.
				_, tsOffsetSecs := ts.Zone()
				if tsOffsetSecs/60 != req.Device.Geo.UTCOffset {
					t.Errorf("ext.ts offset %d min != device.geo.utcoffset %d min",
						tsOffsetSecs/60, req.Device.Geo.UTCOffset)
				}

				// The offset must be the correct DST-aware value for New York at ts.
				_, expectedSecs := ts.In(loc).Zone()
				if tsOffsetSecs != expectedSecs {
					t.Errorf("ext.ts offset %ds != expected New York offset %ds at %s",
						tsOffsetSecs, expectedSecs, ts)
				}
			}
		})
	}
}

