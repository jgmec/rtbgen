package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return NewServer(newTestStore(t))
}

func TestCreateTask(t *testing.T) {
	srv := newTestServer(t)
	now := time.Now()

	body, _ := json.Marshal(CreateTaskRequest{
		CorrelationID: "corr-1",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIP,
		IPAddress:     "1.2.3.4",
		Count:         5,
	})

	r := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want %d — body: %s", w.Code, http.StatusCreated, w.Body)
	}

	var task Task
	if err := json.NewDecoder(w.Body).Decode(&task); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if task.ID == "" {
		t.Error("task ID should not be empty")
	}
	if task.Status != TaskStatusActive {
		t.Errorf("status: got %q, want %q", task.Status, TaskStatusActive)
	}
}

func TestCreateTask_ValidationErrors(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		req  CreateTaskRequest
	}{
		{
			"missing correlation_id",
			CreateTaskRequest{StartTime: now, EndTime: now.Add(time.Hour), CriteriaType: CriteriaIP, IPAddress: "1.2.3.4", Count: 1},
		},
		{
			"end before start",
			CreateTaskRequest{CorrelationID: "c", StartTime: now.Add(time.Hour), EndTime: now, CriteriaType: CriteriaIP, IPAddress: "1.2.3.4", Count: 1},
		},
		{
			"zero count",
			CreateTaskRequest{CorrelationID: "c", StartTime: now, EndTime: now.Add(time.Hour), CriteriaType: CriteriaIP, IPAddress: "1.2.3.4", Count: 0},
		},
		{
			"ip criteria missing ip",
			CreateTaskRequest{CorrelationID: "c", StartTime: now, EndTime: now.Add(time.Hour), CriteriaType: CriteriaIP, Count: 1},
		},
		{
			"ifa criteria missing ifa",
			CreateTaskRequest{CorrelationID: "c", StartTime: now, EndTime: now.Add(time.Hour), CriteriaType: CriteriaIFA, Count: 1},
		},
		{
			"bbox criteria missing bbox",
			CreateTaskRequest{CorrelationID: "c", StartTime: now, EndTime: now.Add(time.Hour), CriteriaType: CriteriaBBox, Count: 1},
		},
		{
			"invalid criteria type",
			CreateTaskRequest{CorrelationID: "c", StartTime: now, EndTime: now.Add(time.Hour), CriteriaType: "unknown", Count: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(t)
			body, _ := json.Marshal(tt.req)
			r := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Errorf("got %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestCreateTask_InvalidJSON(t *testing.T) {
	srv := newTestServer(t)
	r := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader([]byte("not-json")))
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("got %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListTasks(t *testing.T) {
	srv := newTestServer(t)
	now := time.Now()

	// add two tasks via the handler
	for range 2 {
		body, _ := json.Marshal(CreateTaskRequest{
			CorrelationID: "corr",
			StartTime:     now.Add(-time.Hour),
			EndTime:       now.Add(time.Hour),
			CriteriaType:  CriteriaIFA,
			IFA:           "some-ifa",
			Count:         1,
		})
		r := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
		srv.Handler().ServeHTTP(httptest.NewRecorder(), r)
	}

	r := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", w.Code, http.StatusOK)
	}
	var tasks []Task
	if err := json.NewDecoder(w.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tasks) != 2 {
		t.Errorf("got %d tasks, want 2", len(tasks))
	}
}

func TestGetTask(t *testing.T) {
	srv := newTestServer(t)
	now := time.Now()

	body, _ := json.Marshal(CreateTaskRequest{
		CorrelationID: "corr",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaBBox,
		BoundingBox:   &BoundingBox{MaxLat: 51, MaxLon: 1, MinLat: 50, MinLon: 0},
		Count:         3,
	})
	rCreate := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
	wCreate := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wCreate, rCreate)

	var created Task
	json.NewDecoder(wCreate.Body).Decode(&created)

	r := httptest.NewRequest(http.MethodGet, "/tasks/"+created.ID, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want %d", w.Code, http.StatusOK)
	}
	var got Task
	json.NewDecoder(w.Body).Decode(&got)
	if got.ID != created.ID {
		t.Errorf("got ID %q, want %q", got.ID, created.ID)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	srv := newTestServer(t)
	r := httptest.NewRequest(http.MethodGet, "/tasks/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestDeleteTask(t *testing.T) {
	srv := newTestServer(t)
	now := time.Now()

	body, _ := json.Marshal(CreateTaskRequest{
		CorrelationID: "corr",
		StartTime:     now.Add(-time.Hour),
		EndTime:       now.Add(time.Hour),
		CriteriaType:  CriteriaIP,
		IPAddress:     "1.2.3.4",
		Count:         1,
	})
	wCreate := httptest.NewRecorder()
	srv.Handler().ServeHTTP(wCreate, httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body)))

	var created Task
	json.NewDecoder(wCreate.Body).Decode(&created)

	// Delete it
	r := httptest.NewRequest(http.MethodDelete, "/tasks/"+created.ID, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("got %d, want %d", w.Code, http.StatusNoContent)
	}

	// Confirm gone
	r2 := httptest.NewRequest(http.MethodGet, "/tasks/"+created.ID, nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, r2)
	if w2.Code != http.StatusNotFound {
		t.Errorf("got %d after delete, want %d", w2.Code, http.StatusNotFound)
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	srv := newTestServer(t)
	r := httptest.NewRequest(http.MethodDelete, "/tasks/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d, want %d", w.Code, http.StatusNotFound)
	}
}
