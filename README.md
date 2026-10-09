# m4llama

A high-performance terminal LLM cockpit and CLI tool powered by a pure-Go POSIX M4 macro engine and SQLite-backed context management for `llama.cpp` server (`llama-server`).

`m4llama` marries the UNIX philosophy of macro preprocessing with an ultra-responsive, zero-flicker dual-pane TUI. It enables surgical prompt manipulation (**Context Sniping**), real-time SSE token streaming, smooth `$EDITOR` integration, and seamless CLI pipeline interop (`wl-copy`, stdout piping).

---

## What's New in v0.2.0

- **Dual-Pane TUI Cockpit (Yazi / Midnight Commander style)**:
  - Tab 1: **History** (SQLite persistent storage with chronological context preview).
  - Tab 2: **Templates** (Directory tree explorer with inline `mv`, `cp`, and creation).
  - Dedicated focus switching (`l`/`→` to preview, `h`/`←`/`Esc` to return).
  - Vim-style preview scrolling: `j`/`k` (line), `d`/`u` (half-page), `g`/`G` (top/bottom).
- **Context Sniping (Surgical History Sequencing)**:
  - Chain prompt messages directly from past IDs with range and role override syntax (`s33,40~42,$`).
  - Messages are automatically validated, sanitized (no empty assistant turns), and ordered chronologically to prevent 400 Bad Request errors.
- **Real-Time Token Streaming (SSE)**:
  - Instant streaming preview with live token emission counters. Never guess if the GPU is generating.
  - Safe default token limits (4096 tokens) with 10-minute HTTP timeouts to support long-form reasoning models (DeepSeek-R1, Qwen-Coder).
- **Clipboard & UNIX Pipeline Integration**:
  - `y`: Yank completion directly to system clipboard via `wl-copy` (with `xclip` fallback).
  - `w`: **Write & Quit** — Output the active or latest completion directly to `stdout` upon exit, ideal for chaining with `pbcopy`, `glow`, or shell pipes.
- **Interactive History Pruning**:
  - `d`: Delete history records from SQLite with a double-border confirmation modal.
- **Flicker-Free Dynamic Forms**:
  - Annotations (`# @var`) trigger `huh` interactive forms with terminal screen isolation to eliminate layout glitches.

---

## Key Features

- **Pure-Go POSIX M4 Engine**: Embedded, dependency-free M4 macro processor (`define`, `ifelse`, `divert`, `include`, `dnl`).
- **Context Sniping Engine**: Surgical injection of past conversations without context bloat or sequence corruption.
- **Live SSE Token Streaming**: Low-latency incremental token rendering directly in the Bubble Tea viewport.
- **In-TUI Template Operations**: Rename (`m`), copy (`y`), edit (`e`), and run dynamic variable forms (`r`/`Space`).
- **History Surgery**: Press `e` in History to edit past prompts or completions directly in `$EDITOR`.

---

## Installation

Ensure Go 1.22+ is installed:

```bash
git clone [https://github.com/xsigil/m4llama.git](https://github.com/xsigil/m4llama.git)
cd m4llama
make
sudo cp bin/m4llama /usr/local/bin/

```

---

## Configuration

Set the following environment variables in your shell configuration (`~/.zshrc` or `~/.bashrc`):

```bash
# Path to your template directory (default: ~/.config/m4llama/templates)
export M4LLAMA_WORKSPACE="$HOME/.config/m4llama/templates"

# Endpoint of your llama.cpp server (default: [http://127.0.0.1:8080](http://127.0.0.1:8080))
export LLAMA_SERVER_URL="[http://127.0.0.1:8080](http://127.0.0.1:8080)"

# Preferred editor for prompt authoring (default: nvim)
export EDITOR="nvim"

```

---

## Keyboard Shortcuts (Cockpit)

### Global & General

| Key | Action |
| --- | --- |
| `Tab` | Switch between **1:History** and **2:Templates** |
| `l` / `→` | Focus **Right Preview Pane** |
| `h` / `←` / `Esc` | Return to **Left List/Tree Pane** |
| `/` | Filter history or search templates |
| `w` | **Write & Quit**: Output current completion to `stdout` and exit |
| `q` / `Ctrl+C` | Quit cockpit (no stdout) |

### 1:History Tab

| Key | Action |
| --- | --- |
| `j` / `k` | Move selection up / down |
| `Space` | Toggle history selection into current Chain |
| `c` / `:` | Focus Chain input editor |
| `e` | Open selected prompt/completion in `$EDITOR` to rewrite SQLite history |
| `y` | **Yank**: Copy completion directly to clipboard (`wl-copy` / `xclip`) |
| `d` | **Delete**: Prompt confirmation modal to delete history record |
| `Enter` | Run inference with currently defined Chain |

### 2:Templates Tab

| Key | Action |
| --- | --- |
| `j` / `k` | Navigate files and directories |
| `Enter` | Set selected template as Active `(*)` |
| `r` / `Space` | Open dynamic form and run inference |
| `e` | Edit template in `$EDITOR` |
| `m` | Move / Rename template |
| `y` | Copy template |

### Preview Pane Focused

| Key | Action |
| --- | --- |
| `j` / `k` | Scroll preview 1 line down / up |
| `d` / `u` | Scroll half-page down / up |
| `g` / `G` | Jump to top / bottom |
| `y` | Yank completion to clipboard |
| `w` | Write & Quit |
| `h` / `Esc` | Return focus to left list |

---

## Context Sniping Syntax

You can build tailored context chains in the `Chain:` input line:

* `$` : Represents the current prompt being expanded.
* `42` : Injects History `#42` (Prompt as `user`, Completion as `assistant`).
* `33~36` : Injects chronological range from `#33` to `#36`.
* `s10` : Role override — Injects History `#10` as a `system` prompt.
* `u15` : Role override — Injects History `#15` as a `user` prompt.
* `a20` : Role override — Injects History `#20` as an `assistant` prompt.

**Example:**

```text
Chain: s1,33~35,$

```

*(Injects `#1` as system instructions, followed by conversational history `#33` to `#35`, concluding with the current prompt `$`)*.

---

## Pipeline & CLI Workflows

### 1. Interactive Write & Quit (`w`)

Run the TUI, refine your prompt, and send the result directly to your clipboard or a Markdown file:

```bash
m4llama tui -p 28080 | glow -
m4llama tui -p 28080 > response.md

```

### 2. Headless CLI Run

```bash
m4llama run --max-tokens 4096 \
  -DROLE="Senior Go Architect" \
  -DLANG="Go" \
  templates/tasks/code_review.m4

```

### 3. Curl Command Generator

```bash
m4llama run --curl -DNAME="Alice" templates/greeting.m4

```

### 4. Standalone M4 Preprocessing

```bash
m4llama m4 -DFOO=bar input.m4
cat input.m4 | m4llama m4 -DDEBUG=1

```

---

## Template Annotations

`m4llama` parses leading annotations to dynamically generate CLI forms:

```m4
# @var TASK [string] "Task summary" "Refactor auth middleware"
# @var STYLE [select:strict,concise,elaborate] "Output style" "concise"
# @var CODE [text] "Source code snippet"
include(`common/header.m4')dnl

Style: STYLE
Task: TASK

CODE

```

---

## License

MIT License.
