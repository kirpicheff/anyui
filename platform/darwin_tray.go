//go:build darwin && cgo

package platform

/*
#include <stdlib.h>
#include "darwin_bridge.h"
*/
import "C"

import (
	"errors"
	"sync"
	"sync/atomic"
	"unsafe"
)

var (
	darwinNextTrayID atomic.Uint64
	darwinTrayMu     sync.RWMutex
	darwinTrays      = make(map[uint64]*darwinTray)
	darwinNotifyMu   sync.RWMutex
	darwinNotifyCB   func()
)

type darwinTray struct {
	id     uint64
	cfg    TrayConfig
	handle uintptr

	mu        sync.Mutex
	closed    bool
	menuItems []TrayMenuItem
}

var _ Tray = (*darwinTray)(nil)

// NewTray создаёт иконку приложения в строке меню macOS (NSStatusItem).
func NewTray(cfg TrayConfig) (Tray, error) {
	id := darwinNextTrayID.Add(1)
	cTip := C.CString(cfg.Tooltip)
	defer C.free(unsafe.Pointer(cTip))

	handle := uintptr(C.anyui_darwin_tray_create(C.uint64_t(id), cTip))
	if handle == 0 {
		return nil, errors.New("не удалось создать NSStatusItem в строке меню macOS")
	}

	t := &darwinTray{
		id:     id,
		cfg:    cfg,
		handle: handle,
	}

	darwinTrayMu.Lock()
	darwinTrays[id] = t
	darwinTrayMu.Unlock()

	if cfg.OnNotificationClick != nil {
		darwinNotifyMu.Lock()
		darwinNotifyCB = cfg.OnNotificationClick
		darwinNotifyMu.Unlock()
	}

	if len(cfg.IconRGBA) > 0 && cfg.IconWidth > 0 && cfg.IconHeight > 0 {
		_ = t.SetIconRGBA(cfg.IconRGBA, cfg.IconWidth, cfg.IconHeight)
	}
	if len(cfg.Menu) > 0 {
		t.SetMenu(cfg.Menu)
	}
	return t, nil
}

func lookupDarwinTray(id uint64) *darwinTray {
	darwinTrayMu.RLock()
	defer darwinTrayMu.RUnlock()
	return darwinTrays[id]
}

// SetTooltip обновляет всплывающую подсказку иконки трея.
func (t *darwinTray) SetTooltip(tooltip string) error {
	t.mu.Lock()
	handle := t.handle
	closed := t.closed
	t.mu.Unlock()
	if closed || handle == 0 {
		return errors.New("трей закрыт")
	}
	cTip := C.CString(tooltip)
	defer C.free(unsafe.Pointer(cTip))
	C.anyui_darwin_tray_set_tooltip(C.uintptr_t(handle), cTip)
	return nil
}

// SetIconRGBA обновляет монохромную шаблонную иконку трея из RGBA-пикселей.
func (t *darwinTray) SetIconRGBA(rgba []byte, width, height int) error {
	t.mu.Lock()
	handle := t.handle
	closed := t.closed
	t.mu.Unlock()
	if closed || handle == 0 {
		return errors.New("трей закрыт")
	}
	expected := width * height * 4
	if width <= 0 || height <= 0 || len(rgba) < expected {
		return errors.New("некорректный буфер RGBA для иконки трея")
	}
	C.anyui_darwin_tray_set_icon(
		C.uintptr_t(handle),
		unsafe.Pointer(&rgba[0]),
		C.int(width),
		C.int(height),
	)
	return nil
}

// SetMenu задаёт контекстное меню для иконки в строке меню macOS.
func (t *darwinTray) SetMenu(items []TrayMenuItem) {
	t.mu.Lock()
	handle := t.handle
	closed := t.closed
	if !closed && handle != 0 {
		t.menuItems = make([]TrayMenuItem, len(items))
		copy(t.menuItems, items)
	}
	t.mu.Unlock()

	if closed || handle == 0 {
		return
	}

	C.anyui_darwin_tray_clear_menu(C.uintptr_t(handle))
	for i, item := range items {
		sep := C.int(0)
		if item.Separator {
			sep = 1
		}
		chk := C.int(0)
		if item.Checked {
			chk = 1
		}
		dis := C.int(0)
		if item.Disabled {
			dis = 1
		}
		cTitle := C.CString(item.Label)
		C.anyui_darwin_tray_add_menu_item(C.uintptr_t(handle), C.int(i), cTitle, sep, chk, dis)
		C.free(unsafe.Pointer(cTitle))
	}
}

// ShowNotification отображает системное уведомление macOS.
func (t *darwinTray) ShowNotification(n TrayNotification) error {
	t.mu.Lock()
	closed := t.closed
	onClick := t.cfg.OnNotificationClick
	t.mu.Unlock()
	if closed {
		return errors.New("трей закрыт")
	}
	if onClick != nil {
		darwinNotifyMu.Lock()
		darwinNotifyCB = onClick
		darwinNotifyMu.Unlock()
	}
	return sendDarwinNotification(n)
}

// Close удаляет иконку из строки меню macOS.
func (t *darwinTray) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	handle := t.handle
	t.handle = 0
	t.menuItems = nil
	t.mu.Unlock()

	darwinTrayMu.Lock()
	delete(darwinTrays, t.id)
	darwinTrayMu.Unlock()

	if handle != 0 {
		C.anyui_darwin_tray_destroy(C.uintptr_t(handle))
	}
	return nil
}

//export goDarwinOnTrayClick
func goDarwinOnTrayClick(trayID C.uint64_t) {
	t := lookupDarwinTray(uint64(trayID))
	if t == nil {
		return
	}
	t.mu.Lock()
	closed := t.closed
	handle := t.handle
	onClick := t.cfg.OnClick
	hasMenu := len(t.menuItems) > 0
	t.mu.Unlock()
	if closed {
		return
	}
	if onClick != nil {
		onClick()
		return
	}
	if hasMenu && handle != 0 {
		C.anyui_darwin_tray_popup_menu(C.uintptr_t(handle))
	}
}

//export goDarwinOnTrayDoubleClick
func goDarwinOnTrayDoubleClick(trayID C.uint64_t) {
	t := lookupDarwinTray(uint64(trayID))
	if t == nil {
		return
	}
	t.mu.Lock()
	closed := t.closed
	onDouble := t.cfg.OnDoubleClick
	onClick := t.cfg.OnClick
	t.mu.Unlock()
	if closed {
		return
	}
	if onDouble != nil {
		onDouble()
	} else if onClick != nil {
		onClick()
	}
}

//export goDarwinOnTrayRightClick
func goDarwinOnTrayRightClick(trayID C.uint64_t) {
	t := lookupDarwinTray(uint64(trayID))
	if t == nil {
		return
	}
	t.mu.Lock()
	closed := t.closed
	handle := t.handle
	onRight := t.cfg.OnRightClick
	hasMenu := len(t.menuItems) > 0
	t.mu.Unlock()
	if closed {
		return
	}
	if onRight != nil {
		onRight()
		return
	}
	if hasMenu && handle != 0 {
		C.anyui_darwin_tray_popup_menu(C.uintptr_t(handle))
	}
}

//export goDarwinOnTrayMenuItem
func goDarwinOnTrayMenuItem(trayID C.uint64_t, itemID C.int) {
	t := lookupDarwinTray(uint64(trayID))
	if t == nil {
		return
	}
	idx := int(itemID)
	t.mu.Lock()
	closed := t.closed
	var cb func()
	if !closed && idx >= 0 && idx < len(t.menuItems) {
		cb = t.menuItems[idx].OnClick
	}
	t.mu.Unlock()
	if cb != nil {
		cb()
	}
}

//export goDarwinOnNotificationClick
func goDarwinOnNotificationClick() {
	darwinNotifyMu.RLock()
	cb := darwinNotifyCB
	darwinNotifyMu.RUnlock()
	if cb != nil {
		cb()
	}
}

