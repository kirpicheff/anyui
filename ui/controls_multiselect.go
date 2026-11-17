package ui

import (
	"fmt"
	"strconv"
	"strings"

	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/render"
)

var (
	multiSelectSearch = make(map[string]string)
	tagInputDraft     = make(map[string]string)
)

// MultiSelect отрисовывает выпадающий список с множественным выбором (чекбоксы у каждого пункта,
// фильтрация, «Выбрать все / Сбросить» и отображение выбранных значений).
func (c *Context) MultiSelect(id string, items []string, selected *[]bool, opts ...Option) bool {
	if selected == nil {
		return false
	}
	id = c.ResolveID("msel", id)
	if len(*selected) != len(items) {
		resized := make([]bool, len(items))
		copy(resized, *selected)
		*selected = resized
	}

	st := ApplyOptions(opts)
	w := c.S(240)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(34)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	rect := c.allocRect(w, h)

	if c.openState == nil {
		c.openState = make(map[string]bool)
	}

	isOpen := c.openState[id]
	if !st.Disabled {
		c.RegisterHit(id, rect, platform.CursorHand)
	}
	hov := !c.IsInputBlocked() && !st.Disabled && rect.Contains(c.MouseX, c.MouseY)
	if hov {
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			isOpen = !isOpen
			c.openState[id] = isOpen
			c.RequestRedraw()
		}
	}

	borderCol := c.Theme.Border
	if isOpen {
		borderCol = c.Theme.BorderFocus
	}
	c.Canvas.DrawRoundedBox(rect, c.Sf(c.Theme.RadiusSm), c.Theme.SurfaceDark, 1.0, borderCol)

	// Подсчёт выбранных пунктов и отображение чипсов/сводки
	var selLabels []string
	for i, it := range items {
		if (*selected)[i] {
			selLabels = append(selLabels, it)
		}
	}

	loc := c.str()
	if c.Font != nil {
		c.Canvas.PushClip(render.Rect{X: rect.X + c.S(6), Y: rect.Y, W: rect.W - c.S(28), H: rect.H})
		if len(selLabels) == 0 {
			_, th := c.Font.MeasureText(loc.SelectPlaceholder, c.S(13))
			c.Font.DrawText(c.Canvas, rect.X+c.S(10), rect.Y+(rect.H-th)/2, loc.SelectPlaceholder, c.S(13), c.Theme.TextMuted)
		} else {
			// Пытаемся отрисовать чипсы; если их много, показываем сводку
			curX := rect.X + c.S(6)
			maxX := rect.X + rect.W - c.S(26)
			pxChip := c.S(11)
			fitsAll := true
			for _, lbl := range selLabels {
				tw, _ := c.Font.MeasureText(lbl, pxChip)
				cw := tw + c.S(12)
				if curX+cw > maxX {
					fitsAll = false
					break
				}
				curX += cw + c.S(4)
			}
			if fitsAll && len(selLabels) <= 4 {
				curX = rect.X + c.S(6)
				for _, lbl := range selLabels {
					tw, th := c.Font.MeasureText(lbl, pxChip)
					cw := tw + c.S(12)
					ch := c.S(22)
					cr := render.Rect{X: curX, Y: rect.Y + (rect.H-ch)/2, W: cw, H: ch}
					c.Canvas.FillRoundedRect(cr, c.Sf(4), c.Theme.Accent.WithAlpha(45))
					c.Font.DrawText(c.Canvas, cr.X+c.S(6), cr.Y+(cr.H-th)/2, lbl, pxChip, c.Theme.Accent)
					curX += cw + c.S(4)
				}
			} else {
				summary := fmt.Sprintf(loc.SelectedSummaryFmt, len(selLabels), strings.Join(selLabels, ", "))
				_, th := c.Font.MeasureText(summary, c.S(12))
				c.Font.DrawText(c.Canvas, rect.X+c.S(10), rect.Y+(rect.H-th)/2, summary, c.S(12), c.Theme.TextPrimary)
			}
		}
		c.Canvas.PopClip()
	}

	// Стрелка справа
	ax := float32(rect.X + rect.W - c.S(14))
	ay := float32(rect.Y + rect.H/2)
	c.Canvas.DrawLine(ax-c.Sf(4), ay-c.Sf(2), ax, ay+c.Sf(2), 1.5, c.Theme.TextMuted)
	c.Canvas.DrawLine(ax, ay+c.Sf(2), ax+c.Sf(4), ay-c.Sf(2), 1.5, c.Theme.TextMuted)

	changed := false
	if !isOpen {
		return false
	}

	// Формируем фильтрованный список
	q := strings.ToLower(strings.TrimSpace(multiSelectSearch[id]))
	var filtered []int
	for i, it := range items {
		if q == "" || strings.Contains(strings.ToLower(it), q) {
			filtered = append(filtered, i)
		}
	}

	itemH := c.S(28)
	maxListH := c.S(180)
	listH := len(filtered) * itemH
	if listH > maxListH {
		listH = maxListH
	}
	if listH < itemH {
		listH = itemH
	}
	topCtrlH := c.S(62) // Поиск + кнопки Выбрать все / Сбросить
	popH := topCtrlH + listH + c.S(8)
	popW := rect.W
	if popW < c.S(220) {
		popW = c.S(220)
	}
	popX := rect.X
	popY := rect.Y + rect.H + c.S(4)
	if popY+popH > c.Canvas.Height-c.S(4) {
		popY = rect.Y - popH - c.S(4)
	}
	popR := render.Rect{X: popX, Y: popY, W: popW, H: popH}

	if c.MousePressed && !popR.Contains(c.MouseX, c.MouseY) && !rect.Contains(c.MouseX, c.MouseY) {
		c.openState[id] = false
		c.RequestRedraw()
		return false
	}

	c.PushOverlay(popR, func() {
		c.drawDropShadow(popR, c.Theme.RadiusSm, 12, 120)
		c.Canvas.DrawRoundedBox(popR, c.Sf(c.Theme.RadiusSm), c.Theme.Card, 1.0, c.Theme.Border)

		// Поле поиска
		sr := render.Rect{X: popR.X + c.S(8), Y: popR.Y + c.S(8), W: popR.W - c.S(16), H: c.S(26)}
		sVal := multiSelectSearch[id]
		c.drawInlineSearchBox(id+":search", sr, &sVal, loc.SearchPlaceholder)
		multiSelectSearch[id] = sVal

		// Кнопки "Выбрать все" и "Сбросить"
		btnY := sr.Y + sr.H + c.S(6)
		btnW := (popR.W - c.S(22)) / 2
		allR := render.Rect{X: popR.X + c.S(8), Y: btnY, W: btnW, H: c.S(22)}
		clrR := render.Rect{X: allR.X + btnW + c.S(6), Y: btnY, W: btnW, H: c.S(22)}

		drawMiniBtn := func(br render.Rect, key, lbl string, fn func()) {
			c.RegisterHit(id+":mb:"+key, br, platform.CursorHand)
			bhov := br.Contains(c.MouseX, c.MouseY)
			bg := c.Theme.Surface
			if bhov {
				c.nextCursor = platform.CursorHand
				bg = c.Theme.Border
				if c.MousePressed {
					fn()
					changed = true
					c.RequestRedraw()
				}
			}
			c.Canvas.FillRoundedRect(br, c.Sf(4), bg)
			if c.Font != nil {
				tw, th := c.Font.MeasureText(lbl, c.S(11))
				c.Font.DrawText(c.Canvas, br.X+(br.W-tw)/2, br.Y+(br.H-th)/2, lbl, c.S(11), c.Theme.TextMuted)
			}
		}
		drawMiniBtn(allR, "all", loc.SelectAllBtn, func() {
			for _, idx := range filtered {
				(*selected)[idx] = true
			}
		})
		drawMiniBtn(clrR, "reset", loc.ResetBtn, func() {
			for i := range *selected {
				(*selected)[i] = false
			}
		})

		// Список элементов с чекбоксами
		listRect := render.Rect{X: popR.X + c.S(4), Y: btnY + c.S(26), W: popR.W - c.S(8), H: listH}
		si := c.getScrollInfo(id + ":list")
		totalH := len(filtered) * itemH
		maxScr := totalH - listRect.H
		if maxScr < 0 {
			maxScr = 0
		}
		if listRect.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
			si.y -= int(c.WheelY * float32(itemH*2))
			if si.y < 0 {
				si.y = 0
			}
			if si.y > maxScr {
				si.y = maxScr
			}
			c.RequestRedraw()
		}

		c.Canvas.PushClip(listRect)
		startIdx, endIdx := ComputeVisibleRange(si.y, listRect.H, itemH, len(filtered))
		for fi := startIdx; fi < endIdx; fi++ {
			origIdx := filtered[fi]
			iy := listRect.Y + fi*itemH - si.y
			ir := render.Rect{X: listRect.X + c.S(2), Y: iy, W: listRect.W - c.S(4), H: itemH}
			c.RegisterHit(id+":opt:"+strconv.Itoa(origIdx), ir, platform.CursorHand)

			ihov := ir.Contains(c.MouseX, c.MouseY)
			if ihov {
				c.nextCursor = platform.CursorHand
				c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Surface)
				if c.MousePressed {
					(*selected)[origIdx] = !(*selected)[origIdx]
					changed = true
					c.RequestRedraw()
				}
			}

			// Чекбокс
			cbSize := c.S(15)
			cbR := render.Rect{X: ir.X + c.S(6), Y: ir.Y + (ir.H-cbSize)/2, W: cbSize, H: cbSize}
			checked := (*selected)[origIdx]
			if checked {
				c.Canvas.FillRoundedRect(cbR, c.Sf(3), c.Theme.Accent)
				cx := float32(cbR.X + cbR.W/2)
				cy := float32(cbR.Y + cbR.H/2)
				c.Canvas.DrawLine(cx-c.Sf(3.5), cy, cx-c.Sf(0.5), cy+c.Sf(3), 1.8, render.RGBA(255, 255, 255, 255))
				c.Canvas.DrawLine(cx-c.Sf(0.5), cy+c.Sf(3), cx+c.Sf(4), cy-c.Sf(3), 1.8, render.RGBA(255, 255, 255, 255))
			} else {
				c.Canvas.DrawRoundedBox(cbR, c.Sf(3), c.Theme.Bg, 1.0, c.Theme.Border)
			}

			if c.Font != nil {
				_, th := c.Font.MeasureText(items[origIdx], c.S(12))
				c.Font.DrawText(c.Canvas, cbR.X+cbR.W+c.S(8), ir.Y+(ir.H-th)/2, items[origIdx], c.S(12), c.Theme.TextPrimary)
			}
		}
		c.Canvas.PopClip()
	})

	return changed
}

// CheckComboBox — синоним MultiSelect для совместимости с десктопной терминологией.
func (c *Context) CheckComboBox(id string, items []string, selected *[]bool, opts ...Option) bool {
	return c.MultiSelect(id, items, selected, opts...)
}

// TagInput отрисовывает поле ввода тегов/чипсов (добавление по Enter/запятой, удаление по клику на x или Backspace).
func (c *Context) TagInput(id string, tags *[]string, placeholder string, opts ...Option) bool {
	if tags == nil {
		return false
	}
	id = c.ResolveID("tag", id)
	st := ApplyOptions(opts)
	hDp := 36
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	c.registerFocusable(id)
	focused := c.FocusID == id

	borderCol := c.Theme.Border
	if focused {
		borderCol = c.Theme.BorderFocus
	}
	c.Canvas.DrawRoundedBox(r, c.Sf(c.Theme.RadiusSm), c.Theme.SurfaceDark, 1.0, borderCol)

	if !st.Disabled {
		c.RegisterHit(id, r, platform.CursorIBeam)
	}
	if !c.IsInputBlocked() && !st.Disabled && r.Contains(c.MouseX, c.MouseY) {
		c.nextCursor = platform.CursorIBeam
		if c.MousePressed {
			c.FocusID = id
			c.RequestRedraw()
		}
	}

	changed := false
	draft := tagInputDraft[id]

	if focused && !c.IsInputBlocked() && !st.Disabled {
		runes := []rune(draft)
		for _, kp := range c.KeyEvents {
			if kp.Code == KeyEnter {
				trimmed := strings.TrimSpace(string(runes))
				if trimmed != "" {
					*tags = append(*tags, trimmed)
					draft = ""
					runes = nil
					changed = true
					c.RequestRedraw()
				}
			} else if kp.Code == KeyBackspace {
				if len(runes) > 0 {
					runes = runes[:len(runes)-1]
					draft = string(runes)
					c.RequestRedraw()
				} else if len(*tags) > 0 {
					*tags = (*tags)[:len(*tags)-1]
					changed = true
					c.RequestRedraw()
				}
			}
		}
		for _, ch := range c.TypedRunes {
			if ch == ',' {
				trimmed := strings.TrimSpace(string(runes))
				if trimmed != "" {
					*tags = append(*tags, trimmed)
					draft = ""
					runes = nil
					changed = true
					c.RequestRedraw()
				}
			} else if ch >= 32 {
				runes = append(runes, ch)
				draft = string(runes)
				c.RequestRedraw()
			}
		}
		tagInputDraft[id] = draft
	}

	// Отрисовка чипсов и поля ввода
	c.Canvas.PushClip(render.Rect{X: r.X + c.S(4), Y: r.Y, W: r.W - c.S(8), H: r.H})
	curX := r.X + c.S(6)
	chipH := r.H - c.S(10)
	chipY := r.Y + c.S(5)
	pxChip := c.S(11)

	removeIdx := -1
	for i, tag := range *tags {
		tw := 0
		if c.Font != nil {
			tw, _ = c.Font.MeasureText(tag, pxChip)
		}
		closeW := c.S(16)
		cw := tw + c.S(10) + closeW
		cr := render.Rect{X: curX, Y: chipY, W: cw, H: chipH}
		curX += cw + c.S(4)

		c.Canvas.FillRoundedRect(cr, c.Sf(4), c.Theme.Accent.WithAlpha(45))
		if c.Font != nil {
			_, th := c.Font.MeasureText(tag, pxChip)
			c.Font.DrawText(c.Canvas, cr.X+c.S(6), cr.Y+(cr.H-th)/2, tag, pxChip, c.Theme.Accent)
		}

		// Кнопка удаления "x"
		xr := render.Rect{X: cr.X + cr.W - closeW, Y: cr.Y, W: closeW, H: cr.H}
		if !st.Disabled {
			c.RegisterHit(id+":tagx:"+strconv.Itoa(i), xr, platform.CursorHand)
		}
		if !c.IsInputBlocked() && !st.Disabled && xr.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorHand
			if c.MousePressed {
				removeIdx = i
			}
		}
		cx := float32(xr.X + xr.W/2)
		cy := float32(xr.Y + xr.H/2)
		d := c.Sf(3)
		c.Canvas.DrawLine(cx-d, cy-d, cx+d, cy+d, 1.3, c.Theme.Accent)
		c.Canvas.DrawLine(cx-d, cy+d, cx+d, cy-d, 1.3, c.Theme.Accent)
	}

	if removeIdx >= 0 {
		*tags = append((*tags)[:removeIdx], (*tags)[removeIdx+1:]...)
		changed = true
		c.RequestRedraw()
	}

	if c.Font != nil {
		px := c.S(12)
		txt := draft
		col := c.Theme.TextPrimary
		if txt == "" && len(*tags) == 0 {
			txt = placeholder
			col = c.Theme.TextMuted
		}
		if txt != "" {
			_, th := c.Font.MeasureText(txt, px)
			c.Font.DrawText(c.Canvas, curX+c.S(2), r.Y+(r.H-th)/2, txt, px, col)
		}
		if focused {
			tw := 0
			if draft != "" {
				tw, _ = c.Font.MeasureText(draft, px)
			}
			c.Canvas.FillRect(render.Rect{X: curX + c.S(2) + tw + 1, Y: r.Y + c.S(8), W: 1, H: r.H - c.S(16)}, c.Theme.Accent)
		}
	}
	c.Canvas.PopClip()

	return changed
}
