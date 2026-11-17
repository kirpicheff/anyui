//go:build windows

package platform

import (
	"runtime"
	"unsafe"
)

var (
	procGetWindowPlacement = modUser32.NewProc("GetWindowPlacement")
	procSetWindowPlacement = modUser32.NewProc("SetWindowPlacement")
	procMonitorFromRect    = modUser32.NewProc("MonitorFromRect")
)

const (
	monitorDefaultToNull = 0
	swShowNormal         = 1
	swShowMaximized      = 3
)

type windowPlacement struct {
	length           uint32
	flags            uint32
	showCmd          uint32
	ptMinPosition    point
	ptMaxPosition    point
	rcNormalPosition rect
}

// State возвращает текущую сохранённую геометрию окна (позиция, размер и флаг Maximized).
func (w *win32Window) State() WindowState {
	if w.hwnd == 0 {
		return WindowState{}
	}
	var wp windowPlacement
	wp.length = uint32(unsafe.Sizeof(wp))
	r, _, _ := procGetWindowPlacement.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&wp)))
	runtime.KeepAlive(&wp)
	if r == 0 {
		return WindowState{}
	}
	rc := wp.rcNormalPosition
	return WindowState{
		X:         int(rc.left),
		Y:         int(rc.top),
		Width:     int(rc.right - rc.left),
		Height:    int(rc.bottom - rc.top),
		Maximized: wp.showCmd == swShowMaximized || w.IsMaximized(),
	}
}

// RestoreState восстанавливает позицию, размер и состояние разворачивания окна.
// Если сохранённая область не попадает ни на один подключённый монитор, окно переносится на ближайший экран.
func (w *win32Window) RestoreState(state WindowState) {
	if w.hwnd == 0 || !isValidWindowState(state) {
		return
	}

	rc := rect{
		left:   int32(state.X),
		top:    int32(state.Y),
		right:  int32(state.X + state.Width),
		bottom: int32(state.Y + state.Height),
	}

	// Проверяем, что окно хотя бы частично видимо на одном из мониторов
	hMon, _, _ := procMonitorFromRect.Call(uintptr(unsafe.Pointer(&rc)), monitorDefaultToNull)
	runtime.KeepAlive(&rc)
	if hMon == 0 {
		rc = w.clampRectToNearestMonitor(rc)
	}

	var wp windowPlacement
	wp.length = uint32(unsafe.Sizeof(wp))
	procGetWindowPlacement.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&wp)))
	wp.rcNormalPosition = rc
	if state.Maximized {
		wp.showCmd = swShowMaximized
	} else {
		wp.showCmd = swShowNormal
	}
	procSetWindowPlacement.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&wp)))
	runtime.KeepAlive(&wp)
}

func (w *win32Window) clampRectToNearestMonitor(rc rect) rect {
	hMon, _, _ := procMonitorFromWindow.Call(uintptr(w.hwnd), monitorDefaultToNearest)
	if hMon == 0 {
		return rc
	}
	var mi monitorInfo
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	r, _, _ := procGetMonitorInfoW.Call(hMon, uintptr(unsafe.Pointer(&mi)))
	runtime.KeepAlive(&mi)
	if r == 0 {
		return rc
	}
	work := mi.rcWork
	rw := rc.right - rc.left
	rh := rc.bottom - rc.top
	workW := work.right - work.left
	workH := work.bottom - work.top
	if rw > workW {
		rw = workW
	}
	if rh > workH {
		rh = workH
	}
	nx := work.left + (workW-rw)/2
	ny := work.top + (workH-rh)/2
	return rect{
		left:   nx,
		top:    ny,
		right:  nx + rw,
		bottom: ny + rh,
	}
}

func isValidWindowState(s WindowState) bool {
	return s.Width > 0 && s.Height > 0
}

