package m4

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type MacroFunc func(e *Engine, args []string) (string, error)

func registerDefaultBuiltins(e *Engine) {
	e.RegisterBuiltin("define", builtinDefine)
	e.RegisterBuiltin("undefine", builtinUndefine)
	e.RegisterBuiltin("ifdef", builtinIfdef)
	e.RegisterBuiltin("ifelse", builtinIfelse)
	e.RegisterBuiltin("changequote", builtinChangequote)
	e.RegisterBuiltin("divert", builtinDivert)
	e.RegisterBuiltin("undivert", builtinUndivert)
	e.RegisterBuiltin("include", builtinInclude)
	e.RegisterBuiltin("dnl", builtinDnl)
}

func builtinDefine(e *Engine, args []string) (string, error) {
	if len(args) < 1 || args[0] == "" {
		return "", nil
	}
	name := args[0]
	val := ""
	if len(args) >= 2 {
		val = args[1]
	}
	e.macros[name] = val
	return "", nil
}

func builtinUndefine(e *Engine, args []string) (string, error) {
	if len(args) > 0 {
		delete(e.macros, args[0])
		delete(e.builtins, args[0])
	}
	return "", nil
}

func builtinIfdef(e *Engine, args []string) (string, error) {
	if len(args) < 2 {
		return "", nil
	}
	name := args[0]
	_, isMacro := e.macros[name]
	_, isBuiltin := e.builtins[name]
	if isMacro || isBuiltin {
		return args[1], nil
	}
	if len(args) >= 3 {
		return args[2], nil
	}
	return "", nil
}

func builtinIfelse(e *Engine, args []string) (string, error) {
	for len(args) >= 3 {
		if args[0] == args[1] {
			return args[2], nil
		}
		if len(args) == 4 {
			return args[3], nil
		}
		args = args[3:]
	}
	return "", nil
}

func builtinChangequote(e *Engine, args []string) (string, error) {
	if len(args) == 0 || (args[0] == "" && len(args) == 1) {
		e.lexer.leftQuote = "`"
		e.lexer.rightQuote = "'"
		return "", nil
	}
	e.lexer.leftQuote = args[0]
	if len(args) >= 2 {
		e.lexer.rightQuote = args[1]
	} else {
		e.lexer.rightQuote = "'"
	}
	return "", nil
}

func builtinDivert(e *Engine, args []string) (string, error) {
	divNum := 0
	if len(args) > 0 && args[0] != "" {
		if n, err := strconv.Atoi(args[0]); err == nil {
			divNum = n
		}
	}
	e.currentDiv = divNum
	return "", nil
}

func builtinUndivert(e *Engine, args []string) (string, error) {
	var targetDivs []int
	if len(args) == 0 {
		for d := range e.diversions {
			if d > 0 {
				targetDivs = append(targetDivs, d)
			}
		}
	} else {
		for _, arg := range args {
			if d, err := strconv.Atoi(arg); err == nil && d > 0 {
				targetDivs = append(targetDivs, d)
			}
		}
	}

	var out strings.Builder
	for _, d := range targetDivs {
		if buf, exists := e.diversions[d]; exists {
			out.WriteString(buf.String())
			delete(e.diversions, d)
		}
	}
	return out.String(), nil
}

func builtinInclude(e *Engine, args []string) (string, error) {
	if len(args) == 0 || args[0] == "" {
		return "", fmt.Errorf("include: missing file path")
	}
	content, err := os.ReadFile(args[0])
	if err != nil {
		return "", fmt.Errorf("include(%s): %w", args[0], err)
	}
	return string(content), nil
}

// dnl: Delete to NewLine (改行文字までを無視する)
func builtinDnl(e *Engine, args []string) (string, error) {
	for {
		r, ok := e.lexer.readRune()
		if !ok || r == '\n' {
			break
		}
	}
	return "", nil
}
