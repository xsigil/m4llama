package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/xsigil/m4llama/internal/domain/entity"
)

// RunDynamicForm は注釈変数定義をもとに huh フォームを動的生成し、ユーザー入力を取得します
func RunDynamicForm(vars []entity.TemplateVar) (map[string]string, error) {
	values := make(map[string]string)
	if len(vars) == 0 {
		return values, nil
	}

	// 1. 端末画面を完全に消去してカーソルを左上に戻す (ANSI エスケープ)
	fmt.Print("\033[2J\033[H")

	var fields []huh.Field

	for _, v := range vars {
		vName := v.Name
		initVal := v.Default
		desc := v.Description
		if desc == "" {
			desc = fmt.Sprintf("Variable: %s", vName)
		}

		values[vName] = initVal

		switch v.Type {
		case entity.VarTypeText:
			field := huh.NewText().
				Title(vName).
				Description(desc).
				Value(func() *string {
					ptr := new(string)
					*ptr = initVal
					return ptr
				}())
			fields = append(fields, field)

		case entity.VarTypeSelect:
			var opts []huh.Option[string]
			for _, opt := range v.Options {
				cleanOpt := strings.TrimSpace(opt)
				opts = append(opts, huh.NewOption(cleanOpt, cleanOpt))
			}
			ptr := new(string)
			*ptr = initVal
			if len(opts) > 0 && *ptr == "" {
				*ptr = opts[0].Key
			}
			field := huh.NewSelect[string]().
				Title(vName).
				Description(desc).
				Options(opts...).
				Value(ptr)
			fields = append(fields, field)

		case entity.VarTypeConfirm:
			ptr := new(bool)
			*ptr = (strings.ToLower(initVal) == "true" || initVal == "1" || initVal == "yes")
			field := huh.NewConfirm().
				Title(vName).
				Description(desc).
				Value(ptr)
			fields = append(fields, field)

		default: // VarTypeString
			ptr := new(string)
			*ptr = initVal
			field := huh.NewInput().
				Title(vName).
				Description(desc).
				Value(ptr)
			fields = append(fields, field)
		}
	}

	form := huh.NewForm(huh.NewGroup(fields...)).
		WithProgramOptions()

	err := form.Run()

	// 2. フォーム終了時にも画面を消去し、Cockpit への復帰残像を防ぐ
	fmt.Print("\033[2J\033[H")

	if err != nil {
		return nil, err
	}

	for i, f := range fields {
		vName := vars[i].Name
		switch v := f.(type) {
		case *huh.Input:
			values[vName] = v.GetValue().(string)
		case *huh.Text:
			values[vName] = v.GetValue().(string)
		case *huh.Select[string]:
			values[vName] = v.GetValue().(string)
		case *huh.Confirm:
			if v.GetValue().(bool) {
				values[vName] = "true"
			} else {
				values[vName] = "false"
			}
		}
	}

	return values, nil
}
