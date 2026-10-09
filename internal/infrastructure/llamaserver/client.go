package llamaserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xsigil/m4llama/internal/domain/entity"
	"github.com/xsigil/m4llama/internal/domain/repository"
)

type Client struct {
	baseURL     string
	httpClient  *http.Client
	curlEmitter *CurlEmitter
}

var _ repository.LLMClient = (*Client)(nil)

type openAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Role             string `json:"role"`
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	cleanURL := strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: cleanURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		curlEmitter: NewCurlEmitter(cleanURL),
	}
}

func (c *Client) Complete(ctx context.Context, req entity.CompletionRequest) (*entity.CompletionResponse, error) {
	endpoint := fmt.Sprintf("%s/v1/chat/completions", c.baseURL)

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("inference request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llama-server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	content := ""
	if len(oaiResp.Choices) > 0 {
		msg := oaiResp.Choices[0].Message
		if msg.Content != "" {
			content = msg.Content
		} else if msg.ReasoningContent != "" {
			// content が空で reasoning_content のみがある場合は思考内容を出力
			content = msg.ReasoningContent
		}
	}

	return &entity.CompletionResponse{
		ID:      oaiResp.ID,
		Content: content,
		Usage: entity.TokenUsage{
			PromptTokens:     oaiResp.Usage.PromptTokens,
			CompletionTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:      oaiResp.Usage.TotalTokens,
		},
	}, nil
}

func (c *Client) EmitCurl(req entity.CompletionRequest) (string, error) {
	return c.curlEmitter.Emit(req)
}
