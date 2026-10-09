package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xsigil/m4llama/internal/usecase"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func RunPromptCommand(ctx context.Context, queryUC *usecase.QueryLLMUseCase, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)

	var defs arrayFlags
	fs.Var(&defs, "D", "Define variable: NAME=VALUE")
	model := fs.String("model", "/models/model.gguf", "Model name or identifier")
	temp := fs.Float64("temperature", 0.7, "Temperature")
	maxTokens := fs.Int("max-tokens", -1, "Max tokens to generate (-1: unlimited)")
	dryRun := fs.Bool("curl", false, "Output curl command without executing inference")

	if err := fs.Parse(NormalizeDFlags(args)); err != nil {
		return err
	}

	rest := fs.Args()
	if len(rest) == 0 {
		return fmt.Errorf("missing template file path")
	}
	tmplPath := rest[0]

	contentBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	variables := make(map[string]string)
	for _, d := range defs {
		parts := strings.SplitN(d, "=", 2)
		if len(parts) == 2 {
			variables[parts[0]] = parts[1]
		} else {
			variables[parts[0]] = ""
		}
	}

	out, err := queryUC.Execute(ctx, usecase.QueryLLMInput{
		TemplateName:    tmplPath,
		TemplateContent: string(contentBytes),
		Variables:       variables,
		Model:           *model,
		Temperature:     *temp,
		MaxTokens:       *maxTokens,
		DryRunCurl:      *dryRun,
	})
	if err != nil {
		return err
	}

	if *dryRun {
		fmt.Println(out.CurlCommand)
		return nil
	}

	fmt.Printf("[Completion ID: #%d | Total Tokens: %d]\n\n", out.HistoryID, out.Tokens.TotalTokens)
	fmt.Println(out.Completion)
	return nil
}
