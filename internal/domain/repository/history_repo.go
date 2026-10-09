package repository

import (
	"context"

	"github.com/xsigil/m4llama/internal/domain/entity"
)

type HistoryRepository interface {
	Save(ctx context.Context, entry *entity.HistoryEntry) error
	Update(ctx context.Context, entry *entity.HistoryEntry) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*entity.HistoryEntry, error)
	ListRecent(ctx context.Context, limit int) ([]*entity.HistoryEntry, error)
	ListPinned(ctx context.Context) ([]*entity.HistoryEntry, error)
	SetPinned(ctx context.Context, id int64, pinned bool) error
}
