package http

import (
	"encoding/json"
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

func setJSONHeaders(w nethttp.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func writeError(w nethttp.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
