//go:build darwin && !cgo

package platform

import "errors"

var errDarwinRequiresCGO = errors.New("платформа macOS для оконного режима и трея требует сборки с CGO_ENABLED=1 (Cocoa/AppKit мост)")

// NewWindow возвращает ошибку при сборке под macOS без CGO.
func NewWindow(cfg WindowConfig) (Window, error) {
	_ = cfg
	return nil, errDarwinRequiresCGO
}

// RunEventLoop возвращает ошибку при сборке под macOS без CGO.
func RunEventLoop(onStart func() error) error {
	_ = onStart
	return errDarwinRequiresCGO
}

// NewTray возвращает ошибку при сборке под macOS без CGO.
func NewTray(cfg TrayConfig) (Tray, error) {
	_ = cfg
	return nil, errDarwinRequiresCGO
}

// CursorScreenPos возвращает текущие экранные координаты курсора мыши.
func CursorScreenPos() (xPx, yPx int, ok bool) {
	return 0, 0, false
}

