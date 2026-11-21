package anyui

import (
	"errors"
	"image"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
	"github.com/kirpicheff/anyui/ui"
)

// Modifiers хранит состояние модификаторов клавиатуры для горячих клавиш.
type Modifiers = platform.Modifiers

// KeyEvent описывает сырое событие нажатия или отпускания клавиши для InteractiveCanvas.
type KeyEvent = ui.RawKeyEvent

// Key задаёт виртуальный код клавиши клавиатуры.
type Key = uint32

const (
	KeyBackspace Key = ui.KeyBackspace
	KeyTab       Key = ui.KeyTab
	KeyEnter     Key = ui.KeyEnter
	KeyEscape    Key = ui.KeyEscape
	KeySpace     Key = ui.KeySpace
	KeyPageUp    Key = ui.KeyPageUp
	KeyPageDown  Key = ui.KeyPageDown
	KeyEnd       Key = ui.KeyEnd
	KeyHome      Key = ui.KeyHome
	KeyLeft      Key = ui.KeyLeft
	KeyUp        Key = ui.KeyUp
	KeyRight     Key = ui.KeyRight
	KeyDown      Key = ui.KeyDown
	KeyDelete    Key = ui.KeyDelete
	KeyA         Key = ui.KeyA
	KeyC         Key = ui.KeyC
	KeyV         Key = ui.KeyV
	KeyX         Key = ui.KeyX
	KeyY         Key = ui.KeyY
	KeyZ         Key = ui.KeyZ
)

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

// ClipboardText возвращает текст из системного буфера обмена ОС.
func (u *UI) ClipboardText() (string, error) {
	if u == nil {
		return "", errors.New("контекст не инициализирован")
	}
	if u.win != nil {
		return u.win.ClipboardText()
	}
	if u.ctx != nil && u.ctx.Win != nil {
		return u.ctx.Win.ClipboardText()
	}
	return "", errors.New("окно не инициализировано")
}

// SetClipboardText записывает текст в системный буфер обмена ОС.
func (u *UI) SetClipboardText(text string) error {
	if u == nil {
		return errors.New("контекст не инициализирован")
	}
	if u.win != nil {
		return u.win.SetClipboardText(text)
	}
	if u.ctx != nil && u.ctx.Win != nil {
		return u.ctx.Win.SetClipboardText(text)
	}
	return errors.New("окно не инициализировано")
}

// App возвращает экземпляр приложения.
func (u *UI) App() *App {
	return u.app
}

// Canvas возвращает низкоуровневый холст окна для кастомного рисования.
func (u *UI) Canvas() *Canvas {
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

// Scale переводит логические пиксели (Dp) в физические (Px) с учётом DPI-масштаба.
func (u *UI) Scale(dp Dp) Px {
	if u.ctx == nil {
		return 0
	}
	return u.ctx.Scale(dp)
}

// ScaleF переводит дробные логические пиксели (dp) в физические.
func (u *UI) ScaleF(dp float32) float32 {
	if u.ctx == nil {
		return 0
	}
	return u.ctx.ScaleF(dp)
}

// PxToDp переводит физические пиксели (Px) в логические (Dp) с округлением до ближайшего целого.
func (u *UI) PxToDp(px Px) Dp {
	if u.ctx == nil {
		return 0
	}
	return u.ctx.PxToDp(px)
}

// PxToDpf переводит физические пиксели (px) в дробные логические пиксели (dp).
func (u *UI) PxToDpf(px float32) float32 {
	if u.ctx == nil {
		return 0
	}
	return u.ctx.PxToDpf(px)
}

// CurrentArea возвращает прямоугольник текущего контейнера в логических пикселях (Dp).
func (u *UI) CurrentArea() Rect {
	if u.ctx == nil {
		return Rect{}
	}
	return u.ctx.CurrentAreaDp()
}

// CurrentAreaPx возвращает прямоугольник текущего контейнера в физических пикселях (Px).
func (u *UI) CurrentAreaPx() Rect {
	if u.ctx == nil {
		return Rect{}
	}
	return u.ctx.CurrentArea()
}

// Label выводит строку текста с опциональной стилизацией.
func (u *UI) Label(text string, opts ...Option) {
	u.ctx.LabelOpts(text, opts...)
}

// StatusDot рисует цветной кружок-индикатор и подпись справа от него.
func (u *UI) StatusDot(dotColor Color, label string, opts ...Option) {
	u.ctx.StatusDot(dotColor, label, opts...)
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

// Image выводит изображение image.Image в выделенной области.
// Размеры задаются через опции Width, Height, FillWidth, FillHeight (по умолчанию — доступное пространство).
func (u *UI) Image(img image.Image, mode ImageScaleMode, opts ...Option) {
	u.ctx.Image(img, mode, opts...)
}

// CanvasView выделяет область для кастомного рисования (графики, видеокадры, градиенты).
// Размеры задаются через опции Width, Height, FillWidth, FillHeight (по умолчанию — доступное пространство).
// Колбэк drawFn получает координаты области areaPx строго в физических пикселях (Px) для прямого вывода в Canvas.
func (u *UI) CanvasView(drawFn func(canvas *Canvas, areaPx Rect), opts ...Option) {
	u.ctx.CanvasView(drawFn, opts...)
}

// InteractiveCanvas выделяет интерактивную область с поддержкой фокуса,
// захвата перетаскивания мыши и передачи сырых событий клавиатуры (со ScanCode/Extended) и мыши/колеса.
// Размеры задаются через опции Width, Height, FillWidth, FillHeight (по умолчанию — доступное пространство).
// Методы cb (включая OnDraw) работают с областью area в физических пикселях (Px).
func (u *UI) InteractiveCanvas(id string, cb CanvasCallbacks, opts ...Option) {
	u.ctx.InteractiveCanvas(id, cb, opts...)
}

// DrawSVG рисует SVG-иконку в физических координатах (xPx, yPx) с размером sizeDp и цветом tint.
func (u *UI) DrawSVG(svgData string, xPx, yPx int, sizeDp Dp, tint Color) {
	u.ctx.Canvas.DrawSVG(svgData, xPx, yPx, u.ctx.Scale(sizeDp).Int(), tint)
}

// Column располагает дочерние элементы вертикально с отступом gapDp.
func (u *UI) Column(gapDp Dp, fn func()) {
	u.ctx.Column(gapDp, fn)
}

// ColumnCentered располагает дочерние элементы вертикально и центрирует их по горизонтали.
func (u *UI) ColumnCentered(gapDp Dp, fn func()) {
	u.ctx.ColumnCentered(gapDp, fn)
}

// Row располагает дочерние элементы горизонтально с отступом gapDp и высотой heightDp.
func (u *UI) Row(gapDp, heightDp Dp, fn func()) {
	u.ctx.Row(gapDp, heightDp, fn)
}

// RowCentered располагает дочерние элементы горизонтально с выравниванием по вертикальному центру.
func (u *UI) RowCentered(gapDp, heightDp Dp, fn func()) {
	u.ctx.RowCentered(gapDp, heightDp, fn)
}

// Padding добавляет внутренние отступы со всех сторон.
func (u *UI) Padding(padDp Dp, fn func()) {
	u.ctx.Padding(padDp, fn)
}

// PaddingXY добавляет горизонтальный и вертикальный внутренние отступы.
func (u *UI) PaddingXY(pxDp, pyDp Dp, fn func()) {
	u.ctx.PaddingXY(pxDp, pyDp, fn)
}

// Spacer сдвигает курсор раскладки на указанное количество логических пикселей.
func (u *UI) Spacer(dp Dp) {
	u.ctx.Spacer(dp)
}

// FlexSpacer заполняет оставшееся пространство с резервом reserveDp.
func (u *UI) FlexSpacer(reserveDp Dp) {
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
func (u *UI) RowFlex(gapDp, heightDp Dp, items ...FlexItem) {
	u.ctx.RowFlex(gapDp, heightDp, items...)
}

// ColumnFlex располагает элементы вертикально, деля высоту пропорционально весам FlexItem.
func (u *UI) ColumnFlex(gapDp Dp, items ...FlexItem) {
	u.ctx.ColumnFlex(gapDp, items...)
}

// DockBottom резервирует bottomDp снизу для bottomFn, а всю оставшуюся высоту сверху отдаёт bodyFn.
func (u *UI) DockBottom(gapDp, bottomDp Dp, bodyFn, bottomFn func()) {
	u.ctx.DockBottom(gapDp, bottomDp, bodyFn, bottomFn)
}

// DockTop резервирует topDp сверху для topFn, а всю оставшуюся высоту снизу отдаёт bodyFn.
func (u *UI) DockTop(gapDp, topDp Dp, topFn, bodyFn func()) {
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
func (u *UI) RowWith(align Align, justify Justify, gapDp, heightDp Dp, items ...func()) {
	u.ctx.RowWith(align, justify, gapDp, heightDp, items...)
}

// ColumnWith располагает элементы вертикально с заданным выравниванием Align и распределением Justify.
func (u *UI) ColumnWith(align Align, justify Justify, gapDp Dp, items ...func()) {
	u.ctx.ColumnWith(align, justify, gapDp, items...)
}

// Grid располагает дочерние элементы в виде сетки из cols колонок.
func (u *UI) Grid(cols int, gapXDp, gapYDp, rowHeightDp Dp, fn func()) {
	u.ctx.Grid(cols, gapXDp, gapYDp, rowHeightDp, fn)
}

// FormLayout создаёт табличную вёрстку формы «Подпись — Поле» с единой шириной колонки меток.
func (u *UI) FormLayout(labelWidthDp, gapXDp, gapYDp Dp, fn func()) {
	u.ctx.FormLayout(labelWidthDp, gapXDp, gapYDp, fn)
}

// FormRow рисует строку формы с подписью слева и полем справа.
func (u *UI) FormRow(label string, fn func()) {
	u.ctx.FormRow(label, fn)
}

// Flow располагает дочерние элементы слева направо с автоматическим переносом на новую строку.
func (u *UI) Flow(gapXDp, gapYDp Dp, fn func()) {
	u.ctx.Flow(gapXDp, gapYDp, fn)
}

func (u *UI) scaleRect(r Rect) render.Rect {
	if r.Empty() || u.ctx == nil {
		return render.Rect{}
	}
	return render.Rect{
		X: u.ctx.Scale(Dp(r.X)).Int(),
		Y: u.ctx.Scale(Dp(r.Y)).Int(),
		W: u.ctx.Scale(Dp(r.W)).Int(),
		H: u.ctx.Scale(Dp(r.H)).Int(),
	}
}

// PushOverlay регистрирует отложенную отрисовку элемента поверх основного интерфейса.
// Прямоугольник bounds задаётся в логических пикселях (Dp / CurrentArea()) и блокирует клики по виджетам под оверлеем.
func (u *UI) PushOverlay(bounds Rect, drawFn func()) {
	if u.ctx == nil {
		return
	}
	u.ctx.PushOverlay(u.scaleRect(bounds), drawFn)
}

// LayoutRect создаёт область раскладки (колонку без отступов) строго по заданному прямоугольнику rect
// (в логических пикселях Dp / CurrentArea()) с клиппингом содержимого. Удобно для размещения виджетов внутри PushOverlay.
func (u *UI) LayoutRect(rect Rect, fn func()) {
	if u.ctx == nil {
		return
	}
	u.ctx.LayoutRect(u.scaleRect(rect), fn)
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
func (u *UI) Shortcut(key Key, mods Modifiers, fn func()) {
	u.ctx.RegisterShortcut(key, mods, fn)
}

// DrawText рисует строку текста в физических координатах (xPx, yPx) с размером sizeDp и цветом col.
func (u *UI) DrawText(xPx, yPx int, text string, sizeDp Dp, col Color) {
	u.ctx.DrawText(xPx, yPx, text, sizeDp, col)
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

// RegisterHoverArea регистрирует кастомную интерактивную область rect (в логических пикселях Dp / CurrentArea())
// для смены курсора мыши и умной фильтрации перемещений.
func (u *UI) RegisterHoverArea(id string, rect Rect, cursor CursorType, trackMove bool) {
	if u.ctx == nil {
		return
	}
	rPx := u.scaleRect(rect)
	if trackMove {
		u.ctx.RegisterHitTrack(id, rPx, cursor)
	} else {
		u.ctx.RegisterHit(id, rPx, cursor)
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

// LastDirtyRectsPx возвращает список изменённых областей последнего выведенного кадра в физических пикселях окна (Px).
// Внимание: метод возвращает физические пиксели (Px) устройства. Не передавайте их напрямую в методы раскладки и оверлеев (LayoutRect, PushOverlay и др.), работающие строго в Dp.
func (u *UI) LastDirtyRectsPx() []Rect {
	return u.ctx.LastDirtyRects()
}

// SetDebugDirtyRect включает или отключает отображение отладочной рамки DirtyRect.
func (u *UI) SetDebugDirtyRect(enabled bool) {
	u.ctx.SetDebugDirtyRect(enabled)
}

// MousePosPx возвращает текущие физические координаты курсора мыши в окне (в Px).
func (u *UI) MousePosPx() (Px, Px) {
	if u.ctx == nil {
		return 0, 0
	}
	return Px(u.ctx.MouseX), Px(u.ctx.MouseY)
}

// MousePosDp возвращает текущие логические координаты курсора мыши в окне (в dp).
func (u *UI) MousePosDp() (Dp, Dp) {
	if u.ctx == nil {
		return 0, 0
	}
	return u.ctx.PxToDp(Px(u.ctx.MouseX)), u.ctx.PxToDp(Px(u.ctx.MouseY))
}

// MousePressed возвращает true, если в текущем кадре нажата левая кнопка мыши.
func (u *UI) MousePressed() bool {
	return u.ctx.MousePressed
}

// MouseDown возвращает true, если левая кнопка мыши зажата.
func (u *UI) MouseDown() bool {
	return u.ctx.MouseDown
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
