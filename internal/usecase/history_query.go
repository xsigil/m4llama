package usecase

import (
	"context"
	"fmt"

	"m4llama/internal/domain/entity"
	"m4llama/internal/domain/repository"
)

type HistoryQueryUseCase struct {
	historyRepo repository.HistoryRepository
}

func NewHistoryQueryUseCase(historyRepo repository.HistoryRepository) *HistoryQueryUseCase {
	return &HistoryQueryUseCase{
		historyRepo: historyRepo,
	}
}

// GetRecent は最近の実行履歴を取得します
func (u *HistoryQueryUseCase) GetRecent(ctx context.Context, limit int) ([]*entity.HistoryEntry, error) {
	if limit <= 0 {
		limit = 20
	}
	return u.historyRepo.ListRecent(ctx, limit)
}

// GetByID は指定 ID の履歴を取得します（コンテキストスナイピング用）
func (u *HistoryQueryUseCase) GetByID(ctx context.Context, id string) (*entity.HistoryEntry, error) {
	entry, err := u.historyRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history entry: %w", err)
	}
	if entry == nil {
		return nil, fmt.Errorf("history entry not found: %s", id)
	}
	return entry, nil
}

// TogglePin は指定履歴のピン留め状態を反転します
func (u *HistoryQueryUseCase) TogglePin(ctx context.Context, id string) (bool, error) {
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

// ListPinned はピン留めされた履歴の一覧を取得します
func (u *HistoryQueryUseCase) ListPinned(ctx context.Context) ([]*entity.HistoryEntry, error) {
	return u.historyRepo.ListPinned(ctx)
}
