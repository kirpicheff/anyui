//go:build darwin && cgo

package platform

/*
#include <stdlib.h>
#include "darwin_bridge.h"
*/
import "C"

import "unsafe"

// sendDarwinNotification отправляет нативное уведомление через UNUserNotificationCenter,
// а при запуске вне .app bundle использует osascript как резервный вариант.
func sendDarwinNotification(n TrayNotification) error {
	cTitle := C.CString(n.Title)
	cMsg := C.CString(n.Message)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cMsg))

	silent := C.int(0)
	if n.Silent {
		silent = 1
	}
	if C.anyui_darwin_notify_un(cTitle, cMsg, silent) != 0 {
		return nil
	}
	return sendDarwinNotificationAppleScript(n)
}

