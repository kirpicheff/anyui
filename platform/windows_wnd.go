//go:build windows

package platform

import (
	"errors"
	"runtime"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modUser32   = windows.NewLazySystemDLL("user32.dll")
	modKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	modDwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	modShell32  = windows.NewLazySystemDLL("shell32.dll")

	procRegisterClassExW          = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW           = modUser32.NewProc("CreateWindowExW")
	procDestroyWindow             = modUser32.NewProc("DestroyWindow")
	procShowWindow                = modUser32.NewProc("ShowWindow")
	procUpdateWindow              = modUser32.NewProc("UpdateWindow")
	procDefWindowProcW            = modUser32.NewProc("DefWindowProcW")
	procGetMessageW               = modUser32.NewProc("GetMessageW")
	procTranslateMessage          = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW          = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage           = modUser32.NewProc("PostQuitMessage")
	procPostMessageW              = modUser32.NewProc("PostMessageW")
	procInvalidateRect            = modUser32.NewProc("InvalidateRect")
	procBeginPaint                = modUser32.NewProc("BeginPaint")
	procEndPaint                  = modUser32.NewProc("EndPaint")
	procGetClientRect             = modUser32.NewProc("GetClientRect")
	procAdjustWindowRectEx        = modUser32.NewProc("AdjustWindowRectEx")
	procSetWindowTextW            = modUser32.NewProc("SetWindowTextW")
	procLoadCursorW               = modUser32.NewProc("LoadCursorW")
	procSetCursor                 = modUser32.NewProc("SetCursor")
	procGetKeyState               = modUser32.NewProc("GetKeyState")
	procSetProcessDPIAware        = modUser32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwarenessCtx = modUser32.NewProc("SetProcessDpiAwarenessContext")
	procGetDpiForWindow           = modUser32.NewProc("GetDpiForWindow")
	procGetDC                     = modUser32.NewProc("GetDC")
	procReleaseDC                 = modUser32.NewProc("ReleaseDC")
	procSetCapture                = modUser32.NewProc("SetCapture")
	procReleaseCapture            = modUser32.NewProc("ReleaseCapture")
	procGetDoubleClickTime        = modUser32.NewProc("GetDoubleClickTime")
	procOpenClipboard             = modUser32.NewProc("OpenClipboard")
	procCloseClipboard            = modUser32.NewProc("CloseClipboard")
	procEmptyClipboard            = modUser32.NewProc("EmptyClipboard")
	procGetClipboardData          = modUser32.NewProc("GetClipboardData")
	procSetClipboardData          = modUser32.NewProc("SetClipboardData")

	procGetModuleHandleW = modKernel32.NewProc("GetModuleHandleW")
	procGlobalAlloc      = modKernel32.NewProc("GlobalAlloc")
	procGlobalLock       = modKernel32.NewProc("GlobalLock")
	procGlobalUnlock     = modKernel32.NewProc("GlobalUnlock")

	procDwmSetWindowAttribute = modDwmapi.NewProc("DwmSetWindowAttribute")

	procDragAcceptFiles = modShell32.NewProc("DragAcceptFiles")
	procDragQueryFileW  = modShell32.NewProc("DragQueryFileW")
	procDragQueryPoint  = modShell32.NewProc("DragQueryPoint")
	procDragFinish      = modShell32.NewProc("DragFinish")
)

const (
	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	cwUseDefault       = 0x80000000

	csHRedraw = 0x0002
	csVRedraw = 0x0001

	swHide        = 0
	swShow        = 5
	swShowDefault = 10

	wmDestroy       = 0x0002
	wmSize          = 0x0005
	wmKillFocus     = 0x0008
	wmPaint         = 0x000F
	wmClose         = 0x0010
	wmEraseBkgnd    = 0x0014
	wmCancelMode    = 0x001F
	wmGetMinMaxInfo = 0x0024
	wmSetCursor     = 0x0020
	wmKeyDown       = 0x0100
	wmKeyUp         = 0x0101
	wmChar          = 0x0102
	wmSysKeyDown    = 0x0104
	wmSysKeyUp      = 0x0105
	wmMouseMove     = 0x0200
	wmLButtonDown   = 0x0201
	wmLButtonUp     = 0x0202
	wmRButtonDown   = 0x0204
	wmRButtonUp     = 0x0205
	wmMButtonDown   = 0x0207
	wmMButtonUp     = 0x0208
	wmMouseWheel     = 0x020A
	wmMouseHWheel    = 0x020E
	wmCaptureChanged = 0x0215
	wmDropFiles      = 0x0233
	wmDPIChanged    = 0x02E0
	wmAppWakeUp     = 0x8001

	htClient = 1

	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12

	idcArrow  = 32512
	idcIBeam  = 32513
	idcSizeWE = 32644
	idcSizeNS = 32645
	idcHand   = 32649

	cfUnicodeText = 13
	gmemMoveable  = 0x0002

	dwmwaUseImmersiveDarkMode    = 20
	dwmwaUseImmersiveDarkModeOld = 19
)

type wndClassExW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     windows.Handle
	hIcon         windows.Handle
	hCursor       windows.Handle
	hbrBackground windows.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       windows.Handle
}

type point struct {
	x, y int32
}

type rect struct {
	left, top, right, bottom int32
}

type msgStruct struct {
	hwnd    windows.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type paintStruct struct {
	hdc         windows.Handle
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type minMaxInfo struct {
	ptReserved     point
	ptMaxSize      point
	ptMaxPosition  point
	ptMinTrackSize point
	ptMaxTrackSize point
}

var (
	windowMapMu sync.RWMutex
	windowMap   = make(map[windows.Handle]*win32Window)

	wndProcCallback = syscall.NewCallback(win32WndProc)
	dpiInitOnce     sync.Once
)

type win32Window struct {
	cfg     WindowConfig
	hwnd    windows.Handle
	handler func(Event)

	width  int
	height int
	scale  float32
	cursor CursorType

	mu         sync.Mutex
	pixels     []byte
	bufW       int
	bufH       int
	closing          bool
	onClose          func() bool
	fullscreen       bool
	savedStyle       uintptr
	savedRect        rect
	releasingCapture bool
}

// NewWindow создаёт нативное окно для текущей операционной системы.
func NewWindow(cfg WindowConfig) (Window, error) {
	if cfg.Width <= 0 {
		cfg.Width = 960
	}
	if cfg.Height <= 0 {
		cfg.Height = 640
	}
	return &win32Window{
		cfg:   cfg,
		scale: 1.0,
	}, nil
}

func initDPIAwareness() {
	// Пытаемся включить Per-Monitor V2 DPI Awareness; на старых ОС откатываемся на SetProcessDPIAware
	if err := procSetProcessDpiAwarenessCtx.Find(); err == nil {
		const dpiContextPerMonitorAwareV2 = ^uintptr(3) // -4
		r, _, _ := procSetProcessDpiAwarenessCtx.Call(dpiContextPerMonitorAwareV2)
		if r != 0 {
			return
		}
	}
	if err := procSetProcessDPIAware.Find(); err == nil {
		procSetProcessDPIAware.Call()
	}
}

// RunEventLoop запускает единый цикл сообщений Win32 в текущем потоке ОС.
// Функция onStart вызывается уже внутри заблокированного потока перед началом цикла.
func RunEventLoop(onStart func() error) error {
	// Цикл сообщений Win32 жёстко привязан к потоку ОС, создавшему окна
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	dpiInitOnce.Do(initDPIAwareness)

	if onStart != nil {
		if err := onStart(); err != nil {
			return err
		}
	}

	var msg msgStruct
	for {
		ret, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		runtime.KeepAlive(&msg)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	return nil
}

func (w *win32Window) Run(handler func(Event)) error {
	return RunEventLoop(func() error {
		return w.Open(handler)
	})
}

func (w *win32Window) Open(handler func(Event)) error {
	if handler == nil {
		return errors.New("обработчик событий не задан")
	}

	dpiInitOnce.Do(initDPIAwareness)
	w.handler = handler

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className, err := windows.UTF16PtrFromString("AnyUIMainWindowClass")
	if err != nil {
		return err
	}
	titlePtr, err := windows.UTF16PtrFromString(w.cfg.Title)
	if err != nil {
		return err
	}

	hCursor, _, _ := procLoadCursorW.Call(0, idcArrow)
	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		style:         csHRedraw | csVRedraw,
		lpfnWndProc:   wndProcCallback,
		hInstance:     windows.Handle(hInstance),
		hCursor:       windows.Handle(hCursor),
		lpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(&wc)
	runtime.KeepAlive(className)

	// Вычисляем начальный масштаб по системному DPI до создания окна
	hdcScreen, _, _ := procGetDC.Call(0)
	sysDPI := getSystemDPI(windows.Handle(hdcScreen))
	if hdcScreen != 0 {
		procReleaseDC.Call(0, hdcScreen)
	}
	w.scale = float32(sysDPI) / 96.0

	style := uintptr(wsOverlappedWindow)
	if w.cfg.Frameless {
		style = wsFrameless
	}

	winRect := rect{
		left:   0,
		top:    0,
		right:  int32(float32(w.cfg.Width) * w.scale),
		bottom: int32(float32(w.cfg.Height) * w.scale),
	}
	if !w.cfg.Frameless {
		procAdjustWindowRectEx.Call(
			uintptr(unsafe.Pointer(&winRect)),
			style,
			0,
			0,
		)
		runtime.KeepAlive(&winRect)
	}

	var parentHWND uintptr
	if w.cfg.Parent != nil {
		parentHWND = w.cfg.Parent.NativeHandle()
	}

	hwndRaw, _, createErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(titlePtr)),
		style,
		cwUseDefault,
		cwUseDefault,
		uintptr(winRect.right-winRect.left),
		uintptr(winRect.bottom-winRect.top),
		parentHWND,
		0,
		hInstance,
		0,
	)
	runtime.KeepAlive(className)
	runtime.KeepAlive(titlePtr)

	if hwndRaw == 0 {
		return createErr
	}

	w.hwnd = windows.Handle(hwndRaw)
	windowMapMu.Lock()
	windowMap[w.hwnd] = w
	windowMapMu.Unlock()

	// Уточняем DPI конкретного монитора после создания окна
	w.updateScaleFromWindow()
	w.updateClientSize()

	if w.cfg.DarkTitle {
		w.enableDarkTitleBar()
	}
	if w.cfg.Frameless {
		w.enableFramelessShadow()
	}
	if w.cfg.AcceptDropFiles {
		procDragAcceptFiles.Call(uintptr(w.hwnd), 1)
	}

	procShowWindow.Call(uintptr(w.hwnd), swShowDefault)
	procUpdateWindow.Call(uintptr(w.hwnd))

	// Уведомляем приложение о стартовом размере и масштабе
	w.emit(Event{Type: EventDPIChange, Scale: w.scale})
	w.emit(Event{Type: EventResize, Width: w.width, Height: w.height})
	w.emit(Event{Type: EventPaint})

	return nil
}

func (w *win32Window) enableDarkTitleBar() {
	if err := procDwmSetWindowAttribute.Find(); err != nil {
		return
	}
	var dark int32 = 1
	r, _, _ := procDwmSetWindowAttribute.Call(
		uintptr(w.hwnd),
		dwmwaUseImmersiveDarkMode,
		uintptr(unsafe.Pointer(&dark)),
		unsafe.Sizeof(dark),
	)
	if r != 0 {
		procDwmSetWindowAttribute.Call(
			uintptr(w.hwnd),
			dwmwaUseImmersiveDarkModeOld,
			uintptr(unsafe.Pointer(&dark)),
			unsafe.Sizeof(dark),
		)
	}
	runtime.KeepAlive(&dark)
}

func (w *win32Window) updateScaleFromWindow() {
	if err := procGetDpiForWindow.Find(); err == nil && w.hwnd != 0 {
		dpi, _, _ := procGetDpiForWindow.Call(uintptr(w.hwnd))
		if dpi > 0 {
			w.scale = float32(dpi) / 96.0
		}
	}
}

func (w *win32Window) updateClientSize() {
	var rc rect
	procGetClientRect.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(&rc)))
	runtime.KeepAlive(&rc)
	w.width = int(rc.right - rc.left)
	w.height = int(rc.bottom - rc.top)
}

func (w *win32Window) Present(pixels []byte, width, height int) {
	if width <= 0 || height <= 0 || len(pixels) < width*height*4 {
		return
	}
	w.mu.Lock()
	need := width * height * 4
	if cap(w.pixels) < need {
		w.pixels = make([]byte, need)
	} else {
		w.pixels = w.pixels[:need]
	}
	copy(w.pixels, pixels[:need])
	w.bufW = width
	w.bufH = height
	w.mu.Unlock()

	if w.hwnd != 0 {
		procInvalidateRect.Call(uintptr(w.hwnd), 0, 0)
		procUpdateWindow.Call(uintptr(w.hwnd))
	}
}

func (w *win32Window) Invalidate() {
	if w.hwnd != 0 {
		procInvalidateRect.Call(uintptr(w.hwnd), 0, 0)
		procUpdateWindow.Call(uintptr(w.hwnd))
	}
}

func (w *win32Window) WakeUp() {
	if w.hwnd != 0 {
		procPostMessageW.Call(uintptr(w.hwnd), wmAppWakeUp, 0, 0)
	}
}

func (w *win32Window) Show() {
	if w.hwnd != 0 {
		procShowWindow.Call(uintptr(w.hwnd), swShow)
	}
}

func (w *win32Window) Hide() {
	if w.hwnd != 0 {
		procShowWindow.Call(uintptr(w.hwnd), swHide)
	}
}

func (w *win32Window) Close() {
	if w.hwnd != 0 {
		procPostMessageW.Call(uintptr(w.hwnd), wmClose, 0, 0)
	}
}

func (w *win32Window) SetTitle(title string) {
	if w.hwnd == 0 {
		return
	}
	ptr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	procSetWindowTextW.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(ptr)))
	runtime.KeepAlive(ptr)
}

func (w *win32Window) SetCursor(cursor CursorType) {
	w.cursor = cursor
	w.applyCursor()
}

// SetCapture захватывает ввод мыши окном.
func (w *win32Window) SetCapture() {
	if w.hwnd != 0 {
		procSetCapture.Call(uintptr(w.hwnd))
	}
}

// ReleaseCapture освобождает захват ввода мыши.
func (w *win32Window) ReleaseCapture() {
	w.releasingCapture = true
	procReleaseCapture.Call()
	w.releasingCapture = false
}

// DoubleClickTimeMs возвращает системное время двойного клика ОС в миллисекундах.
func (w *win32Window) DoubleClickTimeMs() int {
	ms, _, _ := procGetDoubleClickTime.Call()
	if ms == 0 {
		return 500
	}
	return int(ms)
}

func (w *win32Window) applyCursor() {
	var id uintptr = idcArrow
	switch w.cursor {
	case CursorHand:
		id = idcHand
	case CursorIBeam:
		id = idcIBeam
	case CursorResizeH:
		id = idcSizeWE
	case CursorResizeV:
		id = idcSizeNS
	}
	hCursor, _, _ := procLoadCursorW.Call(0, id)
	if hCursor != 0 {
		procSetCursor.Call(hCursor)
	}
}

func (w *win32Window) Size() (int, int) {
	return w.width, w.height
}

func (w *win32Window) Scale() float32 {
	return w.scale
}

func (w *win32Window) SetOnClose(fn func() bool) {
	w.onClose = fn
}

func (w *win32Window) NativeHandle() uintptr {
	return uintptr(w.hwnd)
}

func (w *win32Window) ClipboardText() (string, error) {
	r, _, _ := procOpenClipboard.Call(uintptr(w.hwnd))
	if r == 0 {
		return "", errors.New("не удалось открыть буфер обмена")
	}
	defer procCloseClipboard.Call()

	hData, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if hData == 0 {
		return "", nil
	}
	ptr, _, _ := procGlobalLock.Call(hData)
	if ptr == 0 {
		return "", errors.New("не удалось заблокировать память буфера обмена")
	}
	defer procGlobalUnlock.Call(hData)

	return windows.UTF16PtrToString((*uint16)(unsafe.Pointer(ptr))), nil
}

func (w *win32Window) SetClipboardText(text string) error {
	encoded := utf16.Encode([]rune(text + "\x00"))
	byteLen := len(encoded) * 2

	hMem, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(byteLen))
	if hMem == 0 {
		return errors.New("ошибка выделения памяти для буфера обмена")
	}
	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return errors.New("ошибка блокировки памяти для буфера обмена")
	}
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(encoded))
	copy(dst, encoded)
	procGlobalUnlock.Call(hMem)

	r, _, _ := procOpenClipboard.Call(uintptr(w.hwnd))
	if r == 0 {
		return errors.New("не удалось открыть буфер обмена")
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	procSetClipboardData.Call(cfUnicodeText, hMem)
	return nil
}

func (w *win32Window) emit(ev Event) {
	if w.handler != nil && !w.closing {
		w.handler(ev)
	}
}

func getModifiers() Modifiers {
	shift, _, _ := procGetKeyState.Call(vkShift)
	ctrl, _, _ := procGetKeyState.Call(vkControl)
	alt, _, _ := procGetKeyState.Call(vkMenu)
	return Modifiers{
		Shift: (uint16(shift) & 0x8000) != 0,
		Ctrl:  (uint16(ctrl) & 0x8000) != 0,
		Alt:   (uint16(alt) & 0x8000) != 0,
	}
}

func signExtend16(v uintptr) int {
	return int(int16(uint16(v & 0xFFFF)))
}

func win32WndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	windowMapMu.RLock()
	w := windowMap[hwnd]
	windowMapMu.RUnlock()

	if w == nil {
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	}

	switch msg {
	case wmEraseBkgnd:
		// Возвращаем 1, чтобы Win32 не очищал фон белым цветом перед WM_PAINT
		return 1

	case wmGetMinMaxInfo:
		if w.cfg.MinWidth > 0 || w.cfg.MinHeight > 0 {
			mmi := (*minMaxInfo)(unsafe.Pointer(lParam))
			minRect := rect{
				right:  int32(float32(w.cfg.MinWidth) * w.scale),
				bottom: int32(float32(w.cfg.MinHeight) * w.scale),
			}
			if !w.cfg.Frameless {
				procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&minRect)), wsOverlappedWindow, 0, 0)
				runtime.KeepAlive(&minRect)
			}
			if w.cfg.MinWidth > 0 {
				mmi.ptMinTrackSize.x = minRect.right - minRect.left
			}
			if w.cfg.MinHeight > 0 {
				mmi.ptMinTrackSize.y = minRect.bottom - minRect.top
			}
			return 0
		}

	case wmNCCalcSize:
		if w.cfg.Frameless && !w.fullscreen {
			return w.handleNCCalcSize(wParam, lParam)
		}

	case wmNCHitTest:
		if w.cfg.Frameless && !w.fullscreen {
			return w.handleNCHitTest(lParam)
		}

	case wmSetCursor:
		if uint16(lParam&0xFFFF) == htClient {
			w.applyCursor()
			return 1
		}

	case wmSize:
		nw := int(lParam & 0xFFFF)
		nh := int((lParam >> 16) & 0xFFFF)
		if nw != w.width || nh != w.height {
			w.width = nw
			w.height = nh
			w.emit(Event{Type: EventResize, Width: nw, Height: nh})
			w.emit(Event{Type: EventPaint})
		}
		return 0

	case wmDropFiles:
		hDrop := wParam
		var pt point
		procDragQueryPoint.Call(hDrop, uintptr(unsafe.Pointer(&pt)))
		runtime.KeepAlive(&pt)

		count, _, _ := procDragQueryFileW.Call(hDrop, ^uintptr(0), 0, 0)
		var files []string
		if count > 0 {
			files = make([]string, 0, count)
			for i := uintptr(0); i < count; i++ {
				length, _, _ := procDragQueryFileW.Call(hDrop, i, 0, 0)
				if length == 0 {
					continue
				}
				buf := make([]uint16, length+1)
				procDragQueryFileW.Call(hDrop, i, uintptr(unsafe.Pointer(&buf[0])), length+1)
				runtime.KeepAlive(buf)
				files = append(files, windows.UTF16ToString(buf))
			}
		}
		procDragFinish.Call(hDrop)
		if len(files) > 0 {
			w.emit(Event{
				Type:  EventDropFiles,
				X:     int(pt.x),
				Y:     int(pt.y),
				Files: files,
			})
		}
		return 0

	case wmDPIChanged:
		newDPI := int(wParam & 0xFFFF)
		if newDPI > 0 {
			w.scale = float32(newDPI) / 96.0
			w.emit(Event{Type: EventDPIChange, Scale: w.scale})
			w.emit(Event{Type: EventPaint})
		}
		return 0

	case wmPaint:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		w.mu.Lock()
		if len(w.pixels) > 0 && w.bufW > 0 && w.bufH > 0 {
			blitBGRA(windows.Handle(hdc), w.width, w.height, w.pixels, w.bufW, w.bufH)
		}
		w.mu.Unlock()
		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		runtime.KeepAlive(&ps)
		return 0

	case wmAppWakeUp:
		w.emit(Event{Type: EventWakeUp})
		return 0

	case wmMouseMove:
		w.emit(Event{
			Type: EventMouseMove,
			X:    signExtend16(lParam),
			Y:    signExtend16(lParam >> 16),
			Mods: getModifiers(),
		})
		return 0

	case wmLButtonDown, wmRButtonDown, wmMButtonDown:
		procSetCapture.Call(uintptr(hwnd))
		btn := MouseLeft
		if msg == wmRButtonDown {
			btn = MouseRight
		} else if msg == wmMButtonDown {
			btn = MouseMiddle
		}
		w.emit(Event{
			Type:   EventMouseDown,
			Button: btn,
			X:      signExtend16(lParam),
			Y:      signExtend16(lParam >> 16),
			Mods:   getModifiers(),
		})
		return 0

	case wmLButtonUp, wmRButtonUp, wmMButtonUp:
		btn := MouseLeft
		if msg == wmRButtonUp {
			btn = MouseRight
		} else if msg == wmMButtonUp {
			btn = MouseMiddle
		}
		w.emit(Event{
			Type:   EventMouseUp,
			Button: btn,
			X:      signExtend16(lParam),
			Y:      signExtend16(lParam >> 16),
			Mods:   getModifiers(),
		})
		w.ReleaseCapture()
		return 0

	case wmMouseWheel:
		delta := float32(int16(uint16((wParam >> 16) & 0xFFFF))) / 120.0
		w.emit(Event{
			Type:   EventMouseWheel,
			WheelY: delta,
			Mods:   getModifiers(),
		})
		return 0

	case wmMouseHWheel:
		delta := float32(int16(uint16((wParam >> 16) & 0xFFFF))) / 120.0
		w.emit(Event{
			Type:   EventMouseWheel,
			WheelX: delta,
			Mods:   getModifiers(),
		})
		return 0

	case wmCaptureChanged:
		if !w.releasingCapture && lParam != uintptr(hwnd) {
			w.emit(Event{Type: EventCaptureLost})
		}
		return 0

	case wmKillFocus:
		// При потере фокуса (Alt+Tab) отпускаем захват мыши и сбрасываем фокус виджетов
		w.ReleaseCapture()
		w.emit(Event{Type: EventFocusLost})
		return 0

	case wmCancelMode:
		w.ReleaseCapture()
		w.emit(Event{Type: EventCaptureLost})
		return 0

	case wmKeyDown, wmSysKeyDown:
		scan := uint16((lParam >> 16) & 0xFF)
		w.emit(Event{
			Type:     EventKeyDown,
			KeyCode:  uint32(wParam),
			ScanCode: scan,
			Mods:     getModifiers(),
		})
		return 0

	case wmKeyUp, wmSysKeyUp:
		scan := uint16((lParam >> 16) & 0xFF)
		w.emit(Event{
			Type:     EventKeyUp,
			KeyCode:  uint32(wParam),
			ScanCode: scan,
			Mods:     getModifiers(),
		})
		return 0

	case wmChar:
		r := rune(wParam)
		if r >= 32 {
			w.emit(Event{
				Type: EventChar,
				Char: r,
				Mods: getModifiers(),
			})
		}
		return 0

	case wmClose:
		if w.onClose != nil && !w.onClose() {
			return 0
		}
		w.closing = true
		if w.handler != nil {
			w.handler(Event{Type: EventClose})
		}
		procDestroyWindow.Call(uintptr(hwnd))
		return 0

	case wmDestroy:
		windowMapMu.Lock()
		delete(windowMap, hwnd)
		remaining := len(windowMap)
		windowMapMu.Unlock()
		if remaining == 0 {
			procPostQuitMessage.Call(0)
		}
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

