package anyui

import (
	"time"

	"github.com/kirpicheff/anyui/render"
	"github.com/kirpicheff/anyui/ui"
)

// TableColumn описывает колонку таблицы.
type TableColumn = ui.TableColumn

// TableSort задаёт состояние сортировки таблицы.
type TableSort = ui.TableSort

// TableEvent сообщает о действиях пользователя в таблице за текущий кадр.
type TableEvent = ui.TableEvent

// TreeNode описывает узел дерева для TreeView.
type TreeNode = ui.TreeNode

// SplitOrientation задаёт направление разделения в SplitPane.
type SplitOrientation = ui.SplitOrientation

const (
	SplitHorizontal = ui.SplitHorizontal
	SplitVertical   = ui.SplitVertical
)

// AccordionItem описывает одну секцию аккордеона.
type AccordionItem = ui.AccordionItem

// ToastKind задаёт тип всплывающего уведомления.
type ToastKind = ui.ToastKind

const (
	ToastInfo    = ui.ToastInfo
	ToastSuccess = ui.ToastSuccess
	ToastWarning = ui.ToastWarning
	ToastError   = ui.ToastError
)

// Table рисует таблицу с заголовками, ресайзом ширины колонок, сортировкой и виртуализацией строк.
func (u *UI) Table(id string, cols []TableColumn, rowCount int, selected *int, sort *TableSort, drawCell func(row, col int, area Rect, isSelected bool), opts ...Option) TableEvent {
	return u.ctx.Table(id, cols, rowCount, selected, sort, drawCell, opts...)
}

// TreeView рисует древовидный список с раскрытием узлов и виртуализацией.
func (u *UI) TreeView(id string, roots []*TreeNode, selectedID *string, opts ...Option) bool {
	return u.ctx.TreeView(id, roots, selectedID, opts...)
}

// SplitPane делит область на 2 панели с перетаскиваемым разделителем.
func (u *UI) SplitPane(id string, orient SplitOrientation, ratio *float32, firstFn, secondFn func(), opts ...Option) {
	u.ctx.SplitPane(id, orient, ratio, firstFn, secondFn, opts...)
}

// Collapsible рисует сворачиваемую секцию с заголовком и содержимым.
func (u *UI) Collapsible(id, title string, expanded *bool, fn func(), opts ...Option) bool {
	return u.ctx.Collapsible(id, title, expanded, fn, opts...)
}

// Accordion рисует группу сворачиваемых секций, где открыта не более чем одна секция.
func (u *UI) Accordion(id string, items []AccordionItem, openIndex *int, opts ...Option) bool {
	return u.ctx.Accordion(id, items, openIndex, opts...)
}

// NumberInput рисует числовое поле с кнопками '−' и '+' (Stepper).
func (u *UI) NumberInput(id string, value *float64, minVal, maxVal, step float64, decimals int, opts ...Option) bool {
	return u.ctx.NumberInput(id, value, minVal, maxVal, step, decimals, opts...)
}

// Slider рисует горизонтальный ползунок для выбора значения в диапазоне [minVal, maxVal].
func (u *UI) Slider(id string, value *float32, minVal, maxVal float32, opts ...Option) bool {
	return u.ctx.Slider(id, value, minVal, maxVal, opts...)
}

// RangeSlider рисует ползунок диапазона с двумя ручками [low, high].
func (u *UI) RangeSlider(id string, low, high *float32, minVal, maxVal float32, opts ...Option) bool {
	return u.ctx.RangeSlider(id, low, high, minVal, maxVal, opts...)
}

// ColorPicker рисует выбор цвета с выпадающей палитрой и HEX-полем.
func (u *UI) ColorPicker(id string, col *render.Color, opts ...Option) bool {
	return u.ctx.ColorPicker(id, col, opts...)
}

// DatePicker рисует поле выбора даты с выпадающим календарём.
func (u *UI) DatePicker(id string, date *time.Time, opts ...Option) bool {
	return u.ctx.DatePicker(id, date, opts...)
}

// ShowToast добавляет всплывающее уведомление в очередь отображения.
func (u *UI) ShowToast(title, text string, kind ToastKind, duration ...time.Duration) {
	u.ctx.ShowToast(title, text, kind, duration...)
}

// Spinner рисует анимированный индикатор загрузки.
func (u *UI) Spinner(id string, sizeDp int, opts ...Option) {
	u.ctx.Spinner(id, sizeDp, opts...)
}

// SelectableLabel отображает статический текст с возможностью выделения мышью и копирования.
func (u *UI) SelectableLabel(id, text string, opts ...Option) {
	u.ctx.SelectableLabel(id, text, opts...)
}

// Hyperlink отображает кликабельную текстовую ссылку и возвращает true при нажатии.
func (u *UI) Hyperlink(id, text string, opts ...Option) bool {
	return u.ctx.Hyperlink(id, text, opts...)
}

// MaskedInput рисует однострочное поле ввода с шаблоном маски.
func (u *UI) MaskedInput(id string, value *string, mask string, opts ...Option) bool {
	return u.ctx.MaskedInput(id, value, mask, opts...)
}

