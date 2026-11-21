package ui

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDocstringSyncDragEvent проверяет идентичность docstring DragEvent в ui/options.go и ui_controls_extra.go.
// Разное содержимое запрещено осознанно: root-алиас anyui.DragEvent должен отдавать тот же контракт через go doc.
func TestDocstringSyncDragEvent(t *testing.T) {
	rootPath := filepath.Join("..", "ui_controls_extra.go")
	if _, err := os.Stat(rootPath); err != nil {
		t.Skip("файл ui_controls_extra.go не найден из текущего рабочего каталога:", err)
	}

	extractDoc := func(path, typeName string) string {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("ошибка парсинга %s: %v", path, err)
		}
		for _, commentGroup := range node.Comments {
			text := commentGroup.Text()
			if strings.Contains(text, "DragEvent описывает событие") {
				return text
			}
		}
		t.Fatalf("не найден doc-комментарий для %s в %s", typeName, path)
		return ""
	}

	normalize := func(s string) string {
		lines := strings.Split(s, "\n")
		var clean []string
		for _, l := range lines {
			l = strings.TrimPrefix(l, "//")
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "ВАЖНО:") {
				clean = append(clean, l)
			}
		}
		return strings.Join(clean, "\n")
	}

	docUI := normalize(extractDoc("options.go", "DragEvent"))
	docRoot := normalize(extractDoc(rootPath, "DragEvent"))

	if docUI != docRoot {
		t.Fatalf("docstring DragEvent рассинхронизирован между ui/options.go и ui_controls_extra.go!\n--- ui/options.go ---\n%s\n--- ui_controls_extra.go ---\n%s", docUI, docRoot)
	}
}
