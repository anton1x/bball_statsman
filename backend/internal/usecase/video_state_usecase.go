package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"bball_statsman_backend/internal/domain"
)

var ErrInvalidURL = errors.New("url is required")

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

	state.UpdatedAt = time.Now().UnixMilli()
	return uc.repo.Save(ctx, state)
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
