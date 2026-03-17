package usecase

import (
	"context"
	"testing"

	"bball_statsman_backend/internal/domain"
)

type inMemoryRepo struct {
	states map[string]domain.VideoState
}

func newInMemoryRepo() *inMemoryRepo {
	return &inMemoryRepo{states: map[string]domain.VideoState{}}
}

func (r *inMemoryRepo) Save(_ context.Context, state domain.VideoState) error {
	r.states[state.URL] = state
	return nil
}

func (r *inMemoryRepo) GetByURL(_ context.Context, url string) (*domain.VideoState, error) {
	state, ok := r.states[url]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (r *inMemoryRepo) DeleteByURL(_ context.Context, url string) error {
	delete(r.states, url)
	return nil
}

func (r *inMemoryRepo) ListSummaries(_ context.Context) ([]domain.VideoSummary, error) {
	return nil, nil
}

func TestApplyOperations_EventLifecycle(t *testing.T) {
	repo := newInMemoryRepo()
	uc := NewVideoStateUseCase(repo)
	ctx := context.Background()

	event := domain.Event{ID: "e-1", Type: "assist", VideoTimeSec: 12, PlayerID: "p-1"}
	state, err := uc.ApplyOperations(ctx, "https://vkvideo.ru/video-1_1", []domain.VideoOperation{{Type: "event_upsert", Event: &event}})
	if err != nil {
		t.Fatalf("apply upsert: %v", err)
	}
	if len(state.Events) != 1 || state.Events[0].ID != event.ID {
		t.Fatalf("expected one event after upsert, got %#v", state.Events)
	}
	if state.Version != 1 {
		t.Fatalf("expected version 1, got %d", state.Version)
	}

	state, err = uc.ApplyOperations(ctx, "https://vkvideo.ru/video-1_1", []domain.VideoOperation{{Type: "event_remove", EventID: event.ID}})
	if err != nil {
		t.Fatalf("apply remove: %v", err)
	}
	if len(state.Events) != 0 {
		t.Fatalf("expected no events after remove, got %d", len(state.Events))
	}
	if state.Version != 2 {
		t.Fatalf("expected version 2, got %d", state.Version)
	}
}

func TestApplyOperations_SettingsAndGamesReplace(t *testing.T) {
	repo := newInMemoryRepo()
	uc := NewVideoStateUseCase(repo)
	ctx := context.Background()

	settings := domain.VideoSettings{SelectedGameFilter: "all", SelectedRosterFilter: "all", Teams: []domain.Team{{ID: "t1", Name: "A"}}}
	games := []domain.GameRange{{ID: "g1", StartSec: 20}}
	state, err := uc.ApplyOperations(ctx, "https://vkvideo.ru/video-2_2", []domain.VideoOperation{
		{Type: "settings_replace", Settings: &settings},
		{Type: "games_replace", Games: games},
	})
	if err != nil {
		t.Fatalf("apply settings/games: %v", err)
	}
	if len(state.Games) != 1 || state.Games[0].ID != "g1" {
		t.Fatalf("unexpected games: %#v", state.Games)
	}
	if len(state.Settings.Teams) != 1 || state.Settings.Teams[0].ID != "t1" {
		t.Fatalf("unexpected settings teams: %#v", state.Settings.Teams)
	}
}
