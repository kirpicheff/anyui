package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

var (
	multiSelectSearch = make(map[string]string)
	tagInputDraft     = make(map[string]string)
)

// MultiSelect отрисовывает выпадающий список с множественным выбором (чекбоксы у каждого пункта,
// фильтрация, «Выбрать все / Сбросить» и отображение выбранных значений по ключам).
func (c *Context) MultiSelect(id string, items []SelectItem, selectedIDs map[string]bool, opts ...Option) bool {
	if selectedIDs == nil {
		return false
	}
	id = c.ResolveID("msel", id)

	st := ApplyOptions(opts)
	w := c.Scale(240).Int()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}
	h := c.Scale(34).Int()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	}
	rect := c.allocRect(w, h)

	if c.openState == nil {
		c.openState = make(map[string]bool)
	}

	isOpen := c.openState[id]
	openedThisFrame := false
	if !st.Disabled {
		c.RegisterHit(id, rect, platform.CursorHand)
	}
	hov := !c.IsInputBlocked() && !st.Disabled && rect.Contains(c.MouseX, c.MouseY)
	if hov {
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			isOpen = !isOpen
			c.openState[id] = isOpen
			if isOpen {
				openedThisFrame = true
			}
			c.RequestRedraw()
		}
	}

	borderCol := c.Theme.Border
	if isOpen {
		borderCol = c.Theme.BorderFocus
	}
	bg := c.Theme.SurfaceDark
	if st.Bg != nil {
		bg = *st.Bg
	}
	rad := c.ScaleF(c.Theme.RadiusSm)
	if st.Radius != nil {
		rad = c.ScaleF(*st.Radius)
	}
	c.Canvas.DrawRoundedBox(rect, rad, bg, 1.0, borderCol)

	// Подсчёт выбранных пунктов и отображение чипсов/сводки
	var selItems []SelectItem
	for _, it := range items {
		if selectedIDs[it.Key] {
			selItems = append(selItems, it)
		}
	}

	loc := c.str()
	if c.Font != nil {
		c.Canvas.PushClip(render.Rect{X: rect.X + c.Scale(6).Int(), Y: rect.Y, W: rect.W - c.Scale(28).Int(), H: rect.H})
		if len(selItems) == 0 {
			_, th := c.Font.MeasureTextStyled(loc.SelectPlaceholder, c.Scale(13).Int(), st.FontStyle)
			c.Font.DrawTextStyled(c.Canvas, rect.X+c.Scale(10).Int(), rect.Y+(rect.H-th)/2, loc.SelectPlaceholder, c.Scale(13).Int(), c.Theme.TextMuted, st.FontStyle)
		} else {
			// Пытаемся отрисовать чипсы; если их много, показываем сводку
			curX := rect.X + c.Scale(6).Int()
			maxX := rect.X + rect.W - c.Scale(26).Int()
			pxChip := c.Scale(11).Int()
			fitsAll := true
			for _, it := range selItems {
				tw, _ := c.Font.MeasureTextStyled(it.Label, pxChip, st.FontStyle)
				cw := tw + c.Scale(12).Int()
				if curX+cw > maxX {
					fitsAll = false
					break
				}
				curX += cw + c.Scale(4).Int()
			}
			if fitsAll && len(selItems) <= 4 {
				curX = rect.X + c.Scale(6).Int()
				for _, it := range selItems {
					tw, th := c.Font.MeasureTextStyled(it.Label, pxChip, st.FontStyle)
					cw := tw + c.Scale(12).Int()
					ch := c.Scale(22).Int()
					cr := render.Rect{X: curX, Y: rect.Y + (rect.H-ch)/2, W: cw, H: ch}
					c.Canvas.FillRoundedRect(cr, c.ScaleF(4), c.Theme.Accent.WithAlpha(45))
					c.Font.DrawTextStyled(c.Canvas, cr.X+c.Scale(6).Int(), cr.Y+(cr.H-th)/2, it.Label, pxChip, c.Theme.Accent, st.FontStyle)
					curX += cw + c.Scale(4).Int()
				}
			} else {
				var names []string
				for _, it := range selItems {
					names = append(names, it.Label)
				}
				summary := fmt.Sprintf(loc.SelectedSummaryFmt, len(names), strings.Join(names, ", "))
				_, th := c.Font.MeasureTextStyled(summary, c.Scale(12).Int(), st.FontStyle)
				txtCol := c.Theme.TextPrimary
				if st.Disabled {
					txtCol = c.Theme.TextMuted
				}
				c.Font.DrawTextStyled(c.Canvas, rect.X+c.Scale(10).Int(), rect.Y+(rect.H-th)/2, summary, c.Scale(12).Int(), txtCol, st.FontStyle)
			}
		}
		c.Canvas.PopClip()
	}

	// Стрелка справа
	ax := float32(rect.X + rect.W - c.Scale(14).Int())
	ay := float32(rect.Y + rect.H/2)
	c.Canvas.DrawLine(ax-c.ScaleF(4), ay-c.ScaleF(2), ax, ay+c.ScaleF(2), 1.5, c.Theme.TextMuted)
	c.Canvas.DrawLine(ax, ay+c.ScaleF(2), ax+c.ScaleF(4), ay-c.ScaleF(2), 1.5, c.Theme.TextMuted)

	changed := false
	if !isOpen {
		return false
	}

	// Формируем фильтрованный список
	q := strings.ToLower(strings.TrimSpace(multiSelectSearch[id]))
	var filtered []int
	for i, it := range items {
		if q == "" || strings.Contains(strings.ToLower(it.Label), q) || strings.Contains(strings.ToLower(it.Key), q) {
			filtered = append(filtered, i)
		}
	}

	itemH := c.Scale(28).Int()
	maxListH := c.Scale(180).Int()
	listH := len(filtered) * itemH
	if listH > maxListH {
		listH = maxListH
	}
	if listH < itemH {
		listH = itemH
	}
	topCtrlH := c.Scale(62).Int( // Поиск + кнопки Выбрать все / Сбросить
	)
	popH := topCtrlH + listH + c.Scale(8).Int()
	popW := rect.W
	if popW < c.Scale(220).Int() {
		popW = c.Scale(220).Int()
	}
	popX := rect.X
	popY := rect.Y + rect.H + c.Scale(4).Int()
	if popY+popH > c.Canvas.Height-c.Scale(4).Int() {
		popY = rect.Y - popH - c.Scale(4).Int()
	}
	popR := render.Rect{X: popX, Y: popY, W: popW, H: popH}

	if !openedThisFrame && c.MousePressed && !popR.Contains(c.MouseX, c.MouseY) && !rect.Contains(c.MouseX, c.MouseY) {
		c.openState[id] = false
		c.RequestRedraw()
		return false
	}

	// Синхронный этап: геометрия элементов попапа и обработка ввода до PushOverlay
	sr := render.Rect{X: popR.X + c.Scale(8).Int(), Y: popR.Y + c.Scale(8).Int(), W: popR.W - c.Scale(16).Int(), H: c.Scale(26).Int()}
	btnY := sr.Y + sr.H + c.Scale(6).Int()
	btnW := (popR.W - c.Scale(22).Int()) / 2
	allR := render.Rect{X: popR.X + c.Scale(8).Int(), Y: btnY, W: btnW, H: c.Scale(22).Int()}
	clrR := render.Rect{X: allR.X + btnW + c.Scale(6).Int(), Y: btnY, W: btnW, H: c.Scale(22).Int()}

	listRect := render.Rect{X: popR.X + c.Scale(4).Int(), Y: btnY + c.Scale(26).Int(), W: popR.W - c.Scale(8).Int(), H: listH}
	si := c.getScrollInfo(id + ":list")
	totalH := len(filtered) * itemH
	maxScr := totalH - listRect.H
	if maxScr < 0 {
		maxScr = 0
	}

	if !openedThisFrame {
		// Клики по кнопкам "Выбрать все" и "Сбросить"
		if c.MousePressed && allR.Contains(c.MouseX, c.MouseY) {
			for _, idx := range filtered {
				selectedIDs[items[idx].Key] = true
			}
			changed = true
			c.RequestRedraw()
		} else if c.MousePressed && clrR.Contains(c.MouseX, c.MouseY) {
			for k := range selectedIDs {
				delete(selectedIDs, k)
			}
			changed = true
			c.RequestRedraw()
		}

		// Прокрутка колесом мыши внутри списка
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

		// Клик по строке списка (переключение чекбокса)
		if c.MousePressed && listRect.Contains(c.MouseX, c.MouseY) {
			startIdx, endIdx := ComputeVisibleRange(si.y, listRect.H, itemH, len(filtered))
			for fi := startIdx; fi < endIdx; fi++ {
				origIdx := filtered[fi]
				itemKey := items[origIdx].Key
				iy := listRect.Y + fi*itemH - si.y
				ir := render.Rect{X: listRect.X + c.Scale(2).Int(), Y: iy, W: listRect.W - c.Scale(4).Int(), H: itemH}
				if ir.Contains(c.MouseX, c.MouseY) {
					selectedIDs[itemKey] = !selectedIDs[itemKey]
					changed = true
					c.RequestRedraw()
					break
				}
			}
		}
	}
	if si.y < 0 {
		si.y = 0
	}
	if si.y > maxScr {
		si.y = maxScr
	}

	// Отложенный этап: отрисовка попапа поверх основного UI
	c.PushOverlay(popR, func() {
		c.drawDropShadow(popR, c.Theme.RadiusSm, 12, 120)
		c.Canvas.DrawRoundedBox(popR, c.ScaleF(c.Theme.RadiusSm), c.Theme.Card, 1.0, c.Theme.Border)

		// Поле поиска
		sVal := multiSelectSearch[id]
		c.drawInlineSearchBox(id+":search", sr, &sVal, loc.SearchPlaceholder)
		multiSelectSearch[id] = sVal

		// Кнопки "Выбрать все" и "Сбросить"
		drawMiniBtn := func(br render.Rect, key, lbl string) {
			c.RegisterHit(id+":mb:"+key, br, platform.CursorHand)
			bhov := br.Contains(c.MouseX, c.MouseY)
			bg := c.Theme.Surface
			if bhov {
				c.nextCursor = platform.CursorHand
				bg = c.Theme.Border
			}
			c.Canvas.FillRoundedRect(br, c.ScaleF(4), bg)
			if c.Font != nil {
				tw, th := c.Font.MeasureText(lbl, c.Scale(11).Int())
				c.Font.DrawText(c.Canvas, br.X+(br.W-tw)/2, br.Y+(br.H-th)/2, lbl, c.Scale(11).Int(), c.Theme.TextMuted)
			}
		}
		drawMiniBtn(allR, "all", loc.SelectAllBtn)
		drawMiniBtn(clrR, "reset", loc.ResetBtn)

		// Список элементов с чекбоксами
		c.Canvas.PushClip(listRect)
		startIdx, endIdx := ComputeVisibleRange(si.y, listRect.H, itemH, len(filtered))
		for fi := startIdx; fi < endIdx; fi++ {
			origIdx := filtered[fi]
			itemKey := items[origIdx].Key
			iy := listRect.Y + fi*itemH - si.y
			ir := render.Rect{X: listRect.X + c.Scale(2).Int(), Y: iy, W: listRect.W - c.Scale(4).Int(), H: itemH}
			c.RegisterHit(id+":opt:"+strconv.Itoa(origIdx), ir, platform.CursorHand)

			ihov := listRect.Contains(c.MouseX, c.MouseY) && ir.Contains(c.MouseX, c.MouseY)
			if ihov {
				c.nextCursor = platform.CursorHand
				c.Canvas.FillRoundedRect(ir, c.ScaleF(4), c.Theme.Surface)
			}

			// Чекбокс
			cbSize := c.Scale(15).Int()
			cbR := render.Rect{X: ir.X + c.Scale(6).Int(), Y: ir.Y + (ir.H-cbSize)/2, W: cbSize, H: cbSize}
			checked := selectedIDs[itemKey]
			if checked {
				c.Canvas.FillRoundedRect(cbR, c.ScaleF(3), c.Theme.Accent)
				cx := float32(cbR.X + cbR.W/2)
				cy := float32(cbR.Y + cbR.H/2)
				c.Canvas.DrawLine(cx-c.ScaleF(3.5), cy, cx-c.ScaleF(0.5), cy+c.ScaleF(3), 1.8, render.RGBA(255, 255, 255, 255))
				c.Canvas.DrawLine(cx-c.ScaleF(0.5), cy+c.ScaleF(3), cx+c.ScaleF(4), cy-c.ScaleF(3), 1.8, render.RGBA(255, 255, 255, 255))
			} else {
				c.Canvas.DrawRoundedBox(cbR, c.ScaleF(3), c.Theme.Bg, 1.0, c.Theme.Border)
			}

			itX := cbR.X + cbR.W + c.Scale(8).Int()
			if items[origIdx].Icon != "" {
				icSz := c.Scale(14).Int()
				c.Canvas.DrawSVG(items[origIdx].Icon, itX, ir.Y+(ir.H-icSz)/2, icSz, c.Theme.TextMuted)
				itX += icSz + c.Scale(6).Int()
			}

			if c.Font != nil {
				_, th := c.Font.MeasureText(items[origIdx].Label, c.Scale(12).Int())
				c.Font.DrawText(c.Canvas, itX, ir.Y+(ir.H-th)/2, items[origIdx].Label, c.Scale(12).Int(), c.Theme.TextPrimary)
			}
		}
		c.Canvas.PopClip()
	})

	return changed
}

// TagInput отрисовывает поле ввода тегов/чипсов (добавление по Enter/запятой, удаление по клику на x или Backspace).
func (c *Context) TagInput(id string, tags *[]string, placeholder string, opts ...Option) bool {
	if tags == nil {
		return false
	}
	id = c.ResolveID("tag", id)
	st := ApplyOptions(opts)
	hDp := Dp(36)
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
	c.Canvas.DrawRoundedBox(r, c.ScaleF(c.Theme.RadiusSm), c.Theme.SurfaceDark, 1.0, borderCol)

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
	c.Canvas.PushClip(render.Rect{X: r.X + c.Scale(4).Int(), Y: r.Y, W: r.W - c.Scale(8).Int(), H: r.H})
	curX := r.X + c.Scale(6).Int()
	chipH := r.H - c.Scale(10).Int()
	chipY := r.Y + c.Scale(5).Int()
	pxChip := c.Scale(11).Int()

	removeIdx := -1
	for i, tag := range *tags {
		tw := 0
		if c.Font != nil {
			tw, _ = c.Font.MeasureText(tag, pxChip)
		}
		closeW := c.Scale(16).Int()
		cw := tw + c.Scale(10).Int() + closeW
		cr := render.Rect{X: curX, Y: chipY, W: cw, H: chipH}
		curX += cw + c.Scale(4).Int()

		c.Canvas.FillRoundedRect(cr, c.ScaleF(4), c.Theme.Accent.WithAlpha(45))
		if c.Font != nil {
			_, th := c.Font.MeasureText(tag, pxChip)
			c.Font.DrawText(c.Canvas, cr.X+c.Scale(6).Int(), cr.Y+(cr.H-th)/2, tag, pxChip, c.Theme.Accent)
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
		d := c.ScaleF(3)
		c.Canvas.DrawLine(cx-d, cy-d, cx+d, cy+d, 1.3, c.Theme.Accent)
		c.Canvas.DrawLine(cx-d, cy+d, cx+d, cy-d, 1.3, c.Theme.Accent)
	}

	if removeIdx >= 0 {
		*tags = append((*tags)[:removeIdx], (*tags)[removeIdx+1:]...)
		changed = true
		c.RequestRedraw()
	}

	if c.Font != nil {
		px := c.Scale(12).Int()
		txt := draft
		col := c.Theme.TextPrimary
		if txt == "" && len(*tags) == 0 {
			txt = placeholder
			col = c.Theme.TextMuted
		}
		if txt != "" {
			_, th := c.Font.MeasureText(txt, px)
			c.Font.DrawText(c.Canvas, curX+c.Scale(2).Int(), r.Y+(r.H-th)/2, txt, px, col)
		}
		if focused {
			tw := 0
			if draft != "" {
				tw, _ = c.Font.MeasureText(draft, px)
			}
			c.Canvas.FillRect(render.Rect{X: curX + c.Scale(2).Int() + tw + 1, Y: r.Y + c.Scale(8).Int(), W: 1, H: r.H - c.Scale(16).Int()}, c.Theme.Accent)
		}
	}
	c.Canvas.PopClip()

	return changed
}
