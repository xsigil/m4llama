package m4

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Engine struct {
	lexer      *Lexer
	macros     map[string]string
	builtins   map[string]MacroFunc
	diversions map[int]*bytes.Buffer
	currentDiv int
}

func NewEngine() *Engine {
	e := &Engine{
		lexer:      NewLexer(),
		macros:     make(map[string]string),
		builtins:   make(map[string]MacroFunc),
		diversions: make(map[int]*bytes.Buffer),
		currentDiv: 0,
	}
	registerDefaultBuiltins(e)
	return e
}

func (e *Engine) RegisterBuiltin(name string, fn MacroFunc) {
	e.builtins[name] = fn
}

func (e *Engine) Define(name, value string) {
	e.macros[name] = value
}

// Expand は入力文字列を展開して結果を返します
func (e *Engine) Expand(input string) (string, error) {
	e.lexer.PushInput(input)

	for {
		tok, err := e.lexer.NextToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch tok.Type {
		case TokenQuoted:
			e.writeOutput(tok.Value)

		case TokenOther:
			e.writeOutput(tok.Value)

		case TokenIdent:
			name := tok.Value
			isMacro := e.hasMacro(name)

			if !isMacro {
				e.writeOutput(name)
				continue
			}

			// マクロの場合、直後に '(' が続くか先読みして確認
			args, hasParen, err := e.parseMacroArgs()
			if err != nil {
				return "", err
			}

			expansion, err := e.evalMacro(name, args, hasParen)
			if err != nil {
				return "", err
			}

			// 展開結果を再度入力スタックに積んで再評価（m4 の原則）
			e.lexer.PushInput(expansion)
		}
	}

	// undivert の残り（0 以外のダイバージョン）を最後に連結
	var finalOut bytes.Buffer
	if out0, ok := e.diversions[0]; ok {
		finalOut.WriteString(out0.String())
	}
	return finalOut.String(), nil
}

func (e *Engine) hasMacro(name string) bool {
	if _, ok := e.builtins[name]; ok {
		return true
	}
	if _, ok := e.macros[name]; ok {
		return true
	}
	return false
}

func (e *Engine) writeOutput(s string) {
	if e.currentDiv < 0 {
		return // 負のダイバージョンは出力を捨てる
	}
	buf, ok := e.diversions[e.currentDiv]
	if !ok {
		buf = &bytes.Buffer{}
		e.diversions[e.currentDiv] = buf
	}
	buf.WriteString(s)
}

func (e *Engine) parseMacroArgs() ([]string, bool, error) {
	// 空白をスキップせずに直後の文字を確認
	r, ok := e.lexer.readRune()
	if !ok {
		return nil, false, nil
	}
	if r != '(' {
		e.lexer.unreadRune()
		return nil, false, nil
	}

	var args []string
	var currentArg bytes.Buffer
	depth := 0

	for {
		tok, err := e.lexer.NextToken()
		if err == io.EOF {
			return nil, true, fmt.Errorf("unexpected EOF while reading macro arguments")
		}
		if err != nil {
			return nil, true, err
		}

		switch tok.Type {
		case TokenQuoted:
			// クォート文字列は、内側のクォート構造を保ったまま復元してバッファへ
			currentArg.WriteString(e.lexer.leftQuote)
			currentArg.WriteString(tok.Value)
			currentArg.WriteString(e.lexer.rightQuote)

		case TokenOther:
			switch tok.Value {
			case "(":
				depth++
				currentArg.WriteString("(")
			case ")":
				if depth == 0 {
					argStr := strings.TrimSpace(currentArg.String())
					// 最初の引数かつ中身が完全に空の場合は引数なし扱い（例: FOO()）
					if len(args) == 0 && argStr == "" {
						return args, true, nil
					}
					args = append(args, unquoteOnce(argStr, e.lexer.leftQuote, e.lexer.rightQuote))
					return args, true, nil
				}
				depth--
				currentArg.WriteString(")")
			case ",":
				if depth == 0 {
					argStr := strings.TrimSpace(currentArg.String())
					args = append(args, unquoteOnce(argStr, e.lexer.leftQuote, e.lexer.rightQuote))
					currentArg.Reset()
				} else {
					currentArg.WriteString(",")
				}
			default:
				currentArg.WriteString(tok.Value)
			}

		default:
			currentArg.WriteString(tok.Value)
		}
	}
}

// unquoteOnce は前後の外側クォートを1組だけ剥がします
func unquoteOnce(s, left, right string) string {
	if strings.HasPrefix(s, left) && strings.HasSuffix(s, right) {
		trimmed := strings.TrimPrefix(s, left)
		trimmed = strings.TrimSuffix(trimmed, right)
		return trimmed
	}
	return s
}

func (e *Engine) evalMacro(name string, args []string, hasParen bool) (string, error) {
	if fn, ok := e.builtins[name]; ok {
		return fn(e, args)
	}

	body, ok := e.macros[name]
	if !ok {
		return "", nil
	}

	// ユーザー定義マクロの $1, $2, $#, $@ 置換
	return substituteArgs(body, name, args), nil
}

func substituteArgs(body, macroName string, args []string) string {
	res := body
	res = strings.ReplaceAll(res, "$0", macroName)
	res = strings.ReplaceAll(res, "$#", strconv.Itoa(len(args)))
	res = strings.ReplaceAll(res, "$*", strings.Join(args, ","))
	res = strings.ReplaceAll(res, "$@", strings.Join(args, ","))

	for i, arg := range args {
		placeholder := fmt.Sprintf("$%d", i+1)
		res = strings.ReplaceAll(res, placeholder, arg)
	}
	return res
}
