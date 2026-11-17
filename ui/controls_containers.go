package ui

import (
	"strconv"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

type scrollInfo struct {
	x         int
	y         int
	contentW  int
	contentH  int
	dragging  bool
	dragOffY  int
	draggingH bool
	dragOffX  int
}

// DrawerSide задаёт сторону окна, с которой выезжает Drawer.
type DrawerSide uint8

const (
	DrawerLeft DrawerSide = iota
	DrawerRight
)

// ContextMenuItem описывает пункт кастомного всплывающего меню.
type ContextMenuItem struct {
	Label     string
	Shortcut  string
	IconSVG   string
	Separator bool
	Disabled  bool
	OnClick   func()
}

var searchQueryStore = make(map[string]string)

func (c *Context) drawDropShadow(r render.Rect, radDp float32, blurDp int, alpha uint8) {
	c.Canvas.DrawBoxShadow(r, render.BoxShadow{
		OffsetY: c.Sf(4),
		Blur:    c.Sf(float32(blurDp)),
		Color:   render.RGBA(0, 0, 0, alpha),
		Radii:   render.UniformRadii(c.Sf(radDp)),
	})
}

func (c *Context) searchQuery(key string) string {
	return searchQueryStore[key]
}

func (c *Context) setSearchQuery(key, val string) {
	searchQueryStore[key] = val
}

func (c *Context) getScrollInfo(id string) *scrollInfo {
	if c.scrollState == nil {
		c.scrollState = make(map[string]*scrollInfo)
	}
	si, ok := c.scrollState[id]
	if !ok {
		si = &scrollInfo{}
		c.scrollState[id] = si
	}
	return si
}

// ScrollView создаёт прокручиваемую область с поддержкой колеса мыши и перетаскивания скроллбара.
func (c *Context) ScrollView(id string, wDp, hDp int, fn func(), opts ...Option) {
	if fn == nil {
		return
	}
	id = c.ResolveID("sv", id)
	st := ApplyOptions(opts)
	w := c.S(wDp)
	if wDp <= 0 {
		w = c.RemainingWidth()
	}
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(hDp)
	if hDp <= 0 {
		h = c.RemainingHeight()
	}
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}

	r := c.allocRectStyled(w, h, st)
	si := c.getScrollInfo("sv:" + id)

	maxSc := si.contentH - r.H
	if maxSc < 0 {
		maxSc = 0
	}
	maxScX := si.contentW - r.W
	if maxScX < 0 {
		maxScX = 0
	}

	hovered := !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered && c.WheelY != 0 && maxSc > 0 {
		si.y -= int(c.WheelY * float32(c.S(36)))
		c.RequestRedraw()
	}
	if hovered && c.WheelX != 0 && maxScX > 0 {
		si.x += int(c.WheelX * float32(c.S(36)))
		c.RequestRedraw()
	}

	// Обработка перетаскивания вертикального и горизонтального скроллбаров
	barW := c.S(8)
	vTrackLen := r.H
	if maxScX > 0 {
		vTrackLen -= barW + c.S(2)
	}
	if maxSc > 0 && si.contentH > 0 && vTrackLen > 0 {
		thumbH := (vTrackLen * r.H) / si.contentH
		if thumbH < c.S(24) {
			thumbH = c.S(24)
		}
		if thumbH > vTrackLen {
			thumbH = vTrackLen
		}
		trackH := vTrackLen - thumbH
		thumbY := r.Y
		if trackH > 0 {
			thumbY += (trackH * si.y) / maxSc
		}
		thumbRect := render.Rect{X: r.X + r.W - barW - c.S(2), Y: thumbY, W: barW, H: thumbH}
		if hovered && c.MousePressed && thumbRect.Contains(c.MouseX, c.MouseY) {
			si.dragging = true
			si.dragOffY = c.MouseY - thumbY
			c.ActiveID = "sv_v:" + id
		}
		if si.dragging {
			if c.MouseDown && trackH > 0 {
				newThumbY := c.MouseY - si.dragOffY
				rel := newThumbY - r.Y
				if rel < 0 {
					rel = 0
				}
				if rel > trackH {
					rel = trackH
				}
				si.y = (rel * maxSc) / trackH
				c.RequestRedraw()
			} else {
				si.dragging = false
			}
		}
	}

	hTrackLen := r.W
	if maxSc > 0 {
		hTrackLen -= barW + c.S(2)
	}
	if maxScX > 0 && si.contentW > 0 && hTrackLen > 0 {
		thumbW := (hTrackLen * r.W) / si.contentW
		if thumbW < c.S(24) {
			thumbW = c.S(24)
		}
		if thumbW > hTrackLen {
			thumbW = hTrackLen
		}
		trackW := hTrackLen - thumbW
		thumbX := r.X
		if trackW > 0 {
			thumbX += (trackW * si.x) / maxScX
		}
		thumbRectH := render.Rect{X: thumbX, Y: r.Y + r.H - barW - c.S(2), W: thumbW, H: barW}
		if hovered && c.MousePressed && thumbRectH.Contains(c.MouseX, c.MouseY) {
			si.draggingH = true
			si.dragOffX = c.MouseX - thumbX
			c.ActiveID = "sv_h:" + id
		}
		if si.draggingH {
			if c.MouseDown && trackW > 0 {
				newThumbX := c.MouseX - si.dragOffX
				rel := newThumbX - r.X
				if rel < 0 {
					rel = 0
				}
				if rel > trackW {
					rel = trackW
				}
				si.x = (rel * maxScX) / trackW
				c.RequestRedraw()
			} else {
				si.draggingH = false
			}
		}
	}

	if si.y < 0 {
		si.y = 0
	}
	if si.y > maxSc {
		si.y = maxSc
	}
	if si.x < 0 {
		si.x = 0
	}
	if si.x > maxScX {
		si.x = maxScX
	}

	pad := c.S(4)
	if st.PadDp != nil && *st.PadDp >= 0 {
		pad = c.S(*st.PadDp)
	}
	innerW := r.W - pad*2
	if maxSc > 0 {
		innerW -= barW + c.S(4)
	}

	c.Canvas.PushClip(r)
	boxX := r.X + pad - si.x
	c.pushBox(render.Rect{
		X: boxX,
		Y: r.Y + pad - si.y,
		W: innerW,
		H: 1000000,
	}, dirColumn, AlignStart, c.S(8))
	startY := c.topLayout().cursorY
	fn()
	top := c.topLayout()
	si.contentH = (top.cursorY - startY) + pad*2
	if maxScX > 0 {
		si.contentH += barW + c.S(4)
	}
	if top.maxUsedX > boxX {
		si.contentW = (top.maxUsedX - boxX) + pad*2
		if maxSc > 0 {
			si.contentW += barW + c.S(4)
		}
	} else {
		si.contentW = innerW + pad*2
	}
	c.popBox()
	c.Canvas.PopClip()

	if maxSc > 0 && si.contentH > 0 && vTrackLen > 0 {
		thumbH := (vTrackLen * r.H) / si.contentH
		if thumbH < c.S(24) {
			thumbH = c.S(24)
		}
		if thumbH > vTrackLen {
			thumbH = vTrackLen
		}
		thumbY := r.Y + ((vTrackLen - thumbH) * si.y / maxSc)
		col := c.Theme.TextMuted.WithAlpha(110)
		if si.dragging {
			col = c.Theme.Accent
		}
		c.Canvas.FillRoundedRect(render.Rect{
			X: r.X + r.W - barW - c.S(2),
			Y: thumbY,
			W: barW,
			H: thumbH,
		}, c.Sf(4), col)
	}
	if maxScX > 0 && si.contentW > 0 && hTrackLen > 0 {
		thumbW := (hTrackLen * r.W) / si.contentW
		if thumbW < c.S(24) {
			thumbW = c.S(24)
		}
		if thumbW > hTrackLen {
			thumbW = hTrackLen
		}
		thumbX := r.X + ((hTrackLen - thumbW) * si.x / maxScX)
		col := c.Theme.TextMuted.WithAlpha(110)
		if si.draggingH {
			col = c.Theme.Accent
		}
		c.Canvas.FillRoundedRect(render.Rect{
			X: thumbX,
			Y: r.Y + r.H - barW - c.S(2),
			W: thumbW,
			H: barW,
		}, c.Sf(4), col)
	}
}

// Modal рисует модальное окно с затемнением фона и блокировкой ввода под ним.
func (c *Context) Modal(id string, open *bool, title string, fn func(), opts ...Option) {
	if open == nil || !*open {
		return
	}
	id = c.ResolveID("mdl", id)
	st := ApplyOptions(opts)
	w := c.S(440)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(280)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}

	for _, kp := range c.KeyEvents {
		if kp.Code == KeyEscape {
			*open = false
			c.RequestRedraw()
			return
		}
	}

	full := render.Rect{X: 0, Y: 0, W: c.Canvas.Width, H: c.Canvas.Height}
	box := render.Rect{
		X: (c.Canvas.Width - w) / 2,
		Y: (c.Canvas.Height - h) / 2,
		W: w,
		H: h,
	}

	c.PushOverlay(full, func() {
		c.Canvas.FillRect(full, render.RGBA(0, 0, 0, 140))
		c.drawDropShadow(box, c.Theme.RadiusLg, 24, 150)
		c.Canvas.DrawRoundedBox(box, c.Sf(c.Theme.RadiusLg), c.Theme.Card, 1.0, c.Theme.Border)

		headerH := c.S(42)
		if c.Font != nil && title != "" {
			pxSize := c.S(15)
			_, th := c.Font.MeasureTextStyled(title, pxSize, render.FontBold)
			c.Font.DrawTextStyled(c.Canvas, box.X+c.S(18), box.Y+(headerH-th)/2, title, pxSize, c.Theme.TextPrimary, render.FontBold)
		}

		closeR := render.Rect{X: box.X + box.W - c.S(36), Y: box.Y + c.S(9), W: c.S(24), H: c.S(24)}
		c.RegisterHit(id+":close", closeR, platform.CursorHand)
		closeHover := closeR.Contains(c.MouseX, c.MouseY)
		if closeHover {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRoundedRect(closeR, c.Sf(6), c.Theme.Surface)
			if c.MousePressed {
				*open = false
				c.RequestRedraw()
				return
			}
		}
		p := float32(c.S(6))
		c.Canvas.DrawLine(float32(closeR.X)+p, float32(closeR.Y)+p, float32(closeR.X+closeR.W)-p, float32(closeR.Y+closeR.H)-p, c.Sf(1.6), c.Theme.TextMuted)
		c.Canvas.DrawLine(float32(closeR.X+closeR.W)-p, float32(closeR.Y)+p, float32(closeR.X)+p, float32(closeR.Y+closeR.H)-p, c.Sf(1.6), c.Theme.TextMuted)

		c.Canvas.FillRect(render.Rect{X: box.X, Y: box.Y + headerH, W: box.W, H: 1}, c.Theme.Border)

		if fn != nil {
			pad := c.S(18)
			body := render.Rect{
				X: box.X + pad,
				Y: box.Y + headerH + pad,
				W: box.W - pad*2,
				H: box.H - headerH - pad*2,
			}
			c.Canvas.PushClip(body)
			c.pushBox(body, dirColumn, AlignStart, c.S(10))
			fn()
			c.popBox()
			c.Canvas.PopClip()
		}
	})
}

// Popover рисует всплывающую панель с произвольными виджетами, привязанную к области anchor.
func (c *Context) Popover(id string, open *bool, anchor render.Rect, fn func(), opts ...Option) {
	if open == nil || !*open {
		return
	}
	id = c.ResolveID("pop", id)
	st := ApplyOptions(opts)
	w := c.S(240)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(160)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}

	x := anchor.X
	y := anchor.Y + anchor.H + c.S(6)
	if x+w > c.Canvas.Width-c.S(8) {
		x = c.Canvas.Width - w - c.S(8)
	}
	if y+h > c.Canvas.Height-c.S(8) {
		y = anchor.Y - h - c.S(6)
	}
	if x < c.S(8) {
		x = c.S(8)
	}
	if y < c.S(8) {
		y = c.S(8)
	}

	popRect := render.Rect{X: x, Y: y, W: w, H: h}
	if c.MousePressed && !popRect.Contains(c.MouseX, c.MouseY) && !anchor.Contains(c.MouseX, c.MouseY) {
		*open = false
		c.RequestRedraw()
		return
	}

	c.PushOverlay(popRect, func() {
		c.drawDropShadow(popRect, c.Theme.RadiusMd, 14, 120)
		c.Canvas.DrawRoundedBox(popRect, c.Sf(c.Theme.RadiusMd), c.Theme.Card, 1.0, c.Theme.Border)
		if fn != nil {
			pad := c.S(12)
			body := render.Rect{X: popRect.X + pad, Y: popRect.Y + pad, W: popRect.W - pad*2, H: popRect.H - pad*2}
			c.Canvas.PushClip(body)
			c.pushBox(body, dirColumn, AlignStart, c.S(8))
			fn()
			c.popBox()
			c.Canvas.PopClip()
		}
	})
}

// FloatingPanel рисует перетаскиваемую плавающую панель поверх основного контента.
func (c *Context) FloatingPanel(id, title string, open *bool, fn func(), opts ...Option) {
	if open == nil || !*open {
		return
	}
	id = c.ResolveID("flt", id)
	if c.floatPos == nil {
		c.floatPos = make(map[string][2]int)
	}
	if c.floatCollapsed == nil {
		c.floatCollapsed = make(map[string]bool)
	}

	st := ApplyOptions(opts)
	w := c.S(280)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(220)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}

	pos, ok := c.floatPos[id]
	if !ok {
		pos = [2]int{c.S(40), c.S(40)}
		c.floatPos[id] = pos
	}
	collapsed := c.floatCollapsed[id]
	headerH := c.S(34)
	curH := h
	if collapsed {
		curH = headerH
	}
	panelRect := render.Rect{X: pos[0], Y: pos[1], W: w, H: curH}

	c.PushOverlay(panelRect, func() {
		headerR := render.Rect{X: panelRect.X, Y: panelRect.Y, W: panelRect.W, H: headerH}
		collapseR := render.Rect{X: panelRect.X + panelRect.W - c.S(50), Y: panelRect.Y + c.S(6), W: c.S(22), H: c.S(22)}
		closeR := render.Rect{X: panelRect.X + panelRect.W - c.S(26), Y: panelRect.Y + c.S(6), W: c.S(22), H: c.S(22)}
		c.RegisterHit(id+":hdr", headerR, platform.CursorMove)
		c.RegisterHit(id+":collapse", collapseR, platform.CursorHand)
		c.RegisterHit(id+":close", closeR, platform.CursorHand)

		if c.MousePressed && headerR.Contains(c.MouseX, c.MouseY) {
			if closeR.Contains(c.MouseX, c.MouseY) {
				*open = false
				c.RequestRedraw()
				return
			}
			if collapseR.Contains(c.MouseX, c.MouseY) {
				c.floatCollapsed[id] = !collapsed
				c.RequestRedraw()
				return
			}
			c.ActiveID = "fp_drag:" + id
			c.floatPos["__off_"+id] = [2]int{c.MouseX - pos[0], c.MouseY - pos[1]}
		}
		if c.ActiveID == "fp_drag:"+id && c.MouseDown {
			off := c.floatPos["__off_"+id]
			c.floatPos[id] = [2]int{c.MouseX - off[0], c.MouseY - off[1]}
			pos = c.floatPos[id]
			panelRect.X = pos[0]
			panelRect.Y = pos[1]
		}

		c.drawDropShadow(panelRect, c.Theme.RadiusMd, 14, 120)
		c.Canvas.DrawRoundedBox(panelRect, c.Sf(c.Theme.RadiusMd), c.Theme.Card, 1.0, c.Theme.Border)

		if c.Font != nil && title != "" {
			pxSize := c.S(13)
			_, th := c.Font.MeasureTextStyled(title, pxSize, render.FontBold)
			c.Font.DrawTextStyled(c.Canvas, panelRect.X+c.S(12), panelRect.Y+(headerH-th)/2, title, pxSize, c.Theme.TextPrimary, render.FontBold)
		}

		// Кнопка сворачивания '-'
		c.Canvas.DrawLine(float32(collapseR.X+c.S(5)), float32(collapseR.Y+collapseR.H/2), float32(collapseR.X+collapseR.W-c.S(5)), float32(collapseR.Y+collapseR.H/2), c.Sf(1.5), c.Theme.TextMuted)
		// Кнопка закрытия 'x'
		p := float32(c.S(5))
		c.Canvas.DrawLine(float32(closeR.X)+p, float32(closeR.Y)+p, float32(closeR.X+closeR.W)-p, float32(closeR.Y+closeR.H)-p, c.Sf(1.5), c.Theme.TextMuted)
		c.Canvas.DrawLine(float32(closeR.X+closeR.W)-p, float32(closeR.Y)+p, float32(closeR.X)+p, float32(closeR.Y+closeR.H)-p, c.Sf(1.5), c.Theme.TextMuted)

		if !collapsed && fn != nil {
			c.Canvas.FillRect(render.Rect{X: panelRect.X, Y: panelRect.Y + headerH, W: panelRect.W, H: 1}, c.Theme.Border)
			pad := c.S(12)
			body := render.Rect{X: panelRect.X + pad, Y: panelRect.Y + headerH + pad, W: panelRect.W - pad*2, H: panelRect.H - headerH - pad*2}
			c.Canvas.PushClip(body)
			c.pushBox(body, dirColumn, AlignStart, c.S(8))
			fn()
			c.popBox()
			c.Canvas.PopClip()
		}
	})
}

// Drawer рисует выезжающую от края окна боковую панель с плавной анимацией.
func (c *Context) Drawer(id string, open *bool, side DrawerSide, fn func(), opts ...Option) {
	if open == nil {
		return
	}
	id = c.ResolveID("drw", id)
	target := float32(0)
	if *open {
		target = 1
	}
	anim := c.animFloat("drw:"+id, target, 0.18)
	if anim <= 0.01 && !*open {
		return
	}

	st := ApplyOptions(opts)
	w := c.S(280)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}

	x := -w + int(float32(w)*anim)
	if side == DrawerRight {
		x = c.Canvas.Width - int(float32(w)*anim)
	}
	panelRect := render.Rect{X: x, Y: 0, W: w, H: c.Canvas.Height}
	full := render.Rect{X: 0, Y: 0, W: c.Canvas.Width, H: c.Canvas.Height}

	if *open && c.MousePressed && !panelRect.Contains(c.MouseX, c.MouseY) {
		*open = false
		c.RequestRedraw()
	}

	c.PushOverlay(full, func() {
		c.Canvas.FillRect(full, render.RGBA(0, 0, 0, uint8(90*anim)))
		c.drawDropShadow(panelRect, 0, 18, 120)
		c.Canvas.FillRect(panelRect, c.Theme.Card)
		if fn != nil {
			pad := c.S(16)
			body := render.Rect{X: panelRect.X + pad, Y: panelRect.Y + pad, W: panelRect.W - pad*2, H: panelRect.H - pad*2}
			c.Canvas.PushClip(body)
			c.pushBox(body, dirColumn, AlignStart, c.S(10))
			fn()
			c.popBox()
			c.Canvas.PopClip()
		}
	})
}

// ContextMenu открывает всплывающее меню по правому клику внутри anchor.
func (c *Context) ContextMenu(id string, anchor render.Rect, items []ContextMenuItem) {
	if len(items) == 0 {
		return
	}
	id = c.ResolveID("ctx", id)
	if c.openState == nil {
		c.openState = make(map[string]bool)
	}
	if c.floatPos == nil {
		c.floatPos = make(map[string][2]int)
	}

	if c.RightMousePressed && anchor.Contains(c.RightMouseX, c.RightMouseY) {
		c.openState["cm:"+id] = true
		c.floatPos["cm:"+id] = [2]int{c.RightMouseX, c.RightMouseY}
		c.RequestRedraw()
	}

	if !c.openState["cm:"+id] {
		return
	}

	pos := c.floatPos["cm:"+id]
	w := c.S(200)
	itemH := c.S(30)
	sepH := c.S(9)
	totalH := c.S(8)
	for _, it := range items {
		if it.Separator {
			totalH += sepH
		} else {
			totalH += itemH
		}
	}

	x, y := pos[0], pos[1]
	if x+w > c.Canvas.Width {
		x = c.Canvas.Width - w - c.S(4)
	}
	if y+totalH > c.Canvas.Height {
		y = c.Canvas.Height - totalH - c.S(4)
	}
	menuRect := render.Rect{X: x, Y: y, W: w, H: totalH}

	if c.MousePressed && !menuRect.Contains(c.MouseX, c.MouseY) {
		c.openState["cm:"+id] = false
		c.RequestRedraw()
		return
	}

	c.PushOverlay(menuRect, func() {
		c.drawDropShadow(menuRect, c.Theme.RadiusSm, 12, 120)
		c.Canvas.DrawRoundedBox(menuRect, c.Sf(c.Theme.RadiusSm), c.Theme.Card, 1.0, c.Theme.Border)

		curY := menuRect.Y + c.S(4)
		pxSize := c.S(13)
		for idx, it := range items {
			if it.Separator {
				c.Canvas.FillRect(render.Rect{X: menuRect.X + c.S(8), Y: curY + sepH/2, W: menuRect.W - c.S(16), H: 1}, c.Theme.Border)
				curY += sepH
				continue
			}
			ir := render.Rect{X: menuRect.X + c.S(4), Y: curY, W: menuRect.W - c.S(8), H: itemH}
			if !it.Disabled {
				c.RegisterHit(id+":cm:"+strconv.Itoa(idx), ir, platform.CursorHand)
			}
			hov := !it.Disabled && ir.Contains(c.MouseX, c.MouseY)
			if hov {
				c.nextCursor = platform.CursorHand
				c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Surface)
				if c.MousePressed {
					c.openState["cm:"+id] = false
					if it.OnClick != nil {
						it.OnClick()
					}
					c.RequestRedraw()
				}
			}
			if c.Font != nil {
				col := c.Theme.TextPrimary
				if it.Disabled {
					col = c.Theme.TextMuted
				}
				_, th := c.Font.MeasureText(it.Label, pxSize)
				c.Font.DrawText(c.Canvas, ir.X+c.S(10), ir.Y+(ir.H-th)/2, it.Label, pxSize, col)
				if it.Shortcut != "" {
					sw, sh := c.Font.MeasureText(it.Shortcut, c.S(11))
					c.Font.DrawText(c.Canvas, ir.X+ir.W-sw-c.S(10), ir.Y+(ir.H-sh)/2, it.Shortcut, c.S(11), c.Theme.TextMuted)
				}
			}
			curY += itemH
		}
	})
}

// Tooltip отображает всплывающую подсказку, когда курсор находится над областью anchor.
func (c *Context) Tooltip(text string, anchor render.Rect) {
	if text == "" || c.Font == nil || c.IsInputBlocked() || !anchor.Contains(c.MouseX, c.MouseY) {
		return
	}
	c.RegisterHitTrack("tip:"+text, anchor, c.nextCursor)
	pxSize := c.S(12)
	tw, th := c.Font.MeasureText(text, pxSize)
	padX, padY := c.S(10), c.S(6)
	tipW, tipH := tw+padX*2, th+padY*2
	x := c.MouseX + c.S(12)
	y := c.MouseY + c.S(18)
	if x+tipW > c.Canvas.Width-c.S(4) {
		x = c.Canvas.Width - tipW - c.S(4)
	}
	if y+tipH > c.Canvas.Height-c.S(4) {
		y = c.MouseY - tipH - c.S(8)
	}
	tipRect := render.Rect{X: x, Y: y, W: tipW, H: tipH}
	c.PushOverlay(tipRect, func() {
		c.drawDropShadow(tipRect, c.Theme.RadiusSm, 8, 110)
		c.Canvas.DrawRoundedBox(tipRect, c.Sf(c.Theme.RadiusSm), c.Theme.Card, 1.0, c.Theme.Border)
		c.Font.DrawText(c.Canvas, tipRect.X+padX, tipRect.Y+padY, text, pxSize, c.Theme.TextPrimary)
	})
}
