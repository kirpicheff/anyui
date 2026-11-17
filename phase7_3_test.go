package anyui

import (
	"path/filepath"
	"testing"

	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/ui"
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
}

func (m *stylerMockWindow) Run(func(platform.Event)) error          { return nil }
func (m *stylerMockWindow) Open(func(platform.Event)) error         { return nil }
func (m *stylerMockWindow) Present([]byte, int, int)                {}
func (m *stylerMockWindow) Invalidate()                             {}
func (m *stylerMockWindow) WakeUp()                                 {}
func (m *stylerMockWindow) Show()                                   {}
func (m *stylerMockWindow) Hide()                                   {}
func (m *stylerMockWindow) Close()                                  {}
func (m *stylerMockWindow) SetTitle(string)                         {}
func (m *stylerMockWindow) SetCursor(platform.CursorType)           {}
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
func (m *stylerMockWindow) SetTitleBarColors(c, t, b *platform.WindowColor) {
	m.captionCol = c
	m.textCol = t
	m.borderCol = b
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
	win.SetTheme(DeepNavyTheme())
	if !mw.darkTitle {
		t.Fatal("для DeepNavyTheme должен устанавливаться тёмный системный заголовок")
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
	ctx.Theme = DeepNavyTheme()

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
				if u.Button("th_navy", "DeepNavy", Ghost(), Width(90), Height(32)) {
					u.SetTheme(DeepNavyTheme())
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

	// Стартовая отрисовка в DeepNavy
	win.handleEvent(platform.Event{Type: platform.EventPaint})
	if ctx.Theme != DeepNavyTheme() || staticCalls != 1 {
		t.Fatalf("некорректное начальное состояние темы: staticCalls=%d", staticCalls)
	}
	bgNavy := ctx.Canvas.Pixels[0:4] // BGRA в точке (0,0)
	if bgNavy[2] != DeepNavyTheme().Bg.R || bgNavy[1] != DeepNavyTheme().Bg.G || bgNavy[0] != DeepNavyTheme().Bg.B {
		t.Fatalf("фон DeepNavy не совпал: got RGB(%d,%d,%d)", bgNavy[2], bgNavy[1], bgNavy[0])
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

