package usecase

import (
	"context"
	"fmt"
	"time"

	"m4llama/internal/domain/entity"
	"m4llama/internal/domain/repository"
)

type QueryLLMInput struct {
	TemplateName    string
	TemplateContent string
	Variables       map[string]string
	Model           string
	Temperature     float64
	DryRunCurl      bool // true の場合は通信せず curl コマンドを返す
}

type QueryLLMOutput struct {
	ExpandedPrompt string
	Completion     string
	CurlCommand    string
	Tokens         entity.TokenUsage
	HistoryID      string
}

type QueryLLMUseCase struct {
	expander    repository.MacroExpander
	llmClient   repository.LLMClient
	historyRepo repository.HistoryRepository
}

func NewQueryLLMUseCase(
	expander repository.MacroExpander,
	llmClient repository.LLMClient,
	historyRepo repository.HistoryRepository,
) *QueryLLMUseCase {
	return &QueryLLMUseCase{
		expander:    expander,
		llmClient:   llmClient,
		historyRepo: historyRepo,
	}
}

func (u *QueryLLMUseCase) Execute(ctx context.Context, in QueryLLMInput) (*QueryLLMOutput, error) {
	// 1. マクロ展開
	expanded, err := u.expander.Expand(ctx, in.TemplateContent, in.Variables)
	if err != nil {
		return nil, fmt.Errorf("macro expansion failed: %w", err)
	}

	req := entity.CompletionRequest{
		Model: in.Model,
		Messages: []entity.ChatMessage{
			{Role: entity.RoleUser, Content: expanded},
		},
		Temperature: in.Temperature,
	}

	// 2. dry-run / curl export の場合は Go で通信せず curl コマンドを生成して終了
	if in.DryRunCurl {
		curlCmd, err := u.llmClient.EmitCurl(req)
		if err != nil {
			return nil, fmt.Errorf("failed to emit curl command: %w", err)
		}
		return &QueryLLMOutput{
			ExpandedPrompt: expanded,
			CurlCommand:    curlCmd,
		}, nil
	}

	// 3. 通常実行: Go の net/http で直接推論リクエスト
	resp, err := u.llmClient.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm inference failed: %w", err)
	}

	// 4. 実行履歴のスナップショットを SQLite に記録
	historyID := fmt.Sprintf("%d", time.Now().UnixNano())
	historyEntry := &entity.HistoryEntry{
		ID:             historyID,
		TemplateName:   in.TemplateName,
		Variables:      in.Variables,
		ExpandedPrompt: expanded,
		Completion:     resp.Content,
		PromptTokens:   resp.Usage.PromptTokens,
		CompTokens:     resp.Usage.CompletionTokens,
		TotalTokens:    resp.Usage.TotalTokens,
		Pinned:         false,
		CreatedAt:      time.Now(),
	}

	if err := u.historyRepo.Save(ctx, historyEntry); err != nil {
		// 履歴保存の失敗はログや警告に留めることも可能だが、ここでは整合性を考慮
		return nil, fmt.Errorf("failed to persist history: %w", err)
	}

	return &QueryLLMOutput{
		ExpandedPrompt: expanded,
		Completion:     resp.Content,
		Tokens:         resp.Usage,
		HistoryID:      historyID,
	}, nil
}
