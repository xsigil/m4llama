package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ScanTemplates は指定されたルート群から .m4 ファイルを再帰スキャンし、重複を排除して返します
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
				// パスを正規化して重複判定
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

type selectorModel struct {
	allTemplates []string
	filtered     []string
	query        string
	cursor       int
	selected     string
	quitting     bool
	canceled     bool
}

func initialSelectorModel(templates []string) selectorModel {
	return selectorModel{
		allTemplates: templates,
		filtered:     templates,
		cursor:       0,
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
				m.selected = m.filtered[m.cursor]
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
	m.filtered = FuzzyRank(m.query, m.allTemplates)
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
	s.WriteString(countStyle.Render(fmt.Sprintf("[%d / %d matches]", len(m.filtered), len(m.allTemplates))) + "\n\n")

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
		path := m.filtered[i]
		if i == m.cursor {
			s.WriteString(cursorStyle.Render(fmt.Sprintf(" > %s", path)) + "\n")
		} else {
			s.WriteString(itemStyle.Render(fmt.Sprintf("   %s", path)) + "\n")
		}
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	s.WriteString("\n" + helpStyle.Render("Use ↑/↓ or Ctrl+N/Ctrl+P to navigate • Enter to select • Esc/Ctrl+C to quit"))

	return s.String()
}

// SelectTemplate は入力ごとに動的に Levenshtein 距離で絞り込まれる TUI セレクターを起動します
func SelectTemplate(templates []string) (string, error) {
	if len(templates) == 0 {
		return "", fmt.Errorf("no .m4 templates found")
	}
	if len(templates) == 1 {
		return templates[0], nil
	}

	p := tea.NewProgram(initialSelectorModel(templates))
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
