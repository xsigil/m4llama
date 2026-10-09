package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	infram4 "github.com/xsigil/m4llama/internal/infrastructure/m4"
	"github.com/xsigil/m4llama/internal/usecase"
)

const createNewOption = "[➕ Create New Template]"

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
	for {
		// 1. テンプレート一覧のスキャン
		candidates, err := ScanTemplates(r.templateDir, ".")
		if err != nil {
			return fmt.Errorf("failed to scan templates: %w", err)
		}

		// 2. セレクター起動 (ベースディレクトリからの相対パス表示)
		chosenTmpl, err := SelectTemplate(r.templateDir, []string{createNewOption}, candidates)
		if err != nil {
			return err
		}
		// 新規作成の場合
		if chosenTmpl == createNewOption {
			newFileName := ""
			form := huh.NewInput().
				Title("New Template Filename").
				Description("例: coding/refactor.m4 (拡張子 .m4 は自動付与)").
				Value(&newFileName)

			if err := form.Run(); err != nil || strings.TrimSpace(newFileName) == "" {
				continue
			}

			if !strings.HasSuffix(newFileName, ".m4") {
				newFileName += ".m4"
			}

			// 保存先ディレクトリを確保
			destPath := filepath.Join(r.templateDir, newFileName)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

			// 初期テンプレート雛形を書き込み
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				initContent := "# @var GOAL [text] \"タスクの目的\"\ndefine(`GOAL', `なし')dnl\nあなたはプロのエンジニアです。\n目的: GOAL\n"
				_ = os.WriteFile(destPath, []byte(initContent), 0644)
			}

			if err := OpenInEditor(destPath); err != nil {
				fmt.Fprintf(os.Stderr, "Editor error: %v\n", err)
			}
			chosenTmpl = destPath
		}

		// 3. アクション選択
		action := "run"
		actionSelect := huh.NewSelect[string]().
			Title(fmt.Sprintf("Template: %s", filepath.Base(chosenTmpl))).
			Options(
				huh.NewOption("🚀 Run Inference (推論実行)", "run"),
				huh.NewOption("📝 Edit in Neovim (エディタで編集)", "edit"),
				huh.NewOption("📋 Export curl command (curl エクスポート)", "curl"),
			).
			Value(&action)

		if err := actionSelect.Run(); err != nil {
			return err
		}

		if action == "edit" {
			if err := OpenInEditor(chosenTmpl); err != nil {
				fmt.Fprintf(os.Stderr, "Editor error: %v\n", err)
			}
			continue
		}

		// 4. テンプレート内容の読み込み
		contentBytes, err := os.ReadFile(chosenTmpl)
		if err != nil {
			return fmt.Errorf("read template failed: %w", err)
		}
		tmplContent := string(contentBytes)

		// 5. 注釈変数の解析
		vars, err := r.m4Expander.ParseAnnotations(ctx, tmplContent)
		if err != nil {
			return fmt.Errorf("parse annotations failed: %w", err)
		}

		// 6. 動的フォーム入力
		userVars, err := RunDynamicForm(vars)
		if err != nil {
			return err
		}

		// 7. 推論実行
		out, err := r.queryUC.Execute(ctx, usecase.QueryLLMInput{
			TemplateName:    chosenTmpl,
			TemplateContent: tmplContent,
			Variables:       userVars,
			Model:           "/models/model.gguf",
			Temperature:     0.7,
			MaxTokens:       256,
			DryRunCurl:      (action == "curl"),
		})
		if err != nil {
			return err
		}

		// 8. 結果出力
		fmt.Println("\n------------------------------------------------------------")
		if action == "curl" {
			fmt.Println("[Exported curl command]")
			fmt.Println(out.CurlCommand)
		} else {
			fmt.Printf("[Completion (ID: %s, Tokens: %d)]\n\n", out.HistoryID, out.Tokens.TotalTokens)
			fmt.Println(out.Completion)
		}
		fmt.Println("------------------------------------------------------------")

		break
	}

	return nil
}
