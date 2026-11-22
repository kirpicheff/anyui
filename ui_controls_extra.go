package anyui

import (
	"time"

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

// DropZone определяет позицию сброса перетаскиваемого элемента относительно целевого.
type DropZone = ui.DropZone

const (
	DropAbove  = ui.DropAbove
	DropBelow  = ui.DropBelow
	DropInside = ui.DropInside
)

// DragPayload описывает типизированные данные для кросс-виджетного перетаскивания (Drag & Drop).
type DragPayload = ui.DragPayload

// ВАЖНО: данный docstring должен быть строго синхронизирован с ui/options.go (ui.DragEvent).

// DragEvent описывает событие завершения перетаскивания элемента (Reorder или кросс-виджетный Drop).
//
// Для списков и таблиц (Table, TableEx, ListView, ListViewEx):
//   - FromIndex: исходный индекс перемещаемого элемента (>= 0; -1 при внешнем переносе из дерева).
//   - ToIndex: при внутреннем Reorder — вычисленная позиция вставки после удаления перемещаемого элемента (0 <= ToIndex < rowCount); при внешнем Drop — позиция вставки (0 <= ToIndex <= rowCount; 0 для пустого списка/таблицы).
//   - FromKey и ToKey строго пусты ("") при внутреннем Reorder; при внешнем Drop из дерева FromKey содержит node.ID.
//   - Допустимые зоны Drop: DropAbove и DropBelow (при внешнем Drop в пустой список/таблицу — DropInside).
//   - Если при внутреннем Reorder ToIndex == FromIndex, операция является no-op и событие не генерируется.
//   - ListViewEx / TableEx: при Reorder приложение обязано синхронно переставлять элементы в срезе чекбоксов и карте выделения, иначе отметки сохранятся за исходными числовыми индексами.
//
// Для иерархических деревьев (TreeView, TreeViewEx):
//   - FromIndex и ToIndex равны -1 при внутреннем Reorder; при внешнем Drop из списка/таблицы FromIndex >= 0.
//   - FromKey: ID перемещаемого узла (sourceNode.ID; "" при внешнем Drop из списка/таблицы).
//   - ToKey: ID якорного узла (targetNode.ID; "" при внешнем Drop в пустое дерево), относительно которого применяется Drop; позицию вставки в список детей родителя вычисляет приложение.
//   - По умолчанию внутреннее перетаскивание узлов разрешено только внутри одного родителя (source.Parent == target.Parent), зоны DropAbove и DropBelow. При попытке сброса на узел другого родителя событие отбрасывается.
//   - По умолчанию DropBelow на узле (в том числе раскрытом) вставляет элемент в того же родителя сразу после всего поддерева ToKey; для раскрытого узла с детьми при включённой AllowReparent() — первым ребёнком в ToKey.Children (см. ниже).
//   - Сброс узла в самого себя (ToKey == FromKey при FromWidget == ToWidget) является no-op и событие не генерирует.
//   - TreeViewEx: при перетаскивании одного узла из группы выделенных перемещается только он, при этом выделение группы не сбрасывается.
//
// Режим AllowReparent() в TreeView и TreeViewEx (смена родителя и вложение):
//   - DropInside: новым родителем становится сам ToKey (вставка в ToKey.Children).
//   - DropAbove: новым родителем становится target.ParentID (для корневого узла — "" / верхний уровень), вставка перед ToKey.
//   - DropBelow: для раскрытого узла с детьми (ToKey.Expanded && len(ToKey.Children) > 0) новым родителем становится сам ToKey (позиция первого ребёнка); для свёрнутого узла или листа — target.ParentID (сразу после поддерева ToKey).
//   - Защита от циклов и no-op: сброс узла в любого своего потомка и DropInside на собственного текущего родителя являются no-op и событие не генерируют. При этом DropAbove/DropBelow на собственного текущего родителя не являются no-op (перемещают узел на уровень родителя).
//
// Кросс-виджетный перенос (DragSource + AcceptDrop + OnDrop):
//   - Payload содержит типизированные данные (*DragPayload), сформированные источником DragSource.
//   - FromWidget != ToWidget; событие доставляется в OnDrop целевого виджета (и в TableEvent.Drag для Table/TableEx).
type DragEvent = ui.DragEvent

// Orientation задаёт направление ориентации элементов интерфейса (горизонтальное или вертикальное).
type Orientation = ui.Orientation

const (
	Horizontal = ui.Horizontal
	Vertical   = ui.Vertical
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

// ToastPosition задаёт позицию всплывающих уведомлений на экране.
type ToastPosition = ui.ToastPosition

const (
	ToastBottomRight  = ui.ToastBottomRight
	ToastBottomLeft   = ui.ToastBottomLeft
	ToastTopRight     = ui.ToastTopRight
	ToastTopLeft      = ui.ToastTopLeft
	ToastBottomCenter = ui.ToastBottomCenter
	ToastTopCenter    = ui.ToastTopCenter
)

// Table рисует таблицу с заголовками, ресайзом ширины колонок, сортировкой и виртуализацией строк.
// Колбэк drawCell вызывается для отрисовки видимых ячеек; координаты ячейки areaPx передаются в физических пикселях (Px).
func (u *UI) Table(id string, cols []TableColumn, rowCount int, selected *int, sort *TableSort, drawCell func(row, col int, areaPx Rect, isSelected bool), opts ...Option) TableEvent {
	return u.ctx.Table(id, cols, rowCount, selected, sort, drawCell, opts...)
}

// TreeView рисует древовидный список с раскрытием узлов и виртуализацией.
func (u *UI) TreeView(id string, roots []*TreeNode, selectedID *string, opts ...Option) bool {
	return u.ctx.TreeView(id, roots, selectedID, opts...)
}

// SplitPane делит область на 2 панели с перетаскиваемым разделителем.
func (u *UI) SplitPane(id string, orient Orientation, ratio *float32, firstFn, secondFn func(), opts ...Option) {
	u.ctx.SplitPane(id, orient, ratio, firstFn, secondFn, opts...)
}

// Collapsible рисует сворачиваемую секцию с заголовком и содержимым.
func (u *UI) Collapsible(id, title string, expanded *bool, fn func(), opts ...Option) bool {
	return u.ctx.Collapsible(id, title, expanded, fn, opts...)
}

// Accordion рисует группу сворачиваемых секций с независимым состоянием раскрытия по строковым ID.
func (u *UI) Accordion(id string, items []AccordionItem, openIDs map[string]bool, opts ...Option) bool {
	return u.ctx.Accordion(id, items, openIDs, opts...)
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
func (u *UI) ColorPicker(id string, col *Color, opts ...Option) bool {
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

// SetToastPosition задаёт угол или сторону окна для отображения всплывающих уведомлений.
func (u *UI) SetToastPosition(pos ToastPosition) {
	u.ctx.SetToastPosition(pos)
}

// SetToastOffset задаёт отступы всплывающих уведомлений от краёв окна в логических пикселях (по умолчанию 16, 16).
func (u *UI) SetToastOffset(xDp, yDp Dp) {
	u.ctx.SetToastOffset(xDp, yDp)
}

// Spinner рисует анимированный индикатор загрузки.
func (u *UI) Spinner(id string, sizeDp Dp, opts ...Option) {
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
