package ui

import (
	"strconv"
	"strings"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

// TabItem описывает вкладку в динамическом TabBar.
type TabItem struct {
	ID          string
	Title       string
	IconSVG     string
	StatusColor *render.Color
	Closable    bool
}

// TabItems создаёт срез TabItem из списка строк, где ID и Title совпадают.
// Метки должны быть уникальны, поскольку они же служат ключами вкладок.
func TabItems(labels ...string) []TabItem {
	items := make([]TabItem, len(labels))
	for i, l := range labels {
		items[i] = TabItem{ID: l, Title: l}
	}
	return items
}

// TabBarEvent сообщает о действиях пользователя в TabBar за текущий кадр.
type TabBarEvent struct {
	SelectedID string
	ClosedID   string
	Added      bool
	MovedFrom  int
	MovedTo    int
	DetachedID string
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

// SelectItem описывает элемент выпадающего списка с устойчивым строковым ключом, подписью и опциональной иконкой.
type SelectItem struct {
	Key   string
	Label string
	Icon  string
}

// SelectItems создает срез SelectItem из списка строк, где Key и Label совпадают.
func SelectItems(labels ...string) []SelectItem {
	items := make([]SelectItem, len(labels))
	for i, l := range labels {
		items[i] = SelectItem{Key: l, Label: l}
	}
	return items
}

// Select рисует выпадающий список (при опции Searchable(true) — со встроенным поиском/фильтрацией)
// и возвращает true при изменении выбранного ключа.
func (c *Context) Select(id string, items []SelectItem, selected *string, opts ...Option) bool {
	if selected == nil {
		return false
	}
	id = c.ResolveID("sel", id)
	if c.openState == nil {
		c.openState = make(map[string]bool)
	}
	if c.scrollState == nil {
		c.scrollState = make(map[string]*scrollInfo)
	}

	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}
	h := c.Scale(38).Int()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	}
	r := c.allocRect(w, h)
	if !st.Disabled {
		c.registerFocusable(id)
		c.RegisterHit(id, r, platform.CursorHand)
	}

	isOpen := c.openState[id]
	openedThisFrame := false
	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered {
		c.HotID = id
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			isOpen = !isOpen
			c.openState[id] = isOpen
			if isOpen {
				openedThisFrame = true
			}
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
	c.Canvas.DrawRoundedBox(r, c.ScaleF(rad), bg, 1.0, borderCol)

	var selectedItem *SelectItem
	selectedIdx := -1
	if *selected != "" {
		for i := range items {
			if items[i].Key == *selected {
				selectedItem = &items[i]
				selectedIdx = i
				break
			}
		}
	}

	label := "Выберите..."
	textCol := c.Theme.TextMuted
	var iconSVG string
	if selectedItem != nil {
		label = selectedItem.Label
		textCol = c.Theme.TextPrimary
		iconSVG = selectedItem.Icon
	}
	if st.Disabled {
		textCol = c.Theme.TextMuted
	}

	contentX := r.X + c.Scale(12).Int()
	if iconSVG != "" {
		iconSz := c.Scale(16).Int()
		c.Canvas.DrawSVG(iconSVG, contentX, r.Y+(r.H-iconSz)/2, iconSz, textCol)
		contentX += iconSz + c.Scale(6).Int()
	}

	if c.Font != nil {
		pxSize := c.Scale(14).Int()
		_, th := c.Font.MeasureTextStyled(label, pxSize, st.FontStyle)
		c.Canvas.PushClip(render.Rect{X: r.X + c.Scale(10).Int(), Y: r.Y, W: r.W - c.Scale(32).Int(), H: r.H})
		c.Font.DrawTextStyled(c.Canvas, contentX, r.Y+(r.H-th)/2, label, pxSize, textCol, st.FontStyle)
		c.Canvas.PopClip()
	}

	// Стрелка раскрытия справа
	ax := float32(r.X + r.W - c.Scale(18).Int())
	ay := float32(r.Y + r.H/2)
	aw := c.ScaleF(5)
	c.Canvas.DrawLine(ax-aw, ay-aw*0.5, ax, ay+aw*0.5, c.ScaleF(1.6), c.Theme.TextMuted)
	c.Canvas.DrawLine(ax, ay+aw*0.5, ax+aw, ay-aw*0.5, c.ScaleF(1.6), c.Theme.TextMuted)

	changed := false
	if c.FocusID == id && !st.Disabled && len(items) > 0 && !isOpen {
		for _, kp := range c.KeyEvents {
			switch kp.Code {
			case KeySpace, KeyEnter:
				isOpen = true
				openedThisFrame = true
				c.openState[id] = true
				c.RequestRedraw()
			case KeyDown:
				if selectedIdx < len(items)-1 {
					*selected = items[selectedIdx+1].Key
					changed = true
					c.RequestRedraw()
				}
			case KeyUp:
				if selectedIdx > 0 {
					*selected = items[selectedIdx-1].Key
					changed = true
					c.RequestRedraw()
				} else if selectedIdx < 0 && len(items) > 0 {
					*selected = items[0].Key
					changed = true
					c.RequestRedraw()
				}
			case KeyHome:
				if len(items) > 0 && *selected != items[0].Key {
					*selected = items[0].Key
					changed = true
					c.RequestRedraw()
				}
			case KeyEnd:
				lastIdx := len(items) - 1
				if len(items) > 0 && *selected != items[lastIdx].Key {
					*selected = items[lastIdx].Key
					changed = true
					c.RequestRedraw()
				}
			case KeyPageUp:
				nxt := selectedIdx - 6
				if nxt < 0 {
					nxt = 0
				}
				if nxt >= 0 && nxt < len(items) && *selected != items[nxt].Key {
					*selected = items[nxt].Key
					changed = true
					c.RequestRedraw()
				}
			case KeyPageDown:
				nxt := selectedIdx + 6
				if nxt >= len(items) {
					nxt = len(items) - 1
				}
				if nxt >= 0 && nxt < len(items) && *selected != items[nxt].Key {
					*selected = items[nxt].Key
					changed = true
					c.RequestRedraw()
				}
			}
		}
	}

	if isOpen {
		itemH := c.Scale(32).Int()
		searchH := 0
		if st.Searchable {
			searchH = c.Scale(38).Int()
		}
		maxVisible := 6
		dropH := len(items)*itemH + searchH + c.Scale(8).Int()
		if len(items) > maxVisible {
			dropH = maxVisible*itemH + searchH + c.Scale(8).Int()
		}
		if dropH < c.Scale(40).Int() {
			dropH = c.Scale(40).Int()
		}

		dropRect := render.Rect{
			X: r.X,
			Y: r.Y + r.H + c.Scale(4).Int(),
			W: r.W,
			H: dropH,
		}

		// Закрытие по Escape или клику вне кнопки и выпадающего списка
		if !openedThisFrame {
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
		}

		// Синхронный этап: расчёт геометрии, фильтрация и обработка ввода до PushOverlay
		listY := dropRect.Y + c.Scale(4).Int()
		searchKey := "sel_q:" + id
		filterText := ""
		if st.Searchable {
			c.ensureTextMaps()
			if !c.openState[searchKey+"_init"] {
				c.openState[searchKey+"_init"] = true
			}
			filterText = strings.ToLower(strings.TrimSpace(c.searchQuery(searchKey)))
			listY += searchH
		}

		buildFiltered := func(ft string) []int {
			var out []int
			for idx, it := range items {
				if ft == "" || strings.Contains(strings.ToLower(it.Label), ft) || strings.Contains(strings.ToLower(it.Key), ft) {
					out = append(out, idx)
				}
			}
			return out
		}
		filtered := buildFiltered(filterText)

		listRect := render.Rect{
			X: dropRect.X + c.Scale(4).Int(),
			Y: listY,
			W: dropRect.W - c.Scale(8).Int(),
			H: dropRect.Y + dropRect.H - listY - c.Scale(4).Int(),
		}

		si := c.getScrollInfo("sel_sc:" + id)
		totalH := len(filtered) * itemH
		clampScroll := func(th int) {
			maxSc := th - listRect.H
			if maxSc < 0 {
				maxSc = 0
			}
			if si.y < 0 {
				si.y = 0
			}
			if si.y > maxSc {
				si.y = maxSc
			}
		}

		if !openedThisFrame {
			if listRect.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
				si.y -= int(c.WheelY * float32(itemH))
			}

			// Клавиатурная навигация внутри открытого списка
			if c.FocusID == id && len(filtered) > 0 {
				curPos := -1
				for p, origIdx := range filtered {
					if items[origIdx].Key == *selected {
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
					targetKey := items[filtered[np]].Key
					if *selected != targetKey {
						*selected = targetKey
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
			clampScroll(totalH)

			// Клик мышью по видимому пункту списка
			if c.openState[id] && c.MousePressed && listRect.Contains(c.MouseX, c.MouseY) {
				for pos, origIdx := range filtered {
					iy := listRect.Y + pos*itemH - si.y
					if iy+itemH < listRect.Y || iy > listRect.Y+listRect.H {
						continue
					}
					ir := render.Rect{X: listRect.X, Y: iy, W: listRect.W, H: itemH}
					if ir.Contains(c.MouseX, c.MouseY) {
						if *selected != items[origIdx].Key {
							*selected = items[origIdx].Key
							changed = true
						}
						c.openState[id] = false
						c.RequestRedraw()
						break
					}
				}
			}
		} else {
			clampScroll(totalH)
		}

		// Отложенный этап: отрисовка поверх основного UI, если список ещё открыт
		if c.openState[id] {
			c.PushOverlay(dropRect, func() {
				c.drawDropShadow(dropRect, rad, 12, 110)
				c.Canvas.DrawRoundedBox(dropRect, c.ScaleF(rad), c.Theme.Card, 1.0, c.Theme.Border)

				drawFiltered := filtered
				if st.Searchable {
					searchY := dropRect.Y + c.Scale(4).Int()
					c.pushBox(render.Rect{X: dropRect.X + c.Scale(6).Int(), Y: searchY, W: dropRect.W - c.Scale(12).Int(), H: searchH}, dirColumn, AlignStart, 0)
					q := c.searchQuery(searchKey)
					if c.TextInputOpts(searchKey, &q, c.str().SearchPlaceholder, Height(32)) {
						c.setSearchQuery(searchKey, q)
						drawFiltered = buildFiltered(strings.ToLower(strings.TrimSpace(q)))
						clampScroll(len(drawFiltered) * itemH)
					}
					c.popBox()
				}

				c.Canvas.PushClip(listRect)
				pxSize := c.Scale(13).Int()
				for pos, origIdx := range drawFiltered {
					iy := listRect.Y + pos*itemH - si.y
					if iy+itemH < listRect.Y || iy > listRect.Y+listRect.H {
						continue
					}
					ir := render.Rect{X: listRect.X, Y: iy, W: listRect.W, H: itemH}
					c.RegisterHit(id+":opt:"+strconv.Itoa(origIdx), ir, platform.CursorHand)
					itemHover := listRect.Contains(c.MouseX, c.MouseY) && ir.Contains(c.MouseX, c.MouseY)
					isSelectedItem := *selected == items[origIdx].Key
					if itemHover {
						c.nextCursor = platform.CursorHand
						c.Canvas.FillRoundedRect(ir, c.ScaleF(4), c.Theme.Surface)
					} else if isSelectedItem {
						c.Canvas.FillRoundedRect(ir, c.ScaleF(4), c.Theme.Accent.WithAlpha(45))
					}

					rowX := ir.X + c.Scale(10).Int()
					if items[origIdx].Icon != "" {
						icSz := c.Scale(14).Int()
						c.Canvas.DrawSVG(items[origIdx].Icon, rowX, ir.Y+(ir.H-icSz)/2, icSz, c.Theme.TextMuted)
						rowX += icSz + c.Scale(6).Int()
					}
					if c.Font != nil {
						_, th := c.Font.MeasureText(items[origIdx].Label, pxSize)
						col := c.Theme.TextPrimary
						if isSelectedItem {
							col = c.Theme.Accent
						}
						c.Font.DrawText(c.Canvas, rowX, ir.Y+(ir.H-th)/2, items[origIdx].Label, pxSize, col)
					}
				}
				c.Canvas.PopClip()
			})
		}
	}

	return changed
}

// ListView рисует виртуализированный прокручиваемый список элементов.
func (c *Context) ListView(id string, count int, itemHeightDp Dp, selected *int, drawItem func(index int, areaPx render.Rect, isSelected bool), opts ...Option) bool {
	id = c.ResolveID("list", id)
	if itemHeightDp <= 0 {
		itemHeightDp = 36
	}
	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}
	h := c.RemainingHeight()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	} else if h <= 0 || h >= c.Scale(50000).Int() {
		h = c.Scale(200).Int()
	}

	r := c.allocRectStyled(w, h, st)
	if r.W <= 0 || r.H <= 0 {
		return false
	}
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
	c.Canvas.DrawRoundedBox(r, c.ScaleF(rad), bg, 1.0, borderCol)

	itemH := c.Scale(itemHeightDp).Int()
	si := c.getScrollInfo("lv:" + id)
	totalH := count * itemH
	maxSc := totalH - r.H
	if maxSc < 0 {
		maxSc = 0
	}

	if st.ScrollToIndex != nil {
		targetIdx := *st.ScrollToIndex
		if targetIdx < 0 {
			si.lastScrollIndex = -1
		} else if targetIdx < count {
			if si.lastScrollIndex != targetIdx {
				targetY := targetIdx * itemH
				if targetY > maxSc {
					targetY = maxSc
				}
				if targetY < 0 {
					targetY = 0
				}
				si.y = targetY
				si.lastScrollIndex = targetIdx
			}
		}
	} else {
		si.lastScrollIndex = -1
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
		ir := render.Rect{X: r.X + c.Scale(2).Int(), Y: iy, W: r.W - c.Scale(4).Int(), H: itemH}
		c.RegisterHit(id+":row:"+strconv.Itoa(i), ir, platform.CursorHand)
		isSel := selected != nil && *selected == i
		itemHover := hovered && ir.Contains(c.MouseX, c.MouseY)
		if itemHover {
			c.nextCursor = platform.CursorHand
			if c.MousePressed {
				if selected != nil && *selected != i {
					*selected = i
					changed = true
				}
				if st.Reorderable || st.DragSourceType != "" {
					c.ArmDrag(id, i, "", st)
				}
				c.RequestRedraw()
			}
		}
		if isSel {
			c.Canvas.FillRoundedRect(ir, c.ScaleF(4), c.Theme.Accent.WithAlpha(55))
		} else if itemHover {
			c.Canvas.FillRoundedRect(ir, c.ScaleF(4), c.Theme.Surface)
		}

		// Приглушение перемещаемого элемента во время Reorder
		fromIdx, _, reorderActive := c.ReorderSource(id)
		if reorderActive && fromIdx == i {
			c.Canvas.FillRoundedRect(ir, c.ScaleF(4), c.Theme.Bg.WithAlpha(160))
		}

		if drawItem != nil {
			drawItem(i, ir, isSel)
		}
	}
	c.Canvas.PopClip()

	// Обработка перетаскивания (Reorder и кросс-виджетный Drop) элементов ListView
	if !st.Disabled {
		contentW := r.W
		if maxSc > 0 {
			contentW -= c.Scale(8).Int()
		}
		lineX := r.X + c.Scale(2).Int()
		lineW := contentW - c.Scale(4).Int()
		if st.Reorderable {
			if c.HandleLinearReorder(id, r, lineX, lineW, itemH, count, si, maxSc, st.OnReorder, nil) {
				changed = true
			}
		} else if st.DragSourceType != "" {
			c.UpdateReorderDrag(id)
		}
		if len(st.AcceptDropTypes) > 0 {
			if c.HandleLinearDrop(id, r, lineX, lineW, itemH, count, si, maxSc, st.AcceptDropTypes, st.OnDrop) {
				changed = true
			}
		}
	}

	// Вертикальный скроллбар при переполнении
	if maxSc > 0 {
		barW := c.Scale(6).Int()
		thumbH := (r.H * r.H) / totalH
		if thumbH < c.Scale(20).Int() {
			thumbH = c.Scale(20).Int()
		}
		thumbY := r.Y + ((r.H - thumbH) * si.y / maxSc)
		c.Canvas.FillRoundedRect(render.Rect{
			X: r.X + r.W - barW - c.Scale(2).Int(),
			Y: thumbY,
			W: barW,
			H: thumbH,
		}, c.ScaleF(3), c.Theme.TextMuted.WithAlpha(120))
	}

	return changed
}

// TabBar рисует панель вкладок (горизонтальную или вертикальную через TabOrientation)
// с опциональными кнопками '×' (TabItem.Closable) и '+' (скрывается через HideAddButton),
// а также поддержкой перетаскивания (drag-reorder) и открепления вкладки за пределы панели.
func (c *Context) TabBar(id string, tabs []TabItem, activeID *string, opts ...Option) TabBarEvent {
	ev := TabBarEvent{MovedFrom: -1, MovedTo: -1}
	if activeID == nil {
		return ev
	}
	id = c.ResolveID("tbar", id)
	if c.tabScroll == nil {
		c.tabScroll = make(map[string]int)
	}

	st := ApplyOptions(opts)
	if st.TabOrientation == Vertical {
		return c.tabBarVertical(id, tabs, activeID, st)
	}

	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}
	h := c.Scale(36).Int()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	}
	barRect := c.allocRect(w, h)

	addBtnW := 0
	tabsArea := barRect
	if !st.HideAddButton {
		addBtnW = c.Scale(32).Int()
		tabsArea = render.Rect{X: barRect.X, Y: barRect.Y, W: barRect.W - addBtnW - c.Scale(4).Int(), H: barRect.H}
		if tabsArea.W < 0 {
			tabsArea.W = 0
		}
	}
	tabW := c.Scale(160).Int()
	if len(tabs) > 0 && len(tabs)*tabW > tabsArea.W {
		// Минимальная ширина вкладки до включения скролла
		if tabsArea.W/len(tabs) >= c.Scale(96).Int() {
			tabW = tabsArea.W / len(tabs)
		}
	}
	totalW := len(tabs) * (tabW + c.Scale(2).Int())
	maxSc := totalW - tabsArea.W
	if maxSc < 0 {
		maxSc = 0
	}
	if tabsArea.Contains(c.MouseX, c.MouseY) && (c.WheelY != 0 || c.WheelX != 0) {
		c.tabScroll[id] += int(c.WheelX*float32(c.Scale(40).Int())) - int(c.WheelY*float32(c.Scale(40).Int()))
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
	pxSize := c.Scale(12).Int()
	for i, tab := range tabs {
		tx := tabsArea.X + i*(tabW+c.Scale(2).Int()) - scrollX
		tr := render.Rect{X: tx, Y: tabsArea.Y, W: tabW, H: tabsArea.H}
		if tr.X+tr.W < tabsArea.X || tr.X > tabsArea.X+tabsArea.W {
			continue
		}

		isAct := *activeID == tab.ID
		visTab := tabsArea.Intersect(tr)
		c.RegisterHit(id+":tab:"+strconv.Itoa(i), visTab, platform.CursorHand)
		hovered := !c.IsInputBlocked() && tabsArea.Contains(c.MouseX, c.MouseY) && tr.Contains(c.MouseX, c.MouseY)

		closeR := render.Rect{X: tr.X + tr.W - c.Scale(22).Int(), Y: tr.Y + (tr.H-c.Scale(16).Int())/2, W: c.Scale(16).Int(), H: c.Scale(16).Int()}
		if tab.Closable {
			c.RegisterHit(id+":close:"+strconv.Itoa(i), tabsArea.Intersect(closeR), platform.CursorHand)
		}
		closeHover := tab.Closable && hovered && closeR.Contains(c.MouseX, c.MouseY)

		if hovered {
			c.nextCursor = platform.CursorHand
			if c.MiddleMousePressed && tab.Closable {
				ev.ClosedID = tab.ID
				c.RequestRedraw()
			} else if c.MousePressed {
				if closeHover {
					ev.ClosedID = tab.ID
					c.RequestRedraw()
				} else {
					if *activeID != tab.ID {
						*activeID = tab.ID
						ev.SelectedID = tab.ID
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
		c.Canvas.DrawRoundedBoxCorners(tr, render.CornerRadii{TL: c.ScaleF(6), TR: c.ScaleF(6)}, bg, 1.0, c.Theme.Border)

		curX := tr.X + c.Scale(10).Int()
		if tab.StatusColor != nil {
			c.Canvas.FillCircle(float32(curX+c.Scale(4).Int()), float32(tr.Y+tr.H/2), c.ScaleF(4), *tab.StatusColor)
			curX += c.Scale(12).Int()
		} else if tab.IconSVG != "" {
			icSz := c.Scale(14).Int()
			c.Canvas.DrawSVG(tab.IconSVG, curX, tr.Y+(tr.H-icSz)/2, icSz, c.Theme.TextPrimary)
			curX += icSz + c.Scale(6).Int()
		}

		if c.Font != nil {
			textMaxW := (tr.X + tr.W - c.Scale(8).Int()) - curX
			if tab.Closable {
				textMaxW -= c.Scale(20).Int()
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
			p := float32(c.Scale(4).Int())
			c.Canvas.DrawLine(float32(closeR.X)+p, float32(closeR.Y)+p, float32(closeR.X+closeR.W)-p, float32(closeR.Y+closeR.H)-p, c.ScaleF(1.4), xCol)
			c.Canvas.DrawLine(float32(closeR.X+closeR.W)-p, float32(closeR.Y)+p, float32(closeR.X)+p, float32(closeR.Y+closeR.H)-p, c.ScaleF(1.4), xCol)
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
			if dxStart > c.Scale(6).Int() || dyStart > c.Scale(6).Int() {
				c.tabDragging = true
			}
			if c.tabDragging {
				if dy <= c.Scale(48).Int() && len(tabs) > 1 {
					step := tabW + c.Scale(2).Int()
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
							c.RequestRedraw()
						}
					}
				} else if dy > c.Scale(48).Int() {
					// Визуальная подсказка открепления вкладки в отдельное окно
					dragTitle := tabs[c.tabDragIndex].Title
					ghostR := render.Rect{X: c.MouseX + c.Scale(8).Int(), Y: c.MouseY + c.Scale(8).Int(), W: c.Scale(150).Int(), H: c.Scale(30).Int()}
					c.PushOverlay(ghostR, func() {
						c.Canvas.DrawRoundedBox(ghostR, c.ScaleF(6), c.Theme.Card.WithAlpha(235), 1.5, c.Theme.Accent)
						if c.Font != nil {
							_, th := c.Font.MeasureText(dragTitle, pxSize)
							c.Font.DrawText(c.Canvas, ghostR.X+c.Scale(10).Int(), ghostR.Y+(ghostR.H-th)/2, dragTitle, pxSize, c.Theme.TextPrimary)
						}
					})
				}
			}
		} else if c.MouseReleased {
			if c.tabDragging && dy > c.Scale(48).Int() {
				ev.DetachedID = tabs[c.tabDragIndex].ID
				c.RequestRedraw()
			}
			c.tabDragID = ""
			c.tabDragging = false
			c.tabDragIndex = -1
		}
	}

	// Кнопка '+' добавления вкладки
	if !st.HideAddButton {
		addR := render.Rect{X: barRect.X + barRect.W - addBtnW, Y: barRect.Y + c.Scale(2).Int(), W: addBtnW, H: barRect.H - c.Scale(4).Int()}
		c.RegisterHit(id+":add", addR, platform.CursorHand)
		addHover := !c.IsInputBlocked() && addR.Contains(c.MouseX, c.MouseY)
		if addHover {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRoundedRect(addR, c.ScaleF(6), c.Theme.Surface)
			if c.MousePressed {
				ev.Added = true
				c.RequestRedraw()
			}
		}
		cx := float32(addR.X) + float32(addR.W)*0.5
		cy := float32(addR.Y) + float32(addR.H)*0.5
		arm := c.ScaleF(5)
		c.Canvas.DrawLine(cx-arm, cy, cx+arm, cy, c.ScaleF(1.6), c.Theme.TextMuted)
		c.Canvas.DrawLine(cx, cy-arm, cx, cy+arm, c.ScaleF(1.6), c.Theme.TextMuted)
	}

	return ev
}

// tabBarVertical рисует вертикальную панель вкладок (например, для боковой навигации и настроек).
func (c *Context) tabBarVertical(id string, tabs []TabItem, activeID *string, st WidgetStyle) TabBarEvent {
	ev := TabBarEvent{MovedFrom: -1, MovedTo: -1}

	tabH := c.Scale(34).Int()
	gap := c.Scale(2).Int()
	addBtnH := 0
	if !st.HideAddButton {
		addBtnH = c.Scale(32).Int()
	}

	// Ширина вертикальной панели вкладок
	w := c.Scale(180).Int()
	if st.FillWidth || (st.WidthDp != nil && *st.WidthDp == Auto) {
		w = c.RemainingWidth()
	} else if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}

	// Высота вертикальной панели вкладок
	contentH := len(tabs) * (tabH + gap)
	if contentH > 0 {
		contentH -= gap
	}
	if addBtnH > 0 {
		if contentH > 0 {
			contentH += c.Scale(4).Int()
		}
		contentH += addBtnH
	}
	if contentH < tabH {
		contentH = tabH
	}

	h := contentH
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	} else if st.FillHeight || (st.HeightDp != nil && *st.HeightDp == Auto) {
		h = c.RemainingHeight()
	} else if remH := c.RemainingHeight(); remH > 0 && h > remH {
		h = remH
	}

	barRect := c.allocRect(w, h)
	tabsArea := barRect
	if addBtnH > 0 {
		tabsArea = render.Rect{X: barRect.X, Y: barRect.Y, W: barRect.W, H: barRect.H - addBtnH - c.Scale(4).Int()}
		if tabsArea.H < 0 {
			tabsArea.H = 0
		}
	}

	totalH := len(tabs) * (tabH + gap)
	maxSc := totalH - tabsArea.H
	if maxSc < 0 {
		maxSc = 0
	}
	if tabsArea.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
		c.tabScroll[id] -= int(c.WheelY * float32(c.Scale(40).Int()))
		c.RequestRedraw()
	}
	if c.tabScroll[id] < 0 {
		c.tabScroll[id] = 0
	}
	if c.tabScroll[id] > maxSc {
		c.tabScroll[id] = maxSc
	}
	scrollY := c.tabScroll[id]

	c.Canvas.PushClip(tabsArea)
	pxSize := c.Scale(12).Int()
	for i, tab := range tabs {
		ty := tabsArea.Y + i*(tabH+gap) - scrollY
		tr := render.Rect{X: tabsArea.X, Y: ty, W: tabsArea.W, H: tabH}
		if tr.Y+tr.H < tabsArea.Y || tr.Y > tabsArea.Y+tabsArea.H {
			continue
		}

		isAct := *activeID == tab.ID
		visTab := tabsArea.Intersect(tr)
		c.RegisterHit(id+":tab:"+strconv.Itoa(i), visTab, platform.CursorHand)
		hovered := !c.IsInputBlocked() && tabsArea.Contains(c.MouseX, c.MouseY) && tr.Contains(c.MouseX, c.MouseY)

		closeR := render.Rect{X: tr.X + tr.W - c.Scale(22).Int(), Y: tr.Y + (tr.H-c.Scale(16).Int())/2, W: c.Scale(16).Int(), H: c.Scale(16).Int()}
		if tab.Closable {
			c.RegisterHit(id+":close:"+strconv.Itoa(i), tabsArea.Intersect(closeR), platform.CursorHand)
		}
		closeHover := tab.Closable && hovered && closeR.Contains(c.MouseX, c.MouseY)

		if hovered {
			c.nextCursor = platform.CursorHand
			if c.MiddleMousePressed && tab.Closable {
				ev.ClosedID = tab.ID
				c.RequestRedraw()
			} else if c.MousePressed {
				if closeHover {
					ev.ClosedID = tab.ID
					c.RequestRedraw()
				} else {
					if *activeID != tab.ID {
						*activeID = tab.ID
						ev.SelectedID = tab.ID
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
		border := c.Theme.Border
		if isDragged {
			bg = c.Theme.Accent.WithAlpha(60)
		} else if isAct {
			bg = c.Theme.Card
		} else if hovered {
			bg = c.Theme.Surface
		}
		c.Canvas.DrawRoundedBox(tr, c.ScaleF(6), bg, 1.0, border)
		if isAct {
			indW := c.Scale(3).Int()
			if indW < 2 {
				indW = 2
			}
			indPad := c.Scale(6).Int()
			indR := render.Rect{X: tr.X + c.Scale(2).Int(), Y: tr.Y + indPad, W: indW, H: tr.H - indPad*2}
			if indR.H > 0 {
				c.Canvas.FillRoundedRect(indR, c.ScaleF(2), c.Theme.Accent)
			}
		}

		curX := tr.X + c.Scale(12).Int()
		if tab.StatusColor != nil {
			c.Canvas.FillCircle(float32(curX+c.Scale(4).Int()), float32(tr.Y+tr.H/2), c.ScaleF(4), *tab.StatusColor)
			curX += c.Scale(12).Int()
		} else if tab.IconSVG != "" {
			icSz := c.Scale(14).Int()
			c.Canvas.DrawSVG(tab.IconSVG, curX, tr.Y+(tr.H-icSz)/2, icSz, c.Theme.TextPrimary)
			curX += icSz + c.Scale(6).Int()
		}

		if c.Font != nil {
			textMaxW := (tr.X + tr.W - c.Scale(8).Int()) - curX
			if tab.Closable {
				textMaxW -= c.Scale(20).Int()
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
			p := float32(c.Scale(4).Int())
			c.Canvas.DrawLine(float32(closeR.X)+p, float32(closeR.Y)+p, float32(closeR.X+closeR.W)-p, float32(closeR.Y+closeR.H)-p, c.ScaleF(1.4), xCol)
			c.Canvas.DrawLine(float32(closeR.X+closeR.W)-p, float32(closeR.Y)+p, float32(closeR.X)+p, float32(closeR.Y+closeR.H)-p, c.ScaleF(1.4), xCol)
		}
	}
	c.Canvas.PopClip()

	// Обработка перетаскивания (drag-reorder) и открепления вкладки (detach) по горизонтали
	if c.tabDragID == id && c.tabDragIndex >= 0 && c.tabDragIndex < len(tabs) {
		dx := c.MouseX - (barRect.X + barRect.W/2)
		if dx < 0 {
			dx = -dx
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
			if dxStart > c.Scale(6).Int() || dyStart > c.Scale(6).Int() {
				c.tabDragging = true
			}
			if c.tabDragging {
				detachThreshold := barRect.W/2 + c.Scale(48).Int()
				if dx <= detachThreshold && len(tabs) > 1 {
					step := tabH + gap
					if step > 0 {
						targetIdx := (c.MouseY - tabsArea.Y + scrollY) / step
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
							c.RequestRedraw()
						}
					}
				} else if dx > detachThreshold {
					dragTitle := tabs[c.tabDragIndex].Title
					ghostR := render.Rect{X: c.MouseX + c.Scale(8).Int(), Y: c.MouseY + c.Scale(8).Int(), W: c.Scale(150).Int(), H: c.Scale(30).Int()}
					c.PushOverlay(ghostR, func() {
						c.Canvas.DrawRoundedBox(ghostR, c.ScaleF(6), c.Theme.Card.WithAlpha(235), 1.5, c.Theme.Accent)
						if c.Font != nil {
							_, th := c.Font.MeasureText(dragTitle, pxSize)
							c.Font.DrawText(c.Canvas, ghostR.X+c.Scale(10).Int(), ghostR.Y+(ghostR.H-th)/2, dragTitle, pxSize, c.Theme.TextPrimary)
						}
					})
				}
			}
		} else if c.MouseReleased {
			detachThreshold := barRect.W/2 + c.Scale(48).Int()
			if c.tabDragging && dx > detachThreshold {
				ev.DetachedID = tabs[c.tabDragIndex].ID
				c.RequestRedraw()
			}
			c.tabDragID = ""
			c.tabDragging = false
			c.tabDragIndex = -1
		}
	}

	// Кнопка '+' добавления вкладки внизу вертикальной панели
	if addBtnH > 0 {
		addR := render.Rect{X: barRect.X, Y: barRect.Y + barRect.H - addBtnH, W: barRect.W, H: addBtnH}
		c.RegisterHit(id+":add", addR, platform.CursorHand)
		addHover := !c.IsInputBlocked() && addR.Contains(c.MouseX, c.MouseY)
		if addHover {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRoundedRect(addR, c.ScaleF(6), c.Theme.Surface)
			if c.MousePressed {
				ev.Added = true
				c.RequestRedraw()
			}
		}
		cx := float32(addR.X) + float32(addR.W)*0.5
		cy := float32(addR.Y) + float32(addR.H)*0.5
		arm := c.ScaleF(5)
		c.Canvas.DrawLine(cx-arm, cy, cx+arm, cy, c.ScaleF(1.6), c.Theme.TextMuted)
		c.Canvas.DrawLine(cx, cy-arm, cx, cy+arm, c.ScaleF(1.6), c.Theme.TextMuted)
	}

	return ev
}
