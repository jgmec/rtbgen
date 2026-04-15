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
	defaultSFTP *SFTPConfig
	stop        chan struct{}
}

func NewScheduler(store *TaskStore, outDir string, interval time.Duration, mmdb *geoip2.Reader, geocoder *ReverseGeocoder, defaultSFTP *SFTPConfig) *Scheduler {
	return &Scheduler{
		store:       store,
		outDir:      outDir,
		interval:    interval,
		mmdb:        mmdb,
		geocoder:    geocoder,
		defaultSFTP: defaultSFTP,
		stop:        make(chan struct{}),
	}
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
	var generated []string
	for _, task := range tasks {
		filename, err := sc.generateForTask(task, now)
		if err != nil {
			log.Printf("task %s: generation error: %v", task.CorrelationID, err)
			continue
		}
		generated = append(generated, filename)
	}
	if len(generated) > 0 {
		zipPath := filepath.Join(sc.outDir, fmt.Sprintf("output_%d.zip", now.Unix()))
		if err := createZip(zipPath, generated...); err != nil {
			log.Printf("tick zip error: %v", err)
			return
		}
		sc.uploadZip(zipPath, generated, tasks)
	}
}

// uploadZip uploads zipPath to all unique SFTP targets from the tick's tasks (falling back to the
// global default when a task has no SFTP config). Deletes the local zip and JSONL files after all
// uploads succeed.
func (sc *Scheduler) uploadZip(zipPath string, jsonlPaths []string, tasks []*Task) {
	// Collect unique SFTP targets.
	seen := make(map[string]*SFTPConfig)
	for _, t := range tasks {
		cfg := t.SFTP
		if cfg == nil {
			cfg = sc.defaultSFTP
		}
		if cfg != nil && cfg.Host != "" {
			seen[sftpKey(cfg)] = cfg
		}
	}
	if len(seen) == 0 {
		return // no SFTP configured — keep zip locally
	}

	allOK := true
	for _, cfg := range seen {
		if err := uploadSFTP(cfg, zipPath); err != nil {
			log.Printf("sftp upload to %s: %v", cfg.addr(), err)
			allOK = false
		} else {
			log.Printf("sftp upload to %s: ok (%s)", cfg.addr(), filepath.Base(zipPath))
		}
	}
	if allOK {
		for _, p := range append(jsonlPaths, zipPath) {
			if err := os.Remove(p); err != nil {
				log.Printf("delete local file %s: %v", p, err)
			}
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
			req := generateRequestForTask(task, t.Add(-sc.interval), t, nil, "")
			if geo != nil {
				req.Device.Geo = sc.geocoder.Enrich(geo)
			}
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
				task.Devices[randomID()] = generateGeoInBBox(bb)
			}
		} else {
			anchor := task.LastGeo
			if anchor == nil {
				anchor = generateGeo()
			}
			for i := 0; i < task.Count; i++ {
				task.Devices[randomID()] = generateGeoNear(anchor.Lat, anchor.Lon, 1.0)
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
		geo := sc.geocoder.Enrich(currentGeo[ifa])
		t := time.Now()
		req := generateRequestForTask(task, t.Add(-sc.interval), t, nil, ifa)
		req.Device.Geo = geo
		line, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		w.Write(line)
		w.WriteByte('\n')
		// Walk the device's location for its next appearance (within this tick or next).
		next := generateGeoNear(geo.Lat, geo.Lon, 1.0)
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

