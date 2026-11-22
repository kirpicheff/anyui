package ui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

type cursorTrackWindow struct {
	mockWindow
	cursor       platform.CursorType
	customCursor *platform.CustomCursor
	customCalls  int
}

func (w *cursorTrackWindow) SetCursor(c platform.CursorType) {
	w.cursor = c
}

func (w *cursorTrackWindow) SetCustomCursor(cur *platform.CustomCursor) {
	w.customCursor = cur
	w.customCalls++
}

func TestInteractiveCanvasCursorNone(t *testing.T) {
	win := &cursorTrackWindow{mockWindow: mockWindow{w: 400, h: 300, scale: 1.0}}
	ctx := NewContext(win, nil)

	draw := func(c *Context) {
		c.Button("btn", "Кнопка", false, 100)
		c.InteractiveCanvas("rdp", CanvasCallbacks{
			Cursor: platform.CursorNone,
		})
	}

	// Наводим мышь на InteractiveCanvas
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 200, Y: 150})
	ctx.Render(draw)
	if win.cursor != platform.CursorNone {
		t.Fatalf("ожидался CursorNone над InteractiveCanvas, получено %v", win.cursor)
	}

	// Уводим мышь на кнопку в верхней панели
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 20})
	ctx.Render(draw)
	if win.cursor != platform.CursorHand {
		t.Fatalf("ожидался CursorHand на кнопке тулбара, получено %v", win.cursor)
	}

	// Уводим мышь на пустой участок верхней строки (правее кнопки)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 300, Y: 20})
	ctx.Render(draw)
	if win.cursor != platform.CursorArrow {
		t.Fatalf("ожидался CursorArrow на пустом участке, получено %v", win.cursor)
	}
}

func TestInteractiveCanvasCustomCursor(t *testing.T) {
	win := &cursorTrackWindow{mockWindow: mockWindow{w: 400, h: 300, scale: 1.0}}
	ctx := NewContext(win, nil)

	remoteCur := &platform.CustomCursor{
		BGRA:   make([]byte, 16*16*4),
		Width:  16,
		Height: 16,
		HotX:   2,
		HotY:   3,
	}
	activeCur := remoteCur
	showOverlay := false

	draw := func(c *Context) {
		c.Button("btn", "Кнопка", false, 100)
		c.InteractiveCanvas("rdp", CanvasCallbacks{
			Cursor:       platform.CursorCrosshair,
			CustomCursor: activeCur,
			DirectInput:  true,
		})
		if showOverlay {
			c.PushOverlay(render.Rect{X: 100, Y: 80, W: 200, H: 150}, func() {})
		}
	}

	// Первый кадр: мышь вне холста (над пустым участком верхней строки)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 300, Y: 20})
	ctx.Render(draw)
	if win.customCursor != nil {
		t.Fatalf("не ожидался CustomCursor вне холста, получено %v", win.customCursor)
	}

	// Наводим мышь на холст -> должен установиться CustomCursor
	if need := ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 200, Y: 150}); !need {
		t.Fatal("вход мыши на InteractiveCanvas должен требовать Render")
	}
	ctx.Render(draw)
	if win.customCursor != remoteCur {
		t.Fatalf("ожидался remoteCur над InteractiveCanvas, получено %v", win.customCursor)
	}

	// Движение мыши внутри холста с DirectInput=true не должно перерисовывать UI и сбрасывать курсор
	callsBefore := win.customCalls
	if need := ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 210, Y: 160}); need {
		t.Fatal("движение внутри DirectInput холста не должно требовать Render")
	}
	if win.customCursor != remoteCur || win.customCalls != callsBefore {
		t.Fatal("движение внутри DirectInput холста не должно менять или переустанавливать CustomCursor")
	}

	// Открытие оверлея над точкой курсора должно сбросить CustomCursor
	showOverlay = true
	ctx.Render(draw)
	if win.customCursor != nil {
		t.Fatalf("ожидался сброс CustomCursor при перекрытии оверлеем, получено %v", win.customCursor)
	}
	showOverlay = false
	ctx.Render(draw)
	ctx.Render(draw)
	if win.customCursor != remoteCur {
		t.Fatalf("ожидалось восстановление remoteCur после закрытия оверлея, получено %v", win.customCursor)
	}

	// Сброс CustomCursor в nil при нахождении над холстом должен вернуть CursorCrosshair
	activeCur = nil
	ctx.Render(draw)
	if win.customCursor != nil {
		t.Fatalf("ожидался сброс CustomCursor при cb.CustomCursor == nil, получено %v", win.customCursor)
	}
	if win.cursor != platform.CursorCrosshair {
		t.Fatalf("ожидался системный CursorCrosshair после сброса CustomCursor, получено %v", win.cursor)
	}

	// Возвращаем CustomCursor и уводим мышь на кнопку тулбара
	activeCur = remoteCur
	ctx.Render(draw)
	if win.customCursor != remoteCur {
		t.Fatal("ожидался remoteCur перед выходом с холста")
	}
	if need := ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 20}); !need {
		t.Fatal("выход мыши с InteractiveCanvas должен требовать Render")
	}
	ctx.Render(draw)
	if win.customCursor != nil {
		t.Fatalf("ожидался сброс CustomCursor при уходе с холста, получено %v", win.customCursor)
	}
	if win.cursor != platform.CursorHand {
		t.Fatalf("ожидался CursorHand на кнопке тулбара, получено %v", win.cursor)
	}
}
