package ui

import (
	"testing"
	"time"

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

func pixelAt(cv *render.Canvas, x, y int) render.Color {
	off := (y*cv.Width + x) * 4
	return render.RGBA(cv.Pixels[off+2], cv.Pixels[off+1], cv.Pixels[off+0], cv.Pixels[off+3])
}

func TestOverlayTextInputSkipsFullBeginFrameAndPreservesStreaming(t *testing.T) {
	ctx := newTestContext(800, 600)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx.Clock = func() time.Time { return now }

	mainDrawCount := 0
	streamDrawCount := 0
	streamColor := render.RGB(20, 40, 60)
	modalOpen := true
	inputText := ""

	draw := func(c *Context) {
		mainDrawCount++
		c.InteractiveCanvas("video_stream", CanvasCallbacks{
			Streaming: true,
			OnDraw: func(canvas *render.Canvas, area render.Rect, focused bool) {
				streamDrawCount++
				canvas.FillRect(area, streamColor)
			},
		}, Width(800), Height(600))

		c.Modal("elevate_modal", &modalOpen, "Повышение прав", func() {
			c.TextInput("pwd_input", &inputText, "Пароль", 240)
			// Визуальный индикатор длины текста без зависимости от FontManager
			if len(inputText) > 0 {
				c.Canvas.FillRect(render.Rect{X: 260, Y: 280, W: len(inputText) * 10, H: 8}, render.RGB(0, 200, 120))
			}
		}, Width(320), Height(180))
	}

	// 1. Первый полный кадр открывает модалку
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)

	if mainDrawCount != 1 || streamDrawCount != 1 {
		t.Fatalf("ожидался 1 полный кадр, получено main=%d stream=%d", mainDrawCount, streamDrawCount)
	}

	// 2. Ставим фокус в TextInput внутри модалки, делаем полный кадр и завершаем focus-анимацию (0.14s)
	ctx.FocusID = "pwd_input"
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)
	for i := 0; i < 4; i++ {
		now = now.Add(50 * time.Millisecond)
		ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
		ctx.Render(draw)
	}

	baseMain := mainDrawCount
	baseStream := streamDrawCount

	// Фиксируем цвет пикселя вне модалки (фон видеопотока под полупрозрачным backdrop)
	bgBefore := pixelAt(ctx.Canvas, 10, 10)

	// 3. Вводим текст в TextInput внутри модалки (KeyDown + Char + KeyUp)
	for _, ch := range []rune{'a', 'b', 'c'} {
		ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: uint32(ch)})
		ctx.Render(draw)
		ctx.HandleEvent(platform.Event{Type: platform.EventChar, Char: ch})
		ctx.Render(draw)
		ctx.HandleEvent(platform.Event{Type: platform.EventKeyUp, KeyCode: uint32(ch)})
		ctx.Render(draw)
	}

	if inputText != "abc" {
		t.Fatalf("ожидался текст %q, получено %q", "abc", inputText)
	}
	if mainDrawCount != baseMain || streamDrawCount != baseStream {
		t.Fatalf("при вводе текста в оверлее не должен вызываться полный draw/stream: main %d->%d, stream %d->%d",
			baseMain, mainDrawCount, baseStream, streamDrawCount)
	}

	// Проверяем, что полупрозрачный backdrop не наслоился повторно
	bgAfter := pixelAt(ctx.Canvas, 10, 10)
	if bgAfter != bgBefore {
		t.Fatalf("пиксель под backdrop не должен меняться при локальной перерисовке оверлея: было %+v, стало %+v", bgBefore, bgAfter)
	}

	// Проверяем, что dirty-область локальна (не весь 800x600 экран)
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyBackspace})
	ctx.Render(draw)
	if inputText != "ab" {
		t.Fatalf("ожидался текст %q после Backspace, получено %q", "ab", inputText)
	}
	if mainDrawCount != baseMain || streamDrawCount != baseStream {
		t.Fatalf("после Backspace не должен вызываться полный draw/stream: main %d->%d, stream %d->%d",
			baseMain, mainDrawCount, baseStream, streamDrawCount)
	}
	dirty := ctx.LastDirtyRects()
	if len(dirty) != 1 || (dirty[0].W >= 800 && dirty[0].H >= 600) {
		t.Fatalf("ожидалась локальная dirty-область для TextInput, получено %+v", dirty)
	}

	// 4. Приходит новый видеокадр (EventWakeUp -> renderStreamingOnly), затем снова ввод символа
	streamColor = render.RGB(100, 150, 200)
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	ctx.Render(draw)
	bgStreamed := pixelAt(ctx.Canvas, 10, 10)
	if bgStreamed == bgBefore {
		t.Fatalf("ожидалось обновление видеопотока под backdrop на EventWakeUp")
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventChar, Char: 'z'})
	ctx.Render(draw)
	if inputText != "abz" {
		t.Fatalf("ожидался текст %q после обновления видеопотока, получено %q", "abz", inputText)
	}
	if mainDrawCount != baseMain {
		t.Fatalf("после обновления видеопотока ввод в оверлее не должен вызывать полный drawFn: main %d->%d", baseMain, mainDrawCount)
	}
	if got := pixelAt(ctx.Canvas, 10, 10); got != bgStreamed {
		t.Fatalf("локальная перерисовка оверлея должна сохранять актуальный видеокадр в underlay: ожидалось %+v, получено %+v", bgStreamed, got)
	}

	// 5. Нажатие Escape должно пойти в полный кадр и закрыть модалку
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEscape})
	ctx.Render(draw)
	if modalOpen {
		t.Fatalf("ожидалось закрытие модалки по KeyEscape")
	}
	if mainDrawCount <= baseMain {
		t.Fatalf("ожидался полный кадр при KeyEscape")
	}
}

func TestOverlayKeyInputWithWakeUpAndDirtyPresenter(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 800, h: 600, scale: 1.0}}
	ctx := NewContext(win, nil)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx.Clock = func() time.Time { return now }

	mainDrawCount := 0
	streamColor := render.RGB(20, 40, 60)
	inputText := ""
	panelRect := render.Rect{X: 500, Y: 100, W: 240, H: 180}

	draw := func(c *Context) {
		mainDrawCount++
		c.InteractiveCanvas("video_stream", CanvasCallbacks{
			Streaming: true,
			OnDraw: func(canvas *render.Canvas, area render.Rect, focused bool) {
				canvas.FillRect(area, streamColor)
			},
		}, Width(800), Height(600))

		c.PushOverlay(panelRect, func() {
			c.Canvas.FillRect(panelRect, render.RGB(30, 35, 45))
			c.LayoutRect(panelRect, func() {
				c.TextInput("chat_input", &inputText, "Сообщение", 200)
			})
		})
	}

	// Первый полный кадр и установка фокуса в оверлей
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)
	ctx.FocusID = "chat_input"
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)
	for i := 0; i < 4; i++ {
		now = now.Add(50 * time.Millisecond)
		ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
		ctx.Render(draw)
	}

	// Потоковый кадр выставляет prevPresentStale = true
	streamColor = render.RGB(40, 80, 120)
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	ctx.Render(draw)
	if !ctx.prevPresentStale {
		t.Fatal("ожидался prevPresentStale == true после стримингового кадра")
	}

	baseMain := mainDrawCount
	baseFull := win.fullPresents
	baseRect := win.rectPresents

	// EventChar + EventWakeUp до Render: EventWakeUp не должен сбрасывать keyInputOnlyFrame
	ctx.HandleEvent(platform.Event{Type: platform.EventChar, Char: 'x'})
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	ctx.Render(draw)

	if inputText != "x" {
		t.Fatalf("ожидался текст %q, получено %q", "x", inputText)
	}
	if mainDrawCount != baseMain {
		t.Fatalf("полный drawFn не должен вызываться при вводе в оверлее: %d -> %d", baseMain, mainDrawCount)
	}
	if win.fullPresents != baseFull {
		t.Fatalf("при DirtyPresenter не должно быть forceFull из-за prevPresentStale: full %d -> %d", baseFull, win.fullPresents)
	}
	if win.rectPresents != baseRect+1 {
		t.Fatalf("ожидался 1 вызов PresentRect, получено %d -> %d", baseRect, win.rectPresents)
	}
	if win.lastRect.W >= 800 && win.lastRect.H >= 600 {
		t.Fatalf("ожидался локальный bbox оверлея в PresentRect, получено %+v", win.lastRect)
	}

	// Обратный порядок: EventWakeUp, затем EventChar до Render
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	ctx.HandleEvent(platform.Event{Type: platform.EventChar, Char: 'y'})
	ctx.Render(draw)

	if inputText != "xy" {
		t.Fatalf("ожидался текст %q, получено %q", "xy", inputText)
	}
	if mainDrawCount != baseMain || win.fullPresents != baseFull || win.rectPresents != baseRect+2 {
		t.Fatalf("ожидался быстрый путь без forceFull: main=%d full=%d rect=%d", mainDrawCount, win.fullPresents, win.rectPresents)
	}
}

func TestStreamingEmptyDrawRedrawsOverlaysOnly(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 800, h: 600, scale: 1.0}}
	ctx := NewContext(win, nil)

	mainDrawCount := 0
	hasNewVideoFrame := true
	videoColor := render.RGB(10, 20, 30)
	termColor := render.RGB(200, 40, 40)
	panelRect := render.Rect{X: 480, Y: 120, W: 260, H: 200}

	draw := func(c *Context) {
		mainDrawCount++
		c.InteractiveCanvas("video_stream", CanvasCallbacks{
			Streaming: true,
			OnDraw: func(canvas *render.Canvas, area render.Rect, focused bool) {
				if hasNewVideoFrame {
					canvas.FillRect(area, videoColor)
				}
			},
		}, Width(800), Height(600))

		c.PushOverlay(panelRect, func() {
			c.Canvas.FillRect(panelRect, termColor)
		})
	}

	// 1. Полный кадр
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)
	if got := pixelAt(ctx.Canvas, 500, 140); got != termColor {
		t.Fatalf("ожидался цвет терминала %+v, получено %+v", termColor, got)
	}

	baseMain := mainDrawCount
	baseFull := win.fullPresents
	baseRect := win.rectPresents

	// 2. Проснулись от терминала/чата (видеокадр не менялся -> drawnRect.Empty())
	hasNewVideoFrame = false
	termColor = render.RGB(40, 210, 90)
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	ctx.Render(draw)

	if mainDrawCount != baseMain {
		t.Fatalf("при пробуждении от терминала полный drawFn не должен вызываться: %d -> %d", baseMain, mainDrawCount)
	}
	if win.fullPresents != baseFull {
		t.Fatalf("не должно быть полного Present: %d -> %d", baseFull, win.fullPresents)
	}
	if win.rectPresents != baseRect+1 {
		t.Fatalf("ожидался 1 вызов PresentRect для оверлея, получено %d -> %d", baseRect, win.rectPresents)
	}
	if win.lastRect.W >= 800 && win.lastRect.H >= 600 {
		t.Fatalf("в PresentRect должен уходить только прямоугольник оверлея, получено %+v", win.lastRect)
	}
	if got := pixelAt(ctx.Canvas, 500, 140); got != termColor {
		t.Fatalf("оверлей терминала должен перерисоваться на новый цвет %+v, получено %+v", termColor, got)
	}
	if got := pixelAt(ctx.Canvas, 20, 20); got != videoColor {
		t.Fatalf("фон видео вне оверлея не должен меняться: ожидалось %+v, получено %+v", videoColor, got)
	}

	// 3. Приходит новый видеокадр: кэш оверлея должен содержать обновлённый termColor
	hasNewVideoFrame = true
	videoColor = render.RGB(70, 90, 110)
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	ctx.Render(draw)

	if got := pixelAt(ctx.Canvas, 20, 20); got != videoColor {
		t.Fatalf("фон видео должен обновиться на %+v, получено %+v", videoColor, got)
	}
	if got := pixelAt(ctx.Canvas, 500, 140); got != termColor {
		t.Fatalf("кэш оверлея после обновления видео должен сохранять новый цвет терминала %+v, получено %+v", termColor, got)
	}
}
