package http

import (
	"encoding/json"
	"fmt"
	nethttp "net/http"

	"bball_statsman_backend/internal/domain"
	"bball_statsman_backend/internal/usecase"
)

type Handler struct {
	uc *usecase.VideoStateUseCase
}

func NewHandler(uc *usecase.VideoStateUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Register(mux *nethttp.ServeMux) {
	mux.HandleFunc("/api/videos", h.handleVideos)
	mux.HandleFunc("/api/videos/state", h.handleVideoState)
	mux.HandleFunc("/api/videos/state/version", h.handleVideoStateVersion)
	mux.HandleFunc("/api/videos/ops", h.handleVideoOperations)
}

func (h *Handler) handleVideos(w nethttp.ResponseWriter, r *nethttp.Request) {
	setJSONHeaders(w)

	switch r.Method {
	case nethttp.MethodGet:
		summaries, err := h.uc.ListStates(r.Context())
		if err != nil {
			writeError(w, nethttp.StatusInternalServerError, err.Error())
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"videos": summaries})
	case nethttp.MethodDelete:
		url := r.URL.Query().Get("url")
		if err := h.uc.DeleteState(r.Context(), url); err != nil {
			status := nethttp.StatusInternalServerError
			if err == usecase.ErrInvalidURL {
				status = nethttp.StatusBadRequest
			}
			writeError(w, status, err.Error())
			return
		}
		w.WriteHeader(nethttp.StatusNoContent)
	default:
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleVideoStateVersion(w nethttp.ResponseWriter, r *nethttp.Request) {
	setJSONHeaders(w)
	if r.Method != nethttp.MethodGet {
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	url := r.URL.Query().Get("url")
	version, exists, err := h.uc.GetStateVersion(r.Context(), url)
	if err != nil {
		status := nethttp.StatusInternalServerError
		if err == usecase.ErrInvalidURL {
			status = nethttp.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	if !exists {
		w.WriteHeader(nethttp.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"version": 0, "exists": false})
		return
	}

	etag := fmt.Sprintf(`"%d"`, version)
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(nethttp.StatusNotModified)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"version": version, "exists": true})
}

func (h *Handler) handleVideoState(w nethttp.ResponseWriter, r *nethttp.Request) {
	setJSONHeaders(w)

	switch r.Method {
	case nethttp.MethodGet:
		url := r.URL.Query().Get("url")
		state, err := h.uc.GetState(r.Context(), url)
		if err != nil {
			status := nethttp.StatusInternalServerError
			if err == usecase.ErrInvalidURL {
				status = nethttp.StatusBadRequest
			}
			writeError(w, status, err.Error())
			return
		}
		if state == nil {
			w.WriteHeader(nethttp.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"state": nil})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"state": state})
	case nethttp.MethodPut:
		var payload struct {
			State domain.VideoState `json:"state"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeError(w, nethttp.StatusBadRequest, "invalid json payload")
			return
		}
		if err := h.uc.SaveState(r.Context(), payload.State); err != nil {
			status := nethttp.StatusInternalServerError
			if err == usecase.ErrInvalidURL {
				status = nethttp.StatusBadRequest
			}
			writeError(w, status, err.Error())
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	default:
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleVideoOperations(w nethttp.ResponseWriter, r *nethttp.Request) {
	setJSONHeaders(w)
	if r.Method != nethttp.MethodPost {
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		URL        string                  `json:"url"`
		Operations []domain.VideoOperation `json:"operations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, nethttp.StatusBadRequest, "invalid json payload")
		return
	}

	state, err := h.uc.ApplyOperations(r.Context(), payload.URL, payload.Operations)
	if err != nil {
		status := nethttp.StatusInternalServerError
		if err == usecase.ErrInvalidURL || err == usecase.ErrInvalidOperation {
			status = nethttp.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"state": state})
}

func setJSONHeaders(w nethttp.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func writeError(w nethttp.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
