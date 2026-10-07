package usecase

import (
	"context"
	"fmt"

	"m4llama/internal/domain/entity"
	"m4llama/internal/domain/repository"
)

type SampleUsecase struct {
	repo      repository.SampleRepository
	txManager repository.TxManager
}

func NewSampleUsecase(repo repository.SampleRepository, txManager repository.TxManager) *SampleUsecase {
	return &SampleUsecase{
		repo:      repo,
		txManager: txManager,
	}
}

// Execute は Functor パターンによりトランザクション境界内で実行される
func (u *SampleUsecase) Execute(ctx context.Context, rawID string, val string) error {
	sample, err := entity.NewSample(entity.ID(rawID), val)
	if err != nil {
		return fmt.Errorf("domain rule violation: %w", err)
	}

	return u.txManager.Do(ctx, func(txCtx context.Context) error {
		// トランザクション内で実行
		if err := u.repo.Save(txCtx, sample); err != nil {
			return err
		}
		fmt.Printf("[+] Entity saved within transaction: %s\n", sample.ID)
		return nil
	})
}
