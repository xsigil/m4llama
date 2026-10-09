package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	infram4 "github.com/xsigil/m4llama/internal/infrastructure/m4"
	"github.com/xsigil/m4llama/internal/usecase"
)

var activeProgram *tea.Program

type Runner struct {
	queryUC     *usecase.QueryLLMUseCase
	historyUC   *usecase.HistoryQueryUseCase
	m4Expander  *infram4.Expander
	templateDir string
	stdoutMode  bool
}

func NewRunner(queryUC *usecase.QueryLLMUseCase, historyUC *usecase.HistoryQueryUseCase, expander *infram4.Expander, templateDir string, stdoutMode bool) *Runner {
	return &Runner{
		queryUC:     queryUC,
		historyUC:   historyUC,
		m4Expander:  expander,
		templateDir: templateDir,
		stdoutMode:  stdoutMode,
	}
}

func (r *Runner) Start(ctx context.Context) error {
	cockpit := NewCockpit(r.queryUC, r.historyUC, r.m4Expander, r.templateDir, r.stdoutMode)
	p := tea.NewProgram(cockpit, tea.WithAltScreen())
	activeProgram = p
	defer func() { activeProgram = nil }()

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if m, ok := finalModel.(CockpitModel); ok {
		if m.OutputText != "" && (r.stdoutMode || m.quitting) {
			fmt.Print(m.OutputText)
		}
	}

	return nil
}
