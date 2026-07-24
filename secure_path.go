package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func securePathComponents(root string, path string) ([]string, error) {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("路径超出允许范围")
	}
	components := strings.Split(rel, string(os.PathSeparator))
	for _, component := range components {
		if component == "" || component == "." || component == ".." {
			return nil, fmt.Errorf("路径包含无效组件")
		}
	}
	return components, nil
}
