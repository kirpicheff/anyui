package anyui

import (
	"image"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
	"github.com/kirpicheff/anyui/ui"
)

// Modifiers хранит состояние модификаторов клавиатуры для горячих клавиш.
type Modifiers = platform.Modifiers

// KeyEvent описывает сырое событие нажатия или отпускания клавиши для InteractiveCanvas.
type KeyEvent = ui.RawKeyEvent

// MouseButton задаёт кнопку мыши для событий InteractiveCanvas.
type MouseButton = platform.MouseButton

const (
	MouseLeft   = platform.MouseLeft
	MouseRight  = platform.MouseRight
	MouseMiddle = platform.MouseMiddle
)

// CanvasCallbacks задаёт обработчики рисования и сырого ввода для InteractiveCanvas.
type CanvasCallbacks = ui.CanvasCallbacks

// CursorType задаёт системный курсор мыши.
type CursorType = platform.CursorType

const (
	CursorArrow      = platform.CursorArrow
	CursorHand       = platform.CursorHand
	CursorIBeam      = platform.CursorIBeam
	CursorResizeH    = platform.CursorResizeH
	CursorResizeV    = platform.CursorResizeV
	CursorResizeNWSE = platform.CursorResizeNWSE
	CursorResizeNESW = platform.CursorResizeNESW
	CursorNotAllowed = platform.CursorNotAllowed
	CursorCrosshair  = platform.CursorCrosshair
	CursorMove       = platform.CursorMove
)

// UI предоставляет публичный Immediate Mode API для построения интерфейса в окне.
type UI struct {
	ctx *ui.Context
	win *Window
	app *App
}

// Theme возвращает текущую тему окна.
func (u *UI) Theme() Theme {
	return u.ctx.Theme
}

// SetTheme меняет тему текущего окна, синхронизирует заголовок ОС и запрашивает перерисовку.
func (u *UI) SetTheme(theme Theme) {
	if u.win != nil {
		u.win.applyTheme(theme, false)
		return
	}
	u.ctx.Theme = theme
	u.ctx.InvalidateStatic()
	u.ctx.RequestRedraw()
}

// Strings возвращает текущий набор локализованных строк встроенных контролов.
func (u *UI) Strings() Strings {
	return u.ctx.Strings
}

// SetStrings задаёт локализацию встроенных строк контролов для текущего окна.
func (u *UI) SetStrings(s Strings) {
	if u.ctx.Strings == s {
		return
	}
	u.ctx.Strings = s
	u.ctx.InvalidateStatic()
	u.ctx.RequestRedraw()
}

// Window возвращает окно, к которому привязан текущий UI-контекст.
func (u *UI) Window() *Window {
	return u.win
}

// App возвращает экземпляр приложения.
func (u *UI) App() *App {
	return u.app
}

// Canvas возвращает низкоуровневый холст окна для кастомного рисования.
func (u *UI) Canvas() *render.Canvas {
	return u.ctx.Canvas
}

// DoubleClicked возвращает true, если в текущем кадре произошёл двойной клик ЛКМ.
func (u *UI) DoubleClicked() bool {
	return u.ctx.DoubleClicked()
}

// MiddleClicked возвращает true, если в текущем кадре нажата средняя кнопка мыши.
func (u *UI) MiddleClicked() bool {
	return u.ctx.MiddleMousePressed
}

// S переводит логические пиксели (dp) в физические с учётом DPI-масштаба.
func (u *UI) S(dp int) int {
	return u.ctx.S(dp)
}

// Sf переводит дробные логические пиксели (dp) в физические.
func (u *UI) Sf(dp float32) float32 {
	return u.ctx.Sf(dp)
}

// CurrentArea возвращает прямоугольник текущего контейнера в физических пикселях.
func (u *UI) CurrentArea() Rect {
	return u.ctx.CurrentArea()
}

// Label выводит строку текста с опциональной стилизацией.
func (u *UI) Label(text string, opts ...Option) {
	u.ctx.LabelOpts(text, opts...)
}

// StatusDot рисует цветной кружок-индикатор и подпись справа от него.
func (u *UI) StatusDot(dotColor Color, label string) {
	u.ctx.StatusDot(dotColor, label)
}

// Card рисует скруглённую карточку и размещает внутри неё дочерние виджеты.
func (u *UI) Card(fn func(), opts ...Option) {
	u.ctx.CardOpts(fn, opts...)
}

// Button рисует кнопку и возвращает true в кадре нажатия.
func (u *UI) Button(id, label string, opts ...Option) bool {
	return u.ctx.ButtonOpts(id, label, opts...)
}

// Toggle рисует переключатель (switch) с подписью и возвращает true при переключении.
func (u *UI) Toggle(id, label string, value *bool, opts ...Option) bool {
	return u.ctx.ToggleOpts(id, label, value, opts...)
}

// TextInput рисует однострочное поле ввода текста и возвращает true при изменении значения.
func (u *UI) TextInput(id string, value *string, placeholder string, opts ...Option) bool {
	return u.ctx.TextInputOpts(id, value, placeholder, opts...)
}

// Image выводит изображение image.Image в области wDp x hDp.
func (u *UI) Image(img image.Image, wDp, hDp int, mode ImageScaleMode) {
	u.ctx.Image(img, wDp, hDp, mode)
}

// CanvasView выделяет область wDp x hDp для кастомного рисования (графики, видеокадры, градиенты).
func (u *UI) CanvasView(wDp, hDp int, drawFn func(canvas *render.Canvas, area Rect)) {
	u.ctx.CanvasView(wDp, hDp, drawFn)
}

// InteractiveCanvas выделяет интерактивную область wDp x hDp с поддержкой фокуса,
// захвата перетаскивания мыши и передачи сырых событий клавиатуры (со ScanCode/Extended) и мыши/колеса.
func (u *UI) InteractiveCanvas(id string, wDp, hDp int, cb CanvasCallbacks, opts ...Option) {
	u.ctx.InteractiveCanvas(id, wDp, hDp, cb, opts...)
}

// DrawSVG рисует SVG-иконку по координатам (x, y) с размером sizeDp и цветом tint.
func (u *UI) DrawSVG(svgData string, x, y, sizeDp int, tint Color) {
	u.ctx.Canvas.DrawSVG(svgData, x, y, u.ctx.S(sizeDp), tint)
}

// Column располагает дочерние элементы вертикально с отступом gapDp.
func (u *UI) Column(gapDp int, fn func()) {
	u.ctx.Column(gapDp, fn)
}

// ColumnCentered располагает дочерние элементы вертикально и центрирует их по горизонтали.
func (u *UI) ColumnCentered(gapDp int, fn func()) {
	u.ctx.ColumnCentered(gapDp, fn)
}

// Row располагает дочерние элементы горизонтально с отступом gapDp и высотой heightDp.
func (u *UI) Row(gapDp, heightDp int, fn func()) {
	u.ctx.Row(gapDp, heightDp, fn)
}

// RowCentered располагает дочерние элементы горизонтально с выравниванием по вертикальному центру.
func (u *UI) RowCentered(gapDp, heightDp int, fn func()) {
	u.ctx.RowCentered(gapDp, heightDp, fn)
}

// Padding добавляет внутренние отступы со всех сторон.
func (u *UI) Padding(padDp int, fn func()) {
	u.ctx.Padding(padDp, fn)
}

// PaddingXY добавляет горизонтальный и вертикальный внутренние отступы.
func (u *UI) PaddingXY(pxDp, pyDp int, fn func()) {
	u.ctx.PaddingXY(pxDp, pyDp, fn)
}

// Spacer сдвигает курсор раскладки на указанное количество логических пикселей.
func (u *UI) Spacer(dp int) {
	u.ctx.Spacer(dp)
}

// FlexSpacer заполняет оставшееся пространство с резервом reserveDp.
func (u *UI) FlexSpacer(reserveDp int) {
	u.ctx.FlexSpacer(reserveDp)
}

// Align задаёт выравнивание элементов по поперечной оси.
type Align = ui.Align

const (
	AlignStart   = ui.AlignStart
	AlignCenter  = ui.AlignCenter
	AlignEnd     = ui.AlignEnd
	AlignStretch = ui.AlignStretch
)

// Justify задаёт распределение элементов по главной оси контейнера.
type Justify = ui.Justify

const (
	JustifyStart        = ui.JustifyStart
	JustifyCenter       = ui.JustifyCenter
	JustifyEnd          = ui.JustifyEnd
	JustifySpaceBetween = ui.JustifySpaceBetween
	JustifySpaceAround  = ui.JustifySpaceAround
)

// FlexItem описывает элемент для пропорционального деления пространства в RowFlex / ColumnFlex.
type FlexItem = ui.FlexItem

// Expand выделяет дочернему блоку всё оставшееся пространство в текущем контейнере.
func (u *UI) Expand(fn func()) {
	u.ctx.Expand(fn)
}

// RowFlex располагает элементы горизонтально, деля ширину пропорционально весам FlexItem.
func (u *UI) RowFlex(gapDp, heightDp int, items ...FlexItem) {
	u.ctx.RowFlex(gapDp, heightDp, items...)
}

// ColumnFlex располагает элементы вертикально, деля высоту пропорционально весам FlexItem.
func (u *UI) ColumnFlex(gapDp int, items ...FlexItem) {
	u.ctx.ColumnFlex(gapDp, items...)
}

// DockBottom резервирует bottomDp снизу для bottomFn, а всю оставшуюся высоту сверху отдаёт bodyFn.
func (u *UI) DockBottom(gapDp, bottomDp int, bodyFn, bottomFn func()) {
	u.ctx.DockBottom(gapDp, bottomDp, bodyFn, bottomFn)
}

// DockTop резервирует topDp сверху для topFn, а всю оставшуюся высоту снизу отдаёт bodyFn.
func (u *UI) DockTop(gapDp, topDp int, topFn, bodyFn func()) {
	u.ctx.DockTop(gapDp, topDp, topFn, bodyFn)
}

// PageConfig задаёт параметры типовой разметки страницы с шапкой, прокручиваемым телом и футером.
type PageConfig = ui.PageConfig

// Page строит типовой экран: опциональная шапка сверху, прокручиваемое тело по центру и опциональный футер снизу.
func (u *UI) Page(cfg PageConfig, headerFn, bodyFn, footerFn func()) {
	u.ctx.Page(cfg, headerFn, bodyFn, footerFn)
}

// PushID добавляет префикс пространства имён в стек ID виджетов.
func (u *UI) PushID(key any) {
	u.ctx.PushID(key)
}

// PopID удаляет верхний префикс из стека ID виджетов.
func (u *UI) PopID() {
	u.ctx.PopID()
}

// IDScope задаёт изолированное пространство имён ID на время выполнения fn.
func (u *UI) IDScope(key any, fn func()) {
	u.ctx.IDScope(key, fn)
}

// RowWith располагает элементы горизонтально с заданным выравниванием Align и распределением Justify.
func (u *UI) RowWith(align Align, justify Justify, gapDp, heightDp int, items ...func()) {
	u.ctx.RowWith(align, justify, gapDp, heightDp, items...)
}

// ColumnWith располагает элементы вертикально с заданным выравниванием Align и распределением Justify.
func (u *UI) ColumnWith(align Align, justify Justify, gapDp int, items ...func()) {
	u.ctx.ColumnWith(align, justify, gapDp, items...)
}

// Grid располагает дочерние элементы в виде сетки из cols колонок.
func (u *UI) Grid(cols, gapXDp, gapYDp, rowHeightDp int, fn func()) {
	u.ctx.Grid(cols, gapXDp, gapYDp, rowHeightDp, fn)
}

// FormLayout создаёт табличную вёрстку формы «Подпись — Поле» с единой шириной колонки меток.
func (u *UI) FormLayout(labelWidthDp, gapXDp, gapYDp int, fn func()) {
	u.ctx.FormLayout(labelWidthDp, gapXDp, gapYDp, fn)
}

// FormRow рисует строку формы с подписью слева и полем справа.
func (u *UI) FormRow(label string, fn func()) {
	u.ctx.FormRow(label, fn)
}

// Flow располагает дочерние элементы слева направо с автоматическим переносом на новую строку.
func (u *UI) Flow(gapXDp, gapYDp int, fn func()) {
	u.ctx.Flow(gapXDp, gapYDp, fn)
}

// PushOverlay регистрирует отложенную отрисовку элемента поверх основного интерфейса.
// Прямоугольник bounds блокирует клики по виджетам под оверлеем.
func (u *UI) PushOverlay(bounds Rect, drawFn func()) {
	u.ctx.PushOverlay(bounds, drawFn)
}

// LayoutRect создаёт область раскладки (колонку без отступов) строго по заданному прямоугольнику rect
// с клиппингом содержимого. Удобно для размещения виджетов внутри PushOverlay.
func (u *UI) LayoutRect(rect Rect, fn func()) {
	u.ctx.LayoutRect(rect, fn)
}

// FocusID возвращает идентификатор виджета, находящегося в фокусе.
func (u *UI) FocusID() string {
	return u.ctx.FocusID
}

// SetFocus устанавливает фокус ввода на указанный виджет.
func (u *UI) SetFocus(id string) {
	u.ctx.FocusID = id
	u.ctx.RequestRedraw()
}

// Shortcut регистрирует глобальную горячую клавишу для текущего окна.
func (u *UI) Shortcut(keyCode uint32, mods Modifiers, fn func()) {
	u.ctx.RegisterShortcut(keyCode, mods, fn)
}

// DrawText рисует строку текста в произвольных физических координатах (для ListView и CanvasView).
func (u *UI) DrawText(x, y int, text string, sizeDp int, col Color) {
	u.ctx.DrawText(x, y, text, sizeDp, col)
}

// StaticLayer кеширует статический блок интерфейса (фон, неизменяемый текст) в отдельный Static Layer.
func (u *UI) StaticLayer(id string, fn func()) {
	u.ctx.StaticLayer(id, fn)
}

// InvalidateStatic помечает кеш статического слоя как устаревший и запрашивает перерисовку.
func (u *UI) InvalidateStatic() {
	u.ctx.InvalidateStatic()
}

// RequestRedraw запрашивает полный проход перерисовки UI в рамках текущего цикла Render
// (в частности, переключает Streaming fast-path на полную отрисовку drawFn).
func (u *UI) RequestRedraw() {
	if u != nil && u.ctx != nil {
		u.ctx.RequestRedraw()
	}
}

// SetCursor задаёт системный курсор для текущего кадра.
func (u *UI) SetCursor(cursor CursorType) {
	u.ctx.SetCursor(cursor)
}

// RegisterHoverArea регистрирует кастомную интерактивную область для умной фильтрации EventMouseMove.
func (u *UI) RegisterHoverArea(id string, rect Rect, cursor CursorType, trackMove bool) {
	if trackMove {
		u.ctx.RegisterHitTrack(id, rect, cursor)
	} else {
		u.ctx.RegisterHit(id, rect, cursor)
	}
}

// SetSmartMouseMove включает или отключает оптимизацию фильтрации EventMouseMove.
func (u *UI) SetSmartMouseMove(enabled bool) {
	u.ctx.SetSmartMouseMove(enabled)
}

// SkippedMouseMoves возвращает количество отфильтрованных событий EventMouseMove без перерисовки.
func (u *UI) SkippedMouseMoves() uint64 {
	return u.ctx.SkippedMouseMoves()
}

// LastFrameStaticBlitted возвращает true, если на последнем кадре статический слой был взят из кеша.
func (u *UI) LastFrameStaticBlitted() bool {
	return u.ctx.LastFrameStaticBlitted()
}

// LastDirtyRects возвращает список изменённых областей последнего выведенного кадра.
func (u *UI) LastDirtyRects() []Rect {
	return u.ctx.LastDirtyRects()
}

// SetDebugDirtyRect включает или отключает отображение отладочной рамки DirtyRect.
func (u *UI) SetDebugDirtyRect(enabled bool) {
	u.ctx.SetDebugDirtyRect(enabled)
}

// MousePos возвращает текущие физические координаты курсора мыши в окне.
func (u *UI) MousePos() (int, int) {
	return u.ctx.MouseX, u.ctx.MouseY
}

// GlyphCacheStats возвращает число записей и объём памяти (в байтах) LRU-кеша глифов.
func (u *UI) GlyphCacheStats() (entries int, bytes int64) {
	if u.ctx.Font == nil {
		return 0, 0
	}
	return u.ctx.Font.GlyphCacheStats()
}

// ClearGlyphCache сбрасывает LRU-кеш растеризованных глифов.
func (u *UI) ClearGlyphCache() {
	if u.ctx.Font != nil {
		u.ctx.Font.ClearGlyphCache()
		u.ctx.RequestRedraw()
	}
}

// ScaledImageCacheStats возвращает число записей и объём памяти (в байтах) LRU-кеша изображений.
func (u *UI) ScaledImageCacheStats() (entries int, bytes int64) {
	return render.ScaledImageCacheStats()
}

// ClearScaledImageCache сбрасывает LRU-кеш масштабированных изображений.
func (u *UI) ClearScaledImageCache() {
	render.ClearScaledImageCache()
	u.ctx.RequestRedraw()
}

