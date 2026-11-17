package ui

import (
	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/render"
)

// IconButton рисует квадратную кнопку с SVG-иконкой без текста.
func (c *Context) IconButton(id, svg string, opts ...Option) bool {
	base := []Option{Width(36), Height(36), Padding(0), Icon(svg)}
	base = append(base, opts...)
	return c.ButtonOpts(id, "", base...)
}

// Checkbox рисует флажок с подписью, плавной анимацией и поддержкой Indeterminate / Disabled.
func (c *Context) Checkbox(id, label string, checked *bool, opts ...Option) bool {
	if checked == nil {
		var dummy bool
		checked = &dummy
	}
	st := ApplyOptions(opts)

	sizeDp := 18
	if st.HeightDp != nil && *st.HeightDp > 0 {
		sizeDp = *st.HeightDp
	}
	boxPx := c.S(sizeDp)
	gap := c.S(8)

	pxSize := c.S(14)
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		pxSize = c.S(*st.FontSizeDp)
	}
	tw, th := 0, boxPx
	if c.Font != nil && label != "" {
		tw, th = c.Font.MeasureTextStyled(label, pxSize, st.FontStyle)
	}
	h := boxPx
	if th > h {
		h = th
	}
	totalW := boxPx
	if tw > 0 {
		totalW += gap + tw
	}

	r := c.allocRect(totalW, h)
	if !st.Disabled {
		c.registerFocusable(id)
		c.RegisterHit(id, r, platform.CursorHand)
	}

	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered {
		c.HotID = id
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			c.ActiveID = id
			c.FocusID = id
		}
	}

	changed := false
	if !st.Disabled && c.MouseReleased && c.ActiveID == id && hovered {
		*checked = !*checked
		changed = true
		c.RequestRedraw()
	}
	if !st.Disabled && c.FocusID == id && c.consumeEnter() {
		*checked = !*checked
		changed = true
		c.RequestRedraw()
	}

	active := *checked || st.Indeterminate
	target := float32(0)
	if active {
		target = 1
	}
	anim := c.animFloat("chk:"+id, target, 0.14)

	boxRect := render.Rect{
		X: r.X,
		Y: r.Y + (r.H-boxPx)/2,
		W: boxPx,
		H: boxPx,
	}

	bg := render.LerpColor(c.Theme.Surface, c.Theme.Accent, anim)
	border := render.LerpColor(c.Theme.Border, c.Theme.Accent, anim)
	if st.Color != nil && active {
		bg = render.LerpColor(c.Theme.Surface, *st.Color, anim)
		border = bg
	}
	if hovered && !active {
		border = c.Theme.BorderFocus
	}
	if c.FocusID == id && !st.Disabled {
		border = c.Theme.BorderFocus
	}
	if st.Disabled {
		bg = c.Theme.Surface.WithAlpha(120)
		border = c.Theme.Border.WithAlpha(120)
	}

	rad := float32(4)
	if st.Radius != nil {
		rad = *st.Radius
	}
	c.Canvas.DrawRoundedBox(boxRect, c.Sf(rad), bg, 1.0, border)

	if anim > 0.05 {
		markCol := render.RGBA(255, 255, 255, uint8(255*anim))
		if st.Disabled {
			markCol = c.Theme.TextMuted
		}
		if st.Indeterminate {
			// Горизонтальная черта для неопределённого состояния
			x1 := float32(boxRect.X) + float32(boxPx)*0.24
			x2 := float32(boxRect.X) + float32(boxPx)*0.76
			ym := float32(boxRect.Y) + float32(boxPx)*0.5
			c.Canvas.DrawLine(x1, ym, x2, ym, c.Sf(2), markCol)
		} else {
			// Галочка двумя сегментами
			x1 := float32(boxRect.X) + float32(boxPx)*0.24
			y1 := float32(boxRect.Y) + float32(boxPx)*0.52
			x2 := float32(boxRect.X) + float32(boxPx)*0.44
			y2 := float32(boxRect.Y) + float32(boxPx)*0.72
			x3 := float32(boxRect.X) + float32(boxPx)*0.76
			y3 := float32(boxRect.Y) + float32(boxPx)*0.30
			c.Canvas.DrawLine(x1, y1, x2, y2, c.Sf(2), markCol)
			c.Canvas.DrawLine(x2, y2, x3, y3, c.Sf(2), markCol)
		}
	}

	if c.Font != nil && label != "" {
		textCol := c.Theme.TextPrimary
		if st.Disabled {
			textCol = c.Theme.TextMuted
		}
		c.Font.DrawTextStyled(c.Canvas, r.X+boxPx+gap, r.Y+(r.H-th)/2, label, pxSize, textCol, st.FontStyle)
	}
	return changed
}

// RadioButton рисует одиночную радиокнопку и возвращает true в кадре выбора.
func (c *Context) RadioButton(id, label string, selected bool, opts ...Option) bool {
	st := ApplyOptions(opts)
	circlePx := c.S(18)
	gap := c.S(8)

	pxSize := c.S(14)
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		pxSize = c.S(*st.FontSizeDp)
	}
	tw, th := 0, circlePx
	if c.Font != nil && label != "" {
		tw, th = c.Font.MeasureTextStyled(label, pxSize, st.FontStyle)
	}
	h := circlePx
	if th > h {
		h = th
	}
	totalW := circlePx
	if tw > 0 {
		totalW += gap + tw
	}

	r := c.allocRect(totalW, h)
	if !st.Disabled {
		c.registerFocusable(id)
		c.RegisterHit(id, r, platform.CursorHand)
	}

	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered {
		c.HotID = id
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			c.ActiveID = id
			c.FocusID = id
		}
	}

	clicked := false
	if !st.Disabled && c.MouseReleased && c.ActiveID == id && hovered {
		clicked = true
		c.RequestRedraw()
	}
	if !st.Disabled && c.FocusID == id && c.consumeEnter() {
		clicked = true
		c.RequestRedraw()
	}

	target := float32(0)
	if selected {
		target = 1
	}
	anim := c.animFloat("rad:"+id, target, 0.14)

	cx := float32(r.X) + float32(circlePx)*0.5
	cy := float32(r.Y) + float32(r.H)*0.5
	outerR := float32(circlePx) * 0.5

	borderCol := render.LerpColor(c.Theme.Border, c.Theme.Accent, anim)
	if hovered && !selected {
		borderCol = c.Theme.BorderFocus
	}
	if st.Disabled {
		borderCol = c.Theme.Border.WithAlpha(120)
	}

	c.Canvas.FillCircle(cx, cy, outerR, borderCol)
	c.Canvas.FillCircle(cx, cy, outerR-c.Sf(1.5), c.Theme.Surface)

	if anim > 0.05 {
		dotCol := c.Theme.Accent
		if st.Disabled {
			dotCol = c.Theme.TextMuted
		}
		c.Canvas.FillCircle(cx, cy, (outerR*0.5)*anim, dotCol)
	}

	if c.Font != nil && label != "" {
		textCol := c.Theme.TextPrimary
		if st.Disabled {
			textCol = c.Theme.TextMuted
		}
		c.Font.DrawTextStyled(c.Canvas, r.X+circlePx+gap, r.Y+(r.H-th)/2, label, pxSize, textCol, st.FontStyle)
	}
	return clicked
}

// RadioGroup рисует вертикальную группу радиокнопок и обновляет selected при выборе.
func (c *Context) RadioGroup(id string, options []string, selected *int, opts ...Option) bool {
	if selected == nil || len(options) == 0 {
		return false
	}
	changed := false
	c.Column(8, func() {
		for i, label := range options {
			itemID := id + "_" + string(rune('0'+i))
			if c.RadioButton(itemID, label, *selected == i, opts...) {
				if *selected != i {
					*selected = i
					changed = true
				}
			}
		}
	})
	return changed
}

// Separator рисует горизонтальный разделитель шириной в доступную область.
func (c *Context) Separator(opts ...Option) {
	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(1)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	col := c.Theme.Border
	if st.Color != nil {
		col = *st.Color
	} else if st.Bg != nil {
		col = *st.Bg
	}

	r := c.allocRect(w, h)
	c.Canvas.FillRect(r, col)
}

// Badge рисует компактный цветной бейдж-пилюлю с текстом.
func (c *Context) Badge(text string, opts ...Option) {
	if c.Font == nil {
		return
	}
	st := ApplyOptions(opts)
	sizeDp := 11
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		sizeDp = *st.FontSizeDp
	}
	pxSize := c.S(sizeDp)
	tw, th := c.Font.MeasureTextStyled(text, pxSize, st.FontStyle)
	padX := c.S(8)
	padY := c.S(3)
	if st.PadDp != nil && *st.PadDp >= 0 {
		padX = c.S(*st.PadDp)
	}

	w := tw + padX*2
	h := th + padY*2
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}

	bg := c.Theme.Accent.WithAlpha(45)
	textCol := c.Theme.Accent
	switch st.Variant {
	case VariantPrimary:
		bg = c.Theme.Accent
		textCol = render.RGB(255, 255, 255)
	case VariantDanger:
		bg = c.Theme.Danger.WithAlpha(45)
		textCol = c.Theme.Danger
	case VariantSecondary:
		bg = c.Theme.Surface
		textCol = c.Theme.TextMuted
	}
	if st.Bg != nil {
		bg = *st.Bg
	}
	if st.Color != nil {
		textCol = *st.Color
	}

	rad := float32(h) / (2.0 * c.Scale)
	if st.Radius != nil {
		rad = *st.Radius
	}

	r := c.allocRect(w, h)
	c.Canvas.FillRoundedRect(r, c.Sf(rad), bg)
	c.Font.DrawTextStyled(c.Canvas, r.X+(r.W-tw)/2, r.Y+(r.H-th)/2, text, pxSize, textCol, st.FontStyle)
}

// ProgressBar рисует полосу прогресса (progress от 0.0 до 1.0).
func (c *Context) ProgressBar(progress float32, opts ...Option) {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(8)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}

	bg := c.Theme.Surface
	if st.Bg != nil {
		bg = *st.Bg
	}
	fillCol := c.Theme.Accent
	if st.Variant == VariantDanger {
		fillCol = c.Theme.Danger
	}
	if st.Color != nil {
		fillCol = *st.Color
	}
	rad := float32(4)
	if st.Radius != nil {
		rad = *st.Radius
	}

	r := c.allocRect(w, h)
	c.Canvas.FillRoundedRect(r, c.Sf(rad), bg)
	if progress > 0 {
		fillW := int(float32(r.W)*progress + 0.5)
		if fillW > 0 {
			fillR := render.Rect{X: r.X, Y: r.Y, W: fillW, H: r.H}
			c.Canvas.FillRoundedRect(fillR, c.Sf(rad), fillCol)
		}
	}
}

