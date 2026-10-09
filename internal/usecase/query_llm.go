package usecase

import (
	"context"
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
	CustomMessages  []entity.Message
}

type QueryLLMOutput struct {
	HistoryID   int64
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
	return uc.ExecuteStream(ctx, in, nil)
}

func (uc *QueryLLMUseCase) ExecuteStream(ctx context.Context, in QueryLLMInput, onToken func(chunk string)) (*QueryLLMOutput, error) {
	var prompt string
	var err error

	if len(in.CustomMessages) == 0 {
		prompt, err = uc.expander.Expand(ctx, in.TemplateContent, in.Variables)
		if err != nil {
			return nil, fmt.Errorf("template expansion failed: %w", err)
		}
	} else {
		for i := len(in.CustomMessages) - 1; i >= 0; i-- {
			if in.CustomMessages[i].Role == "user" {
				prompt = in.CustomMessages[i].Content
				break
			}
		}
	}

	modelName := in.Model
	if modelName == "default" || modelName == "/models/model.gguf" {
		modelName = ""
	}

	maxTokens := in.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	var reqMessages []entity.Message
	if len(in.CustomMessages) > 0 {
		reqMessages = in.CustomMessages
	} else {
		reqMessages = []entity.Message{
			{Role: "user", Content: prompt},
		}
	}

	req := entity.CompletionRequest{
		Model:       modelName,
		Temperature: in.Temperature,
		MaxTokens:   maxTokens,
		Stream:      (onToken != nil),
		Messages:    reqMessages,
	}

	if in.DryRunCurl {
		curlCmd, err := uc.llmClient.EmitCurl(req)
		if err != nil {
			return nil, fmt.Errorf("curl generation failed: %w", err)
		}
		return &QueryLLMOutput{Prompt: prompt, CurlCommand: curlCmd}, nil
	}

	var resp *entity.CompletionResponse
	if onToken != nil {
		resp, err = uc.llmClient.CompleteStream(ctx, req, onToken)
	} else {
		resp, err = uc.llmClient.Complete(ctx, req)
	}
	if err != nil {
		return nil, fmt.Errorf("llm inference failed: %w", err)
	}

	entry := &entity.HistoryEntry{
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
