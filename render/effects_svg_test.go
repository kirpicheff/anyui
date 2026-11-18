package render

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func pixelAt(c *Canvas, x, y int) Color {
	off := (y*c.Width + x) * 4
	return Color{
		B: c.Pixels[off+0],
		G: c.Pixels[off+1],
		R: c.Pixels[off+2],
		A: c.Pixels[off+3],
	}
}

func TestCornerRadiiAndGradients(t *testing.T) {
	c := NewCanvas(100, 100)
	c.Clear(RGB(0, 0, 0))

	r := Rect{X: 10, Y: 10, W: 80, H: 40}
	cr := CornerRadii{TL: 16, TR: 0, BR: 16, BL: 0}
	c.FillLinearGradient(r, cr, LinearGradient{
		StartX: 0, StartY: 0,
		EndX: 1, EndY: 0,
		Stops: []GradientStop{
			{Pos: 0, Color: RGB(255, 0, 0)},
			{Pos: 1, Color: RGB(0, 0, 255)},
		},
	})

	// Верхний левый угол скруглён (самый угол должен остаться чёрным)
	if p := pixelAt(c, 10, 10); p.R != 0 || p.B != 0 {
		t.Fatalf("ожидался чёрный пиксель за скруглённым углом TL, получено %+v", p)
	}
	// Верхний правый угол острый (должен быть синим)
	if p := pixelAt(c, 89, 10); p.B < 200 {
		t.Fatalf("ожидался синий пиксель в остром углу TR, получено %+v", p)
	}
	// Центр должен быть смешанным
	mid := pixelAt(c, 50, 30)
	if mid.R < 100 || mid.B < 100 {
		t.Fatalf("ожидался фиолетовый градиент в центре, получено %+v", mid)
	}
}

func TestBoxShadowAndBackdropBlur(t *testing.T) {
	c := NewCanvas(80, 80)
	c.Clear(RGB(20, 20, 20))
	c.FillRect(Rect{X: 10, Y: 10, W: 20, H: 60}, RGB(250, 250, 250))

	// Размытие области на границе белого и тёмного должно усреднить цвета
	c.BackdropBlur(Rect{X: 15, Y: 15, W: 30, H: 30}, 4, UniformRadii(4))
	blurred := pixelAt(c, 30, 30)
	if blurred.R <= 20 || blurred.R >= 250 {
		t.Fatalf("ожидалось усреднение цвета после BackdropBlur, получено %+v", blurred)
	}

	// Тень вокруг прямоугольника должна изменить фон за его границами
	c.Clear(RGB(255, 255, 255))
	c.DrawBoxShadow(Rect{X: 20, Y: 20, W: 40, H: 40}, BoxShadow{
		OffsetY: 4,
		Blur:    10,
		Color:   RGBA(0, 0, 0, 128),
		Radii:   UniformRadii(6),
	})
	shPx := pixelAt(c, 40, 63)
	if shPx.R >= 255 {
		t.Fatalf("ожидалось затемнение от тени под прямоугольником, получено %+v", shPx)
	}
}

func TestVectorPrimitives(t *testing.T) {
	c := NewCanvas(60, 60)
	c.Clear(RGB(0, 0, 0))

	c.DrawLine(10, 10, 50, 10, 2, RGB(0, 255, 0))
	if p := pixelAt(c, 30, 10); p.G < 200 {
		t.Fatalf("ожидался зелёный пиксель на линии, получено %+v", p)
	}

	c.DrawArc(30, 35, 15, 0, math.Pi, 2, RGB(255, 0, 0))
	// Нижняя точка дуги при y = 35 + 15 = 50
	if p := pixelAt(c, 30, 50); p.R < 180 {
		t.Fatalf("ожидался красный пиксель на дуге, получено %+v", p)
	}
}

func TestSVGRasterizationAndCache(t *testing.T) {
	c := NewCanvas(48, 48)
	c.Clear(RGB(0, 0, 0))

	svg := `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 13l4 4L19 7"/></svg>`
	c.DrawSVG(svg, 12, 12, 24, RGB(0, 200, 255))

	lit := 0
	for y := 12; y < 36; y++ {
		for x := 12; x < 36; x++ {
			if pixelAt(c, x, y).B > 50 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Fatal("SVG-иконка не отрисовала ни одного пикселя")
	}
}

func TestDrawImageAndBGRA(t *testing.T) {
	c := NewCanvas(40, 40)
	c.Clear(RGB(0, 0, 0))

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 128, B: 0, A: 255})
		}
	}

	c.DrawImage(Rect{X: 10, Y: 10, W: 20, H: 20}, img, ScaleFit)
	p := pixelAt(c, 20, 20)
	if p.R < 240 || p.G < 110 || p.B > 10 {
		t.Fatalf("некорректный цвет после DrawImage, получено %+v", p)
	}
}

func TestSVGPathArcs(t *testing.T) {
	// Проверка абсолютной дуги и слитных флагов 01 в относительной дуге
	contours, closed := parseSVGPath("M4 12A8 8 0 0 1 20 12a4 4 0 01-4 4")
	if closed {
		t.Fatal("контур не должен быть замкнут")
	}
	if len(contours) != 1 {
		t.Fatalf("ожидался 1 контур, получено %d", len(contours))
	}
	pts := contours[0]
	if len(pts) < 10 {
		t.Fatalf("ожидалось разбиение дуг на точки, получено %d", len(pts))
	}
	last := pts[len(pts)-1]
	if math.Abs(float64(last.X-16)) > 0.05 || math.Abs(float64(last.Y-16)) > 0.05 {
		t.Fatalf("конечная точка дуги ожидалась (16,16), получено %+v", last)
	}

	// Вырожденная дуга (rx == 0) должна давать прямую линию до (15, 20)
	degContours, _ := parseSVGPath("M5 5A0 5 0 0 1 15 20")
	if len(degContours) != 1 || len(degContours[0]) != 2 {
		t.Fatalf("вырожденная дуга должна давать отрезок из 2 точек, получено %+v", degContours)
	}
	if degContours[0][1].X != 15 || degContours[0][1].Y != 20 {
		t.Fatalf("конечная точка вырожденной дуги ожидалась (15,20), получено %+v", degContours[0][1])
	}

	// Рендеринг иконки с дугой (верхняя полуокружность радиусом 8 вокруг (12, 12))
	c := NewCanvas(32, 32)
	c.Clear(RGB(0, 0, 0))
	svg := `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 12a8 8 0 0116 0"/></svg>`
	c.DrawSVG(svg, 4, 4, 24, RGB(255, 255, 255))
	// Верхняя точка дуги (12, 4) в viewBox -> (16, 8) на холсте
	if px := pixelAt(c, 16, 8); px.R < 100 {
		t.Fatalf("ожидался отрисованный пиксель в верхней точке дуги (16,8), получено %+v", px)
	}
}

