//go:build linux

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsSystemDarkTheme определяет, включена ли тёмная тема в текущем окружении Linux.
func IsSystemDarkTheme() bool {
	// 1. Стандарт XDG / GNOME (color-scheme: 'prefer-dark')
	if out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output(); err == nil {
		val := strings.ToLower(strings.TrimSpace(string(out)))
		if strings.Contains(val, "prefer-dark") {
			return true
		}
		if strings.Contains(val, "prefer-light") {
			return false
		}
	}

	// 2. Название GTK-темы через gsettings
	if out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").Output(); err == nil {
		val := strings.ToLower(strings.TrimSpace(string(out)))
		if strings.Contains(val, "dark") {
			return true
		}
	}

	// 3. Переменная окружения GTK_THEME
	if env := strings.ToLower(os.Getenv("GTK_THEME")); env != "" {
		if strings.Contains(env, "dark") {
			return true
		}
	}

	// 4. Конфигурация KDE (~/.config/kdeglobals)
	if home, err := os.UserHomeDir(); err == nil {
		if data, readErr := os.ReadFile(filepath.Join(home, ".config", "kdeglobals")); readErr == nil {
			for _, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "ColorScheme=") || strings.HasPrefix(trimmed, "LookAndFeelPackage=") {
					if strings.Contains(strings.ToLower(trimmed), "dark") {
						return true
					}
				}
			}
		}
	}

	return false
}

