package cli

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/xsigil/go-safe-m4/pkg/engine"
)

func RunM4Command(args []string) error {
	fs := flag.NewFlagSet("m4", flag.ContinueOnError)
	defines := make(MapFlag)
	fs.Var(&defines, "D", "Define macro variable (e.g. -DNAME=VAL)")

	if err := fs.Parse(NormalizeDFlags(args)); err != nil {
		return err
	}

	eng := engine.NewEngine()
	for k, v := range defines {
		eng.Define(k, v)
	}

	files := fs.Args()
	if len(files) == 0 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		out, err := eng.Expand(string(input))
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
		out, err := eng.Expand(string(content))
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", file, err)
		}
		fmt.Print(out)
	}

	return nil
}
