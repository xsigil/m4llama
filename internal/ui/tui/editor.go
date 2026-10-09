package tui

import (
	"fmt"
	"os"
	"os/exec"
)

// OpenInEditor は環境変数 EDITOR（未設定なら nvim）で指定ファイルを開きます
func OpenInEditor(filePath string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}

	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run editor %s: %w", editor, err)
	}
	return nil
}
