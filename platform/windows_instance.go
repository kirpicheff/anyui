//go:build windows

package platform

import (
	"encoding/binary"
	"errors"
	"hash/fnv"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procCreateMutexW            = modKernel32.NewProc("CreateMutexW")
	procFindWindowW             = modUser32.NewProc("FindWindowW")
	procSendMessageW            = modUser32.NewProc("SendMessageW")
	procGetWindowThreadProcessId = modUser32.NewProc("GetWindowThreadProcessId")
	procAllowSetForegroundWindow = modUser32.NewProc("AllowSetForegroundWindow")
)

const (
	wmCopyData          = 0x004A
	singleInstanceMagic = 0x414E5955 // "ANYU"
)

type copyDataStruct struct {
	dwData uintptr
	cbData uint32
	lpData uintptr
}

var (
	ipcMapMu     sync.RWMutex
	ipcMap       = make(map[windows.Handle]*SingleInstanceGuard)
	ipcWndProcCB = syscall.NewCallback(win32IPCWndProc)
)

// SingleInstanceGuard удерживает блокировку единственного экземпляра приложения
// и принимает аргументы командной строки от повторных запусков.
type SingleInstanceGuard struct {
	appHash  string
	mutex    windows.Handle
	hwnd     windows.Handle
	onSecond func(args []string)
}

// AcquireSingleInstance пытается захватить блокировку единственного экземпляра для appID.
// Если копия уже запущена, отправляет ей args и возвращает (nil, true, nil).
// Если это первый экземпляр, возвращает (guard, false, nil).
func AcquireSingleInstance(appID string, args []string, onSecond func(args []string)) (*SingleInstanceGuard, bool, error) {
	if appID == "" {
		return nil, false, errors.New("идентификатор приложения не задан")
	}
	appHash := hashAppID(appID)

	mutexName, err := windows.UTF16PtrFromString("Local\\AnyUI_SI_" + appHash)
	if err != nil {
		return nil, false, err
	}

	hMutex, _, lastErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(mutexName)))
	runtime.KeepAlive(mutexName)
	if hMutex == 0 {
		return nil, false, lastErr
	}

	if lastErr == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(windows.Handle(hMutex))
		_ = notifyFirstInstance(appHash, args)
		return nil, true, nil
	}

	return &SingleInstanceGuard{
		appHash:  appHash,
		mutex:    windows.Handle(hMutex),
		onSecond: onSecond,
	}, false, nil
}

// SetCallback задаёт обработчик повторного запуска приложения.
func (g *SingleInstanceGuard) SetCallback(fn func(args []string)) {
	if g != nil {
		g.onSecond = fn
	}
}

// StartIPC создаёт скрытое окно для приёма сообщений от повторных запусков.
// Должен вызываться из потока цикла событий Win32.
func (g *SingleInstanceGuard) StartIPC() error {
	if g == nil || g.hwnd != 0 {
		return nil
	}

	classNameStr, winNameStr := ipcWindowNames(g.appHash)
	className, err := windows.UTF16PtrFromString(classNameStr)
	if err != nil {
		return err
	}
	winName, err := windows.UTF16PtrFromString(winNameStr)
	if err != nil {
		return err
	}

	hInstance, _, _ := procGetModuleHandleW.Call(0)
	wc := wndClassExW{
		cbSize:        uint32(unsafe.Sizeof(wndClassExW{})),
		lpfnWndProc:   ipcWndProcCB,
		hInstance:     windows.Handle(hInstance),
		lpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	runtime.KeepAlive(&wc)
	runtime.KeepAlive(className)

	hwndRaw, _, createErr := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(winName)),
		wsPopup,
		0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	runtime.KeepAlive(className)
	runtime.KeepAlive(winName)
	if hwndRaw == 0 {
		return createErr
	}

	g.hwnd = windows.Handle(hwndRaw)
	ipcMapMu.Lock()
	ipcMap[g.hwnd] = g
	ipcMapMu.Unlock()
	return nil
}

// Close освобождает скрытое IPC-окно и именованный мьютекс.
func (g *SingleInstanceGuard) Close() error {
	if g == nil {
		return nil
	}
	if g.hwnd != 0 {
		ipcMapMu.Lock()
		delete(ipcMap, g.hwnd)
		ipcMapMu.Unlock()
		procDestroyWindow.Call(uintptr(g.hwnd))
		g.hwnd = 0
	}
	if g.mutex != 0 {
		_ = windows.CloseHandle(g.mutex)
		g.mutex = 0
	}
	return nil
}

func hashAppID(appID string) string {
	h1 := fnv.New64a()
	_, _ = h1.Write([]byte(appID))
	h2 := fnv.New64a()
	_, _ = h2.Write([]byte("anyui-si:"))
	_, _ = h2.Write([]byte(appID))
	return strconv.FormatUint(h1.Sum64(), 16) + "_" + strconv.FormatUint(h2.Sum64(), 16)
}

func ipcWindowNames(appHash string) (className, windowName string) {
	return "AnyUI_IPC_Class_" + appHash, "AnyUI_IPC_Window_" + appHash
}

func encodeInstanceArgs(args []string) ([]byte, error) {
	total := 4
	for _, a := range args {
		total += 4 + len(a)
	}
	buf := make([]byte, 4, total)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(len(args)))
	var lenBuf [4]byte
	for _, a := range args {
		binary.LittleEndian.PutUint32(lenBuf[:], uint32(len(a)))
		buf = append(buf, lenBuf[:]...)
		buf = append(buf, a...)
	}
	return buf, nil
}

func decodeInstanceArgs(data []byte) ([]string, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if len(data) < 4 {
		return nil, errors.New("некорректный заголовок аргументов IPC")
	}
	count := int(binary.LittleEndian.Uint32(data[0:4]))
	pos := 4
	if count == 0 {
		return []string{}, nil
	}
	if count < 0 || count > (len(data)-4)/4 {
		return nil, errors.New("некорректное число аргументов IPC")
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		if pos+4 > len(data) {
			return nil, errors.New("усечённая длина аргумента IPC")
		}
		n := int(binary.LittleEndian.Uint32(data[pos : pos+4]))
		pos += 4
		if n < 0 || pos+n > len(data) {
			return nil, errors.New("усечённые данные аргумента IPC")
		}
		args = append(args, string(data[pos:pos+n]))
		pos += n
	}
	return args, nil
}

func notifyFirstInstance(appHash string, args []string) error {
	classNameStr, winNameStr := ipcWindowNames(appHash)
	className, err := windows.UTF16PtrFromString(classNameStr)
	if err != nil {
		return err
	}
	winName, err := windows.UTF16PtrFromString(winNameStr)
	if err != nil {
		return err
	}

	hwnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(winName)))
	runtime.KeepAlive(className)
	runtime.KeepAlive(winName)
	if hwnd == 0 {
		return errors.New("окно первого экземпляра не найдено")
	}

	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	runtime.KeepAlive(&pid)
	if pid != 0 {
		procAllowSetForegroundWindow.Call(uintptr(pid))
	}

	payload, err := encodeInstanceArgs(args)
	if err != nil {
		return err
	}

	var cds copyDataStruct
	cds.dwData = singleInstanceMagic
	if len(payload) > 0 {
		cds.cbData = uint32(len(payload))
		cds.lpData = uintptr(unsafe.Pointer(&payload[0]))
	}
	procSendMessageW.Call(hwnd, wmCopyData, 0, uintptr(unsafe.Pointer(&cds)))
	runtime.KeepAlive(payload)
	runtime.KeepAlive(&cds)
	return nil
}

func win32IPCWndProc(hwnd windows.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == wmCopyData && lParam != 0 {
		ipcMapMu.RLock()
		g := ipcMap[hwnd]
		ipcMapMu.RUnlock()

		if g != nil {
			cds := (*copyDataStruct)(unsafe.Pointer(lParam))
			if cds.dwData == singleInstanceMagic {
				var raw []byte
				if cds.cbData > 0 && cds.lpData != 0 {
					src := unsafe.Slice((*byte)(unsafe.Pointer(cds.lpData)), cds.cbData)
					raw = make([]byte, len(src))
					copy(raw, src)
				}
				args, err := decodeInstanceArgs(raw)
				if err == nil && g.onSecond != nil {
					g.onSecond(args)
				}
				return 1
			}
		}
	}
	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

