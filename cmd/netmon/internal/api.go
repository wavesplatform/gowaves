package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type NetworkHealth struct {
	Status bool `json:"safe_deposit_withdraw"`
}

type HealthService struct {
	m *NodeMonitor
}

func NewHealthService(m *NodeMonitor) (*HealthService, error) {
	if m == nil {
		return nil, errors.New("empty node monitor")
	}
	return &HealthService{m: m}, nil
}

func (h *HealthService) Health(w http.ResponseWriter, _ *http.Request) {
	r, err := h.m.Health()
	if err != nil {
		http.Error(w, fmt.Sprintf("Internal service failure: %v", err), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal status to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
