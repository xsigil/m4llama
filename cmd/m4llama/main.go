package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	infram4 "github.com/xsigil/m4llama/internal/infrastructure/m4"
	"github.com/xsigil/m4llama/internal/infrastructure/llamaserver"
	"github.com/xsigil/m4llama/internal/infrastructure/sqlite3"
	"github.com/xsigil/m4llama/internal/ui/cli"
	"github.com/xsigil/m4llama/internal/usecase"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := os.Args[1]
	subArgs := os.Args[2:]

	switch subcmd {
	case "m4":
		if err := cli.RunM4Command(subArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "run":
		ctx := context.Background()

		// DB パスの設定 (~/.local/share/m4llama/history.db)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		dbPath := filepath.Join(homeDir, ".local", "share", "m4llama", "history.db")

		db, err := sqlite3.NewDB(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		baseURL := os.Getenv("LLAMA_SERVER_URL")
		if baseURL == "" {
			baseURL = "http://127.0.0.1:8080"
		}

		expander := infram4.NewExpander()
		llmClient := llamaserver.NewClient(baseURL, 120*time.Second)
		historyRepo := sqlite3.NewHistoryRepository(db)
		queryUC := usecase.NewQueryLLMUseCase(expander, llmClient, historyRepo)

		if err := cli.RunPromptCommand(ctx, queryUC, subArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: m4llama <subcommand> [flags] [args]")
	fmt.Println("\nSubcommands:")
	fmt.Println("  m4    Run pure-Go standalone POSIX m4 macro processor")
	fmt.Println("  run   Expand prompt template and execute local LLM inference")
}
