package anyui

import (
	"image"

	"anyui/platform"
	"anyui/render"
	"anyui/ui"
)

// Modifiers хранит состояние модификаторов клавиатуры для горячих клавиш.
type Modifiers = platform.Modifiers

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

// SetTheme меняет тему текущего окна и запрашивает перерисовку.
func (u *UI) SetTheme(theme Theme) {
	u.ctx.Theme = theme
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

// PushOverlay регистрирует отложенную отрисовку элемента поверх основного интерфейса.
// Прямоугольник bounds блокирует клики по виджетам под оверлеем.
func (u *UI) PushOverlay(bounds Rect, drawFn func()) {
	u.ctx.PushOverlay(bounds, drawFn)
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

