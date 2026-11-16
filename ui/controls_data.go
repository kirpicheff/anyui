package ui

import (
	"anyui/platform"
	"anyui/render"
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
	RowClicked    int
	SortChanged   bool
	ColumnResized bool
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
	ev := TableEvent{RowClicked: -1}
	if len(cols) == 0 {
		return ev
	}

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

	rect := c.allocRect(w, h)
	radius := c.Sf(c.Theme.RadiusSm)
	if st.Radius != nil {
		radius = c.Sf(*st.Radius)
	}
	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, c.Theme.Border)

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
		widthsDp = make([]int, len(cols))
		for i, col := range cols {
			if col.WidthDp > 0 {
				widthsDp[i] = col.WidthDp
			} else {
				widthsDp[i] = 120
			}
		}
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

	// Отрисовка заголовка таблицы
	c.Canvas.PushClip(headerRect)
	c.Canvas.FillRect(headerRect, c.Theme.Surface)
	c.Canvas.DrawLine(float32(headerRect.X), float32(headerRect.Y+headerRect.H-1), float32(headerRect.X+headerRect.W), float32(headerRect.Y+headerRect.H-1), 1.0, c.Theme.Border)

	pxSize := c.S(12)
	curX := headerRect.X
	blocked := c.IsInputBlocked()
	for i, col := range cols {
		cw := c.S(widthsDp[i])
		colRect := render.Rect{X: curX, Y: headerRect.Y, W: cw, H: headerRect.H}
		resizeZone := render.Rect{X: curX + cw - c.S(4), Y: headerRect.Y, W: c.S(8), H: headerRect.H}

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
	si := c.getScrollInfo("tbl:" + id)
	totalH := rowCount * rowH

	if !blocked && bodyRect.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
		si.y -= int(c.WheelY * float32(rowH*2))
		c.RequestRedraw()
	}
	maxSc := totalH - bodyRect.H
	if maxSc < 0 {
		maxSc = 0
	}
	if si.y < 0 {
		si.y = 0
	}
	if si.y > maxSc {
		si.y = maxSc
	}

	sbW := c.S(8)
	hasSB := maxSc > 0
	contentW := bodyRect.W
	if hasSB {
		contentW -= sbW + c.S(2)
	}

	startRow, endRow := ComputeVisibleRange(si.y, bodyRect.H, rowH, rowCount)

	c.Canvas.PushClip(render.Rect{X: bodyRect.X, Y: bodyRect.Y, W: contentW, H: bodyRect.H})
	for r := startRow; r < endRow; r++ {
		ry := bodyRect.Y + r*rowH - si.y
		rowRect := render.Rect{X: bodyRect.X, Y: ry, W: contentW, H: rowH}
		isSel := selected != nil && *selected == r
		rowHover := !blocked && bodyRect.Contains(c.MouseX, c.MouseY) && rowRect.Contains(c.MouseX, c.MouseY) && c.MouseX < bodyRect.X+contentW

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
				c.RequestRedraw()
			}
		}

		// Отрисовка ячеек
		cellX := bodyRect.X
		for ci := range cols {
			cw := c.S(widthsDp[ci])
			cellRect := render.Rect{X: cellX, Y: ry, W: cw, H: rowH}
			if drawCell != nil {
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
			H: bodyRect.H - c.S(4),
		}, si, totalH, bodyRect.H)
	}

	return ev
}

// TreeNode описывает узел дерева для TreeView.
type TreeNode struct {
	ID       string
	Label    string
	IconSVG  string
	Expanded bool
	Children []*TreeNode
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

	rect := c.allocRect(w, h)
	radius := c.Sf(c.Theme.RadiusSm)
	if st.Radius != nil {
		radius = c.Sf(*st.Radius)
	}
	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, c.Theme.Border)

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

	changed := false
	startIdx, endIdx := ComputeVisibleRange(si.y, inner.H, itemH, len(flat))
	pxSize := c.S(13)

	c.Canvas.PushClip(render.Rect{X: inner.X, Y: inner.Y, W: contentW, H: inner.H})
	for i := startIdx; i < endIdx; i++ {
		item := flat[i]
		node := item.Node
		iy := inner.Y + i*itemH - si.y
		rowR := render.Rect{X: inner.X, Y: iy, W: contentW, H: itemH}

		isSel := selectedID != nil && *selectedID == node.ID
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
				} else if selectedID != nil {
					*selectedID = node.ID
					changed = true
					c.RequestRedraw()
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
