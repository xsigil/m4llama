# SPEC.md: m4llama Specification

**Version:** 0.2.0-draft  
**Status:** In Review / Specification Complete  
**Architecture Paradigm:** Domain-Driven Design (DDD), Clean Architecture, Unix Philosophy  
**Target Environment:** POSIX Terminal (Linux / macOS), `llama-server` (OpenAI-compatible local inference)

---

## 1. Overview & Vision

**m4llama** is a terminal-native, highly composable prompt-engineering cockpit and local inference client tailored for `llama-server`.

By combining an embedded pure-Go POSIX `m4` macro processor with interactive fuzzy finding (`fzf`), dynamic TUI form generation (`charmbracelet/huh`), and an ACID-compliant persistence layer (SQLite3 managed via `TxManager`), `m4llama` delivers a deterministic, keyboard-driven alternative to heavy web-based prompt frameworks.

### Core Tenets
1. **Unix-Way Modularity:** Decouple macro expansion (`pkg/m4`), dynamic UI prompt acquisition (`fzf` + `huh`), transport (`net/http` / `curl`), and local persistence (SQLite3).
2. **Zero-Dependency & Self-Contained Engine:** Ships with a bundled pure-Go `m4` subset engine (`pkg/m4`). Host-level GNU `m4` installation is **not required**.
3. **Dual-Role Binary (LLM Client & Standalone m4):** Acts as both a full LLM cockpit and a standalone POSIX `m4` processor via the `m4llama m4` subcommand.
4. **Annotation-Driven UI:** Template headers define runtime variables (`# @var`) that dynamically spin up typed TUI input forms without modifying Go source code.
5. **Contextual Sniping via SQLite3 History:** Store full execution snapshots (templates, variables, expanded prompts, completions, tokens). Allow arbitrary historical outputs to be pinned and injected into subsequent prompts by unique ID.
6. **Zero Cloud Telemetry & Complete Reproducibility:** Every prompt generation is reproducible via CLI flags (`-DVAR=VAL`), dry-run `curl` commands, or interactive TUI sessions.

---

## 2. System Architecture

The project strictly follows Domain-Driven Design (DDD) layered architecture principles:

```text
m4llama/
├── cmd/
│   └── m4llama/
│       └── main.go                 # Subcommand router (m4 / run / tui) & DI wire-up
├── pkg/
│   └── m4/                         # Reusable pure-Go m4 subset macro engine
│       ├── engine.go               # Macro evaluator, quote tracker & stream interpreter
│       ├── builtins.go             # Builtins (define, ifdef, ifelse, changequote, divert, include)
│       └── lexer.go                # Tokenizer & quote/parenthesis balancing
├── internal/
│   ├── domain/                     # Pure domain logic (zero external dependencies)
│   │   ├── entity/
│   │   │   ├── history.go          # Conversation history & token usage entities
│   │   │   ├── completion.go       # Inference request/response models
│   │   │   └── template_var.go     # Extracted template variable definitions & types
│   │   └── repository/
│   │       ├── macro_expander.go   # Macro expansion abstraction contract
│   │       ├── llm_client.go       # llama-server communication contract
│   │       └── history_repo.go     # Conversation persistence interface
│   ├── usecase/                    # Business use cases & transaction orchestration
│   │   ├── query_llm.go            # Expansion -> Context Assembly -> Inference -> Tx Save
│   │   └── history_query.go        # Historical retrieval and branching operations
│   ├── infrastructure/             # Concrete technology adapters
│   │   ├── m4/
│   │   │   ├── expander.go         # Adapter implementing repository.MacroExpander via pkg/m4
│   │   │   └── parser.go           # Template annotation header scanner (# @var)
│   │   ├── llamaserver/
│   │   │   ├── client.go           # net/http client for /v1/chat/completions & /completion
│   │   │   └── curl_emitter.go     # Deterministic curl CLI command generator
│   │   └── sqlite3/
│   │       ├── db.go               # SQLite3 driver connection & schema initialization
│   │       ├── tx_manager.go       # Context-scoped TxManager for ACID transactions
│   │       └── history_repo.go     # Repository implementation using TxManager
│   └── ui/                         # User Interface adapters
│       ├── cli/                    # Headless CLI flags & subcommands (Cobra / Flag)
│       └── tui/
│           ├── runner.go           # Interactive REPL & workflow coordinator
│           ├── fzf_selector.go     # Subprocess fzf template & history picker
│           └── dynamic_form.go     # charmbracelet/huh dynamic form builder
├── templates/                      # User-defined .m4 macro templates
│   ├── base.m4                     # Common system personas and macro guards
│   └── analysis.m4                 # Task-specific template examples
├── migrations/
│   └── 001_init.sql                # SQLite3 DDL
├── Makefile
├── README.md
└── SPEC.md
