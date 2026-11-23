package anyui

import (
	"errors"
	"image"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/ui"
)

// Window представляет независимое нативное окно с собственным холстом, фокусом и анимациями.
type Window struct {
	app         *App
	platWin     platform.Window
	ctx         *ui.Context
	ui          *UI
	drawFn      func(*UI)
	hideOnClose bool
	onClose     func(*Window) bool
	onDropFiles func(files []string, xPx, yPx int)
	onDragEnter func(xPx, yPx int, hasFiles bool)
	onDragOver  func(xPx, yPx int, hasFiles bool)
	onDragLeave func()
}

// SetTitle меняет заголовок окна.
func (w *Window) SetTitle(title string) {
	if w.platWin != nil {
		w.platWin.SetTitle(title)
	}
}

// SetCustomCursor задаёт нативный кастомный курсор окна из BGRA-буфера.
// Передача nil или некорректных размеров сбрасывает кастомный курсор к системному CursorType.
func (w *Window) SetCustomCursor(cur *CustomCursor) {
	if w != nil && w.platWin != nil {
		w.platWin.SetCustomCursor(cur)
	}
}

// SetIcon программно устанавливает иконку окна из RGBA-буфера (widthPx * heightPx * 4).
// Передача пустого буфера и нулевых размеров сбрасывает программную иконку к иконке класса окна.
func (w *Window) SetIcon(rgba []byte, widthPx, heightPx int) error {
	if setter, ok := w.platWin.(platform.WindowIconSetter); ok {
		return setter.SetIcon(rgba, widthPx, heightPx)
	}
	return errors.New("установка иконки окна не поддерживается")
}

// Show отображает окно, если оно было скрыто.
func (w *Window) Show() {
	if w.platWin != nil {
		w.platWin.Show()
	}
}

// Hide скрывает окно без его уничтожения.
func (w *Window) Hide() {
	if w.platWin != nil {
		w.platWin.Hide()
	}
}

// Close запрашивает закрытие окна.
func (w *Window) Close() {
	if w.platWin != nil {
		w.platWin.Close()
	}
}

// Invalidate запрашивает перерисовку окна.
func (w *Window) Invalidate() {
	if w.platWin != nil {
		w.platWin.Invalidate()
	}
}

// WakeUp безопасно пробуждает цикл событий конкретного окна из любой горутины
// для выполнения Render() (с поддержкой fast-path Streaming).
func (w *Window) WakeUp() {
	if w != nil && w.platWin != nil {
		w.platWin.WakeUp()
	}
}

// SetTheme устанавливает новую тему оформления для окна, синхронизирует режим заголовка ОС и перерисовывает окно.
func (w *Window) SetTheme(theme Theme) {
	w.applyTheme(theme, true)
}

func (w *Window) applyTheme(theme Theme, invalidate bool) {
	if w.ctx == nil {
		return
	}
	if w.ctx.Theme == theme {
		return
	}
	w.ctx.Theme = theme
	w.ctx.InvalidateStatic()
	w.SetDarkTitle(isThemeDark(theme))
	if invalidate {
		w.Invalidate()
	} else {
		w.ctx.RequestRedraw()
	}
}

// SetDarkTitle динамически включает или выключает тёмный режим системной шапки окна.
func (w *Window) SetDarkTitle(dark bool) {
	if styler, ok := w.platWin.(platform.TitleBarStyler); ok {
		styler.SetDarkTitle(dark)
	}
}

// SetTitleBarColors задаёт произвольные цвета системной шапки, текста заголовка и рамки окна (Windows 11+).
// Передача nil сбрасывает соответствующий цвет к системному значению по умолчанию.
func (w *Window) SetTitleBarColors(caption, text, border *Color) {
	if styler, ok := w.platWin.(platform.TitleBarStyler); ok {
		styler.SetTitleBarColors(
			toPlatformWindowColor(caption),
			toPlatformWindowColor(text),
			toPlatformWindowColor(border),
		)
	}
}

// State возвращает текущую геометрию окна (позиция, размер, Maximized) для сохранения.
func (w *Window) State() WindowState {
	if stater, ok := w.platWin.(platform.WindowStater); ok {
		return stater.State()
	}
	return WindowState{}
}

// RestoreState восстанавливает ранее сохранённую геометрию окна.
func (w *Window) RestoreState(state WindowState) {
	if stater, ok := w.platWin.(platform.WindowStater); ok {
		stater.RestoreState(state)
	}
}

// ClipboardText возвращает текст из системного буфера обмена ОС.
func (w *Window) ClipboardText() (string, error) {
	if w == nil || w.platWin == nil {
		return "", errors.New("окно не инициализировано")
	}
	return w.platWin.ClipboardText()
}

// SetClipboardText записывает текст в системный буфер обмена ОС.
func (w *Window) SetClipboardText(text string) error {
	if w == nil || w.platWin == nil {
		return errors.New("окно не инициализировано")
	}
	return w.platWin.SetClipboardText(text)
}

// ClipboardImage считывает изображение из буфера обмена ОС.
func (w *Window) ClipboardImage() (image.Image, error) {
	if ci, ok := w.platWin.(platform.ClipboardImager); ok {
		return ci.ClipboardImage()
	}
	return nil, errors.New("буфер обмена для изображений не поддерживается")
}

// SetClipboardImage записывает изображение в буфер обмена ОС.
func (w *Window) SetClipboardImage(img image.Image) error {
	if ci, ok := w.platWin.(platform.ClipboardImager); ok {
		return ci.SetClipboardImage(img)
	}
	return errors.New("буфер обмена для изображений не поддерживается")
}

// BringToFront выводит окно на передний план, восстанавливая его из свёрнутого или скрытого состояния.
func (w *Window) BringToFront() {
	if act, ok := w.platWin.(platform.WindowActivator); ok {
		act.BringToFront()
		return
	}
	w.Show()
}

// SetStrings устанавливает локализацию встроенных строк контролов для данного окна.
func (w *Window) SetStrings(s Strings) {
	if w.ctx != nil {
		w.ctx.Strings = s
		w.ctx.InvalidateStatic()
		w.Invalidate()
	}
}

// Minimize сворачивает окно.
func (w *Window) Minimize() {
	if w.platWin != nil {
		w.platWin.Minimize()
	}
}

// Maximize разворачивает окно на всю рабочую область экрана.
func (w *Window) Maximize() {
	if w.platWin != nil {
		w.platWin.Maximize()
	}
}

// Restore восстанавливает окно из свёрнутого или развёрнутого состояния.
func (w *Window) Restore() {
	if w.platWin != nil {
		w.platWin.Restore()
	}
}

// IsMaximized возвращает true, если окно развёрнуто.
func (w *Window) IsMaximized() bool {
	if w.platWin != nil {
		return w.platWin.IsMaximized()
	}
	return false
}

// IsMinimized возвращает true, если окно свёрнуто.
func (w *Window) IsMinimized() bool {
	if w.platWin != nil {
		return w.platWin.IsMinimized()
	}
	return false
}

// SetFullscreen включает или выключает полноэкранный режим.
func (w *Window) SetFullscreen(fullscreen bool) {
	if w.platWin != nil {
		w.platWin.SetFullscreen(fullscreen)
	}
}

// IsFullscreen возвращает true, если окно находится в полноэкранном режиме.
func (w *Window) IsFullscreen() bool {
	if w.platWin != nil {
		return w.platWin.IsFullscreen()
	}
	return false
}

// SetAlwaysOnTop закрепляет окно поверх всех остальных окон ОС (или снимает закрепление).
func (w *Window) SetAlwaysOnTop(onTop bool) {
	if tm, ok := w.platWin.(platform.WindowTopMoster); ok {
		tm.SetAlwaysOnTop(onTop)
	}
}

// IsAlwaysOnTop возвращает true, если окно закреплено поверх всех окон ОС.
func (w *Window) IsAlwaysOnTop() bool {
	if tm, ok := w.platWin.(platform.WindowTopMoster); ok {
		return tm.IsAlwaysOnTop()
	}
	return false
}

// Position возвращает экранные координаты верхнего левого угла окна в физических пикселях (xPx, yPx).
func (w *Window) Position() (xPx, yPx int) {
	if w.platWin != nil {
		return w.platWin.Position()
	}
	return 0, 0
}

// SetPosition перемещает окно в указанные экранные координаты в физических пикселях (xPx, yPx).
func (w *Window) SetPosition(xPx, yPx int) {
	if w.platWin != nil {
		w.platWin.SetPosition(xPx, yPx)
	}
}

// SetSize изменяет физический размер клиентской области окна (widthPx, heightPx).
func (w *Window) SetSize(widthPx, heightPx int) {
	if w.platWin != nil {
		w.platWin.SetSize(widthPx, heightPx)
	}
}

// NativeHandle возвращает нативный дескриптор окна ОС:
// HWND на Windows, XID на Linux X11, ID поверхности wl_surface на Linux Wayland,
// указатель NSWindow* на macOS; возвращает 0 в Headless-режиме или до создания окна.
func (w *Window) NativeHandle() uintptr {
	if w == nil || w.platWin == nil {
		return 0
	}
	return w.platWin.NativeHandle()
}

// SetHideOnClose управляет скрытием окна вместо закрытия (например, для сворачивания в трей).
func (w *Window) SetHideOnClose(hide bool) {
	w.hideOnClose = hide
}

// SetOnClose задаёт обработчик попытки закрытия окна. Возврат false отменяет закрытие.
func (w *Window) SetOnClose(fn func(*Window) bool) {
	w.onClose = fn
}

// SetOnDropFiles задаёт обработчик перетаскивания файлов из проводника ОС (xPx, yPx — координаты в физических пикселях).
func (w *Window) SetOnDropFiles(fn func(files []string, xPx, yPx int)) {
	w.onDropFiles = fn
}

// SetOnDragEnter задаёт обработчик входа курсора при перетаскивании OLE Drag & Drop (xPx, yPx — координаты в физических пикселях).
func (w *Window) SetOnDragEnter(fn func(xPx, yPx int, hasFiles bool)) {
	w.onDragEnter = fn
}

// SetOnDragOver задаёт обработчик перемещения курсора при перетаскивании OLE Drag & Drop (xPx, yPx — координаты в физических пикселях).
func (w *Window) SetOnDragOver(fn func(xPx, yPx int, hasFiles bool)) {
	w.onDragOver = fn
}

// SetOnDragLeave задаёт обработчик выхода курсора за пределы окна при перетаскивании OLE Drag & Drop.
func (w *Window) SetOnDragLeave(fn func()) {
	w.onDragLeave = fn
}

func (w *Window) handleCloseRequest() bool {
	if w.app != nil && w.app.isQuitting() {
		return true
	}
	if w.onClose != nil && !w.onClose(w) {
		return false
	}
	if w.hideOnClose {
		w.Hide()
		return false
	}
	return true
}

func (w *Window) handleEvent(ev platform.Event) {
	dispatched := false
	if w.app != nil {
		dispatched = w.app.flushDispatch()
		if dispatched {
			w.app.invalidateAllExcept(w)
		}
	}

	if ev.Type == platform.EventClose {
		w.ctx.HandleEvent(ev)
		if w.app != nil {
			w.app.removeWindow(w)
		}
		return
	}

	if ev.Type == platform.EventDropFiles {
		if w.onDropFiles != nil {
			w.onDropFiles(ev.Files, ev.X, ev.Y)
		}
		if w.drawFn != nil {
			w.ctx.Render(func(_ *ui.Context) {
				w.drawFn(w.ui)
			})
		}
		if dispatched {
			w.WakeUp()
		}
		return
	}

	if ev.Type == platform.EventDragEnter {
		if w.onDragEnter != nil {
			w.onDragEnter(ev.X, ev.Y, ev.HasFiles)
		}
		if w.drawFn != nil {
			w.ctx.Render(func(_ *ui.Context) {
				w.drawFn(w.ui)
			})
		}
		if dispatched {
			w.WakeUp()
		}
		return
	}

	if ev.Type == platform.EventDragOver {
		if w.onDragOver != nil {
			w.onDragOver(ev.X, ev.Y, ev.HasFiles)
		}
		if w.drawFn != nil {
			w.ctx.Render(func(_ *ui.Context) {
				w.drawFn(w.ui)
			})
		}
		if dispatched {
			w.WakeUp()
		}
		return
	}

	if ev.Type == platform.EventDragLeave {
		if w.onDragLeave != nil {
			w.onDragLeave()
		}
		if w.drawFn != nil {
			w.ctx.Render(func(_ *ui.Context) {
				w.drawFn(w.ui)
			})
		}
		if dispatched {
			w.WakeUp()
		}
		return
	}

	if ev.Type == platform.EventThemeChange {
		if w.app != nil && w.app.isPrimaryEventWindow(w) {
			w.app.handleSystemThemeChange(ev.DarkTheme)
		}
		if dispatched {
			w.WakeUp()
		}
		return
	}

	if w.ctx.HandleEvent(ev) {
		w.ctx.Render(func(_ *ui.Context) {
			if w.drawFn != nil {
				w.drawFn(w.ui)
			}
		})
	} else if dispatched {
		// Если событие не вызвало перерисовку (например, MouseMove при DirectInput или smartMouseMove),
		// пробуждаем текущее окно через EventWakeUp, чтобы не потерять обновление от Dispatch.
		w.WakeUp()
	}
}
