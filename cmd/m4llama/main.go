package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
		// -H/--host, -p/--port のみを先行抽出して残りを透過
		host, port, remainingArgs := extractHostPort(subArgs)
		baseURL := resolveBaseURL(host, port)

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

		// --list-grammars, --grammar-preset, -D, --curl 等はすべてここで安全にパースされる
		if err := cli.RunPromptCommand(ctx, queryUC, remainingArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "tui":
		host, port, remaining := extractHostPort(subArgs)
		stdoutMode := false
		for _, arg := range remaining {
			if arg == "-s" || arg == "--stdout" {
				stdoutMode = true
			}
		}

		baseURL := resolveBaseURL(host, port)
		runTUI(dbPath, templateDir, baseURL, stdoutMode)

	default:
		printUsage()
		os.Exit(1)
	}
}

// extractHostPort は subArgs から接続先フラグ (-H/--host, -p/--port) のみを抽出し、
// その他のフラグ・引数は順番を保ったまま返します。
func extractHostPort(args []string) (host string, port int, rest []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-H" || arg == "--host":
			if i+1 < len(args) {
				host = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-H="):
			host = strings.TrimPrefix(arg, "-H=")
		case strings.HasPrefix(arg, "--host="):
			host = strings.TrimPrefix(arg, "--host=")

		case arg == "-p" || arg == "--port":
			if i+1 < len(args) {
				port, _ = strconv.Atoi(args[i+1])
				i++
			}
		case strings.HasPrefix(arg, "-p="):
			port, _ = strconv.Atoi(strings.TrimPrefix(arg, "-p="))
		case strings.HasPrefix(arg, "--port="):
			port, _ = strconv.Atoi(strings.TrimPrefix(arg, "--port="))

		default:
			rest = append(rest, arg)
		}
	}
	return host, port, rest
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
	fmt.Println("  tui  [-H host] [-p port] [-s]                 Start interactive cockpit (Default when no args given)")
	fmt.Println("  m4   [m4-args...]                             Run pure-Go standalone POSIX m4 macro processor")
	fmt.Println("  run  [-H host] [-p port] [run-flags] [file]   Expand prompt template and execute local LLM inference")
	fmt.Println("\nRun Flags:")
	fmt.Println("  --list-grammars            List available GBNF grammar presets")
	fmt.Println("  --grammar-preset <NAME>    Apply built-in GBNF grammar preset")
	fmt.Println("  --grammar <PATH>           Apply custom GBNF grammar from file")
	fmt.Println("  --edit-grammar             Open grammar in $EDITOR before sending request")
	fmt.Println("  --curl                     Output curl payload without executing inference")
	fmt.Println("  -D VAR=VAL                 Define m4 template variables")
	fmt.Println("\nConnection Flags:")
	fmt.Println("  -H, --host string   Llama server host (default: 127.0.0.1 or LLAMA_SERVER_URL)")
	fmt.Println("  -p, --port int      Llama server port (default: 8080 or LLAMA_SERVER_URL)")
	fmt.Println("  -s, --stdout        Output completion to stdout and exit on finish (tui only)")
}
