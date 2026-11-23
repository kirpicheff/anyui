//go:build windows

package platform

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procOleInitialize    = modOle32.NewProc("OleInitialize")
	procRegisterDragDrop = modOle32.NewProc("RegisterDragDrop")
	procRevokeDragDrop   = modOle32.NewProc("RevokeDragDrop")
	procReleaseStgMedium = modOle32.NewProc("ReleaseStgMedium")

	procChangeWindowMessageFilterEx = modUser32.NewProc("ChangeWindowMessageFilterEx")
)

const (
	wmCopyGlobalData = 0x0049
	msgfltAllow      = 1

	cfHDrop         = 15
	dvaspectContent = 1
	tymedHGlobal    = 1

	dropEffectNone = 0
	dropEffectCopy = 1

	sOK          = 0
	eNoInterface = 0x80004002
)

// comGUID представляет 128-битный идентификатор интерфейса COM.
type comGUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	iidIUnknown    = comGUID{0x00000000, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIDropTarget = comGUID{0x00000122, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}

	oleInitOnce sync.Once
)

type dropTargetVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	DragEnter      uintptr
	DragOver       uintptr
	DragLeave      uintptr
	Drop           uintptr
}

type dropTarget struct {
	vtbl *dropTargetVtbl
	w    *win32Window
}

var (
	globalDropTargetVtbl dropTargetVtbl
	initVtblOnce         sync.Once
)

type formatEtc struct {
	cfFormat uint16
	ptd      uintptr
	dwAspect uint32
	lindex   int32
	tymed    uint32
}

type stgMedium struct {
	tymed          uint32
	hGlobal        uintptr
	pUnkForRelease uintptr
}

func queryInterface(this uintptr, riid uintptr, ppvObject uintptr) uintptr {
	if ppvObject == 0 {
		return 0x80070057
	}
	guid := (*comGUID)(ptrFromWin32(riid))
	if *guid == iidIUnknown || *guid == iidIDropTarget {
		*(*uintptr)(ptrFromWin32(ppvObject)) = this
		return sOK
	}
	*(*uintptr)(ptrFromWin32(ppvObject)) = 0
	return eNoInterface
}

func addRef(this uintptr) uintptr {
	return 1
}

func release(this uintptr) uintptr {
	return 1
}

func checkDataObjectHasFiles(pDataObj uintptr) bool {
	if pDataObj == 0 {
		return false
	}
	vtbl := *(*uintptr)(ptrFromWin32(pDataObj))
	fnQueryGetData := *(*uintptr)(ptrFromWin32(vtbl + 5*unsafe.Sizeof(uintptr(0))))
	fEtc := formatEtc{
		cfFormat: cfHDrop,
		ptd:      0,
		dwAspect: dvaspectContent,
		lindex:   -1,
		tymed:    tymedHGlobal,
	}
	hr, _, _ := syscall.SyscallN(fnQueryGetData, pDataObj, uintptr(unsafe.Pointer(&fEtc)))
	runtime.KeepAlive(&fEtc)
	return hr == 0
}

func extractFilesFromDataObject(pDataObj uintptr) []string {
	if pDataObj == 0 {
		return nil
	}
	vtbl := *(*uintptr)(ptrFromWin32(pDataObj))
	fnGetData := *(*uintptr)(ptrFromWin32(vtbl + 3*unsafe.Sizeof(uintptr(0))))
	fEtc := formatEtc{
		cfFormat: cfHDrop,
		ptd:      0,
		dwAspect: dvaspectContent,
		lindex:   -1,
		tymed:    tymedHGlobal,
	}
	var medium stgMedium
	hr, _, _ := syscall.SyscallN(fnGetData, pDataObj, uintptr(unsafe.Pointer(&fEtc)), uintptr(unsafe.Pointer(&medium)))
	runtime.KeepAlive(&fEtc)
	runtime.KeepAlive(&medium)
	if hr != 0 || medium.hGlobal == 0 {
		return nil
	}
	defer func() {
		procReleaseStgMedium.Call(uintptr(unsafe.Pointer(&medium)))
		runtime.KeepAlive(&medium)
	}()

	hDrop := medium.hGlobal
	countRet, _, _ := procDragQueryFileW.Call(hDrop, 0xFFFFFFFF, 0, 0)
	count := int(countRet)
	var files []string
	for i := 0; i < count; i++ {
		lenRet, _, _ := procDragQueryFileW.Call(hDrop, uintptr(i), 0, 0)
		length := int(lenRet)
		if length > 0 {
			buf := make([]uint16, length+1)
			procDragQueryFileW.Call(hDrop, uintptr(i), uintptr(unsafe.Pointer(&buf[0])), uintptr(length+1))
			runtime.KeepAlive(buf)
			files = append(files, windows.UTF16ToString(buf))
		}
	}
	return files
}

func screenToClientPoint(hwnd uintptr, ptPacked uintptr) (x, y int) {
	sx := int32(uint32(ptPacked & 0xFFFFFFFF))
	sy := int32(uint32(ptPacked >> 32))
	pt := point{x: sx, y: sy}
	procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
	runtime.KeepAlive(&pt)
	return int(pt.x), int(pt.y)
}

func dragEnter(this uintptr, pDataObj uintptr, grfKeyState uintptr, pt uintptr, pdwEffect uintptr) uintptr {
	dt := (*dropTarget)(ptrFromWin32(this))
	if dt == nil || dt.w == nil {
		return sOK
	}
	hasFiles := checkDataObjectHasFiles(pDataObj)
	if pdwEffect != 0 {
		effPtr := (*uint32)(ptrFromWin32(pdwEffect))
		if hasFiles {
			*effPtr = dropEffectCopy
		} else {
			*effPtr = dropEffectNone
		}
	}
	x, y := screenToClientPoint(uintptr(dt.w.hwnd), pt)
	dt.w.emit(Event{
		Type:     EventDragEnter,
		X:        x,
		Y:        y,
		HasFiles: hasFiles,
	})
	return sOK
}

func dragOver(this uintptr, grfKeyState uintptr, pt uintptr, pdwEffect uintptr) uintptr {
	dt := (*dropTarget)(ptrFromWin32(this))
	if dt == nil || dt.w == nil {
		return sOK
	}
	if pdwEffect != 0 {
		effPtr := (*uint32)(ptrFromWin32(pdwEffect))
		*effPtr = dropEffectCopy
	}
	x, y := screenToClientPoint(uintptr(dt.w.hwnd), pt)
	dt.w.emit(Event{
		Type:     EventDragOver,
		X:        x,
		Y:        y,
		HasFiles: true,
	})
	return sOK
}

func dragLeave(this uintptr) uintptr {
	dt := (*dropTarget)(ptrFromWin32(this))
	if dt == nil || dt.w == nil {
		return sOK
	}
	dt.w.emit(Event{
		Type: EventDragLeave,
	})
	return sOK
}

func drop(this uintptr, pDataObj uintptr, grfKeyState uintptr, pt uintptr, pdwEffect uintptr) uintptr {
	dt := (*dropTarget)(ptrFromWin32(this))
	if dt == nil || dt.w == nil {
		return sOK
	}
	if pdwEffect != 0 {
		*(*uint32)(ptrFromWin32(pdwEffect)) = dropEffectCopy
	}
	x, y := screenToClientPoint(uintptr(dt.w.hwnd), pt)
	files := extractFilesFromDataObject(pDataObj)
	if len(files) > 0 {
		dt.w.emit(Event{
			Type:  EventDropFiles,
			X:     x,
			Y:     y,
			Files: files,
		})
	} else {
		dt.w.emit(Event{
			Type: EventDragLeave,
		})
	}
	return sOK
}

func initGlobalDropTargetVtbl() {
	initVtblOnce.Do(func() {
		globalDropTargetVtbl = dropTargetVtbl{
			QueryInterface: syscall.NewCallback(queryInterface),
			AddRef:         syscall.NewCallback(addRef),
			Release:        syscall.NewCallback(release),
			DragEnter:      syscall.NewCallback(dragEnter),
			DragOver:       syscall.NewCallback(dragOver),
			DragLeave:      syscall.NewCallback(dragLeave),
			Drop:           syscall.NewCallback(drop),
		}
	})
}

func (w *win32Window) setupDragDrop() {
	if !w.cfg.AcceptDropFiles {
		return
	}

	// 1. Изменяем фильтр сообщений окон для обхода ограничений UAC (UIPI) при запуске под администратором
	if procChangeWindowMessageFilterEx.Find() == nil {
		procChangeWindowMessageFilterEx.Call(uintptr(w.hwnd), wmDropFiles, msgfltAllow, 0)
		procChangeWindowMessageFilterEx.Call(uintptr(w.hwnd), wmCopyGlobalData, msgfltAllow, 0)
	}

	// 2. Классический DragAcceptFiles для обратной совместимости
	procDragAcceptFiles.Call(uintptr(w.hwnd), 1)

	// 3. OLE Drag & Drop (IDropTarget)
	oleInitOnce.Do(func() {
		procOleInitialize.Call(0)
	})
	initGlobalDropTargetVtbl()

	dt := &dropTarget{
		vtbl: &globalDropTargetVtbl,
		w:    w,
	}
	hr, _, _ := procRegisterDragDrop.Call(uintptr(w.hwnd), uintptr(unsafe.Pointer(dt)))
	if hr == 0 {
		w.dropTarget = dt
		// OLE зарегистрирован успешно: отключаем устаревший DragAcceptFiles,
		// чтобы Проводник не уходил в устаревший протокол WM_DROPFILES.
		procDragAcceptFiles.Call(uintptr(w.hwnd), 0)
	}
}

func (w *win32Window) revokeDropTarget() {
	if w.dropTarget != nil {
		procRevokeDragDrop.Call(uintptr(w.hwnd))
		w.dropTarget = nil
	}
}
