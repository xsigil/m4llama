package m4

import (
	"context"
	"fmt"
	"strings"

	pkgm4 "github.com/xsigil/m4llama/pkg/m4"
)

func (e *Expander) Expand(ctx context.Context, templateContent string, vars map[string]string) (string, error) {
	engine := pkgm4.NewEngine()

	// 変数を m4 マクロとして事前に定義
	for k, v := range vars {
		engine.Define(k, v)
	}

	// テンプレートヘッダーの注釈行 (# @var ...) を除外してから評価
	lines := strings.Split(templateContent, "\n")
	var bodyLines []string
	headerEnded := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !headerEnded {
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			headerEnded = true
		}
		bodyLines = append(bodyLines, line)
	}

	rawBody := strings.Join(bodyLines, "\n")
	expanded, err := engine.Expand(rawBody)
	if err != nil {
		return "", fmt.Errorf("failed to expand m4 template: %w", err)
	}

	return expanded, nil
}
