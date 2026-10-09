package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xsigil/m4llama/internal/domain/entity"
	infram4 "github.com/xsigil/m4llama/internal/infrastructure/m4"
	"github.com/xsigil/m4llama/internal/usecase"
)

type tabView int

const (
	tabHistory tabView = iota
	tabTemplates
)

type focusField int

const (
	focusList focusField = iota
	focusPreview
	focusFilter
	focusChain
	focusFileAction
	focusDeleteConfirm
)

type fileActionMode int

const (
	actionNone fileActionMode = iota
	actionMove
	actionCopy
)

type CockpitModel struct {
	queryUC     *usecase.QueryLLMUseCase
	historyUC   *usecase.HistoryQueryUseCase
	m4Expander  *infram4.Expander
	templateDir string
	stdoutMode  bool

	width  int
	height int

	currentTab tabView
	focus      focusField

	// History
	historyEntries  []*entity.HistoryEntry
	filteredHistory []*entity.HistoryEntry
	historyCursor   int
	targetDeleteID  int64

	// Templates
	treeItems      []TreeItem
	templateCursor int
	activeTemplate string

	// コンポーネント
	filterInput     textinput.Model
	chainInput      textinput.Model
	fileActionInput textinput.Model
	fileActionKind  fileActionMode
	preview         viewport.Model
	spinner         spinner.Model

	selectedIndices map[int64]SelectedInfo

	isInferring  bool
	streamText   string
	streamTokens int
	statusMsg    string
	quitting     bool
	OutputText   string
}

type formSubmittedMsg struct {
	vars map[string]string
}

type streamTokenMsg string

type inferenceCompleteMsg struct {
	output *usecase.QueryLLMOutput
	err    error
}

func NewCockpit(queryUC *usecase.QueryLLMUseCase, historyUC *usecase.HistoryQueryUseCase, expander *infram4.Expander, templateDir string, stdoutMode bool) CockpitModel {
	fi := textinput.New()
	fi.Prompt = "/ "
	fi.Placeholder = "grep..."

	ci := textinput.New()
	ci.Prompt = "Chain: "
	ci.Placeholder = "e.g. 33~42,$"

	fai := textinput.New()

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	vp := viewport.New(60, 20)

	hist, _ := historyUC.GetRecent(context.Background(), 100)
	tree, _ := BuildTemplateTree(templateDir)

	m := CockpitModel{
		queryUC:         queryUC,
		historyUC:       historyUC,
		m4Expander:      expander,
		templateDir:     templateDir,
		stdoutMode:      stdoutMode,
		currentTab:      tabHistory,
		focus:           focusList,
		historyEntries:  hist,
		filteredHistory: hist,
		treeItems:       tree,
		filterInput:     fi,
		chainInput:      ci,
		fileActionInput: fai,
		preview:         vp,
		spinner:         sp,
		selectedIndices: make(map[int64]SelectedInfo),
	}

	for _, it := range tree {
		if !it.IsDir {
			m.activeTemplate = it.FullPath
			break
		}
	}

	m.resetDefaultChain()
	m.syncSelected()
	m.updatePreview()
	return m
}

func (m *CockpitModel) resetDefaultChain() {
	if len(m.historyEntries) == 0 {
		m.chainInput.SetValue("$")
		return
	}
	latestID := m.historyEntries[0].ID
	count := int64(len(m.historyEntries))
	if count > 10 {
		count = 10
	}
	startID := latestID - count + 1
	if startID < 1 {
		startID = 1
	}
	if startID == latestID {
		m.chainInput.SetValue(fmt.Sprintf("%d,$", latestID))
	} else {
		m.chainInput.SetValue(fmt.Sprintf("%d~%d,$", startID, latestID))
	}
}

func (m CockpitModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m CockpitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()

	case streamTokenMsg:
		m.streamText += string(msg)
		m.streamTokens++
		m.preview.SetContent(m.streamText)
		m.preview.GotoBottom()
		return m, nil

	case formSubmittedMsg:
		if msg.vars != nil {
			m.startInference()
			return m, m.runInferenceStreamCmd(msg.vars)
		}
		return m, nil

	case inferenceCompleteMsg:
		m.isInferring = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Inference error: %v", msg.err)
			m.preview.SetContent(fmt.Sprintf("❌ Error during inference:\n\n%v", msg.err))
		} else {
			m.statusMsg = fmt.Sprintf("Response complete! ID: #%d (%d tokens)", msg.output.HistoryID, msg.output.Tokens.TotalTokens)
			m.OutputText = msg.output.Completion
			if m.stdoutMode {
				m.quitting = true
				return m, tea.Quit
			}

			m.historyEntries, _ = m.historyUC.GetRecent(context.Background(), 100)
			m.applyFilter()
			m.historyCursor = 0
			m.currentTab = tabHistory
			m.resetDefaultChain()
			m.syncSelected()
			m.updatePreview()
		}
		return m, nil

	case tea.KeyMsg:
		if m.isInferring {
			if msg.Type == tea.KeyCtrlC {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		switch m.focus {
		case focusDeleteConfirm:
			switch msg.String() {
			case "y", "Y", "enter":
				if m.targetDeleteID > 0 {
					_ = m.historyUC.DeleteEntry(context.Background(), m.targetDeleteID)
					m.statusMsg = fmt.Sprintf("Deleted history #%d", m.targetDeleteID)
					m.historyEntries, _ = m.historyUC.GetRecent(context.Background(), 100)
					m.applyFilter()
					if m.historyCursor >= len(m.filteredHistory) && m.historyCursor > 0 {
						m.historyCursor--
					}
					m.resetDefaultChain()
					m.syncSelected()
					m.updatePreview()
				}
				m.focus = focusList
				m.targetDeleteID = 0
				return m, nil

			case "n", "N", "esc", "q":
				m.focus = focusList
				m.targetDeleteID = 0
				m.statusMsg = "Deletion canceled"
				return m, nil
			}
			return m, nil

		case focusFilter:
			switch msg.Type {
			case tea.KeyEsc, tea.KeyEnter:
				m.focus = focusList
				m.filterInput.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.applyFilter()
			m.updatePreview()
			return m, cmd

		case focusChain:
			switch msg.Type {
			case tea.KeyEsc, tea.KeyEnter:
				m.focus = focusList
				m.chainInput.Blur()
				m.syncSelected()
				return m, nil
			}
			var cmd tea.Cmd
			m.chainInput, cmd = m.chainInput.Update(msg)
			m.syncSelected()
			return m, cmd

		case focusFileAction:
			switch msg.Type {
			case tea.KeyEsc:
				m.focus = focusList
				m.fileActionInput.Blur()
				m.fileActionKind = actionNone
				return m, nil
			case tea.KeyEnter:
				m.executeFileAction()
				m.focus = focusList
				m.fileActionInput.Blur()
				m.fileActionKind = actionNone
				return m, nil
			}
			var cmd tea.Cmd
			m.fileActionInput, cmd = m.fileActionInput.Update(msg)
			return m, cmd

		case focusPreview:
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				m.OutputText = ""
				return m, tea.Quit

			case "h", "left", "esc":
				m.focus = focusList
				return m, nil

			case "y":
				// プレビューフォーカス時にも y で Completion をクリップボードへコピー
				if m.currentTab == tabHistory && len(m.filteredHistory) > 0 {
					target := m.filteredHistory[m.historyCursor].Completion
					if err := CopyToClipboard(target); err != nil {
						m.statusMsg = fmt.Sprintf("Clipboard error: %v", err)
					} else {
						m.statusMsg = fmt.Sprintf("Copied #%d completion to clipboard via wl-copy!", m.filteredHistory[m.historyCursor].ID)
					}
					return m, nil
				}

			case "j", "down":
				m.preview.LineDown(1)
				return m, nil

			case "k", "up":
				m.preview.LineUp(1)
				return m, nil

			case "d", "ctrl+d":
				m.preview.HalfViewDown()
				return m, nil

			case "u", "ctrl+u":
				m.preview.HalfViewUp()
				return m, nil

			case "g":
				m.preview.GotoTop()
				return m, nil

			case "G":
				m.preview.GotoBottom()
				return m, nil

			case "w":
				if len(m.filteredHistory) > 0 && m.historyCursor < len(m.filteredHistory) {
					m.OutputText = m.filteredHistory[m.historyCursor].Completion
				}
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil

		case focusList:
			switch msg.String() {
			case "q", "ctrl+c":
				m.quitting = true
				m.OutputText = ""
				return m, tea.Quit

			case "y":
				// History タブ: 選択中の Response (Completion) を wl-copy でクリップボードにコピー
				if m.currentTab == tabHistory && len(m.filteredHistory) > 0 {
					target := m.filteredHistory[m.historyCursor].Completion
					if err := CopyToClipboard(target); err != nil {
						m.statusMsg = fmt.Sprintf("Clipboard error: %v", err)
					} else {
						m.statusMsg = fmt.Sprintf("Copied #%d completion to clipboard via wl-copy!", m.filteredHistory[m.historyCursor].ID)
					}
					return m, nil
				}

				// Templates タブ: Yazi風ファイルコピー (cp)
				if m.currentTab == tabTemplates && len(m.treeItems) > 0 {
					m.focus = focusFileAction
					m.fileActionKind = actionCopy
					m.fileActionInput.Prompt = "Copy to: "
					m.fileActionInput.SetValue(m.treeItems[m.templateCursor].RelPath)
					m.fileActionInput.Focus()
					return m, nil
				}

			case "d":
				if m.currentTab == tabHistory && len(m.filteredHistory) > 0 {
					m.targetDeleteID = m.filteredHistory[m.historyCursor].ID
					m.focus = focusDeleteConfirm
					return m, nil
				}

			case "w":
				if len(m.filteredHistory) > 0 && m.historyCursor < len(m.filteredHistory) {
					m.OutputText = m.filteredHistory[m.historyCursor].Completion
				}
				m.quitting = true
				return m, tea.Quit

			case "tab":
				if m.currentTab == tabHistory {
					m.currentTab = tabTemplates
				} else {
					m.currentTab = tabHistory
				}
				m.filterInput.SetValue("")
				m.applyFilter()
				m.updatePreview()
				return m, nil

			case "l", "right":
				m.focus = focusPreview
				return m, nil

			case "j", "down":
				m.cursorDown()
				m.updatePreview()
				return m, nil

			case "k", "up":
				m.cursorUp()
				m.updatePreview()
				return m, nil

			case "ctrl+d":
				m.preview.HalfViewDown()
				return m, nil

			case "ctrl+u":
				m.preview.HalfViewUp()
				return m, nil

			case "/":
				m.focus = focusFilter
				m.filterInput.Focus()
				return m, nil

			case "c", ":":
				if m.currentTab == tabHistory {
					m.focus = focusChain
					m.chainInput.Focus()
					return m, nil
				}

			case "m":
				if m.currentTab == tabTemplates && len(m.treeItems) > 0 {
					m.focus = focusFileAction
					m.fileActionKind = actionMove
					m.fileActionInput.Prompt = "Move/Rename to: "
					m.fileActionInput.SetValue(m.treeItems[m.templateCursor].RelPath)
					m.fileActionInput.Focus()
					return m, nil
				}

			case "e":
				if m.currentTab == tabHistory && len(m.filteredHistory) > 0 {
					targetEntry := m.filteredHistory[m.historyCursor]
					tmpPath, err := PrepareHistoryTempFile(targetEntry)
					if err != nil {
						m.statusMsg = fmt.Sprintf("Error creating temp file: %v", err)
						return m, nil
					}
					c := CreateEditorCmd(tmpPath)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						defer os.Remove(tmpPath)
						if err == nil {
							changed, applyErr := ApplyHistoryTempFile(tmpPath, targetEntry)
							if applyErr == nil && changed {
								_ = m.historyUC.UpdateEntry(context.Background(), targetEntry)
							}
						}
						m.updatePreview()
						return nil
					})
				}
				if m.currentTab == tabTemplates && len(m.treeItems) > 0 && !m.treeItems[m.templateCursor].IsDir {
					target := m.treeItems[m.templateCursor].FullPath
					c := CreateEditorCmd(target)
					return m, tea.ExecProcess(c, func(err error) tea.Msg {
						m.updatePreview()
						return nil
					})
				}

			case " ":
				if m.currentTab == tabHistory {
					m.toggleHistorySelection()
					return m, nil
				}
				return m, m.triggerFormAndRun()

			case "r":
				if m.currentTab == tabTemplates {
					return m, m.triggerFormAndRun()
				}

			case "enter":
				if m.currentTab == tabTemplates {
					if len(m.treeItems) > 0 && !m.treeItems[m.templateCursor].IsDir {
						m.activeTemplate = m.treeItems[m.templateCursor].FullPath
						m.statusMsg = fmt.Sprintf("Active template set to: %s", filepath.Base(m.activeTemplate))
						m.updatePreview()
					}
					return m, nil
				}
				m.startInference()
				return m, m.runInferenceStreamCmd(nil)
			}
		}
	}

	if m.isInferring {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *CockpitModel) startInference() {
	m.isInferring = true
	m.streamText = ""
	m.streamTokens = 0
	m.statusMsg = "Streaming tokens from llama-server..."
	m.preview.SetContent("⚡ Connecting and generating...")
}

func (m *CockpitModel) executeFileAction() {
	if len(m.treeItems) == 0 {
		return
	}
	src := m.treeItems[m.templateCursor].FullPath
	targetRel := strings.TrimSpace(m.fileActionInput.Value())
	if targetRel == "" {
		return
	}
	dst := filepath.Join(m.templateDir, targetRel)

	var err error
	if m.fileActionKind == actionMove {
		err = MoveFile(src, dst)
		if err == nil {
			m.statusMsg = fmt.Sprintf("Moved: %s", targetRel)
		}
	} else if m.fileActionKind == actionCopy {
		err = CopyFile(src, dst)
		if err == nil {
			m.statusMsg = fmt.Sprintf("Copied: %s", targetRel)
		}
	}
	if err != nil {
		m.statusMsg = fmt.Sprintf("Error: %v", err)
	} else {
		m.treeItems, _ = BuildTemplateTree(m.templateDir)
		m.updatePreview()
	}
}

func (m *CockpitModel) triggerFormAndRun() tea.Cmd {
	if len(m.treeItems) == 0 || m.treeItems[m.templateCursor].IsDir {
		return nil
	}
	m.activeTemplate = m.treeItems[m.templateCursor].FullPath
	targetPath := m.activeTemplate

	ctx := context.Background()
	bytes, err := os.ReadFile(targetPath)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Read template error: %v", err)
		return nil
	}
	vars, err := m.m4Expander.ParseAnnotations(ctx, string(bytes))
	if err != nil {
		m.statusMsg = fmt.Sprintf("Parse annotations error: %v", err)
		return nil
	}

	if len(vars) == 0 {
		m.startInference()
		return m.runInferenceStreamCmd(nil)
	}

	var userVars map[string]string
	var formErr error

	c := exec.Command("clear")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return tea.ExecProcess(c, func(err error) tea.Msg {
		userVars, formErr = RunDynamicForm(vars)
		if formErr != nil {
			return nil
		}
		return formSubmittedMsg{vars: userVars}
	})
}

func (m *CockpitModel) layout() {
	listWidth := m.width / 3
	if listWidth < 35 {
		listWidth = 35
	}
	previewWidth := m.width - listWidth - 5
	if previewWidth < 40 {
		previewWidth = 40
	}
	mainHeight := m.height - 8
	if mainHeight < 10 {
		mainHeight = 10
	}

	m.preview.Width = previewWidth
	m.preview.Height = mainHeight
}

func (m *CockpitModel) cursorDown() {
	if m.currentTab == tabHistory {
		if m.historyCursor < len(m.filteredHistory)-1 {
			m.historyCursor++
		}
	} else {
		if m.templateCursor < len(m.treeItems)-1 {
			m.templateCursor++
		}
	}
}

func (m *CockpitModel) cursorUp() {
	if m.currentTab == tabHistory {
		if m.historyCursor > 0 {
			m.historyCursor--
		}
	} else {
		if m.templateCursor > 0 {
			m.templateCursor--
		}
	}
}

func (m *CockpitModel) applyFilter() {
	q := strings.ToLower(m.filterInput.Value())
	if m.currentTab == tabHistory {
		if q == "" {
			m.filteredHistory = m.historyEntries
		} else {
			var res []*entity.HistoryEntry
			for _, h := range m.historyEntries {
				if strings.Contains(strings.ToLower(h.ExpandedPrompt), q) ||
					strings.Contains(strings.ToLower(h.Completion), q) {
					res = append(res, h)
				}
			}
			m.filteredHistory = res
		}
		m.historyCursor = 0
	}
}

func (m *CockpitModel) syncSelected() {
	m.selectedIndices = ParseSelectedIndices(m.chainInput.Value())
}

func (m *CockpitModel) toggleHistorySelection() {
	if len(m.filteredHistory) == 0 || m.historyCursor >= len(m.filteredHistory) {
		return
	}
	currEntry := m.filteredHistory[m.historyCursor]
	currID := currEntry.ID

	_, isSelected := m.selectedIndices[currID]

	tokens := strings.Split(m.chainInput.Value(), ",")
	var nextTokens []string
	targetStr := strconv.FormatInt(currID, 10)

	if isSelected {
		for _, tok := range tokens {
			clean := strings.TrimSpace(tok)
			unprefixed := strings.TrimLeft(clean, "sua")
			if unprefixed != targetStr && clean != "" {
				nextTokens = append(nextTokens, clean)
			}
		}
	} else {
		for _, tok := range tokens {
			clean := strings.TrimSpace(tok)
			if clean == "$" || clean == "s$" || clean == "u$" || clean == "a$" {
				nextTokens = append(nextTokens, targetStr)
			}
			if clean != "" {
				nextTokens = append(nextTokens, clean)
			}
		}
		if !strings.Contains(m.chainInput.Value(), "$") {
			nextTokens = append(nextTokens, targetStr)
		}
	}

	if len(nextTokens) == 0 {
		m.chainInput.SetValue("$")
	} else {
		m.chainInput.SetValue(strings.Join(nextTokens, ","))
	}
	m.syncSelected()
}

func (m *CockpitModel) updatePreview() {
	if m.currentTab == tabHistory {
		if len(m.filteredHistory) == 0 || m.historyCursor >= len(m.filteredHistory) {
			m.preview.SetContent("No history.")
			return
		}
		h := m.filteredHistory[m.historyCursor]
		content := fmt.Sprintf("[History #%d] %s | Tokens: %d\nTemplate: %s\n\n=== [PROMPT] ===\n%s\n\n=== [COMPLETION] ===\n%s",
			h.ID, h.CreatedAt.Format("2006-01-02 15:04:05"), h.TotalTokens, h.TemplateName, h.ExpandedPrompt, h.Completion)
		m.preview.SetContent(content)
		m.preview.GotoTop()
	} else {
		if len(m.treeItems) == 0 || m.templateCursor >= len(m.treeItems) {
			m.preview.SetContent("No template.")
			return
		}
		item := m.treeItems[m.templateCursor]
		if item.IsDir {
			m.preview.SetContent(fmt.Sprintf("Directory: %s", item.RelPath))
			return
		}
		bytes, err := os.ReadFile(item.FullPath)
		if err != nil {
			m.preview.SetContent(fmt.Sprintf("Failed to read: %v", err))
			return
		}
		m.preview.SetContent(fmt.Sprintf("[Template] %s\n\n%s", item.RelPath, string(bytes)))
		m.preview.GotoTop()
	}
}

func (m *CockpitModel) runInferenceStreamCmd(vars map[string]string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		tmplContent := ""
		if m.activeTemplate != "" {
			if b, err := os.ReadFile(m.activeTemplate); err == nil {
				tmplContent = string(b)
			}
		}

		prompt, err := m.m4Expander.Expand(ctx, tmplContent, vars)
		if err != nil {
			return inferenceCompleteMsg{err: err}
		}

		targets, err := ParseContextSpec(m.chainInput.Value())
		if err != nil {
			return inferenceCompleteMsg{err: err}
		}

		messages, err := BuildChronologicalMessages(targets, prompt, m.historyEntries)
		if err != nil {
			return inferenceCompleteMsg{err: err}
		}

		onToken := func(chunk string) {
			if activeProgram != nil {
				activeProgram.Send(streamTokenMsg(chunk))
			}
		}

		out, err := m.queryUC.ExecuteStream(ctx, usecase.QueryLLMInput{
			TemplateName:    m.activeTemplate,
			TemplateContent: tmplContent,
			CustomMessages:  messages,
			Model:           "",
			Temperature:     0.7,
			MaxTokens:       4096,
		}, onToken)

		return inferenceCompleteMsg{output: out, err: err}
	}
}

func (m CockpitModel) View() string {
	if m.quitting {
		return ""
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	tabActive := lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("238")).Foreground(lipgloss.Color("86")).Padding(0, 1)
	tabInactive := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1)

	normalBorder := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238"))
	activeBorder := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("86"))

	leftBorderStyle := normalBorder
	rightBorderStyle := normalBorder

	if m.focus == focusList {
		leftBorderStyle = activeBorder
	} else if m.focus == focusPreview {
		rightBorderStyle = activeBorder
	}

	hTab := tabInactive.Render("1:History")
	tTab := tabInactive.Render("2:Templates")
	if m.currentTab == tabHistory {
		hTab = tabActive.Render("1:History")
	} else {
		tTab = tabActive.Render("2:Templates")
	}

	activeBadge := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(
		fmt.Sprintf("[Active: %s]", filepath.Base(m.activeTemplate)),
	)
	header := lipgloss.JoinHorizontal(lipgloss.Center, headerStyle.Render(" m4llama cockpit "), hTab, tTab, "  ", activeBadge)

	listWidth := m.width / 3
	if listWidth < 35 {
		listWidth = 35
	}
	var listContent strings.Builder

	if m.currentTab == tabHistory {
		for i, h := range m.filteredHistory {
			if i >= m.height-10 {
				break
			}
			star := " "
			roleBadge := ""

			if info, ok := m.selectedIndices[h.ID]; ok {
				star = "*"
				if info.Role != "" {
					roleBadge = fmt.Sprintf("[%s]", strings.ToUpper(info.Role[:1]))
				}
			}

			clean := strings.ReplaceAll(strings.TrimSpace(h.ExpandedPrompt), "\n", " ")
			runes := []rune(clean)
			if len(runes) > 120 {
				clean = string(runes[:120]) + "..."
			}

			line := fmt.Sprintf("%s%s[%d] %s", star, roleBadge, h.ID, clean)
			if i == m.historyCursor {
				listContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).Render("> "+line) + "\n")
			} else {
				listContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("  "+line) + "\n")
			}
		}
	} else {
		for i, it := range m.treeItems {
			if i >= m.height-10 {
				break
			}
			indent := strings.Repeat("  ", it.Depth)
			prefix := "├── "
			if it.IsDir {
				prefix += "📁 "
			}

			name := it.Name
			if it.FullPath == m.activeTemplate {
				name += " (*)"
			}

			line := indent + prefix + name
			if i == m.templateCursor {
				listContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true).Render("> "+line) + "\n")
			} else {
				listContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render("  "+line) + "\n")
			}
		}
	}

	leftPane := leftBorderStyle.Width(listWidth).Height(m.height - 8).Render(listContent.String())
	rightPane := rightBorderStyle.Width(m.width - listWidth - 4).Height(m.height - 8).Render(m.preview.View())

	mainPanes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	status := m.statusMsg
	if m.isInferring {
		status = fmt.Sprintf("%s Streaming tokens... (%d tokens emitted)", m.spinner.View(), m.streamTokens)
	}

	bottomInput := m.chainInput.View()
	if m.focus == focusFileAction {
		bottomInput = m.fileActionInput.View()
	}

	var helpText string
	switch m.focus {
	case focusDeleteConfirm:
		helpText = "Confirm Deletion: [y] Yes, Delete | [n/Esc] Cancel"
	case focusPreview:
		helpText = "[Preview Focus] [j/k] Scroll | [y] Yank to wl-copy | [d/u] Half Page | [g/G] Top/Bottom | [h/Esc] Back"
	default:
		if m.currentTab == tabHistory {
			helpText = "[l/→] Preview | [y] Yank to wl-copy | [d] Delete | [e] Edit | [Space] Toggle | [c] Chain | [Enter] Run | [w] Quit"
		} else {
			helpText = "[l/→] Preview | [e] Edit | [m] Move | [y] Copy | [r/Space] Form & Run | [Enter] Select | [w] Quit"
		}
	}
	if m.isInferring {
		helpText = "Generating in progress... (Ctrl+C to abort)"
	}

	statusLine := lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render(
		fmt.Sprintf("%s  |  %s", helpText, status),
	)

	baseView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		m.filterInput.View(),
		mainPanes,
		bottomInput,
		statusLine,
	)

	if m.focus == focusDeleteConfirm {
		modalBox := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("196")).
			Padding(1, 3).
			Align(lipgloss.Center, lipgloss.Center).
			Render(fmt.Sprintf("⚠️  Delete History #%d?\n\nThis operation cannot be undone.\n\n[y] Yes, Delete   [n] Cancel", m.targetDeleteID))

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			modalBox,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return baseView
}
