package ui

import (
	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

// Orientation задаёт направление ориентации элементов интерфейса (горизонтальное или вертикальное).
type Orientation uint8

const (
	// Horizontal задаёт горизонтальную ориентацию.
	Horizontal Orientation = iota
	// Vertical задаёт вертикальную ориентацию.
	Vertical
)

// SplitOrientation задаёт направление разделения в SplitPane (псевдоним Orientation).
type SplitOrientation = Orientation

const (
	// SplitHorizontal разделяет область на левую и правую панели с вертикальным разделителем.
	SplitHorizontal = Horizontal
	// SplitVertical разделяет область на верхнюю и нижнюю панели с горизонтальным разделителем.
	SplitVertical = Vertical
)

// SplitPane делит область на 2 панели с перетаскиваемым разделителем.
// ratio задаёт долю первой панели в диапазоне [0.1, 0.9].
func (c *Context) SplitPane(id string, orient Orientation, ratio *float32, firstFn, secondFn func(), opts ...Option) {
	if ratio == nil {
		return
	}
	id = c.ResolveID("split", id)
	if *ratio <= 0 {
		*ratio = 0.5
	}
	if *ratio < 0.1 {
		*ratio = 0.1
	} else if *ratio > 0.9 {
		*ratio = 0.9
	}

	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.RemainingHeight()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	} else if h <= 0 || (!st.FillHeight && st.HeightDp == nil && h >= c.S(50000)) {
		h = c.S(260)
	}

	rect := c.allocRectStyled(w, h, st)
	if rect.W <= 0 || rect.H <= 0 {
		return
	}
	radius := c.Sf(c.Theme.RadiusSm)
	if st.Radius != nil {
		radius = c.Sf(*st.Radius)
	}
	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, c.Theme.Border)

	splitterThick := c.S(8)
	blocked := c.IsInputBlocked()
	var firstR, splitR, secondR render.Rect

	if orient == SplitHorizontal {
		availW := rect.W - splitterThick - 2
		if availW < c.S(40) {
			availW = c.S(40)
		}
		firstW := int(float32(availW) * (*ratio))
		firstR = render.Rect{X: rect.X + 1, Y: rect.Y + 1, W: firstW, H: rect.H - 2}
		splitR = render.Rect{X: firstR.X + firstR.W, Y: rect.Y + 1, W: splitterThick, H: rect.H - 2}
		secondR = render.Rect{X: splitR.X + splitR.W, Y: rect.Y + 1, W: rect.X + rect.W - 1 - (splitR.X + splitR.W), H: rect.H - 2}

		hover := !blocked && splitR.Contains(c.MouseX, c.MouseY)
		if hover || c.ActiveID == id {
			c.nextCursor = platform.CursorResizeH
			if hover && c.MousePressed {
				c.ActiveID = id
			}
		}
		if c.ActiveID == id && c.MouseDown && availW > 0 {
			newRatio := float32(c.MouseX-(rect.X+1)-splitterThick/2) / float32(availW)
			if newRatio < 0.1 {
				newRatio = 0.1
			} else if newRatio > 0.9 {
				newRatio = 0.9
			}
			*ratio = newRatio
			c.RequestRedraw()
		}
	} else {
		availH := rect.H - splitterThick - 2
		if availH < c.S(40) {
			availH = c.S(40)
		}
		firstH := int(float32(availH) * (*ratio))
		firstR = render.Rect{X: rect.X + 1, Y: rect.Y + 1, W: rect.W - 2, H: firstH}
		splitR = render.Rect{X: rect.X + 1, Y: firstR.Y + firstR.H, W: rect.W - 2, H: splitterThick}
		secondR = render.Rect{X: rect.X + 1, Y: splitR.Y + splitR.H, W: rect.W - 2, H: rect.Y + rect.H - 1 - (splitR.Y + splitR.H)}

		hover := !blocked && splitR.Contains(c.MouseX, c.MouseY)
		if hover || c.ActiveID == id {
			c.nextCursor = platform.CursorResizeV
			if hover && c.MousePressed {
				c.ActiveID = id
			}
		}
		if c.ActiveID == id && c.MouseDown && availH > 0 {
			newRatio := float32(c.MouseY-(rect.Y+1)-splitterThick/2) / float32(availH)
			if newRatio < 0.1 {
				newRatio = 0.1
			} else if newRatio > 0.9 {
				newRatio = 0.9
			}
			*ratio = newRatio
			c.RequestRedraw()
		}
	}

	// Отрисовка первой панели
	if firstFn != nil && firstR.W > 0 && firstR.H > 0 {
		pad := c.S(8)
		inner := render.Rect{X: firstR.X + pad, Y: firstR.Y + pad, W: firstR.W - pad*2, H: firstR.H - pad*2}
		if inner.W > 0 && inner.H > 0 {
			c.Canvas.PushClip(firstR)
			c.pushBox(inner, dirColumn, AlignStart, c.S(6))
			firstFn()
			c.popBox()
			c.Canvas.PopClip()
		}
	}

	// Отрисовка разделителя
	splitCur := platform.CursorResizeV
	if orient == SplitHorizontal {
		splitCur = platform.CursorResizeH
	}
	c.RegisterHit(id+":split", splitR, splitCur)
	splitBg := c.Theme.Surface
	gripCol := c.Theme.Border
	if c.ActiveID == id || (!blocked && splitR.Contains(c.MouseX, c.MouseY)) {
		splitBg = c.Theme.BtnSecondaryHover
		gripCol = c.Theme.Accent
	}
	c.Canvas.FillRect(splitR, splitBg)
	if orient == SplitHorizontal {
		cx := float32(splitR.X + splitR.W/2)
		cy := float32(splitR.Y + splitR.H/2)
		c.Canvas.DrawLine(cx, cy-c.Sf(14), cx, cy+c.Sf(14), c.Sf(2), gripCol)
	} else {
		cx := float32(splitR.X + splitR.W/2)
		cy := float32(splitR.Y + splitR.H/2)
		c.Canvas.DrawLine(cx-c.Sf(14), cy, cx+c.Sf(14), cy, c.Sf(2), gripCol)
	}

	// Отрисовка второй панели
	if secondFn != nil && secondR.W > 0 && secondR.H > 0 {
		pad := c.S(8)
		inner := render.Rect{X: secondR.X + pad, Y: secondR.Y + pad, W: secondR.W - pad*2, H: secondR.H - pad*2}
		if inner.W > 0 && inner.H > 0 {
			c.Canvas.PushClip(secondR)
			c.pushBox(inner, dirColumn, AlignStart, c.S(6))
			secondFn()
			c.popBox()
			c.Canvas.PopClip()
		}
	}
}

// Collapsible рисует сворачиваемую секцию с заголовком и содержимым.
// Возвращает true, если состояние раскрытия изменилось в текущем кадре.
func (c *Context) Collapsible(id, title string, expanded *bool, fn func(), opts ...Option) bool {
	if expanded == nil {
		return false
	}
	id = c.ResolveID("coll", id)
	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	headerH := c.S(34)
	headerR := c.allocRect(w, headerH)
	c.RegisterHit(id, headerR, platform.CursorHand)

	hovered := !c.IsInputBlocked() && headerR.Contains(c.MouseX, c.MouseY)
	changed := false
	if hovered {
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			*expanded = !*expanded
			changed = true
			c.RequestRedraw()
		}
	}

	bg := c.Theme.Surface
	if hovered {
		bg = c.Theme.BtnSecondaryHover
	}
	radius := c.Sf(c.Theme.RadiusSm)
	c.Canvas.DrawRoundedBox(headerR, radius, bg, 1.0, c.Theme.Border)

	// Стрелка раскрытия
	cx := float32(headerR.X + c.S(14))
	cy := float32(headerR.Y + headerR.H/2)
	arm := c.Sf(4)
	if *expanded {
		c.Canvas.DrawLine(cx-arm, cy-arm*0.5, cx, cy+arm*0.5, c.Sf(1.6), c.Theme.Accent)
		c.Canvas.DrawLine(cx, cy+arm*0.5, cx+arm, cy-arm*0.5, c.Sf(1.6), c.Theme.Accent)
	} else {
		c.Canvas.DrawLine(cx-arm*0.5, cy-arm, cx+arm*0.5, cy, c.Sf(1.6), c.Theme.TextMuted)
		c.Canvas.DrawLine(cx+arm*0.5, cy, cx-arm*0.5, cy+arm, c.Sf(1.6), c.Theme.TextMuted)
	}

	if c.Font != nil {
		pxSize := c.S(13)
		_, th := c.Font.MeasureText(title, pxSize)
		c.Font.DrawText(c.Canvas, headerR.X+c.S(26), headerR.Y+(headerR.H-th)/2, title, pxSize, c.Theme.TextPrimary)
	}

	if *expanded && fn != nil {
		b := c.topLayout()
		startY := b.cursorY
		c.PaddingXY(12, 8, func() {
			c.Column(6, fn)
		})
		endY := b.cursorY
		if endY > startY {
			c.Canvas.DrawLine(float32(headerR.X+c.S(6)), float32(startY+c.S(4)), float32(headerR.X+c.S(6)), float32(endY-c.S(4)), c.Sf(2), c.Theme.Accent.WithAlpha(140))
		}
	}

	return changed
}

// AccordionItem описывает одну секцию аккордеона.
type AccordionItem struct {
	ID    string
	Title string
	Draw  func()
}

// Accordion рисует группу сворачиваемых секций с независимым состоянием раскрытия по строковым ID.
// В openIDs хранится признак открытости каждой секции (true — раскрыта).
func (c *Context) Accordion(id string, items []AccordionItem, openIDs map[string]bool, opts ...Option) bool {
	if openIDs == nil {
		return false
	}
	id = c.nextLocalID("acc", id)
	changed := false
	for _, it := range items {
		exp := openIDs[it.ID]
		if c.Collapsible(id+":"+it.ID, it.Title, &exp, it.Draw, opts...) {
			openIDs[it.ID] = exp
			changed = true
		}
	}
	return changed
}
