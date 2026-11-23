package anyui

import (
	"path/filepath"
	"testing"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
	"github.com/kirpicheff/anyui/ui"
)

type stylerMockWindow struct {
	w, h        int
	scale       float32
	darkTitle   bool
	darkChanged int
	captionCol  *platform.WindowColor
	textCol     *platform.WindowColor
	borderCol   *platform.WindowColor
	brought     int
	wakeUps     int
	presents        int
	presentRects    int
	lastPresentRect render.Rect
	alwaysOnTop     bool
}

func (m *stylerMockWindow) Run(func(platform.Event)) error  { return nil }
func (m *stylerMockWindow) Open(func(platform.Event)) error { return nil }
func (m *stylerMockWindow) Present([]byte, int, int)        { m.presents++ }
func (m *stylerMockWindow) PresentRect(_ []byte, _, _, x, y, w, h int) {
	m.presentRects++
	m.lastPresentRect = render.Rect{X: x, Y: y, W: w, H: h}
}
func (m *stylerMockWindow) Invalidate()                             {}
func (m *stylerMockWindow) WakeUp()                                 { m.wakeUps++ }
func (m *stylerMockWindow) Show()                                   {}
func (m *stylerMockWindow) Hide()                                   {}
func (m *stylerMockWindow) Close()                                  {}
func (m *stylerMockWindow) SetTitle(string)                         {}
func (m *stylerMockWindow) SetCursor(platform.CursorType)        {}
func (m *stylerMockWindow) SetCustomCursor(*platform.CustomCursor) {}
func (m *stylerMockWindow) Size() (int, int)                        { return m.w, m.h }
func (m *stylerMockWindow) Scale() float32                          { return m.scale }
func (m *stylerMockWindow) ClipboardText() (string, error)          { return "", nil }
func (m *stylerMockWindow) SetClipboardText(string) error           { return nil }
func (m *stylerMockWindow) Minimize()                               {}
func (m *stylerMockWindow) Maximize()                               {}
func (m *stylerMockWindow) Restore()                                {}
func (m *stylerMockWindow) IsMaximized() bool                       { return false }
func (m *stylerMockWindow) IsMinimized() bool                       { return false }
func (m *stylerMockWindow) SetFullscreen(bool)                      {}
func (m *stylerMockWindow) IsFullscreen() bool                      { return false }
func (m *stylerMockWindow) Position() (int, int)                    { return 0, 0 }
func (m *stylerMockWindow) SetPosition(int, int)                    {}
func (m *stylerMockWindow) SetSize(int, int)                        {}
func (m *stylerMockWindow) SetOnClose(func() bool)                  {}
func (m *stylerMockWindow) NativeHandle() uintptr                   { return 0 }
func (m *stylerMockWindow) SetCapture()                             {}
func (m *stylerMockWindow) ReleaseCapture()                         {}
func (m *stylerMockWindow) DoubleClickTimeMs() int                  { return 500 }
func (m *stylerMockWindow) SetDarkTitle(dark bool)                  { m.darkTitle = dark; m.darkChanged++ }
func (m *stylerMockWindow) BringToFront()                           { m.brought++ }
func (m *stylerMockWindow) SetAlwaysOnTop(onTop bool)               { m.alwaysOnTop = onTop }
func (m *stylerMockWindow) IsAlwaysOnTop() bool                     { return m.alwaysOnTop }
func (m *stylerMockWindow) SetTitleBarColors(c, t, b *platform.WindowColor) {
	m.captionCol = c
	m.textCol = t
	m.borderCol = b
}

func TestWindowAlwaysOnTop(t *testing.T) {
	mw := &stylerMockWindow{w: 800, h: 600, scale: 1.0}
	w := &Window{platWin: mw}
	if w.IsAlwaysOnTop() {
		t.Fatal("по умолчанию IsAlwaysOnTop() должен быть false")
	}
	w.SetAlwaysOnTop(true)
	if !w.IsAlwaysOnTop() || !mw.alwaysOnTop {
		t.Fatal("ожидалось IsAlwaysOnTop() == true после SetAlwaysOnTop(true)")
	}
	w.SetAlwaysOnTop(false)
	if w.IsAlwaysOnTop() || mw.alwaysOnTop {
		t.Fatal("ожидалось IsAlwaysOnTop() == false после SetAlwaysOnTop(false)")
	}
}

func TestSaveAndLoadWindowState(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nested", "window_state.json")
	orig := WindowState{X: 120, Y: 80, Width: 1024, Height: 700, Maximized: true}
	if err := SaveWindowState(p, orig); err != nil {
		t.Fatalf("SaveWindowState: %v", err)
	}
	got, err := LoadWindowState(p)
	if err != nil {
		t.Fatalf("LoadWindowState: %v", err)
	}
	if got != orig {
		t.Fatalf("ожидалось %+v, получено %+v", orig, got)
	}
}

func TestWindowSetThemeUpdatesDarkTitleAndColors(t *testing.T) {
	mw := &stylerMockWindow{w: 800, h: 600, scale: 1.0}
	ctx := ui.NewContext(mw, nil)
	win := &Window{platWin: mw, ctx: ctx}

	win.SetTheme(LightTheme())
	if mw.darkTitle {
		t.Fatal("для LightTheme должен устанавливаться светлый системный заголовок")
	}
	win.SetTheme(NavyTheme())
	if !mw.darkTitle {
		t.Fatal("для NavyTheme должен устанавливаться тёмный системный заголовок")
	}

	c := RGB(15, 23, 42)
	tx := RGB(248, 250, 252)
	br := RGB(59, 130, 246)
	win.SetTitleBarColors(&c, &tx, &br)
	if mw.captionCol == nil || mw.captionCol.R != 15 || mw.textCol == nil || mw.borderCol == nil || mw.borderCol.B != 246 {
		t.Fatal("цвета рамки и шапки не были переданы в платформенный слой")
	}
}

func TestHandleSystemThemeChange(t *testing.T) {
	var gotCallback bool
	var gotDark bool
	app := NewApp(Config{
		FollowSystemTheme: true,
		OnSystemThemeChange: func(_ *App, dark bool) {
			gotCallback = true
			gotDark = dark
		},
	})
	mw := &stylerMockWindow{w: 800, h: 600, scale: 1.0}
	ctx := ui.NewContext(mw, nil)
	win := &Window{app: app, platWin: mw, ctx: ctx}
	app.windows = []*Window{win}
	app.mainWin = win

	win.handleEvent(platform.Event{Type: platform.EventThemeChange, DarkTheme: false})
	if !gotCallback || gotDark {
		t.Fatalf("коллбэк смены темы сработал некорректно: called=%v dark=%v", gotCallback, gotDark)
	}
	if win.ctx.Theme != LightTheme() {
		t.Fatal("при FollowSystemTheme=true и DarkTheme=false ожидалась LightTheme")
	}

	win.handleEvent(platform.Event{Type: platform.EventThemeChange, DarkTheme: true})
	if !gotDark || win.ctx.Theme != DarkTheme() || !mw.darkTitle {
		t.Fatal("при DarkTheme=true ожидалась DarkTheme и тёмный заголовок окна")
	}
}

func TestThemeSwitchingAndStaticLayer(t *testing.T) {
	mw := &stylerMockWindow{w: 900, h: 300, scale: 1.0}
	ctx := ui.NewContext(mw, nil)
	ctx.Theme = NavyTheme()

	staticCalls := 0
	var win *Window
	drawFn := func(u *UI) {
		th := u.Theme()
		u.PaddingXY(16, 10, func() {
			u.RowCentered(10, 36, func() {
				u.StaticLayer("hdr", func() {
					staticCalls++
					u.Canvas().FillRect(Rect{X: 16, Y: 10, W: 80, H: 24}, th.Card)
					u.Spacer(80)
				})
				u.FlexSpacer(100)
				if u.Button("th_light", "Light", Ghost(), Width(80), Height(32)) {
					u.SetTheme(LightTheme())
				}
				if u.Button("th_dark", "Dark", Ghost(), Width(80), Height(32)) {
					u.SetTheme(DarkTheme())
				}
				if u.Button("th_navy", "Navy", Ghost(), Width(90), Height(32)) {
					u.SetTheme(NavyTheme())
				}
			})
		})
	}
	win = &Window{
		platWin: mw,
		ctx:     ctx,
		drawFn:  drawFn,
	}
	win.ui = &UI{ctx: ctx, win: win}

	// Стартовая отрисовка в Navy
	win.handleEvent(platform.Event{Type: platform.EventPaint})
	if ctx.Theme != NavyTheme() || staticCalls != 1 {
		t.Fatalf("некорректное начальное состояние темы: staticCalls=%d", staticCalls)
	}
	bgNavy := ctx.Canvas.Pixels[0:4] // BGRA в точке (0,0)
	if bgNavy[2] != NavyTheme().Bg.R || bgNavy[1] != NavyTheme().Bg.G || bgNavy[0] != NavyTheme().Bg.B {
		t.Fatalf("фон Navy не совпал: got RGB(%d,%d,%d)", bgNavy[2], bgNavy[1], bgNavy[0])
	}

	clickAt := func(x, y int) {
		win.handleEvent(platform.Event{Type: platform.EventMouseMove, X: x, Y: y})
		win.handleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: x, Y: y})
		win.handleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: x, Y: y})
	}

	// Клик по кнопке Light (в правом углу строки: X около 580..660, Y=26; проверим диапазон)
	// RowCentered: Width=868, FlexSpacer(100) оставляет 100px справа, плюс кнопки 80+10+80+10+90=270:
	// проще всего проверить координаты через ctx: 16 + 80 + (868 - 80 - 100) = 784 -> Light в [784..864]
	// Используем фактический поиск центра пикселя кнопки не хардкодя, а проверив переключение на Light -> Dark -> DeepNavy:
	// Кнопка Light первая после FlexSpacer: X = 16 + (868 - 100) = 784? Нет, FlexSpacer ставит cursorX = 16+868-100 = 784, затем th_light в [784..864].
	clickAt(820, 26)
	if ctx.Theme != LightTheme() {
		t.Fatalf("после клика по th_light ожидалась LightTheme, получено %+v", ctx.Theme.Bg)
	}
	if mw.darkTitle {
		t.Fatal("после переключения на LightTheme заголовок ОС должен стать светлым")
	}
	bgLight := ctx.Canvas.Pixels[0:4]
	if bgLight[2] != LightTheme().Bg.R || bgLight[1] != LightTheme().Bg.G || bgLight[0] != LightTheme().Bg.B {
		t.Fatalf("фон LightTheme не перерисовался: got RGB(%d,%d,%d)", bgLight[2], bgLight[1], bgLight[0])
	}

	// Переключение через UI.SetTheme на Dark и обратно на DeepNavy
	win.ui.SetTheme(DarkTheme())
	win.handleEvent(platform.Event{Type: platform.EventPaint})
	if ctx.Theme != DarkTheme() || !mw.darkTitle {
		t.Fatal("ожидалась DarkTheme и тёмный заголовок ОС")
	}
	bgDark := ctx.Canvas.Pixels[0:4]
	if bgDark[2] != DarkTheme().Bg.R || bgDark[1] != DarkTheme().Bg.G || bgDark[0] != DarkTheme().Bg.B {
		t.Fatalf("фон DarkTheme не перерисовался: got RGB(%d,%d,%d)", bgDark[2], bgDark[1], bgDark[0])
	}
}

func TestMultiWindowDispatchMouseMoveDirectInputNoLostWakeup(t *testing.T) {
	app := NewApp(Config{})

	mw1 := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx1 := ui.NewContext(mw1, nil)
	w1Renders := 0
	w1 := &Window{
		app:     app,
		platWin: mw1,
		ctx:     ctx1,
		drawFn: func(u *UI) {
			w1Renders++
		},
	}
	w1.ui = &UI{ctx: ctx1, win: w1}

	mw2 := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx2 := ui.NewContext(mw2, nil)
	w2Renders := 0
	w2CanvasDraws := 0
	frameReady := false
	w2 := &Window{
		app:     app,
		platWin: mw2,
		ctx:     ctx2,
		drawFn: func(u *UI) {
			w2Renders++
			u.ctx.InteractiveCanvas("viewer", ui.CanvasCallbacks{
				DirectInput: true,
				Streaming:   true,
				OnDraw: func(cv *render.Canvas, area render.Rect, _ bool) {
					w2CanvasDraws++
					if frameReady {
						cv.FillRect(render.Rect{X: area.X, Y: area.Y, W: 50, H: 50}, render.RGB(255, 0, 0))
					}
				},
			}, ui.Width(200), ui.Height(150))
		},
	}
	w2.ui = &UI{ctx: ctx2, win: w2}

	app.windows = []*Window{w1, w2}
	app.mainWin = w1

	// Начальная отрисовка обоих окон
	w1.handleEvent(platform.Event{Type: platform.EventPaint})
	w2.handleEvent(platform.Event{Type: platform.EventPaint})

	mw1.wakeUps, mw2.wakeUps = 0, 0
	mw1.presents, mw2.presents = 0, 0
	mw1.presentRects, mw2.presentRects = 0, 0
	w1Renders, w2Renders, w2CanvasDraws = 0, 0, 0

	// Фоновая горутина ставит задачу в Dispatch — WakeUp уходит в a.windows[0] (w1)
	dispatchedExecuted := false
	app.Dispatch(func() {
		dispatchedExecuted = true
		frameReady = true
	})
	if mw1.wakeUps != 1 || mw2.wakeUps != 0 {
		t.Fatalf("ожидалось пробуждение только w1: w1=%d w2=%d", mw1.wakeUps, mw2.wakeUps)
	}

	// До обработки WakeUp в w1 во второе окно w2 приходит MouseMove над DirectInput-холстом
	mw1.wakeUps = 0
	w2.handleEvent(platform.Event{Type: platform.EventMouseMove, X: 20, Y: 20})

	if !dispatchedExecuted {
		t.Fatal("ожидалось выполнение коллбека из dispatchQueue в w2.handleEvent")
	}
	if mw1.wakeUps != 1 {
		t.Fatalf("ожидался invalidateAllExcept(w2) для w1: got %d", mw1.wakeUps)
	}
	if mw2.wakeUps != 1 {
		t.Fatalf("потеряно пробуждение для w2 при HandleEvent(MouseMove)==false: got wakeUps=%d", mw2.wakeUps)
	}
	if w2Renders != 0 {
		t.Fatalf("DirectInput MouseMove не должен вызывать синхронный Render: got %d", w2Renders)
	}

	// При последующем EventWakeUp окно w2 должно перерисоваться с fast-path Streaming (OnDraw + PresentRect без полного drawFn)
	w2.handleEvent(platform.Event{Type: platform.EventWakeUp})
	if w2Renders != 0 {
		t.Fatalf("при wakeOnly + Streaming полный drawFn должен пропускаться: got w2Renders=%d", w2Renders)
	}
	if w2CanvasDraws != 1 {
		t.Fatalf("ожидался вызов OnDraw в w2 по EventWakeUp: got %d", w2CanvasDraws)
	}
	if mw2.presentRects != 1 || mw2.presents != 0 {
		t.Fatalf("ожидался fast-path PresentRect для Streaming=true: presentRects=%d presents=%d", mw2.presentRects, mw2.presents)
	}
}

func TestWindowWakeUpTargeted(t *testing.T) {
	app := NewApp(Config{})

	mw1 := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx1 := ui.NewContext(mw1, nil)
	w1Renders := 0
	w1 := &Window{
		app:     app,
		platWin: mw1,
		ctx:     ctx1,
		drawFn: func(u *UI) {
			w1Renders++
		},
	}
	w1.ui = &UI{ctx: ctx1, win: w1}

	mw2 := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx2 := ui.NewContext(mw2, nil)
	w2Renders := 0
	w2CanvasDraws := 0
	w2 := &Window{
		app:     app,
		platWin: mw2,
		ctx:     ctx2,
		drawFn: func(u *UI) {
			w2Renders++
			u.ctx.InteractiveCanvas("viewer", ui.CanvasCallbacks{
				DirectInput: true,
				Streaming:   true,
				OnDraw: func(cv *render.Canvas, area render.Rect, _ bool) {
					w2CanvasDraws++
					cv.FillRect(render.Rect{X: area.X, Y: area.Y, W: 40, H: 40}, render.RGB(0, 255, 0))
				},
			}, ui.Width(200), ui.Height(150))
		},
	}
	w2.ui = &UI{ctx: ctx2, win: w2}

	app.windows = []*Window{w1, w2}
	app.mainWin = w1

	w1.handleEvent(platform.Event{Type: platform.EventPaint})
	w2.handleEvent(platform.Event{Type: platform.EventPaint})

	mw1.wakeUps, mw2.wakeUps = 0, 0
	mw1.presents, mw2.presents = 0, 0
	mw1.presentRects, mw2.presentRects = 0, 0
	w1Renders, w2Renders, w2CanvasDraws = 0, 0, 0

	// Адресное пробуждение только w2
	w2.WakeUp()
	if mw2.wakeUps != 1 || mw1.wakeUps != 0 {
		t.Fatalf("w2.WakeUp() должен будить только w2: w1=%d w2=%d", mw1.wakeUps, mw2.wakeUps)
	}

	w2.handleEvent(platform.Event{Type: platform.EventWakeUp})
	if w1Renders != 0 || w2Renders != 0 || w2CanvasDraws != 1 {
		t.Fatalf("ожидался fast-path OnDraw только у w2 без полного drawFn: w1=%d w2=%d canvas=%d", w1Renders, w2Renders, w2CanvasDraws)
	}
	if mw2.presentRects != 1 || mw1.presents != 0 || mw1.presentRects != 0 {
		t.Fatalf("ожидался fast-path PresentRect только у w2: w2.presentRects=%d w1.presents=%d", mw2.presentRects, mw1.presents)
	}
}

func TestAppInitFontManager(t *testing.T) {
	app := NewApp(Config{})
	fm, err := app.initFontManager()
	if err != nil {
		t.Skipf("системный шрифт недоступен в тестовой среде: %v", err)
	}
	if fm == nil {
		t.Fatal("ожидался непустой FontManager")
	}

	appBad := NewApp(Config{
		FontRegular: []byte("not-a-ttf-font"),
	})
	if _, err := appBad.initFontManager(); err == nil {
		t.Fatal("ожидалась ошибка при передаче невалидных байтов FontRegular")
	}
}

func TestUIMousePressedAndMouseDown(t *testing.T) {
	mockWin := &stylerMockWindow{w: 300, h: 200, scale: 1.0}
	ctx := ui.NewContext(mockWin, nil)
	u := &UI{ctx: ctx}

	ctx.MousePressed = true
	ctx.MouseDown = true
	if !u.MousePressed() || !u.MouseDown() {
		t.Fatalf("ожидалось true для MousePressed и MouseDown")
	}

	ctx.MousePressed = false
	ctx.MouseDown = false
	if u.MousePressed() || u.MouseDown() {
		t.Fatalf("ожидалось false для MousePressed и MouseDown")
	}
}

func TestStreamingWakeUpWithNonIntersectingOverlay(t *testing.T) {
	app := NewApp(Config{})
	mw := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx := ui.NewContext(mw, nil)

	renders := 0
	canvasDraws := 0
	overlayDraws := 0

	w := &Window{
		app:     app,
		platWin: mw,
		ctx:     ctx,
		drawFn: func(u *UI) {
			renders++
			u.ctx.InteractiveCanvas("viewer", ui.CanvasCallbacks{
				Streaming: true,
				OnDraw: func(cv *render.Canvas, area render.Rect, _ bool) {
					canvasDraws++
					cv.FillRect(area, render.RGB(20, 40, 60))
				},
			}, ui.Width(200), ui.Height(150))

			// Оверлей далеко справа от видеохолста (вне 200 + 32px запаса под тень)
			ovRect := render.Rect{X: 280, Y: 100, W: 80, H: 80}
			u.ctx.PushOverlay(ovRect, func() {
				overlayDraws++
				u.ctx.Canvas.FillRect(ovRect, render.RGB(200, 200, 200))
			})
		},
	}
	w.ui = &UI{ctx: ctx, win: w}
	app.windows = []*Window{w}
	app.mainWin = w

	w.handleEvent(platform.Event{Type: platform.EventPaint})
	if renders != 1 || canvasDraws != 1 || overlayDraws != 1 {
		t.Fatalf("ожидался 1 полный кадр: renders=%d canvas=%d overlay=%d", renders, canvasDraws, overlayDraws)
	}

	mw.presents, mw.presentRects = 0, 0
	renders, canvasDraws, overlayDraws = 0, 0, 0

	w.handleEvent(platform.Event{Type: platform.EventWakeUp})
	if renders != 0 {
		t.Fatalf("при непересекающемся оверлее полный drawFn должен пропускаться: got renders=%d", renders)
	}
	if canvasDraws != 1 {
		t.Fatalf("ожидался 1 вызов OnDraw на EventWakeUp: got %d", canvasDraws)
	}
	if overlayDraws != 0 {
		t.Fatalf("непересекающийся оверлей не должен перерисовываться: got %d", overlayDraws)
	}
	if mw.presentRects != 1 || mw.presents != 0 {
		t.Fatalf("ожидался fast-path PresentRect без Present: presentRects=%d presents=%d", mw.presentRects, mw.presents)
	}
}

func TestStreamingWakeUpWithIntersectingOverlayAndLetterbox(t *testing.T) {
	app := NewApp(Config{})
	mw := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx := ui.NewContext(mw, nil)

	renders := 0
	canvasDraws := 0
	overlayDraws := 0
	// Вписанный прямоугольник видео внутри entry.rect (0,0,200,150), оставляя letterbox-полосы по краям
	videoDst := render.Rect{X: 20, Y: 20, W: 100, H: 80}
	ovRect := render.Rect{X: 0, Y: 0, W: 160, H: 140}

	w := &Window{
		app:     app,
		platWin: mw,
		ctx:     ctx,
		drawFn: func(u *UI) {
			renders++
			u.ctx.InteractiveCanvas("viewer", ui.CanvasCallbacks{
				Streaming: true,
				OnDraw: func(cv *render.Canvas, area render.Rect, _ bool) {
					canvasDraws++
					// Рисуем только в videoDst, не трогая letterbox-полосы вокруг него
					cv.FillRect(videoDst, render.RGB(200, 100, 50))
				},
			}, ui.Width(200), ui.Height(150))

			// Полупрозрачный оверлей, перекрывающий и videoDst, и letterbox-полосы
			u.ctx.PushOverlay(ovRect, func() {
				overlayDraws++
				u.ctx.RegisterHit("ov_btn", render.Rect{X: 10, Y: 10, W: 40, H: 20}, platform.CursorHand)
				u.ctx.Canvas.FillRect(ovRect, render.RGBA(0, 0, 0, 140))
			})
		},
	}
	w.ui = &UI{ctx: ctx, win: w}
	app.windows = []*Window{w}
	app.mainWin = w

	w.handleEvent(platform.Event{Type: platform.EventPaint})

	pixelAt := func(x, y int) [4]byte {
		off := (y*ctx.Canvas.Width + x) * 4
		return [4]byte{
			ctx.Canvas.Pixels[off],
			ctx.Canvas.Pixels[off+1],
			ctx.Canvas.Pixels[off+2],
			ctx.Canvas.Pixels[off+3],
		}
	}

	// Запоминаем эталонные пиксели после первого полного кадра:
	// (5, 5) — на letterbox-полосе под полупрозрачным оверлеем,
	// (50, 50) — внутри videoDst под полупрозрачным оверлеем.
	letterboxAfterFull := pixelAt(5, 5)
	videoAfterFull := pixelAt(50, 50)

	mw.presents, mw.presentRects = 0, 0
	renders, canvasDraws, overlayDraws = 0, 0, 0

	for i := 0; i < 10; i++ {
		w.handleEvent(platform.Event{Type: platform.EventWakeUp})
	}

	if renders != 0 {
		t.Fatalf("при пересекающемся оверлее полный drawFn должен пропускаться: got renders=%d", renders)
	}
	if canvasDraws != 10 {
		t.Fatalf("ожидались 10 перерисовок видео: canvas=%d", canvasDraws)
	}
	if overlayDraws != 0 {
		t.Fatalf("благодаря кэшированию оверлея в OverlayLayer функция draw оверлея не должна вызываться на wakeOnly-кадрах: overlayDraws=%d", overlayDraws)
	}
	if mw.presentRects != 10 || mw.presents != 0 {
		t.Fatalf("ожидался fast-path PresentRect на всех 10 кадрах: presentRects=%d presents=%d", mw.presentRects, mw.presents)
	}
	if mw.lastPresentRect != videoDst {
		t.Fatalf("ожидался PresentRect строго по videoDst %+v, got %+v", videoDst, mw.lastPresentRect)
	}
	if got := pixelAt(5, 5); got != letterboxAfterFull {
		t.Fatalf("альфа-накопление на letterbox-полосе: было %v, стало %v", letterboxAfterFull, got)
	}
	if got := pixelAt(50, 50); got != videoAfterFull {
		t.Fatalf("цвет внутри videoDst под полупрозрачным оверлеем изменился: было %v, стало %v", videoAfterFull, got)
	}
}

func TestStreamingWakeUpOverlayLayoutSeqPreserved(t *testing.T) {
	app := NewApp(Config{})
	mw := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx := ui.NewContext(mw, nil)

	renders := 0
	canvasDraws := 0
	overlayDraws := 0
	ovRect := render.Rect{X: 10, Y: 10, W: 180, H: 120}

	w := &Window{
		app:     app,
		platWin: mw,
		ctx:     ctx,
		drawFn: func(u *UI) {
			renders++
			u.ctx.InteractiveCanvas("viewer", ui.CanvasCallbacks{
				Streaming: true,
				OnDraw: func(cv *render.Canvas, area render.Rect, _ bool) {
					canvasDraws++
					cv.FillRect(area, render.RGB(30, 60, 90))
				},
			}, ui.Width(200), ui.Height(150))

			// Оверлей с RowWith/ColumnWith (используют nextJustifyKey -> c.layoutSeq и c.justifySizes)
			// и виджетом с пустым id (использует c.idSeqStack).
			u.ctx.PushOverlay(ovRect, func() {
				overlayDraws++
				u.ctx.LayoutRect(ovRect, func() {
					u.ctx.ColumnWith(ui.AlignStart, ui.JustifySpaceBetween, 4, func() {
						u.ctx.RowWith(ui.AlignStart, ui.JustifyCenter, 4, 32, func() {
							u.ctx.Button("", "OK", false, 80)
						})
					})
				})
			})
		},
	}
	w.ui = &UI{ctx: ctx, win: w}
	app.windows = []*Window{w}
	app.mainWin = w

	// Первый полный кадр прогревает justifySizes (внутри Render выполняется 2 прохода)
	w.handleEvent(platform.Event{Type: platform.EventPaint})

	mw.presents, mw.presentRects = 0, 0
	renders, canvasDraws, overlayDraws = 0, 0, 0

	// На серии EventWakeUp ключи justifySizes и ResolveID в оверлее должны совпадать с полным кадром,
	// без ложного RequestRedraw() и без повторного вызова drawFn/OnDraw.
	for i := 0; i < 10; i++ {
		w.handleEvent(platform.Event{Type: platform.EventWakeUp})
	}

	if renders != 0 {
		t.Fatalf("оверлей с ColumnWith/RowWith не должен вызывать ложный RequestRedraw и полный drawFn: renders=%d", renders)
	}
	if canvasDraws != 10 {
		t.Fatalf("OnDraw должен вызываться ровно 1 раз на EventWakeUp (без двойного прохода): got %d", canvasDraws)
	}
	if overlayDraws != 0 {
		t.Fatalf("благодаря кэшированию оверлея в OverlayLayer функция draw оверлея не должна вызываться на wakeOnly-кадрах: got %d", overlayDraws)
	}
	if mw.presentRects != 10 || mw.presents != 0 {
		t.Fatalf("ожидался fast-path PresentRect на всех 10 кадрах: presentRects=%d presents=%d", mw.presentRects, mw.presents)
	}
}

func TestDirectInputWithOpenOverlay(t *testing.T) {
	app := NewApp(Config{})
	mw := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx := ui.NewContext(mw, nil)

	renders := 0
	mouseDowns := 0
	mouseMoves := 0
	mouseUps := 0
	mouseWheels := 0
	ovRect := render.Rect{X: 200, Y: 100, W: 150, H: 120}

	w := &Window{
		app:     app,
		platWin: mw,
		ctx:     ctx,
		drawFn: func(u *UI) {
			renders++
			u.ctx.InteractiveCanvas("viewer", ui.CanvasCallbacks{
				DirectInput: true,
				Streaming:   true,
				OnMouseDown: func(_, _ int, _ platform.MouseButton, _ platform.Modifiers) {
					mouseDowns++
				},
				OnMouseMove: func(_, _ int, _ platform.Modifiers) {
					mouseMoves++
				},
				OnMouseUp: func(_, _ int, _ platform.MouseButton, _ platform.Modifiers) {
					mouseUps++
				},
				OnMouseWheel: func(_, _ int, _, _ float32, _ platform.Modifiers) {
					mouseWheels++
				},
			}, ui.Width(400), ui.Height(300))

			u.ctx.PushOverlay(ovRect, func() {
				u.ctx.Canvas.FillRect(ovRect, render.RGB(40, 40, 40))
			})
		},
	}
	w.ui = &UI{ctx: ctx, win: w}
	app.windows = []*Window{w}
	app.mainWin = w

	// Первый кадр: рисуем холст и открытый оверлей, затем ставим фокус на холст кликом мимо оверлея
	w.handleEvent(platform.Event{Type: platform.EventPaint})
	w.handleEvent(platform.Event{Type: platform.EventMouseDown, X: 50, Y: 50, Button: platform.MouseLeft})
	w.handleEvent(platform.Event{Type: platform.EventMouseUp, X: 50, Y: 50, Button: platform.MouseLeft})
	w.handleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 50})

	renders, mouseDowns, mouseMoves, mouseUps, mouseWheels = 0, 0, 0, 0, 0

	// Взаимодействие мышью МИМО оверлея (x=60, y=60) должно идти по DirectInput без вызова drawFn
	w.handleEvent(platform.Event{Type: platform.EventMouseMove, X: 60, Y: 60})
	w.handleEvent(platform.Event{Type: platform.EventMouseDown, X: 60, Y: 60, Button: platform.MouseLeft})
	w.handleEvent(platform.Event{Type: platform.EventMouseMove, X: 70, Y: 70})
	w.handleEvent(platform.Event{Type: platform.EventMouseMove, X: 80, Y: 80})
	w.handleEvent(platform.Event{Type: platform.EventMouseUp, X: 80, Y: 80, Button: platform.MouseLeft})
	w.handleEvent(platform.Event{Type: platform.EventMouseWheel, X: 80, Y: 80, WheelY: 1})

	if renders != 0 {
		t.Fatalf("события мыши мимо оверлея не должны вызывать полный drawFn: got renders=%d", renders)
	}
	if mouseDowns != 1 || mouseMoves != 3 || mouseUps != 1 || mouseWheels != 1 {
		t.Fatalf("ожидались прямые колбэки холста мимо оверлея: down=%d move=%d up=%d wheel=%d",
			mouseDowns, mouseMoves, mouseUps, mouseWheels)
	}

	// Клик ВНУТРИ оверлея (x=250, y=150) НЕ должен перехватываться DirectInput-холстом
	renders, mouseDowns, mouseMoves, mouseUps = 0, 0, 0, 0
	w.handleEvent(platform.Event{Type: platform.EventMouseDown, X: 250, Y: 150, Button: platform.MouseLeft})
	if renders == 0 {
		t.Fatalf("клик внутри оверлея должен идти через полный HandleEvent/Render, а не DirectInput")
	}
	if mouseDowns != 0 {
		t.Fatalf("основной DirectInput-холст не должен получать клик внутри оверлея: got mouseDowns=%d", mouseDowns)
	}
}





