package ui

import (
	"strconv"
	"strings"

	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/render"
)

// TabItem описывает вкладку в динамическом TabBar.
type TabItem struct {
	ID          string
	Title       string
	IconSVG     string
	StatusColor *render.Color
	Closable    bool
}

// TabBarEvent сообщает о действиях пользователя в TabBar за текущий кадр.
type TabBarEvent struct {
	Selected  int
	Closed    int
	Added     bool
	MovedFrom int
	MovedTo   int
	Detached  int
}

// ComputeVisibleRange вычисляет диапазон [start, end) видимых элементов для виртуализации списка.
func ComputeVisibleRange(scrollY, viewH, itemH, count int) (int, int) {
	if count <= 0 || itemH <= 0 || viewH <= 0 {
		return 0, 0
	}
	if scrollY < 0 {
		scrollY = 0
	}
	start := scrollY / itemH
	if start >= count {
		return count, count
	}
	end := (scrollY+viewH+itemH-1)/itemH + 1
	if end > count {
		end = count
	}
	return start, end
}

// Select рисует выпадающий список (при Searchable(true) — ComboBox с фильтром поиска)
// и возвращает true при изменении выбранного индекса.
func (c *Context) Select(id string, items []string, selected *int, opts ...Option) bool {
	if selected == nil {
		return false
	}
	if c.openState == nil {
		c.openState = make(map[string]bool)
	}
	if c.scrollState == nil {
		c.scrollState = make(map[string]*scrollInfo)
	}

	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(38)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	r := c.allocRect(w, h)
	if !st.Disabled {
		c.registerFocusable(id)
		c.RegisterHit(id, r, platform.CursorHand)
	}

	isOpen := c.openState[id]
	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered {
		c.HotID = id
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			isOpen = !isOpen
			c.openState[id] = isOpen
			c.FocusID = id
			c.RequestRedraw()
		}
	}

	bg := c.Theme.Surface
	if st.Bg != nil {
		bg = *st.Bg
	}
	borderCol := c.Theme.Border
	if isOpen || c.FocusID == id {
		borderCol = c.Theme.BorderFocus
	}
	rad := c.Theme.RadiusSm
	if st.Radius != nil {
		rad = *st.Radius
	}
	c.Canvas.DrawRoundedBox(r, c.Sf(rad), bg, 1.0, borderCol)

	label := "Выберите..."
	textCol := c.Theme.TextMuted
	if *selected >= 0 && *selected < len(items) {
		label = items[*selected]
		textCol = c.Theme.TextPrimary
	}
	if st.Disabled {
		textCol = c.Theme.TextMuted
	}

	if c.Font != nil {
		pxSize := c.S(14)
		_, th := c.Font.MeasureTextStyled(label, pxSize, st.FontStyle)
		c.Canvas.PushClip(render.Rect{X: r.X + c.S(10), Y: r.Y, W: r.W - c.S(32), H: r.H})
		c.Font.DrawTextStyled(c.Canvas, r.X+c.S(12), r.Y+(r.H-th)/2, label, pxSize, textCol, st.FontStyle)
		c.Canvas.PopClip()
	}

	// Стрелка раскрытия справа
	ax := float32(r.X + r.W - c.S(18))
	ay := float32(r.Y + r.H/2)
	aw := c.Sf(5)
	c.Canvas.DrawLine(ax-aw, ay-aw*0.5, ax, ay+aw*0.5, c.Sf(1.6), c.Theme.TextMuted)
	c.Canvas.DrawLine(ax, ay+aw*0.5, ax+aw, ay-aw*0.5, c.Sf(1.6), c.Theme.TextMuted)

	changed := false
	if c.FocusID == id && !st.Disabled && len(items) > 0 && !isOpen {
		for _, kp := range c.KeyEvents {
			switch kp.Code {
			case KeySpace, KeyEnter:
				isOpen = true
				c.openState[id] = true
				c.RequestRedraw()
			case KeyDown:
				if *selected < len(items)-1 {
					*selected++
					changed = true
					c.RequestRedraw()
				}
			case KeyUp:
				if *selected > 0 {
					*selected--
					changed = true
					c.RequestRedraw()
				} else if *selected < 0 {
					*selected = 0
					changed = true
					c.RequestRedraw()
				}
			case KeyHome:
				if *selected != 0 {
					*selected = 0
					changed = true
					c.RequestRedraw()
				}
			case KeyEnd:
				if *selected != len(items)-1 {
					*selected = len(items) - 1
					changed = true
					c.RequestRedraw()
				}
			case KeyPageUp:
				nxt := *selected - 6
				if nxt < 0 {
					nxt = 0
				}
				if nxt != *selected {
					*selected = nxt
					changed = true
					c.RequestRedraw()
				}
			case KeyPageDown:
				nxt := *selected + 6
				if nxt >= len(items) {
					nxt = len(items) - 1
				}
				if nxt != *selected {
					*selected = nxt
					changed = true
					c.RequestRedraw()
				}
			}
		}
	}

	if isOpen {
		itemH := c.S(32)
		searchH := 0
		if st.Searchable {
			searchH = c.S(38)
		}
		maxVisible := 6
		dropH := len(items)*itemH + searchH + c.S(8)
		if len(items) > maxVisible {
			dropH = maxVisible*itemH + searchH + c.S(8)
		}
		if dropH < c.S(40) {
			dropH = c.S(40)
		}

		dropRect := render.Rect{
			X: r.X,
			Y: r.Y + r.H + c.S(4),
			W: r.W,
			H: dropH,
		}

		// Закрытие по Escape или клику вне кнопки и выпадающего списка
		for _, kp := range c.KeyEvents {
			if kp.Code == KeyEscape {
				c.openState[id] = false
				c.RequestRedraw()
				return false
			}
		}
		if c.MousePressed && !r.Contains(c.MouseX, c.MouseY) && !dropRect.Contains(c.MouseX, c.MouseY) {
			c.openState[id] = false
			c.RequestRedraw()
			return false
		}

		c.PushOverlay(dropRect, func() {
			c.drawDropShadow(dropRect, rad, 12, 110)
			c.Canvas.DrawRoundedBox(dropRect, c.Sf(rad), c.Theme.Card, 1.0, c.Theme.Border)

			listY := dropRect.Y + c.S(4)
			filterText := ""
			if st.Searchable {
				c.ensureTextMaps()
				searchKey := "sel_q:" + id
				if c.openState[searchKey+"_init"] == false {
					c.openState[searchKey+"_init"] = true
				}
				// Простой инлайн-фильтр через локальный буфер в layout
				c.pushBox(render.Rect{X: dropRect.X + c.S(6), Y: listY, W: dropRect.W - c.S(12), H: searchH}, dirColumn, AlignStart, 0)
				q := c.searchQuery(searchKey)
				if c.TextInputOpts(searchKey, &q, c.str().SearchPlaceholder, Height(32)) {
					c.setSearchQuery(searchKey, q)
				}
				c.popBox()
				filterText = strings.ToLower(strings.TrimSpace(q))
				listY += searchH
			}

			var filtered []int
			for idx, it := range items {
				if filterText == "" || strings.Contains(strings.ToLower(it), filterText) {
					filtered = append(filtered, idx)
				}
			}

			listRect := render.Rect{
				X: dropRect.X + c.S(4),
				Y: listY,
				W: dropRect.W - c.S(8),
				H: dropRect.Y + dropRect.H - listY - c.S(4),
			}

			si := c.getScrollInfo("sel_sc:" + id)
			totalH := len(filtered) * itemH
			if listRect.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
				si.y -= int(c.WheelY * float32(itemH))
			}

			// Клавиатурная навигация внутри открытого списка
			if c.FocusID == id && len(filtered) > 0 {
				curPos := -1
				for p, origIdx := range filtered {
					if origIdx == *selected {
						curPos = p
						break
					}
				}
				movePos := func(np int) {
					if np < 0 {
						np = 0
					}
					if np >= len(filtered) {
						np = len(filtered) - 1
					}
					if *selected != filtered[np] {
						*selected = filtered[np]
						changed = true
					}
					if np*itemH < si.y {
						si.y = np * itemH
					} else if (np+1)*itemH > si.y+listRect.H {
						si.y = (np+1)*itemH - listRect.H
					}
					c.RequestRedraw()
				}
				for _, kp := range c.KeyEvents {
					switch kp.Code {
					case KeyDown:
						movePos(curPos + 1)
					case KeyUp:
						if curPos <= 0 {
							movePos(0)
						} else {
							movePos(curPos - 1)
						}
					case KeyPageDown:
						movePos(curPos + maxVisible)
					case KeyPageUp:
						movePos(curPos - maxVisible)
					case KeyHome:
						movePos(0)
					case KeyEnd:
						movePos(len(filtered) - 1)
					case KeyEnter:
						c.openState[id] = false
						c.RequestRedraw()
					case KeySpace:
						if !st.Searchable {
							c.openState[id] = false
							c.RequestRedraw()
						}
					}
				}
			}
			maxSc := totalH - listRect.H
			if maxSc < 0 {
				maxSc = 0
			}
			if si.y < 0 {
				si.y = 0
			}
			if si.y > maxSc {
				si.y = maxSc
			}

			c.Canvas.PushClip(listRect)
			pxSize := c.S(13)
			for pos, origIdx := range filtered {
				iy := listRect.Y + pos*itemH - si.y
				if iy+itemH < listRect.Y || iy > listRect.Y+listRect.H {
					continue
				}
				ir := render.Rect{X: listRect.X, Y: iy, W: listRect.W, H: itemH}
				c.RegisterHit(id+":opt:"+strconv.Itoa(origIdx), ir, platform.CursorHand)
				itemHover := ir.Contains(c.MouseX, c.MouseY)
				if itemHover {
					c.nextCursor = platform.CursorHand
					c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Surface)
					if c.MousePressed {
						if *selected != origIdx {
							*selected = origIdx
							changed = true
						}
						c.openState[id] = false
						c.RequestRedraw()
					}
				} else if *selected == origIdx {
					c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Accent.WithAlpha(45))
				}
				if c.Font != nil {
					_, th := c.Font.MeasureText(items[origIdx], pxSize)
					col := c.Theme.TextPrimary
					if *selected == origIdx {
						col = c.Theme.Accent
					}
					c.Font.DrawText(c.Canvas, ir.X+c.S(10), ir.Y+(ir.H-th)/2, items[origIdx], pxSize, col)
				}
			}
			c.Canvas.PopClip()
		})
	}

	return changed
}

// ListView рисует виртуализированный прокручиваемый список элементов.
func (c *Context) ListView(id string, count int, itemHeightDp int, selected *int, drawItem func(index int, area render.Rect, isSelected bool), opts ...Option) bool {
	if itemHeightDp <= 0 {
		itemHeightDp = 36
	}
	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.RemainingHeight()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	if h <= 0 {
		h = c.S(200)
	}

	r := c.allocRect(w, h)
	if !st.Disabled {
		c.registerFocusable(id)
		c.RegisterHit(id, r, platform.CursorArrow)
	}

	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered && (c.MousePressed || c.MiddleMousePressed) {
		c.FocusID = id
		c.RequestRedraw()
	}

	bg := c.Theme.SurfaceDark
	if st.Bg != nil {
		bg = *st.Bg
	}
	rad := c.Theme.RadiusSm
	if st.Radius != nil {
		rad = *st.Radius
	}
	borderCol := c.Theme.Border
	if c.FocusID == id {
		borderCol = c.Theme.BorderFocus
	}
	c.Canvas.DrawRoundedBox(r, c.Sf(rad), bg, 1.0, borderCol)

	itemH := c.S(itemHeightDp)
	si := c.getScrollInfo("lv:" + id)
	totalH := count * itemH
	maxSc := totalH - r.H
	if maxSc < 0 {
		maxSc = 0
	}

	if hovered && c.WheelY != 0 {
		si.y -= int(c.WheelY * float32(itemH) * 2)
		c.RequestRedraw()
	}

	changed := false
	if !st.Disabled && selected != nil && count > 0 {
		nav := funcNavigable{
			count:    count,
			selected: *selected,
			pageSize: r.H / itemH,
			onSelect: func(newIdx int) {
				*selected = newIdx
				if newIdx*itemH < si.y {
					si.y = newIdx * itemH
				} else if (newIdx+1)*itemH > si.y+r.H {
					si.y = (newIdx+1)*itemH - r.H
				}
			},
		}
		if c.handleNavigable(id, nav, nil, nil) {
			changed = true
		}
	}

	if si.y < 0 {
		si.y = 0
	}
	if si.y > maxSc {
		si.y = maxSc
	}
	c.Canvas.PushClip(r)
	start, end := ComputeVisibleRange(si.y, r.H, itemH, count)
	for i := start; i < end; i++ {
		iy := r.Y + i*itemH - si.y
		ir := render.Rect{X: r.X + c.S(2), Y: iy, W: r.W - c.S(4), H: itemH}
		c.RegisterHit(id+":row:"+strconv.Itoa(i), ir, platform.CursorHand)
		isSel := selected != nil && *selected == i
		itemHover := hovered && ir.Contains(c.MouseX, c.MouseY)
		if itemHover {
			c.nextCursor = platform.CursorHand
			if c.MousePressed && selected != nil {
				if *selected != i {
					*selected = i
					changed = true
					c.RequestRedraw()
				}
			}
		}
		if isSel {
			c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Accent.WithAlpha(55))
		} else if itemHover {
			c.Canvas.FillRoundedRect(ir, c.Sf(4), c.Theme.Surface)
		}
		if drawItem != nil {
			drawItem(i, ir, isSel)
		}
	}
	c.Canvas.PopClip()

	// Вертикальный скроллбар при переполнении
	if maxSc > 0 {
		barW := c.S(6)
		thumbH := (r.H * r.H) / totalH
		if thumbH < c.S(20) {
			thumbH = c.S(20)
		}
		thumbY := r.Y + ((r.H - thumbH) * si.y / maxSc)
		c.Canvas.FillRoundedRect(render.Rect{
			X: r.X + r.W - barW - c.S(2),
			Y: thumbY,
			W: barW,
			H: thumbH,
		}, c.Sf(3), c.Theme.TextMuted.WithAlpha(120))
	}

	return changed
}

// Tabs рисует статические горизонтальные вкладки или вертикальное боковое меню.
func (c *Context) Tabs(id string, items []string, active *int, vertical bool, opts ...Option) bool {
	if active == nil || len(items) == 0 {
		return false
	}
	st := ApplyOptions(opts)
	changed := false
	pxSize := c.S(14)
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		pxSize = c.S(*st.FontSizeDp)
	}

	drawOne := func(i int, label string) {
		isAct := *active == i
		tw, th := 60, c.S(18)
		if c.Font != nil {
			tw, th = c.Font.MeasureTextStyled(label, pxSize, st.FontStyle)
		}
		w := tw + c.S(24)
		if vertical {
			w = c.RemainingWidth()
			if st.WidthDp != nil && *st.WidthDp > 0 {
				w = c.S(*st.WidthDp)
			}
		}
		h := th + c.S(14)
		r := c.allocRect(w, h)
		if !st.Disabled {
			c.RegisterHit(id+":tab:"+strconv.Itoa(i), r, platform.CursorHand)
		}
		hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
		if hovered {
			c.nextCursor = platform.CursorHand
			if c.MousePressed && *active != i {
				*active = i
				changed = true
				c.RequestRedraw()
			}
		}

		bg := c.Theme.Bg.WithAlpha(0)
		textCol := c.Theme.TextMuted
		if isAct {
			bg = c.Theme.Surface
			textCol = c.Theme.TextPrimary
		} else if hovered {
			bg = c.Theme.Surface.WithAlpha(120)
			textCol = c.Theme.TextPrimary
		}
		c.Canvas.FillRoundedRect(r, c.Sf(c.Theme.RadiusSm), bg)
		if isAct {
			if vertical {
				c.Canvas.FillRect(render.Rect{X: r.X, Y: r.Y + c.S(6), W: c.S(3), H: r.H - c.S(12)}, c.Theme.Accent)
			} else {
				c.Canvas.FillRect(render.Rect{X: r.X + c.S(8), Y: r.Y + r.H - c.S(3), W: r.W - c.S(16), H: c.S(3)}, c.Theme.Accent)
			}
		}
		if c.Font != nil {
			tx := r.X + (r.W-tw)/2
			if vertical {
				tx = r.X + c.S(14)
			}
			c.Font.DrawTextStyled(c.Canvas, tx, r.Y+(r.H-th)/2, label, pxSize, textCol, st.FontStyle)
		}
	}

	if vertical {
		c.Column(4, func() {
			for i, item := range items {
				drawOne(i, item)
			}
		})
	} else {
		c.Row(4, 36, func() {
			for i, item := range items {
				drawOne(i, item)
			}
		})
	}
	return changed
}

// TabBar рисует динамические закрываемые вкладки (как в браузере/IDE) с кнопками '×' и '+',
// а также поддержкой перетаскивания (drag-reorder) и открепления вкладки за пределы панели.
func (c *Context) TabBar(id string, tabs []TabItem, active *int, opts ...Option) TabBarEvent {
	ev := TabBarEvent{Selected: -1, Closed: -1, MovedFrom: -1, MovedTo: -1, Detached: -1}
	if active == nil {
		return ev
	}
	if c.tabScroll == nil {
		c.tabScroll = make(map[string]int)
	}

	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(36)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	barRect := c.allocRect(w, h)

	addBtnW := c.S(32)
	tabsArea := render.Rect{X: barRect.X, Y: barRect.Y, W: barRect.W - addBtnW - c.S(4), H: barRect.H}
	tabW := c.S(160)
	if len(tabs) > 0 && len(tabs)*tabW > tabsArea.W {
		// Минимальная ширина вкладки до включения скролла
		if tabsArea.W/len(tabs) >= c.S(96) {
			tabW = tabsArea.W / len(tabs)
		}
	}
	totalW := len(tabs) * (tabW + c.S(2))
	maxSc := totalW - tabsArea.W
	if maxSc < 0 {
		maxSc = 0
	}
	if tabsArea.Contains(c.MouseX, c.MouseY) && (c.WheelY != 0 || c.WheelX != 0) {
		c.tabScroll[id] += int(c.WheelX*float32(c.S(40))) - int(c.WheelY*float32(c.S(40)))
		c.RequestRedraw()
	}
	if c.tabScroll[id] < 0 {
		c.tabScroll[id] = 0
	}
	if c.tabScroll[id] > maxSc {
		c.tabScroll[id] = maxSc
	}
	scrollX := c.tabScroll[id]

	c.Canvas.PushClip(tabsArea)
	pxSize := c.S(12)
	for i, tab := range tabs {
		tx := tabsArea.X + i*(tabW+c.S(2)) - scrollX
		tr := render.Rect{X: tx, Y: tabsArea.Y, W: tabW, H: tabsArea.H}
		if tr.X+tr.W < tabsArea.X || tr.X > tabsArea.X+tabsArea.W {
			continue
		}

		isAct := *active == i
		visTab := tabsArea.Intersect(tr)
		c.RegisterHit(id+":tab:"+strconv.Itoa(i), visTab, platform.CursorHand)
		hovered := !c.IsInputBlocked() && tabsArea.Contains(c.MouseX, c.MouseY) && tr.Contains(c.MouseX, c.MouseY)

		closeR := render.Rect{X: tr.X + tr.W - c.S(22), Y: tr.Y + (tr.H-c.S(16))/2, W: c.S(16), H: c.S(16)}
		if tab.Closable {
			c.RegisterHit(id+":close:"+strconv.Itoa(i), tabsArea.Intersect(closeR), platform.CursorHand)
		}
		closeHover := tab.Closable && hovered && closeR.Contains(c.MouseX, c.MouseY)

		if hovered {
			c.nextCursor = platform.CursorHand
			if c.MiddleMousePressed && tab.Closable {
				ev.Closed = i
				c.RequestRedraw()
			} else if c.MousePressed {
				if closeHover {
					ev.Closed = i
					c.RequestRedraw()
				} else {
					if *active != i {
						*active = i
						ev.Selected = i
						c.RequestRedraw()
					}
					c.tabDragID = id
					c.tabDragIndex = i
					c.tabDragStartX = c.MouseX
					c.tabDragStartY = c.MouseY
					c.tabDragging = false
				}
			}
		}

		isDragged := c.tabDragID == id && c.tabDragging && c.tabDragIndex == i
		bg := c.Theme.SurfaceDark
		if isDragged {
			bg = c.Theme.Accent.WithAlpha(60)
		} else if isAct {
			bg = c.Theme.Card
		} else if hovered {
			bg = c.Theme.Surface
		}
		c.Canvas.DrawRoundedBoxCorners(tr, render.CornerRadii{TL: c.Sf(6), TR: c.Sf(6)}, bg, 1.0, c.Theme.Border)

		curX := tr.X + c.S(10)
		if tab.StatusColor != nil {
			c.Canvas.FillCircle(float32(curX+c.S(4)), float32(tr.Y+tr.H/2), c.Sf(4), *tab.StatusColor)
			curX += c.S(12)
		} else if tab.IconSVG != "" {
			icSz := c.S(14)
			c.Canvas.DrawSVG(tab.IconSVG, curX, tr.Y+(tr.H-icSz)/2, icSz, c.Theme.TextPrimary)
			curX += icSz + c.S(6)
		}

		if c.Font != nil {
			textMaxW := (tr.X + tr.W - c.S(8)) - curX
			if tab.Closable {
				textMaxW -= c.S(20)
			}
			if textMaxW > 0 {
				c.Canvas.PushClip(render.Rect{X: curX, Y: tr.Y, W: textMaxW, H: tr.H})
				_, th := c.Font.MeasureText(tab.Title, pxSize)
				col := c.Theme.TextMuted
				if isAct {
					col = c.Theme.TextPrimary
				}
				c.Font.DrawText(c.Canvas, curX, tr.Y+(tr.H-th)/2, tab.Title, pxSize, col)
				c.Canvas.PopClip()
			}
		}

		if tab.Closable {
			xCol := c.Theme.TextMuted
			if closeHover {
				xCol = c.Theme.Danger
			}
			p := float32(c.S(4))
			c.Canvas.DrawLine(float32(closeR.X)+p, float32(closeR.Y)+p, float32(closeR.X+closeR.W)-p, float32(closeR.Y+closeR.H)-p, c.Sf(1.4), xCol)
			c.Canvas.DrawLine(float32(closeR.X+closeR.W)-p, float32(closeR.Y)+p, float32(closeR.X)+p, float32(closeR.Y+closeR.H)-p, c.Sf(1.4), xCol)
		}
	}
	c.Canvas.PopClip()

	// Обработка перетаскивания (drag-reorder) и открепления вкладки (detach)
	if c.tabDragID == id && c.tabDragIndex >= 0 && c.tabDragIndex < len(tabs) {
		dy := c.MouseY - (barRect.Y + barRect.H/2)
		if dy < 0 {
			dy = -dy
		}
		if c.MouseDown {
			dxStart := c.MouseX - c.tabDragStartX
			dyStart := c.MouseY - c.tabDragStartY
			if dxStart < 0 {
				dxStart = -dxStart
			}
			if dyStart < 0 {
				dyStart = -dyStart
			}
			if dxStart > c.S(6) || dyStart > c.S(6) {
				c.tabDragging = true
			}
			if c.tabDragging {
				if dy <= c.S(48) && len(tabs) > 1 {
					step := tabW + c.S(2)
					if step > 0 {
						targetIdx := (c.MouseX - tabsArea.X + scrollX) / step
						if targetIdx < 0 {
							targetIdx = 0
						}
						if targetIdx >= len(tabs) {
							targetIdx = len(tabs) - 1
						}
						if targetIdx != c.tabDragIndex {
							ev.MovedFrom = c.tabDragIndex
							ev.MovedTo = targetIdx
							c.tabDragIndex = targetIdx
							*active = targetIdx
							c.RequestRedraw()
						}
					}
				} else if dy > c.S(48) {
					// Визуальная подсказка открепления вкладки в отдельное окно
					dragTitle := tabs[c.tabDragIndex].Title
					ghostR := render.Rect{X: c.MouseX + c.S(8), Y: c.MouseY + c.S(8), W: c.S(150), H: c.S(30)}
					c.PushOverlay(ghostR, func() {
						c.Canvas.DrawRoundedBox(ghostR, c.Sf(6), c.Theme.Card.WithAlpha(235), 1.5, c.Theme.Accent)
						if c.Font != nil {
							_, th := c.Font.MeasureText(dragTitle, pxSize)
							c.Font.DrawText(c.Canvas, ghostR.X+c.S(10), ghostR.Y+(ghostR.H-th)/2, dragTitle, pxSize, c.Theme.TextPrimary)
						}
					})
				}
			}
		} else if c.MouseReleased {
			if c.tabDragging && dy > c.S(48) {
				ev.Detached = c.tabDragIndex
				c.RequestRedraw()
			}
			c.tabDragID = ""
			c.tabDragging = false
			c.tabDragIndex = -1
		}
	}

	// Кнопка '+' добавления вкладки
	addR := render.Rect{X: barRect.X + barRect.W - addBtnW, Y: barRect.Y + c.S(2), W: addBtnW, H: barRect.H - c.S(4)}
	c.RegisterHit(id+":add", addR, platform.CursorHand)
	addHover := !c.IsInputBlocked() && addR.Contains(c.MouseX, c.MouseY)
	if addHover {
		c.nextCursor = platform.CursorHand
		c.Canvas.FillRoundedRect(addR, c.Sf(6), c.Theme.Surface)
		if c.MousePressed {
			ev.Added = true
			c.RequestRedraw()
		}
	}
	cx := float32(addR.X) + float32(addR.W)*0.5
	cy := float32(addR.Y) + float32(addR.H)*0.5
	arm := c.Sf(5)
	c.Canvas.DrawLine(cx-arm, cy, cx+arm, cy, c.Sf(1.6), c.Theme.TextMuted)
	c.Canvas.DrawLine(cx, cy-arm, cx, cy+arm, c.Sf(1.6), c.Theme.TextMuted)

	return ev
}
