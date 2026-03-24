package main

import (
	"os"
	"testing"
	"time"
)

var (
	pastTime   = time.Now().Add(-2 * time.Hour)
	futureTime = time.Now().Add(2 * time.Hour)
)

func TestComputeStatus(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name   string
		start  time.Time
		end    time.Time
		want   TaskStatus
	}{
		{"pending: start in future", now.Add(time.Hour), now.Add(2 * time.Hour), TaskStatusPending},
		{"active: now within window", now.Add(-time.Hour), now.Add(time.Hour), TaskStatusActive},
		{"completed: end in past", now.Add(-2 * time.Hour), now.Add(-time.Hour), TaskStatusCompleted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &Task{StartTime: tt.start, EndTime: tt.end}
			got := computeStatus(task, now)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func newTestStore(t *testing.T) *TaskStore {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "tasks-*.json")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	f.Close()
	store, err := NewTaskStore(f.Name())
	if err != nil {
		t.Fatalf("new task store: %v", err)
	}
	return store
}

func sampleTask() *Task {
	return &Task{
		ID:            randomID(),
		CorrelationID: "corr-1",
		StartTime:     pastTime,
		EndTime:       futureTime,
		CriteriaType:  CriteriaIP,
		IPAddress:     "1.2.3.4",
		Count:         10,
		CreatedAt:     time.Now(),
	}
}

func TestTaskStoreAddAndGet(t *testing.T) {
	store := newTestStore(t)
	task := sampleTask()

	if err := store.Add(task); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, ok := store.Get(task.ID)
	if !ok {
		t.Fatal("Get: task not found")
	}
	if got.ID != task.ID {
		t.Errorf("got ID %q, want %q", got.ID, task.ID)
	}
	if got.Status != TaskStatusActive {
		t.Errorf("got status %q, want %q", got.Status, TaskStatusActive)
	}
}

func TestTaskStoreGetNotFound(t *testing.T) {
	store := newTestStore(t)
	_, ok := store.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestTaskStoreList(t *testing.T) {
	store := newTestStore(t)
	store.Add(sampleTask())
	store.Add(sampleTask())

	tasks := store.List()
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(tasks))
	}
}

func TestTaskStoreDelete(t *testing.T) {
	store := newTestStore(t)
	task := sampleTask()
	store.Add(task)

	if !store.Delete(task.ID) {
		t.Error("Delete: expected true for existing task")
	}
	if _, ok := store.Get(task.ID); ok {
		t.Error("task still present after delete")
	}
	if store.Delete(task.ID) {
		t.Error("Delete: expected false for already-deleted task")
	}
}

func TestTaskStoreActiveAt(t *testing.T) {
	store := newTestStore(t)
	now := time.Now()

	active := &Task{ID: randomID(), StartTime: now.Add(-time.Hour), EndTime: now.Add(time.Hour), Count: 1}
	pending := &Task{ID: randomID(), StartTime: now.Add(time.Hour), EndTime: now.Add(2 * time.Hour), Count: 1}
	done := &Task{ID: randomID(), StartTime: now.Add(-2 * time.Hour), EndTime: now.Add(-time.Hour), Count: 1}

	store.Add(active)
	store.Add(pending)
	store.Add(done)

	got := store.ActiveAt(now)
	if len(got) != 1 {
		t.Fatalf("got %d active tasks, want 1", len(got))
	}
	if got[0].ID != active.ID {
		t.Errorf("wrong active task: got %q, want %q", got[0].ID, active.ID)
	}
}

func TestTaskStorePersistence(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/tasks.json"

	store1, _ := NewTaskStore(path)
	task := sampleTask()
	store1.Add(task)

	// Load from same file
	store2, err := NewTaskStore(path)
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if _, ok := store2.Get(task.ID); !ok {
		t.Error("task not found after reload")
	}
}

func TestTaskStoreLoadCorruptFile(t *testing.T) {
	path := t.TempDir() + "/corrupt.json"
	os.WriteFile(path, []byte("not-valid-json"), 0644)
	_, err := NewTaskStore(path)
	if err == nil {
		t.Error("expected error for corrupt JSON file")
	}
}

func TestTaskStoreReadError(t *testing.T) {
	// A directory path cannot be read as a file.
	path := t.TempDir() + "/dir"
	os.Mkdir(path, 0755)
	_, err := NewTaskStore(path)
	if err == nil {
		t.Error("expected error when path is a directory")
	}
}

func TestTaskStoreSaveError(t *testing.T) {
	dir := t.TempDir()
	store, _ := NewTaskStore(dir + "/tasks.json")
	os.Chmod(dir, 0444)
	defer os.Chmod(dir, 0755)
	if err := store.Add(sampleTask()); err == nil {
		t.Error("expected error when directory is read-only")
	}
}

func TestTaskStoreLoadEmptyFile(t *testing.T) {
	// Non-existent file should not error
	path := t.TempDir() + "/new.json"
	store, err := NewTaskStore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.List()) != 0 {
		t.Error("expected empty store")
	}
}
