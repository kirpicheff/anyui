package ui

import (
	"testing"

	"github.com/kirpicheff/anyui/render"
)

func TestRowFlexAndColumnFlex(t *testing.T) {
	ctx := newTestContext(420, 320)
	var r0, r1, r2 render.Rect
	var c0, c1 render.Rect

	ctx.Render(func(c *Context) {
		c.RowFlex(10, 40,
			FlexItem{FixedDp: 100, Fn: func() { r0 = c.CurrentArea() }},
			FlexItem{Weight: 1, Fn: func() { r1 = c.CurrentArea() }},
			FlexItem{Weight: 2, Fn: func() { r2 = c.CurrentArea() }},
		)
		// Оставшаяся высота: 320 - 40 = 280; минус gap=10 -> 270; 1:2 -> 90 и 180
		c.ColumnFlex(10,
			FlexItem{Weight: 1, Fn: func() { c0 = c.CurrentArea() }},
			FlexItem{Weight: 2, Fn: func() { c1 = c.CurrentArea() }},
		)
	})

	// Доступная ширина: 420 - 2*10 - 100 = 300 -> 100 и 200
	if r0.W != 100 || r1.W != 100 || r2.W != 200 {
		t.Fatalf("RowFlex widths mismatch: r0=%d r1=%d r2=%d", r0.W, r1.W, r2.W)
	}
	if r1.X != 110 || r2.X != 220 {
		t.Fatalf("RowFlex X positions mismatch: r1.X=%d r2.X=%d", r1.X, r2.X)
	}
	if c0.H != 90 || c1.H != 180 {
		t.Fatalf("ColumnFlex heights mismatch: c0=%d c1=%d", c0.H, c1.H)
	}
}

func TestGridAndFlowLayout(t *testing.T) {
	ctx := newTestContext(320, 400)
	var cells [4]render.Rect
	var flowRects [3]render.Rect
	var afterY int

	ctx.Render(func(c *Context) {
		c.Grid(2, 10, 10, 40, func() {
			for i := 0; i < 4; i++ {
				cells[i] = c.allocRect(10, 10)
			}
		})
		// Grid занял 2 строки по 40 + gap 10 = 90
		c.Flow(10, 10, func() {
			flowRects[0] = c.allocRect(150, 30)
			flowRects[1] = c.allocRect(150, 30)
			flowRects[2] = c.allocRect(150, 30) // не влезает в 320 (150+10+150+10+150 > 320) -> перенос
		})
		afterY = c.topLayout().cursorY
	})

	// Ячейки Grid: (320 - 10)/2 = 155
	if cells[0] != (render.Rect{X: 0, Y: 0, W: 155, H: 40}) {
		t.Fatalf("Grid cell[0] = %+v", cells[0])
	}
	if cells[1] != (render.Rect{X: 165, Y: 0, W: 155, H: 40}) {
		t.Fatalf("Grid cell[1] = %+v", cells[1])
	}
	if cells[2] != (render.Rect{X: 0, Y: 50, W: 155, H: 40}) {
		t.Fatalf("Grid cell[2] = %+v", cells[2])
	}
	if cells[3] != (render.Rect{X: 165, Y: 50, W: 155, H: 40}) {
		t.Fatalf("Grid cell[3] = %+v", cells[3])
	}

	// Flow начинается с y=90
	if flowRects[0].Y != 90 || flowRects[1].Y != 90 || flowRects[1].X != 160 {
		t.Fatalf("Flow first row mismatch: r0=%+v r1=%+v", flowRects[0], flowRects[1])
	}
	if flowRects[2].X != 0 || flowRects[2].Y != 130 {
		t.Fatalf("Flow wrapped item mismatch: r2=%+v", flowRects[2])
	}
	if afterY != 160 {
		t.Fatalf("expected cursorY=160 after Grid+Flow, got %d", afterY)
	}
}

func TestAlignAndJustifySpaceBetween(t *testing.T) {
	ctx := newTestContext(300, 200)
	var a, b, d render.Rect

	draw := func(c *Context) {
		c.RowWith(AlignEnd, JustifySpaceBetween, 0, 100,
			func() { a = c.allocRect(50, 20) },
			func() { b = c.allocRect(50, 40) },
			func() { d = c.allocRect(50, 30) },
		)
	}

	// Render автоматически выполняет 2-й проход при needsRedraw, поэтому SpaceBetween сходится сразу
	ctx.Render(draw)

	if a.X != 0 || b.X != 125 || d.X != 250 {
		t.Fatalf("SpaceBetween X mismatch: a=%d b=%d d=%d", a.X, b.X, d.X)
	}
	if a.Y != 80 || b.Y != 60 || d.Y != 70 {
		t.Fatalf("AlignEnd Y mismatch: a=%d b=%d d=%d", a.Y, b.Y, d.Y)
	}
}

func TestSizeConstraintsAndFormLayout(t *testing.T) {
	ctx := newTestContext(400, 300)
	var rMin, rMax, rFill, rFormField render.Rect

	ctx.Render(func(c *Context) {
		rMin = c.allocRectStyled(20, 10, ApplyOptions([]Option{MinWidth(80), MinHeight(30)}))
		rMax = c.allocRectStyled(300, 100, ApplyOptions([]Option{MaxWidth(120), MaxHeight(50)}))
		rFill = c.allocRectStyled(10, 20, ApplyOptions([]Option{FillWidth()}))
		c.FormLayout(120, 10, 8, func() {
			c.FormRow("Имя:", func() {
				rFormField = c.allocRectStyled(10, 28, ApplyOptions([]Option{FillWidth()}))
			})
		})
	})

	if rMin.W != 80 || rMin.H != 30 {
		t.Fatalf("Min constraints failed: %+v", rMin)
	}
	if rMax.W != 120 || rMax.H != 50 {
		t.Fatalf("Max constraints failed: %+v", rMax)
	}
	if rFill.W != 400 {
		t.Fatalf("FillWidth failed: got %d, want 400", rFill.W)
	}
	if rFormField.X != 130 || rFormField.W != 270 {
		t.Fatalf("FormRow field rect mismatch: %+v", rFormField)
	}
}

func TestCardAutoHeight(t *testing.T) {
	ctx := newTestContext(300, 400)
	var afterCardY int

	draw := func(c *Context) {
		c.CardOpts(func() {
			c.allocRect(100, 30)
			c.allocRect(100, 20)
		}, Padding(10), AutoHeight())
		afterCardY = c.topLayout().cursorY
	}

	// Render автоматически выполняет 2-й проход при needsRedraw: pad(10)*2 + 30 + gap(8) + 20 = 78
	ctx.Render(draw)
	if afterCardY != 78 {
		t.Fatalf("expected Card AutoHeight=78, got %d", afterCardY)
	}
}
