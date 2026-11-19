package ui

import (
	"image"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
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

func (c *Context) textRenderOpts(st WidgetStyle) render.TextRenderOptions {
	opts := render.TextRenderOptions{
		Style:           st.FontStyle,
		LetterSpacingPx: c.Sf(st.LetterSpacing),
	}
	if st.TextShadow != nil && st.TextShadow.Color.A > 0 {
		opts.Shadow = &render.TextShadowStyle{
			OffsetX: c.Sf(st.TextShadow.OffsetX),
			OffsetY: c.Sf(st.TextShadow.OffsetY),
			Blur:    c.Sf(st.TextShadow.Blur),
			Color:   st.TextShadow.Color,
		}
	}
	return opts
}

// LabelOpts выводит строку текста с опциями стилизации, переносом строк (WordWrap) и обрезкой (Ellipsis).
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
	textOpts := c.textRenderOpts(st)
	if st.WordWrap {
		maxW := c.RemainingWidth()
		if st.WidthDp != nil && *st.WidthDp > 0 {
			maxW = c.S(*st.WidthDp)
		}
		if st.MaxWidthDp != nil && *st.MaxWidthDp > 0 && c.S(*st.MaxWidthDp) < maxW {
			maxW = c.S(*st.MaxWidthDp)
		}
		if maxW <= 0 {
			maxW = c.S(200)
		}
		runes := []rune(text)
		lines := c.WrapTextLines(runes, maxW, pxSize, st.FontStyle)
		_, lineH := c.Font.MeasureTextWithOptions("Аy", pxSize, textOpts)
		if lineH <= 0 {
			_, lineH = c.Font.MeasureTextWithOptions(text, pxSize, textOpts)
		}
		gap := c.S(3)
		lineStep := lineH + gap
		totalH := lineH
		if len(lines) > 1 {
			totalH = len(lines)*lineStep - gap
		}
		totalW := 0
		for _, sp := range lines {
			lw, _ := c.Font.MeasureTextWithOptions(string(runes[sp.start:sp.end]), pxSize, textOpts)
			if lw > totalW {
				totalW = lw
			}
		}
		if st.WidthDp != nil && *st.WidthDp > 0 {
			totalW = c.S(*st.WidthDp)
		}
		r := c.allocRectStyled(totalW, totalH, st)
		for i, sp := range lines {
			c.Font.DrawTextWithOptions(c.Canvas, r.X, r.Y+i*lineStep, string(runes[sp.start:sp.end]), pxSize, col, textOpts)
		}
		return
	}

	drawText := text
	w, h := c.Font.MeasureTextWithOptions(drawText, pxSize, textOpts)
	if st.Ellipsis {
		maxW := c.RemainingWidth()
		if st.WidthDp != nil && *st.WidthDp > 0 {
			maxW = c.S(*st.WidthDp)
		}
		if st.MaxWidthDp != nil && *st.MaxWidthDp > 0 && c.S(*st.MaxWidthDp) < maxW {
			maxW = c.S(*st.MaxWidthDp)
		}
		if maxW > 0 && w > maxW {
			runes := []rune(drawText)
			cut := len(runes)
			for cut > 0 {
				cand := string(runes[:cut]) + "..."
				cw, _ := c.Font.MeasureTextWithOptions(cand, pxSize, textOpts)
				if cw <= maxW {
					drawText = cand
					w = cw
					break
				}
				cut--
			}
			if cut == 0 {
				drawText = "..."
				w, _ = c.Font.MeasureTextWithOptions(drawText, pxSize, textOpts)
			}
		}
	}
	r := c.allocRectStyled(w, h, st)
	c.Font.DrawTextWithOptions(c.Canvas, r.X, r.Y, drawText, pxSize, col, textOpts)
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
	useAutoHeight := st.AutoHeight || ((st.HeightDp == nil || *st.HeightDp <= 0) && !st.FillHeight)
	autoKey := ""
	if useAutoHeight {
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

	if useAutoHeight && autoKey != "" {
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
	id = c.ResolveID("btn", id)
	st := ApplyOptions(opts)

	fontSizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		fontSizeDp = *st.FontSizeDp
	}
	pxSize := c.S(fontSizeDp)
	textOpts := c.textRenderOpts(st)
	tw, th := 0, pxSize
	if c.Font != nil {
		tw, th = c.Font.MeasureTextWithOptions(label, pxSize, textOpts)
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
	if !st.Disabled && st.HoverTextColor != nil {
		textCol = render.LerpColor(textCol, *st.HoverTextColor, hoverAnim)
	}

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
		c.Font.DrawTextWithOptions(c.Canvas, startX, r.Y+(r.H-th)/2, label, pxSize, textCol, textOpts)
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
		if bg.A == 0 {
			hoverBg = bg
			downBg = bg
		} else {
			hoverBg = render.LerpColor(bg, render.RGB(255, 255, 255), 0.08)
			downBg = render.LerpColor(bg, render.RGB(0, 0, 0), 0.12)
		}
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
	id = c.ResolveID("tgl", id)
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

// CanvasCallbacks задаёт обработчики рисования и сырого ввода для InteractiveCanvas.
// Все координаты мыши (x, y) передаются в физических пикселях относительно левого верхнего угла area (area.X, area.Y).
type CanvasCallbacks struct {
	// OnDraw вызывается для отрисовки содержимого холста. focused == true, если холст в фокусе.
	OnDraw func(canvas *render.Canvas, area render.Rect, focused bool)
	// OnKey вызывается при KeyDown/KeyUp (со ScanCode и флагом Extended), когда холст находится в фокусе.
	OnKey func(ev RawKeyEvent)
	// OnMouseMove вызывается при движении мыши над холстом или при захвате перетаскивания (Mouse Capture).
	OnMouseMove func(x, y int, mods platform.Modifiers)
	// OnMouseDown вызывается при нажатии кнопки мыши внутри холста (x, y — физические пиксели относительно area).
	OnMouseDown func(x, y int, button platform.MouseButton, mods platform.Modifiers)
	// OnMouseUp вызывается при отпускании кнопки мыши над холстом или при активном захвате перетаскивания.
	OnMouseUp func(x, y int, button platform.MouseButton, mods platform.Modifiers)
	// OnMouseWheel вызывается при прокрутке колеса мыши над холстом (x, y — физические пиксели относительно area).
	OnMouseWheel func(x, y int, wheelX, wheelY float32, mods platform.Modifiers)
	// OnFocusLost вызывается в момент потери фокуса холстом (клик вне холста, переключение окна / Alt+Tab).
	OnFocusLost func()
	// Cursor задаёт системный курсор над областью холста (по умолчанию CursorArrow).
	Cursor platform.CursorType
	// KeepFocusNavigation отключает перехват Tab/Escape/горячих клавиш при фокусе на холсте.
	// По умолчанию (false) все клавиши передаются в OnKey без переключения фокуса по Tab/Escape.
	KeepFocusNavigation bool
	// Streaming включает fast-path в presentFrame(): при фоновом обновлении кадра (EventWakeUp)
	// пропускается computeDiffRect и копирование в prevPresentPixels, а область холста отправляется
	// напрямую в PresentRect.
	Streaming bool
	// DirectInput включает прямую доставку событий мыши и клавиатуры в коллбеки без полного Render(),
	// когда курсор и фокус находятся внутри холста и нет открытых оверлеев.
	// При необходимости перерисовать UI коллбек может вызвать RequestRedraw().
	DirectInput bool
}

// InteractiveCanvas выделяет интерактивную область wDp x hDp с поддержкой фокуса,
// захвата мыши при перетаскивании и передачи сырых событий клавиатуры (включая ScanCode/Extended) и мыши.
func (c *Context) InteractiveCanvas(id string, wDp, hDp int, cb CanvasCallbacks, opts ...Option) {
	id = c.ResolveID("canvas", id)
	_ = ApplyOptions(opts)

	w := c.S(wDp)
	if wDp <= 0 {
		w = c.RemainingWidth()
	}
	h := c.S(hDp)
	if hDp <= 0 {
		h = c.RemainingHeight()
	}
	r := c.allocRect(w, h)
	if r.Empty() {
		if c.prevFocusID == id && c.FocusID != id && cb.OnFocusLost != nil {
			cb.OnFocusLost()
		}
		return
	}

	c.registerFocusable(id)
	if cb.DirectInput {
		c.RegisterHit(id, r, cb.Cursor)
		c.nextDirectCanvases = append(c.nextDirectCanvases, directCanvasEntry{
			id:   id,
			rect: r,
			cb:   cb,
		})
	} else {
		c.RegisterHitTrack(id, r, cb.Cursor)
	}
	if cb.Streaming {
		c.SetStreamingDirtyRect(r)
		if cb.OnDraw != nil {
			c.nextStreamingCanvases = append(c.nextStreamingCanvases, streamingCanvasEntry{
				id:     id,
				rect:   r,
				onDraw: cb.OnDraw,
			})
		}
	}

	blocked := c.IsInputBlocked()
	if !blocked && r.Contains(c.MouseX, c.MouseY) {
		c.HotID = id
	}

	for _, mev := range c.rawMouseEvents {
		lx, ly := mev.X-r.X, mev.Y-r.Y
		switch mev.Kind {
		case RawMouseDown:
			if !blocked && r.Contains(mev.X, mev.Y) {
				c.FocusID = id
				c.ActiveID = id
				if cb.OnMouseDown != nil {
					cb.OnMouseDown(lx, ly, mev.Button, mev.Mods)
				}
			}
		case RawMouseMove:
			if c.ActiveID == id || (!blocked && r.Contains(mev.X, mev.Y)) {
				if cb.OnMouseMove != nil {
					cb.OnMouseMove(lx, ly, mev.Mods)
				}
			}
		case RawMouseUp:
			if c.ActiveID == id || (!blocked && r.Contains(mev.X, mev.Y)) {
				if cb.OnMouseUp != nil {
					cb.OnMouseUp(lx, ly, mev.Button, mev.Mods)
				}
			}
		case RawMouseWheel:
			if !blocked && r.Contains(mev.X, mev.Y) {
				if cb.OnMouseWheel != nil {
					cb.OnMouseWheel(lx, ly, mev.WheelX, mev.WheelY, mev.Mods)
				}
			}
		}
	}

	if c.prevFocusID == id && c.FocusID != id && cb.OnFocusLost != nil {
		cb.OnFocusLost()
	}

	focused := c.FocusID == id
	if focused {
		if !cb.KeepFocusNavigation {
			c.nextCaptureKeys = true
		}
		if cb.OnKey != nil {
			for _, kev := range c.rawKeyEvents {
				cb.OnKey(kev)
			}
		}
	}

	if cb.OnDraw != nil {
		c.Canvas.PushClip(r)
		cb.OnDraw(c.Canvas, r, focused)
		c.Canvas.PopClip()
	}
}

