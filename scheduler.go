package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/oschwald/geoip2-golang"
)

// sftpKey returns a deduplication key for an SFTPConfig.
func sftpKey(c *SFTPConfig) string {
	return fmt.Sprintf("%s:%d:%s:%s", c.Host, c.Port, c.User, c.remoteDir())
}

type Scheduler struct {
	store       *TaskStore
	outDir      string
	interval    time.Duration
	mmdb        *geoip2.Reader
	geocoder    *ReverseGeocoder
	tzClient    *TimezoneClient
	defaultSFTP *SFTPConfig
	stop        chan struct{}
}

func NewScheduler(store *TaskStore, outDir string, interval time.Duration, mmdb *geoip2.Reader, geocoder *ReverseGeocoder, tzClient *TimezoneClient, defaultSFTP *SFTPConfig) *Scheduler {
	return &Scheduler{
		store:       store,
		outDir:      outDir,
		interval:    interval,
		mmdb:        mmdb,
		geocoder:    geocoder,
		tzClient:    tzClient,
		defaultSFTP: defaultSFTP,
		stop:        make(chan struct{}),
	}
}

// enrichGeo applies reverse geocoding metadata and, when a timezone client is
// configured, sets geo.UTCOffset from the IANA timezone at the request time t.
func (sc *Scheduler) enrichGeo(geo *Geo, t time.Time) *Geo {
	geo = sc.geocoder.Enrich(geo)
	if geo == nil || sc.tzClient == nil {
		return geo
	}
	tz := sc.tzClient.Timezone(geo.Lat, geo.Lon)
	if tz == "" {
		return geo
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return geo
	}
	out := *geo
	_, offsetSecs := t.In(loc).Zone()
	out.UTCOffset = offsetSecs / 60
	return &out
}

// Start runs the scheduler loop. Call in a goroutine.
func (sc *Scheduler) Start() {
	ticker := time.NewTicker(sc.interval)
	defer ticker.Stop()
	log.Printf("scheduler started: interval=%s out_dir=%s", sc.interval, sc.outDir)
	for {
		select {
		case t := <-ticker.C:
			sc.run(t)
		case <-sc.stop:
			log.Println("scheduler stopped")
			return
		}
	}
}

func (sc *Scheduler) Stop() {
	close(sc.stop)
}

func (sc *Scheduler) run(now time.Time) {
	tasks := sc.store.ActiveAt(now)
	if len(tasks) == 0 {
		return
	}
	log.Printf("scheduler tick: %d active task(s)", len(tasks))

	// Generate JSONL files and group them by their effective SFTP target.
	// Tasks that share the same SFTP config are bundled into one zip.
	// Tasks with no SFTP config (and no global default) are bundled into a
	// local-only zip that is kept on disk.
	type sftpGroup struct {
		cfg   *SFTPConfig // nil = no upload, keep locally
		files []string
	}
	groups := make(map[string]*sftpGroup)

	for _, task := range tasks {
		filename, err := sc.generateForTask(task, now)
		if err != nil {
			log.Printf("task %s: generation error: %v", task.CorrelationID, err)
			continue
		}

		cfg := task.SFTP
		if cfg == nil {
			cfg = sc.defaultSFTP
		}

		key := "local"
		if cfg != nil && cfg.Host != "" {
			key = sftpKey(cfg)
		} else {
			cfg = nil
		}

		if groups[key] == nil {
			groups[key] = &sftpGroup{cfg: cfg}
		}
		groups[key].files = append(groups[key].files, filename)
	}

	// Create one zip per group and upload where configured.
	i := 0
	for _, g := range groups {
		zipPath := filepath.Join(sc.outDir, fmt.Sprintf("output_%d_%d.zip", now.Unix(), i))
		i++
		if err := createZip(zipPath, g.files...); err != nil {
			log.Printf("tick zip error: %v", err)
			continue
		}
		if g.cfg != nil {
			sc.uploadAndClean(zipPath, g.files, g.cfg)
		}
	}
}

// uploadAndClean uploads zipPath to a single SFTP target and removes the zip and
// all JSONL files on success. On failure, all local files are kept intact.
func (sc *Scheduler) uploadAndClean(zipPath string, jsonlPaths []string, cfg *SFTPConfig) {
	if err := uploadSFTP(cfg, zipPath); err != nil {
		log.Printf("sftp upload to %s: %v", cfg.addr(), err)
		return
	}
	log.Printf("sftp upload to %s: ok (%s)", cfg.addr(), filepath.Base(zipPath))
	for _, p := range append(jsonlPaths, zipPath) {
		if err := os.Remove(p); err != nil {
			log.Printf("delete local file %s: %v", p, err)
		}
	}
}

func (sc *Scheduler) generateForTask(task *Task, now time.Time) (string, error) {
	if err := os.MkdirAll(sc.outDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	filename := filepath.Join(sc.outDir,
		fmt.Sprintf("task_%s_%d.jsonl", task.CorrelationID, now.Unix()))

	f, err := os.Create(filename)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	if task.CriteriaType == CriteriaIP || task.CriteriaType == CriteriaBBox {
		if err := sc.generateForDeviceTask(task, w, now); err != nil {
			return "", err
		}
	} else {
		geo := task.LastGeo

		for i := 0; i < task.Count; i++ {
			t := time.Now()
			ts := randomTimestamp(t.Add(-sc.interval), t)
			req := generateRequestForTask(task, ts, nil, "")
			if geo != nil {
				req.Device.Geo = sc.enrichGeo(geo, ts)
			}
			offsetMins := 0
			if req.Device.Geo != nil {
				offsetMins = req.Device.Geo.UTCOffset
			}
			req.Ext.(map[string]any)["ts"] = ts.In(time.FixedZone("", offsetMins*60)).Format(tsFormat)
			line, err := json.Marshal(req)
			if err != nil {
				return "", fmt.Errorf("marshal request: %w", err)
			}
			w.Write(line)
			w.WriteByte('\n')

			// Walk up to 2 km for the next request within this tick.
			if geo != nil {
				geo = generateGeoNear(geo.Lat, geo.Lon, 2.0)
			}
		}

		// Persist the final position for the next tick.
		if geo != nil {
			task.LastGeo = geo
			sc.store.Add(task)
		}
	}

	if err := w.Flush(); err != nil {
		return "", fmt.Errorf("flush: %w", err)
	}

	log.Printf("task %s: wrote %d requests -> %s", task.CorrelationID, task.Count, filename)
	return filename, nil
}

// createZip creates a new zip archive at zipPath containing all provided files.
func createZip(zipPath string, filePaths ...string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}

	zw := zip.NewWriter(out)

	for _, filePath := range filePaths {
		w, err := zw.Create(filepath.Base(filePath))
		if err != nil {
			zw.Close()
			out.Close()
			os.Remove(zipPath)
			return fmt.Errorf("create zip entry: %w", err)
		}
		src, err := os.Open(filePath)
		if err != nil {
			zw.Close()
			out.Close()
			os.Remove(zipPath)
			return fmt.Errorf("open source file: %w", err)
		}
		_, copyErr := io.Copy(w, src)
		src.Close()
		if copyErr != nil {
			zw.Close()
			out.Close()
			os.Remove(zipPath)
			return fmt.Errorf("write zip entry: %w", copyErr)
		}
	}

	if err := zw.Close(); err != nil {
		out.Close()
		os.Remove(zipPath)
		return fmt.Errorf("close zip writer: %w", err)
	}
	return out.Close()
}

// generateForDeviceTask generates count requests across a persistent device pool for IP and bbox tasks.
// Each device may appear 1..count/10 times per tick (randomised). Within a tick a device's location
// walks up to 1 km between its consecutive appearances. The updated device locations are persisted after
// the tick. For bbox tasks, device locations are constrained within the polygon's bounding box.
func (sc *Scheduler) generateForDeviceTask(task *Task, w *bufio.Writer, now time.Time) error {
	// Resolve bounding box for bbox tasks.
	var bb *BoundingBox
	if task.CriteriaType == CriteriaBBox && task.Geometry != nil {
		if b, err := task.Geometry.bbox(); err == nil {
			bb = b
		}
	}

	// Initialize device pool on first tick.
	if len(task.Devices) == 0 {
		task.Devices = make(map[string]*Geo, task.Count)
		if bb != nil {
			for i := 0; i < task.Count; i++ {
				task.Devices[uuid.New().String()] = generateGeoInBBox(bb)
			}
		} else {
			anchor := task.LastGeo
			if anchor == nil {
				anchor = generateGeo()
			}
			for i := 0; i < task.Count; i++ {
				task.Devices[uuid.New().String()] = generateGeoNear(anchor.Lat, anchor.Lon, 1.0)
			}
		}
	}

	maxPerDevice := task.Count / 10
	if maxPerDevice < 1 {
		maxPerDevice = 1
	}

	// Build a shuffled slot list: each IFA appears 1..maxPerDevice times, total = count.
	ifaKeys := make([]string, 0, len(task.Devices))
	for ifa := range task.Devices {
		ifaKeys = append(ifaKeys, ifa)
	}
	rand.Shuffle(len(ifaKeys), func(i, j int) { ifaKeys[i], ifaKeys[j] = ifaKeys[j], ifaKeys[i] })

	slots := make([]string, 0, task.Count)
	for _, ifa := range ifaKeys {
		n := rand.Intn(maxPerDevice) + 1
		for i := 0; i < n && len(slots) < task.Count; i++ {
			slots = append(slots, ifa)
		}
		if len(slots) >= task.Count {
			break
		}
	}
	// Fill any remaining slots from random devices.
	for len(slots) < task.Count {
		slots = append(slots, ifaKeys[rand.Intn(len(ifaKeys))])
	}
	rand.Shuffle(len(slots), func(i, j int) { slots[i], slots[j] = slots[j], slots[i] })

	// Track each device's current position within this tick.
	currentGeo := make(map[string]*Geo, len(task.Devices))
	for ifa, geo := range task.Devices {
		currentGeo[ifa] = geo
	}

	for _, ifa := range slots {
		t := time.Now()
		ts := randomTimestamp(t.Add(-sc.interval), t)
		req := generateRequestForTask(task, ts, nil, ifa)
		req.Device.Geo = sc.enrichGeo(currentGeo[ifa], ts)
		offsetMins := 0
		if req.Device.Geo != nil {
			offsetMins = req.Device.Geo.UTCOffset
		}
		req.Ext.(map[string]any)["ts"] = ts.In(time.FixedZone("", offsetMins*60)).Format(tsFormat)
		line, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		w.Write(line)
		w.WriteByte('\n')
		// Walk the device's location for its next appearance (within this tick or next).
		next := generateGeoNear(currentGeo[ifa].Lat, currentGeo[ifa].Lon, 1.0)
		if bb != nil && (next.Lat < bb.MinLat || next.Lat > bb.MaxLat ||
			next.Lon < bb.MinLon || next.Lon > bb.MaxLon) {
			next = generateGeoInBBox(bb)
		}
		currentGeo[ifa] = next
	}

	// Persist updated device locations.
	for ifa, geo := range currentGeo {
		task.Devices[ifa] = geo
	}
	sc.store.Add(task)
	return nil
}

