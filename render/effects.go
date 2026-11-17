package render

import "math"

// GradientStop задаёт цветовую точку градиента на позиции Pos в диапазоне [0..1].
type GradientStop struct {
	Pos   float32
	Color Color
}

// LinearGradient описывает линейный градиент в нормализованных координатах [0..1] прямоугольника.
type LinearGradient struct {
	StartX float32
	StartY float32
	EndX   float32
	EndY   float32
	Stops  []GradientStop
}

// RadialGradient описывает радиальный градиент в нормализованных координатах [0..1] прямоугольника.
type RadialGradient struct {
	CenterX float32
	CenterY float32
	Radius  float32
	Stops   []GradientStop
}

// BoxShadow описывает мягкую тень для прямоугольника со скруглением.
type BoxShadow struct {
	OffsetX float32
	OffsetY float32
	Blur    float32
	Spread  float32
	Color   Color
	Radii   CornerRadii
}

func sampleStops(stops []GradientStop, t float32) Color {
	if len(stops) == 0 {
		return Color{}
	}
	if len(stops) == 1 || t <= stops[0].Pos {
		return stops[0].Color
	}
	last := stops[len(stops)-1]
	if t >= last.Pos {
		return last.Color
	}
	for i := 0; i < len(stops)-1; i++ {
		a := stops[i]
		b := stops[i+1]
		if t >= a.Pos && t <= b.Pos {
			span := b.Pos - a.Pos
			if span <= 0 {
				return b.Color
			}
			return LerpColor(a.Color, b.Color, (t-a.Pos)/span)
		}
	}
	return last.Color
}

// FillLinearGradient заливает скруглённый прямоугольник линейным градиентом.
func (c *Canvas) FillLinearGradient(r Rect, cr CornerRadii, grad LinearGradient) {
	if r.Empty() || len(grad.Stops) == 0 {
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
	cx := float32(r.X) + hx
	cy := float32(r.Y) + hy

	sx := float32(r.X) + grad.StartX*float32(r.W)
	sy := float32(r.Y) + grad.StartY*float32(r.H)
	ex := float32(r.X) + grad.EndX*float32(r.W)
	ey := float32(r.Y) + grad.EndY*float32(r.H)

	vx := ex - sx
	vy := ey - sy
	lenSq := vx*vx + vy*vy

	for y := area.Y; y < area.Y+area.H; y++ {
		py := float32(y) + 0.5
		for x := area.X; x < area.X+area.W; x++ {
			px := float32(x) + 0.5
			dist := roundedBoxSDF(px, py, cx, cy, hx, hy, cr)
			cov := clamp01(0.5 - dist)
			if cov <= 0 {
				continue
			}

			var t float32
			if lenSq > 1e-6 {
				t = ((px-sx)*vx + (py-sy)*vy) / lenSq
			}
			col := sampleStops(grad.Stops, t)
			if col.A > 0 {
				c.blendPixelUnchecked(x, y, col, uint8(cov*255+0.5))
			}
		}
	}
}

// FillRadialGradient заливает скруглённый прямоугольник радиальным градиентом.
func (c *Canvas) FillRadialGradient(r Rect, cr CornerRadii, grad RadialGradient) {
	if r.Empty() || len(grad.Stops) == 0 {
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
	cx := float32(r.X) + hx
	cy := float32(r.Y) + hy

	gcx := float32(r.X) + grad.CenterX*float32(r.W)
	gcy := float32(r.Y) + grad.CenterY*float32(r.H)
	maxSide := float32(r.W)
	if float32(r.H) > maxSide {
		maxSide = float32(r.H)
	}
	rad := grad.Radius * maxSide
	if rad <= 1e-3 {
		rad = 1
	}

	for y := area.Y; y < area.Y+area.H; y++ {
		py := float32(y) + 0.5
		for x := area.X; x < area.X+area.W; x++ {
			px := float32(x) + 0.5
			dist := roundedBoxSDF(px, py, cx, cy, hx, hy, cr)
			cov := clamp01(0.5 - dist)
			if cov <= 0 {
				continue
			}

			t := float32(math.Hypot(float64(px-gcx), float64(py-gcy))) / rad
			col := sampleStops(grad.Stops, t)
			if col.A > 0 {
				c.blendPixelUnchecked(x, y, col, uint8(cov*255+0.5))
			}
		}
	}
}

// DrawBoxShadow рисует мягкую SDF-тень вокруг прямоугольника r.
func (c *Canvas) DrawBoxShadow(r Rect, sh BoxShadow) {
	if r.Empty() || sh.Color.A == 0 {
		return
	}
	blur := sh.Blur
	if blur < 0 {
		blur = 0
	}

	pad := int(math.Ceil(float64(blur + sh.Spread)))
	if pad < 1 {
		pad = 1
	}
	bounds := Rect{
		X: r.X + int(math.Round(float64(sh.OffsetX))) - pad,
		Y: r.Y + int(math.Round(float64(sh.OffsetY))) - pad,
		W: r.W + pad*2,
		H: r.H + pad*2,
	}
	area := c.CurrentClip().Intersect(bounds)
	if area.Empty() {
		return
	}
	c.AddDirtyRect(area)

	hx := float32(r.W)*0.5 + sh.Spread
	hy := float32(r.H)*0.5 + sh.Spread
	if hx < 0 {
		hx = 0
	}
	if hy < 0 {
		hy = 0
	}
	cr := CornerRadii{
		TL: sh.Radii.TL + sh.Spread,
		TR: sh.Radii.TR + sh.Spread,
		BR: sh.Radii.BR + sh.Spread,
		BL: sh.Radii.BL + sh.Spread,
	}
	cr = clampRadii(cr, hx, hy)

	cx := float32(r.X) + float32(r.W)*0.5 + sh.OffsetX
	cy := float32(r.Y) + float32(r.H)*0.5 + sh.OffsetY

	halfBlur := blur * 0.5
	if halfBlur < 0.5 {
		halfBlur = 0.5
	}

	for y := area.Y; y < area.Y+area.H; y++ {
		py := float32(y) + 0.5
		for x := area.X; x < area.X+area.W; x++ {
			px := float32(x) + 0.5
			dist := roundedBoxSDF(px, py, cx, cy, hx, hy, cr)
			if dist >= halfBlur {
				continue
			}

			var alpha float32
			if dist <= -halfBlur {
				alpha = 1
			} else {
				// Плавное затухание тени (smoothstep)
				u := clamp01((halfBlur - dist) / (2 * halfBlur))
				alpha = u * u * (3 - 2*u)
			}
			if alpha > 0 {
				c.blendPixelUnchecked(x, y, sh.Color, uint8(alpha*255+0.5))
			}
		}
	}
}

// BackdropBlur выполняет быстрое многопроходное размытие фона под областью r (Glassmorphism).
func (c *Canvas) BackdropBlur(r Rect, radius int, cr CornerRadii) {
	if radius <= 0 || r.Empty() {
		return
	}
	area := c.CurrentClip().Intersect(r)
	if area.Empty() {
		return
	}
	c.AddDirtyRect(area)

	w, h := area.W, area.H
	bufA := make([]byte, w*h*4)
	bufB := make([]byte, w*h*4)

	for y := 0; y < h; y++ {
		srcOff := ((area.Y+y)*c.Width + area.X) * 4
		copy(bufA[y*w*4:(y+1)*w*4], c.Pixels[srcOff:srcOff+w*4])
	}

	// 2 прохода box-blur дают визуально гладкий результат, близкий к гауссовому размытию
	for pass := 0; pass < 2; pass++ {
		boxBlurH(bufA, bufB, w, h, radius)
		boxBlurV(bufB, bufA, w, h, radius)
	}

	hx := float32(r.W) * 0.5
	hy := float32(r.H) * 0.5
	cr = clampRadii(cr, hx, hy)
	cx := float32(r.X) + hx
	cy := float32(r.Y) + hy
	hasCorners := cr.TL > 0 || cr.TR > 0 || cr.BR > 0 || cr.BL > 0

	for y := 0; y < h; y++ {
		py := float32(area.Y+y) + 0.5
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			col := Color{
				B: bufA[off+0],
				G: bufA[off+1],
				R: bufA[off+2],
				A: 255,
			}
			if !hasCorners {
				dstOff := ((area.Y+y)*c.Width + (area.X + x)) * 4
				c.Pixels[dstOff+0] = col.B
				c.Pixels[dstOff+1] = col.G
				c.Pixels[dstOff+2] = col.R
				c.Pixels[dstOff+3] = 255
				continue
			}

			dist := roundedBoxSDF(float32(area.X+x)+0.5, py, cx, cy, hx, hy, cr)
			cov := clamp01(0.5 - dist)
			if cov > 0 {
				c.blendPixelUnchecked(area.X+x, area.Y+y, col, uint8(cov*255+0.5))
			}
		}
	}
}

func boxBlurH(src, dst []byte, w, h, r int) {
	div := 2*r + 1
	for y := 0; y < h; y++ {
		row := y * w * 4
		var sumB, sumG, sumR int
		for k := -r; k <= r; k++ {
			kx := k
			if kx < 0 {
				kx = 0
			} else if kx >= w {
				kx = w - 1
			}
			off := row + kx*4
			sumB += int(src[off+0])
			sumG += int(src[off+1])
			sumR += int(src[off+2])
		}
		for x := 0; x < w; x++ {
			out := row + x*4
			dst[out+0] = uint8(sumB / div)
			dst[out+1] = uint8(sumG / div)
			dst[out+2] = uint8(sumR / div)
			dst[out+3] = 255

			xAdd := x + r + 1
			if xAdd >= w {
				xAdd = w - 1
			}
			xSub := x - r
			if xSub < 0 {
				xSub = 0
			}
			addOff := row + xAdd*4
			subOff := row + xSub*4
			sumB += int(src[addOff+0]) - int(src[subOff+0])
			sumG += int(src[addOff+1]) - int(src[subOff+1])
			sumR += int(src[addOff+2]) - int(src[subOff+2])
		}
	}
}

func boxBlurV(src, dst []byte, w, h, r int) {
	div := 2*r + 1
	for x := 0; x < w; x++ {
		var sumB, sumG, sumR int
		for k := -r; k <= r; k++ {
			ky := k
			if ky < 0 {
				ky = 0
			} else if ky >= h {
				ky = h - 1
			}
			off := (ky*w + x) * 4
			sumB += int(src[off+0])
			sumG += int(src[off+1])
			sumR += int(src[off+2])
		}
		for y := 0; y < h; y++ {
			out := (y*w + x) * 4
			dst[out+0] = uint8(sumB / div)
			dst[out+1] = uint8(sumG / div)
			dst[out+2] = uint8(sumR / div)
			dst[out+3] = 255

			yAdd := y + r + 1
			if yAdd >= h {
				yAdd = h - 1
			}
			ySub := y - r
			if ySub < 0 {
				ySub = 0
			}
			addOff := (yAdd*w + x) * 4
			subOff := (ySub*w + x) * 4
			sumB += int(src[addOff+0]) - int(src[subOff+0])
			sumG += int(src[addOff+1]) - int(src[subOff+1])
			sumR += int(src[addOff+2]) - int(src[subOff+2])
		}
	}
}

