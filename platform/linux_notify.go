//go:build linux

package platform

import (
	"errors"
	"os/exec"
)

// sendLinuxNotification отправляет системное уведомление через notify-send.
func sendLinuxNotification(n TrayNotification) error {
	if _, err := exec.LookPath("notify-send"); err != nil {
		return errors.New("утилита notify-send не найдена в PATH")
	}

	args := make([]string, 0, 8)
	switch n.Icon {
	case NotifyInfo:
		args = append(args, "--icon=dialog-information", "--urgency=normal")
	case NotifyWarning:
		args = append(args, "--icon=dialog-warning", "--urgency=normal")
	case NotifyError:
		args = append(args, "--icon=dialog-error", "--urgency=critical")
	}

	title := n.Title
	if title == "" {
		title = "Notification"
	}
	args = append(args, "--", title)
	if n.Message != "" {
		args = append(args, n.Message)
	}

	return exec.Command("notify-send", args...).Run()
}

