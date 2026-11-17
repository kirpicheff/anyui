package ui

import (
	"strconv"

	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/render"
)

// BarMenuItem описывает пункт выпадающего меню в MenuBar (поддерживает вложенные подменю).
type BarMenuItem struct {
	Label     string
	Shortcut  string
	IconSVG   string
	Separator bool
	Disabled  bool
	Checked   bool
	SubItems  []BarMenuItem
	OnClick   func()
}

// MenuBarItem описывает раздел верхнего уровня в MenuBar (например, «Файл», «Правка»).
type MenuBarItem struct {
	Label string
	Items []BarMenuItem
}

// ToolbarItem описывает элемент панели инструментов Toolbar.
type ToolbarItem struct {
	ID        string
	Label     string
	IconSVG   string
	Tooltip   string
	Separator bool
	Disabled  bool
	Active    bool
	OnClick   func()
}

// StatusBarSection описывает секцию нижней строки состояния StatusBar.
type StatusBarSection struct {
	Text     string
	WidthDp  int
	Flex     float32
	DotColor *render.Color
	AlignEnd bool
	OnClick  func()
}

// BreadcrumbItem описывает сегмент навигационной цепочки Breadcrumbs.
type BreadcrumbItem struct {
	Label    string
	IconSVG  string
	Disabled bool
}

var (
	menuBarActiveIdx = make(map[string]int)
	menuBarHoverSub  = make(map[string]int)
)

func (c *Context) allocWidth(hDp int, st WidgetStyle) render.Rect {
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	return c.allocRectStyled(w, c.S(hDp), st)
}

func (c *Context) hasKey(code uint32) bool {
	for _, kp := range c.KeyEvents {
		if kp.Code == code {
			return true
		}
	}
	return false
}

func (c *Context) currentMods() platform.Modifiers {
	return c.MouseMods
}

// MenuBar отрисовывает главное верхнее меню окна с выпадающими подменю и хоткеями.
func (c *Context) MenuBar(id string, menus []MenuBarItem, opts ...Option) {
	id = c.ResolveID("mbar", id)
	st := ApplyOptions(opts)
	hDp := 30
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	bg := c.Theme.Surface
	if st.Bg != nil {
		bg = *st.Bg
	}
	c.Canvas.FillRect(r, bg)
	c.Canvas.FillRect(render.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, c.Theme.Border)

	if len(menus) == 0 {
		return
	}

	activeIdx, isOpen := menuBarActiveIdx[id]
	if !isOpen {
		activeIdx = -1
	}

	// Закрытие по Escape
	if isOpen && c.hasKey(KeyEscape) {
		delete(menuBarActiveIdx, id)
		delete(menuBarHoverSub, id)
		c.RequestRedraw()
		isOpen = false
		activeIdx = -1
	}

	pxSize := c.S(13)
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		pxSize = c.S(*st.FontSizeDp)
	}
	padX := c.S(12)
	curX := r.X + c.S(4)

	var itemRects []render.Rect
	for i, m := range menus {
		tw := 0
		if c.Font != nil {
			tw, _ = c.Font.MeasureText(m.Label, pxSize)
		}
		iw := tw + padX*2
		ir := render.Rect{X: curX, Y: r.Y + c.S(2), W: iw, H: r.H - c.S(4)}
		itemRects = append(itemRects, ir)
		curX += iw + c.S(2)
		c.RegisterHit(id+":m:"+strconv.Itoa(i), ir, platform.CursorHand)

		hov := ir.Contains(c.MouseX, c.MouseY)
		selected := isOpen && activeIdx == i

		if selected {
			c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Accent)
		} else if hov && !c.IsInputBlocked() {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Card)
		}

		if c.Font != nil {
			col := c.Theme.TextPrimary
			if selected {
				col = render.RGBA(255, 255, 255, 255)
			}
			_, th := c.Font.MeasureText(m.Label, pxSize)
			c.Font.DrawText(c.Canvas, ir.X+padX, ir.Y+(ir.H-th)/2, m.Label, pxSize, col)
		}

		if hov {
			if c.MousePressed && !c.IsInputBlocked() {
				if selected {
					delete(menuBarActiveIdx, id)
					delete(menuBarHoverSub, id)
				} else {
					menuBarActiveIdx[id] = i
					delete(menuBarHoverSub, id)
				}
				c.RequestRedraw()
			} else if isOpen && activeIdx != i {
				menuBarActiveIdx[id] = i
				delete(menuBarHoverSub, id)
				c.RequestRedraw()
			}
		}
	}

	activeIdx, isOpen = menuBarActiveIdx[id]
	if !isOpen || activeIdx < 0 || activeIdx >= len(menus) {
		return
	}

	anchor := itemRects[activeIdx]
	items := menus[activeIdx].Items
	if len(items) == 0 {
		return
	}

	menuRect := c.calcMenuPopupRect(anchor.X, anchor.Y+anchor.H+c.S(2), items)

	var subRect render.Rect
	var subItems []BarMenuItem
	if hoverSubIdx, ok := menuBarHoverSub[id]; ok && hoverSubIdx >= 0 && hoverSubIdx < len(items) {
		if len(items[hoverSubIdx].SubItems) > 0 && !items[hoverSubIdx].Disabled {
			subItems = items[hoverSubIdx].SubItems
			subY := c.calcMenuItemY(menuRect, items, hoverSubIdx)
			subX := menuRect.X + menuRect.W - c.S(2)
			subRect = c.calcMenuPopupRect(subX, subY, subItems)
			if subRect.X+subRect.W > c.Canvas.Width {
				subRect.X = menuRect.X - subRect.W + c.S(2)
			}
		}
	}

	if c.MousePressed {
		inBar := false
		for _, ir := range itemRects {
			if ir.Contains(c.MouseX, c.MouseY) {
				inBar = true
				break
			}
		}
		inMenu := menuRect.Contains(c.MouseX, c.MouseY) || (!subRect.Empty() && subRect.Contains(c.MouseX, c.MouseY))
		if !inBar && !inMenu {
			delete(menuBarActiveIdx, id)
			delete(menuBarHoverSub, id)
			c.RequestRedraw()
			return
		}
	}

	c.PushOverlay(menuRect, func() {
		c.drawMenuPopup(id, menuRect, items, true)
	})
	if !subRect.Empty() && len(subItems) > 0 {
		c.PushOverlay(subRect, func() {
			c.drawMenuPopup(id, subRect, subItems, false)
		})
	}
}

func (c *Context) calcMenuPopupRect(x, y int, items []BarMenuItem) render.Rect {
	w := c.S(220)
	itemH := c.S(28)
	sepH := c.S(9)
	totalH := c.S(8)
	for _, it := range items {
		if it.Separator {
			totalH += sepH
		} else {
			totalH += itemH
		}
	}
	if x+w > c.Canvas.Width-c.S(4) {
		x = c.Canvas.Width - w - c.S(4)
	}
	if x < c.S(4) {
		x = c.S(4)
	}
	if y+totalH > c.Canvas.Height-c.S(4) {
		y = c.Canvas.Height - totalH - c.S(4)
	}
	if y < c.S(4) {
		y = c.S(4)
	}
	return render.Rect{X: x, Y: y, W: w, H: totalH}
}

func (c *Context) calcMenuItemY(menuRect render.Rect, items []BarMenuItem, targetIdx int) int {
	curY := menuRect.Y + c.S(4)
	itemH := c.S(28)
	sepH := c.S(9)
	for i, it := range items {
		if i == targetIdx {
			return curY
		}
		if it.Separator {
			curY += sepH
		} else {
			curY += itemH
		}
	}
	return curY
}

func (c *Context) drawMenuPopup(barID string, menuRect render.Rect, items []BarMenuItem, isRoot bool) {
	c.drawDropShadow(menuRect, c.Theme.RadiusSm, 12, 120)
	c.Canvas.DrawRoundedBox(menuRect, c.Sf(c.Theme.RadiusSm), c.Theme.Card, 1.0, c.Theme.Border)

	itemH := c.S(28)
	sepH := c.S(9)
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
			c.RegisterHit(barID+":mi:"+strconv.Itoa(idx)+":"+it.Label, ir, platform.CursorHand)
		}
		hov := !it.Disabled && ir.Contains(c.MouseX, c.MouseY)
		if hov {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Surface)
			if isRoot {
				if len(it.SubItems) > 0 {
					if menuBarHoverSub[barID] != idx {
						menuBarHoverSub[barID] = idx
						c.RequestRedraw()
					}
				} else if _, ok := menuBarHoverSub[barID]; ok {
					delete(menuBarHoverSub, barID)
					c.RequestRedraw()
				}
			}
			if c.MousePressed && len(it.SubItems) == 0 {
				delete(menuBarActiveIdx, barID)
				delete(menuBarHoverSub, barID)
				if it.OnClick != nil {
					it.OnClick()
				}
				c.RequestRedraw()
			}
		}

		textX := ir.X + c.S(24)
		iconSize := c.S(14)
		if it.Checked {
			cx := float32(ir.X + c.S(10))
			cy := float32(ir.Y + ir.H/2)
			col := c.Theme.Accent
			if it.Disabled {
				col = c.Theme.TextMuted
			}
			c.Canvas.DrawLine(cx-c.Sf(3), cy, cx, cy+c.Sf(3), 1.8, col)
			c.Canvas.DrawLine(cx, cy+c.Sf(3), cx+c.Sf(5), cy-c.Sf(4), 1.8, col)
		} else if it.IconSVG != "" {
			col := c.Theme.TextMuted
			c.Canvas.DrawSVG(it.IconSVG, ir.X+c.S(6), ir.Y+(ir.H-iconSize)/2, iconSize, col)
		}

		if c.Font != nil {
			col := c.Theme.TextPrimary
			if it.Disabled {
				col = c.Theme.TextMuted
			}
			_, th := c.Font.MeasureText(it.Label, pxSize)
			c.Font.DrawText(c.Canvas, textX, ir.Y+(ir.H-th)/2, it.Label, pxSize, col)

			if len(it.SubItems) > 0 {
				aw, ah := c.Font.MeasureText(">", c.S(11))
				c.Font.DrawText(c.Canvas, ir.X+ir.W-aw-c.S(8), ir.Y+(ir.H-ah)/2, ">", c.S(11), c.Theme.TextMuted)
			} else if it.Shortcut != "" {
				sw, sh := c.Font.MeasureText(it.Shortcut, c.S(11))
				c.Font.DrawText(c.Canvas, ir.X+ir.W-sw-c.S(10), ir.Y+(ir.H-sh)/2, it.Shortcut, c.S(11), c.Theme.TextMuted)
			}
		}
		curY += itemH
	}
}

// Toolbar отрисовывает панель инструментов с кнопками, иконками, разделителями и подсказками.
func (c *Context) Toolbar(id string, items []ToolbarItem, opts ...Option) {
	id = c.ResolveID("tbar", id)
	st := ApplyOptions(opts)
	hDp := 38
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	bg := c.Theme.Surface
	if st.Bg != nil {
		bg = *st.Bg
	}
	c.Canvas.DrawRoundedBox(r, c.Sf(c.Theme.RadiusSm), bg, 1.0, c.Theme.Border)

	curX := r.X + c.S(6)
	btnH := r.H - c.S(8)
	btnY := r.Y + c.S(4)
	pxSize := c.S(12)
	iconSize := c.S(16)

	for ti, it := range items {
		if it.Separator {
			c.Canvas.FillRect(render.Rect{X: curX + c.S(3), Y: r.Y + c.S(7), W: 1, H: r.H - c.S(14)}, c.Theme.Border)
			curX += c.S(8)
			continue
		}

		tw := 0
		if it.Label != "" && c.Font != nil {
			tw, _ = c.Font.MeasureText(it.Label, pxSize)
		}
		w := c.S(12)
		if it.IconSVG != "" {
			w += iconSize
		}
		if it.IconSVG != "" && it.Label != "" {
			w += c.S(6)
		}
		if it.Label != "" {
			w += tw
		}
		if w < btnH {
			w = btnH
		}

		br := render.Rect{X: curX, Y: btnY, W: w, H: btnH}
		curX += w + c.S(4)
		if curX > r.X+r.W {
			break
		}
		if !it.Disabled {
			if it.Tooltip != "" {
				c.RegisterHitTrack(id+":tb:"+strconv.Itoa(ti), br, platform.CursorHand)
			} else {
				c.RegisterHit(id+":tb:"+strconv.Itoa(ti), br, platform.CursorHand)
			}
		}

		hov := !it.Disabled && !c.IsInputBlocked() && br.Contains(c.MouseX, c.MouseY)
		if it.Active {
			c.Canvas.DrawRoundedBox(br, c.Sf(4), c.Theme.Accent.WithAlpha(40), 1.0, c.Theme.Accent)
		} else if hov {
			c.nextCursor = platform.CursorHand
			bgBtn := c.Theme.Card
			if c.MouseDown {
				bgBtn = c.Theme.Border
			}
			c.Canvas.FillRoundedRect(br, c.Sf(4), bgBtn)
		}

		contentX := br.X + c.S(6)
		if it.IconSVG != "" && it.Label == "" {
			contentX = br.X + (br.W-iconSize)/2
		}

		fg := c.Theme.TextPrimary
		if it.Disabled {
			fg = c.Theme.TextMuted
		} else if it.Active {
			fg = c.Theme.Accent
		}

		if it.IconSVG != "" {
			c.Canvas.DrawSVG(it.IconSVG, contentX, br.Y+(br.H-iconSize)/2, iconSize, fg)
			contentX += iconSize + c.S(6)
		}
		if it.Label != "" && c.Font != nil {
			_, th := c.Font.MeasureText(it.Label, pxSize)
			c.Font.DrawText(c.Canvas, contentX, br.Y+(br.H-th)/2, it.Label, pxSize, fg)
		}

		if hov && c.MousePressed && it.OnClick != nil {
			it.OnClick()
			c.RequestRedraw()
		}
		if it.Tooltip != "" && br.Contains(c.MouseX, c.MouseY) {
			c.Tooltip(it.Tooltip, br)
		}
	}
}

// StatusBar отрисовывает нижнюю строку состояния с секциями и индикаторами.
func (c *Context) StatusBar(id string, sections []StatusBarSection, opts ...Option) {
	id = c.ResolveID("sbar", id)
	st := ApplyOptions(opts)
	hDp := 26
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	bg := c.Theme.Surface
	if st.Bg != nil {
		bg = *st.Bg
	}
	c.Canvas.FillRect(r, bg)
	c.Canvas.FillRect(render.Rect{X: r.X, Y: r.Y, W: r.W, H: 1}, c.Theme.Border)

	if len(sections) == 0 {
		return
	}

	pxSize := c.S(11)
	padX := c.S(10)

	widths := make([]int, len(sections))
	fixedSum := 0
	var totalFlex float32
	for i, sec := range sections {
		if sec.WidthDp > 0 {
			widths[i] = c.S(sec.WidthDp)
			fixedSum += widths[i]
		} else if sec.Flex > 0 {
			totalFlex += sec.Flex
		} else {
			w := padX * 2
			if sec.DotColor != nil {
				w += c.S(12)
			}
			if c.Font != nil && sec.Text != "" {
				tw, _ := c.Font.MeasureText(sec.Text, pxSize)
				w += tw
			}
			widths[i] = w
			fixedSum += w
		}
	}

	remW := r.W - fixedSum
	if remW < 0 {
		remW = 0
	}
	if totalFlex > 0 {
		for i, sec := range sections {
			if sec.WidthDp <= 0 && sec.Flex > 0 {
				widths[i] = int(float32(remW) * (sec.Flex / totalFlex))
			}
		}
	}

	curX := r.X
	for i, sec := range sections {
		w := widths[i]
		sr := render.Rect{X: curX, Y: r.Y + 1, W: w, H: r.H - 1}
		curX += w

		if i > 0 {
			c.Canvas.FillRect(render.Rect{X: sr.X, Y: sr.Y + c.S(4), W: 1, H: sr.H - c.S(8)}, c.Theme.Border)
		}

		if sec.OnClick != nil {
			c.RegisterHit(id+":sec:"+strconv.Itoa(i), sr, platform.CursorHand)
		}
		hov := sec.OnClick != nil && !c.IsInputBlocked() && sr.Contains(c.MouseX, c.MouseY)
		if hov {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRect(sr, c.Theme.Card)
			if c.MousePressed {
				sec.OnClick()
				c.RequestRedraw()
			}
		}

		c.Canvas.PushClip(sr)
		dotOff := 0
		if sec.DotColor != nil {
			dotR := c.Sf(3.5)
			dotX := float32(sr.X+padX) + dotR
			dotY := float32(sr.Y) + float32(sr.H)*0.5
			c.Canvas.FillCircle(dotX, dotY, dotR, *sec.DotColor)
			dotOff = c.S(12)
		}

		if c.Font != nil && sec.Text != "" {
			tw, th := c.Font.MeasureText(sec.Text, pxSize)
			tx := sr.X + padX + dotOff
			if sec.AlignEnd {
				tx = sr.X + sr.W - padX - tw
			}
			c.Font.DrawText(c.Canvas, tx, sr.Y+(sr.H-th)/2, sec.Text, pxSize, c.Theme.TextMuted)
		}
		c.Canvas.PopClip()
	}
}

// Breadcrumbs отрисовывает навигационную цепочку и возвращает индекс кликнутого сегмента (-1 если клика не было).
func (c *Context) Breadcrumbs(id string, items []BreadcrumbItem, opts ...Option) int {
	id = c.ResolveID("bc", id)
	st := ApplyOptions(opts)
	hDp := 28
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)
	if len(items) == 0 {
		return -1
	}

	clicked := -1
	pxSize := c.S(13)
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		pxSize = c.S(*st.FontSizeDp)
	}
	iconSize := c.S(14)
	curX := r.X

	for i, it := range items {
		isLast := i == len(items)-1
		if i > 0 {
			sep := "/"
			if c.Font != nil {
				sw, sh := c.Font.MeasureText(sep, pxSize)
				c.Font.DrawText(c.Canvas, curX+c.S(4), r.Y+(r.H-sh)/2, sep, pxSize, c.Theme.TextMuted)
				curX += sw + c.S(10)
			} else {
				curX += c.S(12)
			}
		}

		tw := 0
		if c.Font != nil {
			tw, _ = c.Font.MeasureText(it.Label, pxSize)
		}
		segW := tw + c.S(8)
		if it.IconSVG != "" {
			segW += iconSize + c.S(4)
		}
		sr := render.Rect{X: curX, Y: r.Y, W: segW, H: r.H}
		curX += segW

		canClick := !isLast && !it.Disabled && !c.IsInputBlocked()
		if !isLast && !it.Disabled {
			c.RegisterHit(id+":bc:"+strconv.Itoa(i), sr, platform.CursorHand)
		}
		hov := canClick && sr.Contains(c.MouseX, c.MouseY)
		if hov {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRoundedRect(sr, c.Sf(4), c.Theme.Surface)
			if c.MousePressed {
				clicked = i
				c.RequestRedraw()
			}
		}

		fg := c.Theme.TextMuted
		if isLast {
			fg = c.Theme.TextPrimary
		} else if hov {
			fg = c.Theme.Accent
		}

		tx := sr.X + c.S(4)
		if it.IconSVG != "" {
			c.Canvas.DrawSVG(it.IconSVG, tx, sr.Y+(sr.H-iconSize)/2, iconSize, fg)
			tx += iconSize + c.S(4)
		}
		if c.Font != nil {
			_, th := c.Font.MeasureText(it.Label, pxSize)
			c.Font.DrawText(c.Canvas, tx, sr.Y+(sr.H-th)/2, it.Label, pxSize, fg)
		}
	}
	return clicked
}

// Pagination отрисовывает постраничную навигацию (номера страниц 1..totalPages) и возвращает true при смене страницы.
func (c *Context) Pagination(id string, currentPage *int, totalPages int, opts ...Option) bool {
	if currentPage == nil || totalPages <= 0 {
		return false
	}
	id = c.ResolveID("page", id)
	if *currentPage < 1 {
		*currentPage = 1
	}
	if *currentPage > totalPages {
		*currentPage = totalPages
	}

	st := ApplyOptions(opts)
	hDp := 32
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	var tokens []int
	if totalPages <= 7 {
		for p := 1; p <= totalPages; p++ {
			tokens = append(tokens, p)
		}
	} else {
		tokens = append(tokens, 1)
		start := *currentPage - 1
		end := *currentPage + 1
		if start < 2 {
			start = 2
			end = 4
		}
		if end > totalPages-1 {
			end = totalPages - 1
			start = totalPages - 3
		}
		if start > 2 {
			tokens = append(tokens, -1)
		}
		for p := start; p <= end; p++ {
			tokens = append(tokens, p)
		}
		if end < totalPages-1 {
			tokens = append(tokens, -1)
		}
		tokens = append(tokens, totalPages)
	}

	btnSize := r.H
	gap := c.S(4)
	pxSize := c.S(12)
	curX := r.X
	changed := false

	btnIdx := 0
	drawPageBtn := func(label string, enabled bool, active bool, onClick func()) {
		br := render.Rect{X: curX, Y: r.Y, W: btnSize, H: btnSize}
		curX += btnSize + gap
		if curX-gap > r.X+r.W {
			return
		}
		btnIdx++
		if enabled {
			c.RegisterHit(id+":pg:"+strconv.Itoa(btnIdx)+":"+label, br, platform.CursorHand)
		}
		hov := enabled && !c.IsInputBlocked() && br.Contains(c.MouseX, c.MouseY)
		bg := c.Theme.Surface
		border := c.Theme.Border
		fg := c.Theme.TextPrimary
		if active {
			bg = c.Theme.Accent
			border = c.Theme.Accent
			fg = render.RGBA(255, 255, 255, 255)
		} else if !enabled {
			fg = c.Theme.TextMuted
		} else if hov {
			c.nextCursor = platform.CursorHand
			bg = c.Theme.Card
		}
		c.Canvas.DrawRoundedBox(br, c.Sf(c.Theme.RadiusSm), bg, 1.0, border)
		if c.Font != nil {
			tw, th := c.Font.MeasureText(label, pxSize)
			c.Font.DrawText(c.Canvas, br.X+(br.W-tw)/2, br.Y+(br.H-th)/2, label, pxSize, fg)
		}
		if hov && c.MousePressed && onClick != nil {
			onClick()
			changed = true
			c.RequestRedraw()
		}
	}

	drawPageBtn("<", *currentPage > 1, false, func() {
		*currentPage--
	})

	for _, tok := range tokens {
		if tok == -1 {
			er := render.Rect{X: curX, Y: r.Y, W: btnSize, H: btnSize}
			curX += btnSize + gap
			if c.Font != nil {
				tw, th := c.Font.MeasureText("...", pxSize)
				c.Font.DrawText(c.Canvas, er.X+(er.W-tw)/2, er.Y+(er.H-th)/2, "...", pxSize, c.Theme.TextMuted)
			}
			continue
		}
		pageNum := tok
		drawPageBtn(strconv.Itoa(pageNum), true, pageNum == *currentPage, func() {
			*currentPage = pageNum
		})
	}

	drawPageBtn(">", *currentPage < totalPages, false, func() {
		*currentPage++
	})

	return changed
}
