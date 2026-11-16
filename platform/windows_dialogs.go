//go:build windows

package platform

import (
	"runtime"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modComdlg32 = windows.NewLazySystemDLL("comdlg32.dll")
	modOle32    = windows.NewLazySystemDLL("ole32.dll")

	procGetOpenFileNameW     = modComdlg32.NewProc("GetOpenFileNameW")
	procGetSaveFileNameW     = modComdlg32.NewProc("GetSaveFileNameW")
	procSHBrowseForFolderW   = modShell32.NewProc("SHBrowseForFolderW")
	procSHGetPathFromIDListW = modShell32.NewProc("SHGetPathFromIDListW")
	procCoTaskMemFree        = modOle32.NewProc("CoTaskMemFree")
	procCoInitializeEx       = modOle32.NewProc("CoInitializeEx")
	procCoUninitialize       = modOle32.NewProc("CoUninitialize")
	procMessageBoxW          = modUser32.NewProc("MessageBoxW")
)

const (
	ofnAllowMultiSelect = 0x00000200
	ofnPathMustExist    = 0x00000800
	ofnFileMustExist    = 0x00001000
	ofnOverwritePrompt  = 0x00000002
	ofnExplorer         = 0x00080000
	ofnNoChangeDir      = 0x00000008

	bifReturnOnlyFSDirs = 0x00000001
	bifNewDialogStyle   = 0x00000040

	mbOK          = 0x00000000
	mbOKCancel    = 0x00000001
	mbYesNoCancel = 0x00000003
	mbYesNo       = 0x00000004

	mbIconError       = 0x00000010
	mbIconQuestion    = 0x00000020
	mbIconWarning     = 0x00000030
	mbIconInformation = 0x00000040

	idOK     = 1
	idCancel = 2
	idYes    = 6
	idNo     = 7
)

type openFileNameW struct {
	lStructSize       uint32
	hwndOwner         windows.Handle
	hInstance         windows.Handle
	lpstrFilter       *uint16
	lpstrCustomFilter *uint16
	nMaxCustFilter    uint32
	nFilterIndex      uint32
	lpstrFile         *uint16
	nMaxFile          uint32
	lpstrFileTitle    *uint16
	nMaxFileTitle     uint32
	lpstrInitialDir   *uint16
	lpstrTitle        *uint16
	flags             uint32
	nFileOffset       uint16
	nFileExtension    uint16
	lpstrDefExt       *uint16
	lCustData         uintptr
	lpfnHook          uintptr
	lpTemplateName    *uint16
	pvReserved        unsafe.Pointer
	dwReserved        uint32
	flagsEx           uint32
}

type browseInfoW struct {
	hwndOwner      windows.Handle
	pidlRoot       uintptr
	pszDisplayName *uint16
	lpszTitle      *uint16
	ulFlags        uint32
	lpfn           uintptr
	lParam         uintptr
	iImage         int32
}

// BuildWin32Filter формирует double-null-terminated UTF-16 буфер фильтров для GetOpenFileNameW.
func BuildWin32Filter(filters []FileFilter) []uint16 {
	if len(filters) == 0 {
		return nil
	}
	var sb strings.Builder
	for _, f := range filters {
		name := f.Name
		pat := f.Pattern
		if name == "" {
			name = pat
		}
		if pat == "" {
			pat = "*.*"
		}
		sb.WriteString(name)
		sb.WriteByte(0)
		sb.WriteString(pat)
		sb.WriteByte(0)
	}
	sb.WriteByte(0)
	return utf16.Encode([]rune(sb.String()))
}

// ParseOpenFileNames разбирает результат GetOpenFileNameW при одиночном или множественном выборе.
func ParseOpenFileNames(buf []uint16) []string {
	if len(buf) == 0 || buf[0] == 0 {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i < len(buf); i++ {
		if buf[i] == 0 {
			if i == start {
				break
			}
			parts = append(parts, string(utf16.Decode(buf[start:i])))
			start = i + 1
		}
	}
	if len(parts) <= 1 {
		return parts
	}
	dir := parts[0]
	sep := `\`
	if strings.HasSuffix(dir, `\`) || strings.HasSuffix(dir, `/`) {
		sep = ""
	}
	res := make([]string, 0, len(parts)-1)
	for _, name := range parts[1:] {
		res = append(res, dir+sep+name)
	}
	return res
}

func ownerHandle(owner Window) windows.Handle {
	if owner == nil {
		return 0
	}
	return windows.Handle(owner.NativeHandle())
}

// OpenFileDialog открывает системный диалог выбора одного файла.
func OpenFileDialog(opts FileDialogOptions) (string, bool, error) {
	opts.AllowMultiple = false
	files, ok, err := OpenFilesDialog(opts)
	if err != nil || !ok || len(files) == 0 {
		return "", ok, err
	}
	return files[0], true, nil
}

func initCOMSTA() func() {
	// COINIT_APARTMENTTHREADED (0x2) | COINIT_DISABLE_OLE1DDE (0x4) для стабильной работы Shell/Explorer диалогов
	hr, _, _ := procCoInitializeEx.Call(0, 0x2|0x4)
	if hr == 0 || hr == 1 {
		return func() {
			procCoUninitialize.Call()
		}
	}
	return func() {}
}

// OpenFilesDialog открывает системный диалог выбора одного или нескольких файлов.
func OpenFilesDialog(opts FileDialogOptions) ([]string, bool, error) {
	defer initCOMSTA()()
	bufSize := 4096
	if opts.AllowMultiple {
		bufSize = 65536
	}
	fileBuf := make([]uint16, bufSize)
	if opts.DefaultName != "" {
		enc := utf16.Encode([]rune(opts.DefaultName))
		n := len(enc)
		if n >= len(fileBuf) {
			n = len(fileBuf) - 1
		}
		copy(fileBuf, enc[:n])
	}

	var ofn openFileNameW
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	ofn.hwndOwner = ownerHandle(opts.Owner)
	ofn.lpstrFile = &fileBuf[0]
	ofn.nMaxFile = uint32(len(fileBuf))
	ofn.flags = ofnExplorer | ofnPathMustExist | ofnFileMustExist | ofnNoChangeDir
	if opts.AllowMultiple {
		ofn.flags |= ofnAllowMultiSelect
	}

	filterBuf := BuildWin32Filter(opts.Filters)
	if len(filterBuf) > 0 {
		ofn.lpstrFilter = &filterBuf[0]
		ofn.nFilterIndex = 1
	}

	var titlePtr *uint16
	if opts.Title != "" {
		titlePtr, _ = windows.UTF16PtrFromString(opts.Title)
		ofn.lpstrTitle = titlePtr
	}
	var dirPtr *uint16
	if opts.InitialDir != "" {
		dirPtr, _ = windows.UTF16PtrFromString(opts.InitialDir)
		ofn.lpstrInitialDir = dirPtr
	}

	r, _, _ := procGetOpenFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(&ofn)
	runtime.KeepAlive(fileBuf)
	runtime.KeepAlive(filterBuf)
	runtime.KeepAlive(titlePtr)
	runtime.KeepAlive(dirPtr)

	if r == 0 {
		return nil, false, nil
	}
	return ParseOpenFileNames(fileBuf), true, nil
}

// SaveFileDialog открывает системный диалог сохранения файла.
func SaveFileDialog(opts FileDialogOptions) (string, bool, error) {
	defer initCOMSTA()()
	fileBuf := make([]uint16, 4096)
	if opts.DefaultName != "" {
		enc := utf16.Encode([]rune(opts.DefaultName))
		n := len(enc)
		if n >= len(fileBuf) {
			n = len(fileBuf) - 1
		}
		copy(fileBuf, enc[:n])
	}

	var ofn openFileNameW
	ofn.lStructSize = uint32(unsafe.Sizeof(ofn))
	ofn.hwndOwner = ownerHandle(opts.Owner)
	ofn.lpstrFile = &fileBuf[0]
	ofn.nMaxFile = uint32(len(fileBuf))
	ofn.flags = ofnExplorer | ofnPathMustExist | ofnOverwritePrompt | ofnNoChangeDir

	filterBuf := BuildWin32Filter(opts.Filters)
	if len(filterBuf) > 0 {
		ofn.lpstrFilter = &filterBuf[0]
		ofn.nFilterIndex = 1
	}

	var titlePtr *uint16
	if opts.Title != "" {
		titlePtr, _ = windows.UTF16PtrFromString(opts.Title)
		ofn.lpstrTitle = titlePtr
	}
	var dirPtr *uint16
	if opts.InitialDir != "" {
		dirPtr, _ = windows.UTF16PtrFromString(opts.InitialDir)
		ofn.lpstrInitialDir = dirPtr
	}

	r, _, _ := procGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	runtime.KeepAlive(&ofn)
	runtime.KeepAlive(fileBuf)
	runtime.KeepAlive(filterBuf)
	runtime.KeepAlive(titlePtr)
	runtime.KeepAlive(dirPtr)

	if r == 0 {
		return "", false, nil
	}
	return windows.UTF16ToString(fileBuf), true, nil
}

// SelectFolderDialog открывает системный диалог выбора каталога.
func SelectFolderDialog(opts FolderDialogOptions) (string, bool, error) {
	defer initCOMSTA()()
	displayBuf := make([]uint16, 4096)
	var titlePtr *uint16
	if opts.Title != "" {
		titlePtr, _ = windows.UTF16PtrFromString(opts.Title)
	}

	bi := browseInfoW{
		hwndOwner:      ownerHandle(opts.Owner),
		pszDisplayName: &displayBuf[0],
		lpszTitle:      titlePtr,
		ulFlags:        bifReturnOnlyFSDirs | bifNewDialogStyle,
	}

	pidl, _, _ := procSHBrowseForFolderW.Call(uintptr(unsafe.Pointer(&bi)))
	runtime.KeepAlive(&bi)
	runtime.KeepAlive(displayBuf)
	runtime.KeepAlive(titlePtr)

	if pidl == 0 {
		return "", false, nil
	}
	defer procCoTaskMemFree.Call(pidl)

	pathBuf := make([]uint16, 4096)
	r, _, _ := procSHGetPathFromIDListW.Call(pidl, uintptr(unsafe.Pointer(&pathBuf[0])))
	runtime.KeepAlive(pathBuf)
	if r == 0 {
		return "", false, nil
	}
	return windows.UTF16ToString(pathBuf), true, nil
}

// MessageBox отображает системное диалоговое окно с сообщением.
func MessageBox(owner Window, title, text string, style MessageBoxStyle, buttons MessageBoxButtons) MessageBoxResult {
	var flags uintptr
	switch style {
	case MsgWarning:
		flags |= mbIconWarning
	case MsgError:
		flags |= mbIconError
	case MsgQuestion:
		flags |= mbIconQuestion
	default:
		flags |= mbIconInformation
	}

	switch buttons {
	case MsgOKCancel:
		flags |= mbOKCancel
	case MsgYesNo:
		flags |= mbYesNo
	case MsgYesNoCancel:
		flags |= mbYesNoCancel
	default:
		flags |= mbOK
	}

	titlePtr, _ := windows.UTF16PtrFromString(title)
	textPtr, _ := windows.UTF16PtrFromString(text)
	r, _, _ := procMessageBoxW.Call(
		uintptr(ownerHandle(owner)),
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		flags,
	)
	runtime.KeepAlive(titlePtr)
	runtime.KeepAlive(textPtr)

	switch r {
	case idOK:
		return ResOK
	case idCancel:
		return ResCancel
	case idYes:
		return ResYes
	case idNo:
		return ResNo
	default:
		return ResNone
	}
}

