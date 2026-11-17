package ui

import (
	"testing"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

type dirtyMockWindow struct {
	mockWindow
	fullPresents int
	rectPresents int
	lastRect     render.Rect
	lastCursor   platform.CursorType
}

func (d *dirtyMockWindow) Present([]byte, int, int) {
	d.fullPresents++
}

func (d *dirtyMockWindow) PresentRect(_ []byte, _, _, rx, ry, rw, rh int) {
	d.rectPresents++
	d.lastRect = render.Rect{X: rx, Y: ry, W: rw, H: rh}
}

func (d *dirtyMockWindow) SetCursor(c platform.CursorType) {
	d.lastCursor = c
}

func TestSmartMouseMoveFiltering(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 400, h: 300, scale: 1.0}}
	ctx := NewContext(win, nil)

	draw := func(c *Context) {
		c.Padding(16, func() {
			c.ButtonOpts("btn1", "OK", Width(120), Height(36))
		})
	}

	// Первая отрисовка: кнопка оказывается в [16..136, 16..52]
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)

	// Движение мыши по пустому фону не должно вызывать перерисовку
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 300, Y: 200}) {
		t.Fatal("ожидался отказ от перерисовки при движении мыши по пустому фону")
	}
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 310, Y: 210}) {
		t.Fatal("ожидался отказ от перерисовки при повторном движении мыши по фону")
	}
	if ctx.SkippedMouseMoves() != 2 {
		t.Fatalf("ожидалось 2 отфильтрованных EventMouseMove, получено %d", ctx.SkippedMouseMoves())
	}

	// Вход курсора на кнопку должен вызвать перерисовку и сменить курсор на CursorHand
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 40, Y: 30}) {
		t.Fatal("ожидалась перерисовка при входе курсора на кнопку")
	}
	ctx.Render(draw)
	if win.lastCursor != platform.CursorHand {
		t.Fatalf("ожидался курсор CursorHand над кнопкой, получено %v", win.lastCursor)
	}

	// Дожидаемся завершения hover-анимации кнопки
	for i := 0; i < 20 && ctx.Animator.HasActive(); i++ {
		time.Sleep(20 * time.Millisecond)
		ctx.Render(draw)
	}
	ctx.stopAnimTimer()

	// Движение внутри той же кнопки без активных анимаций не должно вызывать перерисовку
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 70, Y: 35}) {
		t.Fatal("ожидался отказ от перерисовки при движении внутри одного HotID после завершения анимации")
	}

	// Выход курсора с кнопки должен вызвать перерисовку и вернуть CursorArrow
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 320, Y: 220}) {
		t.Fatal("ожидалась перерисовка при уходе курсора с кнопки")
	}
	ctx.Render(draw)
	if win.lastCursor != platform.CursorArrow {
		t.Fatalf("ожидался возврат курсора CursorArrow, получено %v", win.lastCursor)
	}

	// При зажатой ЛКМ (перетаскивание) фильтрация не блокирует перерисовку
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 320, Y: 220})
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 325, Y: 225}) {
		t.Fatal("ожидалась перерисовка при активном зажатии кнопки мыши")
	}
}

func TestSmartMouseMoveChartTrack(t *testing.T) {
	ctx := newTestContext(400, 300)
	draw := func(c *Context) {
		c.Sparkline("spk", []float64{10, 20, 15, 30}, Width(200), Height(60))
	}
	ctx.Render(draw)

	// На графике включён trackMove, поэтому движение внутри графика перерисовывает подсказку
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 25}) {
		t.Fatal("ожидалась перерисовка при входе на график")
	}
	ctx.Render(draw)
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 90, Y: 25}) {
		t.Fatal("ожидалась перерисовка при движении курсора внутри графика с тултипом")
	}
}

func TestStaticLayerAndDirtyRectangles(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 400, h: 300, scale: 1.0}}
	ctx := NewContext(win, nil)

	staticCalls := 0
	accentOn := false

	draw := func(c *Context) {
		c.Padding(10, func() {
			c.StaticLayer("hdr", func() {
				staticCalls++
				c.Canvas.FillRect(render.Rect{X: 10, Y: 10, W: 200, H: 40}, render.RGB(40, 80, 120))
				c.Spacer(48)
			})
			c.ButtonOpts("btn", "Action", Width(120), Height(36))
			if accentOn {
				c.Canvas.FillRect(render.Rect{X: 10, Y: 120, W: 50, H: 20}, render.RGB(255, 0, 0))
			}
		})
	}

	// Кадр 1: рисуется Static Layer и вызывается полный Present
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)
	if staticCalls != 1 || win.fullPresents != 1 {
		t.Fatalf("ожидались staticCalls=1 и fullPresents=1, получено static=%d full=%d", staticCalls, win.fullPresents)
	}
	if ctx.LastFrameStaticBlitted() {
		t.Fatal("первый кадр не должен был считаться блиттингом кеша")
	}

	// Кадр 2: без изменений (только Render) — статический блок взят из кеша, а Present пропущен
	ctx.Render(draw)
	if staticCalls != 1 {
		t.Fatalf("статический блок не должен был перерисовываться, staticCalls=%d", staticCalls)
	}
	if !ctx.LastFrameStaticBlitted() {
		t.Fatal("ожидался блиттинг статического слоя на втором кадре")
	}
	if win.fullPresents != 1 || win.rectPresents != 0 {
		t.Fatalf("при визуально идентичном кадре Present не должен вызываться, full=%d rect=%d", win.fullPresents, win.rectPresents)
	}

	// Кадр 3: меняется только динамический элемент -> StaticLayer взят из кеша, вызван PresentRect только для dirty rect
	accentOn = true
	ctx.Render(draw)
	if staticCalls != 1 {
		t.Fatalf("статический блок не должен перерисовываться при изменении динамического слоя, staticCalls=%d", staticCalls)
	}
	if win.rectPresents != 1 {
		t.Fatalf("ожидался 1 вызов PresentRect, получено %d", win.rectPresents)
	}
	if win.lastRect.W != 50 || win.lastRect.H != 20 {
		t.Fatalf("ожидался dirty rect 50x20, получено %+v", win.lastRect)
	}

	// Проверяем, что статический пиксель в (20, 20) остался сохранён после блиттинга
	off := (20*ctx.Canvas.Width + 20) * 4
	if ctx.Canvas.Pixels[off+2] != 40 || ctx.Canvas.Pixels[off+1] != 80 || ctx.Canvas.Pixels[off+0] != 120 {
		t.Fatalf("статический пиксель не был восстановлен из Static Layer")
	}

	// Кадр 4: InvalidateStatic или смена темы должны заставить перерисовать StaticLayer
	ctx.Theme = LightTheme()
	ctx.Render(draw)
	if staticCalls != 2 {
		t.Fatalf("после смены темы StaticLayer должен был перерисоваться, staticCalls=%d", staticCalls)
	}

	// Ресайз окна также инвалидирует StaticLayer
	ctx.HandleEvent(platform.Event{Type: platform.EventResize, Width: 500, Height: 350})
	ctx.Render(draw)
	if staticCalls != 3 {
		t.Fatalf("после Resize StaticLayer должен был перерисоваться, staticCalls=%d", staticCalls)
	}
}

func TestStaticLayerInRowCenteredAndScroll(t *testing.T) {
	ctx := newTestContext(500, 400)
	staticHeaderCalls := 0
	staticBodyCalls := 0
	clicked := false

	draw := func(c *Context) {
		c.RowCentered(10, 36, func() {
			c.StaticLayer("hdr_brand", func() {
				staticHeaderCalls++
				c.Badge("v0.7", Primary())
			})
			if c.ButtonOpts("dyn_btn", "Click") {
				clicked = true
			}
		})
		c.ScrollView("sv_perf", 400, 200, func() {
			c.StaticLayer("sv_static", func() {
				staticBodyCalls++
				c.CardOpts(func() {
					c.Spacer(40)
				}, Width(200), Height(60))
			})
		})
	}

	ctx.Render(draw)
	if staticHeaderCalls != 1 || staticBodyCalls != 1 {
		t.Fatalf("ожидались однократные вызовы StaticLayer, получено hdr=%d body=%d", staticHeaderCalls, staticBodyCalls)
	}

	// Повторная отрисовка: оба статических блока должны взяться из кеша
	ctx.Render(draw)
	if staticHeaderCalls != 1 || staticBodyCalls != 1 || !ctx.LastFrameStaticBlitted() {
		t.Fatalf("ожидался блиттинг обоих StaticLayer, hdr=%d body=%d blitted=%v", staticHeaderCalls, staticBodyCalls, ctx.LastFrameStaticBlitted())
	}

	// Клик по динамической кнопке справа от StaticLayer в RowCentered должен сработать и не перерисовать статику
	btnX := 0
	for _, reg := range ctx.hitRegions {
		if reg.id == "dyn_btn" {
			btnX = reg.rect.X + reg.rect.W/2
			break
		}
	}
	if btnX <= 0 {
		t.Fatal("кнопка dyn_btn не найдена в hitRegions")
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: btnX, Y: 18})
	ctx.Render(draw)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: btnX, Y: 18})
	ctx.Render(draw)
	if !clicked {
		t.Fatal("клик по кнопке после StaticLayer в RowCentered не сработал")
	}
	if staticHeaderCalls != 1 || staticBodyCalls != 1 {
		t.Fatalf("клик по кнопке не должен вызывать перерисовку StaticLayer, hdr=%d body=%d", staticHeaderCalls, staticBodyCalls)
	}
}

func TestHoverAreaCursorDoesNotLoopRedraws(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 400, h: 200, scale: 1.0}}
	ctx := NewContext(win, nil)

	zone := render.Rect{X: 20, Y: 20, W: 200, H: 60}
	draw := func(c *Context) {
		c.RegisterHit("hover_zone", zone, platform.CursorHand)
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)

	// Вход в зону вызывает перерисовку и устанавливает CursorHand
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 40}) {
		t.Fatal("ожидалась перерисовка при входе в hover_zone")
	}
	ctx.Render(draw)
	if win.lastCursor != platform.CursorHand {
		t.Fatalf("ожидался CursorHand в зоне hover_zone, получено %v", win.lastCursor)
	}

	// Последующее движение внутри зоны с trackMove=false не должно вызывать перерисовку
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 90, Y: 45}) {
		t.Fatal("движение внутри зоны trackMove=false не должно запускать Render")
	}
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 140, Y: 50}) {
		t.Fatal("повторное движение внутри зоны trackMove=false не должно запускать Render")
	}
}

func TestDebugDirtyRectDoesNotSelfInflate(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 500, h: 400, scale: 1.0}}
	ctx := NewContext(win, nil)
	ctx.SetDebugDirtyRect(true)

	toggle := false
	draw := func(c *Context) {
		if toggle {
			c.Canvas.FillRect(render.Rect{X: 40, Y: 40, W: 30, H: 20}, render.RGB(0, 200, 100))
		}
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)

	toggle = true
	ctx.Render(draw)
	rects := ctx.LastDirtyRects()
	if len(rects) != 1 || rects[0] != (render.Rect{X: 40, Y: 40, W: 30, H: 20}) {
		t.Fatalf("ожидался точный DirtyRect (40,40,30,20), получено %+v", rects)
	}

	// Повторный кадр без изменений: diff должен быть пустым, а не раздуваться из-за рамки
	ctx.Render(draw)
	if len(ctx.LastDirtyRects()) != 0 {
		t.Fatalf("без изменений diff должен быть пустым, получено %+v", ctx.LastDirtyRects())
	}
}
