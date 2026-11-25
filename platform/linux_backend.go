//go:build linux

package platform

import (
	"os"
	"strings"
	"sync/atomic"
)

type linuxBackend uint8

const (
	linuxBackendX11 linuxBackend = iota
	linuxBackendWayland
)

var (
	// selectedLinuxBackend хранит выбранный в рантайме бэкенд (0 — не инициализирован, 1 — X11, 2 — Wayland).
	selectedLinuxBackend atomic.Uint32
)

// detectLinuxBackend определяет предпочтительный графический бэкенд по переменным окружения.
func detectLinuxBackend(getenv func(string) string) linuxBackend {
	override := strings.ToLower(strings.TrimSpace(getenv("ANYUI_BACKEND")))
	switch override {
	case "wayland":
		return linuxBackendWayland
	case "x11":
		return linuxBackendX11
	}
	if strings.TrimSpace(getenv("WAYLAND_DISPLAY")) != "" {
		return linuxBackendWayland
	}
	return linuxBackendX11
}

func isWaylandBackendSelected() bool {
	if v := selectedLinuxBackend.Load(); v != 0 {
		return v == uint32(linuxBackendWayland)+1
	}
	return detectLinuxBackend(os.Getenv) == linuxBackendWayland
}

func setSelectedLinuxBackend(b linuxBackend) {
	selectedLinuxBackend.Store(uint32(b) + 1)
}

// NewWindow создаёт новое нативное окно для Linux (Wayland или X11 в зависимости от окружения).
func NewWindow(cfg WindowConfig) (Window, error) {
	override := strings.ToLower(strings.TrimSpace(os.Getenv("ANYUI_BACKEND")))
	preferred := detectLinuxBackend(os.Getenv)

	if preferred == linuxBackendWayland {
		win, err := newWaylandWindow(cfg)
		if err == nil {
			setSelectedLinuxBackend(linuxBackendWayland)
			return win, nil
		}
		// Если Wayland не удалось инициализировать и бэкенд не зафиксирован жёстко, пробуем X11.
		if override != "wayland" && strings.TrimSpace(os.Getenv("DISPLAY")) != "" {
			if xWin, xErr := newX11Window(cfg); xErr == nil {
				setSelectedLinuxBackend(linuxBackendX11)
				return xWin, nil
			}
		}
		return nil, err
	}

	setSelectedLinuxBackend(linuxBackendX11)
	return newX11Window(cfg)
}

// RunEventLoop запускает главный цикл событий активного Linux-бэкенда.
func RunEventLoop(onStart func() error) error {
	if isWaylandBackendSelected() {
		return runWaylandEventLoop(onStart)
	}
	return runX11EventLoop(onStart)
}

// CursorScreenPos возвращает текущие экранные координаты курсора мыши (пока не поддерживается на Linux).
func CursorScreenPos() (xPx, yPx int, ok bool) {
	return 0, 0, false
}

