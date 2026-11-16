package ui

import (
	"sync"
	"sync/atomic"
	"time"

	"anyui/platform"
	"anyui/render"
)

// Коды клавиш для обработки редактирования текста и навигации.
const (
	KeyBackspace uint32 = 0x08
	KeyEnter     uint32 = 0x0D
	KeyEscape    uint32 = 0x1B
	KeyEnd       uint32 = 0x23
	KeyHome      uint32 = 0x24
	KeyLeft      uint32 = 0x25
	KeyUp        uint32 = 0x26
	KeyRight     uint32 = 0x27
	KeyDown      uint32 = 0x28
	KeyDelete    uint32 = 0x2E
	KeyA         uint32 = 0x41
	KeyC         uint32 = 0x43
	KeyV         uint32 = 0x56
	KeyX         uint32 = 0x58
)

// KeyPress описывает одно нажатие клавиши за кадр.
type KeyPress struct {
	Code uint32
	Mods platform.Modifiers
}

// Context хранит состояние текущего кадра Immediate Mode UI.
type Context struct {
	Win      platform.Window
	Canvas   *render.Canvas
	Font     *render.FontManager
	Theme    Theme
	Scale    float32
	Animator *Animator
	Overlays OverlayLayer

	// Состояние указателя мыши (в физических пикселях)
	MouseX            int
	MouseY            int
	MouseDown         bool
	MousePressed      bool
	MouseReleased     bool
	WheelY            float32
	RightMousePressed bool
	RightMouseX       int
	RightMouseY       int

	// Ввод с клавиатуры за текущий кадр
	TypedRunes []rune
	KeyEvents  []KeyPress

	// Идентификаторы активных виджетов
	HotID    string
	ActiveID string
	FocusID  string

	// Внутреннее состояние для текстовых полей и контролов
	caretPos       map[string]int
	textSelStart   map[string]int
	textScroll     map[string]int
	scrollState    map[string]*scrollInfo
	openState      map[string]bool
	floatPos       map[string][2]int
	floatCollapsed map[string]bool
	tabScroll      map[string]int

	// Состояние контролов Фазы 5
	tableColWidths    map[string][]int
	tableResizeCol    map[string]int
	tableResizeStartX map[string]int
	tableResizeStartW map[string]int
	tabDragID         string
	tabDragIndex      int
	tabDragStartX     int
	tabDragStartY     int
	tabDragging       bool
	numEditText       map[string]string
	colorHexText      map[string]string
	colorHSV          map[string][3]float32
	dateViewMonth     map[string]time.Time
	spinnerAngle      map[string]float32
	toasts            []*ToastMessage
	nextToastID       int

	focusOrder     []string
	nextFocusOrder []string
	shortcuts      []Shortcut

	layoutStack []layoutBox
	nextCursor  platform.CursorType
	lastCursor  platform.CursorType
	needsRedraw bool
	inRender    bool
	wakePending atomic.Bool

	animMu          sync.Mutex
	animTimerActive bool
	animStopCh      chan struct{}
}

// NewContext создаёт новый UI-контекст поверх окна и менеджера шрифтов.
func NewContext(win platform.Window, fm *render.FontManager) *Context {
	w, h := win.Size()
	if w <= 0 {
		w = 960
	}
	if h <= 0 {
		h = 600
	}
	scale := win.Scale()
	if scale <= 0 {
		scale = 1.0
	}
	return &Context{
		Win:        win,
		Canvas:     render.NewCanvas(w, h),
		Font:       fm,
		Theme:      DeepNavyTheme(),
		Scale:      scale,
		Animator:   NewAnimator(),
		caretPos:   make(map[string]int),
		textScroll: make(map[string]int),
	}
}

// S переводит логические пиксели (dp) в физические с учётом DPI-масштаба.
func (c *Context) S(dp int) int {
	if dp == 0 {
		return 0
	}
	v := int(float32(dp)*c.Scale + 0.5)
	if v == 0 && dp > 0 {
		return 1
	}
	if v == 0 && dp < 0 {
		return -1
	}
	return v
}

// Sf переводит дробные логические пиксели (dp) в физические.
func (c *Context) Sf(dp float32) float32 {
	return dp * c.Scale
}

// DrawText рисует строку текста в произвольных физических координатах (для кастомного рисования и ListView).
func (c *Context) DrawText(x, y int, text string, sizeDp int, col render.Color) {
	if c.Font == nil || text == "" {
		return
	}
	c.Font.DrawText(c.Canvas, x, y, text, c.S(sizeDp), col)
}

// RequestRedraw запрашивает повторную перерисовку после изменения состояния.
func (c *Context) RequestRedraw() {
	c.needsRedraw = true
}

// PushOverlay регистрирует отложенную отрисовку всплывающего элемента поверх основного UI.
func (c *Context) PushOverlay(bounds render.Rect, drawFn func()) {
	c.Overlays.Push(bounds, drawFn)
}

// IsInputBlocked возвращает true, если текущая позиция курсора перекрыта активным оверлеем.
func (c *Context) IsInputBlocked() bool {
	return c.Overlays.IsBlocked(c.MouseX, c.MouseY)
}

// Run запускает главный цикл событий окна и вызывает drawFn при необходимости перерисовки.
func (c *Context) Run(drawFn func(ctx *Context)) error {
	return c.Win.Run(func(ev platform.Event) {
		if c.HandleEvent(ev) {
			c.Render(drawFn)
		}
	})
}

// HandleEvent обрабатывает платформенное событие и возвращает true, если требуется перерисовка.
func (c *Context) HandleEvent(ev platform.Event) bool {
	redraw := false

	switch ev.Type {
	case platform.EventClose:
		c.stopAnimTimer()
	case platform.EventDPIChange:
		if ev.Scale > 0 {
			c.Scale = ev.Scale
			redraw = true
		}
	case platform.EventResize:
		c.Canvas.Resize(ev.Width, ev.Height)
		redraw = true
	case platform.EventPaint:
		redraw = true
	case platform.EventWakeUp:
		c.wakePending.Store(false)
		redraw = true
	case platform.EventMouseMove:
		c.MouseX = ev.X
		c.MouseY = ev.Y
		redraw = true
	case platform.EventMouseDown:
		if ev.Button == platform.MouseLeft {
			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.MouseDown = true
			c.MousePressed = true
			// Клик мимо сфокусированного поля сбросит фокус, если виджет не перехватит его обратно
			c.FocusID = ""
			redraw = true
		} else if ev.Button == platform.MouseRight {
			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.RightMousePressed = true
			c.RightMouseX = ev.X
			c.RightMouseY = ev.Y
			redraw = true
		}
	case platform.EventMouseWheel:
		c.WheelY += ev.WheelY
		redraw = true
	case platform.EventMouseUp:
		if ev.Button == platform.MouseLeft {
			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.MouseDown = false
			c.MouseReleased = true
			redraw = true
		}
	case platform.EventKeyDown:
		c.KeyEvents = append(c.KeyEvents, KeyPress{Code: ev.KeyCode, Mods: ev.Mods})
		redraw = true
	case platform.EventChar:
		c.TypedRunes = append(c.TypedRunes, ev.Char)
		redraw = true
	}

	return redraw && c.Canvas.Width > 0 && c.Canvas.Height > 0
}

// Render выполняет отрисовку кадра через drawFn и выводит результат в окно.
func (c *Context) Render(drawFn func(ctx *Context)) {
	if c.inRender || drawFn == nil || c.Canvas.Width <= 0 || c.Canvas.Height <= 0 {
		return
	}
	c.inRender = true
	defer func() { c.inRender = false }()

	// Максимум 2 прохода на случай, если событие изменило состояние виджета
	for pass := 0; pass < 2; pass++ {
		c.beginFrame()
		drawFn(c)
		c.endFrame()

		if !c.needsRedraw {
			break
		}
	}

	c.Win.Present(c.Canvas.Pixels, c.Canvas.Width, c.Canvas.Height)
	c.syncAnimTimer()
}

func (c *Context) syncAnimTimer() {
	active := c.Animator != nil && c.Animator.HasActive()
	if active {
		c.startAnimTimer()
	} else {
		c.stopAnimTimer()
		if c.Animator != nil {
			c.Animator.ResetClock()
		}
	}
}

func (c *Context) startAnimTimer() {
	c.animMu.Lock()
	if c.animTimerActive {
		c.animMu.Unlock()
		return
	}
	stopCh := make(chan struct{})
	c.animStopCh = stopCh
	c.animTimerActive = true
	win := c.Win
	c.animMu.Unlock()

	go func() {
		ticker := time.NewTicker(16 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				if c.wakePending.CompareAndSwap(false, true) {
					win.WakeUp()
				}
			}
		}
	}()
}

func (c *Context) stopAnimTimer() {
	c.animMu.Lock()
	if !c.animTimerActive {
		c.animMu.Unlock()
		return
	}
	close(c.animStopCh)
	c.animStopCh = nil
	c.animTimerActive = false
	c.animMu.Unlock()
}

func (c *Context) beginFrame() {
	c.HotID = ""
	c.nextCursor = platform.CursorArrow
	c.needsRedraw = false
	c.nextFocusOrder = c.nextFocusOrder[:0]

	if c.Animator != nil {
		c.Animator.BeginFrame(time.Now())
	}
	c.Overlays.BeginFrame()
	c.processFocusAndShortcuts()

	c.Canvas.Clear(c.Theme.Bg)
	c.layoutStack = c.layoutStack[:0]
	c.layoutStack = append(c.layoutStack, layoutBox{
		rect:    render.Rect{X: 0, Y: 0, W: c.Canvas.Width, H: c.Canvas.Height},
		dir:     dirColumn,
		cursorX: 0,
		cursorY: 0,
	})
}

func (c *Context) endFrame() {
	c.drawToasts()
	c.Overlays.Flush()
	c.focusOrder, c.nextFocusOrder = c.nextFocusOrder, c.focusOrder[:0]

	if c.MouseReleased {
		c.ActiveID = ""
	}
	if c.nextCursor != c.lastCursor {
		c.Win.SetCursor(c.nextCursor)
		c.lastCursor = c.nextCursor
	}
	c.MousePressed = false
	c.MouseReleased = false
	c.WheelY = 0
	c.RightMousePressed = false
	c.TypedRunes = c.TypedRunes[:0]
	c.KeyEvents = c.KeyEvents[:0]
}

