package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ScanTemplates は指定されたルート群から .m4 ファイルをスキャンし、フルパスのリストを返します
func ScanTemplates(roots ...string) ([]string, error) {
	seen := make(map[string]bool)
	var templates []string

	for _, root := range roots {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".m4") {
				cleanPath := filepath.Clean(path)
				if !seen[cleanPath] {
					seen[cleanPath] = true
					templates = append(templates, cleanPath)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return templates, nil
}

type TemplateItem struct {
	Display  string // セレクターに表示する相対パス
	FullPath string // 実際のフルパス
}

type selectorModel struct {
	items    []TemplateItem
	filtered []TemplateItem
	query    string
	cursor   int
	selected string
	quitting bool
	canceled bool
}

func initialSelectorModel(items []TemplateItem) selectorModel {
	return selectorModel{
		items:    items,
		filtered: items,
		cursor:   0,
	}
}

func (m selectorModel) Init() tea.Cmd {
	return nil
}

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.canceled = true
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = m.filtered[m.cursor].FullPath
			}
			m.quitting = true
			return m, tea.Quit

		case tea.KeyUp, tea.KeyCtrlP:
			if m.cursor > 0 {
				m.cursor--
			}

		case tea.KeyDown, tea.KeyCtrlN:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}

		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.query) > 0 {
				runes := []rune(m.query)
				m.query = string(runes[:len(runes)-1])
				m.updateFiltered()
			}

		case tea.KeyRunes, tea.KeySpace:
			m.query += string(msg.Runes)
			m.updateFiltered()
		}
	}

	return m, nil
}

func (m *selectorModel) updateFiltered() {
	if m.query == "" {
		m.filtered = m.items
		m.cursor = 0
		return
	}

	displays := make([]string, len(m.items))
	for i, it := range m.items {
		displays[i] = it.Display
	}

	ranked := FuzzyRank(m.query, displays)
	m.filtered = make([]TemplateItem, 0, len(ranked))

	displayMap := make(map[string]TemplateItem)
	for _, it := range m.items {
		displayMap[it.Display] = it
	}

	for _, d := range ranked {
		if it, ok := displayMap[d]; ok {
			m.filtered = append(m.filtered, it)
		}
	}
	m.cursor = 0
}

func (m selectorModel) View() string {
	if m.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var s strings.Builder
	s.WriteString(titleStyle.Render("m4llama Template Selection") + "\n")
	s.WriteString(promptStyle.Render("> ") + m.query + "\n")
	s.WriteString(countStyle.Render(fmt.Sprintf("[%d / %d matches]", len(m.filtered), len(m.items))) + "\n\n")

	maxDisplay := 10
	start := 0
	if m.cursor >= maxDisplay {
		start = m.cursor - maxDisplay + 1
	}
	end := start + maxDisplay
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := start; i < end; i++ {
		item := m.filtered[i]
		if i == m.cursor {
			s.WriteString(cursorStyle.Render(fmt.Sprintf(" > %s", item.Display)) + "\n")
		} else {
			s.WriteString(itemStyle.Render(fmt.Sprintf("   %s", item.Display)) + "\n")
		}
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	s.WriteString("\n" + helpStyle.Render("Use ↑/↓ or Ctrl+N/Ctrl+P to navigate • Enter to select • Esc/Ctrl+C to quit"))

	return s.String()
}

// SelectTemplate は WORKSPACE からの相対パスで表示し、選択された実パスを返します
func SelectTemplate(baseDir string, specialOptions []string, fullPaths []string) (string, error) {
	var items []TemplateItem

	// 特殊オプション（[➕ Create New Template] など）
	for _, opt := range specialOptions {
		items = append(items, TemplateItem{
			Display:  opt,
			FullPath: opt,
		})
	}

	// テンプレートパスを相対パス表示に変換
	for _, p := range fullPaths {
		display := p
		if rel, err := filepath.Rel(baseDir, p); err == nil && !strings.HasPrefix(rel, "..") {
			display = rel
		}
		items = append(items, TemplateItem{
			Display:  display,
			FullPath: p,
		})
	}

	if len(items) == 0 {
		return "", fmt.Errorf("no templates available")
	}

	p := tea.NewProgram(initialSelectorModel(items))
	finalModel, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run selector: %w", err)
	}

	m := finalModel.(selectorModel)
	if m.canceled {
		return "", fmt.Errorf("selection canceled")
	}
	if m.selected == "" {
		return "", fmt.Errorf("no template selected")
	}

	return m.selected, nil
}
