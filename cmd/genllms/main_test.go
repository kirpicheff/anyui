package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAndLintLLMs(t *testing.T) {
	rootDir := filepath.Join("..", "..")
	shortBytes, fullBytes, err := generateAll(rootDir, "v0.8.1")
	if err != nil {
		t.Fatalf("generateAll завершилась с ошибкой: %v", err)
	}
	shortText := string(shortBytes)
	fullText := string(fullBytes)

	if len(shortText) >= len(fullText) {
		t.Fatalf("llms.txt (%d байт) должен быть компактнее llms-full.txt (%d байт)", len(shortText), len(fullText))
	}
	if !strings.Contains(fullText, "KeyBackspace   Key = 0x08") {
		t.Error("llms-full.txt должен содержать явные значения констант Key*")
	}
	if !strings.Contains(fullText, "MouseLeft      MouseButton = 1") {
		t.Error("llms-full.txt должен содержать явные значения констант Mouse*")
	}
}

func TestLLMsFilesUpToDate(t *testing.T) {
	rootDir := filepath.Join("..", "..")
	// Извлекаем версию из первой строки существующего llms.txt, чтобы тест не зависел от git describe.
	raw, err := os.ReadFile(filepath.Join(rootDir, "llms.txt"))
	if err != nil {
		t.Fatalf("не удалось прочитать llms.txt: %v", err)
	}
	firstLine := strings.SplitN(string(raw), "\n", 2)[0]
	ver := "dev"
	if l := strings.Index(firstLine, "("); l >= 0 {
		if r := strings.Index(firstLine[l:], ")"); r > 1 {
			ver = firstLine[l+1 : l+r]
		}
	}

	shortBytes, fullBytes, err := generateAll(rootDir, ver)
	if err != nil {
		t.Fatalf("generateAll завершилась с ошибкой: %v", err)
	}
	if err := checkFile(filepath.Join(rootDir, "llms.txt"), shortBytes); err != nil {
		t.Fatal(err)
	}
	if err := checkFile(filepath.Join(rootDir, "llms-full.txt"), fullBytes); err != nil {
		t.Fatal(err)
	}
}
