//go:build darwin

package platform

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OpenURL открывает веб-ссылку или mailto-адрес в приложении по умолчанию (через open).
func OpenURL(rawURL string) error {
	if err := validateOpenURL(rawURL); err != nil {
		return err
	}
	cmd := exec.Command("open", strings.TrimSpace(rawURL))
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// RevealInFileManager открывает Finder и выделяет указанный файл или папку.
func RevealInFileManager(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("путь не задан")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// open -R выделяет элемент в Finder, если путь существует
	if _, statErr := os.Stat(absPath); statErr == nil {
		cmd := exec.Command("open", "-R", absPath)
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() { _ = cmd.Wait() }()
		return nil
	}

	// Если файл не существует, открываем родительский каталог
	parent := filepath.Dir(absPath)
	cmd := exec.Command("open", parent)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

