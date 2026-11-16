package anyui

import (
	"anyui/platform"
	"anyui/ui"
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

// SetTheme устанавливает новую тему оформления для окна и перерисовывает его.
func (w *Window) SetTheme(theme Theme) {
	if w.ctx != nil {
		w.ctx.Theme = theme
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
	if w.app != nil {
		if w.app.flushDispatch() {
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
		return
	}

	if w.ctx.HandleEvent(ev) {
		w.ctx.Render(func(_ *ui.Context) {
			if w.drawFn != nil {
				w.drawFn(w.ui)
			}
		})
	}
}

