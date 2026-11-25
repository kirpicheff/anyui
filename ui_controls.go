package anyui

import "github.com/kirpicheff/anyui/ui"

// TabItem описывает вкладку в динамическом TabBar.
type TabItem = ui.TabItem

// TabItems создаёт срез TabItem из списка строк, где ID и Title совпадают.
// Метки должны быть уникальны, поскольку они же служат ключами вкладок.
func TabItems(labels ...string) []TabItem {
	return ui.TabItems(labels...)
}

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
func SelectItems(labels ...string) []SelectItem {
	return ui.SelectItems(labels...)
}

// RadioButton рисует одиночную радиокнопку и возвращает true при выборе значения.
func (u *UI) RadioButton(id, label string, selected *string, value string, opts ...Option) bool {
	return u.ctx.RadioButton(id, label, selected, value, opts...)
}

// RadioGroup рисует вертикальную группу радиокнопок с выбором по строковому ключу.
func (u *UI) RadioGroup(id string, items []SelectItem, selected *string, opts ...Option) bool {
	return u.ctx.RadioGroup(id, items, selected, opts...)
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
// Для выпадающего списка с поиском/фильтрацией используйте ui.Select с опцией anyui.Searchable().
func (u *UI) Select(id string, items []SelectItem, selected *string, opts ...Option) bool {
	return u.ctx.Select(id, items, selected, opts...)
}

// ListView рисует виртуализированный прокручиваемый список.
// Колбэк drawItem получает координаты элемента areaPx в физических пикселях (Px).
func (u *UI) ListView(id string, count int, itemHeightDp Dp, selected *int, drawItem func(index int, areaPx Rect, selected bool), opts ...Option) bool {
	return u.ctx.ListView(id, count, itemHeightDp, selected, drawItem, opts...)
}

// TabBar рисует панель вкладок (горизонтальную или вертикальную через TabOrientation)
// с опциональными кнопками '×' (TabItem.Closable) и '+' (скрывается через HideAddButton).
func (u *UI) TabBar(id string, tabs []TabItem, activeID *string, opts ...Option) TabBarEvent {
	return u.ctx.TabBar(id, tabs, activeID, opts...)
}

// ScrollView создаёт прокручиваемую область с поддержкой колеса мыши и скроллбара.
// Размеры задаются через опции Width, Height, FillWidth, FillHeight (по умолчанию — доступное пространство).
func (u *UI) ScrollView(id string, fn func(), opts ...Option) {
	u.ctx.ScrollView(id, fn, opts...)
}

// Modal рисует модальное окно с затемнением фона и блокировкой кликов под ним.
// В open передавайте указатель напрямую на долгоживущее поле состояния (не локальную копию),
// так как отрисовка и закрытие по '×' выполняются в конце кадра внутри PushOverlay.
func (u *UI) Modal(id string, open *bool, title string, fn func(), opts ...Option) {
	u.ctx.Modal(id, open, title, fn, opts...)
}

// Popover рисует всплывающую панель, привязанную к области anchor (в логических пикселях Dp / CurrentArea()).
func (u *UI) Popover(id string, open *bool, anchor Rect, fn func(), opts ...Option) {
	if u.ctx == nil {
		return
	}
	u.ctx.Popover(id, open, u.scaleRect(anchor), fn, opts...)
}

// FloatingPanel рисует перетаскиваемую плавающую панель поверх контента.
func (u *UI) FloatingPanel(id, title string, open *bool, fn func(), opts ...Option) {
	u.ctx.FloatingPanel(id, title, open, fn, opts...)
}

// Drawer рисует выезжающую боковую панель от края окна.
func (u *UI) Drawer(id string, open *bool, side DrawerSide, fn func(), opts ...Option) {
	u.ctx.Drawer(id, open, side, fn, opts...)
}

// ContextMenu открывает кастомное всплывающее меню по правому клику внутри anchor (в логических пикселях Dp / CurrentArea()).
func (u *UI) ContextMenu(id string, anchor Rect, items []ContextMenuItem, opts ...Option) {
	if u.ctx == nil {
		return
	}
	u.ctx.ContextMenu(id, u.scaleRect(anchor), items, opts...)
}

// Tooltip отображает всплывающую подсказку при наведении на область anchor (в логических пикселях Dp / CurrentArea()).
func (u *UI) Tooltip(text string, anchor Rect, opts ...Option) {
	if u.ctx == nil {
		return
	}
	u.ctx.Tooltip(text, u.scaleRect(anchor), opts...)
}

// Anchor размещает маркер якоря с заданным id для ScrollToID.
func (u *UI) Anchor(id string) {
	u.ctx.Anchor(id)
}
