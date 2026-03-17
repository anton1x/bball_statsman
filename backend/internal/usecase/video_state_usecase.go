package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"bball_statsman_backend/internal/domain"
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
	repo VideoStateRepository
}

func NewVideoStateUseCase(repo VideoStateRepository) *VideoStateUseCase {
	return &VideoStateUseCase{repo: repo}
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
		state.Version = existing.Version + 1
	} else if state.Version <= 0 {
		state.Version = 1
	}

	return uc.repo.Save(ctx, state)
}

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

	state.UpdatedAt = time.Now().UnixMilli()
	if state.Version <= 0 {
		state.Version = 1
	} else {
		state.Version += 1
	}

	if err := uc.repo.Save(ctx, state); err != nil {
		return nil, err
	}

	return &state, nil
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
