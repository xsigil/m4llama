package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xsigil/m4llama/internal/infrastructure/grammar"
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

	grammarFlag := fs.String("grammar", "", "Path to custom GBNF grammar file")
	presetFlag := fs.String("grammar-preset", "", "Built-in grammar preset name (e.g. http_request, smtp_session)")
	editGrammar := fs.Bool("edit-grammar", false, "Extract grammar to /tmp and edit in $EDITOR before inference")
	listGrammars := fs.Bool("list-grammars", false, "List available built-in grammar presets")

	if err := fs.Parse(NormalizeDFlags(args)); err != nil {
		return err
	}

	reg := grammar.GetRegistry()

	if *listGrammars {
		fmt.Println("Available GBNF Grammar Presets:")
		for _, p := range reg.List() {
			fmt.Printf("  %-15s %s (%s)\n", p.Name, p.Title, p.Description)
		}
		return nil
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

	// 文法指定の解決
	var grammarContent string
	var grammarBaseName string

	if *presetFlag != "" {
		p, ok := reg.Get(*presetFlag)
		if !ok {
			return fmt.Errorf("unknown grammar preset: %q (use --list-grammars)", *presetFlag)
		}
		grammarContent = p.Content
		grammarBaseName = p.Name
	} else if *grammarFlag != "" {
		b, err := os.ReadFile(*grammarFlag)
		if err != nil {
			return fmt.Errorf("failed to read grammar file: %w", err)
		}
		grammarContent = string(b)
		grammarBaseName = "custom"
	}

	// -e / --edit-grammar オプション処理 ($EDITOR 起動)
	if *editGrammar {
		tmpPath, err := grammar.PrepareTempFile(grammarBaseName, grammarContent)
		if err != nil {
			return err
		}
		defer os.Remove(tmpPath)

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
			if editor == "" {
				editor = "vi"
			}
		}

		cmd := exec.Command(editor, tmpPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		editedBytes, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("failed to read edited grammar: %w", err)
		}
		grammarContent = string(editedBytes)
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
		Grammar:         grammarContent,
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