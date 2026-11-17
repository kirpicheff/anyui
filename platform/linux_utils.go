//go:build linux

package platform

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OpenURL открывает веб-ссылку или mailto-адрес в приложении по умолчанию (через xdg-open).
func OpenURL(rawURL string) error {
	if err := validateOpenURL(rawURL); err != nil {
		return err
	}
	cmd := exec.Command("xdg-open", strings.TrimSpace(rawURL))
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// RevealInFileManager открывает файловый менеджер и выделяет указанный файл или папку.
func RevealInFileManager(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("путь не задан")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Пробуем стандартный D-Bus интерфейс freedesktop FileManager1 (работает в Nautilus, Dolphin, Thunar, Nemo)
	fileURI := (&url.URL{Scheme: "file", Path: absPath}).String()
	if _, lookErr := exec.LookPath("dbus-send"); lookErr == nil {
		cmd := exec.Command(
			"dbus-send",
			"--session",
			"--dest=org.freedesktop.FileManager1",
			"--type=method_call",
			"/org/freedesktop/FileManager1",
			"org.freedesktop.FileManager1.ShowItems",
			"array:string:"+fileURI,
			"string:",
		)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Fallback: открываем сам каталог или родительскую директорию через xdg-open
	target := absPath
	if info, statErr := os.Stat(absPath); statErr == nil && !info.IsDir() {
		target = filepath.Dir(absPath)
	} else if statErr != nil {
		target = filepath.Dir(absPath)
	}

	cmd := exec.Command("xdg-open", target)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

