//go:build windows

package platform

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

const (
	dwmwaBorderColor  = 34
	dwmwaCaptionColor = 35
	dwmwaTextColor    = 36
	dwmwaColorDefault = 0xFFFFFFFF

	wmSettingChange = 0x001A
)

// IsSystemDarkTheme определяет, включена ли в ОС тёмная тема приложений.
// На старых версиях Windows без ключа Personalize возвращает false.
func IsSystemDarkTheme() bool {
	k, err := registry.OpenKey(
		registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer k.Close()

	val, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil {
		return false
	}
	return val == 0
}

// colorToCOLORREF конвертирует RGB в формат Win32 COLORREF (0x00BBGGRR).
func colorToCOLORREF(c WindowColor) uint32 {
	return uint32(c.R) | (uint32(c.G) << 8) | (uint32(c.B) << 16)
}

// SetDarkTitle динамически включает или выключает тёмный режим системной шапки окна (DWM).
func (w *win32Window) SetDarkTitle(dark bool) {
	if w.hwnd == 0 {
		return
	}
	if err := procDwmSetWindowAttribute.Find(); err != nil {
		return
	}
	var flag int32
	if dark {
		flag = 1
	}
	r, _, _ := procDwmSetWindowAttribute.Call(
		uintptr(w.hwnd),
		dwmwaUseImmersiveDarkMode,
		uintptr(unsafe.Pointer(&flag)),
		unsafe.Sizeof(flag),
	)
	if r != 0 {
		procDwmSetWindowAttribute.Call(
			uintptr(w.hwnd),
			dwmwaUseImmersiveDarkModeOld,
			uintptr(unsafe.Pointer(&flag)),
			unsafe.Sizeof(flag),
		)
	}
	runtime.KeepAlive(&flag)
}

// SetTitleBarColors задаёт произвольные цвета системного заголовка, текста шапки и рамки окна (Windows 11+).
// Передача nil сбрасывает соответствующий цвет на системный по умолчанию.
func (w *win32Window) SetTitleBarColors(caption, text, border *WindowColor) {
	if w.hwnd == 0 {
		return
	}
	if err := procDwmSetWindowAttribute.Find(); err != nil {
		return
	}
	w.applyDwmColor(dwmwaCaptionColor, caption, true)
	w.applyDwmColor(dwmwaTextColor, text, true)
	w.applyDwmColor(dwmwaBorderColor, border, true)
}

func (w *win32Window) applyDwmColor(attr uintptr, c *WindowColor, resetOnNil bool) {
	if c == nil && !resetOnNil {
		return
	}
	var val uint32 = dwmwaColorDefault
	if c != nil {
		val = colorToCOLORREF(*c)
	}
	procDwmSetWindowAttribute.Call(
		uintptr(w.hwnd),
		attr,
		uintptr(unsafe.Pointer(&val)),
		unsafe.Sizeof(val),
	)
	runtime.KeepAlive(&val)
}

func (w *win32Window) applyInitialWindowTheme() {
	if w.cfg.DarkTitle {
		w.SetDarkTitle(true)
	}
	if err := procDwmSetWindowAttribute.Find(); err != nil {
		return
	}
	w.applyDwmColor(dwmwaCaptionColor, w.cfg.TitleBarColor, false)
	w.applyDwmColor(dwmwaTextColor, w.cfg.TitleTextColor, false)
	w.applyDwmColor(dwmwaBorderColor, w.cfg.BorderColor, false)
}

