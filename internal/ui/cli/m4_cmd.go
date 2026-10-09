package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	pkgm4 "github.com/xsigil/m4llama/pkg/m4"
)

func RunM4Command(args []string) error {
	fs := flag.NewFlagSet("m4", flag.ContinueOnError)
	defines := make(MapFlag)
	fs.Var(&defines, "D", "Define macro variable (e.g. -DNAME=VAL)")

	if err := fs.Parse(NormalizeDFlags(args)); err != nil {
		return err
	}

	engine := pkgm4.NewEngine()
	for k, v := range defines {
		engine.Define(k, v)
	}

	files := fs.Args()
	if len(files) == 0 {
		// 標準入力から読み込み
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		out, err := engine.Expand(string(input))
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}
		out, err := engine.Expand(string(content))
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", file, err)
		}
		fmt.Print(out)
	}

	return nil
}
