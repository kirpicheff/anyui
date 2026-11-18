//go:build windows

package platform

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modGdi32          = windows.NewLazySystemDLL("gdi32.dll")
	procStretchDIBits = modGdi32.NewProc("StretchDIBits")
	procGetDeviceCaps = modGdi32.NewProc("GetDeviceCaps")
)

const (
	biRGB         = 0
	dibRGBColors  = 0
	srccopy       = 0x00CC0020
	logPixelsX    = 88
)

// bitmapInfoHeader соответствует структуре Win32 BITMAPINFOHEADER.
type bitmapInfoHeader struct {
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

// bitmapInfo соответствует структуре Win32 BITMAPINFO (без палитры).
type bitmapInfo struct {
	bmiHeader bitmapInfoHeader
}

// blitBGRA выводит BGRA-буфер (top-down) в указанный контекст устройства HDC.
func blitBGRA(hdc windows.Handle, dstW, dstH int, pixels []byte, srcW, srcH int) {
	if hdc == 0 || srcW <= 0 || srcH <= 0 || dstW <= 0 || dstH <= 0 {
		return
	}
	expected := srcW * srcH * 4
	if len(pixels) < expected {
		return
	}

	bmi := bitmapInfo{
		bmiHeader: bitmapInfoHeader{
			biSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			biWidth:       int32(srcW),
			biHeight:      -int32(srcH), // Отрицательная высота задаёт порядок строк сверху вниз
			biPlanes:      1,
			biBitCount:    32,
			biCompression: biRGB,
		},
	}

	procStretchDIBits.Call(
		uintptr(hdc),
		0,
		0,
		uintptr(dstW),
		uintptr(dstH),
		0,
		0,
		uintptr(srcW),
		uintptr(srcH),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bmi)),
		dibRGBColors,
		srccopy,
	)

	// Удерживаем буферы живыми на время системного вызова
	runtime.KeepAlive(pixels)
	runtime.KeepAlive(&bmi)
}

// blitBGRARect выводит только заданный прямоугольник (rx, ry, rw, rh) из top-down BGRA-буфера в HDC.
func blitBGRARect(hdc windows.Handle, rx, ry, rw, rh int, pixels []byte, srcW, srcH int) {
	if hdc == 0 || srcW <= 0 || srcH <= 0 || rw <= 0 || rh <= 0 {
		return
	}
	if rx < 0 {
		rw += rx
		rx = 0
	}
	if ry < 0 {
		rh += ry
		ry = 0
	}
	if rx+rw > srcW {
		rw = srcW - rx
	}
	if ry+rh > srcH {
		rh = srcH - ry
	}
	if rw <= 0 || rh <= 0 || len(pixels) < srcW*srcH*4 {
		return
	}

	bmi := bitmapInfo{
		bmiHeader: bitmapInfoHeader{
			biSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
			biWidth:       int32(srcW),
			biHeight:      -int32(srcH),
			biPlanes:      1,
			biBitCount:    32,
			biCompression: biRGB,
		},
	}

	// В StretchDIBits координата ySrc всегда отсчитывается от нижнего края DIB, даже при biHeight < 0
	ySrc := srcH - (ry + rh)

	procStretchDIBits.Call(
		uintptr(hdc),
		uintptr(rx),
		uintptr(ry),
		uintptr(rw),
		uintptr(rh),
		uintptr(rx),
		uintptr(ySrc),
		uintptr(rw),
		uintptr(rh),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&bmi)),
		dibRGBColors,
		srccopy,
	)

	// Удерживаем буферы живыми на время системного вызова
	runtime.KeepAlive(pixels)
	runtime.KeepAlive(&bmi)
}

// getSystemDPI возвращает системный DPI экрана через GDI (фолбэк для старых ОС).
func getSystemDPI(hdc windows.Handle) int {
	if hdc == 0 {
		return 96
	}
	dpi, _, _ := procGetDeviceCaps.Call(uintptr(hdc), logPixelsX)
	if dpi == 0 {
		return 96
	}
	return int(dpi)
}

