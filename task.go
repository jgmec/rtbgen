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

// BoundingBox is used internally by the generator. Not exposed in the API.
type BoundingBox struct {
	MaxLat float64
	MaxLon float64
	MinLat float64
	MinLon float64
}

// GeoJSONGeometry is a GeoJSON geometry object (https://geojson.org).
// Only the Polygon type is supported for bbox criteria.
// Coordinates follow the GeoJSON spec: [longitude, latitude].
type GeoJSONGeometry struct {
	Type        string           `json:"type"`
	Coordinates [][][2]float64   `json:"coordinates"`
}

// bbox extracts the axis-aligned bounding box from a GeoJSON Polygon.
func (g *GeoJSONGeometry) bbox() (*BoundingBox, error) {
	if g.Type != "Polygon" {
		return nil, fmt.Errorf("unsupported geometry type %q, expected Polygon", g.Type)
	}
	if len(g.Coordinates) == 0 || len(g.Coordinates[0]) == 0 {
		return nil, fmt.Errorf("polygon has no coordinates")
	}
	ring := g.Coordinates[0] // exterior ring; GeoJSON [lon, lat]
	minLon, maxLon := ring[0][0], ring[0][0]
	minLat, maxLat := ring[0][1], ring[0][1]
	for _, pt := range ring[1:] {
		if pt[0] < minLon {
			minLon = pt[0]
		}
		if pt[0] > maxLon {
			maxLon = pt[0]
		}
		if pt[1] < minLat {
			minLat = pt[1]
		}
		if pt[1] > maxLat {
			maxLat = pt[1]
		}
	}
	return &BoundingBox{MaxLat: maxLat, MaxLon: maxLon, MinLat: minLat, MinLon: minLon}, nil
}

type Task struct {
	CorrelationID string           `json:"correlation_id"`
	StartTime     time.Time        `json:"start_time"`
	EndTime       time.Time        `json:"end_time"`
	CriteriaType  CriteriaType     `json:"criteria_type"`
	IPAddress     string           `json:"ip_address,omitempty"`
	IFA           string           `json:"ifa,omitempty"`
	Geometry      *GeoJSONGeometry `json:"geometry,omitempty"`
	LastGeo       *Geo             `json:"last_geo,omitempty"`
	Count         int              `json:"count"`
	Status        TaskStatus       `json:"status"`
	CreatedAt     time.Time        `json:"created_at"`
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
	s.tasks[task.CorrelationID] = task
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
		s.tasks[t.CorrelationID] = t
	}
	return nil
}
