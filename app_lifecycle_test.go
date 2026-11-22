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
	presents     int
	presentRects int
	alwaysOnTop  bool
}

func (m *stylerMockWindow) Run(func(platform.Event)) error          { return nil }
func (m *stylerMockWindow) Open(func(platform.Event)) error         { return nil }
func (m *stylerMockWindow) Present([]byte, int, int)                { m.presents++ }
func (m *stylerMockWindow) PresentRect([]byte, int, int, int, int, int, int) { m.presentRects++ }
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


