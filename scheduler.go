package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

const schedulerInterval = 5 * time.Minute

type Scheduler struct {
	store  *TaskStore
	outDir string
	stop   chan struct{}
}

func NewScheduler(store *TaskStore, outDir string) *Scheduler {
	return &Scheduler{
		store:  store,
		outDir: outDir,
		stop:   make(chan struct{}),
	}
}

// Start runs the scheduler loop. Call in a goroutine.
func (sc *Scheduler) Start() {
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()
	log.Printf("scheduler started: interval=%s out_dir=%s", schedulerInterval, sc.outDir)
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
			log.Printf("task %s: generation error: %v", task.ID, err)
		}
	}
}

func (sc *Scheduler) generateForTask(task *Task, now time.Time) error {
	if err := os.MkdirAll(sc.outDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	filename := filepath.Join(sc.outDir,
		fmt.Sprintf("task_%s_%d.jsonl", task.ID, now.Unix()))

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	windowStart := now.Add(-schedulerInterval)
	w := bufio.NewWriter(f)

	for i := 0; i < task.Count; i++ {
		req := generateRequestForTask(task, windowStart, now)
		line, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		w.Write(line)
		w.WriteByte('\n')
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	log.Printf("task %s: wrote %d requests -> %s", task.ID, task.Count, filename)
	return nil
}
