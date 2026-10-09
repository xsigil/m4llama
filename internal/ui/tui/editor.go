package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xsigil/m4llama/internal/domain/entity"
)

func CreateEditorCmd(filePath string) *exec.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}
	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// PrepareHistoryTempFile は過去メッセージを一時ファイルに書き出し、編集用パスを返します
func PrepareHistoryTempFile(h *entity.HistoryEntry) (string, error) {
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("m4llama_history_%d_*.txt", h.ID))
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	content := fmt.Sprintf("=== [PROMPT] ===\n%s\n\n=== [COMPLETION] ===\n%s\n", h.ExpandedPrompt, h.Completion)
	if _, err := tmpFile.WriteString(content); err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

// ApplyHistoryTempFile は一時ファイルの内容をパースして HistoryEntry に反映します
func ApplyHistoryTempFile(tmpPath string, h *entity.HistoryEntry) (bool, error) {
	updatedBytes, err := os.ReadFile(tmpPath)
	if err != nil {
		return false, err
	}
	updatedStr := string(updatedBytes)

	promptMarker := "=== [PROMPT] ==="
	compMarker := "=== [COMPLETION] ==="

	pIdx := strings.Index(updatedStr, promptMarker)
	cIdx := strings.Index(updatedStr, compMarker)

	if pIdx == -1 || cIdx == -1 || pIdx >= cIdx {
		return false, fmt.Errorf("invalid format: markers removed or corrupted")
	}

	newPrompt := strings.TrimSpace(updatedStr[pIdx+len(promptMarker) : cIdx])
	newComp := strings.TrimSpace(updatedStr[cIdx+len(compMarker):])

	changed := (newPrompt != h.ExpandedPrompt || newComp != h.Completion)
	h.ExpandedPrompt = newPrompt
	h.Completion = newComp

	return changed, nil
}
