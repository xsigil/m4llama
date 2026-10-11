package grammar

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

//go:embed assets/*.gbnf
var embeddedGrammars embed.FS

// Preset は組み込みの文法プリセット定義です
type Preset struct {
	Name        string
	Title       string
	Description string
	Filename    string
	Content     string
}

// Registry は利用可能な文法プリセットを保持・管理します
type Registry struct {
	presets map[string]Preset
}

var defaultRegistry *Registry

func init() {
	defaultRegistry = &Registry{
		presets: make(map[string]Preset),
	}
	_ = defaultRegistry.loadEmbedded()
}

// GetRegistry はデフォルトの Registry インスタンスを返します
func GetRegistry() *Registry {
	return defaultRegistry
}

func (r *Registry) loadEmbedded() error {
	entries, err := fs.ReadDir(embeddedGrammars, "assets")
	if err != nil {
		return err
	}

	meta := map[string]struct {
		title string
		desc  string
	}{
		"http_request": {"HTTP/1.1 Request", "RFC 7230 compliant raw HTTP request format"},
		"smtp_session": {"SMTP Session", "RFC 5321 client command sequence (EHLO to QUIT)"},
		"sip_request":  {"SIP Request", "RFC 3261 SIP/2.0 signaling request format"},
		"pop3_session": {"POP3 Session", "RFC 1939 client retrieval sequence"},
		"irc_message":  {"IRC Message", "RFC 1459 text chat command format"},
		"rtsp_request": {"RTSP Request", "RFC 2326 streaming control request"},
		"cypher":       {"openCypher Query", "Read-only MATCH ... RETURN graph query"},
		"pikchr":       {"Pikchr Diagram", "Diagram DSL format for technical documentation"},
		"csv":          {"CSV Format", "RFC 4180 comma-separated values"},
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gbnf") {
			continue
		}

		content, err := embeddedGrammars.ReadFile("assets/" + entry.Name())
		if err != nil {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".gbnf")
		title := name
		desc := "GBNF grammar specification"

		if m, ok := meta[name]; ok {
			title = m.title
			desc = m.desc
		}

		r.presets[name] = Preset{
			Name:        name,
			Title:       title,
			Description: desc,
			Filename:    entry.Name(),
			Content:     string(content),
		}
	}
	return nil
}

// List は登録済みのプリセット名一覧をソートして返します
func (r *Registry) List() []Preset {
	var list []Preset
	for _, p := range r.presets {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}

// Get は指定された名前のプリセットを取得します
func (r *Registry) Get(name string) (Preset, bool) {
	p, ok := r.presets[name]
	return p, ok
}

// PrepareTempFile はプリセットまたは任意の文法を /tmp に書き出してパスを返します
func PrepareTempFile(name, content string) (string, error) {
	pattern := fmt.Sprintf("m4llama_grammar_%s_*.gbnf", name)
	if name == "" {
		pattern = "m4llama_grammar_custom_*.gbnf"
	}

	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary grammar file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write grammar content: %w", err)
	}

	return tmpFile.Name(), nil
}

// LoadFileOrPreset はファイルパスまたはプリセット名から GBNF 文字列を取得します
func (r *Registry) LoadFileOrPreset(input string) (string, error) {
	if p, ok := r.Get(input); ok {
		return p.Content, nil
	}

	if _, err := os.Stat(input); err == nil {
		content, err := os.ReadFile(input)
		if err != nil {
			return "", fmt.Errorf("failed to read grammar file %q: %w", input, err)
		}
		return string(content), nil
	}

	return "", fmt.Errorf("unknown grammar preset or file: %q", input)
}
