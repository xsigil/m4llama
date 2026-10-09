# m4llama

A terminal cockpit and CLI tool for templated local LLM inference powered by a pure-Go M4 macro engine and `llama.cpp` server[cite: 1].

`m4llama` combines the UNIX philosophy of macro preprocessing with an interactive TUI workflow[cite: 1]. It lets you structure reusable, modular prompts with standard M4 directives (`define`, `include`, `ifelse`), edit them seamlessly inside Neovim without IME issues, and run inferences against local or remote `llama-server` instances[cite: 1].

---

## Features

- **Pure-Go M4 Macro Engine**: Embedded POSIX-compliant M4 processor with macro expansion (`define`, `ifelse`, `divert`, `include`, etc.)[cite: 1].
- **Interactive TUI Cockpit**: Built with Charm's `bubbletea` and `huh` for fuzzy template discovery, dynamic variable forms, and actions[cite: 1].
- **Seamless Editor Loop**: Edit templates and prompts directly inside `$EDITOR` (Neovim by default) to comfortably use native IME for Japanese and complex inputs.
- **Hierarchical Workspace Management**: Discover and organize prompt templates with relative path navigation via `M4LLAMA_WORKSPACE`.
- **Reasoning & Token Control**: Native support for reasoning models (`reasoning_content`) and token caps to avoid unbounded generation loops[cite: 1].
- **cURL Exporter**: Export exact inference requests as executable `curl` commands for reproduction and debugging[cite: 1].

---

## Installation

### From Source

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

## Usage

### 1. Interactive TUI (Default)

Running `m4llama` without arguments launches the interactive picker:

```bash
m4llama

```

* **Fuzzy Search**: Filter templates by relative workspace path using Levenshtein ranking.


* **[➕ Create New Template]**: Create and immediately edit a new `.m4` file inside Neovim.
* **Actions**:
* `🚀 Run Inference`: Fill dynamic variables defined in the template header and run inference.


* `📝 Edit in Neovim`: Open the selected file in Neovim, modify it, and return to the cockpit.
* `📋 Export curl command`: Generate a ready-to-use `curl` command.





### 2. CLI Execution

Directly expand and run templates headlessly:

```bash
m4llama run --max-tokens 256 \
  -DROLE="Senior Go Architect" \
  -DLANG="Go" \
  -DGOAL="Review concurrent pipeline" \
  templates/tasks/code_review.m4

```

Export as cURL without running:

```bash
m4llama run --curl -DNAME="Alice" templates/greeting.m4

```

### 3. Standalone M4 Preprocessor

Run pure-Go M4 macro expansion on arbitrary files or standard input:

```bash
m4llama m4 -DFOO=bar input.m4
cat input.m4 | m4llama m4 -DDEBUG=1

```

---

## Template Annotations

`m4llama` parses leading comment annotations to dynamically construct interactive TUI forms:

```m4
# @var LANG [string] "Target programming language" "Go"
# @var STYLE [select:strict,friendly,normal] "Review style" "strict"
# @var CODE [text] "Source code snippet to review"
include(`common/safety.m4')dnl
include(`common/persona.m4')dnl

SELECT_PERSONA(STYLE)

Target Language: LANG

SAFETY_RULES

Review the following code:
CODE
```

---

## License

MIT License.
