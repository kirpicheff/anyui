package ui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

func TestOverlayStreamingCacheAndReset(t *testing.T) {
	ctx := newTestContext(400, 300)
	drawCount := 0

	draw := func(c *Context) {
		c.InteractiveCanvas("video_stream", CanvasCallbacks{
			Streaming: true,
			OnDraw: func(canvas *render.Canvas, area render.Rect, focused bool) {
				canvas.FillRect(area, render.RGB(10, 20, 30))
			},
		}, Width(200), Height(150))

		c.PushOverlay(render.Rect{X: 80, Y: 80, W: 100, H: 80}, func() {
			drawCount++
			c.Canvas.FillRect(render.Rect{X: 80, Y: 80, W: 100, H: 80}, render.RGBA(255, 0, 0, 200))
		})
	}

	// 1. Полный кадр с оверлеем и потоковым холстом (видео)
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)

	if drawCount != 1 {
		t.Fatalf("ожидался 1 вызов draw оверлея на полном кадре, получено %d", drawCount)
	}
	if len(ctx.Overlays.items) != 1 {
		t.Fatalf("ожидался 1 оверлей в Overlays, получено %d", len(ctx.Overlays.items))
	}
	cachedItem := ctx.Overlays.items[0]
	if cachedItem.cache == nil {
		t.Fatalf("ожидался заполненный кэш-буфер оверлея it.cache != nil")
	}

	// 2. wakeOnly-кадр (видеопоток обновляет кадр через EventWakeUp)
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	ctx.Render(draw)

	// drawCount НЕ должен увеличиться, так как оверлей наложился из кэша через CompositeFrom!
	if drawCount != 1 {
		t.Fatalf("на wakeOnly-кадре drawCount не должен увеличиваться (ожидался 1, получено %d)", drawCount)
	}

	// 3. Следующий полный кадр (например, EventMouseMove)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 10, Y: 10})
	ctx.Render(draw)

	if drawCount != 2 {
		t.Fatalf("на новом полном кадре draw оверлея должен вызваться снова (ожидался 2, получено %d)", drawCount)
	}
	// Проверяем, что кэш работает и при этом размер пула ограничен
	if len(ctx.Overlays.canvasPool) > 1 {
		t.Fatalf("ожидалось переиспользование холстов из пула, размер пула: %d", len(ctx.Overlays.canvasPool))
	}
}
