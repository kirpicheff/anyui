package anyui

import (
	"time"

	"anyui/ui"
)

// BarMenuItem описывает пункт выпадающего меню в MenuBar.
type BarMenuItem = ui.BarMenuItem

// MenuBarItem описывает раздел верхнего уровня в MenuBar.
type MenuBarItem = ui.MenuBarItem

// ToolbarItem описывает элемент панели инструментов Toolbar.
type ToolbarItem = ui.ToolbarItem

// StatusBarSection описывает секцию нижней строки состояния StatusBar.
type StatusBarSection = ui.StatusBarSection

// BreadcrumbItem описывает сегмент навигационной цепочки Breadcrumbs.
type BreadcrumbItem = ui.BreadcrumbItem

// LogLevel задаёт уровень важности записи лога для LogView.
type LogLevel = ui.LogLevel

const (
	LogInfo  = ui.LogInfo
	LogWarn  = ui.LogWarn
	LogError = ui.LogError
	LogDebug = ui.LogDebug
)

// LogEntry описывает одну строку в LogView.
type LogEntry = ui.LogEntry

// ChartSeries описывает одну серию данных для LineChart и BarChart.
type ChartSeries = ui.ChartSeries

// TableExConfig задаёт расширенные настройки таблицы TableEx.
type TableExConfig = ui.TableExConfig

// TitleBarConfig задаёт настройки кастомной шапки окна TitleBar.
type TitleBarConfig = ui.TitleBarConfig

// SetTreeNodeChecked каскадно устанавливает состояние Checked узлу и всем его дочерним элементам.
func SetTreeNodeChecked(node *TreeNode, checked bool) {
	ui.SetTreeNodeChecked(node, checked)
}

// SyncTreeCheckState пересчитывает состояния Checked и Indeterminate для всех родительских узлов дерева.
func SyncTreeCheckState(roots []*TreeNode) {
	ui.SyncTreeCheckState(roots)
}

// MenuBar отрисовывает главное верхнее меню окна с выпадающими подменю и хоткеями.
func (u *UI) MenuBar(id string, menus []MenuBarItem, opts ...Option) {
	u.ctx.MenuBar(id, menus, opts...)
}

// Toolbar отрисовывает панель инструментов с кнопками, иконками, разделителями и подсказками.
func (u *UI) Toolbar(id string, items []ToolbarItem, opts ...Option) {
	u.ctx.Toolbar(id, items, opts...)
}

// StatusBar отрисовывает нижнюю строку состояния с секциями и индикаторами.
func (u *UI) StatusBar(id string, sections []StatusBarSection, opts ...Option) {
	u.ctx.StatusBar(id, sections, opts...)
}

// Breadcrumbs отрисовывает навигационную цепочку и возвращает индекс кликнутого сегмента (-1 если клика не было).
func (u *UI) Breadcrumbs(id string, items []BreadcrumbItem, opts ...Option) int {
	return u.ctx.Breadcrumbs(id, items, opts...)
}

// Pagination отрисовывает постраничную навигацию (номера страниц 1..totalPages) и возвращает true при смене страницы.
func (u *UI) Pagination(id string, currentPage *int, totalPages int, opts ...Option) bool {
	return u.ctx.Pagination(id, currentPage, totalPages, opts...)
}

// LogView отображает виртуальный список записей лога с Follow tail, раскраской уровней, поиском и копированием.
func (u *UI) LogView(id string, entries []LogEntry, followTail *bool, opts ...Option) {
	u.ctx.LogView(id, entries, followTail, opts...)
}

// VirtualTextView отображает большой массив текстовых строк с виртуализацией и выделением.
func (u *UI) VirtualTextView(id string, lines []string, opts ...Option) {
	u.ctx.VirtualTextView(id, lines, opts...)
}

// Sparkline отрисовывает компактный мини-график тренда без осей, с тултипом при наведении.
func (u *UI) Sparkline(id string, values []float64, opts ...Option) {
	u.ctx.Sparkline(id, values, opts...)
}

// LineChart отрисовывает линейный график с сеткой, осями, легендой и всплывающей подсказкой.
func (u *UI) LineChart(id string, labels []string, series []ChartSeries, opts ...Option) {
	u.ctx.LineChart(id, labels, series, opts...)
}

// BarChart отрисовывает столбчатую диаграмму с сеткой, осями, легендой и тултипами.
func (u *UI) BarChart(id string, labels []string, series []ChartSeries, opts ...Option) {
	u.ctx.BarChart(id, labels, series, opts...)
}

// TimePicker отрисовывает выбор времени (ЧЧ:ММ:СС) с кнопками +/- и колесом мыши.
func (u *UI) TimePicker(id string, t *time.Time, opts ...Option) bool {
	return u.ctx.TimePicker(id, t, opts...)
}

// DateTimePicker объединяет выбор даты и времени в одной строке.
func (u *UI) DateTimePicker(id string, dt *time.Time, opts ...Option) bool {
	return u.ctx.DateTimePicker(id, dt, opts...)
}

// MultiSelect отрисовывает выпадающий список с множественным выбором и чекбоксами.
func (u *UI) MultiSelect(id string, items []string, selected *[]bool, opts ...Option) bool {
	return u.ctx.MultiSelect(id, items, selected, opts...)
}

// CheckComboBox — синоним MultiSelect.
func (u *UI) CheckComboBox(id string, items []string, selected *[]bool, opts ...Option) bool {
	return u.ctx.CheckComboBox(id, items, selected, opts...)
}

// TagInput отрисовывает поле ввода тегов/чипсов.
func (u *UI) TagInput(id string, tags *[]string, placeholder string, opts ...Option) bool {
	return u.ctx.TagInput(id, tags, placeholder, opts...)
}

// ListViewEx отрисовывает список с мультивыбором и опциональными чекбоксами.
func (u *UI) ListViewEx(id string, count int, itemHeightDp int, selected map[int]bool, checked []bool, drawItem func(index int, area Rect, isSelected bool), opts ...Option) bool {
	return u.ctx.ListViewEx(id, count, itemHeightDp, selected, checked, drawItem, opts...)
}

// TreeViewEx отрисовывает дерево с мультивыбором узлов и опциональными каскадными чекбоксами.
func (u *UI) TreeViewEx(id string, roots []*TreeNode, selectedIDs map[string]bool, showCheckboxes bool, opts ...Option) bool {
	return u.ctx.TreeViewEx(id, roots, selectedIDs, showCheckboxes, opts...)
}

// TableEx отрисовывает расширенную таблицу с мультивыбором, чекбоксами, закреплением колонок и inline-редактированием.
func (u *UI) TableEx(id string, cols []TableColumn, rowCount int, cfg *TableExConfig, sort *TableSort, drawCell func(row, col int, area Rect, isSelected bool), opts ...Option) TableEvent {
	return u.ctx.TableEx(id, cols, rowCount, cfg, sort, drawCell, opts...)
}

// TitleBar отрисовывает кастомную шапку окна с иконкой, заголовком, пользовательскими слотами и кнопками управления.
func (u *UI) TitleBar(id, title string, cfg TitleBarConfig, opts ...Option) {
	u.ctx.TitleBar(id, title, cfg, opts...)
}

// TitleBarHitTest определяет роль точки для HitTest в безрамочном окне с виджетом TitleBar.
func (u *UI) TitleBarHitTest(x, y int) HitTestResult {
	return u.ctx.TitleBarHitTest(x, y)
}
