//go:build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modOle32Utils = windows.NewLazySystemDLL("ole32.dll")

	procShellExecuteW            = modShell32.NewProc("ShellExecuteW")
	procILCreateFromPathW        = modShell32.NewProc("ILCreateFromPathW")
	procILFree                   = modShell32.NewProc("ILFree")
	procSHOpenFolderAndSelectItems = modShell32.NewProc("SHOpenFolderAndSelectItems")
	procCoInitializeExUtils      = modOle32Utils.NewProc("CoInitializeEx")
	procCoUninitializeUtils      = modOle32Utils.NewProc("CoUninitialize")
)

const swShowNormalExec = 1

// OpenURL открывает веб-ссылку или mailto-адрес в приложении по умолчанию.
func OpenURL(rawURL string) error {
	if err := validateOpenURL(rawURL); err != nil {
		return err
	}
	verbPtr, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	urlPtr, err := windows.UTF16PtrFromString(strings.TrimSpace(rawURL))
	if err != nil {
		return err
	}

	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(urlPtr)),
		0,
		0,
		swShowNormalExec,
	)
	runtime.KeepAlive(verbPtr)
	runtime.KeepAlive(urlPtr)
	if ret <= 32 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return errors.New("не удалось открыть URL")
	}
	return nil
}

// RevealInFileManager открывает Проводник Windows и выделяет указанный файл или папку.
func RevealInFileManager(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("путь не задан")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absPath); err != nil {
		return err
	}

	// Предпочитаем нативный SHOpenFolderAndSelectItems без запуска дочерних процессов
	if err := procSHOpenFolderAndSelectItems.Find(); err == nil &&
		procILCreateFromPathW.Find() == nil &&
		procILFree.Find() == nil {
		hr, _, _ := procCoInitializeExUtils.Call(0, 0x2) // COINIT_APARTMENTTHREADED
		if hr == 0 || hr == 1 {
			defer procCoUninitializeUtils.Call()
		}
		pathPtr, err := windows.UTF16PtrFromString(absPath)
		if err == nil {
			pidl, _, _ := procILCreateFromPathW.Call(uintptr(unsafe.Pointer(pathPtr)))
			runtime.KeepAlive(pathPtr)
			if pidl != 0 {
				defer procILFree.Call(pidl)
				res, _, _ := procSHOpenFolderAndSelectItems.Call(pidl, 0, 0, 0)
				if res == 0 {
					return nil
				}
			}
		}
	}

	// Запасной вариант через explorer.exe /select
	verbPtr, _ := windows.UTF16PtrFromString("open")
	exePtr, _ := windows.UTF16PtrFromString("explorer.exe")
	argPtr, err := windows.UTF16PtrFromString(`/select,"` + absPath + `"`)
	if err != nil {
		return err
	}
	ret, _, callErr := procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argPtr)),
		0,
		swShowNormalExec,
	)
	runtime.KeepAlive(verbPtr)
	runtime.KeepAlive(exePtr)
	runtime.KeepAlive(argPtr)
	if ret <= 32 {
		if callErr != nil && callErr != windows.ERROR_SUCCESS {
			return callErr
		}
		return errors.New("не удалось открыть файл в проводнике")
	}
	return nil
}

// bringToFront поднимает окно на передний план, даже если оно было свёрнуто.
func (w *win32Window) BringToFront() {
	if w.hwnd == 0 {
		return
	}
	if w.IsMinimized() {
		w.Restore()
	} else {
		w.Show()
	}
	procSetForegroundWindow.Call(uintptr(w.hwnd))
}

