package m4

import (
	"bufio"
	"context"
	"regexp"
	"strings"

	"github.com/xsigil/m4llama/internal/domain/entity"
)

// 例: # @var TASK_NAME [string] "タスクの概要を入力してください" "デフォルト値"
// または # @var STYLE [select:formal,casual] "文体" "formal"
var varRegex = regexp.MustCompile(`^#\s*@var\s+([A-Za-z0-9_]+)(?:\s+\[([a-z0-9_,:]+)\])?(?:\s+"([^"]*)")?(?:\s+"([^"]*)")?`)

type Expander struct{}

func NewExpander() *Expander {
	return &Expander{}
}

func (e *Expander) ParseAnnotations(ctx context.Context, templateContent string) ([]entity.TemplateVar, error) {
	var vars []entity.TemplateVar
	scanner := bufio.NewScanner(strings.NewReader(templateContent))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// 注釈行以外に達したらヘッダー終了とみなす
		if !strings.HasPrefix(line, "#") {
			break
		}

		matches := varRegex.FindStringSubmatch(line)
		if len(matches) > 0 {
			name := matches[1]
			rawType := matches[2]
			desc := matches[3]
			defaultVal := matches[4]

			varType := entity.VarTypeString
			var options []string

			if strings.HasPrefix(rawType, "select:") {
				varType = entity.VarTypeSelect
				options = strings.Split(strings.TrimPrefix(rawType, "select:"), ",")
			} else {
				switch rawType {
				case "text":
					varType = entity.VarTypeText
				case "confirm":
					varType = entity.VarTypeConfirm
				default:
					varType = entity.VarTypeString
				}
			}

			vars = append(vars, entity.TemplateVar{
				Name:        name,
				Type:        varType,
				Description: desc,
				Default:     defaultVal,
				Options:     options,
			})
		}
	}

	return vars, scanner.Err()
}
