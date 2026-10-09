package repository

import (
	"context"
	"m4llama/internal/domain/entity"
)

type HistoryRepository interface {
	Save(ctx context.Context, entry *entity.HistoryEntry) error
	FindByID(ctx context.Context, id string) (*entity.HistoryEntry, error)
	ListRecent(ctx context.Context, limit int) ([]*entity.HistoryEntry, error)
	ListPinned(ctx context.Context) ([]*entity.HistoryEntry, error)
	SetPinned(ctx context.Context, id string, pinned bool) error
}
