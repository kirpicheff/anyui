//go:build windows

package platform

import (
	"errors"
	"hash/fnv"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// SetCustomCursor устанавливает нативный 32-битный HCURSOR из BGRA-буфера.
// Передача nil или некорректных размеров сбрасывает кастомный курсор и возвращает обычный CursorType.
func (w *win32Window) SetCustomCursor(cur *CustomCursor) {
	if cur == nil || cur.Width <= 0 || cur.Height <= 0 || len(cur.BGRA) < cur.Width*cur.Height*4 {
		w.clearCustomCursor(true)
		return
	}

	byteLen := cur.Width * cur.Height * 4
	digest := customCursorDigest(cur.BGRA[:byteLen])

	if w.customHCursor != 0 &&
		w.customCursorHash == digest &&
		w.customCursorW == cur.Width &&
		w.customCursorH == cur.Height &&
		w.customCursorHotX == cur.HotX &&
		w.customCursorHotY == cur.HotY {
		w.applyCursor()
		return
	}

	hCursor, err := createHCursorFromBGRA(cur)
	if err != nil || hCursor == 0 {
		w.clearCustomCursor(true)
		return
	}

	old := w.customHCursor
	w.customHCursor = hCursor
	w.customCursorHash = digest
	w.customCursorW = cur.Width
	w.customCursorH = cur.Height
	w.customCursorHotX = cur.HotX
	w.customCursorHotY = cur.HotY

	// Сначала активируем новый курсор, затем освобождаем старый HCURSOR
	w.applyCursor()
	if old != 0 {
		procDestroyIcon.Call(uintptr(old))
	}
}

func (w *win32Window) clearCustomCursor(reapply bool) {
	old := w.customHCursor
	w.customHCursor = 0
	w.customCursorHash = 0
	w.customCursorW = 0
	w.customCursorH = 0
	w.customCursorHotX = 0
	w.customCursorHotY = 0
	if reapply {
		w.applyCursor()
	}
	if old != 0 {
		procDestroyIcon.Call(uintptr(old))
	}
}

func customCursorDigest(bgra []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(bgra)
	return h.Sum64()
}

func premultiplyBGRA(dst, src []byte) {
	n := len(src)
	if len(dst) < n {
		n = len(dst)
	}
	n &^= 3
	for i := 0; i < n; i += 4 {
		a := src[i+3]
		switch a {
		case 0:
			dst[i+0] = 0
			dst[i+1] = 0
			dst[i+2] = 0
			dst[i+3] = 0
		case 255:
			dst[i+0] = src[i+0]
			dst[i+1] = src[i+1]
			dst[i+2] = src[i+2]
			dst[i+3] = 255
		default:
			aa := uint32(a)
			dst[i+0] = uint8((uint32(src[i+0])*aa + 127) / 255)
			dst[i+1] = uint8((uint32(src[i+1])*aa + 127) / 255)
			dst[i+2] = uint8((uint32(src[i+2])*aa + 127) / 255)
			dst[i+3] = a
		}
	}
}

// createHCursorFromBGRA создаёт нативный 32-битный HCURSOR с альфа-каналом
// через CreateDIBSection (top-down BGRA) + CreateBitmap (маска) + CreateIconIndirect.
func createHCursorFromBGRA(cur *CustomCursor) (windows.Handle, error) {
	if cur == nil || cur.Width <= 0 || cur.Height <= 0 || len(cur.BGRA) < cur.Width*cur.Height*4 {
		return 0, errors.New("некорректные параметры кастомного курсора")
	}

	bi := bitmapInfoHeaderTray{
		biSize:        uint32(unsafe.Sizeof(bitmapInfoHeaderTray{})),
		biWidth:       int32(cur.Width),
		biHeight:      -int32(cur.Height), // top-down DIB
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
	runtime.KeepAlive(&bits)
	if hColor == 0 || bits == nil {
		return 0, errors.New("не удалось создать DIB для кастомного курсора")
	}
	defer procDeleteObjectTray.Call(hColor)

	byteLen := cur.Width * cur.Height * 4
	dst := unsafe.Slice((*byte)(bits), byteLen)
	premultiplyBGRA(dst, cur.BGRA[:byteLen])
	runtime.KeepAlive(cur.BGRA)

	hMask, _, _ := procCreateBitmap.Call(uintptr(cur.Width), uintptr(cur.Height), 1, 1, 0)
	if hMask == 0 {
		return 0, errors.New("не удалось создать монохромную маску для кастомного курсора")
	}
	defer procDeleteObjectTray.Call(hMask)

	hotX := cur.HotX
	if hotX < 0 {
		hotX = 0
	} else if hotX >= cur.Width {
		hotX = cur.Width - 1
	}
	hotY := cur.HotY
	if hotY < 0 {
		hotY = 0
	} else if hotY >= cur.Height {
		hotY = cur.Height - 1
	}

	ii := iconInfo{
		fIcon:    0, // 0 = HCURSOR
		xHotspot: uint32(hotX),
		yHotspot: uint32(hotY),
		hbmMask:  windows.Handle(hMask),
		hbmColor: windows.Handle(hColor),
	}
	hCursor, _, _ := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	runtime.KeepAlive(&ii)
	if hCursor == 0 {
		return 0, errors.New("ошибка CreateIconIndirect при создании кастомного курсора")
	}
	return windows.Handle(hCursor), nil
}

