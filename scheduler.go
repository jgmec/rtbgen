package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/oschwald/geoip2-golang"
)

type Scheduler struct {
	store    *TaskStore
	outDir   string
	interval time.Duration
	mmdb     *geoip2.Reader
	stop     chan struct{}
}

func NewScheduler(store *TaskStore, outDir string, interval time.Duration, mmdb *geoip2.Reader) *Scheduler {
	return &Scheduler{
		store:    store,
		outDir:   outDir,
		interval: interval,
		mmdb:     mmdb,
		stop:     make(chan struct{}),
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
	for _, task := range tasks {
		if err := sc.generateForTask(task, now); err != nil {
			log.Printf("task %s: generation error: %v", task.CorrelationID, err)
		}
	}
}

func (sc *Scheduler) generateForTask(task *Task, now time.Time) error {
	if err := os.MkdirAll(sc.outDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	filename := filepath.Join(sc.outDir,
		fmt.Sprintf("task_%s_%d.jsonl", task.CorrelationID, now.Unix()))

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	var lastReq *BidRequest
	for i := 0; i < task.Count; i++ {
		t := time.Now()
		lastReq = generateRequestForTask(task, t.Add(-sc.interval), t, task.LastGeo)
		line, err := json.Marshal(lastReq)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		w.Write(line)
		w.WriteByte('\n')
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	// Persist the last generated location so the next tick starts from here.
	if lastReq != nil && lastReq.Device != nil && lastReq.Device.Geo != nil {
		task.LastGeo = lastReq.Device.Geo
		sc.store.Add(task)
	}

	log.Printf("task %s: wrote %d requests -> %s", task.CorrelationID, task.Count, filename)
	return nil
}

