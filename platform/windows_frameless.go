//go:build windows

package platform

import (
	"runtime"
	"unsafe"
)

var (
	procGetWindowRect     = modUser32.NewProc("GetWindowRect")
	procSetWindowPos      = modUser32.NewProc("SetWindowPos")
	procGetWindowLongPtrW = modUser32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW = modUser32.NewProc("SetWindowLongPtrW")
	procIsZoomed          = modUser32.NewProc("IsZoomed")
	procIsIconic          = modUser32.NewProc("IsIconic")
	procScreenToClient    = modUser32.NewProc("ScreenToClient")
	procMonitorFromWindow = modUser32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW   = modUser32.NewProc("GetMonitorInfoW")
	procDwmExtendFrame    = modDwmapi.NewProc("DwmExtendFrameIntoClientArea")
)

const (
	wsCaption     = 0x00C00000
	wsThickFrame  = 0x00040000
	wsMinimizeBox = 0x00020000
	wsMaximizeBox = 0x00010000
	wsSysMenu     = 0x00080000
	wsPopup       = 0x80000000

	// Стиль для Frameless с поддержкой Aero Snap, минимизации и системной тени
	wsFrameless = wsPopup | wsThickFrame | wsCaption | wsSysMenu | wsMinimizeBox | wsMaximizeBox

	swMinimize = 6
	swMaximize = 3
	swRestore  = 9

	wmNCCalcSize = 0x0083
	wmNCHitTest  = 0x0084

	htCaption     = 2
	htLeft        = 10
	htRight       = 11
	htTop         = 12
	htTopLeft     = 13
	htTopRight    = 14
	htBottom      = 15
	htBottomLeft  = 16
	htBottomRight = 17

	gwlStyle = ^uintptr(15) // -16 (GWL_STYLE)

	swpNoSize       = 0x0001
	swpNoMove       = 0x0002
	swpNoZOrder     = 0x0004
	swpFrameChanged = 0x0020
	hwndTop         = 0

	monitorDefaultToNearest = 2
)

type ncCalcSizeParams struct {
	rgrc  [3]rect
	lppos uintptr
}

type monitorInfo struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

type margins struct {
	cxLeftWidth    int32
	cxRightWidth   int32
	cyTopHeight    int32
	cyBottomHeight int32
}

func hitTestResultToWin32(r HitTestResult) uintptr {
	switch r {
	case HitCaption:
		return htCaption
	case HitLeft:
		return htLeft
	case HitRight:
		return htRight
	case HitTop:
		return htTop
	case HitBottom:
		return htBottom
	case HitTopLeft:
		return htTopLeft
	case HitTopRight:
		return htTopRight
	case HitBottomLeft:
		return htBottomLeft
	case HitBottomRight:
		return htBottomRight
	default:
		return htClient
	}
}

func (w *win32Window) enableFramelessShadow() {
	if w.hwnd == 0 {
		return
	}
	// WM_NCCALCSIZE при создании окна приходит до регистрации в windowMap и обрабатывается системой
	// по умолчанию (со стандартным заголовком); принудительный пересчёт рамки применяет безрамочный режим
	procSetWindowPos.Call(uintptr(w.hwnd), 0, 0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoZOrder|swpFrameChanged)
	if err := procDwmExtendFrame.Find(); err != nil {
		return
	}
	// Расширение рамки на 1 пиксель включает тень DWM вокруг безрамочного окна
	m := margins{cyTopHeight: 1}
	procDwmExtendFrame.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&m)))
	runtime.KeepAlive(&m)
}

func (w *win32Window) handleNCCalcSize(wParam, lParam uintptr) uintptr {
	if wParam == 0 {
		return 0
	}
	if w.IsMaximized() {
		params := (*ncCalcSizeParams)(ptrFromWin32(lParam))
		hMon, _, _ := procMonitorFromWindow.Call(uintptr(w.hwnd), monitorDefaultToNearest)
		if hMon != 0 {
			var mi monitorInfo
			mi.cbSize = uint32(unsafe.Sizeof(mi))
			r, _, _ := procGetMonitorInfoW.Call(hMon, uintptr(unsafe.Pointer(&mi)))
			runtime.KeepAlive(&mi)
			if r != 0 {
				params.rgrc[0] = mi.rcWork
			}
		}
	}
	return 0
}

func (w *win32Window) handleNCHitTest(lParam uintptr) uintptr {
	pt := point{
		x: int32(signExtend16(lParam)),
		y: int32(signExtend16(lParam >> 16)),
	}
	procScreenToClient.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&pt)))
	runtime.KeepAlive(&pt)

	borderPx := int(6*w.scale + 0.5)
	if borderPx < 4 {
		borderPx = 4
	}
	titleH := w.cfg.TitleBarHeight
	if titleH == 0 && w.cfg.HitTest == nil {
		titleH = 36
	}
	titlePx := int(float32(titleH)*w.scale + 0.5)

	res := ComputeFramelessHitTest(
		int(pt.x),
		int(pt.y),
		w.width,
		w.height,
		borderPx,
		titlePx,
		w.IsMaximized(),
		w.cfg.HitTest,
	)
	return hitTestResultToWin32(res)
}

func (w *win32Window) Minimize() {
	if w.hwnd != 0 {
		procShowWindow.Call(uintptr(w.hwnd), swMinimize)
	}
}

func (w *win32Window) Maximize() {
	if w.hwnd != 0 {
		procShowWindow.Call(uintptr(w.hwnd), swMaximize)
	}
}

func (w *win32Window) Restore() {
	if w.hwnd != 0 {
		procShowWindow.Call(uintptr(w.hwnd), swRestore)
	}
}

func (w *win32Window) IsMaximized() bool {
	if w.hwnd == 0 {
		return false
	}
	r, _, _ := procIsZoomed.Call(uintptr(w.hwnd))
	return r != 0
}

func (w *win32Window) IsMinimized() bool {
	if w.hwnd == 0 {
		return false
	}
	r, _, _ := procIsIconic.Call(uintptr(w.hwnd))
	return r != 0
}

func (w *win32Window) IsFullscreen() bool {
	return w.fullscreen
}

const (
	wsExTopMost   = 0x00000008
	swpNoActivate = 0x0010
	hwndTopMost   = ^uintptr(0) // HWND_TOPMOST (-1)
	hwndNoTopMost = ^uintptr(1) // HWND_NOTOPMOST (-2)
)

func (w *win32Window) SetAlwaysOnTop(onTop bool) {
	w.alwaysOnTop = onTop
	if w.hwnd == 0 {
		return
	}
	insertAfter := hwndNoTopMost
	if onTop {
		insertAfter = hwndTopMost
	}
	procSetWindowPos.Call(
		uintptr(w.hwnd),
		insertAfter,
		0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoActivate,
	)
}

func (w *win32Window) IsAlwaysOnTop() bool {
	return w.alwaysOnTop
}

func (w *win32Window) SetFullscreen(fullscreen bool) {
	if w.hwnd == 0 || w.fullscreen == fullscreen {
		return
	}
	if fullscreen {
		style, _, _ := procGetWindowLongPtrW.Call(uintptr(w.hwnd), gwlStyle)
		w.savedStyle = style
		procGetWindowRect.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&w.savedRect)))
		runtime.KeepAlive(&w.savedRect)

		hMon, _, _ := procMonitorFromWindow.Call(uintptr(w.hwnd), monitorDefaultToNearest)
		var mi monitorInfo
		mi.cbSize = uint32(unsafe.Sizeof(mi))
		r, _, _ := procGetMonitorInfoW.Call(hMon, uintptr(unsafe.Pointer(&mi)))
		runtime.KeepAlive(&mi)
		if r == 0 {
			return
		}

		newStyle := style &^ uintptr(wsCaption|wsThickFrame)
		procSetWindowLongPtrW.Call(uintptr(w.hwnd), gwlStyle, newStyle)
		procSetWindowPos.Call(
			uintptr(w.hwnd),
			hwndTop,
			uintptr(mi.rcMonitor.left),
			uintptr(mi.rcMonitor.top),
			uintptr(mi.rcMonitor.right-mi.rcMonitor.left),
			uintptr(mi.rcMonitor.bottom-mi.rcMonitor.top),
			swpFrameChanged,
		)
		w.fullscreen = true
		return
	}

	procSetWindowLongPtrW.Call(uintptr(w.hwnd), gwlStyle, w.savedStyle)
	procSetWindowPos.Call(
		uintptr(w.hwnd),
		hwndTop,
		uintptr(w.savedRect.left),
		uintptr(w.savedRect.top),
		uintptr(w.savedRect.right-w.savedRect.left),
		uintptr(w.savedRect.bottom-w.savedRect.top),
		swpFrameChanged,
	)
	w.fullscreen = false
}

func (w *win32Window) Position() (int, int) {
	if w.hwnd == 0 {
		return 0, 0
	}
	var rc rect
	procGetWindowRect.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&rc)))
	runtime.KeepAlive(&rc)
	return int(rc.left), int(rc.top)
}

func (w *win32Window) SetPosition(x, y int) {
	if w.hwnd == 0 {
		return
	}
	procSetWindowPos.Call(
		uintptr(w.hwnd),
		0,
		uintptr(x),
		uintptr(y),
		0,
		0,
		swpNoSize|swpNoZOrder,
	)
}

func (w *win32Window) SetSize(width, height int) {
	if w.hwnd == 0 || width <= 0 || height <= 0 {
		return
	}
	winRect := rect{
		right:  int32(float32(width) * w.scale),
		bottom: int32(float32(height) * w.scale),
	}
	if !w.cfg.Frameless {
		procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&winRect)), wsOverlappedWindow, 0, 0)
		runtime.KeepAlive(&winRect)
	}
	procSetWindowPos.Call(
		uintptr(w.hwnd),
		0,
		0,
		0,
		uintptr(winRect.right-winRect.left),
		uintptr(winRect.bottom-winRect.top),
		swpNoMove|swpNoZOrder,
	)
}
