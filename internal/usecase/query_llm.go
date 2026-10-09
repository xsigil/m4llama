package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/xsigil/m4llama/internal/domain/entity"
	"github.com/xsigil/m4llama/internal/domain/repository"
)

type QueryLLMInput struct {
	TemplateName    string
	TemplateContent string
	Variables       map[string]string
	Model           string
	Temperature     float64
	MaxTokens       int
	DryRunCurl      bool
}

type QueryLLMOutput struct {
	HistoryID   string
	Prompt      string
	Completion  string
	Tokens      entity.TokenUsage
	CurlCommand string
}

type QueryLLMUseCase struct {
	expander    repository.MacroExpander
	llmClient   repository.LLMClient
	historyRepo repository.HistoryRepository
}

func NewQueryLLMUseCase(expander repository.MacroExpander, client repository.LLMClient, history repository.HistoryRepository) *QueryLLMUseCase {
	return &QueryLLMUseCase{
		expander:    expander,
		llmClient:   client,
		historyRepo: history,
	}
}

func (uc *QueryLLMUseCase) Execute(ctx context.Context, in QueryLLMInput) (*QueryLLMOutput, error) {
	prompt, err := uc.expander.Expand(ctx, in.TemplateContent, in.Variables)
	if err != nil {
		return nil, fmt.Errorf("template expansion failed: %w", err)
	}

	modelName := in.Model
	if modelName == "" || modelName == "default" {
		modelName = "/models/model.gguf"
	}

	maxTokens := in.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 256 // デフォルトで過度な無限ループ思考を防止
	}

	req := entity.CompletionRequest{
		Model:       modelName,
		Temperature: in.Temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
		Messages: []entity.Message{
			{Role: "user", Content: prompt},
		},
	}

	if in.DryRunCurl {
		curlCmd, err := uc.llmClient.EmitCurl(req)
		if err != nil {
			return nil, fmt.Errorf("curl generation failed: %w", err)
		}
		return &QueryLLMOutput{Prompt: prompt, CurlCommand: curlCmd}, nil
	}

	resp, err := uc.llmClient.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("llm inference failed: %w", err)
	}

	// ランダム ID の生成
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	entryID := hex.EncodeToString(b)

	entry := &entity.HistoryEntry{
		ID:             entryID,
		TemplateName:   in.TemplateName,
		Variables:      in.Variables,
		ExpandedPrompt: prompt,
		Completion:     resp.Content,
		PromptTokens:   resp.Usage.PromptTokens,
		CompTokens:     resp.Usage.CompletionTokens,
		TotalTokens:    resp.Usage.TotalTokens,
		Pinned:         false,
		CreatedAt:      time.Now().UTC(),
	}
	_ = uc.historyRepo.Save(ctx, entry)

	return &QueryLLMOutput{
		HistoryID:  entry.ID,
		Prompt:     prompt,
		Completion: resp.Content,
		Tokens:     resp.Usage,
	}, nil
}
