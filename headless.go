package anyui

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
	"github.com/kirpicheff/anyui/ui"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
)

// DefaultTestFonts возвращает детерминированные встроенные шрифты GoFont (Regular, Bold, Mono).
func DefaultTestFonts() (regular, bold, mono []byte) {
	return goregular.TTF, gobold.TTF, gomono.TTF
}

type headlessWindow struct {
	w, h       int
	scale      float32
	cursor     platform.CursorType
	clipboard  string
	clipImg    image.Image
	onClose    func() bool
	lastPixels []byte
	handler    func(platform.Event)
}

func (m *headlessWindow) Run(handler func(platform.Event)) error  { m.handler = handler; return nil }
func (m *headlessWindow) Open(handler func(platform.Event)) error { m.handler = handler; return nil }
func (m *headlessWindow) Present(pixels []byte, width, height int) {
	need := width * height * 4
	if cap(m.lastPixels) < need {
		m.lastPixels = make([]byte, need)
	} else {
		m.lastPixels = m.lastPixels[:need]
	}
	copy(m.lastPixels, pixels[:need])
}
func (m *headlessWindow) PresentRect(pixels []byte, width, height, rx, ry, rw, rh int) {
	m.Present(pixels, width, height)
}
func (m *headlessWindow) Invalidate()                     {}
func (m *headlessWindow) WakeUp()                         {}
func (m *headlessWindow) Show()                           {}
func (m *headlessWindow) Hide()                           {}
func (m *headlessWindow) Close()                          {}
func (m *headlessWindow) SetTitle(string)                 {}
func (m *headlessWindow) SetCursor(c platform.CursorType) { m.cursor = c }
func (m *headlessWindow) Size() (int, int)                { return m.w, m.h }
func (m *headlessWindow) Scale() float32 {
	if m.scale > 0 {
		return m.scale
	}
	return 1.0
}
func (m *headlessWindow) ClipboardText() (string, error)          { return m.clipboard, nil }
func (m *headlessWindow) SetClipboardText(text string) error      { m.clipboard = text; return nil }
func (m *headlessWindow) ClipboardImage() (image.Image, error)    { return m.clipImg, nil }
func (m *headlessWindow) SetClipboardImage(img image.Image) error { m.clipImg = img; return nil }
func (m *headlessWindow) Minimize()                               {}
func (m *headlessWindow) Maximize()                               {}
func (m *headlessWindow) Restore()                                {}
func (m *headlessWindow) IsMaximized() bool                       { return false }
func (m *headlessWindow) IsMinimized() bool                       { return false }
func (m *headlessWindow) SetFullscreen(bool)                      {}
func (m *headlessWindow) IsFullscreen() bool                      { return false }
func (m *headlessWindow) Position() (int, int)                    { return 0, 0 }
func (m *headlessWindow) SetPosition(int, int)                    {}
func (m *headlessWindow) SetSize(width, height int)               { m.w = width; m.h = height }
func (m *headlessWindow) SetOnClose(fn func() bool)               { m.onClose = fn }
func (m *headlessWindow) NativeHandle() uintptr                   { return 0 }
func (m *headlessWindow) SetCapture()                             {}
func (m *headlessWindow) ReleaseCapture()                         {}
func (m *headlessWindow) DoubleClickTimeMs() int                  { return 500 }
func (m *headlessWindow) SetDarkTitle(bool)                       {}
func (m *headlessWindow) BringToFront()                           {}
func (m *headlessWindow) SetTitleBarColors(*platform.WindowColor, *platform.WindowColor, *platform.WindowColor) {
}

// Headless представляет изолированный рантайм AnyUI для тестирования и рендеринга без создания окон ОС.
type Headless struct {
	app         *App
	win         *Window
	platWin     *headlessWindow
	virtualTime time.Time
	pendingEvs  []platform.Event
	mu          sync.Mutex
}

// NewHeadless создаёт новый экземпляр headless-рантайма с фиксированным масштабом scale = 1.0.
// Указание Config.FontRegular обязательно для обеспечения 100% детерминированного рендеринга на всех ОС и CI.
func NewHeadless(cfg Config, drawFn ...func(*UI)) (*Headless, error) {
	if len(cfg.FontRegular) == 0 {
		return nil, errors.New("headless: Config.FontRegular обязателен для детерминированного рендеринга")
	}
	if cfg.Width <= 0 {
		cfg.Width = 800
	}
	if cfg.Height <= 0 {
		cfg.Height = 600
	}
	if cfg.Theme.Bg.A == 0 && cfg.Theme.TextPrimary.A == 0 {
		cfg.Theme = NavyTheme()
	}

	fm, err := render.NewFontFromBytes(cfg.FontRegular)
	if err != nil {
		return nil, fmt.Errorf("headless: ошибка загрузки FontRegular: %w", err)
	}
	if len(cfg.FontBold) > 0 {
		if err := fm.RegisterFontBytes(render.FontBold, cfg.FontBold); err != nil {
			return nil, fmt.Errorf("headless: ошибка загрузки FontBold: %w", err)
		}
	}
	if len(cfg.FontMono) > 0 {
		if err := fm.RegisterFontBytes(render.FontMono, cfg.FontMono); err != nil {
			return nil, fmt.Errorf("headless: ошибка загрузки FontMono: %w", err)
		}
	}

	platWin := &headlessWindow{
		w: cfg.Width,
		h: cfg.Height,
	}

	app := &App{
		cfg:  cfg,
		font: fm,
	}

	ctx := ui.NewContext(platWin, fm)
	ctx.Theme = cfg.Theme
	if cfg.Strings != nil {
		ctx.Strings = *cfg.Strings
	}

	var fn func(*UI)
	if len(drawFn) > 0 && drawFn[0] != nil {
		fn = drawFn[0]
	}

	win := &Window{
		app:         app,
		platWin:     platWin,
		ctx:         ctx,
		drawFn:      fn,
		hideOnClose: cfg.HideOnClose,
		onClose:     cfg.OnClose,
		onDropFiles: cfg.OnDropFiles,
	}
	win.ui = &UI{
		ctx: ctx,
		win: win,
		app: app,
	}
	platWin.SetOnClose(win.handleCloseRequest)
	platWin.handler = win.handleEvent

	app.windows = []*Window{win}
	app.mainWin = win

	h := &Headless{
		app:         app,
		win:         win,
		platWin:     platWin,
		virtualTime: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	ctx.Clock = h.Time

	// Первоначальное задание размера холста
	platWin.handler(platform.Event{Type: platform.EventResize, Width: cfg.Width, Height: cfg.Height})

	return h, nil
}

// SetDraw устанавливает или обновляет функцию отрисовки интерфейса.
func (h *Headless) SetDraw(drawFn func(*UI)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.win.drawFn = drawFn
}

// Time возвращает текущее виртуальное системное время.
func (h *Headless) Time() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.virtualTime
}

// SetTime устанавливает виртуальное системное время.
func (h *Headless) SetTime(t time.Time) {
	h.mu.Lock()
	h.virtualTime = t
	h.mu.Unlock()
}

// AdvanceTime перематывает виртуальное системное время на указанную величину.
func (h *Headless) AdvanceTime(d time.Duration) {
	h.mu.Lock()
	h.virtualTime = h.virtualTime.Add(d)
	h.mu.Unlock()
}

// SetScale устанавливает системный масштаб DPI (например, 1.0, 1.5, 2.0) и пересчитывает Scale контекста.
func (h *Headless) SetScale(scale float32) {
	if scale <= 0 {
		scale = 1.0
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.platWin.scale = scale
	h.pendingEvs = append(h.pendingEvs, platform.Event{
		Type:  platform.EventDPIChange,
		Scale: scale,
	})
}

// Step выполняет синхронную отрисовку ровно одного кадра с обработкой ввода и очереди Dispatch.
func (h *Headless) Step() {
	h.mu.Lock()
	h.app.flushDispatch()

	events := h.pendingEvs
	h.pendingEvs = nil
	h.mu.Unlock()

	for _, ev := range events {
		h.win.ctx.HandleEvent(ev)
	}

	h.win.ctx.Render(func(_ *ui.Context) {
		h.mu.Lock()
		fn := h.win.drawFn
		h.mu.Unlock()
		if fn != nil {
			fn(h.win.ui)
		}
	})
}

func (h *Headless) dpToPx(xDp, yDp int) (int, int) {
	h.mu.Lock()
	sc := h.platWin.Scale()
	h.mu.Unlock()
	if sc <= 0 {
		sc = 1.0
	}
	return int(float32(xDp)*sc + 0.5), int(float32(yDp)*sc + 0.5)
}

// InjectMouseMove добавляет событие перемещения мыши в координатах логических пикселей (Dp).
// Координаты автоматически масштабируются в физические пиксели ОС через текущий Scale.
func (h *Headless) InjectMouseMove(xDp, yDp int) {
	xPx, yPx := h.dpToPx(xDp, yDp)
	h.InjectMouseMovePx(xPx, yPx)
}

// InjectMouseMovePx добавляет событие перемещения мыши напрямую в физических пикселях (Px).
func (h *Headless) InjectMouseMovePx(xPx, yPx int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingEvs = append(h.pendingEvs, platform.Event{
		Type: platform.EventMouseMove,
		X:    xPx,
		Y:    yPx,
	})
}

// InjectMouseDown добавляет событие нажатия кнопки мыши в координатах логических пикселей (Dp).
func (h *Headless) InjectMouseDown(xDp, yDp int, btn MouseButton) {
	xPx, yPx := h.dpToPx(xDp, yDp)
	h.InjectMouseDownPx(xPx, yPx, btn)
}

// InjectMouseDownPx добавляет событие нажатия кнопки мыши напрямую в физических пикселях (Px).
func (h *Headless) InjectMouseDownPx(xPx, yPx int, btn MouseButton) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingEvs = append(h.pendingEvs, platform.Event{
		Type:   platform.EventMouseDown,
		X:      xPx,
		Y:      yPx,
		Button: btn,
	})
}

// InjectMouseUp добавляет событие отпускания кнопки мыши в координатах логических пикселей (Dp).
func (h *Headless) InjectMouseUp(xDp, yDp int, btn MouseButton) {
	xPx, yPx := h.dpToPx(xDp, yDp)
	h.InjectMouseUpPx(xPx, yPx, btn)
}

// InjectMouseUpPx добавляет событие отпускания кнопки мыши напрямую в физических пикселях (Px).
func (h *Headless) InjectMouseUpPx(xPx, yPx int, btn MouseButton) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingEvs = append(h.pendingEvs, platform.Event{
		Type:   platform.EventMouseUp,
		X:      xPx,
		Y:      yPx,
		Button: btn,
	})
}

// InjectMouseClick эмулирует полный клик мышью в координатах логических пикселей (Dp) (MouseMove -> MouseDown -> MouseUp).
func (h *Headless) InjectMouseClick(xDp, yDp int, btn MouseButton) {
	h.InjectMouseMove(xDp, yDp)
	h.InjectMouseDown(xDp, yDp, btn)
	h.InjectMouseUp(xDp, yDp, btn)
}

// InjectMouseClickPx эмулирует полный клик мышью напрямую в физических пикселях (Px).
func (h *Headless) InjectMouseClickPx(xPx, yPx int, btn MouseButton) {
	h.InjectMouseMovePx(xPx, yPx)
	h.InjectMouseDownPx(xPx, yPx, btn)
	h.InjectMouseUpPx(xPx, yPx, btn)
}

// InjectMouseWheel добавляет событие вращения колеса мыши в координатах логических шагов (Dp).
func (h *Headless) InjectMouseWheel(dxDp, dyDp int) {
	h.InjectMouseWheelPx(dxDp, dyDp)
}

// InjectMouseWheelPx добавляет событие вращения колеса мыши напрямую в физических шагах колеса (Px).
func (h *Headless) InjectMouseWheelPx(dxPx, dyPx int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingEvs = append(h.pendingEvs, platform.Event{
		Type:   platform.EventMouseWheel,
		WheelX: float32(dxPx),
		WheelY: float32(dyPx),
	})
}

// InjectKeyDown добавляет событие нажатия клавиши в очередь следующего шага.
func (h *Headless) InjectKeyDown(key Key, mod Modifiers) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingEvs = append(h.pendingEvs, platform.Event{
		Type:    platform.EventKeyDown,
		KeyCode: uint32(key),
		Mods:    mod,
	})
}

// InjectKeyUp добавляет событие отпускания клавиши в очередь следующего шага.
func (h *Headless) InjectKeyUp(key Key, mod Modifiers) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingEvs = append(h.pendingEvs, platform.Event{
		Type:    platform.EventKeyUp,
		KeyCode: uint32(key),
		Mods:    mod,
	})
}

// InjectKey эмулирует полный цикл нажатия и отпускания клавиши клавиатуры.
func (h *Headless) InjectKey(key Key, mod Modifiers) {
	h.InjectKeyDown(key, mod)
	h.InjectKeyUp(key, mod)
}

// InjectText эмулирует ввод текстовых символов Unicode (EventChar).
func (h *Headless) InjectText(text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range text {
		h.pendingEvs = append(h.pendingEvs, platform.Event{
			Type: platform.EventChar,
			Char: r,
		})
	}
}

// InjectFocus устанавливает программный фокус ввода на указанный идентификатор виджета.
func (h *Headless) InjectFocus(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.win.ctx.FocusID = id
	h.win.ctx.RequestRedraw()
}

// FocusID возвращает идентификатор текущего сфокусированного виджета.
func (h *Headless) FocusID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.win.ctx.FocusID
}

// FocusOrder возвращает порядок обхода фокусируемых виджетов по Tab.
func (h *Headless) FocusOrder() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.win.ctx.FocusOrder()
}

// CaptureImage возвращает снимок текущего кадра в виде стандартного image.Image (NRGBA).
func (h *Headless) CaptureImage() image.Image {
	h.mu.Lock()
	defer h.mu.Unlock()

	w, height := h.win.ctx.Canvas.Width, h.win.ctx.Canvas.Height
	if w <= 0 || height <= 0 {
		return image.NewNRGBA(image.Rect(0, 0, 0, 0))
	}
	src := h.platWin.lastPixels
	if len(src) < w*height*4 {
		src = h.win.ctx.Canvas.Pixels
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, height))
	for y := 0; y < height; y++ {
		for x := 0; x < w; x++ {
			off := (y*w + x) * 4
			b := src[off+0]
			g := src[off+1]
			r := src[off+2]
			a := src[off+3]
			imgOff := (y*w + x) * 4
			img.Pix[imgOff+0] = r
			img.Pix[imgOff+1] = g
			img.Pix[imgOff+2] = b
			img.Pix[imgOff+3] = a
		}
	}
	return img
}

// CapturePNG сохраняет текущий кадр в PNG-файл.
func (h *Headless) CapturePNG(path string) error {
	img := h.CaptureImage()
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// TestHelper описывает минимальный интерфейс тестирования, совместимый с testing.TB (*testing.T / *testing.B).
type TestHelper interface {
	Helper()
	Fatalf(format string, args ...any)
	Logf(format string, args ...any)
}

// AssertGolden сравнивает текущий снимок Headless с эталонным файлом testdata/<goldenName>.png.
// Если передан флаг -update (go test -update), файл создаётся или обновляется.
func AssertGolden(t TestHelper, h *Headless, goldenName string) {
	t.Helper()
	dir := "testdata"
	goldenPath := filepath.Join(dir, goldenName+".png")

	update := false
	if f := flag.Lookup("update"); f != nil {
		if v, ok := f.Value.(flag.Getter); ok {
			if b, ok := v.Get().(bool); ok {
				update = b
			}
		}
	}

	if update {
		if err := h.CapturePNG(goldenPath); err != nil {
			t.Fatalf("AssertGolden: не удалось сохранить эталон %s: %v", goldenPath, err)
		}
		t.Logf("AssertGolden: обновлен эталон %s", goldenPath)
		return
	}

	f, err := os.Open(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("AssertGolden: эталонный файл %s не найден; запустите с флагом -update для его создания", goldenPath)
		}
		t.Fatalf("AssertGolden: ошибка открытия %s: %v", goldenPath, err)
	}
	defer f.Close()

	expectedImg, err := png.Decode(f)
	if err != nil {
		t.Fatalf("AssertGolden: ошибка декодирования PNG %s: %v", goldenPath, err)
	}

	actualImg := h.CaptureImage()

	expBounds := expectedImg.Bounds()
	actBounds := actualImg.Bounds()
	if expBounds != actBounds {
		t.Fatalf("AssertGolden: несовпадение размеров изображения: ожидалось %v, получено %v", expBounds, actBounds)
	}

	diffCount := 0
	firstX, firstY := -1, -1
	var firstExp, firstAct any

	w := expBounds.Dx()
	height := expBounds.Dy()
	for y := 0; y < height; y++ {
		for x := 0; x < w; x++ {
			c1 := expectedImg.At(x+expBounds.Min.X, y+expBounds.Min.Y)
			c2 := actualImg.At(x+actBounds.Min.X, y+actBounds.Min.Y)
			r1, g1, b1, a1 := c1.RGBA()
			r2, g2, b2, a2 := c2.RGBA()

			// Приводим из 16-битного диапазона [0..65535] к 8-битному [0..255]
			dr := int(r1>>8) - int(r2>>8)
			if dr < 0 {
				dr = -dr
			}
			dg := int(g1>>8) - int(g2>>8)
			if dg < 0 {
				dg = -dg
			}
			db := int(b1>>8) - int(b2>>8)
			if db < 0 {
				db = -db
			}
			da := int(a1>>8) - int(a2>>8)
			if da < 0 {
				da = -da
			}

			// Допускаем погрешность не более 1 кванта цвета (1/255) из-за кроссплатформенных различий FPU/FMA (ARM64 vs x86)
			if dr > 1 || dg > 1 || db > 1 || da > 1 {
				if diffCount == 0 {
					firstX, firstY = x, y
					firstExp = c1
					firstAct = c2
				}
				diffCount++
			}
		}
	}

	if diffCount > 0 {
		failPath := filepath.Join(dir, goldenName+".failed.png")
		_ = h.CapturePNG(failPath)
		t.Fatalf("AssertGolden %s: обнаружено %d расхождений пикселей (первое на %d,%d: ожидалось %v, получено %v); актуальный снимок сохранён в %s",
			goldenName, diffCount, firstX, firstY, firstExp, firstAct, failPath)
	}
}

// UI возвращает привязанный экземпляр UI.
func (h *Headless) UI() *UI {
	return h.win.ui
}

// Window возвращает привязанное окно.
func (h *Headless) Window() *Window {
	return h.win
}

// App возвращает экземпляр приложения.
func (h *Headless) App() *App {
	return h.app
}

// Close освобождает ресурсы headless-рантайма.
func (h *Headless) Close() {
	if h.win != nil {
		h.win.Close()
	}
}
