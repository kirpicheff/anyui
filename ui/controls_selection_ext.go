package ui

import (
	"strconv"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

var (
	listAnchorStore  = make(map[string]int)
	treeAnchorStore  = make(map[string]int)
	tableAnchorStore = make(map[string]int)
	tableEditRow     = make(map[string]int)
	tableEditCol     = make(map[string]int)
	tableEditBuf     = make(map[string]string)
)

// ListViewEx отрисовывает виртуализированный список с поддержкой мультивыбора
// (Ctrl+Клик, Shift+Клик, Ctrl+A) и опциональных чекбоксов (если checked != nil).
func (c *Context) ListViewEx(
	id string,
	count int,
	itemHeightDp Dp,
	selected map[int]bool,
	checked []bool,
	drawItem func(index int, areaPx render.Rect, isSelected bool),
	opts ...Option,
) bool {
	if itemHeightDp <= 0 {
		itemHeightDp = 32
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
		h = c.Scale(220).Int()
	}
	r := c.allocRectStyled(w, h, st)
	if r.W <= 0 || r.H <= 0 {
		return false
	}
	c.registerFocusable(id)

	c.Canvas.DrawRoundedBox(r, c.ScaleF(c.Theme.RadiusSm), c.Theme.Card, 1.0, c.Theme.Border)

	rowH := c.Scale(itemHeightDp).Int()
	sbW := c.Scale(8).Int()
	viewRect := render.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
	contentW := viewRect.W
	totalH := count * rowH
	hasScroll := totalH > viewRect.H
	if hasScroll {
		contentW -= sbW
	}

	si := c.getScrollInfo(id)
	maxScroll := totalH - viewRect.H
	if maxScroll < 0 {
		maxScroll = 0
	}
	if !c.IsInputBlocked() && viewRect.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
		si.y -= int(c.WheelY * float32(rowH*2))
		if si.y < 0 {
			si.y = 0
		}
		if si.y > maxScroll {
			si.y = maxScroll
		}
		c.RequestRedraw()
	}
	if si.y > maxScroll {
		si.y = maxScroll
	}
	if si.y < 0 {
		si.y = 0
	}

	changed := false
	// Ctrl+A при фокусе
	if c.FocusID == id && !c.IsInputBlocked() && selected != nil {
		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl && kp.Code == KeyA {
				for i := 0; i < count; i++ {
					selected[i] = true
				}
				changed = true
				c.RequestRedraw()
			}
		}
	}

	showCheck := checked != nil && len(checked) >= count
	mods := c.currentMods()

	contentRect := render.Rect{X: viewRect.X, Y: viewRect.Y, W: contentW, H: viewRect.H}
	c.Canvas.PushClip(contentRect)

	startIdx, endIdx := ComputeVisibleRange(si.y, viewRect.H, rowH, count)
	for i := startIdx; i < endIdx; i++ {
		iy := viewRect.Y + i*rowH - si.y
		ir := render.Rect{X: contentRect.X, Y: iy, W: contentRect.W, H: rowH}
		c.RegisterHit(id+":row:"+strconv.Itoa(i), contentRect.Intersect(ir), platform.CursorArrow)
		isSel := selected != nil && selected[i]
		hov := !c.IsInputBlocked() && ir.Contains(c.MouseX, c.MouseY)

		if isSel {
			c.Canvas.FillRect(ir, c.Theme.Accent.WithAlpha(45))
		} else if hov {
			c.Canvas.FillRect(ir, c.Theme.Surface)
		}

		// Приглушение перемещаемой строки в ListViewEx
		fromIdx, _, reorderActive := c.ReorderSource(id)
		if reorderActive && fromIdx == i {
			c.Canvas.FillRect(ir, c.Theme.Bg.WithAlpha(160))
		}

		itemArea := ir
		if showCheck {
			cbSize := c.Scale(16).Int()
			cbR := render.Rect{X: ir.X + c.Scale(8).Int(), Y: ir.Y + (ir.H-cbSize)/2, W: cbSize, H: cbSize}
			c.RegisterHit(id+":cb:"+strconv.Itoa(i), contentRect.Intersect(cbR), platform.CursorHand)
			c.drawMiniCheckbox(cbR, checked[i], false)

			if hov && cbR.Contains(c.MouseX, c.MouseY) {
				c.nextCursor = platform.CursorHand
				if c.MousePressed {
					checked[i] = !checked[i]
					changed = true
					c.RequestRedraw()
					hov = false
				}
			}
			itemArea = render.Rect{X: cbR.X + cbR.W + c.Scale(6).Int(), Y: ir.Y, W: ir.W - (cbR.X + cbR.W + c.Scale(6).Int() - ir.X), H: ir.H}
		}

		if hov && c.MousePressed {
			if st.Reorderable {
				c.ArmReorder(id, i, "")
			}
			if selected != nil {
				c.FocusID = id
				anchor, hasAnchor := listAnchorStore[id]
				if mods.Shift && hasAnchor && anchor >= 0 && anchor < count {
					lo, hi := anchor, i
					if lo > hi {
						lo, hi = hi, lo
					}
					if !mods.Ctrl {
						for k := range selected {
							delete(selected, k)
						}
					}
					for k := lo; k <= hi; k++ {
						selected[k] = true
					}
				} else if mods.Ctrl {
					if selected[i] {
						delete(selected, i)
					} else {
						selected[i] = true
					}
					listAnchorStore[id] = i
				} else {
					for k := range selected {
						delete(selected, k)
					}
					selected[i] = true
					listAnchorStore[id] = i
				}
				changed = true
				c.RequestRedraw()
			}
		}

		if drawItem != nil {
			drawItem(i, itemArea, isSel)
		}
	}

	c.Canvas.PopClip()

	// Обработка перетаскивания (Reorder) строк ListViewEx
	if st.Reorderable {
		if c.HandleLinearReorder(id, viewRect, viewRect.X, contentW, rowH, count, si, maxScroll, st.OnReorder, func() {
			delete(listAnchorStore, id)
		}) {
			changed = true
		}
	}

	if hasScroll {
		sbTrack := render.Rect{X: viewRect.X + contentW, Y: viewRect.Y, W: sbW, H: viewRect.H}
		c.drawVerticalScrollbar(id+":vscroll", sbTrack, si, totalH, viewRect.H)
	}

	return changed
}

// SetTreeNodeChecked каскадно устанавливает состояние Checked узлу и всем его дочерним элементам.
func SetTreeNodeChecked(node *TreeNode, checked bool) {
	if node == nil {
		return
	}
	node.Checked = checked
	node.Indeterminate = false
	for _, ch := range node.Children {
		SetTreeNodeChecked(ch, checked)
	}
}

// SyncTreeCheckState пересчитывает состояния Checked и Indeterminate для всех родительских узлов дерева.
func SyncTreeCheckState(roots []*TreeNode) {
	var walk func(n *TreeNode) (allChecked, anyChecked bool)
	walk = func(n *TreeNode) (bool, bool) {
		if n == nil {
			return false, false
		}
		if len(n.Children) == 0 {
			n.Indeterminate = false
			return n.Checked, n.Checked
		}
		all := true
		any := false
		for _, ch := range n.Children {
			cAll, cAny := walk(ch)
			if !cAll {
				all = false
			}
			if cAny {
				any = true
			}
		}
		n.Checked = all
		n.Indeterminate = any && !all
		return all, any
	}
	for _, r := range roots {
		walk(r)
	}
}

// TreeViewEx отрисовывает дерево с мультивыбором узлов и опциональными каскадными чекбоксами.
func (c *Context) TreeViewEx(
	id string,
	roots []*TreeNode,
	selectedIDs map[string]bool,
	showCheckboxes bool,
	opts ...Option,
) bool {
	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}
	h := c.Scale(240).Int()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	}
	r := c.allocRectStyled(w, h, st)
	c.registerFocusable(id)

	c.Canvas.DrawRoundedBox(r, c.ScaleF(c.Theme.RadiusSm), c.Theme.Card, 1.0, c.Theme.Border)

	flat := FlattenTree(roots)
	rowH := c.Scale(28).Int()
	sbW := c.Scale(8).Int()
	viewRect := render.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
	contentW := viewRect.W
	totalH := len(flat) * rowH
	hasScroll := totalH > viewRect.H
	if hasScroll {
		contentW -= sbW
	}

	si := c.getScrollInfo(id)
	maxScroll := totalH - viewRect.H
	if maxScroll < 0 {
		maxScroll = 0
	}
	if !c.IsInputBlocked() && viewRect.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
		si.y -= int(c.WheelY * float32(rowH*2))
		if si.y < 0 {
			si.y = 0
		}
		if si.y > maxScroll {
			si.y = maxScroll
		}
		c.RequestRedraw()
	}

	changed := false
	if c.FocusID == id && !c.IsInputBlocked() && selectedIDs != nil {
		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl && kp.Code == KeyA {
				for _, item := range flat {
					selectedIDs[item.Node.ID] = true
				}
				changed = true
				c.RequestRedraw()
			}
		}
	}

	mods := c.currentMods()
	contentRect := render.Rect{X: viewRect.X, Y: viewRect.Y, W: contentW, H: viewRect.H}
	c.Canvas.PushClip(contentRect)

	startIdx, endIdx := ComputeVisibleRange(si.y, viewRect.H, rowH, len(flat))
	pxSize := c.Scale(13).Int()

	for i := startIdx; i < endIdx; i++ {
		item := flat[i]
		n := item.Node
		iy := viewRect.Y + i*rowH - si.y
		ir := render.Rect{X: contentRect.X, Y: iy, W: contentRect.W, H: rowH}
		c.RegisterHit(id+":node:"+n.ID, contentRect.Intersect(ir), platform.CursorArrow)
		isSel := selectedIDs != nil && selectedIDs[n.ID]
		hov := !c.IsInputBlocked() && ir.Contains(c.MouseX, c.MouseY)

		if isSel {
			c.Canvas.FillRect(ir, c.Theme.Accent.WithAlpha(45))
		} else if hov {
			c.Canvas.FillRect(ir, c.Theme.Surface)
		}

		curX := ir.X + c.Scale(8).Int() + item.Depth*c.Scale(18).Int()

		// Стрелка раскрытия/сворачивания
		arrowR := render.Rect{X: curX, Y: ir.Y + (ir.H-c.Scale(16).Int())/2, W: c.Scale(16).Int(), H: c.Scale(16).Int()}
		if len(n.Children) > 0 {
			c.RegisterHit(id+":arrow:"+n.ID, contentRect.Intersect(arrowR), platform.CursorHand)
			cx := float32(arrowR.X + arrowR.W/2)
			cy := float32(arrowR.Y + arrowR.H/2)
			if n.Expanded {
				c.Canvas.DrawLine(cx-c.ScaleF(4), cy-c.ScaleF(2), cx, cy+c.ScaleF(2), 1.5, c.Theme.TextMuted)
				c.Canvas.DrawLine(cx, cy+c.ScaleF(2), cx+c.ScaleF(4), cy-c.ScaleF(2), 1.5, c.Theme.TextMuted)
			} else {
				c.Canvas.DrawLine(cx-c.ScaleF(2), cy-c.ScaleF(4), cx+c.ScaleF(2), cy, 1.5, c.Theme.TextMuted)
				c.Canvas.DrawLine(cx+c.ScaleF(2), cy, cx-c.ScaleF(2), cy+c.ScaleF(4), 1.5, c.Theme.TextMuted)
			}
			if hov && arrowR.Contains(c.MouseX, c.MouseY) {
				c.nextCursor = platform.CursorHand
				if c.MousePressed {
					n.Expanded = !n.Expanded
					changed = true
					c.RequestRedraw()
					hov = false
				}
			}
		}
		curX += c.Scale(18).Int()

		// Чекбокс узла
		if showCheckboxes {
			cbSize := c.Scale(15).Int()
			cbR := render.Rect{X: curX, Y: ir.Y + (ir.H-cbSize)/2, W: cbSize, H: cbSize}
			c.RegisterHit(id+":tcb:"+n.ID, contentRect.Intersect(cbR), platform.CursorHand)
			c.drawMiniCheckbox(cbR, n.Checked, n.Indeterminate)
			if hov && cbR.Contains(c.MouseX, c.MouseY) {
				c.nextCursor = platform.CursorHand
				if c.MousePressed {
					nextVal := !n.Checked
					SetTreeNodeChecked(n, nextVal)
					SyncTreeCheckState(roots)
					changed = true
					c.RequestRedraw()
					hov = false
				}
			}
			curX += cbSize + c.Scale(6).Int()
		}

		if n.IconSVG != "" {
			icSize := c.Scale(14).Int()
			c.Canvas.DrawSVG(n.IconSVG, curX, ir.Y+(ir.H-icSize)/2, icSize, c.Theme.TextMuted)
			curX += icSize + c.Scale(6).Int()
		}

		if c.Font != nil {
			_, th := c.Font.MeasureText(n.Label, pxSize)
			c.Font.DrawText(c.Canvas, curX, ir.Y+(ir.H-th)/2, n.Label, pxSize, c.Theme.TextPrimary)
		}

		if hov && c.MousePressed && selectedIDs != nil {
			c.FocusID = id
			anchor, hasAnchor := treeAnchorStore[id]
			if mods.Shift && hasAnchor && anchor >= 0 && anchor < len(flat) {
				lo, hi := anchor, i
				if lo > hi {
					lo, hi = hi, lo
				}
				if !mods.Ctrl {
					for k := range selectedIDs {
						delete(selectedIDs, k)
					}
				}
				for k := lo; k <= hi; k++ {
					selectedIDs[flat[k].Node.ID] = true
				}
			} else if mods.Ctrl {
				if selectedIDs[n.ID] {
					delete(selectedIDs, n.ID)
				} else {
					selectedIDs[n.ID] = true
				}
				treeAnchorStore[id] = i
			} else {
				for k := range selectedIDs {
					delete(selectedIDs, k)
				}
				selectedIDs[n.ID] = true
				treeAnchorStore[id] = i
			}
			changed = true
			c.RequestRedraw()
		}
	}

	c.Canvas.PopClip()

	if hasScroll {
		sbTrack := render.Rect{X: viewRect.X + contentW, Y: viewRect.Y, W: sbW, H: viewRect.H}
		c.drawVerticalScrollbar(id+":vscroll", sbTrack, si, totalH, viewRect.H)
	}

	return changed
}

// TableExConfig задаёт расширенные возможности таблицы TableEx:
// мультивыбор, чекбоксы, закреплённые колонки, inline-редактирование и автоподбор ширины.
type TableExConfig struct {
	SelectedRows         map[int]bool
	ShowCheckboxes       bool
	CheckedRows          []bool
	FrozenCols           int
	GetCellText          func(row, col int) string
	OnCellEdit           func(row, col int, newValue string)
	AutoFitOnDoubleClick bool
}

// TableEx отрисовывает расширенную таблицу с мультивыбором строк, колонкой чекбоксов,
// закреплением колонок, редактированием ячеек по двойному клику и автоподбором ширины.
func (c *Context) TableEx(
	id string,
	cols []TableColumn,
	rowCount int,
	cfg *TableExConfig,
	sort *TableSort,
	drawCell func(row, col int, areaPx render.Rect, isSelected bool),
	opts ...Option,
) TableEvent {
	ev := TableEvent{RowClicked: -1, RowDoubleClicked: -1, RowMiddleClicked: -1}
	if len(cols) == 0 {
		return ev
	}
	if cfg == nil {
		cfg = &TableExConfig{}
	}

	st := ApplyOptions(opts)
	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}
	h := c.Scale(260).Int()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	}
	r := c.allocRectStyled(w, h, st)
	c.registerFocusable(id)

	c.Canvas.DrawRoundedBox(r, c.ScaleF(c.Theme.RadiusSm), c.Theme.Card, 1.0, c.Theme.Border)

	// Инициализация ширин колонок
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

	checkColW := 0
	if cfg.ShowCheckboxes {
		checkColW = c.Scale(34).Int()
	}

	colWidths := c.tableColWidths[id]
	if len(colWidths) != len(cols) {
		scale := c.DpiScale
		if scale <= 0 {
			scale = 1
		}
		availDp := Dp(float32(r.W-checkColW) / scale)
		widthsDp := calcTableColWidthsDp(cols, availDp)
		colWidths = make([]int, len(cols))
		for i, wdp := range widthsDp {
			colWidths[i] = c.Scale(Dp(wdp)).Int()
		}
		c.tableColWidths[id] = colWidths
	}

	frozenCols := cfg.FrozenCols
	if frozenCols < 0 {
		frozenCols = 0
	}
	if frozenCols > len(cols) {
		frozenCols = len(cols)
	}

	frozenW := checkColW
	for i := 0; i < frozenCols; i++ {
		frozenW += colWidths[i]
	}

	totalColsW := checkColW
	for _, cw := range colWidths {
		totalColsW += cw
	}

	hdrH := c.Scale(32).Int()
	rowH := c.Scale(30).Int()
	sbSize := c.Scale(8).Int()

	viewRect := render.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: r.H - 2}
	needVScroll := rowCount*rowH > (viewRect.H - hdrH)
	needHScroll := totalColsW > viewRect.W

	contentW := viewRect.W
	if needVScroll {
		contentW -= sbSize
	}
	bodyH := viewRect.H - hdrH
	if needHScroll {
		bodyH -= sbSize
	}
	if bodyH < 0 {
		bodyH = 0
	}

	si := c.getScrollInfo(id)
	maxScrollY := rowCount*rowH - bodyH
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	maxScrollX := totalColsW - contentW
	if maxScrollX < 0 {
		maxScrollX = 0
	}

	if !c.IsInputBlocked() && viewRect.Contains(c.MouseX, c.MouseY) {
		if c.WheelY != 0 {
			si.y -= int(c.WheelY * float32(rowH*2))
			if si.y < 0 {
				si.y = 0
			}
			if si.y > maxScrollY {
				si.y = maxScrollY
			}
			c.RequestRedraw()
		}
		if c.WheelX != 0 {
			si.x += int(c.WheelX * float32(c.Scale(30).Int()))
			if si.x < 0 {
				si.x = 0
			}
			if si.x > maxScrollX {
				si.x = maxScrollX
			}
			c.RequestRedraw()
		}
	}

	// Ctrl+A при фокусе
	if c.FocusID == id && !c.IsInputBlocked() && cfg.SelectedRows != nil {
		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl && kp.Code == KeyA {
				for i := 0; i < rowCount; i++ {
					cfg.SelectedRows[i] = true
				}
				c.RequestRedraw()
			}
		}
	}

	// Обработка активного inline-редактирования
	editRow, hasEdit := tableEditRow[id]
	editCol := tableEditCol[id]
	if hasEdit && editRow >= 0 && editCol >= 0 && !c.IsInputBlocked() {
		buf := []rune(tableEditBuf[id])
		for _, kp := range c.KeyEvents {
			if kp.Code == KeyEnter {
				if cfg.OnCellEdit != nil {
					cfg.OnCellEdit(editRow, editCol, string(buf))
				}
				delete(tableEditRow, id)
				delete(tableEditCol, id)
				c.RequestRedraw()
				hasEdit = false
				break
			} else if kp.Code == KeyEscape {
				delete(tableEditRow, id)
				delete(tableEditCol, id)
				c.RequestRedraw()
				hasEdit = false
				break
			} else if kp.Code == KeyBackspace && len(buf) > 0 {
				buf = buf[:len(buf)-1]
				tableEditBuf[id] = string(buf)
				c.RequestRedraw()
			}
		}
		if hasEdit {
			for _, ch := range c.TypedRunes {
				if ch >= 32 {
					buf = append(buf, ch)
					tableEditBuf[id] = string(buf)
					c.RequestRedraw()
				}
			}
		}
	}

	// Ресайз колонки и автоподбор ширины по двойному клику на разделителе
	if resCol, ok := c.tableResizeCol[id]; ok && resCol >= 0 {
		if c.MouseDown {
			delta := c.MouseX - c.tableResizeStartX[id]
			nw := c.tableResizeStartW[id] + delta
			minW := c.Scale(40).Int()
			if cols[resCol].MinWidthDp > 0 {
				minW = c.Scale(cols[resCol].MinWidthDp).Int()
			}
			if nw < minW {
				nw = minW
			}
			colWidths[resCol] = nw
			ev.ColumnResized = true
			c.RequestRedraw()
		} else {
			delete(c.tableResizeCol, id)
		}
	}

	// Вспомогательная функция вычисления X-позиции колонки с учётом frozenCols
	colStartX := func(colIdx int) int {
		if colIdx < frozenCols {
			x := viewRect.X + checkColW
			for k := 0; k < colIdx; k++ {
				x += colWidths[k]
			}
			return x
		}
		x := viewRect.X + checkColW - si.x
		for k := 0; k < colIdx; k++ {
			x += colWidths[k]
		}
		return x
	}

	// --- 1. Отрисовка шапки ---
	hdrRect := render.Rect{X: viewRect.X, Y: viewRect.Y, W: contentW, H: hdrH}
	c.Canvas.FillRect(hdrRect, c.Theme.Surface)
	c.Canvas.FillRect(render.Rect{X: hdrRect.X, Y: hdrRect.Y + hdrRect.H - 1, W: hdrRect.W, H: 1}, c.Theme.Border)

	drawHeaderCol := func(ci int) {
		cx := colStartX(ci)
		cw := colWidths[ci]
		cr := render.Rect{X: cx, Y: hdrRect.Y, W: cw, H: hdrH}
		resZone := render.Rect{X: cr.X + cr.W - c.Scale(4).Int(), Y: cr.Y, W: c.Scale(8).Int(), H: cr.H}
		if cols[ci].Sortable && sort != nil {
			c.RegisterHit(id+":hdr:"+strconv.Itoa(ci), hdrRect.Intersect(cr), platform.CursorHand)
		}
		c.RegisterHit(id+":res:"+strconv.Itoa(ci), hdrRect.Intersect(resZone), platform.CursorResizeH)

		if !c.IsInputBlocked() && resZone.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorResizeH
			if c.DoubleClicked() && cfg.AutoFitOnDoubleClick && cfg.GetCellText != nil && c.Font != nil {
				maxTw, _ := c.Font.MeasureText(cols[ci].Title, c.Scale(12).Int())
				limit := rowCount
				if limit > 200 {
					limit = 200
				}
				for ri := 0; ri < limit; ri++ {
					tw, _ := c.Font.MeasureText(cfg.GetCellText(ri, ci), c.Scale(12).Int())
					if tw > maxTw {
						maxTw = tw
					}
				}
				colWidths[ci] = maxTw + c.Scale(24).Int()
				ev.ColumnResized = true
				c.RequestRedraw()
			} else if c.MousePressed {
				c.tableResizeCol[id] = ci
				c.tableResizeStartX[id] = c.MouseX
				c.tableResizeStartW[id] = cw
			}
		} else if cols[ci].Sortable && sort != nil && !c.IsInputBlocked() && cr.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorHand
			if c.MousePressed {
				if sort.Column == ci {
					sort.Desc = !sort.Desc
				} else {
					sort.Column = ci
					sort.Desc = false
				}
				ev.SortChanged = true
				c.RequestRedraw()
			}
		}

		c.Canvas.PushClip(cr)
		if c.Font != nil {
			title := cols[ci].Title
			if sort != nil && sort.Column == ci {
				if sort.Desc {
					title += " v"
				} else {
					title += " ^"
				}
			}
			_, th := c.Font.MeasureText(title, c.Scale(12).Int())
			c.Font.DrawText(c.Canvas, cr.X+c.Scale(8).Int(), cr.Y+(cr.H-th)/2, title, c.Scale(12).Int(), c.Theme.TextMuted)
		}
		c.Canvas.PopClip()
		c.Canvas.FillRect(render.Rect{X: cr.X + cr.W - 1, Y: cr.Y + c.Scale(6).Int(), W: 1, H: cr.H - c.Scale(12).Int()}, c.Theme.Border)
	}

	// Сначала прокручиваемые колонки шапки (в клипе правее frozenW)
	scrollClipX := viewRect.X + frozenW
	scrollClipW := contentW - frozenW
	if scrollClipW > 0 {
		c.Canvas.PushClip(render.Rect{X: scrollClipX, Y: hdrRect.Y, W: scrollClipW, H: hdrRect.H})
		for ci := frozenCols; ci < len(cols); ci++ {
			drawHeaderCol(ci)
		}
		c.Canvas.PopClip()
	}

	// Затем закреплённые колонки шапки + чекбокс "Выбрать все"
	if frozenW > 0 {
		c.Canvas.PushClip(render.Rect{X: viewRect.X, Y: hdrRect.Y, W: frozenW, H: hdrRect.H})
		c.Canvas.FillRect(render.Rect{X: viewRect.X, Y: hdrRect.Y, W: frozenW, H: hdrRect.H - 1}, c.Theme.Surface)
		if cfg.ShowCheckboxes && cfg.CheckedRows != nil && len(cfg.CheckedRows) >= rowCount {
			allChecked := rowCount > 0
			anyChecked := false
			for ri := 0; ri < rowCount; ri++ {
				if cfg.CheckedRows[ri] {
					anyChecked = true
				} else {
					allChecked = false
				}
			}
			cbSize := c.Scale(15).Int()
			cbR := render.Rect{X: viewRect.X + (checkColW-cbSize)/2, Y: hdrRect.Y + (hdrH-cbSize)/2, W: cbSize, H: cbSize}
			c.RegisterHit(id+":allcb", cbR, platform.CursorHand)
			c.drawMiniCheckbox(cbR, allChecked, anyChecked && !allChecked)
			if !c.IsInputBlocked() && cbR.Contains(c.MouseX, c.MouseY) {
				c.nextCursor = platform.CursorHand
				if c.MousePressed {
					target := !allChecked
					for ri := 0; ri < rowCount; ri++ {
						cfg.CheckedRows[ri] = target
					}
					c.RequestRedraw()
				}
			}
		}
		for ci := 0; ci < frozenCols; ci++ {
			drawHeaderCol(ci)
		}
		c.Canvas.PopClip()
	}

	// --- 2. Отрисовка строк ---
	bodyRect := render.Rect{X: viewRect.X, Y: viewRect.Y + hdrH, W: contentW, H: bodyH}
	startRow, endRow := ComputeVisibleRange(si.y, bodyH, rowH, rowCount)
	mods := c.currentMods()

	drawRowRange := func(colStart, colEnd int, clipR render.Rect, allowRowClick bool) {
		if clipR.W <= 0 || clipR.H <= 0 {
			return
		}
		c.Canvas.PushClip(clipR)
		for ri := startRow; ri < endRow; ri++ {
			ry := bodyRect.Y + ri*rowH - si.y
			rowR := render.Rect{X: clipR.X, Y: ry, W: clipR.W, H: rowH}
			c.RegisterHit(id+":row:"+strconv.Itoa(ri), clipR.Intersect(rowR), platform.CursorArrow)
			isSel := cfg.SelectedRows != nil && cfg.SelectedRows[ri]
			hovRow := !c.IsInputBlocked() && render.Rect{X: bodyRect.X, Y: ry, W: bodyRect.W, H: rowH}.Contains(c.MouseX, c.MouseY)

			if isSel {
				c.Canvas.FillRect(rowR, c.Theme.Accent.WithAlpha(40))
			} else if hovRow {
				c.Canvas.FillRect(rowR, c.Theme.Surface)
			} else if ri%2 == 1 {
				c.Canvas.FillRect(rowR, c.Theme.Surface.WithAlpha(60))
			}

			if colStart == 0 && cfg.ShowCheckboxes && cfg.CheckedRows != nil && len(cfg.CheckedRows) >= rowCount {
				cbSize := c.Scale(15).Int()
				cbR := render.Rect{X: viewRect.X + (checkColW-cbSize)/2, Y: ry + (rowH-cbSize)/2, W: cbSize, H: cbSize}
				c.RegisterHit(id+":rcb:"+strconv.Itoa(ri), clipR.Intersect(cbR), platform.CursorHand)
				c.drawMiniCheckbox(cbR, cfg.CheckedRows[ri], false)
				if !c.IsInputBlocked() && cbR.Contains(c.MouseX, c.MouseY) {
					c.nextCursor = platform.CursorHand
					if c.MousePressed {
						cfg.CheckedRows[ri] = !cfg.CheckedRows[ri]
						c.RequestRedraw()
						allowRowClick = false
					}
				}
			}

			for ci := colStart; ci < colEnd; ci++ {
				cx := colStartX(ci)
				cw := colWidths[ci]
				cellR := render.Rect{X: cx, Y: ry, W: cw, H: rowH}

				// Двойной клик по ячейке запускает inline-редактирование
				if !c.IsInputBlocked() && cellR.Contains(c.MouseX, c.MouseY) && c.DoubleClicked() {
					ev.RowDoubleClicked = ri
					if cfg.OnCellEdit != nil && cfg.GetCellText != nil {
						tableEditRow[id] = ri
						tableEditCol[id] = ci
						tableEditBuf[id] = cfg.GetCellText(ri, ci)
						c.RequestRedraw()
					}
				}

				if hasEdit && editRow == ri && editCol == ci {
					c.Canvas.DrawRoundedBox(render.Rect{X: cellR.X + 2, Y: cellR.Y + 2, W: cellR.W - 4, H: cellR.H - 4}, 3, c.Theme.Bg, 1.5, c.Theme.Accent)
					if c.Font != nil {
						txt := tableEditBuf[id]
						_, th := c.Font.MeasureText(txt, c.Scale(12).Int())
						c.Font.DrawText(c.Canvas, cellR.X+c.Scale(6).Int(), cellR.Y+(cellR.H-th)/2, txt, c.Scale(12).Int(), c.Theme.TextPrimary)
					}
				} else if drawCell != nil {
					c.Canvas.PushClip(cellR)
					drawCell(ri, ci, cellR, isSel)
					c.Canvas.PopClip()
				}
			}

			if allowRowClick && hovRow && c.MousePressed {
				ev.RowClicked = ri
				c.FocusID = id
				if cfg.SelectedRows != nil {
					anchor, hasAnchor := tableAnchorStore[id]
					if mods.Shift && hasAnchor && anchor >= 0 && anchor < rowCount {
						lo, hi := anchor, ri
						if lo > hi {
							lo, hi = hi, lo
						}
						if !mods.Ctrl {
							for k := range cfg.SelectedRows {
								delete(cfg.SelectedRows, k)
							}
						}
						for k := lo; k <= hi; k++ {
							cfg.SelectedRows[k] = true
						}
					} else if mods.Ctrl {
						if cfg.SelectedRows[ri] {
							delete(cfg.SelectedRows, ri)
						} else {
							cfg.SelectedRows[ri] = true
						}
						tableAnchorStore[id] = ri
					} else {
						for k := range cfg.SelectedRows {
							delete(cfg.SelectedRows, k)
						}
						cfg.SelectedRows[ri] = true
						tableAnchorStore[id] = ri
					}
				}
				c.RequestRedraw()
			}
		}
		c.Canvas.PopClip()
	}

	// Сначала прокручиваемая часть строк
	drawRowRange(frozenCols, len(cols), render.Rect{X: scrollClipX, Y: bodyRect.Y, W: scrollClipW, H: bodyH}, true)
	// Затем закреплённая часть строк поверх
	if frozenW > 0 {
		drawRowRange(0, frozenCols, render.Rect{X: bodyRect.X, Y: bodyRect.Y, W: frozenW, H: bodyH}, true)
		c.Canvas.FillRect(render.Rect{X: bodyRect.X + frozenW - 1, Y: viewRect.Y, W: 1, H: viewRect.H}, c.Theme.Border)
	}

	if needVScroll {
		c.drawVerticalScrollbar(id+":vscroll", render.Rect{X: viewRect.X + contentW, Y: bodyRect.Y, W: sbSize, H: bodyH}, si, rowCount*rowH, bodyH)
	}
	if needHScroll {
		c.drawHorizontalScrollbar(id+":hscroll", render.Rect{X: bodyRect.X, Y: bodyRect.Y + bodyH, W: contentW, H: sbSize}, si, totalColsW, contentW)
	}

	return ev
}

func (c *Context) drawMiniCheckbox(r render.Rect, checked, indeterminate bool) {
	if checked || indeterminate {
		c.Canvas.FillRoundedRect(r, c.ScaleF(3), c.Theme.Accent)
		cx := float32(r.X + r.W/2)
		cy := float32(r.Y + r.H/2)
		white := render.RGBA(255, 255, 255, 255)
		if indeterminate {
			c.Canvas.DrawLine(cx-c.ScaleF(4), cy, cx+c.ScaleF(4), cy, 2.0, white)
		} else {
			c.Canvas.DrawLine(cx-c.ScaleF(3.5), cy, cx-c.ScaleF(0.5), cy+c.ScaleF(3), 1.8, white)
			c.Canvas.DrawLine(cx-c.ScaleF(0.5), cy+c.ScaleF(3), cx+c.ScaleF(4), cy-c.ScaleF(3), 1.8, white)
		}
	} else {
		c.Canvas.DrawRoundedBox(r, c.ScaleF(3), c.Theme.Bg, 1.0, c.Theme.Border)
	}
}
