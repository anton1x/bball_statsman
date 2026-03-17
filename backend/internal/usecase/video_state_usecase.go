package usecase

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	"bball_statsman_backend/internal/domain"
	"bball_statsman_backend/internal/pubsub"
)

var ErrInvalidURL = errors.New("url is required")
var ErrInvalidOperation = errors.New("invalid operation payload")

type VideoStateRepository interface {
	Save(ctx context.Context, state domain.VideoState) error
	GetByURL(ctx context.Context, url string) (*domain.VideoState, error)
	DeleteByURL(ctx context.Context, url string) error
	ListSummaries(ctx context.Context) ([]domain.VideoSummary, error)
}

type VideoStateUseCase struct {
	repo   VideoStateRepository
	broker *pubsub.Broker
}

func NewVideoStateUseCase(repo VideoStateRepository, broker *pubsub.Broker) *VideoStateUseCase {
	return &VideoStateUseCase{repo: repo, broker: broker}
}

func (uc *VideoStateUseCase) SaveState(ctx context.Context, state domain.VideoState) error {
	state.URL = strings.TrimSpace(state.URL)
	if state.URL == "" {
		return ErrInvalidURL
	}

	existing, err := uc.repo.GetByURL(ctx, state.URL)
	if err != nil {
		return err
	}

	state.UpdatedAt = time.Now().UnixMilli()
	if existing != nil {
		if videoStatePayloadEqual(*existing, state) {
			state.Version = existing.Version
			state.UpdatedAt = existing.UpdatedAt
		} else {
			state.Version = existing.Version + 1
		}
	} else if state.Version <= 0 {
		state.Version = 1
	}

	return uc.repo.Save(ctx, state)
}

// ApplyOperations applies a batch of operations to the video state, persists it,
// and broadcasts the operations to all SSE subscribers of that video URL.
func (uc *VideoStateUseCase) ApplyOperations(ctx context.Context, url string, operations []domain.VideoOperation) (*domain.VideoState, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, ErrInvalidURL
	}
	if len(operations) == 0 {
		return uc.GetState(ctx, url)
	}

	current, err := uc.repo.GetByURL(ctx, url)
	if err != nil {
		return nil, err
	}

	state := domain.VideoState{URL: url, Events: []domain.Event{}, Games: []domain.GameRange{}}
	if current != nil {
		state = *current
	}

	for _, op := range operations {
		if err := applyOperation(&state, op); err != nil {
			return nil, err
		}
	}

	if current != nil && videoStatePayloadEqual(*current, state) {
		return current, nil
	}

	state.UpdatedAt = time.Now().UnixMilli()
	if state.Version <= 0 {
		state.Version = 1
	} else {
		state.Version += 1
	}

	if err := uc.repo.Save(ctx, state); err != nil {
		return nil, err
	}

	uc.broker.Publish(url, pubsub.Event{
		VideoURL:   url,
		Operations: operations,
		Version:    state.Version,
	})

	return &state, nil
}

func videoStatePayloadEqual(a, b domain.VideoState) bool {
	a.Version = 0
	a.UpdatedAt = 0
	b.Version = 0
	b.UpdatedAt = 0
	return reflect.DeepEqual(a, b)
}

func applyOperation(state *domain.VideoState, op domain.VideoOperation) error {
	switch op.Type {
	case "event_upsert":
		if op.Event == nil || strings.TrimSpace(op.Event.ID) == "" {
			return ErrInvalidOperation
		}
		for i := range state.Events {
			if state.Events[i].ID == op.Event.ID {
				state.Events[i] = *op.Event
				return nil
			}
		}
		state.Events = append(state.Events, *op.Event)
		return nil
	case "event_remove":
		if strings.TrimSpace(op.EventID) == "" {
			return ErrInvalidOperation
		}
		next := make([]domain.Event, 0, len(state.Events))
		for _, event := range state.Events {
			if event.ID != op.EventID {
				next = append(next, event)
			}
		}
		state.Events = next
		return nil
	case "events_clear":
		state.Events = []domain.Event{}
		return nil
	case "games_replace":
		if op.Games == nil {
			state.Games = []domain.GameRange{}
			return nil
		}
		state.Games = op.Games
		return nil
	case "settings_replace":
		if op.Settings == nil {
			return ErrInvalidOperation
		}
		state.Settings = *op.Settings
		return nil
	default:
		return ErrInvalidOperation
	}
}

func (uc *VideoStateUseCase) GetState(ctx context.Context, url string) (*domain.VideoState, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, ErrInvalidURL
	}

	return uc.repo.GetByURL(ctx, url)
}

func (uc *VideoStateUseCase) GetStateVersion(ctx context.Context, url string) (int64, bool, error) {
	state, err := uc.GetState(ctx, url)
	if err != nil {
		return 0, false, err
	}
	if state == nil {
		return 0, false, nil
	}

	return state.Version, true, nil
}

func (uc *VideoStateUseCase) DeleteState(ctx context.Context, url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return ErrInvalidURL
	}

	return uc.repo.DeleteByURL(ctx, url)
}

func (uc *VideoStateUseCase) ListStates(ctx context.Context) ([]domain.VideoSummary, error) {
	return uc.repo.ListSummaries(ctx)
}
