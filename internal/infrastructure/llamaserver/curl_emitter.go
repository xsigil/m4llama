package llamaserver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xsigil/m4llama/internal/domain/entity"
)

type CurlEmitter struct {
	baseURL string
}

func NewCurlEmitter(baseURL string) *CurlEmitter {
	return &CurlEmitter{
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (e *CurlEmitter) Emit(req entity.CompletionRequest) (string, error) {
	endpoint := fmt.Sprintf("%s/v1/chat/completions", e.baseURL)

	payload, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal completion request: %w", err)
	}

	escapedPayload := strings.ReplaceAll(string(payload), "'", "'\\''")

	cmd := fmt.Sprintf("curl -s -X POST %s \\\n  -H \"Content-Type: application/json\" \\\n  -d '%s'",
		endpoint, escapedPayload)

	return cmd, nil
}
