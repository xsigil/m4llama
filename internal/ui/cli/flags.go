package cli

import (
	"fmt"
	"strings"
)

type MapFlag map[string]string

func (m MapFlag) String() string {
	var pairs []string
	for k, v := range m {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(pairs, ", ")
}

func (m MapFlag) Set(val string) error {
	parts := strings.SplitN(val, "=", 2)
	if len(parts) == 1 {
		m[parts[0]] = ""
	} else {
		m[parts[0]] = parts[1]
	}
	return nil
}

// NormalizeDFlags は -DVAR=VAL や --DVAR=VAL の形式を ["-D", "VAR=VAL"] に分割正規化します
func NormalizeDFlags(args []string) []string {
	var normalized []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-D") && len(arg) > 2 && arg[2] != '=' {
			normalized = append(normalized, "-D", arg[2:])
		} else if strings.HasPrefix(arg, "--D") && len(arg) > 3 && arg[3] != '=' {
			normalized = append(normalized, "-D", arg[3:])
		} else {
			normalized = append(normalized, arg)
		}
	}
	return normalized
}
