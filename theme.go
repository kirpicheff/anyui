package anyui

import (
	"github.com/kirpicheff/anyui/render"
	"github.com/kirpicheff/anyui/ui"
)

// Color описывает цвет в формате RGBA.
type Color = render.Color

// Canvas представляет программный 2D-холст для кастомного рисования.
type Canvas = render.Canvas

// Rect описывает прямоугольник в пиксельных координатах.
type Rect = render.Rect

// Dp задаёт логические аппаратно-независимые пиксели (Density-independent Pixels).
type Dp = ui.Dp

// Px задаёт физические пиксели буфера отрисовки (Device Pixels).
type Px = ui.Px

// Theme хранит цвета и базовые радиусы оформления интерфейса.
type Theme = ui.Theme

// CornerRadii задаёт независимые радиусы скругления углов прямоугольника.
type CornerRadii = render.CornerRadii

// GradientStop задаёт цветовую точку градиента.
type GradientStop = render.GradientStop

// LinearGradient описывает линейный градиент.
type LinearGradient = render.LinearGradient

// RadialGradient описывает радиальный градиент.
type RadialGradient = render.RadialGradient

// BoxShadow описывает мягкую тень прямоугольника.
type BoxShadow = render.BoxShadow

// PointF описывает точку с вещественными координатами.
type PointF = render.PointF

// ImageScaleMode задаёт режим масштабирования изображений.
type ImageScaleMode = render.ImageScaleMode

const (
	ScaleFit      = render.ScaleFit
	ScaleFill     = render.ScaleFill
	ScaleOneToOne = render.ScaleOneToOne
)

// FontStyle задаёт начертание шрифта.
type FontStyle = render.FontStyle

const (
	FontRegular = render.FontRegular
	FontBold    = render.FontBold
	FontMono    = render.FontMono
)

// UniformRadii возвращает CornerRadii с одинаковым радиусом для всех углов.
func UniformRadii(r float32) CornerRadii {
	return render.UniformRadii(r)
}

// RGB создаёт непрозрачный цвет из компонентов R, G, B.
func RGB(r, g, b uint8) Color {
	return render.RGB(r, g, b)
}

// RGBA создаёт цвет из компонентов R, G, B, A.
func RGBA(r, g, b, a uint8) Color {
	return render.RGBA(r, g, b, a)
}

// Hex создаёт непрозрачный цвет из 24-битного шестнадцатеричного значения 0xRRGGBB.
func Hex(rgb uint32) Color {
	return render.Hex(rgb)
}

// NavyTheme возвращает стандартную тёмно-синюю тему.
func NavyTheme() Theme {
	return ui.NavyTheme()
}

// DarkTheme возвращает нейтральную тёмную тему.
func DarkTheme() Theme {
	return ui.DarkTheme()
}

// LightTheme возвращает светлую тему.
func LightTheme() Theme {
	return ui.LightTheme()
}

