package usecase

import (
	"context"
	"fmt"

	"github.com/xsigil/m4llama/internal/domain/entity"
	"github.com/xsigil/m4llama/internal/domain/repository"
)

type HistoryQueryUseCase struct {
	historyRepo repository.HistoryRepository
}

func NewHistoryQueryUseCase(historyRepo repository.HistoryRepository) *HistoryQueryUseCase {
	return &HistoryQueryUseCase{
		historyRepo: historyRepo,
	}
}

func (u *HistoryQueryUseCase) GetRecent(ctx context.Context, limit int) ([]*entity.HistoryEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	return u.historyRepo.ListRecent(ctx, limit)
}

func (u *HistoryQueryUseCase) GetByID(ctx context.Context, id int64) (*entity.HistoryEntry, error) {
	entry, err := u.historyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history entry: %w", err)
	}
	if entry == nil {
		return nil, fmt.Errorf("history entry not found: %d", id)
	}
	return entry, nil
}

func (u *HistoryQueryUseCase) UpdateEntry(ctx context.Context, entry *entity.HistoryEntry) error {
	return u.historyRepo.Update(ctx, entry)
}

func (u *HistoryQueryUseCase) DeleteEntry(ctx context.Context, id int64) error {
	return u.historyRepo.Delete(ctx, id)
}

func (u *HistoryQueryUseCase) TogglePin(ctx context.Context, id int64) (bool, error) {
	entry, err := u.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	newStatus := !entry.Pinned
	if err := u.historyRepo.SetPinned(ctx, id, newStatus); err != nil {
		return false, fmt.Errorf("failed to update pin status: %w", err)
	}
	return newStatus, nil
}

func (u *HistoryQueryUseCase) ListPinned(ctx context.Context) ([]*entity.HistoryEntry, error) {
	return u.historyRepo.ListPinned(ctx)
}
