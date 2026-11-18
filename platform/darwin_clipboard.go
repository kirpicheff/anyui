//go:build darwin && cgo

package platform

/*
#include <stdlib.h>
#include "darwin_bridge.h"
*/
import "C"

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"unsafe"
)

var _ ClipboardImager = (*darwinWindow)(nil)

// ClipboardText возвращает текст из системного буфера обмена macOS (NSPasteboard).
func (w *darwinWindow) ClipboardText() (string, error) {
	cStr := C.anyui_darwin_clipboard_get_text()
	if cStr == nil {
		return "", nil
	}
	defer C.free(unsafe.Pointer(cStr))
	return C.GoString(cStr), nil
}

// SetClipboardText записывает текст в системный буфер обмена macOS (NSPasteboard).
func (w *darwinWindow) SetClipboardText(text string) error {
	cStr := C.CString(text)
	defer C.free(unsafe.Pointer(cStr))
	if C.anyui_darwin_clipboard_set_text(cStr) == 0 {
		return errors.New("не удалось записать текст в NSPasteboard")
	}
	return nil
}

// ClipboardImage считывает изображение (PNG/TIFF) из буфера обмена macOS.
func (w *darwinWindow) ClipboardImage() (image.Image, error) {
	var outLen C.int
	ptr := C.anyui_darwin_clipboard_get_image(&outLen)
	if ptr == nil || outLen <= 0 {
		return nil, errors.New("в буфере обмена macOS нет изображения")
	}
	defer C.free(ptr)

	data := C.GoBytes(ptr, outLen)
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return img, nil
}

// SetClipboardImage записывает изображение в буфер обмена macOS в формате PNG.
func (w *darwinWindow) SetClipboardImage(img image.Image) error {
	if img == nil {
		return errors.New("передано пустое изображение")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	raw := buf.Bytes()
	if len(raw) == 0 {
		return errors.New("не удалось закодировать изображение в PNG")
	}
	if C.anyui_darwin_clipboard_set_image(unsafe.Pointer(&raw[0]), C.int(len(raw))) == 0 {
		return errors.New("не удалось записать изображение в NSPasteboard")
	}
	return nil
}

