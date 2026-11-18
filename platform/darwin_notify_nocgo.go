//go:build darwin && !cgo

package platform

// sendDarwinNotification отправляет системное уведомление через osascript (без CGO).
func sendDarwinNotification(n TrayNotification) error {
	return sendDarwinNotificationAppleScript(n)
}

