package llamaserver

import (
	"bufio"
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

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
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
	req.Stream = false
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
		return nil, fmt.Errorf("llama-server error %d: %s", resp.StatusCode, string(respBody))
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

func (c *Client) CompleteStream(ctx context.Context, req entity.CompletionRequest, onToken func(chunk string)) (*entity.CompletionResponse, error) {
	req.Stream = true
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
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("inference request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("llama-server error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(resp.Body)
	var fullContent strings.Builder
	totalTokens := 0

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			tokenText := delta.Content
			if tokenText == "" {
				tokenText = delta.ReasoningContent
			}
			if tokenText != "" {
				fullContent.WriteString(tokenText)
				if onToken != nil {
					onToken(tokenText)
				}
			}
		}

		if chunk.Usage != nil {
			totalTokens = chunk.Usage.TotalTokens
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stream read error: %w", err)
	}

	if totalTokens == 0 {
		totalTokens = len(strings.Fields(fullContent.String()))
	}

	return &entity.CompletionResponse{
		Content: fullContent.String(),
		Usage: entity.TokenUsage{
			TotalTokens: totalTokens,
		},
	}, nil
}

func (c *Client) EmitCurl(req entity.CompletionRequest) (string, error) {
	return c.curlEmitter.Emit(req)
}
