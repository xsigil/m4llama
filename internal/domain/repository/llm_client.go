package repository

import (
	"context"
	"m4llama/internal/domain/entity"
)

type LLMClient interface {
	Complete(ctx context.Context, req entity.CompletionRequest) (*entity.CompletionResponse, error)
	EmitCurl(req entity.CompletionRequest) (string, error)
}
