package ui

import (
	"strconv"

	"git.gtw.su/kirpich/anyui/render"
)

type layoutDir uint8

const (
	dirColumn layoutDir = iota
	dirRow
	dirGrid
	dirFlow
)

// Align задаёт выравнивание элементов по поперечной оси.
type Align uint8

const (
	AlignStart Align = iota
	AlignCenter
	AlignEnd
	AlignStretch
)

// Justify задаёт распределение элементов по главной оси контейнера.
type Justify uint8

const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
	JustifySpaceAround
)

// FlexItem описывает элемент для пропорционального деления пространства в RowFlex / ColumnFlex.
type FlexItem struct {
	Weight  float32 // Пропорциональный вес (используется, если FixedDp <= 0)
	FixedDp int     // Фиксированный размер вдоль главной оси в dp (если > 0)
	Fn      func()  // Функция отрисовки содержимого слота
}

type layoutBox struct {
	rect     render.Rect
	dir      layoutDir
	align    Align
	gap      int
	cursorX  int
	cursorY  int
	maxUsedX int
	maxUsedY int

	// Параметры для Grid
	gridCols    int
	gridGapX    int
	gridGapY    int
	gridRowH    int
	gridCellIdx int

	// Параметры для Flow
	flowGapX int
	flowGapY int
	flowRowH int

	// Параметры для FormLayout
	formLabelW int
	formGapX   int
	formGapY   int
}

func (c *Context) topLayout() *layoutBox {
	return &c.layoutStack[len(c.layoutStack)-1]
}

// CurrentArea возвращает прямоугольник текущего контейнера в физических пикселях.
func (c *Context) CurrentArea() render.Rect {
	return c.topLayout().rect
}

// RemainingWidth возвращает оставшуюся ширину в текущем Row (в физических пикселях).
func (c *Context) RemainingWidth() int {
	b := c.topLayout()
	rem := (b.rect.X + b.rect.W) - b.cursorX
	if rem < 0 {
		return 0
	}
	return rem
}

// RemainingHeight возвращает оставшуюся высоту в текущем Column (в физических пикселях).
func (c *Context) RemainingHeight() int {
	b := c.topLayout()
	rem := (b.rect.Y + b.rect.H) - b.cursorY
	if rem < 0 {
		return 0
	}
	return rem
}

// applySizeConstraints применяет ограничения размеров из WidgetStyle к базовым (w, h) в физических пикселях.
func (c *Context) applySizeConstraints(w, h int, st WidgetStyle) (int, int) {
	if st.FillWidth {
		w = c.RemainingWidth()
	}
	if st.FillHeight {
		h = c.RemainingHeight()
	}
	if st.MinWidthDp != nil {
		minW := c.S(*st.MinWidthDp)
		if w < minW {
			w = minW
		}
	}
	if st.MaxWidthDp != nil {
		maxW := c.S(*st.MaxWidthDp)
		if maxW > 0 && w > maxW {
			w = maxW
		}
	}
	if st.MinHeightDp != nil {
		minH := c.S(*st.MinHeightDp)
		if h < minH {
			h = minH
		}
	}
	if st.MaxHeightDp != nil {
		maxH := c.S(*st.MaxHeightDp)
		if maxH > 0 && h > maxH {
			h = maxH
		}
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return w, h
}

// allocRectStyled выделяет прямоугольник с учётом ограничений размеров из WidgetStyle.
func (c *Context) allocRectStyled(w, h int, st WidgetStyle) render.Rect {
	w, h = c.applySizeConstraints(w, h, st)
	return c.allocRect(w, h)
}

// allocRect выделяет прямоугольник размером (w, h) в физических пикселях внутри текущего контейнера.
func (c *Context) allocRect(w, h int) render.Rect {
	b := c.topLayout()
	var x, y int

	switch b.dir {
	case dirColumn:
		if b.align == AlignStretch && b.rect.W > 0 {
			w = b.rect.W
		}
		x = b.cursorX
		switch b.align {
		case AlignCenter:
			x = b.rect.X + (b.rect.W-w)/2
		case AlignEnd:
			x = b.rect.X + b.rect.W - w
		}
		y = b.cursorY
		b.cursorY += h + b.gap

	case dirRow:
		if b.align == AlignStretch && b.rect.H > 0 {
			h = b.rect.H
		}
		x = b.cursorX
		y = b.cursorY
		switch b.align {
		case AlignCenter:
			y = b.rect.Y + (b.rect.H-h)/2
		case AlignEnd:
			y = b.rect.Y + b.rect.H - h
		}
		b.cursorX += w + b.gap

	case dirGrid:
		cols := b.gridCols
		if cols < 1 {
			cols = 1
		}
		totalGapW := (cols - 1) * b.gridGapX
		cellW := (b.rect.W - totalGapW) / cols
		if cellW < 0 {
			cellW = 0
		}
		col := b.gridCellIdx % cols
		row := b.gridCellIdx / cols
		rowH := b.gridRowH
		if rowH <= 0 {
			rowH = h
		}
		x = b.rect.X + col*(cellW+b.gridGapX)
		y = b.rect.Y + row*(rowH+b.gridGapY)
		w = cellW
		h = rowH
		b.gridCellIdx++

	case dirFlow:
		if b.cursorX > b.rect.X && b.cursorX+w > b.rect.X+b.rect.W {
			b.cursorX = b.rect.X
			b.cursorY += b.flowRowH + b.flowGapY
			b.flowRowH = 0
		}
		x = b.cursorX
		y = b.cursorY
		b.cursorX += w + b.flowGapX
		if h > b.flowRowH {
			b.flowRowH = h
		}
	}

	if x+w > b.maxUsedX {
		b.maxUsedX = x + w
	}
	if y+h > b.maxUsedY {
		b.maxUsedY = y + h
	}

	return render.Rect{X: x, Y: y, W: w, H: h}
}

func (c *Context) pushBox(r render.Rect, dir layoutDir, align Align, gapPx int) {
	c.layoutStack = append(c.layoutStack, layoutBox{
		rect:     r,
		dir:      dir,
		align:    align,
		gap:      gapPx,
		cursorX:  r.X,
		cursorY:  r.Y,
		maxUsedX: r.X,
		maxUsedY: r.Y,
	})
}

func (c *Context) popBox() {
	if len(c.layoutStack) > 1 {
		c.layoutStack = c.layoutStack[:len(c.layoutStack)-1]
	}
}

// advanceParent сдвигает курсор родительского контейнера и обновляет maxUsedX/maxUsedY.
func advanceParent(b *layoutBox, usedW, usedH int) {
	if b.dir == dirColumn && usedH > 0 {
		if b.cursorY+usedH > b.maxUsedY {
			b.maxUsedY = b.cursorY + usedH
		}
		if b.cursorX+usedW > b.maxUsedX {
			b.maxUsedX = b.cursorX + usedW
		}
		b.cursorY += usedH + b.gap
	} else if b.dir == dirRow && usedW > 0 {
		if b.cursorX+usedW > b.maxUsedX {
			b.maxUsedX = b.cursorX + usedW
		}
		if b.cursorY+usedH > b.maxUsedY {
			b.maxUsedY = b.cursorY + usedH
		}
		b.cursorX += usedW + b.gap
	}
}

// Column располагает дочерние элементы вертикально с отступом gapDp.
func (c *Context) Column(gapDp int, fn func()) {
	b := c.topLayout()
	r := render.Rect{
		X: b.cursorX,
		Y: b.cursorY,
		W: c.RemainingWidth(),
		H: c.RemainingHeight(),
	}
	c.pushBox(r, dirColumn, AlignStart, c.S(gapDp))
	fn()
	inner := c.topLayout()
	usedW := inner.maxUsedX - r.X
	usedH := inner.maxUsedY - r.Y
	c.popBox()
	advanceParent(c.topLayout(), usedW, usedH)
}

// ColumnCentered располагает дочерние элементы вертикально и центрирует их по горизонтали.
func (c *Context) ColumnCentered(gapDp int, fn func()) {
	b := c.topLayout()
	r := render.Rect{
		X: b.rect.X,
		Y: b.cursorY,
		W: b.rect.W,
		H: c.RemainingHeight(),
	}
	c.pushBox(r, dirColumn, AlignCenter, c.S(gapDp))
	fn()
	inner := c.topLayout()
	usedW := inner.maxUsedX - r.X
	usedH := inner.maxUsedY - r.Y
	c.popBox()
	advanceParent(c.topLayout(), usedW, usedH)
}

// Row располагает дочерние элементы горизонтально с отступом gapDp.
func (c *Context) Row(gapDp int, heightDp int, fn func()) {
	h := c.S(heightDp)
	if heightDp <= 0 {
		h = c.RemainingHeight()
	}
	r := c.allocRect(c.RemainingWidth(), h)
	c.pushBox(r, dirRow, AlignStart, c.S(gapDp))
	fn()
	c.popBox()
}

// RowCentered располагает дочерние элементы горизонтально и выравнивает их по вертикальному центру.
func (c *Context) RowCentered(gapDp int, heightDp int, fn func()) {
	h := c.S(heightDp)
	if heightDp <= 0 {
		h = c.RemainingHeight()
	}
	r := c.allocRect(c.RemainingWidth(), h)
	c.pushBox(r, dirRow, AlignCenter, c.S(gapDp))
	fn()
	c.popBox()
}

// Padding добавляет внутренние отступы со всех сторон.
func (c *Context) Padding(padDp int, fn func()) {
	c.PaddingXY(padDp, padDp, fn)
}

// PaddingXY добавляет горизонтальный и вертикальный внутренние отступы.
func (c *Context) PaddingXY(pxDp, pyDp int, fn func()) {
	px := c.S(pxDp)
	py := c.S(pyDp)
	b := c.topLayout()
	r := render.Rect{
		X: b.cursorX + px,
		Y: b.cursorY + py,
		W: c.RemainingWidth() - px*2,
		H: c.RemainingHeight() - py*2,
	}
	if r.W < 0 {
		r.W = 0
	}
	if r.H < 0 {
		r.H = 0
	}
	c.pushBox(r, b.dir, b.align, b.gap)
	fn()
	inner := c.topLayout()
	usedH := inner.maxUsedY - r.Y
	usedW := inner.maxUsedX - r.X
	c.popBox()

	b = c.topLayout()
	if b.dir == dirColumn && usedH > 0 {
		if b.cursorY+usedH+py*2 > b.maxUsedY {
			b.maxUsedY = b.cursorY + usedH + py*2
		}
		b.cursorY += usedH + py*2
	} else if b.dir == dirRow && usedW > 0 {
		if b.cursorX+usedW+px*2 > b.maxUsedX {
			b.maxUsedX = b.cursorX + usedW + px*2
		}
		b.cursorX += usedW + px*2
	}
}

// Spacer сдвигает курсор на указанное количество логических пикселей.
func (c *Context) Spacer(dp int) {
	px := c.S(dp)
	b := c.topLayout()
	if b.dir == dirColumn {
		b.cursorY += px
	} else {
		b.cursorX += px
	}
}

// FlexSpacer заполняет оставшееся пространство (например, чтобы прижать статус-бар вниз).
func (c *Context) FlexSpacer(reserveDp int) {
	res := c.S(reserveDp)
	b := c.topLayout()
	if b.dir == dirColumn {
		rem := (b.rect.Y + b.rect.H) - b.cursorY - res
		if rem > 0 {
			b.cursorY += rem
		}
	} else {
		rem := (b.rect.X + b.rect.W) - b.cursorX - res
		if rem > 0 {
			b.cursorX += rem
		}
	}
}

// Expand выделяет дочернему блоку всё оставшееся пространство в текущем контейнере.
func (c *Context) Expand(fn func()) {
	w := c.RemainingWidth()
	h := c.RemainingHeight()
	r := c.allocRect(w, h)
	c.pushBox(r, dirColumn, AlignStart, 0)
	if fn != nil {
		fn()
	}
	c.popBox()
}

// calcFlexSegments делит totalSize между элементами с фиксированными размерами и пропорциональными весами.
func calcFlexSegments(totalSize, gapPx int, items []FlexItem, scale func(int) int) []int {
	n := len(items)
	sizes := make([]int, n)
	if n == 0 {
		return sizes
	}
	avail := totalSize - (n-1)*gapPx
	if avail < 0 {
		avail = 0
	}

	var totalWeight float32
	for i, it := range items {
		if it.FixedDp > 0 {
			sz := scale(it.FixedDp)
			if sz > avail {
				sz = avail
			}
			sizes[i] = sz
			avail -= sz
		} else {
			w := it.Weight
			if w <= 0 {
				w = 1
			}
			totalWeight += w
		}
	}

	if totalWeight > 0 && avail > 0 {
		rem := avail
		lastFlex := -1
		for i, it := range items {
			if it.FixedDp <= 0 {
				lastFlex = i
			}
		}
		for i, it := range items {
			if it.FixedDp > 0 {
				continue
			}
			w := it.Weight
			if w <= 0 {
				w = 1
			}
			if i == lastFlex {
				sizes[i] = rem
			} else {
				sz := int(float32(avail) * (w / totalWeight))
				if sz > rem {
					sz = rem
				}
				sizes[i] = sz
				rem -= sz
			}
		}
	}
	return sizes
}

// RowFlex располагает элементы горизонтально, деля ширину пропорционально весам FlexItem.
func (c *Context) RowFlex(gapDp, heightDp int, items ...FlexItem) {
	h := c.S(heightDp)
	if heightDp <= 0 {
		h = c.RemainingHeight()
	}
	gapPx := c.S(gapDp)
	totalW := c.RemainingWidth()
	rowRect := c.allocRect(totalW, h)
	segs := calcFlexSegments(totalW, gapPx, items, c.S)

	x := rowRect.X
	for i, it := range items {
		slotRect := render.Rect{X: x, Y: rowRect.Y, W: segs[i], H: rowRect.H}
		c.pushBox(slotRect, dirRow, AlignStart, 0)
		if it.Fn != nil {
			it.Fn()
		}
		c.popBox()
		x += segs[i] + gapPx
	}
}

// ColumnFlex располагает элементы вертикально, деля высоту пропорционально весам FlexItem.
func (c *Context) ColumnFlex(gapDp int, items ...FlexItem) {
	b := c.topLayout()
	gapPx := c.S(gapDp)
	totalW := c.RemainingWidth()
	totalH := c.RemainingHeight()
	colRect := render.Rect{X: b.cursorX, Y: b.cursorY, W: totalW, H: totalH}
	segs := calcFlexSegments(totalH, gapPx, items, c.S)

	y := colRect.Y
	for i, it := range items {
		slotRect := render.Rect{X: colRect.X, Y: y, W: colRect.W, H: segs[i]}
		c.pushBox(slotRect, dirColumn, AlignStart, 0)
		if it.Fn != nil {
			it.Fn()
		}
		c.popBox()
		y += segs[i] + gapPx
	}
	advanceParent(c.topLayout(), totalW, totalH)
}

// DockBottom резервирует bottomDp снизу для bottomFn (кнопки, статус-бар и т.д.),
// а всю оставшуюся высоту сверху отдаёт bodyFn.
func (c *Context) DockBottom(gapDp, bottomDp int, bodyFn, bottomFn func()) {
	c.ColumnFlex(gapDp,
		FlexItem{Weight: 1, Fn: bodyFn},
		FlexItem{FixedDp: bottomDp, Fn: bottomFn},
	)
}

// DockTop резервирует topDp сверху для topFn (шапка, тулбар, поиск),
// а всю оставшуюся высоту снизу отдаёт bodyFn.
func (c *Context) DockTop(gapDp, topDp int, topFn, bodyFn func()) {
	c.ColumnFlex(gapDp,
		FlexItem{FixedDp: topDp, Fn: topFn},
		FlexItem{Weight: 1, Fn: bodyFn},
	)
}

// PageConfig задаёт параметры типовой разметки страницы с шапкой, прокручиваемым телом и футером.
type PageConfig struct {
	ID       string // Идентификатор ScrollView тела (по умолчанию "page_body")
	GapDp    int    // Отступ между секциями в dp (по умолчанию 8)
	HeaderDp int    // Высота шапки в dp (по умолчанию 48, если headerFn != nil)
	FooterDp int    // Высота футера в dp (по умолчанию 44, если footerFn != nil)
	NoScroll bool   // Отключает автоматическую обёртку bodyFn в ScrollView
}

// Page строит типовой экран: опциональная шапка сверху, прокручиваемое тело по центру и опциональный футер снизу.
func (c *Context) Page(cfg PageConfig, headerFn, bodyFn, footerFn func()) {
	gap := cfg.GapDp
	if gap <= 0 {
		gap = 8
	}
	var items []FlexItem
	if headerFn != nil {
		hd := cfg.HeaderDp
		if hd <= 0 {
			hd = 48
		}
		items = append(items, FlexItem{FixedDp: hd, Fn: headerFn})
	}
	if bodyFn != nil {
		items = append(items, FlexItem{Weight: 1, Fn: func() {
			if cfg.NoScroll {
				bodyFn()
				return
			}
			svID := cfg.ID
			if svID == "" {
				svID = "page_body"
			}
			c.ScrollView(svID, 0, 0, bodyFn)
		}})
	}
	if footerFn != nil {
		ft := cfg.FooterDp
		if ft <= 0 {
			ft = 44
		}
		items = append(items, FlexItem{FixedDp: ft, Fn: footerFn})
	}
	if len(items) == 0 {
		return
	}
	c.ColumnFlex(gap, items...)
}

// calcJustifyOffsets вычисляет начальное смещение и массив шагов между элементами по главной оси.
func calcJustifyOffsets(justify Justify, totalSize, gapPx int, sizes []int) (int, []int) {
	n := len(sizes)
	gaps := make([]int, n)
	if n == 0 {
		return 0, gaps
	}
	sumSizes := 0
	for _, s := range sizes {
		sumSizes += s
	}
	free := totalSize - sumSizes
	if free < 0 {
		free = 0
	}

	switch justify {
	case JustifyCenter:
		content := sumSizes + (n-1)*gapPx
		start := (totalSize - content) / 2
		if start < 0 {
			start = 0
		}
		for i := range gaps {
			gaps[i] = gapPx
		}
		return start, gaps
	case JustifyEnd:
		content := sumSizes + (n-1)*gapPx
		start := totalSize - content
		if start < 0 {
			start = 0
		}
		for i := range gaps {
			gaps[i] = gapPx
		}
		return start, gaps
	case JustifySpaceBetween:
		if n > 1 {
			step := free / (n - 1)
			rem := free % (n - 1)
			for i := 0; i < n-1; i++ {
				gaps[i] = step
				if i < rem {
					gaps[i]++
				}
			}
		}
		return 0, gaps
	case JustifySpaceAround:
		slot := free / n
		start := slot / 2
		for i := 0; i < n-1; i++ {
			gaps[i] = slot
		}
		return start, gaps
	default: // JustifyStart
		for i := range gaps {
			gaps[i] = gapPx
		}
		return 0, gaps
	}
}

func (c *Context) nextJustifyKey(prefix string) string {
	c.layoutSeq++
	return prefix + strconv.Itoa(c.layoutSeq)
}

// RowWith располагает элементы горизонтально с заданным выравниванием Align и распределением Justify.
func (c *Context) RowWith(align Align, justify Justify, gapDp, heightDp int, items ...func()) {
	h := c.S(heightDp)
	if heightDp <= 0 {
		h = c.RemainingHeight()
	}
	gapPx := c.S(gapDp)
	r := c.allocRect(c.RemainingWidth(), h)

	if justify == JustifyStart || len(items) == 0 {
		c.pushBox(r, dirRow, align, gapPx)
		for _, fn := range items {
			if fn != nil {
				fn()
			}
		}
		c.popBox()
		return
	}

	key := c.nextJustifyKey("rw_")
	if c.justifySizes == nil {
		c.justifySizes = make(map[string][]int)
	}
	cached := c.justifySizes[key]
	hasCache := len(cached) == len(items)

	var startOff int
	var gaps []int
	if hasCache {
		startOff, gaps = calcJustifyOffsets(justify, r.W, gapPx, cached)
	} else {
		_, gaps = calcJustifyOffsets(JustifyStart, r.W, gapPx, make([]int, len(items)))
	}

	measured := make([]int, len(items))
	x := r.X + startOff
	for i, fn := range items {
		slotRect := render.Rect{X: x, Y: r.Y, W: r.X + r.W - x, H: r.H}
		if slotRect.W < 0 {
			slotRect.W = 0
		}
		c.pushBox(slotRect, dirRow, align, 0)
		if fn != nil {
			fn()
		}
		inner := c.topLayout()
		w := inner.maxUsedX - slotRect.X
		c.popBox()
		measured[i] = w
		x += w + gaps[i]
	}

	changed := !hasCache
	if !changed {
		for i := range measured {
			if measured[i] != cached[i] {
				changed = true
				break
			}
		}
	}
	if changed {
		c.justifySizes[key] = measured
		c.RequestRedraw()
	}
}

// ColumnWith располагает элементы вертикально с заданным выравниванием Align и распределением Justify.
func (c *Context) ColumnWith(align Align, justify Justify, gapDp int, items ...func()) {
	b := c.topLayout()
	gapPx := c.S(gapDp)
	r := render.Rect{
		X: b.cursorX,
		Y: b.cursorY,
		W: c.RemainingWidth(),
		H: c.RemainingHeight(),
	}

	if justify == JustifyStart || len(items) == 0 {
		c.pushBox(r, dirColumn, align, gapPx)
		for _, fn := range items {
			if fn != nil {
				fn()
			}
		}
		inner := c.topLayout()
		usedW := inner.maxUsedX - r.X
		usedH := inner.maxUsedY - r.Y
		c.popBox()
		advanceParent(c.topLayout(), usedW, usedH)
		return
	}

	key := c.nextJustifyKey("cw_")
	if c.justifySizes == nil {
		c.justifySizes = make(map[string][]int)
	}
	cached := c.justifySizes[key]
	hasCache := len(cached) == len(items)

	var startOff int
	var gaps []int
	if hasCache {
		startOff, gaps = calcJustifyOffsets(justify, r.H, gapPx, cached)
	} else {
		_, gaps = calcJustifyOffsets(JustifyStart, r.H, gapPx, make([]int, len(items)))
	}

	measured := make([]int, len(items))
	y := r.Y + startOff
	maxW := 0
	for i, fn := range items {
		slotRect := render.Rect{X: r.X, Y: y, W: r.W, H: r.Y + r.H - y}
		if slotRect.H < 0 {
			slotRect.H = 0
		}
		c.pushBox(slotRect, dirColumn, align, 0)
		if fn != nil {
			fn()
		}
		inner := c.topLayout()
		h := inner.maxUsedY - slotRect.Y
		if inner.maxUsedX-slotRect.X > maxW {
			maxW = inner.maxUsedX - slotRect.X
		}
		c.popBox()
		measured[i] = h
		y += h + gaps[i]
	}

	changed := !hasCache
	if !changed {
		for i := range measured {
			if measured[i] != cached[i] {
				changed = true
				break
			}
		}
	}
	if changed {
		c.justifySizes[key] = measured
		c.RequestRedraw()
	}
	advanceParent(c.topLayout(), maxW, r.H)
}

// Grid располагает дочерние элементы в виде сетки из cols колонок с отступами gapXDp, gapYDp и высотой строки rowHeightDp (0 = по содержимому).
func (c *Context) Grid(cols, gapXDp, gapYDp, rowHeightDp int, fn func()) {
	if cols < 1 {
		cols = 1
	}
	b := c.topLayout()
	r := render.Rect{
		X: b.cursorX,
		Y: b.cursorY,
		W: c.RemainingWidth(),
		H: c.RemainingHeight(),
	}
	c.layoutStack = append(c.layoutStack, layoutBox{
		rect:     r,
		dir:      dirGrid,
		align:    AlignStretch,
		cursorX:  r.X,
		cursorY:  r.Y,
		maxUsedX: r.X,
		maxUsedY: r.Y,
		gridCols: cols,
		gridGapX: c.S(gapXDp),
		gridGapY: c.S(gapYDp),
		gridRowH: c.S(rowHeightDp),
	})
	if fn != nil {
		fn()
	}
	inner := c.topLayout()
	usedW := inner.maxUsedX - r.X
	usedH := inner.maxUsedY - r.Y
	c.popBox()
	advanceParent(c.topLayout(), usedW, usedH)
}

// FormLayout создаёт табличную вёрстку формы «Подпись — Поле» с единой шириной колонки меток labelWidthDp.
func (c *Context) FormLayout(labelWidthDp, gapXDp, gapYDp int, fn func()) {
	b := c.topLayout()
	r := render.Rect{
		X: b.cursorX,
		Y: b.cursorY,
		W: c.RemainingWidth(),
		H: c.RemainingHeight(),
	}
	c.layoutStack = append(c.layoutStack, layoutBox{
		rect:       r,
		dir:        dirColumn,
		align:      AlignStart,
		gap:        c.S(gapYDp),
		cursorX:    r.X,
		cursorY:    r.Y,
		maxUsedX:   r.X,
		maxUsedY:   r.Y,
		formLabelW: c.S(labelWidthDp),
		formGapX:   c.S(gapXDp),
		formGapY:   c.S(gapYDp),
	})
	if fn != nil {
		fn()
	}
	inner := c.topLayout()
	usedW := inner.maxUsedX - r.X
	usedH := inner.maxUsedY - r.Y
	c.popBox()
	advanceParent(c.topLayout(), usedW, usedH)
}

// FormRow рисует строку формы с подписью слева (фиксированной ширины из FormLayout) и полем справа.
func (c *Context) FormRow(label string, fn func()) {
	b := c.topLayout()
	labelW := b.formLabelW
	if labelW <= 0 {
		labelW = c.S(140)
	}
	gapX := b.formGapX
	if gapX <= 0 {
		gapX = c.S(12)
	}

	rowW := c.RemainingWidth()
	rowRect := c.allocRect(rowW, c.S(36))

	if c.Font != nil && label != "" {
		pxSize := c.S(14)
		_, th := c.Font.MeasureText(label, pxSize)
		ly := rowRect.Y + (rowRect.H-th)/2
		c.Font.DrawText(c.Canvas, rowRect.X, ly, label, pxSize, c.Theme.TextMuted)
	}

	fieldX := rowRect.X + labelW + gapX
	fieldW := rowRect.W - labelW - gapX
	if fieldW < 0 {
		fieldW = 0
	}
	fieldRect := render.Rect{X: fieldX, Y: rowRect.Y, W: fieldW, H: rowRect.H}
	c.pushBox(fieldRect, dirRow, AlignCenter, c.S(8))
	if fn != nil {
		fn()
	}
	inner := c.topLayout()
	extra := 0
	if inner.maxUsedY > rowRect.Y+rowRect.H {
		extra = inner.maxUsedY - (rowRect.Y + rowRect.H)
	}
	maxY := inner.maxUsedY
	c.popBox()
	if extra > 0 {
		parent := c.topLayout()
		if parent.dir == dirColumn {
			parent.cursorY += extra
			if maxY > parent.maxUsedY {
				parent.maxUsedY = maxY
			}
		}
	}
}

// Flow располагает дочерние элементы слева направо с автоматическим переносом на новую строку при нехватке ширины.
func (c *Context) Flow(gapXDp, gapYDp int, fn func()) {
	b := c.topLayout()
	r := render.Rect{
		X: b.cursorX,
		Y: b.cursorY,
		W: c.RemainingWidth(),
		H: c.RemainingHeight(),
	}
	c.layoutStack = append(c.layoutStack, layoutBox{
		rect:     r,
		dir:      dirFlow,
		align:    AlignStart,
		cursorX:  r.X,
		cursorY:  r.Y,
		maxUsedX: r.X,
		maxUsedY: r.Y,
		flowGapX: c.S(gapXDp),
		flowGapY: c.S(gapYDp),
	})
	if fn != nil {
		fn()
	}
	inner := c.topLayout()
	usedW := inner.maxUsedX - r.X
	usedH := inner.maxUsedY - r.Y
	c.popBox()
	advanceParent(c.topLayout(), usedW, usedH)
}
