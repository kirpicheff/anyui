//go:build windows

package platform

import (
	"encoding/binary"
	"errors"
	"image"
	"image/draw"
	"runtime"
	"unsafe"
)

const (
	cfDIB         = 8
	biBitfields   = 3
	dibHeaderSize = 40
)

// ClipboardImage считывает изображение из буфера обмена (формат CF_DIB).
func (w *win32Window) ClipboardImage() (image.Image, error) {
	r, _, _ := procOpenClipboard.Call(uintptr(w.hwnd))
	if r == 0 {
		return nil, errors.New("не удалось открыть буфер обмена")
	}
	defer procCloseClipboard.Call()

	hData, _, _ := procGetClipboardData.Call(cfDIB)
	if hData == 0 {
		return nil, errors.New("в буфере обмена нет изображения")
	}
	size, _, _ := procGlobalSize.Call(hData)
	if size < dibHeaderSize {
		return nil, errors.New("некорректный размер данных DIB")
	}
	ptr, _, _ := procGlobalLock.Call(hData)
	if ptr == 0 {
		return nil, errors.New("не удалось заблокировать память буфера обмена")
	}
	defer procGlobalUnlock.Call(hData)

	raw := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)
	buf := make([]byte, len(raw))
	copy(buf, raw)
	return decodeDIBToImage(buf)
}

// SetClipboardImage записывает изображение в буфер обмена в формате CF_DIB (32-bit BGRA).
func (w *win32Window) SetClipboardImage(img image.Image) error {
	dib, err := encodeImageToDIB(img)
	if err != nil {
		return err
	}

	hMem, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(len(dib)))
	if hMem == 0 {
		return errors.New("ошибка выделения памяти для изображения в буфере обмена")
	}
	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return errors.New("ошибка блокировки памяти буфера обмена")
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(dib))
	copy(dst, dib)
	procGlobalUnlock.Call(hMem)

	r, _, _ := procOpenClipboard.Call(uintptr(w.hwnd))
	if r == 0 {
		return errors.New("не удалось открыть буфер обмена")
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	res, _, _ := procSetClipboardData.Call(cfDIB, hMem)
	runtime.KeepAlive(dib)
	if res == 0 {
		return errors.New("не удалось записать изображение в буфер обмена")
	}
	return nil
}

func encodeImageToDIB(img image.Image) ([]byte, error) {
	if img == nil {
		return nil, errors.New("изображение не задано")
	}
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("некорректные размеры изображения")
	}

	nrgba := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(nrgba, nrgba.Bounds(), img, b.Min, draw.Src)

	pixelBytes := w * h * 4
	out := make([]byte, dibHeaderSize+pixelBytes)

	binary.LittleEndian.PutUint32(out[0:4], dibHeaderSize)
	binary.LittleEndian.PutUint32(out[4:8], uint32(int32(w)))
	binary.LittleEndian.PutUint32(out[8:12], uint32(int32(h))) // bottom-up DIB для максимальной совместимости
	binary.LittleEndian.PutUint16(out[12:14], 1)               // biPlanes
	binary.LittleEndian.PutUint16(out[14:16], 32)              // biBitCount
	binary.LittleEndian.PutUint32(out[16:20], biRGB)           // biCompression
	binary.LittleEndian.PutUint32(out[20:24], uint32(pixelBytes))

	dst := out[dibHeaderSize:]
	for y := 0; y < h; y++ {
		srcY := h - 1 - y
		srcRow := nrgba.Pix[srcY*nrgba.Stride:]
		dstRow := dst[y*w*4:]
		for x := 0; x < w; x++ {
			so := x * 4
			do := x * 4
			dstRow[do+0] = srcRow[so+2] // B
			dstRow[do+1] = srcRow[so+1] // G
			dstRow[do+2] = srcRow[so+0] // R
			dstRow[do+3] = srcRow[so+3] // A
		}
	}
	return out, nil
}

func decodeDIBToImage(dib []byte) (image.Image, error) {
	if len(dib) < dibHeaderSize {
		return nil, errors.New("буфер DIB слишком мал")
	}
	headerSize := int(binary.LittleEndian.Uint32(dib[0:4]))
	if headerSize < dibHeaderSize || headerSize > len(dib) {
		return nil, errors.New("некорректный заголовок DIB")
	}
	w := int(int32(binary.LittleEndian.Uint32(dib[4:8])))
	rawH := int(int32(binary.LittleEndian.Uint32(dib[8:12])))
	bpp := int(binary.LittleEndian.Uint16(dib[14:16]))
	comp := binary.LittleEndian.Uint32(dib[16:20])

	topDown := false
	h := rawH
	if h < 0 {
		topDown = true
		h = -h
	}
	if w <= 0 || h <= 0 {
		return nil, errors.New("некорректные размеры DIB")
	}
	if bpp != 32 && bpp != 24 {
		return nil, errors.New("поддерживаются только 24-битные и 32-битные DIB изображения")
	}

	pixOffset := headerSize
	if comp == biBitfields && headerSize == dibHeaderSize {
		pixOffset += 12 // 3 цветовые маски по 4 байта
	} else if comp != biRGB && comp != biBitfields {
		return nil, errors.New("неподдерживаемое сжатие DIB")
	}

	rowStride := ((w*bpp + 31) / 32) * 4
	if pixOffset+rowStride*h > len(dib) {
		return nil, errors.New("данные пикселей DIB усечены")
	}

	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	pix := dib[pixOffset:]
	bytesPerPx := bpp / 8

	for y := 0; y < h; y++ {
		srcY := y
		if !topDown {
			srcY = h - 1 - y
		}
		srcRow := pix[srcY*rowStride:]
		dstRow := out.Pix[y*out.Stride:]
		for x := 0; x < w; x++ {
			so := x * bytesPerPx
			do := x * 4
			dstRow[do+0] = srcRow[so+2] // R
			dstRow[do+1] = srcRow[so+1] // G
			dstRow[do+2] = srcRow[so+0] // B
			if bpp == 32 {
				a := srcRow[so+3]
				// Многие приложения в Windows кладут 0 в альфа-канал 32-bit DIB;
				// если альфа полностью 0 у всего кадра, ниже можно при необходимости сделать непрозрачным,
				// но побайтно сохраняем как есть, если альфа используется.
				dstRow[do+3] = a
			} else {
				dstRow[do+3] = 255
			}
		}
	}

	// Если 32-битный DIB пришёл без альфа-канала (все байты A == 0), делаем изображение непрозрачным
	if bpp == 32 {
		hasAlpha := false
		for i := 3; i < len(out.Pix); i += 4 {
			if out.Pix[i] != 0 {
				hasAlpha = true
				break
			}
		}
		if !hasAlpha {
			for i := 3; i < len(out.Pix); i += 4 {
				out.Pix[i] = 255
			}
		}
	}

	return out, nil
}
