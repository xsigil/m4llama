package tui

import (
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

func MoveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

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
