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

// TestDocstringSyncReorderOptions проверяет идентичность docstring функций Reorderable, OnReorder и AllowReparent
// между ui/options.go и корневым options.go, а также наличие канонического списка поддерживаемых виджетов.
func TestDocstringSyncReorderOptions(t *testing.T) {
	rootOptionsPath := filepath.Join("..", "options.go")
	if _, err := os.Stat(rootOptionsPath); err != nil {
		t.Skip("файл options.go не найден из текущего рабочего каталога:", err)
	}

	extractFuncDoc := func(path, funcName string) string {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("ошибка парсинга %s: %v", path, err)
		}
		for _, decl := range node.Decls {
			// Ищем объявления функций по имени через строковое представление нельзя — используем go/ast
			_ = decl
		}
		// Читаем группы комментариев, начинающиеся с "<funcName> "
		prefix := funcName + " "
		for _, cg := range node.Comments {
			text := strings.TrimSpace(cg.Text())
			if strings.HasPrefix(text, prefix) {
				return text
			}
		}
		t.Fatalf("не найден doc-комментарий для функции %s в %s", funcName, path)
		return ""
	}

	const canonicalWidgets = "Table, TableEx, ListView, ListViewEx, TreeView, TreeViewEx"
	for _, fn := range []string{"Reorderable", "OnReorder", "AllowReparent"} {
		docUI := extractFuncDoc("options.go", fn)
		docRoot := extractFuncDoc(rootOptionsPath, fn)
		if docUI != docRoot {
			t.Fatalf("docstring %s рассинхронизирован между ui/options.go и options.go!\n--- ui/options.go ---\n%s\n--- options.go ---\n%s", fn, docUI, docRoot)
		}
		if fn == "Reorderable" || fn == "OnReorder" {
			if !strings.Contains(docUI, canonicalWidgets) {
				t.Fatalf("docstring %s не содержит канонический список виджетов %q:\n%s", fn, canonicalWidgets, docUI)
			}
		}
	}
}
