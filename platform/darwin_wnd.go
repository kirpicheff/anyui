//go:build darwin && cgo

package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework QuartzCore -framework UserNotifications
#include <stdlib.h>
#include "darwin_bridge.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

func init() {
	// Закрепляем главную горутину за главным потоком ОС (Thread 0) до старта планировщика,
	// так как Cocoa/AppKit и dispatch_get_main_queue требуют выполнения на main thread.
	runtime.LockOSThread()
}

var (
	darwinNextWinID atomic.Uint64
	darwinWinMu     sync.RWMutex
	darwinWindows   = make(map[uint64]*darwinWindow)
)

type darwinWindow struct {
	id      uint64
	cfg     WindowConfig
	handle  uintptr
	handler func(Event)

	mu           sync.Mutex
	onClose      func() bool
	width        int
	height       int
	scale        float32
	closed       bool
	capturing    bool
	lastModFlags uint64
}

var (
	_ Window          = (*darwinWindow)(nil)
	_ WindowActivator = (*darwinWindow)(nil)
	_ WindowTopMoster = (*darwinWindow)(nil)
	_ TitleBarStyler  = (*darwinWindow)(nil)
	_ WindowStater    = (*darwinWindow)(nil)
)

// NewWindow создаёт экземпляр нативного окна macOS (AppKit).
func NewWindow(cfg WindowConfig) (Window, error) {
	if cfg.Width <= 0 {
		cfg.Width = 800
	}
	if cfg.Height <= 0 {
		cfg.Height = 600
	}
	return &darwinWindow{
		id:    darwinNextWinID.Add(1),
		cfg:   cfg,
		scale: 1.0,
	}, nil
}

// RunEventLoop запускает главный цикл событий Cocoa/AppKit на текущем потоке ОС.
func RunEventLoop(onStart func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	fmt.Fprintln(os.Stderr, "[anyui-darwin] RunEventLoop: старт инициализации AppKit")
	C.anyui_darwin_init_app()

	if onStart != nil {
		fmt.Fprintln(os.Stderr, "[anyui-darwin] RunEventLoop: вызов onStart")
		if err := onStart(); err != nil {
			fmt.Fprintf(os.Stderr, "[anyui-darwin] RunEventLoop: ошибка onStart: %v\n", err)
			return err
		}
	}

	fmt.Fprintln(os.Stderr, "[anyui-darwin] RunEventLoop: запуск [NSApp run]")
	C.anyui_darwin_run_app()
	fmt.Fprintln(os.Stderr, "[anyui-darwin] RunEventLoop: выход из [NSApp run]")
	return nil
}

func (w *darwinWindow) Run(handler func(Event)) error {
	return RunEventLoop(func() error {
		return w.Open(handler)
	})
}

func (w *darwinWindow) Open(handler func(Event)) error {
	if handler == nil {
		return errors.New("обработчик событий не задан")
	}

	w.mu.Lock()
	w.handler = handler
	w.mu.Unlock()

	cTitle := C.CString(w.cfg.Title)
	defer C.free(unsafe.Pointer(cTitle))

	darkTitle := C.int(0)
	if w.cfg.DarkTitle {
		darkTitle = 1
	}
	frameless := C.int(0)
	if w.cfg.Frameless {
		frameless = 1
	}
	acceptDrop := C.int(0)
	if w.cfg.AcceptDropFiles {
		acceptDrop = 1
	}

	fmt.Fprintf(os.Stderr, "[anyui-darwin] Open(id=%d): создание NSWindow\n", w.id)
	handle := uintptr(C.anyui_darwin_create_window(
		C.uint64_t(w.id),
		cTitle,
		C.int(w.cfg.Width),
		C.int(w.cfg.Height),
		C.int(w.cfg.MinWidth),
		C.int(w.cfg.MinHeight),
		darkTitle,
		frameless,
		acceptDrop,
	))
	if handle == 0 {
		return errors.New("не удалось создать NSWindow")
	}

	w.mu.Lock()
	w.handle = handle
	w.closed = false
	w.mu.Unlock()

	darwinWinMu.Lock()
	darwinWindows[w.id] = w
	darwinWinMu.Unlock()

	if w.cfg.InitialState != nil && isValidWindowState(*w.cfg.InitialState) {
		w.RestoreState(*w.cfg.InitialState)
	}
	if w.cfg.AlwaysOnTop {
		w.SetAlwaysOnTop(true)
	}

	fmt.Fprintf(os.Stderr, "[anyui-darwin] Open(id=%d): показ окна и первичный рендер\n", w.id)
	C.anyui_darwin_show_window(C.uintptr_t(handle))

	var cw, ch C.int
	var cscale C.float
	C.anyui_darwin_get_size(C.uintptr_t(handle), &cw, &ch, &cscale)

	scale := normalizeDarwinScale(float64(cscale))
	pw, ph := int(cw), int(ch)
	if pw < 1 {
		pw = w.cfg.Width
	}
	if ph < 1 {
		ph = w.cfg.Height
	}

	w.mu.Lock()
	w.width = pw
	w.height = ph
	w.scale = scale
	w.mu.Unlock()

	handler(Event{Type: EventDPIChange, Scale: scale})
	handler(Event{Type: EventResize, Width: pw, Height: ph})
	handler(Event{Type: EventPaint})

	return nil
}

func (w *darwinWindow) Present(pixels []byte, width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	need := width * height * 4
	if len(pixels) < need {
		return
	}

	w.mu.Lock()
	handle := w.handle
	closed := w.closed
	w.mu.Unlock()
	if handle == 0 || closed {
		return
	}

	C.anyui_darwin_present(C.uintptr_t(handle), unsafe.Pointer(&pixels[0]), C.int(width), C.int(height))
	runtime.KeepAlive(pixels)
}

func (w *darwinWindow) Invalidate() {
	w.mu.Lock()
	handle := w.handle
	closed := w.closed
	w.mu.Unlock()
	if handle != 0 && !closed {
		C.anyui_darwin_invalidate(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) WakeUp() {
	w.mu.Lock()
	closed := w.closed
	w.mu.Unlock()
	if !closed {
		C.anyui_darwin_wakeup(C.uint64_t(w.id))
	}
}

func (w *darwinWindow) Show() {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle != 0 {
		fmt.Fprintf(os.Stderr, "[anyui-darwin] Open(id=%d): показ окна и первичный рендер\n", w.id)
	C.anyui_darwin_show_window(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) Hide() {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle != 0 {
		C.anyui_darwin_hide_window(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) Close() {
	w.mu.Lock()
	handle := w.handle
	closed := w.closed
	w.mu.Unlock()
	if handle != 0 && !closed {
		C.anyui_darwin_close_window(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) SetTitle(title string) {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return
	}
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.anyui_darwin_set_title(C.uintptr_t(handle), cTitle)
}

func (w *darwinWindow) SetCursor(cursor CursorType) {
	w.mu.Lock()
	handle := w.handle
	closed := w.closed
	w.mu.Unlock()
	if handle != 0 && !closed {
		C.anyui_darwin_set_cursor(C.uintptr_t(handle), C.int(cursor))
	}
}

func (w *darwinWindow) SetCustomCursor(cur *CustomCursor) {
	// Кастомный BGRA-курсор на macOS пока не реализован; оставляем текущий системный курсор.
	_ = cur
}

func (w *darwinWindow) Size() (int, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.width, w.height
}

func (w *darwinWindow) Scale() float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.scale
}

func (w *darwinWindow) Minimize() {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle != 0 {
		C.anyui_darwin_minimize(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) Maximize() {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle != 0 {
		C.anyui_darwin_maximize(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) Restore() {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle != 0 {
		C.anyui_darwin_restore(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) IsMaximized() bool {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return false
	}
	return C.anyui_darwin_is_maximized(C.uintptr_t(handle)) != 0
}

func (w *darwinWindow) IsMinimized() bool {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return false
	}
	return C.anyui_darwin_is_minimized(C.uintptr_t(handle)) != 0
}

func (w *darwinWindow) SetFullscreen(fullscreen bool) {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return
	}
	flag := C.int(0)
	if fullscreen {
		flag = 1
	}
	C.anyui_darwin_set_fullscreen(C.uintptr_t(handle), flag)
}

func (w *darwinWindow) IsFullscreen() bool {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return false
	}
	return C.anyui_darwin_is_fullscreen(C.uintptr_t(handle)) != 0
}

func (w *darwinWindow) SetAlwaysOnTop(onTop bool) {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return
	}
	flag := C.int(0)
	if onTop {
		flag = 1
	}
	C.anyui_darwin_set_always_on_top(C.uintptr_t(handle), flag)
}

func (w *darwinWindow) IsAlwaysOnTop() bool {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return false
	}
	return C.anyui_darwin_is_always_on_top(C.uintptr_t(handle)) != 0
}

func (w *darwinWindow) Position() (int, int) {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return 0, 0
	}
	var cx, cy C.int
	C.anyui_darwin_get_position(C.uintptr_t(handle), &cx, &cy)
	return int(cx), int(cy)
}

func (w *darwinWindow) SetPosition(x, y int) {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle != 0 {
		C.anyui_darwin_set_position(C.uintptr_t(handle), C.int(x), C.int(y))
	}
}

func (w *darwinWindow) SetSize(width, height int) {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle != 0 {
		C.anyui_darwin_set_size(C.uintptr_t(handle), C.int(width), C.int(height))
	}
}

func (w *darwinWindow) SetOnClose(fn func() bool) {
	w.mu.Lock()
	w.onClose = fn
	w.mu.Unlock()
}

func (w *darwinWindow) NativeHandle() uintptr {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.handle
}

func (w *darwinWindow) SetCapture() {
	w.mu.Lock()
	handle := w.handle
	closed := w.closed
	if handle != 0 && !closed {
		w.capturing = true
	}
	w.mu.Unlock()
	if handle != 0 && !closed {
		C.anyui_darwin_set_capture(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) ReleaseCapture() {
	w.mu.Lock()
	handle := w.handle
	wasCapturing := w.capturing
	w.capturing = false
	w.mu.Unlock()
	if handle != 0 && wasCapturing {
		C.anyui_darwin_release_capture(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) DoubleClickTimeMs() int {
	return int(C.anyui_darwin_double_click_ms())
}

func (w *darwinWindow) BringToFront() {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle != 0 {
		C.anyui_darwin_bring_to_front(C.uintptr_t(handle))
	}
}

func (w *darwinWindow) SetDarkTitle(dark bool) {
	w.mu.Lock()
	handle := w.handle
	w.mu.Unlock()
	if handle == 0 {
		return
	}
	flag := C.int(0)
	if dark {
		flag = 1
	}
	C.anyui_darwin_set_dark_title(C.uintptr_t(handle), flag)
}

func (w *darwinWindow) SetTitleBarColors(caption, text, border *WindowColor) {
	_, _, _ = caption, text, border
}

func (w *darwinWindow) State() WindowState {
	x, y := w.Position()
	pw, ph := w.Size()
	scale := w.Scale()
	if scale <= 0 {
		scale = 1.0
	}
	return WindowState{
		X:         x,
		Y:         y,
		Width:     int(float32(pw) / scale),
		Height:    int(float32(ph) / scale),
		Maximized: w.IsMaximized(),
	}
}

func (w *darwinWindow) RestoreState(state WindowState) {
	if !isValidWindowState(state) {
		return
	}
	w.SetSize(state.Width, state.Height)
	w.SetPosition(state.X, state.Y)
	if state.Maximized {
		w.Maximize()
	}
}

func lookupDarwinWindow(winID uint64) *darwinWindow {
	darwinWinMu.RLock()
	w := darwinWindows[winID]
	darwinWinMu.RUnlock()
	return w
}

//export goDarwinOnClose
func goDarwinOnClose(winID C.uint64_t) C.int {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return 1
	}
	w.mu.Lock()
	fn := w.onClose
	w.mu.Unlock()
	if fn != nil && !fn() {
		return 0
	}
	return 1
}

//export goDarwinOnDestroy
func goDarwinOnDestroy(winID C.uint64_t) {
	darwinWinMu.Lock()
	w := darwinWindows[uint64(winID)]
	delete(darwinWindows, uint64(winID))
	remaining := len(darwinWindows)
	darwinWinMu.Unlock()

	if w != nil {
		w.mu.Lock()
		handle := w.handle
		handler := w.handler
		w.handle = 0
		w.closed = true
		w.mu.Unlock()

		if handler != nil {
			handler(Event{Type: EventClose})
		}
		if handle != 0 {
			C.anyui_darwin_destroy_window(C.uintptr_t(handle))
		}
	}

	if remaining == 0 {
		C.anyui_darwin_stop_app()
	}
}

//export goDarwinOnResize
func goDarwinOnResize(winID C.uint64_t, width, height C.int) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	pw, ph := int(width), int(height)
	w.mu.Lock()
	w.width = pw
	w.height = ph
	handler := w.handler
	w.mu.Unlock()
	if handler != nil {
		handler(Event{Type: EventResize, Width: pw, Height: ph})
	}
}

//export goDarwinOnDPIChange
func goDarwinOnDPIChange(winID C.uint64_t, scale C.float) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	s := normalizeDarwinScale(float64(scale))
	w.mu.Lock()
	w.scale = s
	handler := w.handler
	w.mu.Unlock()
	if handler != nil {
		handler(Event{Type: EventDPIChange, Scale: s})
	}
}

//export goDarwinOnPaint
func goDarwinOnPaint(winID C.uint64_t) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	w.mu.Lock()
	handler := w.handler
	closed := w.closed
	w.mu.Unlock()
	if handler != nil && !closed {
		handler(Event{Type: EventPaint})
	}
}

//export goDarwinOnWakeUp
func goDarwinOnWakeUp(winID C.uint64_t) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	w.mu.Lock()
	handler := w.handler
	closed := w.closed
	w.mu.Unlock()
	if handler != nil && !closed {
		handler(Event{Type: EventWakeUp})
	}
}

func darwinMouseButton(btn int) MouseButton {
	switch btn {
	case 0:
		return MouseLeft
	case 1:
		return MouseRight
	case 2:
		return MouseMiddle
	default:
		return 0
	}
}

//export goDarwinOnMouse
func goDarwinOnMouse(winID C.uint64_t, eventKind C.int, x, y, viewH, scale C.double, button C.int, mods C.uint64_t) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	w.mu.Lock()
	handler := w.handler
	closed := w.closed
	if eventKind == 3 {
		w.capturing = false
	}
	w.mu.Unlock()
	if handler == nil || closed {
		return
	}

	px, py := appKitPointToGoUIPixels(float64(x), float64(y), float64(viewH), float64(scale))
	m := darwinModifiersFromFlags(uint64(mods))
	btn := darwinMouseButton(int(button))

	var evType EventType
	switch int(eventKind) {
	case 1:
		evType = EventMouseMove
		btn = 0
	case 2:
		evType = EventMouseDown
	case 3:
		evType = EventMouseUp
	default:
		return
	}

	handler(Event{
		Type:   evType,
		X:      px,
		Y:      py,
		Button: btn,
		Mods:   m,
	})
}

//export goDarwinOnWheel
func goDarwinOnWheel(winID C.uint64_t, x, y, viewH, scale, deltaX, deltaY C.double, precise C.int, mods C.uint64_t) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	w.mu.Lock()
	handler := w.handler
	closed := w.closed
	w.mu.Unlock()
	if handler == nil || closed {
		return
	}

	px, py := appKitPointToGoUIPixels(float64(x), float64(y), float64(viewH), float64(scale))
	wx, wy := darwinNormalizeWheel(float64(deltaX), float64(deltaY), precise != 0)
	if wx == 0 && wy == 0 {
		return
	}

	handler(Event{
		Type:   EventMouseWheel,
		X:      px,
		Y:      py,
		WheelX: wx,
		WheelY: wy,
		Mods:   darwinModifiersFromFlags(uint64(mods)),
	})
}

//export goDarwinOnKey
func goDarwinOnKey(winID C.uint64_t, isDown C.int, keyCode C.uint16_t, mods C.uint64_t, charsUTF8 *C.char) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	w.mu.Lock()
	handler := w.handler
	closed := w.closed
	w.mu.Unlock()
	if handler == nil || closed {
		return
	}

	vk := darwinKeycodeToVK(uint16(keyCode))
	m := darwinModifiersFromFlags(uint64(mods))

	if isDown != 0 {
		if vk != 0 {
			handler(Event{
				Type:     EventKeyDown,
				KeyCode:  vk,
				ScanCode: uint16(keyCode),
				Mods:     m,
			})
		}
		if charsUTF8 != nil {
			for _, r := range darwinExtractChars(C.GoString(charsUTF8), m) {
				handler(Event{
					Type: EventChar,
					Char: r,
					Mods: m,
				})
			}
		}
		return
	}

	if vk != 0 {
		handler(Event{
			Type:     EventKeyUp,
			KeyCode:  vk,
			ScanCode: uint16(keyCode),
			Mods:     m,
		})
	}
}

//export goDarwinOnFlagsChanged
func goDarwinOnFlagsChanged(winID C.uint64_t, mods C.uint64_t) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	newFlags := uint64(mods)
	w.mu.Lock()
	prevFlags := w.lastModFlags
	w.lastModFlags = newFlags
	handler := w.handler
	closed := w.closed
	w.mu.Unlock()
	if handler == nil || closed {
		return
	}

	m := darwinModifiersFromFlags(newFlags)
	for _, ch := range darwinChangedModifiers(prevFlags, newFlags) {
		evType := EventKeyUp
		if ch.down {
			evType = EventKeyDown
		}
		handler(Event{
			Type:    evType,
			KeyCode: ch.vk,
			Mods:    m,
		})
	}
}

//export goDarwinOnFocusLost
func goDarwinOnFocusLost(winID C.uint64_t) {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	w.mu.Lock()
	handler := w.handler
	closed := w.closed
	wasCapturing := w.capturing
	w.capturing = false
	w.lastModFlags = 0
	w.mu.Unlock()
	if handler == nil || closed {
		return
	}
	if wasCapturing {
		handler(Event{Type: EventCaptureLost})
	}
	handler(Event{Type: EventFocusLost})
}

//export goDarwinHitTest
func goDarwinHitTest(winID C.uint64_t, x, y, viewH, scale C.double) C.int {
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return C.int(HitClient)
	}
	w.mu.Lock()
	ww := w.width
	wh := w.height
	s := float64(normalizeDarwinScale(float64(scale)))
	customHit := w.cfg.HitTest
	titleH := w.cfg.TitleBarHeight
	w.mu.Unlock()

	if titleH <= 0 {
		titleH = 36
	}
	borderPx := int(math.Round(6.0 * s))
	titlePx := int(math.Round(float64(titleH) * s))
	px, py := appKitPointToGoUIPixels(float64(x), float64(y), float64(viewH), float64(scale))

	hit := ComputeFramelessHitTest(px, py, ww, wh, borderPx, titlePx, w.IsMaximized(), customHit)
	return C.int(hit)
}

//export goDarwinOnDropFiles
func goDarwinOnDropFiles(winID C.uint64_t, pathsNewlineSep *C.char) {
	if pathsNewlineSep == nil {
		return
	}
	w := lookupDarwinWindow(uint64(winID))
	if w == nil {
		return
	}
	w.mu.Lock()
	handler := w.handler
	closed := w.closed
	w.mu.Unlock()
	if handler == nil || closed {
		return
	}
	raw := C.GoString(pathsNewlineSep)
	var files []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	if len(files) > 0 {
		handler(Event{
			Type:  EventDropFiles,
			Files: files,
		})
	}
}

// CursorScreenPos возвращает текущие экранные координаты курсора мыши (пока не поддерживается на macOS).
func CursorScreenPos() (xPx, yPx int, ok bool) {
	return 0, 0, false
}

