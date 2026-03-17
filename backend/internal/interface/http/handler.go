package http

import (
	"encoding/json"
	"fmt"
	nethttp "net/http"

	"bball_statsman_backend/internal/domain"
	"bball_statsman_backend/internal/pubsub"
	"bball_statsman_backend/internal/usecase"
)

type Handler struct {
	uc     *usecase.VideoStateUseCase
	broker *pubsub.Broker
}

func NewHandler(uc *usecase.VideoStateUseCase, broker *pubsub.Broker) *Handler {
	return &Handler{uc: uc, broker: broker}
}

func (h *Handler) Register(mux *nethttp.ServeMux) {
	mux.HandleFunc("/api/videos", h.handleVideos)
	mux.HandleFunc("/api/videos/state", h.handleVideoState)
	mux.HandleFunc("/api/videos/state/version", h.handleVideoStateVersion)
	mux.HandleFunc("/api/videos/ops", h.handleVideoOperations)
	mux.HandleFunc("/api/videos/ops/stream", h.handleVideoOpsStream)
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

// handleVideoOpsStream is an SSE endpoint. Clients subscribe to real-time
// operation broadcasts for a specific video URL.
// GET /api/videos/ops/stream?url=<videoUrl>
func (h *Handler) handleVideoOpsStream(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodGet {
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(nethttp.Flusher)
	if !ok {
		nethttp.Error(w, "streaming not supported", nethttp.StatusInternalServerError)
		return
	}

	videoURL := r.URL.Query().Get("url")
	if videoURL == "" {
		nethttp.Error(w, "url is required", nethttp.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	ch, cancel := h.broker.Subscribe(videoURL)
	defer cancel()

	// Send a keep-alive comment immediately so the client knows the stream is open.
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func setJSONHeaders(w nethttp.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func writeError(w nethttp.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
