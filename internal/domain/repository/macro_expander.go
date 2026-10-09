package repository

import (
	"context"
	"m4llama/internal/domain/entity"
)

type MacroExpander interface {
	ParseAnnotations(ctx context.Context, templateContent string) ([]entity.TemplateVar, error)
	Expand(ctx context.Context, templateContent string, vars map[string]string) (string, error)
}
