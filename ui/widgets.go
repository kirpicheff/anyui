package ui

import (
	"image"

	"anyui/platform"
	"anyui/render"
)

// animFloat безопасно запрашивает анимированное значение даже если Animator не задан.
func (c *Context) animFloat(key string, target, dur float32) float32 {
	if c.Animator == nil {
		return target
	}
	return c.Animator.Float(key, target, dur)
}

// consumeEnter проверяет нажатие Enter для сфокусированного виджета.
func (c *Context) consumeEnter() bool {
	for _, kp := range c.KeyEvents {
		if kp.Code == KeyEnter && !kp.Mods.Ctrl && !kp.Mods.Alt {
			return true
		}
	}
	return false
}

// Label выводит строку текста.
func (c *Context) Label(text string, sizeDp int, col render.Color) {
	c.LabelOpts(text, FontSize(sizeDp), TextColor(col))
}

// LabelOpts выводит строку текста с опциями стилизации.
func (c *Context) LabelOpts(text string, opts ...Option) {
	if c.Font == nil {
		return
	}
	st := ApplyOptions(opts)
	sizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		sizeDp = *st.FontSizeDp
	}
	col := c.Theme.TextPrimary
	if st.Color != nil {
		col = *st.Color
	}
	if st.Disabled {
		col = c.Theme.TextMuted
	}

	pxSize := c.S(sizeDp)
	w, h := c.Font.MeasureTextStyled(text, pxSize, st.FontStyle)
	r := c.allocRectStyled(w, h, st)
	c.Font.DrawTextStyled(c.Canvas, r.X, r.Y, text, pxSize, col, st.FontStyle)
}

// StatusDot рисует цветной кружок-индикатор и подпись справа от него.
func (c *Context) StatusDot(dotColor render.Color, label string) {
	if c.Font == nil {
		return
	}
	pxSize := c.S(12)
	tw, th := c.Font.MeasureText(label, pxSize)
	dotR := c.Sf(4)
	gap := c.S(8)
	totalW := int(dotR*2) + gap + tw
	h := th
	if int(dotR*2) > h {
		h = int(dotR * 2)
	}

	r := c.allocRect(totalW, h)
	c.Canvas.FillCircle(float32(r.X)+dotR, float32(r.Y)+float32(r.H)*0.5, dotR, dotColor)
	c.Font.DrawText(c.Canvas, r.X+int(dotR*2)+gap, r.Y+(r.H-th)/2, label, pxSize, c.Theme.TextMuted)
}

// Card рисует скруглённую карточку с 1px-обводкой и размещает внутри неё дочерние виджеты.
func (c *Context) Card(wDp, hDp, padDp int, fn func()) {
	c.CardOpts(fn, Width(wDp), Height(hDp), Padding(padDp))
}

// CardOpts рисует скруглённую карточку с опциями стилизации.
func (c *Context) CardOpts(fn func(), opts ...Option) {
	st := ApplyOptions(opts)
	wDp := 0
	if st.WidthDp != nil {
		wDp = *st.WidthDp
	}
	hDp := 0
	if st.HeightDp != nil {
		hDp = *st.HeightDp
	}
	padDp := 16
	if st.PadDp != nil {
		padDp = *st.PadDp
	}

	w := c.S(wDp)
	if wDp <= 0 {
		w = c.RemainingWidth()
	}
	h := c.S(hDp)
	autoKey := ""
	if st.AutoHeight {
		autoKey = c.nextJustifyKey("card_")
		if c.autoHeights != nil {
			if cachedH, ok := c.autoHeights[autoKey]; ok && cachedH > 0 {
				h = cachedH
			}
		}
		if h <= 0 {
			h = c.RemainingHeight()
		}
	} else if hDp <= 0 {
		h = c.RemainingHeight()
	}

	bg := c.Theme.Card
	if st.Bg != nil {
		bg = *st.Bg
	}
	border := c.Theme.Border
	if st.BorderColor != nil {
		border = *st.BorderColor
	}
	rad := c.Theme.RadiusLg
	if st.Radius != nil {
		rad = *st.Radius
	}

	r := c.allocRectStyled(w, h, st)
	c.Canvas.DrawRoundedBox(r, c.Sf(rad), bg, 1.0, border)

	pad := c.S(padDp)
	inner := render.Rect{
		X: r.X + pad,
		Y: r.Y + pad,
		W: r.W - pad*2,
		H: r.H - pad*2,
	}
	if inner.W < 0 {
		inner.W = 0
	}
	if inner.H < 0 {
		inner.H = 0
	}

	c.Canvas.PushClip(inner)
	c.pushBox(inner, dirColumn, AlignStart, c.S(8))
	if fn != nil {
		fn()
	}
	box := c.topLayout()
	usedH := box.maxUsedY - inner.Y + pad*2
	c.popBox()
	c.Canvas.PopClip()

	if st.AutoHeight && autoKey != "" {
		if usedH < pad*2 {
			usedH = pad * 2
		}
		_, constrainedH := c.applySizeConstraints(r.W, usedH, st)
		if c.autoHeights == nil {
			c.autoHeights = make(map[string]int)
		}
		if prev, ok := c.autoHeights[autoKey]; !ok || prev != constrainedH {
			c.autoHeights[autoKey] = constrainedH
			c.RequestRedraw()
		}
	}
}

// Button рисует кнопку и возвращает true в кадре нажатия.
// Если widthDp <= 0, ширина подбирается автоматически по тексту.
func (c *Context) Button(id, label string, primary bool, widthDp int) bool {
	var opts []Option
	if primary {
		opts = append(opts, Primary())
	} else {
		opts = append(opts, Secondary())
	}
	if widthDp > 0 {
		opts = append(opts, Width(widthDp))
	}
	return c.ButtonOpts(id, label, opts...)
}

// ButtonOpts рисует кнопку с опциями стилизации и плавными анимациями.
func (c *Context) ButtonOpts(id, label string, opts ...Option) bool {
	st := ApplyOptions(opts)

	fontSizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		fontSizeDp = *st.FontSizeDp
	}
	pxSize := c.S(fontSizeDp)
	tw, th := 0, pxSize
	if c.Font != nil {
		tw, th = c.Font.MeasureText(label, pxSize)
	}

	padX := 16
	if st.PadDp != nil && *st.PadDp >= 0 {
		padX = *st.PadDp
	}

	hasIcon := st.IconSVG != ""
	iconSize := c.S(16)
	iconGap := 0
	if hasIcon && label != "" {
		iconGap = c.S(8)
	}
	contentW := tw
	if hasIcon {
		contentW += iconSize + iconGap
	}

	w := contentW + c.S(padX*2)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(38)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}

	r := c.allocRectStyled(w, h, st)
	if !st.Disabled {
		c.registerFocusable(id)
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

	hoverTarget := float32(0)
	if hovered {
		hoverTarget = 1
	}
	pressTarget := float32(0)
	if !st.Disabled && c.ActiveID == id && hovered {
		pressTarget = 1
	}
	hoverAnim := c.animFloat("btn_h:"+id, hoverTarget, 0.14)
	pressAnim := c.animFloat("btn_p:"+id, pressTarget, 0.10)

	bg, hoverBg, downBg, border, textCol := c.resolveButtonColors(st)
	bg = render.LerpColor(bg, hoverBg, hoverAnim)
	bg = render.LerpColor(bg, downBg, pressAnim)

	if c.FocusID == id && !st.Disabled {
		border = c.Theme.BorderFocus
	}

	rad := c.Theme.RadiusSm
	if st.Radius != nil {
		rad = *st.Radius
	}

	c.Canvas.DrawRoundedBox(r, c.Sf(rad), bg, 1.0, border)
	startX := r.X + (r.W-contentW)/2
	if hasIcon {
		iconY := r.Y + (r.H-iconSize)/2
		c.Canvas.DrawSVG(st.IconSVG, startX, iconY, iconSize, textCol)
		startX += iconSize + iconGap
	}
	if c.Font != nil && label != "" {
		c.Font.DrawTextStyled(c.Canvas, startX, r.Y+(r.H-th)/2, label, pxSize, textCol, st.FontStyle)
	}

	return clicked
}

func (c *Context) resolveButtonColors(st WidgetStyle) (bg, hoverBg, downBg, border, textCol render.Color) {
	textCol = c.Theme.TextPrimary
	switch st.Variant {
	case VariantPrimary:
		bg = c.Theme.Accent
		hoverBg = c.Theme.AccentHover
		downBg = c.Theme.AccentDown
		border = render.RGBA(255, 255, 255, 35)
		textCol = render.RGB(255, 255, 255)
	case VariantDanger:
		bg = c.Theme.Danger
		hoverBg = render.LerpColor(c.Theme.Danger, render.RGB(0, 0, 0), 0.15)
		downBg = render.LerpColor(c.Theme.Danger, render.RGB(0, 0, 0), 0.30)
		border = render.RGBA(255, 255, 255, 35)
		textCol = render.RGB(255, 255, 255)
	case VariantGhost:
		bg = c.Theme.Bg.WithAlpha(0)
		hoverBg = c.Theme.Surface
		downBg = c.Theme.BtnSecondary
		border = c.Theme.Border.WithAlpha(0)
	default:
		bg = c.Theme.BtnSecondary
		hoverBg = c.Theme.BtnSecondaryHover
		downBg = c.Theme.Surface
		border = c.Theme.Border
	}

	if st.Bg != nil {
		bg = *st.Bg
		hoverBg = render.LerpColor(bg, render.RGB(255, 255, 255), 0.08)
		downBg = render.LerpColor(bg, render.RGB(0, 0, 0), 0.12)
	}
	if st.BorderColor != nil {
		border = *st.BorderColor
	}
	if st.Color != nil {
		textCol = *st.Color
	}
	if st.Disabled {
		bg = c.Theme.Surface
		hoverBg = bg
		downBg = bg
		border = c.Theme.Border
		textCol = c.Theme.TextMuted
	}
	return
}

// Toggle рисует переключатель (switch) с подписью и возвращает true при переключении.
func (c *Context) Toggle(id, label string, value *bool) bool {
	return c.ToggleOpts(id, label, value)
}

// ToggleOpts рисует переключатель с опциями стилизации и плавной анимацией ползунка.
func (c *Context) ToggleOpts(id, label string, value *bool, opts ...Option) bool {
	if value == nil {
		return false
	}
	st := ApplyOptions(opts)

	fontSizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		fontSizeDp = *st.FontSizeDp
	}
	pxSize := c.S(fontSizeDp)
	tw, th := 0, pxSize
	if c.Font != nil && label != "" {
		tw, th = c.Font.MeasureText(label, pxSize)
	}

	swW := c.S(42)
	swH := c.S(24)
	gap := 0
	if label != "" {
		gap = c.S(12)
	}

	totalW := swW + gap + tw
	totalH := swH
	if th > totalH {
		totalH = th
	}

	r := c.allocRect(totalW, totalH)
	if !st.Disabled {
		c.registerFocusable(id)
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
		*value = !*value
		changed = true
		c.RequestRedraw()
	}
	if !st.Disabled && c.FocusID == id && c.consumeEnter() {
		*value = !*value
		changed = true
		c.RequestRedraw()
	}

	swRect := render.Rect{
		X: r.X,
		Y: r.Y + (r.H-swH)/2,
		W: swW,
		H: swH,
	}

	target := float32(0)
	if *value {
		target = 1
	}
	anim := c.animFloat("tgl:"+id, target, 0.16)

	onTrack := c.Theme.Accent
	if st.Bg != nil {
		onTrack = *st.Bg
	}
	trackCol := render.LerpColor(c.Theme.Surface, onTrack, anim)
	borderCol := render.LerpColor(c.Theme.Border, onTrack, anim)
	if c.FocusID == id && !st.Disabled {
		borderCol = c.Theme.BorderFocus
	}
	if st.Disabled {
		trackCol = c.Theme.Surface
		borderCol = c.Theme.Border
	}

	c.Canvas.DrawRoundedBox(swRect, float32(swH)*0.5, trackCol, 1.0, borderCol)

	knobR := float32(swH)*0.5 - c.Sf(3)
	minX := float32(swRect.X) + float32(swH)*0.5
	maxX := float32(swRect.X+swRect.W) - float32(swH)*0.5
	knobX := minX + (maxX-minX)*anim
	knobY := float32(swRect.Y) + float32(swH)*0.5
	knobCol := render.RGB(255, 255, 255)
	if !*value && anim < 0.5 {
		knobCol = c.Theme.TextPrimary
	}
	if st.Disabled {
		knobCol = c.Theme.TextMuted
	}
	c.Canvas.FillCircle(knobX, knobY, knobR, knobCol)

	if c.Font != nil && label != "" {
		lblCol := c.Theme.TextPrimary
		if st.Color != nil {
			lblCol = *st.Color
		}
		if st.Disabled {
			lblCol = c.Theme.TextMuted
		}
		c.Font.DrawText(c.Canvas, r.X+swW+gap, r.Y+(r.H-th)/2, label, pxSize, lblCol)
	}

	return changed
}

// Image отображает image.Image в области размером wDp x hDp с заданным режимом масштабирования.
func (c *Context) Image(img image.Image, wDp, hDp int, mode render.ImageScaleMode) {
	w := c.S(wDp)
	if wDp <= 0 {
		w = c.RemainingWidth()
	}
	h := c.S(hDp)
	if hDp <= 0 {
		h = c.RemainingHeight()
	}
	r := c.allocRect(w, h)
	c.Canvas.DrawImage(r, img, mode)
}

// CanvasView выделяет область wDp x hDp в раскладке и вызывает drawFn для кастомного рисования с клиппингом.
func (c *Context) CanvasView(wDp, hDp int, drawFn func(canvas *render.Canvas, area render.Rect)) {
	w := c.S(wDp)
	if wDp <= 0 {
		w = c.RemainingWidth()
	}
	h := c.S(hDp)
	if hDp <= 0 {
		h = c.RemainingHeight()
	}
	r := c.allocRect(w, h)
	if drawFn == nil || r.Empty() {
		return
	}
	c.Canvas.PushClip(r)
	drawFn(c.Canvas, r)
	c.Canvas.PopClip()
}

