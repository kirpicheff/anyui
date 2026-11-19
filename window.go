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
	onDropFiles func(files []string, x, y int)
}

// SetTitle меняет заголовок окна.
func (w *Window) SetTitle(title string) {
	if w.platWin != nil {
		w.platWin.SetTitle(title)
	}
}

// SetIcon программно устанавливает иконку окна из RGBA-буфера (width * height * 4).
// Передача пустого буфера и нулевых размеров сбрасывает программную иконку к иконке класса окна.
func (w *Window) SetIcon(rgba []byte, width, height int) error {
	if setter, ok := w.platWin.(platform.WindowIconSetter); ok {
		return setter.SetIcon(rgba, width, height)
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

// Position возвращает экранные координаты верхнего левого угла окна.
func (w *Window) Position() (int, int) {
	if w.platWin != nil {
		return w.platWin.Position()
	}
	return 0, 0
}

// SetPosition перемещает окно в указанные экранные координаты.
func (w *Window) SetPosition(x, y int) {
	if w.platWin != nil {
		w.platWin.SetPosition(x, y)
	}
}

// SetSize изменяет логический размер клиентской области окна.
func (w *Window) SetSize(width, height int) {
	if w.platWin != nil {
		w.platWin.SetSize(width, height)
	}
}

// SetHideOnClose управляет скрытием окна вместо закрытия (например, для сворачивания в трей).
func (w *Window) SetHideOnClose(hide bool) {
	w.hideOnClose = hide
}

// SetOnClose задаёт обработчик попытки закрытия окна. Возврат false отменяет закрытие.
func (w *Window) SetOnClose(fn func(*Window) bool) {
	w.onClose = fn
}

// SetOnDropFiles задаёт обработчик перетаскивания файлов из проводника ОС.
func (w *Window) SetOnDropFiles(fn func(files []string, x, y int)) {
	w.onDropFiles = fn
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
			w.Invalidate()
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

