package render

import "math"

// DrawRoundedBox рисует скруглённый прямоугольник с заливкой и обводкой за один проход.
// Смешивание заливки и рамки до наложения на фон исключает артефакты на стыке скруглений.
func (c *Canvas) DrawRoundedBox(r Rect, radius float32, fill Color, borderW float32, borderCol Color) {
	if r.Empty() {
		return
	}
	if fill.A == 0 && (borderW <= 0 || borderCol.A == 0) {
		return
	}

	// Ограничиваем область рисования текущим клипом (+1 пиксель запаса под сглаживание)
	area := c.CurrentClip().Intersect(r)
	if area.Empty() {
		return
	}
	c.AddDirtyRect(area)

	hx := float32(r.W) * 0.5
	hy := float32(r.H) * 0.5
	maxR := hx
	if hy < maxR {
		maxR = hy
	}
	if radius < 0 {
		radius = 0
	} else if radius > maxR {
		radius = maxR
	}

	if borderW < 0 {
		borderW = 0
	}

	cx := float32(r.X) + hx
	cy := float32(r.Y) + hy
	bx := hx - radius
	by := hy - radius

	// Внутренний прямоугольник, гарантированно свободный от углов и рамки, заливаем быстрым FillRect
	margin := int(math.Ceil(float64(radius + borderW + 1)))
	inner := Rect{
		X: r.X + margin,
		Y: r.Y + margin,
		W: r.W - margin*2,
		H: r.H - margin*2,
	}.Intersect(area)

	hasInner := !inner.Empty()
	if hasInner && fill.A > 0 {
		c.FillRect(inner, fill)
	}

	hasBorder := borderW > 0 && borderCol.A > 0

	for y := area.Y; y < area.Y+area.H; y++ {
		py := float32(y) + 0.5
		dy := float32(math.Abs(float64(py-cy))) - by
		inInnerY := hasInner && y >= inner.Y && y < inner.Y+inner.H

		for x := area.X; x < area.X+area.W; x++ {
			if inInnerY && x >= inner.X && x < inner.X+inner.W {
				// Пропускаем уже залитую середину
				x = inner.X + inner.W - 1
				continue
			}

			px := float32(x) + 0.5
			dx := float32(math.Abs(float64(px-cx))) - bx

			var dist float32
			if dx > 0 && dy > 0 {
				dist = float32(math.Hypot(float64(dx), float64(dy))) - radius
			} else {
				m := dx
				if dy > m {
					m = dy
				}
				dist = m - radius
			}

			// Внешнее покрытие фигуры (0..1)
			outerCov := clamp01(0.5 - dist)
			if outerCov <= 0 {
				continue
			}

			if !hasBorder {
				if fill.A > 0 {
					c.blendPixelUnchecked(x, y, fill, uint8(outerCov*255+0.5))
				}
				continue
			}

			// Внутреннее покрытие заливки (внутри рамки)
			innerCov := clamp01(0.5 - (dist + borderW))
			borderCov := outerCov - innerCov
			if borderCov < 0 {
				borderCov = 0
			}

			if fill.A > 0 && innerCov > 0 {
				c.blendPixelUnchecked(x, y, fill, uint8(innerCov*255+0.5))
			}
			if borderCov > 0 {
				c.blendPixelUnchecked(x, y, borderCol, uint8(borderCov*255+0.5))
			}
		}
	}
}

// FillRoundedRect рисует скруглённый прямоугольник без обводки.
func (c *Canvas) FillRoundedRect(r Rect, radius float32, fill Color) {
	c.DrawRoundedBox(r, radius, fill, 0, Color{})
}

// FillCircle рисует сглаженный круг с центром (cx, cy) и заданным радиусом.
func (c *Canvas) FillCircle(cx, cy, radius float32, col Color) {
	if radius <= 0 || col.A == 0 {
		return
	}
	bounds := Rect{
		X: int(math.Floor(float64(cx - radius - 1))),
		Y: int(math.Floor(float64(cy - radius - 1))),
		W: int(math.Ceil(float64(radius*2 + 2))),
		H: int(math.Ceil(float64(radius*2 + 2))),
	}
	area := c.CurrentClip().Intersect(bounds)
	if area.Empty() {
		return
	}
	c.AddDirtyRect(area)

	for y := area.Y; y < area.Y+area.H; y++ {
		dy := float32(y) + 0.5 - cy
		for x := area.X; x < area.X+area.W; x++ {
			dx := float32(x) + 0.5 - cx
			dist := float32(math.Hypot(float64(dx), float64(dy))) - radius
			cov := clamp01(0.5 - dist)
			if cov > 0 {
				c.blendPixelUnchecked(x, y, col, uint8(cov*255+0.5))
			}
		}
	}
}

// CornerRadii задаёт независимые радиусы скругления для каждого угла прямоугольника.
type CornerRadii struct {
	TL float32
	TR float32
	BR float32
	BL float32
}

// UniformRadii возвращает CornerRadii с одинаковым радиусом для всех углов.
func UniformRadii(r float32) CornerRadii {
	return CornerRadii{TL: r, TR: r, BR: r, BL: r}
}

// clampRadii ограничивает радиусы половиной меньшей стороны прямоугольника.
func clampRadii(cr CornerRadii, hx, hy float32) CornerRadii {
	maxR := hx
	if hy < maxR {
		maxR = hy
	}
	clamp := func(v float32) float32 {
		if v < 0 {
			return 0
		}
		if v > maxR {
			return maxR
		}
		return v
	}
	return CornerRadii{
		TL: clamp(cr.TL),
		TR: clamp(cr.TR),
		BR: clamp(cr.BR),
		BL: clamp(cr.BL),
	}
}

// roundedBoxSDF вычисляет знаковое расстояние от точки (px, py) до границ прямоугольника со скруглёнными углами.
func roundedBoxSDF(px, py, cx, cy, hx, hy float32, cr CornerRadii) float32 {
	rx := px - cx
	ry := py - cy
	var rad float32
	if rx >= 0 {
		if ry >= 0 {
			rad = cr.BR
		} else {
			rad = cr.TR
		}
	} else {
		if ry >= 0 {
			rad = cr.BL
		} else {
			rad = cr.TL
		}
	}

	dx := float32(math.Abs(float64(rx))) - (hx - rad)
	dy := float32(math.Abs(float64(ry))) - (hy - rad)
	if dx > 0 && dy > 0 {
		return float32(math.Hypot(float64(dx), float64(dy))) - rad
	}
	m := dx
	if dy > m {
		m = dy
	}
	return m - rad
}

// DrawRoundedBoxCorners рисует прямоугольник с разными радиусами по углам, заливкой и обводкой.
func (c *Canvas) DrawRoundedBoxCorners(r Rect, cr CornerRadii, fill Color, borderW float32, borderCol Color) {
	if r.Empty() {
		return
	}
	if fill.A == 0 && (borderW <= 0 || borderCol.A == 0) {
		return
	}
	area := c.CurrentClip().Intersect(r)
	if area.Empty() {
		return
	}
	c.AddDirtyRect(area)

	hx := float32(r.W) * 0.5
	hy := float32(r.H) * 0.5
	cr = clampRadii(cr, hx, hy)
	if borderW < 0 {
		borderW = 0
	}

	maxR := cr.TL
	for _, v := range []float32{cr.TR, cr.BR, cr.BL} {
		if v > maxR {
			maxR = v
		}
	}

	cx := float32(r.X) + hx
	cy := float32(r.Y) + hy

	margin := int(math.Ceil(float64(maxR + borderW + 1)))
	inner := Rect{
		X: r.X + margin,
		Y: r.Y + margin,
		W: r.W - margin*2,
		H: r.H - margin*2,
	}.Intersect(area)

	hasInner := !inner.Empty()
	if hasInner && fill.A > 0 {
		c.FillRect(inner, fill)
	}

	hasBorder := borderW > 0 && borderCol.A > 0

	for y := area.Y; y < area.Y+area.H; y++ {
		py := float32(y) + 0.5
		inInnerY := hasInner && y >= inner.Y && y < inner.Y+inner.H

		for x := area.X; x < area.X+area.W; x++ {
			if inInnerY && x >= inner.X && x < inner.X+inner.W {
				x = inner.X + inner.W - 1
				continue
			}

			dist := roundedBoxSDF(float32(x)+0.5, py, cx, cy, hx, hy, cr)
			outerCov := clamp01(0.5 - dist)
			if outerCov <= 0 {
				continue
			}

			if !hasBorder {
				if fill.A > 0 {
					c.blendPixelUnchecked(x, y, fill, uint8(outerCov*255+0.5))
				}
				continue
			}

			innerCov := clamp01(0.5 - (dist + borderW))
			borderCov := outerCov - innerCov
			if borderCov < 0 {
				borderCov = 0
			}

			if fill.A > 0 && innerCov > 0 {
				c.blendPixelUnchecked(x, y, fill, uint8(innerCov*255+0.5))
			}
			if borderCov > 0 {
				c.blendPixelUnchecked(x, y, borderCol, uint8(borderCov*255+0.5))
			}
		}
	}
}

// FillRoundedRectCorners заливает прямоугольник с независимыми радиусами углов.
func (c *Canvas) FillRoundedRectCorners(r Rect, cr CornerRadii, fill Color) {
	c.DrawRoundedBoxCorners(r, cr, fill, 0, Color{})
}

func clamp01(v float32) float32 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v
}

