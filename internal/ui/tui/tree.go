package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type TreeItem struct {
	FullPath string
	RelPath  string
	IsDir    bool
	Depth    int
	Name     string
}

// BuildTemplateTree scans the root template directory and returns all items hierarchically
func BuildTemplateTree(rootDir string) ([]TreeItem, error) {
	var items []TreeItem

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return items, nil
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == rootDir {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		depth := strings.Count(rel, string(os.PathSeparator))
		name := info.Name()

		if info.IsDir() {
			items = append(items, TreeItem{
				FullPath: path,
				RelPath:  rel,
				IsDir:    true,
				Depth:    depth,
				Name:     name,
			})
			return nil
		}

		if strings.HasSuffix(name, ".m4") {
			items = append(items, TreeItem{
				FullPath: path,
				RelPath:  rel,
				IsDir:    false,
				Depth:    depth,
				Name:     name,
			})
		}
		return nil
	})

	return items, err
}

// MoveFile moves or renames a template file or directory
func MoveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// CopyFile copies a template file
func CopyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// CreateTemplateFile creates a new .m4 template file with initial boilerplate
func CreateTemplateFile(targetPath string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("file already exists: %s", filepath.Base(targetPath))
	}

	initialBoilerplate := `# @var TASK [string] "Instruction or task for LLM" "Summarize the text"
# @var STYLE [select:formal,casual,bullet_points] "Output style" "formal"
#
You are a helpful and precise assistant.
Task: TASK
Style: STYLE

`

	return os.WriteFile(targetPath, []byte(initialBoilerplate), 0644)
}

// CreateTemplateDir creates a new directory inside the template workspace
func CreateTemplateDir(targetPath string) error {
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("directory already exists: %s", filepath.Base(targetPath))
	}
	return os.MkdirAll(targetPath, 0755)
}

// DeleteTemplate removes a template file or directory (recursively if directory)
func DeleteTemplate(targetPath string) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return os.RemoveAll(targetPath)
	}
	return os.Remove(targetPath)
}