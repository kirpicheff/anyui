package ui

import (
	"strconv"

	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/render"
)

// TableColumn описывает одну колонку таблицы.
type TableColumn struct {
	Title      string
	WidthDp    int
	MinWidthDp int
	Sortable   bool
}

// TableSort задаёт текущее состояние сортировки таблицы.
type TableSort struct {
	Column int
	Desc   bool
}

// TableEvent сообщает о действиях пользователя в таблице за текущий кадр.
type TableEvent struct {
	RowClicked       int
	RowDoubleClicked int
	RowMiddleClicked int
	SortChanged      bool
	ColumnResized    bool
}

// calcTableColWidthsDp рассчитывает стартовые ширины колонок в dp.
// Колонки с WidthDp <= 0 равномерно делят оставшуюся ширину таблицы.
func calcTableColWidthsDp(cols []TableColumn, totalWidthDp int) []int {
	widths := make([]int, len(cols))
	fixedSum := 0
	flexCount := 0
	for i, col := range cols {
		if col.WidthDp > 0 {
			widths[i] = col.WidthDp
			fixedSum += col.WidthDp
		} else {
			flexCount++
		}
	}
	if flexCount == 0 {
		return widths
	}
	rem := totalWidthDp - fixedSum
	flexW := 120
	if totalWidthDp > 0 {
		flexW = rem / flexCount
	}
	for i, col := range cols {
		if col.WidthDp > 0 {
			continue
		}
		minW := col.MinWidthDp
		if minW <= 0 {
			minW = 60
		}
		w := flexW
		if w < minW {
			w = minW
		}
		widths[i] = w
	}
	return widths
}

// Table рисует таблицу с заголовками, ресайзом ширины колонок, сортировкой и виртуализацией строк.
func (c *Context) Table(
	id string,
	cols []TableColumn,
	rowCount int,
	selected *int,
	sort *TableSort,
	drawCell func(row, col int, area render.Rect, isSelected bool),
	opts ...Option,
) TableEvent {
	ev := TableEvent{RowClicked: -1, RowDoubleClicked: -1, RowMiddleClicked: -1}
	if len(cols) == 0 {
		return ev
	}
	id = c.ResolveID("tbl", id)

	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(260)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	if w <= 0 || h <= 0 {
		return ev
	}

	rect := c.allocRectStyled(w, h, st)
	if !st.Disabled {
		c.registerFocusable(id)
	}
	if !c.IsInputBlocked() && rect.Contains(c.MouseX, c.MouseY) && (c.MousePressed || c.MiddleMousePressed) {
		c.FocusID = id
		c.RequestRedraw()
	}
	radius := c.Sf(c.Theme.RadiusSm)
	if st.Radius != nil {
		radius = c.Sf(*st.Radius)
	}
	borderCol := c.Theme.Border
	if c.FocusID == id {
		borderCol = c.Theme.BorderFocus
	}
	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, borderCol)

	// Инициализируем ширины колонок в dp
	if c.tableColWidths == nil {
		c.tableColWidths = make(map[string][]int)
	}
	if c.tableResizeCol == nil {
		c.tableResizeCol = make(map[string]int)
	}
	if c.tableResizeStartX == nil {
		c.tableResizeStartX = make(map[string]int)
	}
	if c.tableResizeStartW == nil {
		c.tableResizeStartW = make(map[string]int)
	}

	widthsDp, ok := c.tableColWidths[id]
	if !ok || len(widthsDp) != len(cols) {
		scale := c.Scale
		if scale <= 0 {
			scale = 1
		}
		totalDp := int(float32(rect.W) / scale)
		widthsDp = calcTableColWidthsDp(cols, totalDp)
		c.tableColWidths[id] = widthsDp
		c.tableResizeCol[id] = -1
	}

	headerH := c.S(32)
	if headerH > rect.H {
		headerH = rect.H
	}
	headerRect := render.Rect{X: rect.X + 1, Y: rect.Y + 1, W: rect.W - 2, H: headerH}
	bodyRect := render.Rect{X: rect.X + 1, Y: rect.Y + 1 + headerH, W: rect.W - 2, H: rect.H - headerH - 2}

	// Обработка активного ресайза колонки
	resizeID := id + ":colresize"
	if c.ActiveID == resizeID && c.MouseDown {
		colIdx := c.tableResizeCol[id]
		if colIdx >= 0 && colIdx < len(widthsDp) {
			deltaPx := c.MouseX - c.tableResizeStartX[id]
			scale := c.Scale
			if scale <= 0 {
				scale = 1
			}
			newW := c.tableResizeStartW[id] + int(float32(deltaPx)/scale)
			minW := cols[colIdx].MinWidthDp
			if minW <= 0 {
				minW = 48
			}
			if newW < minW {
				newW = minW
			}
			if widthsDp[colIdx] != newW {
				widthsDp[colIdx] = newW
				ev.ColumnResized = true
				c.RequestRedraw()
			}
		}
	} else if c.MouseReleased && c.ActiveID == resizeID {
		c.tableResizeCol[id] = -1
	}

	si := c.getScrollInfo("tbl:" + id)
	totalW := 0
	for i := range cols {
		totalW += c.S(widthsDp[i])
	}

	// Отрисовка заголовка таблицы
	c.Canvas.PushClip(headerRect)
	c.Canvas.FillRect(headerRect, c.Theme.Surface)
	c.Canvas.DrawLine(float32(headerRect.X), float32(headerRect.Y+headerRect.H-1), float32(headerRect.X+headerRect.W), float32(headerRect.Y+headerRect.H-1), 1.0, c.Theme.Border)

	pxSize := c.S(12)
	curX := headerRect.X - si.x
	blocked := c.IsInputBlocked()
	for i, col := range cols {
		cw := c.S(widthsDp[i])
		colRect := render.Rect{X: curX, Y: headerRect.Y, W: cw, H: headerRect.H}
		resizeZone := render.Rect{X: curX + cw - c.S(4), Y: headerRect.Y, W: c.S(8), H: headerRect.H}
		if col.Sortable && sort != nil {
			c.RegisterHit(id+":hdr:"+strconv.Itoa(i), headerRect.Intersect(colRect), platform.CursorHand)
		}
		c.RegisterHit(id+":res:"+strconv.Itoa(i), headerRect.Intersect(resizeZone), platform.CursorResizeH)

		inResize := !blocked && resizeZone.Contains(c.MouseX, c.MouseY)
		inHeader := !blocked && colRect.Contains(c.MouseX, c.MouseY) && !inResize

		if inResize || (c.ActiveID == resizeID && c.tableResizeCol[id] == i) {
			c.nextCursor = platform.CursorResizeH
			if c.MousePressed && inResize {
				c.ActiveID = resizeID
				c.tableResizeCol[id] = i
				c.tableResizeStartX[id] = c.MouseX
				c.tableResizeStartW[id] = widthsDp[i]
			}
		} else if inHeader && col.Sortable {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRect(colRect, c.Theme.BtnSecondaryHover.WithAlpha(120))
			if c.MousePressed && sort != nil {
				if sort.Column == i {
					sort.Desc = !sort.Desc
				} else {
					sort.Column = i
					sort.Desc = false
				}
				ev.SortChanged = true
				c.RequestRedraw()
			}
		}

		// Вертикальный разделитель колонки
		c.Canvas.DrawLine(float32(curX+cw-1), float32(headerRect.Y+c.S(6)), float32(curX+cw-1), float32(headerRect.Y+headerRect.H-c.S(6)), 1.0, c.Theme.Border)

		if c.Font != nil && cw > c.S(16) {
			c.Canvas.PushClip(render.Rect{X: colRect.X + c.S(8), Y: colRect.Y, W: colRect.W - c.S(22), H: colRect.H})
			_, th := c.Font.MeasureText(col.Title, pxSize)
			c.Font.DrawText(c.Canvas, colRect.X+c.S(8), colRect.Y+(colRect.H-th)/2, col.Title, pxSize, c.Theme.TextPrimary)
			c.Canvas.PopClip()
		}

		// Индикатор сортировки
		if sort != nil && sort.Column == i && cw > c.S(24) {
			ax := float32(colRect.X + colRect.W - c.S(14))
			ay := float32(colRect.Y + colRect.H/2)
			sz := c.Sf(3.5)
			if sort.Desc {
				c.Canvas.DrawLine(ax-sz, ay-sz*0.5, ax, ay+sz*0.5, c.Sf(1.5), c.Theme.Accent)
				c.Canvas.DrawLine(ax, ay+sz*0.5, ax+sz, ay-sz*0.5, c.Sf(1.5), c.Theme.Accent)
			} else {
				c.Canvas.DrawLine(ax-sz, ay+sz*0.5, ax, ay-sz*0.5, c.Sf(1.5), c.Theme.Accent)
				c.Canvas.DrawLine(ax, ay-sz*0.5, ax+sz, ay+sz*0.5, c.Sf(1.5), c.Theme.Accent)
			}
		}

		curX += cw
	}
	c.Canvas.PopClip()

	// Тело таблицы и виртуализация строк
	if bodyRect.H <= 0 {
		return ev
	}
	rowH := c.S(30)
	totalH := rowCount * rowH

	sbW := c.S(8)
	sbH := c.S(8)
	hasHSB := totalW > bodyRect.W
	viewH := bodyRect.H
	if hasHSB {
		viewH -= sbH + c.S(2)
		if viewH < 0 {
			viewH = 0
		}
	}
	hasSB := totalH > viewH
	contentW := bodyRect.W
	if hasSB {
		contentW -= sbW + c.S(2)
	}
	// Пересчитываем hasHSB с учётом возможного вертикального скроллбара
	hasHSB = totalW > contentW
	if hasHSB {
		viewH = bodyRect.H - sbH - c.S(2)
		if viewH < 0 {
			viewH = 0
		}
	}

	if !blocked && bodyRect.Contains(c.MouseX, c.MouseY) {
		if c.WheelY != 0 {
			si.y -= int(c.WheelY * float32(rowH*2))
			c.RequestRedraw()
		}
		if c.WheelX != 0 && totalW > contentW {
			si.x += int(c.WheelX * float32(c.S(36)))
			c.RequestRedraw()
		}
	}

	// Клавиатурная навигация в фокусе через общий Navigable
	if !st.Disabled && selected != nil && rowCount > 0 {
		nav := funcNavigable{
			count:    rowCount,
			selected: *selected,
			pageSize: viewH / rowH,
			onSelect: func(newRow int) {
				*selected = newRow
				ev.RowClicked = newRow
				if newRow*rowH < si.y {
					si.y = newRow * rowH
				} else if (newRow+1)*rowH > si.y+viewH && viewH > 0 {
					si.y = (newRow+1)*rowH - viewH
				}
			},
		}
		c.handleNavigable(id, nav, func(idx int) {
			ev.RowClicked = idx
		}, func(kp KeyPress) {
			switch kp.Code {
			case KeyLeft:
				if hasHSB {
					si.x -= c.S(40)
					c.RequestRedraw()
				}
			case KeyRight:
				if hasHSB {
					si.x += c.S(40)
					c.RequestRedraw()
				}
			case KeyEnter:
				if *selected >= 0 && *selected < rowCount {
					ev.RowClicked = *selected
					c.RequestRedraw()
				}
			}
		})
	}

	maxSc := totalH - viewH
	if maxSc < 0 {
		maxSc = 0
	}
	if si.y < 0 {
		si.y = 0
	}
	if si.y > maxSc {
		si.y = maxSc
	}
	maxScX := totalW - contentW
	if maxScX < 0 {
		maxScX = 0
	}
	if si.x < 0 {
		si.x = 0
	}
	if si.x > maxScX {
		si.x = maxScX
	}

	startRow, endRow := ComputeVisibleRange(si.y, viewH, rowH, rowCount)

	c.Canvas.PushClip(render.Rect{X: bodyRect.X, Y: bodyRect.Y, W: contentW, H: viewH})
	for r := startRow; r < endRow; r++ {
		ry := bodyRect.Y + r*rowH - si.y
		rowRect := render.Rect{X: bodyRect.X, Y: ry, W: contentW, H: rowH}
		c.RegisterHit(id+":row:"+strconv.Itoa(r), bodyRect.Intersect(rowRect), platform.CursorHand)
		isSel := selected != nil && *selected == r
		rowHover := !blocked && bodyRect.Contains(c.MouseX, c.MouseY) && c.MouseY < bodyRect.Y+viewH && rowRect.Contains(c.MouseX, c.MouseY) && c.MouseX < bodyRect.X+contentW

		if isSel {
			c.Canvas.FillRect(rowRect, c.Theme.Accent.WithAlpha(45))
		} else if rowHover {
			c.Canvas.FillRect(rowRect, c.Theme.Surface.WithAlpha(180))
		} else if r%2 == 1 {
			c.Canvas.FillRect(rowRect, c.Theme.Surface.WithAlpha(60))
		}

		if rowHover {
			c.nextCursor = platform.CursorHand
			if c.MousePressed {
				if selected != nil {
					*selected = r
				}
				ev.RowClicked = r
				if c.DoubleClicked() {
					ev.RowDoubleClicked = r
				}
				c.RequestRedraw()
			} else if c.MiddleMousePressed {
				if selected != nil {
					*selected = r
				}
				ev.RowMiddleClicked = r
				c.RequestRedraw()
			}
		}

		// Отрисовка ячеек с учётом горизонтальной прокрутки
		cellX := bodyRect.X - si.x
		for ci := range cols {
			cw := c.S(widthsDp[ci])
			cellRect := render.Rect{X: cellX, Y: ry, W: cw, H: rowH}
			if drawCell != nil && cellRect.X+cellRect.W > bodyRect.X && cellRect.X < bodyRect.X+contentW {
				c.Canvas.PushClip(cellRect)
				drawCell(r, ci, cellRect, isSel)
				c.Canvas.PopClip()
			}
			cellX += cw
		}

		c.Canvas.DrawLine(float32(rowRect.X), float32(ry+rowH-1), float32(rowRect.X+rowRect.W), float32(ry+rowH-1), 1.0, c.Theme.Border.WithAlpha(80))
	}
	c.Canvas.PopClip()

	// Вертикальный скроллбар
	if hasSB {
		c.drawVerticalScrollbar("tbl_sb:"+id, render.Rect{
			X: bodyRect.X + contentW + c.S(1),
			Y: bodyRect.Y + c.S(2),
			W: sbW - c.S(2),
			H: viewH - c.S(4),
		}, si, totalH, viewH)
	}
	// Горизонтальный скроллбар
	if hasHSB {
		c.drawHorizontalScrollbar("tbl_hsb:"+id, render.Rect{
			X: bodyRect.X + c.S(2),
			Y: bodyRect.Y + viewH + c.S(1),
			W: contentW - c.S(4),
			H: sbH - c.S(2),
		}, si, totalW, contentW)
	}

	return ev
}

// TreeNode описывает узел дерева для TreeView.
type TreeNode struct {
	ID            string
	Label         string
	IconSVG       string
	Expanded      bool
	Checked       bool
	Indeterminate bool
	Children      []*TreeNode
}

// FlatTreeItem представляет видимый узел дерева с глубиной вложенности.
type FlatTreeItem struct {
	Node  *TreeNode
	Depth int
}

// FlattenTree разворачивает раскрытые узлы дерева в плоский список для отображения.
func FlattenTree(roots []*TreeNode) []FlatTreeItem {
	var out []FlatTreeItem
	var walk func(nodes []*TreeNode, depth int)
	walk = func(nodes []*TreeNode, depth int) {
		for _, n := range nodes {
			if n == nil {
				continue
			}
			out = append(out, FlatTreeItem{Node: n, Depth: depth})
			if n.Expanded && len(n.Children) > 0 {
				walk(n.Children, depth+1)
			}
		}
	}
	walk(roots, 0)
	return out
}

// TreeView рисует древовидный список с раскрытием узлов и виртуализацией.
func (c *Context) TreeView(id string, roots []*TreeNode, selectedID *string, opts ...Option) bool {
	id = c.ResolveID("tree", id)
	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(240)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	if w <= 0 || h <= 0 {
		return false
	}

	rect := c.allocRectStyled(w, h, st)
	if !st.Disabled {
		c.registerFocusable(id)
	}
	if !c.IsInputBlocked() && rect.Contains(c.MouseX, c.MouseY) && (c.MousePressed || c.MiddleMousePressed) {
		c.FocusID = id
		c.RequestRedraw()
	}
	radius := c.Sf(c.Theme.RadiusSm)
	if st.Radius != nil {
		radius = c.Sf(*st.Radius)
	}
	borderCol := c.Theme.Border
	if c.FocusID == id {
		borderCol = c.Theme.BorderFocus
	}
	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, borderCol)

	flat := FlattenTree(roots)
	itemH := c.S(28)
	inner := render.Rect{X: rect.X + c.S(4), Y: rect.Y + c.S(4), W: rect.W - c.S(8), H: rect.H - c.S(8)}
	si := c.getScrollInfo("tree:" + id)
	totalH := len(flat) * itemH

	blocked := c.IsInputBlocked()
	if !blocked && inner.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
		si.y -= int(c.WheelY * float32(itemH*2))
		c.RequestRedraw()
	}

	changed := false
	if !st.Disabled && selectedID != nil && len(flat) > 0 {
		curIdx := -1
		for i, it := range flat {
			if it.Node.ID == *selectedID {
				curIdx = i
				break
			}
		}
		selectIdx := func(idx int) {
			*selectedID = flat[idx].Node.ID
			curIdx = idx
			if idx*itemH < si.y {
				si.y = idx * itemH
			} else if (idx+1)*itemH > si.y+inner.H {
				si.y = (idx+1)*itemH - inner.H
			}
		}
		toggleExpand := func(idx int) {
			if idx >= 0 && idx < len(flat) && len(flat[idx].Node.Children) > 0 {
				flat[idx].Node.Expanded = !flat[idx].Node.Expanded
				flat = FlattenTree(roots)
				totalH = len(flat) * itemH
				changed = true
				c.RequestRedraw()
			}
		}
		nav := funcNavigable{
			count:    len(flat),
			selected: curIdx,
			pageSize: inner.H / itemH,
			onSelect: selectIdx,
		}
		if c.handleNavigable(id, nav, toggleExpand, func(kp KeyPress) {
			switch kp.Code {
			case KeyRight:
				if curIdx >= 0 && curIdx < len(flat) {
					nd := flat[curIdx].Node
					if len(nd.Children) > 0 {
						if !nd.Expanded {
							nd.Expanded = true
							flat = FlattenTree(roots)
							totalH = len(flat) * itemH
							changed = true
							c.RequestRedraw()
						} else if curIdx+1 < len(flat) {
							selectIdx(curIdx + 1)
							changed = true
							c.RequestRedraw()
						}
					}
				}
			case KeyLeft:
				if curIdx >= 0 && curIdx < len(flat) {
					nd := flat[curIdx].Node
					if len(nd.Children) > 0 && nd.Expanded {
						nd.Expanded = false
						flat = FlattenTree(roots)
						totalH = len(flat) * itemH
						changed = true
						c.RequestRedraw()
					} else if flat[curIdx].Depth > 0 {
						targetDepth := flat[curIdx].Depth - 1
						for p := curIdx - 1; p >= 0; p-- {
							if flat[p].Depth == targetDepth {
								selectIdx(p)
								changed = true
								c.RequestRedraw()
								break
							}
						}
					}
				}
			case KeyEnter:
				toggleExpand(curIdx)
			}
		}) {
			changed = true
		}
	}

	maxSc := totalH - inner.H
	if maxSc < 0 {
		maxSc = 0
	}
	if si.y < 0 {
		si.y = 0
	}
	if si.y > maxSc {
		si.y = maxSc
	}

	hasSB := maxSc > 0
	sbW := c.S(8)
	contentW := inner.W
	if hasSB {
		contentW -= sbW + c.S(2)
	}

	startIdx, endIdx := ComputeVisibleRange(si.y, inner.H, itemH, len(flat))
	pxSize := c.S(13)

	c.Canvas.PushClip(render.Rect{X: inner.X, Y: inner.Y, W: contentW, H: inner.H})
	for i := startIdx; i < endIdx; i++ {
		item := flat[i]
		node := item.Node
		iy := inner.Y + i*itemH - si.y
		rowR := render.Rect{X: inner.X, Y: iy, W: contentW, H: itemH}

		isSel := selectedID != nil && *selectedID == node.ID
		c.RegisterHit(id+":node:"+node.ID, inner.Intersect(rowR), platform.CursorHand)
		rowHover := !blocked && inner.Contains(c.MouseX, c.MouseY) && rowR.Contains(c.MouseX, c.MouseY) && c.MouseX < inner.X+contentW

		if isSel {
			c.Canvas.FillRoundedRect(rowR, c.Sf(4), c.Theme.Accent.WithAlpha(45))
		} else if rowHover {
			c.Canvas.FillRoundedRect(rowR, c.Sf(4), c.Theme.Surface)
		}

		indent := item.Depth * c.S(18)
		toggleR := render.Rect{X: rowR.X + indent + c.S(4), Y: rowR.Y + (rowR.H-c.S(18))/2, W: c.S(18), H: c.S(18)}
		hasChildren := len(node.Children) > 0

		if rowHover {
			c.nextCursor = platform.CursorHand
			if c.MousePressed {
				if hasChildren && toggleR.Contains(c.MouseX, c.MouseY) {
					node.Expanded = !node.Expanded
					changed = true
					c.RequestRedraw()
				} else {
					if selectedID != nil {
						*selectedID = node.ID
						changed = true
						c.RequestRedraw()
					}
					if hasChildren && c.DoubleClicked() {
						node.Expanded = !node.Expanded
						changed = true
						c.RequestRedraw()
					}
				}
			}
		}

		if hasChildren {
			cx := float32(toggleR.X + toggleR.W/2)
			cy := float32(toggleR.Y + toggleR.H/2)
			arm := c.Sf(3.5)
			if node.Expanded {
				c.Canvas.DrawLine(cx-arm, cy-arm*0.5, cx, cy+arm*0.5, c.Sf(1.5), c.Theme.TextMuted)
				c.Canvas.DrawLine(cx, cy+arm*0.5, cx+arm, cy-arm*0.5, c.Sf(1.5), c.Theme.TextMuted)
			} else {
				c.Canvas.DrawLine(cx-arm*0.5, cy-arm, cx+arm*0.5, cy, c.Sf(1.5), c.Theme.TextMuted)
				c.Canvas.DrawLine(cx+arm*0.5, cy, cx-arm*0.5, cy+arm, c.Sf(1.5), c.Theme.TextMuted)
			}
		}

		curX := toggleR.X + toggleR.W + c.S(4)
		if node.IconSVG != "" {
			icSz := c.S(14)
			icCol := c.Theme.TextMuted
			if isSel {
				icCol = c.Theme.Accent
			}
			c.Canvas.DrawSVG(node.IconSVG, curX, rowR.Y+(rowR.H-icSz)/2, icSz, icCol)
			curX += icSz + c.S(6)
		}

		if c.Font != nil {
			_, th := c.Font.MeasureText(node.Label, pxSize)
			col := c.Theme.TextPrimary
			if isSel {
				col = c.Theme.Accent
			}
			c.Font.DrawText(c.Canvas, curX, rowR.Y+(rowR.H-th)/2, node.Label, pxSize, col)
		}
	}
	c.Canvas.PopClip()

	if hasSB {
		c.drawVerticalScrollbar("tree_sb:"+id, render.Rect{
			X: inner.X + contentW + c.S(1),
			Y: inner.Y,
			W: sbW - c.S(2),
			H: inner.H,
		}, si, totalH, inner.H)
	}

	return changed
}

// drawVerticalScrollbar рисует и обрабатывает перетаскиваемый вертикальный скроллбар.
func (c *Context) drawVerticalScrollbar(sbID string, track render.Rect, si *scrollInfo, totalH, viewH int) {
	if totalH <= viewH || track.H <= 0 {
		return
	}
	c.Canvas.FillRoundedRect(track, float32(track.W)/2, c.Theme.Surface)
	thumbH := (track.H * viewH) / totalH
	if thumbH < c.S(24) {
		thumbH = c.S(24)
	}
	maxSc := totalH - viewH
	thumbRange := track.H - thumbH
	thumbY := track.Y
	if maxSc > 0 && thumbRange > 0 {
		thumbY += (si.y * thumbRange) / maxSc
	}
	thumbR := render.Rect{X: track.X, Y: thumbY, W: track.W, H: thumbH}

	hover := !c.IsInputBlocked() && track.Contains(c.MouseX, c.MouseY)
	if hover && c.MousePressed {
		c.ActiveID = sbID
	}
	if c.ActiveID == sbID && c.MouseDown && thumbRange > 0 {
		relY := c.MouseY - track.Y - thumbH/2
		if relY < 0 {
			relY = 0
		}
		if relY > thumbRange {
			relY = thumbRange
		}
		si.y = (relY * maxSc) / thumbRange
		c.RequestRedraw()
	}

	thumbCol := c.Theme.Border
	if hover || c.ActiveID == sbID {
		thumbCol = c.Theme.TextMuted
	}
	c.Canvas.FillRoundedRect(thumbR, float32(track.W)/2, thumbCol)
}

// drawHorizontalScrollbar рисует и обрабатывает перетаскиваемый горизонтальный скроллбар.
func (c *Context) drawHorizontalScrollbar(sbID string, track render.Rect, si *scrollInfo, totalW, viewW int) {
	if totalW <= viewW || track.W <= 0 {
		return
	}
	c.Canvas.FillRoundedRect(track, float32(track.H)/2, c.Theme.Surface)
	thumbW := (track.W * viewW) / totalW
	if thumbW < c.S(24) {
		thumbW = c.S(24)
	}
	if thumbW > track.W {
		thumbW = track.W
	}
	maxSc := totalW - viewW
	thumbRange := track.W - thumbW
	thumbX := track.X
	if maxSc > 0 && thumbRange > 0 {
		thumbX += (si.x * thumbRange) / maxSc
	}
	thumbR := render.Rect{X: thumbX, Y: track.Y, W: thumbW, H: track.H}

	hover := !c.IsInputBlocked() && track.Contains(c.MouseX, c.MouseY)
	if hover && c.MousePressed {
		c.ActiveID = sbID
	}
	if c.ActiveID == sbID && c.MouseDown && thumbRange > 0 {
		relX := c.MouseX - track.X - thumbW/2
		if relX < 0 {
			relX = 0
		}
		if relX > thumbRange {
			relX = thumbRange
		}
		si.x = (relX * maxSc) / thumbRange
		c.RequestRedraw()
	}

	thumbCol := c.Theme.Border
	if hover || c.ActiveID == sbID {
		thumbCol = c.Theme.TextMuted
	}
	c.Canvas.FillRoundedRect(thumbR, float32(track.H)/2, thumbCol)
}
