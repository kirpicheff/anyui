package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"anyui/platform"
	"anyui/render"
)

// NumberInput рисует числовое поле с кнопками '−' и '+' (Stepper) и возможностью прямого ввода.
func (c *Context) NumberInput(id string, value *float64, minVal, maxVal, step float64, decimals int, opts ...Option) bool {
	if value == nil {
		return false
	}
	if step <= 0 {
		step = 1
	}
	if decimals < 0 {
		decimals = 0
	}
	if minVal > maxVal {
		minVal, maxVal = maxVal, minVal
	}
	clampVal := func(v float64) float64 {
		if v < minVal {
			return minVal
		}
		if v > maxVal {
			return maxVal
		}
		return v
	}

	st := ApplyOptions(opts)
	w := c.S(160)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(34)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}

	rect := c.allocRect(w, h)
	radius := c.Sf(c.Theme.RadiusSm)
	if st.Radius != nil {
		radius = c.Sf(*st.Radius)
	}

	btnW := h
	if btnW > w/3 {
		btnW = w / 3
	}
	minusR := render.Rect{X: rect.X, Y: rect.Y, W: btnW, H: rect.H}
	plusR := render.Rect{X: rect.X + rect.W - btnW, Y: rect.Y, W: btnW, H: rect.H}
	midR := render.Rect{X: rect.X + btnW, Y: rect.Y, W: rect.W - btnW*2, H: rect.H}
	if !st.Disabled {
		c.RegisterHit(id+":minus", minusR, platform.CursorHand)
		c.RegisterHit(id+":plus", plusR, platform.CursorHand)
	}

	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, c.Theme.Border)

	blocked := c.IsInputBlocked()
	changed := false

	// Кнопка '−'
	minusHover := !blocked && !st.Disabled && minusR.Contains(c.MouseX, c.MouseY)
	if minusHover {
		c.nextCursor = platform.CursorHand
		c.Canvas.FillRoundedRectCorners(minusR, render.CornerRadii{TL: radius, BL: radius}, c.Theme.Surface)
		if c.MousePressed {
			nv := clampVal(*value - step)
			if nv != *value {
				*value = nv
				changed = true
				c.RequestRedraw()
			}
		}
	}
	c.Canvas.DrawLine(float32(minusR.X+minusR.W), float32(rect.Y+2), float32(minusR.X+minusR.W), float32(rect.Y+rect.H-2), 1.0, c.Theme.Border)
	mx := float32(minusR.X + minusR.W/2)
	my := float32(minusR.Y + minusR.H/2)
	c.Canvas.DrawLine(mx-c.Sf(5), my, mx+c.Sf(5), my, c.Sf(1.8), c.Theme.TextPrimary)

	// Кнопка '+'
	plusHover := !blocked && !st.Disabled && plusR.Contains(c.MouseX, c.MouseY)
	if plusHover {
		c.nextCursor = platform.CursorHand
		c.Canvas.FillRoundedRectCorners(plusR, render.CornerRadii{TR: radius, BR: radius}, c.Theme.Surface)
		if c.MousePressed {
			nv := clampVal(*value + step)
			if nv != *value {
				*value = nv
				changed = true
				c.RequestRedraw()
			}
		}
	}
	c.Canvas.DrawLine(float32(plusR.X), float32(rect.Y+2), float32(plusR.X), float32(rect.Y+rect.H-2), 1.0, c.Theme.Border)
	px := float32(plusR.X + plusR.W/2)
	py := float32(plusR.Y + plusR.H/2)
	c.Canvas.DrawLine(px-c.Sf(5), py, px+c.Sf(5), py, c.Sf(1.8), c.Theme.TextPrimary)
	c.Canvas.DrawLine(px, py-c.Sf(5), px, py+c.Sf(5), c.Sf(1.8), c.Theme.TextPrimary)

	// Колесо мыши над полем
	if !blocked && !st.Disabled && midR.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
		delta := step
		if c.WheelY < 0 {
			delta = -step
		}
		nv := clampVal(*value + delta)
		if nv != *value {
			*value = nv
			changed = true
			c.RequestRedraw()
		}
	}

	// Отображение значения
	if c.Font != nil && midR.W > c.S(8) {
		txt := strconv.FormatFloat(*value, 'f', decimals, 64)
		pxSize := c.S(13)
		tw, th := c.Font.MeasureText(txt, pxSize)
		col := c.Theme.TextPrimary
		if st.Disabled {
			col = c.Theme.TextMuted
		}
		c.Canvas.PushClip(midR)
		c.Font.DrawText(c.Canvas, midR.X+(midR.W-tw)/2, midR.Y+(midR.H-th)/2, txt, pxSize, col)
		c.Canvas.PopClip()
	}

	return changed
}

// Slider рисует горизонтальный ползунок для выбора значения в диапазоне [minVal, maxVal].
func (c *Context) Slider(id string, value *float32, minVal, maxVal float32, opts ...Option) bool {
	if value == nil {
		return false
	}
	if maxVal <= minVal {
		maxVal = minVal + 1
	}

	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(28)
	rect := c.allocRect(w, h)

	pad := c.S(10)
	trackW := rect.W - pad*2
	if trackW <= 0 {
		return false
	}
	trackH := c.S(6)
	trackY := rect.Y + (rect.H-trackH)/2
	trackR := render.Rect{X: rect.X + pad, Y: trackY, W: trackW, H: trackH}

	blocked := c.IsInputBlocked()
	if !st.Disabled {
		c.RegisterHit(id, rect, platform.CursorHand)
	}
	hover := !blocked && !st.Disabled && rect.Contains(c.MouseX, c.MouseY)
	changed := false

	if hover {
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			c.ActiveID = id
		}
	}
	if c.ActiveID == id && c.MouseDown && !st.Disabled {
		t := float32(c.MouseX-trackR.X) / float32(trackR.W)
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		nv := minVal + t*(maxVal-minVal)
		if nv != *value {
			*value = nv
			changed = true
		}
	}

	norm := (*value - minVal) / (maxVal - minVal)
	if norm < 0 {
		norm = 0
	} else if norm > 1 {
		norm = 1
	}

	c.Canvas.FillRoundedRect(trackR, float32(trackH)/2, c.Theme.SurfaceDark)
	fillW := int(float32(trackW) * norm)
	if fillW > 0 {
		c.Canvas.FillRoundedRect(render.Rect{X: trackR.X, Y: trackR.Y, W: fillW, H: trackH}, float32(trackH)/2, c.Theme.Accent)
	}

	kx := float32(trackR.X) + float32(trackW)*norm
	ky := float32(rect.Y + rect.H/2)
	kr := c.Sf(8)
	if hover || c.ActiveID == id {
		kr = c.Sf(9.5)
	}
	c.Canvas.FillCircle(kx, ky, kr, c.Theme.TextPrimary)
	c.Canvas.FillCircle(kx, ky, kr-c.Sf(2.5), c.Theme.Accent)

	return changed
}

// RangeSlider рисует ползунок диапазона с двумя ручками [low, high] внутри [minVal, maxVal].
func (c *Context) RangeSlider(id string, low, high *float32, minVal, maxVal float32, opts ...Option) bool {
	if low == nil || high == nil {
		return false
	}
	if maxVal <= minVal {
		maxVal = minVal + 1
	}
	if *low > *high {
		*low = *high
	}

	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(28)
	rect := c.allocRect(w, h)

	pad := c.S(10)
	trackW := rect.W - pad*2
	if trackW <= 0 {
		return false
	}
	trackH := c.S(6)
	trackY := rect.Y + (rect.H-trackH)/2
	trackR := render.Rect{X: rect.X + pad, Y: trackY, W: trackW, H: trackH}

	normLow := (*low - minVal) / (maxVal - minVal)
	normHigh := (*high - minVal) / (maxVal - minVal)
	if normLow < 0 {
		normLow = 0
	} else if normLow > 1 {
		normLow = 1
	}
	if normHigh < 0 {
		normHigh = 0
	} else if normHigh > 1 {
		normHigh = 1
	}

	lx := float32(trackR.X) + float32(trackW)*normLow
	hx := float32(trackR.X) + float32(trackW)*normHigh

	lowID := id + ":low"
	highID := id + ":high"
	blocked := c.IsInputBlocked()
	if !st.Disabled {
		c.RegisterHit(id, rect, platform.CursorHand)
	}
	hover := !blocked && !st.Disabled && rect.Contains(c.MouseX, c.MouseY)
	changed := false

	if hover {
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			dLow := math.Abs(float64(c.MouseX) - float64(lx))
			dHigh := math.Abs(float64(c.MouseX) - float64(hx))
			if dLow <= dHigh {
				c.ActiveID = lowID
			} else {
				c.ActiveID = highID
			}
		}
	}

	if !st.Disabled && c.MouseDown && (c.ActiveID == lowID || c.ActiveID == highID) {
		t := float32(c.MouseX-trackR.X) / float32(trackR.W)
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		nv := minVal + t*(maxVal-minVal)
		if c.ActiveID == lowID {
			if nv > *high {
				nv = *high
			}
			if nv != *low {
				*low = nv
				lx = float32(trackR.X) + float32(trackW)*((*low-minVal)/(maxVal-minVal))
				changed = true
			}
		} else {
			if nv < *low {
				nv = *low
			}
			if nv != *high {
				*high = nv
				hx = float32(trackR.X) + float32(trackW)*((*high-minVal)/(maxVal-minVal))
				changed = true
			}
		}
	}

	c.Canvas.FillRoundedRect(trackR, float32(trackH)/2, c.Theme.SurfaceDark)
	x1 := int(lx)
	x2 := int(hx)
	if x2 > x1 {
		c.Canvas.FillRoundedRect(render.Rect{X: x1, Y: trackR.Y, W: x2 - x1, H: trackH}, float32(trackH)/2, c.Theme.Accent)
	}

	ky := float32(rect.Y + rect.H/2)
	for _, info := range []struct {
		x      float32
		active bool
	}{
		{lx, c.ActiveID == lowID},
		{hx, c.ActiveID == highID},
	} {
		kr := c.Sf(8)
		if info.active {
			kr = c.Sf(9.5)
		}
		c.Canvas.FillCircle(info.x, ky, kr, c.Theme.TextPrimary)
		c.Canvas.FillCircle(info.x, ky, kr-c.Sf(2.5), c.Theme.Accent)
	}

	return changed
}

// FormatHexColor форматирует цвет в строку вида #RRGGBB.
func FormatHexColor(col render.Color) string {
	return fmt.Sprintf("#%02X%02X%02X", col.R, col.G, col.B)
}

// ParseHexColor разбирает строку вида #RRGGBB или RRGGBB в render.Color.
func ParseHexColor(s string) (render.Color, bool) {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(s) != 6 {
		return render.Color{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return render.Color{}, false
	}
	return render.RGBA(uint8((v>>16)&0xFF), uint8((v>>8)&0xFF), uint8(v&0xFF), 255), true
}

func hsvToRGB(h, s, v float32) render.Color {
	h = float32(math.Mod(float64(h), 360))
	if h < 0 {
		h += 360
	}
	c := v * s
	x := c * float32(1-math.Abs(math.Mod(float64(h/60), 2)-1))
	m := v - c

	var r, g, b float32
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return render.RGBA(uint8((r+m)*255+0.5), uint8((g+m)*255+0.5), uint8((b+m)*255+0.5), 255)
}

func rgbToHSV(col render.Color) (float32, float32, float32) {
	r := float32(col.R) / 255.0
	g := float32(col.G) / 255.0
	b := float32(col.B) / 255.0
	maxC := r
	if g > maxC {
		maxC = g
	}
	if b > maxC {
		maxC = b
	}
	minC := r
	if g < minC {
		minC = g
	}
	if b < minC {
		minC = b
	}
	d := maxC - minC

	var h float32
	if d > 0 {
		switch maxC {
		case r:
			h = 60 * float32(math.Mod(float64((g-b)/d), 6))
		case g:
			h = 60 * ((b-r)/d + 2)
		case b:
			h = 60 * ((r-g)/d + 4)
		}
		if h < 0 {
			h += 360
		}
	}
	var s float32
	if maxC > 0 {
		s = d / maxC
	}
	return h, s, maxC
}

// ColorPicker рисует кнопку выбора цвета с выпадающей палитрой (SV-квадрат, Hue-полоса, свотчи и HEX-поле).
func (c *Context) ColorPicker(id string, col *render.Color, opts ...Option) bool {
	if col == nil {
		return false
	}
	if c.openState == nil {
		c.openState = make(map[string]bool)
	}
	if c.colorHSV == nil {
		c.colorHSV = make(map[string][3]float32)
	}
	if c.colorHexText == nil {
		c.colorHexText = make(map[string]string)
	}

	st := ApplyOptions(opts)
	w := c.S(160)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(34)
	rect := c.allocRect(w, h)
	radius := c.Sf(c.Theme.RadiusSm)

	isOpen := c.openState[id]
	if !st.Disabled {
		c.RegisterHit(id, rect, platform.CursorHand)
	}
	hover := !c.IsInputBlocked() && !st.Disabled && rect.Contains(c.MouseX, c.MouseY)
	if hover {
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			isOpen = !isOpen
			c.openState[id] = isOpen
			if isOpen {
				hv, sv, vv := rgbToHSV(*col)
				c.colorHSV[id] = [3]float32{hv, sv, vv}
				c.colorHexText[id] = FormatHexColor(*col)
			}
			c.RequestRedraw()
		}
	}

	borderCol := c.Theme.Border
	if isOpen {
		borderCol = c.Theme.BorderFocus
	}
	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, borderCol)

	swatchR := render.Rect{X: rect.X + c.S(6), Y: rect.Y + c.S(6), W: rect.H - c.S(12), H: rect.H - c.S(12)}
	c.Canvas.DrawRoundedBox(swatchR, c.Sf(4), *col, 1.0, c.Theme.Border)

	if c.Font != nil {
		hexStr := FormatHexColor(*col)
		pxSize := c.S(13)
		_, th := c.Font.MeasureText(hexStr, pxSize)
		c.Font.DrawText(c.Canvas, swatchR.X+swatchR.W+c.S(10), rect.Y+(rect.H-th)/2, hexStr, pxSize, c.Theme.TextPrimary)
	}

	changed := false
	if isOpen {
		popW := c.S(236)
		popH := c.S(250)
		popX := rect.X
		popY := rect.Y + rect.H + c.S(4)
		if popX+popW > c.Canvas.Width-c.S(8) {
			popX = c.Canvas.Width - popW - c.S(8)
		}
		if popY+popH > c.Canvas.Height-c.S(8) {
			popY = rect.Y - popH - c.S(4)
		}
		popR := render.Rect{X: popX, Y: popY, W: popW, H: popH}

		if c.MousePressed && !popR.Contains(c.MouseX, c.MouseY) && !rect.Contains(c.MouseX, c.MouseY) {
			c.openState[id] = false
			c.RequestRedraw()
			return false
		}

		c.PushOverlay(popR, func() {
			c.Canvas.DrawBoxShadow(popR, render.BoxShadow{
				OffsetY: c.Sf(4),
				Blur:    c.Sf(14),
				Color:   render.RGBA(0, 0, 0, 120),
				Radii:   render.UniformRadii(c.Sf(10)),
			})
			c.Canvas.DrawRoundedBox(popR, c.Sf(10), c.Theme.Card, 1.0, c.Theme.Border)

			hsv := c.colorHSV[id]
			svR := render.Rect{X: popR.X + c.S(12), Y: popR.Y + c.S(12), W: popR.W - c.S(24), H: c.S(130)}

			// Отрисовка SV-квадрата (шагом 2px для скорости)
			stepPx := 2
			for py := 0; py < svR.H; py += stepPx {
				vv := 1.0 - float32(py)/float32(svR.H)
				for px := 0; px < svR.W; px += stepPx {
					ss := float32(px) / float32(svR.W)
					c.Canvas.FillRect(render.Rect{X: svR.X + px, Y: svR.Y + py, W: stepPx, H: stepPx}, hsvToRGB(hsv[0], ss, vv))
				}
			}
			c.Canvas.DrawRoundedBox(svR, 0, render.RGBA(0, 0, 0, 0), 1.0, c.Theme.Border)

			// Взаимодействие с SV-квадратом
			svID := id + ":sv"
			if svR.Contains(c.MouseX, c.MouseY) && c.MousePressed {
				c.ActiveID = svID
			}
			if c.ActiveID == svID && c.MouseDown {
				ss := float32(c.MouseX-svR.X) / float32(svR.W)
				vv := 1.0 - float32(c.MouseY-svR.Y)/float32(svR.H)
				if ss < 0 {
					ss = 0
				} else if ss > 1 {
					ss = 1
				}
				if vv < 0 {
					vv = 0
				} else if vv > 1 {
					vv = 1
				}
				hsv[1], hsv[2] = ss, vv
				c.colorHSV[id] = hsv
				*col = hsvToRGB(hsv[0], hsv[1], hsv[2])
				c.colorHexText[id] = FormatHexColor(*col)
				changed = true
				c.RequestRedraw()
			}

			// Маркер на SV-квадрате
			mx := float32(svR.X) + hsv[1]*float32(svR.W)
			my := float32(svR.Y) + (1.0-hsv[2])*float32(svR.H)
			c.Canvas.FillCircle(mx, my, c.Sf(5), render.RGBA(255, 255, 255, 240))
			c.Canvas.FillCircle(mx, my, c.Sf(3), *col)

			// Полоса оттенка Hue
			hueR := render.Rect{X: svR.X, Y: svR.Y + svR.H + c.S(10), W: svR.W, H: c.S(16)}
			for px := 0; px < hueR.W; px += 2 {
				hh := (float32(px) / float32(hueR.W)) * 360.0
				c.Canvas.FillRect(render.Rect{X: hueR.X + px, Y: hueR.Y, W: 2, H: hueR.H}, hsvToRGB(hh, 1, 1))
			}
			c.Canvas.DrawRoundedBox(hueR, 0, render.RGBA(0, 0, 0, 0), 1.0, c.Theme.Border)

			hueID := id + ":hue"
			if hueR.Contains(c.MouseX, c.MouseY) && c.MousePressed {
				c.ActiveID = hueID
			}
			if c.ActiveID == hueID && c.MouseDown {
				hh := (float32(c.MouseX-hueR.X) / float32(hueR.W)) * 360.0
				if hh < 0 {
					hh = 0
				} else if hh > 359.9 {
					hh = 359.9
				}
				hsv[0] = hh
				c.colorHSV[id] = hsv
				*col = hsvToRGB(hsv[0], hsv[1], hsv[2])
				c.colorHexText[id] = FormatHexColor(*col)
				changed = true
				c.RequestRedraw()
			}

			hx := float32(hueR.X) + (hsv[0]/360.0)*float32(hueR.W)
			c.Canvas.FillRect(render.Rect{X: int(hx) - c.S(2), Y: hueR.Y - c.S(2), W: c.S(4), H: hueR.H + c.S(4)}, render.RGBA(255, 255, 255, 255))

			// Быстрые свотчи
			swatches := []render.Color{
				render.Hex(0x3B82F6), render.Hex(0x22C55E), render.Hex(0xF59E0B), render.Hex(0xEF4444),
				render.Hex(0x8B5CF6), render.Hex(0xEC4899), render.Hex(0x06B6D4), render.Hex(0xF8FAFC),
			}
			swY := hueR.Y + hueR.H + c.S(10)
			swSize := c.S(20)
			swGap := c.S(6)
			for i, swCol := range swatches {
				sr := render.Rect{X: svR.X + i*(swSize+swGap), Y: swY, W: swSize, H: swSize}
				c.RegisterHit(id+":sw:"+strconv.Itoa(i), sr, platform.CursorHand)
				c.Canvas.DrawRoundedBox(sr, c.Sf(4), swCol, 1.0, c.Theme.Border)
				if sr.Contains(c.MouseX, c.MouseY) {
					c.nextCursor = platform.CursorHand
					if c.MousePressed {
						*col = swCol
						hv, sv, vv := rgbToHSV(swCol)
						c.colorHSV[id] = [3]float32{hv, sv, vv}
						c.colorHexText[id] = FormatHexColor(swCol)
						changed = true
						c.RequestRedraw()
					}
				}
			}

			// HEX-поле ввода внутри оверлея
			hexBox := render.Rect{X: svR.X, Y: swY + swSize + c.S(10), W: svR.W, H: c.S(30)}
			c.pushBox(hexBox, dirColumn, AlignStart, 0)
			hexStr := c.colorHexText[id]
			if c.TextInput(id+":hex", &hexStr, "#RRGGBB", int(float32(svR.W)/c.Scale)) {
				c.colorHexText[id] = hexStr
				if parsed, ok := ParseHexColor(hexStr); ok {
					*col = parsed
					hv, sv, vv := rgbToHSV(parsed)
					c.colorHSV[id] = [3]float32{hv, sv, vv}
					changed = true
				}
			}
			c.popBox()
		})
	}

	return changed
}

// DatePicker рисует поле выбора даты с выпадающим календарём в оверлее.
func (c *Context) DatePicker(id string, date *time.Time, opts ...Option) bool {
	if date == nil {
		return false
	}
	if date.IsZero() {
		*date = time.Now()
	}
	if c.openState == nil {
		c.openState = make(map[string]bool)
	}
	if c.dateViewMonth == nil {
		c.dateViewMonth = make(map[string]time.Time)
	}

	st := ApplyOptions(opts)
	w := c.S(180)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(34)
	rect := c.allocRect(w, h)
	radius := c.Sf(c.Theme.RadiusSm)

	isOpen := c.openState[id]
	changed := false

	// Клик по полю открывает/закрывает выпадающий календарь; двойной клик по текстовой части включает ручной ввод
	calZone := render.Rect{X: rect.X + rect.W - c.S(30), Y: rect.Y, W: c.S(30), H: rect.H}
	textZone := render.Rect{X: rect.X, Y: rect.Y, W: rect.W - c.S(30), H: rect.H}
	if !st.Disabled {
		c.RegisterHit(id, rect, platform.CursorHand)
	}

	if !c.IsInputBlocked() && !st.Disabled {
		if textZone.Contains(c.MouseX, c.MouseY) && c.DoubleClicked() {
			dpEditActive[id] = true
			dpEditText[id] = date.Format("2006-01-02")
			c.openState[id] = false
			isOpen = false
			c.FocusID = id + ":edit"
			c.RequestRedraw()
		} else if (calZone.Contains(c.MouseX, c.MouseY) || (textZone.Contains(c.MouseX, c.MouseY) && !dpEditActive[id])) && c.MousePressed {
			c.nextCursor = platform.CursorHand
			isOpen = !isOpen
			c.openState[id] = isOpen
			dpEditActive[id] = false
			if isOpen {
				dpModeStore[id] = dpModeDays
				c.dateViewMonth[id] = time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
			}
			c.RequestRedraw()
		} else if c.MousePressed && dpEditActive[id] && !textZone.Contains(c.MouseX, c.MouseY) {
			if parsed, ok := parseDateInput(dpEditText[id], date.Location()); ok {
				*date = parsed
				changed = true
			}
			dpEditActive[id] = false
			if c.FocusID == id+":edit" {
				c.FocusID = ""
			}
			c.RequestRedraw()
		} else if rect.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorHand
		}
	}

	if dpEditActive[id] && c.FocusID == id+":edit" && !c.IsInputBlocked() {
		buf := []rune(dpEditText[id])
		for _, kp := range c.KeyEvents {
			if kp.Code == KeyBackspace && len(buf) > 0 {
				buf = buf[:len(buf)-1]
				dpEditText[id] = string(buf)
				c.RequestRedraw()
			} else if kp.Code == KeyEnter {
				if parsed, ok := parseDateInput(dpEditText[id], date.Location()); ok {
					*date = parsed
					changed = true
				}
				dpEditActive[id] = false
				c.FocusID = ""
				c.RequestRedraw()
			} else if kp.Code == KeyEscape {
				dpEditActive[id] = false
				c.FocusID = ""
				c.RequestRedraw()
			}
		}
		for _, ch := range c.TypedRunes {
			if (ch >= '0' && ch <= '9') || ch == '-' {
				if len(buf) < 10 {
					buf = append(buf, ch)
					dpEditText[id] = string(buf)
					c.RequestRedraw()
				}
			}
		}
	}

	borderCol := c.Theme.Border
	if isOpen || dpEditActive[id] {
		borderCol = c.Theme.BorderFocus
	}
	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, borderCol)

	if c.Font != nil {
		pxSize := c.S(13)
		label := date.Format("2006-01-02")
		if dpEditActive[id] {
			label = dpEditText[id]
		}
		_, th := c.Font.MeasureText(label, pxSize)
		tx := rect.X + c.S(10)
		ty := rect.Y + (rect.H-th)/2
		c.Font.DrawText(c.Canvas, tx, ty, label, pxSize, c.Theme.TextPrimary)
		if dpEditActive[id] && c.FocusID == id+":edit" {
			tw, _ := c.Font.MeasureText(label, pxSize)
			c.Canvas.FillRect(render.Rect{X: tx + tw + 1, Y: rect.Y + c.S(7), W: 1, H: rect.H - c.S(14)}, c.Theme.Accent)
		}
	}

	// Иконка календаря справа
	calX := rect.X + rect.W - c.S(24)
	calY := rect.Y + (rect.H-c.S(14))/2
	calR := render.Rect{X: calX, Y: calY, W: c.S(14), H: c.S(14)}
	c.Canvas.DrawRoundedBox(calR, c.Sf(2), render.RGBA(0, 0, 0, 0), 1.2, c.Theme.TextMuted)
	c.Canvas.DrawLine(float32(calR.X), float32(calR.Y+c.S(4)), float32(calR.X+calR.W), float32(calR.Y+c.S(4)), 1.2, c.Theme.TextMuted)

	if isOpen {
		popW := c.S(246)
		popH := c.S(284)
		popX := rect.X
		popY := rect.Y + rect.H + c.S(4)
		if popX+popW > c.Canvas.Width-c.S(8) {
			popX = c.Canvas.Width - popW - c.S(8)
		}
		if popY+popH > c.Canvas.Height-c.S(8) {
			popY = rect.Y - popH - c.S(4)
		}
		popR := render.Rect{X: popX, Y: popY, W: popW, H: popH}

		if c.MousePressed && !popR.Contains(c.MouseX, c.MouseY) && !rect.Contains(c.MouseX, c.MouseY) {
			c.openState[id] = false
			c.RequestRedraw()
			return false
		}

		c.PushOverlay(popR, func() {
			c.Canvas.DrawBoxShadow(popR, render.BoxShadow{
				OffsetY: c.Sf(4),
				Blur:    c.Sf(14),
				Color:   render.RGBA(0, 0, 0, 120),
				Radii:   render.UniformRadii(c.Sf(10)),
			})
			c.Canvas.DrawRoundedBox(popR, c.Sf(10), c.Theme.Card, 1.0, c.Theme.Border)

			vm := c.dateViewMonth[id]
			if vm.IsZero() {
				vm = time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
				c.dateViewMonth[id] = vm
			}

			mode := dpModeStore[id]

			// Шапка месяца/года с кнопками < и >
			hdrR := render.Rect{X: popR.X + c.S(10), Y: popR.Y + c.S(10), W: popR.W - c.S(20), H: c.S(28)}
			prevR := render.Rect{X: hdrR.X, Y: hdrR.Y, W: c.S(28), H: hdrR.H}
			nextR := render.Rect{X: hdrR.X + hdrR.W - c.S(28), Y: hdrR.Y, W: c.S(28), H: hdrR.H}
			c.RegisterHit(id+":prev", prevR, platform.CursorHand)
			c.RegisterHit(id+":next", nextR, platform.CursorHand)

			if prevR.Contains(c.MouseX, c.MouseY) {
				c.nextCursor = platform.CursorHand
				c.Canvas.FillRoundedRect(prevR, c.Sf(6), c.Theme.Surface)
				if c.MousePressed {
					if mode == dpModeYears {
						c.dateViewMonth[id] = vm.AddDate(-12, 0, 0)
					} else if mode == dpModeMonths {
						c.dateViewMonth[id] = vm.AddDate(-1, 0, 0)
					} else {
						c.dateViewMonth[id] = vm.AddDate(0, -1, 0)
					}
					c.RequestRedraw()
				}
			}
			if nextR.Contains(c.MouseX, c.MouseY) {
				c.nextCursor = platform.CursorHand
				c.Canvas.FillRoundedRect(nextR, c.Sf(6), c.Theme.Surface)
				if c.MousePressed {
					if mode == dpModeYears {
						c.dateViewMonth[id] = vm.AddDate(12, 0, 0)
					} else if mode == dpModeMonths {
						c.dateViewMonth[id] = vm.AddDate(1, 0, 0)
					} else {
						c.dateViewMonth[id] = vm.AddDate(0, 1, 0)
					}
					c.RequestRedraw()
				}
			}

			if c.Font != nil {
				px := c.S(13)
				months := []string{"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь", "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"}
				if mode == dpModeYears {
					title := formatYearRangeTitle(vm.Year())
					tw, th := c.Font.MeasureText(title, px)
					c.Font.DrawText(c.Canvas, hdrR.X+(hdrR.W-tw)/2, hdrR.Y+(hdrR.H-th)/2, title, px, c.Theme.TextPrimary)
				} else {
					mName := months[int(vm.Month())-1]
					yName := strconv.Itoa(vm.Year())
					mw, mh := c.Font.MeasureText(mName, px)
					yw, yh := c.Font.MeasureText(yName, px)
					gap := c.S(6)
					totalW := mw + gap + yw
					startX := hdrR.X + (hdrR.W-totalW)/2
					mRect := render.Rect{X: startX - c.S(4), Y: hdrR.Y, W: mw + c.S(8), H: hdrR.H}
					yRect := render.Rect{X: startX + mw + gap - c.S(4), Y: hdrR.Y, W: yw + c.S(8), H: hdrR.H}
					c.RegisterHit(id+":month", mRect, platform.CursorHand)
					c.RegisterHit(id+":year", yRect, platform.CursorHand)

					mCol := c.Theme.TextPrimary
					if mRect.Contains(c.MouseX, c.MouseY) {
						c.nextCursor = platform.CursorHand
						c.Canvas.FillRoundedRect(mRect, c.Sf(6), c.Theme.Surface)
						mCol = c.Theme.Accent
						if c.MousePressed {
							if mode == dpModeMonths {
								dpModeStore[id] = dpModeDays
							} else {
								dpModeStore[id] = dpModeMonths
							}
							c.RequestRedraw()
						}
					}
					yCol := c.Theme.TextPrimary
					if yRect.Contains(c.MouseX, c.MouseY) {
						c.nextCursor = platform.CursorHand
						c.Canvas.FillRoundedRect(yRect, c.Sf(6), c.Theme.Surface)
						yCol = c.Theme.Accent
						if c.MousePressed {
							dpModeStore[id] = dpModeYears
							c.RequestRedraw()
						}
					}
					c.Font.DrawText(c.Canvas, startX, hdrR.Y+(hdrR.H-mh)/2, mName, px, mCol)
					c.Font.DrawText(c.Canvas, startX+mw+gap, hdrR.Y+(hdrR.H-yh)/2, yName, px, yCol)
				}

				pw, ph := c.Font.MeasureText("<", px)
				c.Font.DrawText(c.Canvas, prevR.X+(prevR.W-pw)/2, prevR.Y+(prevR.H-ph)/2, "<", px, c.Theme.TextPrimary)
				nw, nh := c.Font.MeasureText(">", px)
				c.Font.DrawText(c.Canvas, nextR.X+(nextR.W-nw)/2, nextR.Y+(nextR.H-nh)/2, ">", px, c.Theme.TextPrimary)
			}

			gridX := popR.X + c.S(10)
			gridY := hdrR.Y + hdrR.H + c.S(8)

			if mode == dpModeMonths {
				shortMonths := []string{"Янв", "Фев", "Мар", "Апр", "Май", "Июн", "Июл", "Авг", "Сен", "Окт", "Ноя", "Дек"}
				mW := (popR.W - c.S(20)) / 3
				mH := c.S(42)
				for mi := 0; mi < 12; mi++ {
					colIdx := mi % 3
					rowIdx := mi / 3
					mr := render.Rect{X: gridX + colIdx*mW + c.S(2), Y: gridY + rowIdx*mH + c.S(2), W: mW - c.S(4), H: mH - c.S(4)}
					c.RegisterHit(id+":m:"+strconv.Itoa(mi), mr, platform.CursorHand)
					isCur := int(vm.Month())-1 == mi
					hov := mr.Contains(c.MouseX, c.MouseY)
					if isCur {
						c.Canvas.FillRoundedRect(mr, c.Sf(6), c.Theme.Accent)
					} else if hov {
						c.Canvas.FillRoundedRect(mr, c.Sf(6), c.Theme.Surface)
					}
					if hov {
						c.nextCursor = platform.CursorHand
						if c.MousePressed {
							c.dateViewMonth[id] = time.Date(vm.Year(), time.Month(mi+1), 1, 0, 0, 0, 0, vm.Location())
							dpModeStore[id] = dpModeDays
							c.RequestRedraw()
						}
					}
					if c.Font != nil {
						tc := c.Theme.TextPrimary
						if isCur {
							tc = render.RGBA(255, 255, 255, 255)
						}
						tw, th := c.Font.MeasureText(shortMonths[mi], c.S(12))
						c.Font.DrawText(c.Canvas, mr.X+(mr.W-tw)/2, mr.Y+(mr.H-th)/2, shortMonths[mi], c.S(12), tc)
					}
				}
			} else if mode == dpModeYears {
				startYear := (vm.Year() / 12) * 12
				yW := (popR.W - c.S(20)) / 3
				yH := c.S(42)
				for yi := 0; yi < 12; yi++ {
					yr := startYear + yi
					colIdx := yi % 3
					rowIdx := yi / 3
					yrRect := render.Rect{X: gridX + colIdx*yW + c.S(2), Y: gridY + rowIdx*yH + c.S(2), W: yW - c.S(4), H: yH - c.S(4)}
					c.RegisterHit(id+":y:"+strconv.Itoa(yr), yrRect, platform.CursorHand)
					isCur := vm.Year() == yr
					hov := yrRect.Contains(c.MouseX, c.MouseY)
					if isCur {
						c.Canvas.FillRoundedRect(yrRect, c.Sf(6), c.Theme.Accent)
					} else if hov {
						c.Canvas.FillRoundedRect(yrRect, c.Sf(6), c.Theme.Surface)
					}
					if hov {
						c.nextCursor = platform.CursorHand
						if c.MousePressed {
							c.dateViewMonth[id] = time.Date(yr, vm.Month(), 1, 0, 0, 0, 0, vm.Location())
							dpModeStore[id] = dpModeMonths
							c.RequestRedraw()
						}
					}
					if c.Font != nil {
						tc := c.Theme.TextPrimary
						if isCur {
							tc = render.RGBA(255, 255, 255, 255)
						}
						ys := strconv.Itoa(yr)
						tw, th := c.Font.MeasureText(ys, c.S(12))
						c.Font.DrawText(c.Canvas, yrRect.X+(yrRect.W-tw)/2, yrRect.Y+(yrRect.H-th)/2, ys, c.S(12), tc)
					}
				}
			} else {
				// Дни недели (Пн..Вс)
				cellW := (popR.W - c.S(20)) / 7
				cellH := c.S(28)

				wdNames := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
				if c.Font != nil {
					pxSmall := c.S(11)
					for i, wd := range wdNames {
						tw, th := c.Font.MeasureText(wd, pxSmall)
						cx := gridX + i*cellW + (cellW-tw)/2
						c.Font.DrawText(c.Canvas, cx, gridY+(c.S(20)-th)/2, wd, pxSmall, c.Theme.TextMuted)
					}
				}

				// Сетка чисел месяца
				daysY := gridY + c.S(22)
				firstWD := int(vm.Weekday())
				if firstWD == 0 {
					firstWD = 7 // Воскресенье -> 7
				}
				offset := firstWD - 1
				daysInMonth := time.Date(vm.Year(), vm.Month()+1, 0, 0, 0, 0, 0, vm.Location()).Day()
				now := time.Now()

				for d := 1; d <= daysInMonth; d++ {
					slot := offset + d - 1
					colIdx := slot % 7
					rowIdx := slot / 7
					dr := render.Rect{X: gridX + colIdx*cellW + c.S(1), Y: daysY + rowIdx*cellH + c.S(1), W: cellW - c.S(2), H: cellH - c.S(2)}
					c.RegisterHit(id+":d:"+strconv.Itoa(d), dr, platform.CursorHand)

					isSel := date.Year() == vm.Year() && date.Month() == vm.Month() && date.Day() == d
					isToday := now.Year() == vm.Year() && now.Month() == vm.Month() && now.Day() == d
					dHover := dr.Contains(c.MouseX, c.MouseY)

					if isSel {
						c.Canvas.FillRoundedRect(dr, c.Sf(6), c.Theme.Accent)
					} else if dHover {
						c.Canvas.FillRoundedRect(dr, c.Sf(6), c.Theme.Surface)
					} else if isToday {
						c.Canvas.DrawRoundedBox(dr, c.Sf(6), render.RGBA(0, 0, 0, 0), 1.0, c.Theme.Accent)
					}

					if dHover {
						c.nextCursor = platform.CursorHand
						if c.MousePressed {
							*date = time.Date(vm.Year(), vm.Month(), d, 0, 0, 0, 0, date.Location())
							c.openState[id] = false
							changed = true
							c.RequestRedraw()
						}
					}

					if c.Font != nil {
						px := c.S(12)
						ds := strconv.Itoa(d)
						tw, th := c.Font.MeasureText(ds, px)
						tc := c.Theme.TextPrimary
						if isSel {
							tc = render.RGBA(255, 255, 255, 255)
						}
						c.Font.DrawText(c.Canvas, dr.X+(dr.W-tw)/2, dr.Y+(dr.H-th)/2, ds, px, tc)
					}
				}
			}

			// Кнопка "Сегодня" внизу календаря
			todayR := render.Rect{X: popR.X + c.S(10), Y: popR.Y + popR.H - c.S(34), W: popR.W - c.S(20), H: c.S(24)}
			c.RegisterHit(id+":today", todayR, platform.CursorHand)
			hovToday := todayR.Contains(c.MouseX, c.MouseY)
			todayBg := c.Theme.Surface
			if hovToday {
				c.nextCursor = platform.CursorHand
				todayBg = c.Theme.Accent.WithAlpha(40)
				if c.MousePressed {
					now := time.Now()
					*date = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, date.Location())
					c.dateViewMonth[id] = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, date.Location())
					dpModeStore[id] = dpModeDays
					c.openState[id] = false
					changed = true
					c.RequestRedraw()
				}
			}
			c.Canvas.DrawRoundedBox(todayR, c.Sf(6), todayBg, 1.0, c.Theme.Border)
			if c.Font != nil {
				tw, th := c.Font.MeasureText("Сегодня", c.S(11))
				c.Font.DrawText(c.Canvas, todayR.X+(todayR.W-tw)/2, todayR.Y+(todayR.H-th)/2, "Сегодня", c.S(11), c.Theme.Accent)
			}
		})
	}

	return changed
}
