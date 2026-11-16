package ui

import "anyui/platform"

// KeyTab — виртуальный код клавиши Tab на Windows.
const KeyTab uint32 = 0x09

// Shortcut описывает глобальную горячую клавишу в рамках окна.
type Shortcut struct {
	Code      uint32
	Mods      platform.Modifiers
	OnTrigger func()
}

// RegisterShortcut добавляет горячую клавишу в контекст окна.
func (c *Context) RegisterShortcut(code uint32, mods platform.Modifiers, fn func()) {
	if fn == nil {
		return
	}
	c.shortcuts = append(c.shortcuts, Shortcut{
		Code:      code,
		Mods:      mods,
		OnTrigger: fn,
	})
}

// ClearShortcuts удаляет все зарегистрированные горячие клавиши.
func (c *Context) ClearShortcuts() {
	c.shortcuts = c.shortcuts[:0]
}

// registerFocusable добавляет виджет в порядок обхода фокуса по Tab.
func (c *Context) registerFocusable(id string) {
	if id == "" {
		return
	}
	c.nextFocusOrder = append(c.nextFocusOrder, id)
}

// processFocusAndShortcuts обрабатывает Tab / Shift+Tab, Escape и зарегистрированные горячие клавиши.
func (c *Context) processFocusAndShortcuts() {
	if len(c.KeyEvents) == 0 {
		return
	}

	remaining := c.KeyEvents[:0]
	for _, kp := range c.KeyEvents {
		// Проверяем зарегистрированные горячие клавиши
		triggered := false
		for _, sc := range c.shortcuts {
			if sc.Code == kp.Code && sc.Mods == kp.Mods {
				sc.OnTrigger()
				triggered = true
				c.RequestRedraw()
			}
		}
		if triggered {
			continue
		}

		// Навигация фокуса по Tab / Shift+Tab
		if kp.Code == KeyTab && !kp.Mods.Ctrl && !kp.Mods.Alt {
			c.stepFocus(kp.Mods.Shift)
			c.RequestRedraw()
			continue
		}

		// Сброс фокуса по Escape, если не в оверлее
		if kp.Code == KeyEscape && c.FocusID != "" {
			c.FocusID = ""
			c.RequestRedraw()
			continue
		}

		remaining = append(remaining, kp)
	}
	c.KeyEvents = remaining
}

func (c *Context) stepFocus(reverse bool) {
	n := len(c.focusOrder)
	if n == 0 {
		return
	}

	idx := -1
	for i, id := range c.focusOrder {
		if id == c.FocusID {
			idx = i
			break
		}
	}

	if idx == -1 {
		if reverse {
			c.FocusID = c.focusOrder[n-1]
		} else {
			c.FocusID = c.focusOrder[0]
		}
		return
	}

	if reverse {
		idx = (idx - 1 + n) % n
	} else {
		idx = (idx + 1) % n
	}
	c.FocusID = c.focusOrder[idx]
}

