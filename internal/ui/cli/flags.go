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
