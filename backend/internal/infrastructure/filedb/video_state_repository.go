package filedb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"bball_statsman_backend/internal/domain"
)

type VideoStateRepository struct {
	path string
	mu   sync.Mutex
}

type dbPayload struct {
	Videos map[string]domain.VideoState `json:"videos"`
}

func NewVideoStateRepository(path string) *VideoStateRepository {
	return &VideoStateRepository{path: path}
}

func (r *VideoStateRepository) InitSchema(ctx context.Context) error {
	_ = ctx
	if err := os.MkdirAll(filepath.Dir(r.path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(r.path); os.IsNotExist(err) {
		return r.write(dbPayload{Videos: map[string]domain.VideoState{}})
	}
	return nil
}

func (r *VideoStateRepository) Save(ctx context.Context, state domain.VideoState) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	db, err := r.read()
	if err != nil {
		return err
	}
	if db.Videos == nil {
		db.Videos = map[string]domain.VideoState{}
	}
	db.Videos[state.URL] = state
	return r.write(db)
}

func (r *VideoStateRepository) GetByURL(ctx context.Context, url string) (*domain.VideoState, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	db, err := r.read()
	if err != nil {
		return nil, err
	}
	state, ok := db.Videos[url]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (r *VideoStateRepository) DeleteByURL(ctx context.Context, url string) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	db, err := r.read()
	if err != nil {
		return err
	}
	delete(db.Videos, url)
	return r.write(db)
}

func (r *VideoStateRepository) ListSummaries(ctx context.Context) ([]domain.VideoSummary, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()

	db, err := r.read()
	if err != nil {
		return nil, err
	}
	summaries := make([]domain.VideoSummary, 0, len(db.Videos))
	for _, state := range db.Videos {
		summaries = append(summaries, domain.VideoSummary{
			URL: state.URL, UpdatedAt: state.UpdatedAt, EventsCount: len(state.Events),
		})
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].UpdatedAt > summaries[j].UpdatedAt })
	return summaries, nil
}

func (r *VideoStateRepository) read() (dbPayload, error) {
	raw, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return dbPayload{Videos: map[string]domain.VideoState{}}, nil
		}
		return dbPayload{}, err
	}
	if len(raw) == 0 {
		return dbPayload{Videos: map[string]domain.VideoState{}}, nil
	}
	var payload dbPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return dbPayload{}, err
	}
	if payload.Videos == nil {
		payload.Videos = map[string]domain.VideoState{}
	}
	return payload, nil
}

func (r *VideoStateRepository) write(payload dbPayload) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, raw, 0o644)
}
