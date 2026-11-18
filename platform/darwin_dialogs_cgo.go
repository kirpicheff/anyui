//go:build darwin && cgo

package platform

/*
#include <stdlib.h>
#include "darwin_bridge.h"
*/
import "C"

import (
	"strings"
	"unsafe"
)

// OpenFileDialog открывает нативный диалог выбора одного файла (NSOpenPanel).
func OpenFileDialog(opts FileDialogOptions) (string, bool, error) {
	opts.AllowMultiple = false
	files, ok, err := runDarwinNativeOpenPanel(opts, false)
	if err != nil || !ok || len(files) == 0 {
		return "", false, err
	}
	return files[0], true, nil
}

// OpenFilesDialog открывает нативный диалог выбора нескольких файлов (NSOpenPanel).
func OpenFilesDialog(opts FileDialogOptions) ([]string, bool, error) {
	opts.AllowMultiple = true
	return runDarwinNativeOpenPanel(opts, false)
}

// SaveFileDialog открывает нативный диалог сохранения файла (NSSavePanel).
func SaveFileDialog(opts FileDialogOptions) (string, bool, error) {
	_ = opts.Owner
	cTitle := C.CString(opts.Title)
	cDir := C.CString(opts.InitialDir)
	cName := C.CString(opts.DefaultName)
	cExts := C.CString(strings.Join(extractDarwinFileExtensions(opts.Filters), ","))
	defer func() {
		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cDir))
		C.free(unsafe.Pointer(cName))
		C.free(unsafe.Pointer(cExts))
	}()

	out := C.anyui_darwin_save_file_dialog(cTitle, cDir, cName, cExts)
	if out == nil {
		return "", false, nil
	}
	defer C.free(unsafe.Pointer(out))

	path := strings.TrimSpace(C.GoString(out))
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}

// SelectFolderDialog открывает нативный диалог выбора папки (NSOpenPanel).
func SelectFolderDialog(opts FolderDialogOptions) (string, bool, error) {
	fileOpts := FileDialogOptions{
		Title:      opts.Title,
		InitialDir: opts.InitialDir,
		Owner:      opts.Owner,
	}
	files, ok, err := runDarwinNativeOpenPanel(fileOpts, true)
	if err != nil || !ok || len(files) == 0 {
		return "", false, err
	}
	return files[0], true, nil
}

// MessageBox отображает нативное диалоговое окно (NSAlert).
func MessageBox(owner Window, title, text string, style MessageBoxStyle, buttons MessageBoxButtons) MessageBoxResult {
	_ = owner
	cTitle := C.CString(title)
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cText))

	res := C.anyui_darwin_message_box(cTitle, cText, C.int(style), C.int(buttons))
	return MessageBoxResult(res)
}

func runDarwinNativeOpenPanel(opts FileDialogOptions, chooseDirs bool) ([]string, bool, error) {
	_ = opts.Owner
	cTitle := C.CString(opts.Title)
	cDir := C.CString(opts.InitialDir)
	var extsCSV string
	if !chooseDirs {
		extsCSV = strings.Join(extractDarwinFileExtensions(opts.Filters), ",")
	}
	cExts := C.CString(extsCSV)
	defer func() {
		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cDir))
		C.free(unsafe.Pointer(cExts))
	}()

	multi := C.int(0)
	if opts.AllowMultiple && !chooseDirs {
		multi = 1
	}
	dirs := C.int(0)
	if chooseDirs {
		dirs = 1
	}

	out := C.anyui_darwin_open_file_dialog(cTitle, cDir, cExts, multi, dirs)
	if out == nil {
		return nil, false, nil
	}
	defer C.free(unsafe.Pointer(out))

	raw := strings.TrimSpace(C.GoString(out))
	if raw == "" {
		return nil, false, nil
	}
	var res []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			res = append(res, line)
		}
	}
	return res, len(res) > 0, nil
}

