package ui

import "anyui/render"

type layoutDir uint8

const (
	dirColumn layoutDir = iota
	dirRow
)

// Align задаёт выравнивание элементов по поперечной оси.
type Align uint8

const (
	AlignStart Align = iota
	AlignCenter
)

type layoutBox struct {
	rect    render.Rect
	dir     layoutDir
	align   Align
	gap     int
	cursorX int
	cursorY int
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

// allocRect выделяет прямоугольник размером (w, h) в физических пикселях внутри текущего контейнера.
func (c *Context) allocRect(w, h int) render.Rect {
	b := c.topLayout()
	var x, y int

	if b.dir == dirColumn {
		x = b.cursorX
		if b.align == AlignCenter {
			x = b.rect.X + (b.rect.W-w)/2
		}
		y = b.cursorY
		b.cursorY += h + b.gap
	} else {
		x = b.cursorX
		y = b.cursorY
		if b.align == AlignCenter {
			y = b.rect.Y + (b.rect.H-h)/2
		}
		b.cursorX += w + b.gap
	}

	return render.Rect{X: x, Y: y, W: w, H: h}
}

func (c *Context) pushBox(r render.Rect, dir layoutDir, align Align, gapPx int) {
	c.layoutStack = append(c.layoutStack, layoutBox{
		rect:    r,
		dir:     dir,
		align:   align,
		gap:     gapPx,
		cursorX: r.X,
		cursorY: r.Y,
	})
}

func (c *Context) popBox() {
	if len(c.layoutStack) > 1 {
		c.layoutStack = c.layoutStack[:len(c.layoutStack)-1]
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
	usedH := c.topLayout().cursorY - r.Y
	c.popBox()
	if b.dir == dirColumn && usedH > 0 {
		b.cursorY += usedH
	}
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
	usedH := c.topLayout().cursorY - r.Y
	c.popBox()
	if b.dir == dirColumn && usedH > 0 {
		b.cursorY += usedH
	}
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
	usedH := inner.cursorY - r.Y
	usedW := inner.cursorX - r.X
	c.popBox()

	if b.dir == dirColumn && usedH > 0 {
		b.cursorY += usedH + py*2
	} else if b.dir == dirRow && usedW > 0 {
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
