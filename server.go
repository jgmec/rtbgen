package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/oschwald/geoip2-golang"
)

type Server struct {
	store *TaskStore
	mmdb  *geoip2.Reader
}

func NewServer(store *TaskStore, mmdb *geoip2.Reader) *Server {
	return &Server{store: store, mmdb: mmdb}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", s.createTask)
	mux.HandleFunc("GET /tasks", s.listTasks)
	mux.HandleFunc("GET /tasks/{id}", s.getTask)
	mux.HandleFunc("DELETE /tasks/{id}", s.deleteTask)
	return mux
}

type CreateTaskRequest struct {
	CorrelationID string           `json:"correlation_id"`
	StartTime     time.Time        `json:"start_time"`
	EndTime       time.Time        `json:"end_time"`
	CriteriaType  CriteriaType     `json:"criteria_type"`
	IPAddress     string           `json:"ip_address,omitempty"`
	IFA           string           `json:"ifa,omitempty"`
	Geometry      *GeoJSONGeometry `json:"geometry,omitempty"`
	Count         int              `json:"count"`
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}
	if err := validateCreateTaskRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	task := &Task{
		CorrelationID: req.CorrelationID,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		CriteriaType:  req.CriteriaType,
		IPAddress:     req.IPAddress,
		IFA:           req.IFA,
		Geometry:      req.Geometry,
		Count:         req.Count,
		CreatedAt:     time.Now(),
		LastGeo:       s.resolveInitialGeo(req.CriteriaType, req.IPAddress),
	}
	if err := s.store.Add(task); err != nil {
		http.Error(w, fmt.Sprintf("failed to save task: %v", err), http.StatusInternalServerError)
		return
	}
	task.Status = computeStatus(task, time.Now())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func validateCreateTaskRequest(req CreateTaskRequest) error {
	if req.CorrelationID == "" {
		return fmt.Errorf("correlation_id is required")
	}
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		return fmt.Errorf("start_time and end_time are required")
	}
	if !req.EndTime.After(req.StartTime) {
		return fmt.Errorf("end_time must be after start_time")
	}
	if req.Count <= 0 {
		return fmt.Errorf("count must be positive")
	}
	switch req.CriteriaType {
	case CriteriaIP:
		if req.IPAddress == "" {
			return fmt.Errorf("ip_address required for criteria_type=ip")
		}
	case CriteriaIFA:
		if req.IFA == "" {
			return fmt.Errorf("ifa required for criteria_type=ifa")
		}
	case CriteriaBBox:
		if req.Geometry == nil {
			return fmt.Errorf("geometry required for criteria_type=bbox")
		}
		if _, err := req.Geometry.bbox(); err != nil {
			return fmt.Errorf("invalid geometry: %w", err)
		}
	default:
		return fmt.Errorf("criteria_type must be one of: ip, ifa, bbox")
	}
	return nil
}

func (s *Server) resolveInitialGeo(criteriaType CriteriaType, ipAddress string) *Geo {
	switch criteriaType {
	case CriteriaIFA:
		return generateGeo()
	case CriteriaIP:
		return s.lookupIPGeo(ipAddress)
	}
	return nil
}

func (s *Server) lookupIPGeo(ipStr string) *Geo {
	if s.mmdb != nil {
		ip := net.ParseIP(ipStr)
		if ip != nil {
			if record, err := s.mmdb.City(ip); err == nil &&
				(record.Location.Latitude != 0 || record.Location.Longitude != 0) {
				return &Geo{
					Lat:     record.Location.Latitude,
					Lon:     record.Location.Longitude,
					Country: record.Country.IsoCode,
					City:    record.City.Names["en"],
					Type:    2,
				}
			}
		}
	}
	return generateGeo()
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.store.Delete(id) {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks := s.store.List()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, ok := s.store.Get(id)
	if !ok {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}
