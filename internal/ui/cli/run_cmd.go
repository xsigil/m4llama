package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/xsigil/m4llama/internal/usecase"
)

func RunPromptCommand(ctx context.Context, queryUC *usecase.QueryLLMUseCase, args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)

	defines := make(MapFlag)
	fs.Var(&defines, "D", "Template variable (e.g. -DVAR=VAL)")

	model := fs.String("model", "default", "Target model name")
	temp := fs.Float64("temperature", 0.7, "Temperature parameter")
	dryRun := fs.Bool("dry-run", false, "Output curl command without sending inference request")
	exportCurl := fs.Bool("export-curl", false, "Alias for --dry-run")

	if err := fs.Parse(args); err != nil {
		return err
	}

	templateFiles := fs.Args()
	if len(templateFiles) == 0 {
		return fmt.Errorf("usage: m4llama run [flags] <template.m4>")
	}

	tmplPath := templateFiles[0]
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	out, err := queryUC.Execute(ctx, usecase.QueryLLMInput{
		TemplateName:    tmplPath,
		TemplateContent: string(content),
		Variables:       defines,
		Model:           *model,
		Temperature:     *temp,
		DryRunCurl:      *dryRun || *exportCurl,
	})
	if err != nil {
		return err
	}

	if *dryRun || *exportCurl {
		fmt.Println(out.CurlCommand)
		return nil
	}

	fmt.Println(out.Completion)
	return nil
}
