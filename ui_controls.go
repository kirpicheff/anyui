package anyui

import "github.com/kirpicheff/anyui/ui"

// TabItem описывает вкладку в динамическом TabBar.
type TabItem = ui.TabItem

// TabBarEvent сообщает о действиях пользователя в TabBar за текущий кадр.
type TabBarEvent = ui.TabBarEvent

// DrawerSide задаёт сторону окна для выезжающей панели Drawer.
type DrawerSide = ui.DrawerSide

const (
	DrawerLeft   = ui.DrawerLeft
	DrawerRight  = ui.DrawerRight
	DrawerTop    = ui.DrawerTop
	DrawerBottom = ui.DrawerBottom
)

// ContextMenuItem описывает пункт кастомного контекстного меню.
type ContextMenuItem = ui.ContextMenuItem

// IconButton рисует квадратную кнопку с SVG-иконкой.
func (u *UI) IconButton(id, svg string, opts ...Option) bool {
	return u.ctx.IconButton(id, svg, opts...)
}

// Checkbox рисует флажок с подписью и возвращает true при изменении состояния.
func (u *UI) Checkbox(id, label string, checked *bool, opts ...Option) bool {
	return u.ctx.Checkbox(id, label, checked, opts...)
}

// SelectItem описывает элемент выпадающего списка с устойчивым строковым ключом, подписью и опциональной иконкой.
type SelectItem = ui.SelectItem

// SelectItems создает срез SelectItem из списка строк, где Key и Label совпадают.
var SelectItems = ui.SelectItems

// RadioButton рисует одиночную радиокнопку и возвращает true при выборе значения.
func (u *UI) RadioButton(id, label string, selected *string, value string, opts ...Option) bool {
	return u.ctx.RadioButton(id, label, selected, value, opts...)
}

// RadioGroup рисует вертикальную группу радиокнопок и обновляет selected.
func (u *UI) RadioGroup(id string, options []string, selected *int, opts ...Option) bool {
	return u.ctx.RadioGroup(id, options, selected, opts...)
}

// Separator рисует горизонтальный разделитель.
func (u *UI) Separator(opts ...Option) {
	u.ctx.Separator(opts...)
}

// Badge рисует компактный цветной бейдж с текстом.
func (u *UI) Badge(text string, opts ...Option) {
	u.ctx.Badge(text, opts...)
}

// ProgressBar рисует полосу прогресса (от 0.0 до 1.0).
func (u *UI) ProgressBar(progress float32, opts ...Option) {
	u.ctx.ProgressBar(progress, opts...)
}

// TextArea рисует многострочное поле ввода текста с переносами и вертикальным скроллом.
func (u *UI) TextArea(id string, value *string, placeholder string, opts ...Option) bool {
	return u.ctx.TextArea(id, value, placeholder, opts...)
}

// Select рисует выпадающий список с оверлеем поверх интерфейса.
func (u *UI) Select(id string, items []SelectItem, selected *string, opts ...Option) bool {
	return u.ctx.Select(id, items, selected, opts...)
}

// ComboBox рисует выпадающий список с полем поиска/фильтрации элементов.
func (u *UI) ComboBox(id string, items []SelectItem, selected *string, opts ...Option) bool {
	base := make([]Option, 0, len(opts)+1)
	base = append(base, Searchable(true))
	base = append(base, opts...)
	return u.ctx.Select(id, items, selected, base...)
}

// ListView рисует виртуализированный прокручиваемый список.
func (u *UI) ListView(id string, count int, itemHeightDp int, selected *int, drawItem func(index int, area Rect, selected bool), opts ...Option) bool {
	return u.ctx.ListView(id, count, itemHeightDp, selected, drawItem, opts...)
}

// Tabs рисует статические горизонтальные вкладки или вертикальное боковое меню.
func (u *UI) Tabs(id string, items []string, active *int, orient Orientation, opts ...Option) bool {
	return u.ctx.Tabs(id, items, active, orient, opts...)
}

// TabBar рисует динамические закрываемые вкладки с кнопками '×' и '+'.
func (u *UI) TabBar(id string, tabs []TabItem, active *int, opts ...Option) TabBarEvent {
	return u.ctx.TabBar(id, tabs, active, opts...)
}

// ScrollView создаёт прокручиваемую область с поддержкой колеса мыши и скроллбара.
func (u *UI) ScrollView(id string, wDp, hDp int, fn func(), opts ...Option) {
	u.ctx.ScrollView(id, wDp, hDp, fn, opts...)
}

// Modal рисует модальное окно с затемнением фона и блокировкой кликов под ним.
func (u *UI) Modal(id string, open *bool, title string, fn func(), opts ...Option) {
	u.ctx.Modal(id, open, title, fn, opts...)
}

// Popover рисует всплывающую панель, привязанную к области anchor.
func (u *UI) Popover(id string, open *bool, anchor Rect, fn func(), opts ...Option) {
	u.ctx.Popover(id, open, anchor, fn, opts...)
}

// FloatingPanel рисует перетаскиваемую плавающую панель поверх контента.
func (u *UI) FloatingPanel(id, title string, open *bool, fn func(), opts ...Option) {
	u.ctx.FloatingPanel(id, title, open, fn, opts...)
}

// Drawer рисует выезжающую боковую панель от края окна.
func (u *UI) Drawer(id string, open *bool, side DrawerSide, fn func(), opts ...Option) {
	u.ctx.Drawer(id, open, side, fn, opts...)
}

// ContextMenu открывает кастомное всплывающее меню по правому клику внутри anchor.
func (u *UI) ContextMenu(id string, anchor Rect, items []ContextMenuItem) {
	u.ctx.ContextMenu(id, anchor, items)
}

// Tooltip отображает всплывающую подсказку при наведении на область anchor.
func (u *UI) Tooltip(text string, anchor Rect) {
	u.ctx.Tooltip(text, anchor)
}

