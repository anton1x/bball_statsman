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

func TestGetStateVersion(t *testing.T) {
	repo := newInMemoryRepo()
	uc := NewVideoStateUseCase(repo)
	ctx := context.Background()

	version, exists, err := uc.GetStateVersion(ctx, "https://vkvideo.ru/video-3_3")
	if err != nil {
		t.Fatalf("get missing version: %v", err)
	}
	if exists || version != 0 {
		t.Fatalf("expected missing state with version 0, got exists=%v version=%d", exists, version)
	}

	if err := uc.SaveState(ctx, domain.VideoState{URL: "https://vkvideo.ru/video-3_3"}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	version, exists, err = uc.GetStateVersion(ctx, "https://vkvideo.ru/video-3_3")
	if err != nil {
		t.Fatalf("get version: %v", err)
	}
	if !exists || version != 1 {
		t.Fatalf("expected version 1, got exists=%v version=%d", exists, version)
	}
}

func TestSaveState_DoesNotIncreaseVersionWithoutChanges(t *testing.T) {
	repo := newInMemoryRepo()
	uc := NewVideoStateUseCase(repo)
	ctx := context.Background()

	initial := domain.VideoState{
		URL:    "https://vkvideo.ru/video-4_4",
		Events: []domain.Event{{ID: "e1", Type: "score", VideoTimeSec: 10}},
		Games:  []domain.GameRange{{ID: "g1", StartSec: 1}},
	}
	if err := uc.SaveState(ctx, initial); err != nil {
		t.Fatalf("first save: %v", err)
	}

	first, err := uc.GetState(ctx, initial.URL)
	if err != nil {
		t.Fatalf("get first state: %v", err)
	}

	if err := uc.SaveState(ctx, initial); err != nil {
		t.Fatalf("second save: %v", err)
	}

	second, err := uc.GetState(ctx, initial.URL)
	if err != nil {
		t.Fatalf("get second state: %v", err)
	}

	if second.Version != first.Version {
		t.Fatalf("expected version to stay %d, got %d", first.Version, second.Version)
	}
	if second.UpdatedAt != first.UpdatedAt {
		t.Fatalf("expected updatedAt to stay %d, got %d", first.UpdatedAt, second.UpdatedAt)
	}
}

func TestApplyOperations_DoesNotIncreaseVersionWithoutChanges(t *testing.T) {
	repo := newInMemoryRepo()
	uc := NewVideoStateUseCase(repo)
	ctx := context.Background()
	url := "https://vkvideo.ru/video-5_5"

	event := domain.Event{ID: "e-1", Type: "assist", VideoTimeSec: 12, PlayerID: "p-1"}
	state, err := uc.ApplyOperations(ctx, url, []domain.VideoOperation{{Type: "event_upsert", Event: &event}})
	if err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	next, err := uc.ApplyOperations(ctx, url, []domain.VideoOperation{{Type: "event_upsert", Event: &event}})
	if err != nil {
		t.Fatalf("repeat upsert: %v", err)
	}

	if next.Version != state.Version {
		t.Fatalf("expected version to stay %d, got %d", state.Version, next.Version)
	}
	if next.UpdatedAt != state.UpdatedAt {
		t.Fatalf("expected updatedAt to stay %d, got %d", state.UpdatedAt, next.UpdatedAt)
	}
}
