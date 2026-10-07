package repository

import (
	"context"

	"m4llama/internal/domain/entity"
)

type SampleRepository interface {
	FindByID(ctx context.Context, id entity.ID) (*entity.Sample, error)
	Save(ctx context.Context, sample *entity.Sample) error
}
