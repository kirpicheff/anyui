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
	modGdi32Tray = windows.NewLazySystemDLL("gdi32.dll")

	procShellNotifyIconW     = modShell32.NewProc("Shell_NotifyIconW")
	procCreateIconIndirect   = modUser32.NewProc("CreateIconIndirect")
	procDestroyIcon          = modUser32.NewProc("DestroyIcon")
	procCreatePopupMenu      = modUser32.NewProc("CreatePopupMenu")
	procDestroyMenu          = modUser32.NewProc("DestroyMenu")
	procAppendMenuW          = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenuEx     = modUser32.NewProc("TrackPopupMenuEx")
	procGetCursorPos         = modUser32.NewProc("GetCursorPos")
	procSetForegroundWindow  = modUser32.NewProc("SetForegroundWindow")
	procRegisterWindowMsgW   = modUser32.NewProc("RegisterWindowMessageW")
	procCreateBitmap         = modGdi32Tray.NewProc("CreateBitmap")
	procCreateDIBSectionTray = modGdi32Tray.NewProc("CreateDIBSection")
	procDeleteObjectTray     = modGdi32Tray.NewProc("DeleteObject")
)

const (
	nimAdd    = 0x00000000
	nimModify = 0x00000001
	nimDelete = 0x00000002

	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	wmTrayCallback = 0x8002
	wmLButtonDbl   = 0x0203
	wmRButtonUpMsg = 0x0205
	wmLButtonUpMsg = 0x0202

	mfString    = 0x00000000
	mfSeparator = 0x00000800
	mfChecked   = 0x00000008
	mfGrayed    = 0x00000001

	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
	tpmBottomAlign = 0x0020
)

type notifyIconDataW struct {
	cbSize           uint32
	hWnd             windows.Handle
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            windows.Handle
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     windows.Handle
}

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  windows.Handle
	hbmColor windows.Handle
}

type bitmapInfoHeaderTray struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

var (
	trayMapMu        sync.RWMutex
	trayMap          = make(map[windows.Handle]*win32Tray)
	trayWndProcCB    = syscall.NewCallback(win32TrayWndProc)
	msgTaskbarCreate uint32
)

type win32Tray struct {
	hwnd  windows.Handle
	hIcon windows.Handle
	cfg   TrayConfig

	mu   sync.Mutex
	menu []TrayMenuItem
}

// NewTray создаёт иконку в системном трее Windows.
// Должен вызываться из потока цикла событий Win32.
func NewTray(cfg TrayConfig) (Tray, error) {
	if msgTaskbarCreate == 0 {
		name, _ := windows.UTF16PtrFromString("TaskbarCreated")
		r, _, _ := procRegisterWindowMsgW.Call(uintptr(unsafe.Pointer(name)))
		runtime.KeepAlive(name)
		msgTaskbarCreate = uint32(r)
	}

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	className, err := windows.UTF16PtrFromString("AnyUITrayWindowClass")
	if err != nil {
		return nil, err
	}
	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		lpfnWndProc:   trayWndProcCB,
		hInstance:     windows.Handle(hInstance),
		lpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(&wc)
	runtime.KeepAlive(className)

	hwndRaw, _, createErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0,
		0,
		0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	runtime.KeepAlive(className)
	if hwndRaw == 0 {
		return nil, createErr
	}

	t := &win32Tray{
		hwnd: windows.Handle(hwndRaw),
		cfg:  cfg,
		menu: append([]TrayMenuItem(nil), cfg.Menu...),
	}

	rgba, w, h := cfg.IconRGBA, cfg.IconWidth, cfg.IconHeight
	if len(rgba) == 0 || w <= 0 || h <= 0 {
		rgba, w, h = defaultTrayIconRGBA()
	}
	hIcon, err := createHIconFromRGBA(rgba, w, h)
	if err != nil {
		procDestroyWindow.Call(uintptr(t.hwnd))
		return nil, err
	}
	t.hIcon = hIcon

	trayMapMu.Lock()
	trayMap[t.hwnd] = t
	trayMapMu.Unlock()

	if err := t.shellNotify(nimAdd); err != nil {
		t.Close()
		return nil, err
	}
	return t, nil
}

func (t *win32Tray) shellNotify(action uintptr) error {
	var nid notifyIconDataW
	nid.cbSize = uint32(unsafe.Sizeof(nid))
	nid.hWnd = t.hwnd
	nid.uID = 1
	nid.uFlags = nifMessage | nifIcon | nifTip
	nid.uCallbackMessage = wmTrayCallback
	nid.hIcon = t.hIcon
	copyUTF16Fixed(nid.szTip[:], t.cfg.Tooltip)

	r, _, _ := procShellNotifyIconW.Call(action, uintptr(unsafe.Pointer(&nid)))
	runtime.KeepAlive(&nid)
	if r == 0 && action != nimDelete {
		return errors.New("ошибка Shell_NotifyIconW")
	}
	return nil
}

func (t *win32Tray) SetTooltip(tooltip string) error {
	t.mu.Lock()
	t.cfg.Tooltip = tooltip
	t.mu.Unlock()
	return t.shellNotify(nimModify)
}

func (t *win32Tray) SetIconRGBA(rgba []byte, width, height int) error {
	hIcon, err := createHIconFromRGBA(rgba, width, height)
	if err != nil {
		return err
	}
	oldIcon := t.hIcon
	t.hIcon = hIcon
	err = t.shellNotify(nimModify)
	if oldIcon != 0 {
		procDestroyIcon.Call(uintptr(oldIcon))
	}
	return err
}

func (t *win32Tray) SetMenu(items []TrayMenuItem) {
	t.mu.Lock()
	t.menu = append([]TrayMenuItem(nil), items...)
	t.mu.Unlock()
}

func (t *win32Tray) Close() error {
	if t.hwnd != 0 {
		_ = t.shellNotify(nimDelete)
		trayMapMu.Lock()
		delete(trayMap, t.hwnd)
		trayMapMu.Unlock()
		procDestroyWindow.Call(uintptr(t.hwnd))
		t.hwnd = 0
	}
	if t.hIcon != 0 {
		procDestroyIcon.Call(uintptr(t.hIcon))
		t.hIcon = 0
	}
	return nil
}

func (t *win32Tray) showPopupMenu() {
	t.mu.Lock()
	items := append([]TrayMenuItem(nil), t.menu...)
	t.mu.Unlock()

	if len(items) == 0 {
		return
	}

	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	for i, item := range items {
		cmdID := uintptr(i + 1)
		if item.Separator {
			procAppendMenuW.Call(hMenu, mfSeparator, 0, 0)
			continue
		}
		flags := uintptr(mfString)
		if item.Checked {
			flags |= mfChecked
		}
		if item.Disabled {
			flags |= mfGrayed
		}
		ptr, err := windows.UTF16PtrFromString(item.Label)
		if err != nil {
			continue
		}
		procAppendMenuW.Call(hMenu, flags, cmdID, uintptr(unsafe.Pointer(ptr)))
		runtime.KeepAlive(ptr)
	}

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	runtime.KeepAlive(&pt)

	// SetForegroundWindow обязателен, чтобы меню закрывалось при клике вне его области
	procSetForegroundWindow.Call(uintptr(t.hwnd))
	cmd, _, _ := procTrackPopupMenuEx.Call(
		hMenu,
		tpmRightButton|tpmReturnCmd|tpmBottomAlign,
		uintptr(pt.x),
		uintptr(pt.y),
		uintptr(t.hwnd),
		0,
	)
	procPostMessageW.Call(uintptr(t.hwnd), 0, 0, 0)

	if cmd > 0 {
		idx := int(cmd - 1)
		if idx >= 0 && idx < len(items) && !items[idx].Disabled && items[idx].OnClick != nil {
			items[idx].OnClick()
		}
	}
}

func win32TrayWndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	trayMapMu.RLock()
	t := trayMap[hwnd]
	trayMapMu.RUnlock()

	if t == nil {
		ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	}

	if msgTaskbarCreate != 0 && msg == msgTaskbarCreate {
		_ = t.shellNotify(nimAdd)
		return 0
	}

	if msg == wmTrayCallback {
		ev := uint32(lParam & 0xFFFF)
		switch ev {
		case wmLButtonUpMsg:
			if t.cfg.OnClick != nil {
				t.cfg.OnClick()
			}
		case wmLButtonDbl:
			if t.cfg.OnDoubleClick != nil {
				t.cfg.OnDoubleClick()
			}
		case wmRButtonUpMsg:
			if t.cfg.OnRightClick != nil {
				t.cfg.OnRightClick()
			}
			t.showPopupMenu()
		}
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func copyUTF16Fixed(dst []uint16, s string) {
	if len(dst) == 0 {
		return
	}
	enc := utf16.Encode([]rune(s))
	n := len(enc)
	if n >= len(dst) {
		n = len(dst) - 1
	}
	copy(dst[:n], enc[:n])
	dst[n] = 0
}

func createHIconFromRGBA(rgba []byte, width, height int) (windows.Handle, error) {
	if width <= 0 || height <= 0 || len(rgba) < width*height*4 {
		return 0, errors.New("некорректные размеры буфера иконки")
	}

	bi := bitmapInfoHeaderTray{
		biSize:        uint32(unsafe.Sizeof(bitmapInfoHeaderTray{})),
		biWidth:       int32(width),
		biHeight:      -int32(height), // top-down DIB
		biPlanes:      1,
		biBitCount:    32,
		biCompression: 0,
	}

	var bits unsafe.Pointer
	hColor, _, _ := procCreateDIBSectionTray.Call(
		0,
		uintptr(unsafe.Pointer(&bi)),
		0,
		uintptr(unsafe.Pointer(&bits)),
		0,
		0,
	)
	runtime.KeepAlive(&bi)
	if hColor == 0 || bits == nil {
		return 0, errors.New("не удалось создать DIB для иконки")
	}
	defer procDeleteObjectTray.Call(hColor)

	// Конвертируем RGBA -> BGRA в DIB-буфер
	dst := unsafe.Slice((*byte)(bits), width*height*4)
	for i := 0; i < width*height*4; i += 4 {
		dst[i+0] = rgba[i+2] // B
		dst[i+1] = rgba[i+1] // G
		dst[i+2] = rgba[i+0] // R
		dst[i+3] = rgba[i+3] // A
	}

	hMask, _, _ := procCreateBitmap.Call(uintptr(width), uintptr(height), 1, 1, 0)
	if hMask == 0 {
		return 0, errors.New("не удалось создать маску для иконки")
	}
	defer procDeleteObjectTray.Call(hMask)

	ii := iconInfo{
		fIcon:    1,
		hbmMask:  windows.Handle(hMask),
		hbmColor: windows.Handle(hColor),
	}
	hIcon, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	runtime.KeepAlive(&ii)
	if hIcon == 0 {
		return 0, errors.New("ошибка CreateIconIndirect")
	}
	return windows.Handle(hIcon), nil
}

// defaultTrayIconRGBA генерирует простую 16x16 иконку скруглённого квадрата, если пользователь не задал свою.
func defaultTrayIconRGBA() ([]byte, int, int) {
	const size = 16
	buf := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := x - 7
			dy := y - 7
			if x >= 1 && x <= 14 && y >= 1 && y <= 14 && !(dx*dx+dy*dy > 72) {
				idx := (y*size + x) * 4
				buf[idx+0] = 59  // R
				buf[idx+1] = 130 // G
				buf[idx+2] = 246 // B
				buf[idx+3] = 255 // A
			}
		}
	}
	return buf, size, size
}

