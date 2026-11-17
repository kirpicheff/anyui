//go:build darwin

package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

// escapeAppleScriptString экранирует строку для безопасной подстановки в строковый литерал AppleScript.
func escapeAppleScriptString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// buildNotificationScript формирует AppleScript для показа системного уведомления macOS.
func buildNotificationScript(n TrayNotification) string {
	msg := escapeAppleScriptString(n.Message)
	title := escapeAppleScriptString(n.Title)
	if title != "" {
		return fmt.Sprintf(`display notification "%s" with title "%s"`, msg, title)
	}
	return fmt.Sprintf(`display notification "%s"`, msg)
}

// sendDarwinNotification отправляет системное уведомление через osascript (без CGO).
func sendDarwinNotification(n TrayNotification) error {
	script := buildNotificationScript(n)
	return exec.Command("osascript", "-e", script).Run()
}

