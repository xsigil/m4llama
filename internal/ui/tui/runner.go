package tui

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	infram4 "github.com/xsigil/m4llama/internal/infrastructure/m4"
	"github.com/xsigil/m4llama/internal/usecase"
)

type Runner struct {
	queryUC     *usecase.QueryLLMUseCase
	historyUC   *usecase.HistoryQueryUseCase
	m4Expander  *infram4.Expander
	templateDir string
}

func NewRunner(queryUC *usecase.QueryLLMUseCase, historyUC *usecase.HistoryQueryUseCase, expander *infram4.Expander, templateDir string) *Runner {
	return &Runner{
		queryUC:     queryUC,
		historyUC:   historyUC,
		m4Expander:  expander,
		templateDir: templateDir,
	}
}

func (r *Runner) Start(ctx context.Context) error {
	// 1. .m4 テンプレートの収集
	candidates, err := ScanTemplates(r.templateDir, ".")
	if err != nil {
		return fmt.Errorf("failed to scan templates: %w", err)
	}
	if len(candidates) == 0 {
		return fmt.Errorf("no .m4 templates found in %s or current directory", r.templateDir)
	}

	// 2. Levenshtein ファジー検索による選択
	chosenTmpl, err := SelectTemplate(candidates)
	if err != nil {
		return err
	}

	contentBytes, err := os.ReadFile(chosenTmpl)
	if err != nil {
		return fmt.Errorf("read template failed: %w", err)
	}
	tmplContent := string(contentBytes)

	// 3. テンプレート注釈ヘッダーの解析
	vars, err := r.m4Expander.ParseAnnotations(ctx, tmplContent)
	if err != nil {
		return fmt.Errorf("parse annotations failed: %w", err)
	}

	// 4. 動的フォームで変数値の対話収集
	userVars, err := RunDynamicForm(vars)
	if err != nil {
		return err
	}

	// 5. 実行アクションの選択（推論実行 / curl 出力）
	action := "run"
	actionSelect := huh.NewSelect[string]().
		Title("Action").
		Options(
			huh.NewOption("Run Inference (推論実行)", "run"),
			huh.NewOption("Export curl command (curl エクスポート)", "curl"),
		).
		Value(&action)

	if err := actionSelect.Run(); err != nil {
		return err
	}

	// 6. ユースケース実行
	out, err := r.queryUC.Execute(ctx, usecase.QueryLLMInput{
		TemplateName:    chosenTmpl,
		TemplateContent: tmplContent,
		Variables:       userVars,
		Model:           "default",
		Temperature:     0.7,
		DryRunCurl:      (action == "curl"),
	})
	if err != nil {
		return err
	}

	// 7. 結果出力
	fmt.Println("\n------------------------------------------------------------")
	if action == "curl" {
		fmt.Println("[Exported curl command]")
		fmt.Println(out.CurlCommand)
	} else {
		fmt.Printf("[Completion (ID: %s, Tokens: %d)]\n\n", out.HistoryID, out.Tokens.TotalTokens)
		fmt.Println(out.Completion)
	}
	fmt.Println("------------------------------------------------------------")

	return nil
}
