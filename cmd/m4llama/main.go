package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xsigil/m4llama/internal/infrastructure/llamaserver"
	infram4 "github.com/xsigil/m4llama/internal/infrastructure/m4"
	"github.com/xsigil/m4llama/internal/infrastructure/sqlite3"
	"github.com/xsigil/m4llama/internal/ui/cli"
	"github.com/xsigil/m4llama/internal/ui/tui"
	"github.com/xsigil/m4llama/internal/usecase"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	dbPath := filepath.Join(homeDir, ".local", "share", "m4llama", "history.db")

	templateDir := os.Getenv("M4LLAMA_WORKSPACE")
	if templateDir == "" {
		if _, err := os.Stat("templates"); err == nil {
			templateDir = "templates"
		} else {
			templateDir = filepath.Join(homeDir, ".config", "m4llama", "templates")
		}
	}

	// 引数なし起動はデフォルト設定で TUI を開始
	if len(os.Args) < 2 {
		baseURL := resolveBaseURL("", 0)
		runTUI(dbPath, templateDir, baseURL, false)
		return
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
		runFlags := flag.NewFlagSet("run", flag.ContinueOnError)
		hostFlag := runFlags.String("host", "", "Target host (default: 127.0.0.1)")
		runFlags.StringVar(hostFlag, "H", "", "Target host (shorthand)")
		portFlag := runFlags.Int("port", 0, "Target port (default: 8080)")
		runFlags.IntVar(portFlag, "p", 0, "Target port (shorthand)")

		if err := runFlags.Parse(subArgs); err != nil {
			os.Exit(1)
		}
		remainingArgs := runFlags.Args()

		baseURL := resolveBaseURL(*hostFlag, *portFlag)

		ctx := context.Background()
		db, err := sqlite3.NewDB(dbPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing DB: %v\n", err)
			os.Exit(1)
		}
		defer db.Close()

		expander := infram4.NewExpander()
		llmClient := llamaserver.NewClient(baseURL, 600*time.Second)
		historyRepo := sqlite3.NewHistoryRepository(db)
		queryUC := usecase.NewQueryLLMUseCase(expander, llmClient, historyRepo)

		if err := cli.RunPromptCommand(ctx, queryUC, remainingArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "tui":
		tuiFlags := flag.NewFlagSet("tui", flag.ContinueOnError)
		hostFlag := tuiFlags.String("host", "", "Target host (default: 127.0.0.1)")
		tuiFlags.StringVar(hostFlag, "H", "", "Target host (shorthand)")
		portFlag := tuiFlags.Int("port", 0, "Target port (default: 8080)")
		tuiFlags.IntVar(portFlag, "p", 0, "Target port (shorthand)")
		stdoutFlag := tuiFlags.Bool("stdout", false, "Output completion to stdout and exit on finish")
		tuiFlags.BoolVar(stdoutFlag, "s", false, "Output completion to stdout (shorthand)")

		if err := tuiFlags.Parse(subArgs); err != nil {
			os.Exit(1)
		}

		baseURL := resolveBaseURL(*hostFlag, *portFlag)
		runTUI(dbPath, templateDir, baseURL, *stdoutFlag)

	default:
		printUsage()
		os.Exit(1)
	}
}

func resolveBaseURL(host string, port int) string {
	rawURL := os.Getenv("LLAMA_SERVER_URL")
	if rawURL == "" {
		rawURL = "http://127.0.0.1:8080"
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		u = &url.URL{
			Scheme: "http",
			Host:   "127.0.0.1:8080",
		}
	}

	currHost := u.Hostname()
	currPort := u.Port()

	if host != "" {
		currHost = host
	}
	if port != 0 {
		currPort = fmt.Sprintf("%d", port)
	}

	if currPort != "" {
		u.Host = fmt.Sprintf("%s:%s", currHost, currPort)
	} else {
		u.Host = currHost
	}

	return strings.TrimRight(u.String(), "/")
}

func runTUI(dbPath, templateDir, baseURL string, stdoutMode bool) {
	ctx := context.Background()
	db, err := sqlite3.NewDB(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	expander := infram4.NewExpander()
	llmClient := llamaserver.NewClient(baseURL, 600*time.Second)
	historyRepo := sqlite3.NewHistoryRepository(db)
	queryUC := usecase.NewQueryLLMUseCase(expander, llmClient, historyRepo)
	historyUC := usecase.NewHistoryQueryUseCase(historyRepo)

	runner := tui.NewRunner(queryUC, historyUC, expander, templateDir, stdoutMode)
	if err := runner.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "TUI Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: m4llama [subcommand] [flags] [args]")
	fmt.Println("\nSubcommands:")
	fmt.Println("  tui  [-H host] [-p port] [-s]  Start interactive cockpit (Default when no args given)")
	fmt.Println("  m4   [m4-args...]              Run pure-Go standalone POSIX m4 macro processor")
	fmt.Println("  run  [-H host] [-p port]       Expand prompt template and execute local LLM inference")
	fmt.Println("\nFlags:")
	fmt.Println("  -H, --host string   Llama server host (default: 127.0.0.1 or LLAMA_SERVER_URL)")
	fmt.Println("  -p, --port int      Llama server port (default: 8080 or LLAMA_SERVER_URL)")
	fmt.Println("  -s, --stdout        Output completion to stdout and exit on finish (tui only)")
}
