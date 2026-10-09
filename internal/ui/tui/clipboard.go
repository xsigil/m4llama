package tui

import (
	"fmt"
	"os/exec"
	"strings"
)

// CopyToClipboard はテキストを wl-copy (Wayland) または xclip (X11) に送信します
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	// 1. wl-copy を優先探索
	if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd = exec.Command("wl-copy")
	} else if _, err := exec.LookPath("xclip"); err == nil {
		cmd = exec.Command("xclip", "-selection", "clipboard")
	} else {
		return fmt.Errorf("wl-copy (or xclip) not found")
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
