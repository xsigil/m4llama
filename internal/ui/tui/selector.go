package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
)

// ScanTemplates はテンプレートディレクトリから .m4 ファイルを再帰スキャンします
func ScanTemplates(roots ...string) ([]string, error) {
	var templates []string

	for _, root := range roots {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".m4") {
				templates = append(templates, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return templates, nil
}

// SelectTemplate は自前ファジーマッチを用いた対話型テンプレート選択を提供します
func SelectTemplate(templates []string) (string, error) {
	if len(templates) == 0 {
		return "", fmt.Errorf("no .m4 templates found")
	}

	// テンプレートが1つだけならそのまま返す
	if len(templates) == 1 {
		return templates[0], nil
	}

	var searchQuery string
	for {
		// 1. 絞り込みクエリ入力
		var formhuh = huh.NewInput().
			Title("Template Search (fuzzy)").
			Description("Enter keyword to sort via Levenshtein distance (leave empty to show all)").
			Value(&searchQuery)

		if err := formhuh.Run(); err != nil {
			return "", err
		}

		// 2. レーベンシュタイン距離でソート
		ranked := FuzzyRank(searchQuery, templates)

		var options []huh.Option[string]
		for _, tmpl := range ranked {
			options = append(options, huh.NewOption(tmpl, tmpl))
		}
		// 検索やり直し用の選択肢を追加
		options = append(options, huh.NewOption("[ Search again / 検索し直す ]", "__RETRY__"))

		var selected string
		selectField := huh.NewSelect[string]().
			Title("Choose a template").
			Options(options...).
			Value(&selected)

		if err := selectField.Run(); err != nil {
			return "", err
		}

		if selected != "__RETRY__" {
			return selected, nil
		}
	}
}
