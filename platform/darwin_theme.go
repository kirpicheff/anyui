//go:build darwin

package platform

import (
	"os/exec"
	"strings"
)

// IsSystemDarkTheme определяет, включена ли тёмная тема в macOS.
func IsSystemDarkTheme() bool {
	out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").Output()
	if err != nil {
		return false
	}
	return parseAppleInterfaceStyle(string(out))
}

// parseAppleInterfaceStyle разбирает вывод команды defaults read -g AppleInterfaceStyle.
func parseAppleInterfaceStyle(out string) bool {
	return strings.EqualFold(strings.TrimSpace(out), "Dark")
}

