package repository

import (
	"context"
	"github.com/xsigil/m4llama/internal/domain/entity"
)

type LLMClient interface {
	Complete(ctx context.Context, req entity.CompletionRequest) (*entity.CompletionResponse, error)
	EmitCurl(req entity.CompletionRequest) (string, error)
}
