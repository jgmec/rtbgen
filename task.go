package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type TaskStatus string
type CriteriaType string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusActive    TaskStatus = "active"
	TaskStatusCompleted TaskStatus = "completed"
)

func computeStatus(t *Task, now time.Time) TaskStatus {
	if now.Before(t.StartTime) {
		return TaskStatusPending
	}
	if now.After(t.EndTime) {
		return TaskStatusCompleted
	}
	return TaskStatusActive
}

const (
	CriteriaIP   CriteriaType = "ip"
	CriteriaIFA  CriteriaType = "ifa"
	CriteriaBBox CriteriaType = "bbox"
)

type BoundingBox struct {
	MaxLat float64 `json:"max_lat"`
	MaxLon float64 `json:"max_lon"`
	MinLat float64 `json:"min_lat"`
	MinLon float64 `json:"min_lon"`
}

type Task struct {
	ID            string       `json:"id"`
	CorrelationID string       `json:"correlation_id"`
	StartTime     time.Time    `json:"start_time"`
	EndTime       time.Time    `json:"end_time"`
	CriteriaType  CriteriaType `json:"criteria_type"`
	IPAddress     string       `json:"ip_address,omitempty"`
	IFA           string       `json:"ifa,omitempty"`
	BoundingBox   *BoundingBox `json:"bounding_box,omitempty"`
	Count         int          `json:"count"`
	Status        TaskStatus   `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
}

// TaskStore is a thread-safe, JSON-file-backed store for Tasks.
type TaskStore struct {
	mu       sync.RWMutex
	tasks    map[string]*Task
	filePath string
}

func NewTaskStore(filePath string) (*TaskStore, error) {
	store := &TaskStore{
		tasks:    make(map[string]*Task),
		filePath: filePath,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *TaskStore) Add(task *Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	return s.save()
}

func (s *TaskStore) Get(id string) (*Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, false
	}
	copy := *t
	copy.Status = computeStatus(t, time.Now())
	return &copy, true
}

func (s *TaskStore) List() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	tasks := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		copy := *t
		copy.Status = computeStatus(t, now)
		tasks = append(tasks, &copy)
	}
	return tasks
}

// ActiveAt returns tasks whose time window contains now.
func (s *TaskStore) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return false
	}
	delete(s.tasks, id)
	s.save()
	return true
}

func (s *TaskStore) ActiveAt(now time.Time) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var active []*Task
	for _, t := range s.tasks {
		if !now.Before(t.StartTime) && !now.After(t.EndTime) {
			active = append(active, t)
		}
	}
	return active
}

// save must be called with s.mu write lock held.
func (s *TaskStore) save() error {
	tasks := make([]*Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tasks: %w", err)
	}
	return os.WriteFile(s.filePath, data, 0644)
}

func (s *TaskStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read tasks file: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("unmarshal tasks: %w", err)
	}
	for _, t := range tasks {
		s.tasks[t.ID] = t
	}
	return nil
}
