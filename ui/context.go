package ui

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

// Коды клавиш для обработки редактирования текста и навигации.
const (
	KeyBackspace uint32 = 0x08
	KeyEnter     uint32 = 0x0D
	KeyEscape    uint32 = 0x1B
	KeySpace     uint32 = 0x20
	KeyPageUp    uint32 = 0x21
	KeyPageDown  uint32 = 0x22
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
	KeyY         uint32 = 0x59
	KeyZ         uint32 = 0x5A
)

// KeyPress описывает одно нажатие клавиши за кадр.
type KeyPress struct {
	Code uint32
	Mods platform.Modifiers
}

// RawKeyEvent описывает сырое событие нажатия или отпускания клавиши за кадр.
type RawKeyEvent struct {
	Pressed  bool               // true — KeyDown, false — KeyUp
	KeyCode  uint32             // Виртуальный код клавиши (VK_* на Windows)
	ScanCode uint16             // Аппаратный скан-код клавиши
	Extended bool               // Признак расширенной клавиши (bit 24 lParam на Windows)
	Mods     platform.Modifiers // Состояние модификаторов
}

// RawMouseEventKind задаёт тип сырого события мыши или колеса.
type RawMouseEventKind uint8

const (
	RawMouseMove RawMouseEventKind = iota
	RawMouseDown
	RawMouseUp
	RawMouseWheel
)

// RawMouseEvent описывает сырое событие мыши или колеса за кадр.
type RawMouseEvent struct {
	Kind   RawMouseEventKind
	X      int
	Y      int
	Button platform.MouseButton
	WheelX float32
	WheelY float32
	Mods   platform.Modifiers
}

type directCanvasEntry struct {
	id   string
	rect render.Rect
	cb   CanvasCallbacks
}

type streamingCanvasEntry struct {
	id     string
	rect   render.Rect
	onDraw func(canvas *render.Canvas, area render.Rect, focused bool)
}

// Context хранит состояние текущего кадра Immediate Mode UI.
type Context struct {
	Win      platform.Window
	Canvas   *render.Canvas
	Font     *render.FontManager
	Theme    Theme
	DpiScale float32
	Animator *Animator
	Overlays OverlayLayer
	Clock    func() time.Time // Виртуальный или пользовательский источник системного времени

	// Состояние указателя мыши (в физических пикселях)
	MouseX            int
	MouseY            int
	MouseDown         bool
	MousePressed        bool
	MouseReleased       bool
	WheelX              float32
	WheelY              float32
	RightMouseDown      bool
	RightMousePressed   bool
	RightMouseReleased  bool
	RightMouseX         int
	RightMouseY         int
	MiddleMouseDown     bool
	MiddleMousePressed  bool
	MiddleMouseReleased bool
	MiddleMouseX        int
	MiddleMouseY        int
	MouseMods           platform.Modifiers

	// Ввод с клавиатуры за текущий кадр
	TypedRunes       []rune
	KeyEvents        []KeyPress
	rawKeyEvents     []RawKeyEvent
	rawMouseEvents   []RawMouseEvent
	captureKeysFocus bool
	nextCaptureKeys  bool
	prevFocusID      string

	// Идентификаторы активных виджетов
	HotID    string
	ActiveID string
	FocusID  string

	// Внутреннее состояние для текстовых полей и контролов
	caretPos        map[string]int
	textSelStart    map[string]int
	textScroll      map[string]int
	textPrefX       map[string]int
	textHistory     map[string]*textUndoState
	textMenuChanged map[string]bool
	passwordVisible map[string]bool
	scrollState     map[string]*scrollInfo
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
	toastPos          ToastPosition
	toastOffsetXDp    Dp
	toastOffsetYDp    Dp
	toastHasOffset    bool

	focusOrder     []string
	nextFocusOrder []string
	shortcuts      []Shortcut

	layoutStack   []layoutBox
	idScopeStack  []string
	idSeqStack    []int
	rememberMu    sync.Mutex
	rememberStore map[string]*RememberEntry
	autoHeights   map[string]int
	justifySizes map[string][]int
	layoutSeq    int
	nextCursor   platform.CursorType
	lastCursor  platform.CursorType
	needsRedraw bool
	inRender    bool
	wakePending atomic.Bool

	lastClickTime time.Time
	lastClickX    int
	lastClickY    int
	clickCount    int

	animMu          sync.Mutex
	animTimerActive bool
	animStopCh      chan struct{}

	// Локализуемые строки встроенных контролов и состояние кастомной шапки
	Strings             Strings
	titleBarRect        render.Rect
	titleBarInteractive []render.Rect

	// Слои, кеш hit-областей и фильтрация EventMouseMove (Фаза 7.2)
	layers            LayerManager
	lastTheme         Theme
	lastScale         float32
	hitRegions        []hitRegion
	nextHitRegions    []hitRegion
	lastHotID         string
	lastHitID         string
	smartMouseMove    bool
	skippedMouseMove  uint64
	lastDirtyRects    []render.Rect
	prevPresentPixels []byte
	prevPresentW      int
	prevPresentH      int
	paintRequested     bool
	debugDirtyRect     bool
	prevDebugDirty     render.Rect
	prevPresentStale   bool
	wakeOnlyFrame         bool
	streamingDirty        render.Rect
	directCanvases        []directCanvasEntry
	nextDirectCanvases    []directCanvasEntry
	streamingCanvases     []streamingCanvasEntry
	nextStreamingCanvases []streamingCanvasEntry
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
	canvas := render.NewCanvas(0, 0)
	if bp, ok := win.(platform.BackBufferProvider); ok {
		canvas.ResizeWithBuffer(w, h, bp.BackBuffer(w, h))
	} else {
		canvas.Resize(w, h)
	}
	return &Context{
		Win:            win,
		Canvas:         canvas,
		Font:           fm,
		Theme:          NavyTheme(),
		DpiScale:       scale,
		Animator:       NewAnimator(),
		Strings:        RussianStrings(),
		caretPos:       make(map[string]int),
		textScroll:      make(map[string]int),
		passwordVisible: make(map[string]bool),
		layers:          newLayerManager(),
		smartMouseMove: true,
	}
}

// Scale переводит логические пиксели (Dp) в физические (Px) с учётом DPI-масштаба.
func (c *Context) Scale(dp Dp) Px {
	if dp == 0 {
		return 0
	}
	v := int(float32(dp)*c.DpiScale + 0.5)
	if v == 0 && dp > 0 {
		return 1
	}
	if v == 0 && dp < 0 {
		return -1
	}
	return Px(v)
}

// ScaleF переводит дробные логические пиксели (dp) в физические.
func (c *Context) ScaleF(dp float32) float32 {
	return dp * c.DpiScale
}

// PxToDp переводит физические пиксели (Px) в логические (Dp) с округлением до ближайшего целого.
func (c *Context) PxToDp(px Px) Dp {
	if px == 0 {
		return 0
	}
	scale := c.DpiScale
	if scale <= 0 {
		scale = 1.0
	}
	if px > 0 {
		return Dp(float32(px)/scale + 0.5)
	}
	return Dp(float32(px)/scale - 0.5)
}

// PxToDpf переводит физические пиксели (px) в дробные логические пиксели (dp).
func (c *Context) PxToDpf(px float32) float32 {
	scale := c.DpiScale
	if scale <= 0 {
		scale = 1.0
	}
	return px / scale
}

// Now возвращает текущее виртуальное или системное время контекста.
func (c *Context) Now() time.Time {
	if c.Clock != nil {
		return c.Clock()
	}
	return time.Now()
}

// DrawText рисует строку текста в произвольных физических координатах (для кастомного рисования и ListView).
func (c *Context) DrawText(x, y int, text string, sizeDp Dp, col render.Color) {
	if c.Font == nil || text == "" {
		return
	}
	c.Font.DrawText(c.Canvas, x, y, text, c.Scale(sizeDp).Int(), col)
}

// RequestRedraw запрашивает повторную перерисовку после изменения состояния.
func (c *Context) RequestRedraw() {
	c.needsRedraw = true
}

// SetCursor задаёт курсор мыши на текущий кадр (актуально при наведении на кастомные области).
func (c *Context) SetCursor(cursor platform.CursorType) {
	c.nextCursor = cursor
}

// PushOverlay регистрирует отложенную отрисовку всплывающего элемента поверх основного UI.
func (c *Context) PushOverlay(bounds render.Rect, drawFn func()) {
	c.Overlays.Push(bounds, drawFn)
}

// IsInputBlocked возвращает true, если текущая позиция курсора перекрыта активным оверлеем.
func (c *Context) IsInputBlocked() bool {
	return c.Overlays.IsBlocked(c.MouseX, c.MouseY)
}

// ClickCount возвращает количество последовательных кликов ЛКМ (1, 2 или 3) в текущем кадре.
func (c *Context) ClickCount() int {
	if !c.MousePressed {
		return 0
	}
	if c.clickCount <= 0 {
		return 1
	}
	return c.clickCount
}

// DoubleClicked возвращает true, если в текущем кадре произошёл двойной клик ЛКМ.
func (c *Context) DoubleClicked() bool {
	return c.ClickCount() == 2
}

// Run запускает главный цикл событий окна и вызывает drawFn при необходимости перерисовки.
func (c *Context) Run(drawFn func(ctx *Context)) error {
	return c.Win.Run(func(ev platform.Event) {
		if c.HandleEvent(ev) {
			c.Render(drawFn)
		}
	})
}

// SetStreamingDirtyRect регистрирует область потокового обновления (например, видеокадр InteractiveCanvas),
// для которой в presentFrame() можно пропустить computeDiffRect и копирование в prevPresentPixels.
func (c *Context) SetStreamingDirtyRect(r render.Rect) {
	if r.Empty() {
		return
	}
	c.streamingDirty = c.streamingDirty.Union(r)
}

func (c *Context) findDirectCanvasByID(id string) *directCanvasEntry {
	for i := len(c.directCanvases) - 1; i >= 0; i-- {
		if c.directCanvases[i].id == id {
			return &c.directCanvases[i]
		}
	}
	return nil
}

func (c *Context) findDirectCanvasAt(x, y int) *directCanvasEntry {
	hitID, _, _, found := c.hitTest(x, y)
	if !found {
		return nil
	}
	for i := len(c.directCanvases) - 1; i >= 0; i-- {
		e := &c.directCanvases[i]
		if e.id == hitID && e.rect.Contains(x, y) {
			return e
		}
	}
	return nil
}

// tryDirectCanvasEvent пытается доставить событие напрямую в InteractiveCanvas с DirectInput=true
// без запуска полного цикла Render(). Возвращает (handled, needRedraw).
func (c *Context) tryDirectCanvasEvent(ev platform.Event) (bool, bool) {
	if len(c.directCanvases) == 0 || c.Overlays.HasActive() {
		return false, false
	}

	switch ev.Type {
	case platform.EventMouseMove:
		var entry *directCanvasEntry
		if c.ActiveID != "" {
			entry = c.findDirectCanvasByID(c.ActiveID)
		} else {
			cand := c.findDirectCanvasAt(ev.X, ev.Y)
			if cand != nil && c.lastHitID == cand.id {
				entry = cand
			}
		}
		if entry == nil {
			return false, false
		}
		c.MouseX = ev.X
		c.MouseY = ev.Y
		c.needsRedraw = false
		if entry.cb.OnMouseMove != nil {
			entry.cb.OnMouseMove(ev.X-entry.rect.X, ev.Y-entry.rect.Y, ev.Mods)
		}
		if !c.needsRedraw {
			c.skippedMouseMove++
		}
		return true, c.needsRedraw

	case platform.EventMouseDown:
		if ev.Button != platform.MouseLeft && ev.Button != platform.MouseRight && ev.Button != platform.MouseMiddle {
			return false, false
		}
		entry := c.findDirectCanvasAt(ev.X, ev.Y)
		if entry == nil || c.FocusID != entry.id {
			return false, false
		}
		c.MouseX = ev.X
		c.MouseY = ev.Y
		c.MouseMods = ev.Mods
		switch ev.Button {
		case platform.MouseLeft:
			c.MouseDown = true
		case platform.MouseRight:
			c.RightMouseDown = true
			c.RightMouseX = ev.X
			c.RightMouseY = ev.Y
		case platform.MouseMiddle:
			c.MiddleMouseDown = true
			c.MiddleMouseX = ev.X
			c.MiddleMouseY = ev.Y
		}
		c.ActiveID = entry.id
		c.needsRedraw = false
		if entry.cb.OnMouseDown != nil {
			entry.cb.OnMouseDown(ev.X-entry.rect.X, ev.Y-entry.rect.Y, ev.Button, ev.Mods)
		}
		return true, c.needsRedraw

	case platform.EventMouseUp:
		if ev.Button != platform.MouseLeft && ev.Button != platform.MouseRight && ev.Button != platform.MouseMiddle {
			return false, false
		}
		var entry *directCanvasEntry
		if c.ActiveID != "" {
			entry = c.findDirectCanvasByID(c.ActiveID)
		} else {
			entry = c.findDirectCanvasAt(ev.X, ev.Y)
		}
		if entry == nil || c.FocusID != entry.id {
			return false, false
		}
		c.MouseX = ev.X
		c.MouseY = ev.Y
		switch ev.Button {
		case platform.MouseLeft:
			c.MouseDown = false
		case platform.MouseRight:
			c.RightMouseDown = false
		case platform.MouseMiddle:
			c.MiddleMouseDown = false
		}
		if !c.MouseDown && !c.RightMouseDown && !c.MiddleMouseDown {
			c.ActiveID = ""
		}
		c.needsRedraw = false
		if entry.cb.OnMouseUp != nil {
			entry.cb.OnMouseUp(ev.X-entry.rect.X, ev.Y-entry.rect.Y, ev.Button, ev.Mods)
		}
		return true, c.needsRedraw

	case platform.EventMouseWheel:
		mx, my := ev.X, ev.Y
		if mx == 0 && my == 0 {
			mx, my = c.MouseX, c.MouseY
		}
		entry := c.findDirectCanvasAt(mx, my)
		if entry == nil || c.FocusID != entry.id {
			return false, false
		}
		c.needsRedraw = false
		if entry.cb.OnMouseWheel != nil {
			entry.cb.OnMouseWheel(mx-entry.rect.X, my-entry.rect.Y, ev.WheelX, ev.WheelY, ev.Mods)
		}
		return true, c.needsRedraw

	case platform.EventKeyDown, platform.EventKeyUp:
		if c.FocusID == "" || !c.captureKeysFocus {
			return false, false
		}
		entry := c.findDirectCanvasByID(c.FocusID)
		if entry == nil {
			return false, false
		}
		c.needsRedraw = false
		if entry.cb.OnKey != nil {
			entry.cb.OnKey(RawKeyEvent{
				Pressed:  ev.Type == platform.EventKeyDown,
				KeyCode:  ev.KeyCode,
				ScanCode: ev.ScanCode,
				Extended: ev.Extended,
				Mods:     ev.Mods,
			})
		}
		return true, c.needsRedraw

	case platform.EventChar:
		if c.FocusID == "" || !c.captureKeysFocus {
			return false, false
		}
		if c.findDirectCanvasByID(c.FocusID) != nil {
			return true, false
		}
	}

	return false, false
}

// HandleEvent обрабатывает платформенное событие и возвращает true, если требуется перерисовка.
func (c *Context) HandleEvent(ev platform.Event) bool {
	if handled, needRedraw := c.tryDirectCanvasEvent(ev); handled {
		if needRedraw {
			c.wakeOnlyFrame = false
		}
		return needRedraw && c.Canvas.Width > 0 && c.Canvas.Height > 0
	}

	redraw := false

	switch ev.Type {
	case platform.EventClose:
		c.stopAnimTimer()
		if _, ok := c.Win.(platform.BackBufferProvider); ok {
			c.Canvas.Pixels = nil
			c.Canvas.Width = 0
			c.Canvas.Height = 0
		}
	case platform.EventDPIChange:
		if ev.Scale > 0 {
			c.DpiScale = ev.Scale
			c.layers.InvalidateStatic()
			c.wakeOnlyFrame = false
			redraw = true
		}
	case platform.EventResize:
		if bp, ok := c.Win.(platform.BackBufferProvider); ok {
			c.Canvas.Pixels = nil
			c.Canvas.ResizeWithBuffer(ev.Width, ev.Height, bp.BackBuffer(ev.Width, ev.Height))
		} else {
			c.Canvas.Resize(ev.Width, ev.Height)
		}
		c.layers.InvalidateStatic()
		c.wakeOnlyFrame = false
		redraw = true
	case platform.EventPaint:
		c.paintRequested = true
		c.wakeOnlyFrame = false
		redraw = true
	case platform.EventWakeUp:
		c.wakePending.Store(false)
		c.wakeOnlyFrame = true
		redraw = true
	case platform.EventMouseMove:
		c.MouseX = ev.X
		c.MouseY = ev.Y
		c.rawMouseEvents = append(c.rawMouseEvents, RawMouseEvent{
			Kind: RawMouseMove,
			X:    ev.X,
			Y:    ev.Y,
			Mods: ev.Mods,
		})
		if c.shouldRedrawOnMouseMove(ev.X, ev.Y) {
			redraw = true
		} else {
			c.skippedMouseMove++
		}
	case platform.EventMouseDown:
		if ev.Button == platform.MouseLeft {
			now := c.Now()
			dist := c.Scale(5).Int()
			if dist < 5 {
				dist = 5
			}
			dx := ev.X - c.lastClickX
			dy := ev.Y - c.lastClickY
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}
			dblTime := 500 * time.Millisecond
			if c.Win != nil {
				if ms := c.Win.DoubleClickTimeMs(); ms > 0 {
					dblTime = time.Duration(ms) * time.Millisecond
				}
			}
			if !c.lastClickTime.IsZero() && now.Sub(c.lastClickTime) <= dblTime && dx <= dist && dy <= dist {
				if c.clickCount < 3 {
					c.clickCount++
				} else {
					c.clickCount = 1
				}
			} else {
				c.clickCount = 1
			}
			c.lastClickTime = now
			c.lastClickX = ev.X
			c.lastClickY = ev.Y

			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.MouseDown = true
			c.MousePressed = true
			c.MouseMods = ev.Mods
			// Клик мимо сфокусированного поля сбросит фокус, если виджет не перехватит его обратно
			c.FocusID = ""
			redraw = true
		} else if ev.Button == platform.MouseRight {
			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.RightMouseDown = true
			c.RightMousePressed = true
			c.RightMouseX = ev.X
			c.RightMouseY = ev.Y
			c.MouseMods = ev.Mods
			redraw = true
		} else if ev.Button == platform.MouseMiddle {
			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.MiddleMouseDown = true
			c.MiddleMousePressed = true
			c.MiddleMouseX = ev.X
			c.MiddleMouseY = ev.Y
			c.MouseMods = ev.Mods
			redraw = true
		}
		if ev.Button == platform.MouseLeft || ev.Button == platform.MouseRight || ev.Button == platform.MouseMiddle {
			c.rawMouseEvents = append(c.rawMouseEvents, RawMouseEvent{
				Kind:   RawMouseDown,
				X:      ev.X,
				Y:      ev.Y,
				Button: ev.Button,
				Mods:   ev.Mods,
			})
		}
	case platform.EventMouseWheel:
		mx, my := ev.X, ev.Y
		if mx == 0 && my == 0 {
			mx, my = c.MouseX, c.MouseY
		}
		c.rawMouseEvents = append(c.rawMouseEvents, RawMouseEvent{
			Kind:   RawMouseWheel,
			X:      mx,
			Y:      my,
			WheelX: ev.WheelX,
			WheelY: ev.WheelY,
			Mods:   ev.Mods,
		})
		c.WheelX += ev.WheelX
		if ev.Mods.Shift && ev.WheelY != 0 && ev.WheelX == 0 {
			c.WheelX -= ev.WheelY
		} else {
			c.WheelY += ev.WheelY
		}
		redraw = true
	case platform.EventMouseUp:
		if ev.Button == platform.MouseLeft {
			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.MouseDown = false
			c.MouseReleased = true
			redraw = true
		} else if ev.Button == platform.MouseRight {
			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.RightMouseDown = false
			c.RightMouseReleased = true
			redraw = true
		} else if ev.Button == platform.MouseMiddle {
			c.MouseX = ev.X
			c.MouseY = ev.Y
			c.MiddleMouseDown = false
			c.MiddleMouseReleased = true
			redraw = true
		}
		if ev.Button == platform.MouseLeft || ev.Button == platform.MouseRight || ev.Button == platform.MouseMiddle {
			c.rawMouseEvents = append(c.rawMouseEvents, RawMouseEvent{
				Kind:   RawMouseUp,
				X:      ev.X,
				Y:      ev.Y,
				Button: ev.Button,
				Mods:   ev.Mods,
			})
		}
	case platform.EventCaptureLost:
		c.MouseDown = false
		c.RightMouseDown = false
		c.MiddleMouseDown = false
		c.ActiveID = ""
		redraw = true
	case platform.EventFocusLost:
		c.FocusID = ""
		c.MouseDown = false
		c.RightMouseDown = false
		c.MiddleMouseDown = false
		c.ActiveID = ""
		redraw = true
	case platform.EventKeyDown:
		c.KeyEvents = append(c.KeyEvents, KeyPress{Code: ev.KeyCode, Mods: ev.Mods})
		c.rawKeyEvents = append(c.rawKeyEvents, RawKeyEvent{
			Pressed:  true,
			KeyCode:  ev.KeyCode,
			ScanCode: ev.ScanCode,
			Extended: ev.Extended,
			Mods:     ev.Mods,
		})
		redraw = true
	case platform.EventKeyUp:
		c.rawKeyEvents = append(c.rawKeyEvents, RawKeyEvent{
			Pressed:  false,
			KeyCode:  ev.KeyCode,
			ScanCode: ev.ScanCode,
			Extended: ev.Extended,
			Mods:     ev.Mods,
		})
		redraw = true
	case platform.EventChar:
		c.TypedRunes = append(c.TypedRunes, ev.Char)
		redraw = true
	}

	if redraw && ev.Type != platform.EventWakeUp {
		c.wakeOnlyFrame = false
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

	// Быстрый путь для фонового потокового кадра (EventWakeUp + Streaming):
	// пропускаем полную очистку Canvas и перестроение дерева виджетов,
	// перерисовывая только зарегистрированные потоковые холсты поверх готового буфера.
	if c.canSkipFullFrameForStreaming() {
		c.renderStreamingOnly()
		if !c.needsRedraw {
			c.presentFrame()
			c.syncAnimTimer()
			return
		}
		c.wakeOnlyFrame = false
	}

	// Максимум 2 прохода на случай, если событие изменило состояние виджета
	for pass := 0; pass < 2; pass++ {
		c.beginFrame()
		drawFn(c)
		c.endFrame()

		if !c.needsRedraw {
			break
		}
	}

	c.presentFrame()
	c.syncAnimTimer()
}

func (c *Context) canSkipFullFrameForStreaming() bool {
	if !c.wakeOnlyFrame || c.paintRequested || len(c.streamingCanvases) == 0 {
		return false
	}
	w, h := c.Canvas.Width, c.Canvas.Height
	need := w * h * 4
	if c.prevPresentW != w || c.prevPresentH != h || len(c.prevPresentPixels) != need {
		return false
	}
	if c.debugDirtyRect || c.needsRedraw || c.layers.staticDirty {
		return false
	}
	if c.Theme != c.lastTheme || c.DpiScale != c.lastScale {
		return false
	}
	if c.Overlays.HasActive() || len(c.toasts) > 0 {
		return false
	}
	if c.Animator != nil && c.Animator.HasActive() {
		return false
	}
	return true
}

func (c *Context) renderStreamingOnly() {
	c.streamingDirty = render.Rect{}
	for _, entry := range c.streamingCanvases {
		if entry.onDraw == nil || entry.rect.Empty() {
			continue
		}
		focused := c.FocusID == entry.id
		c.Canvas.PushClip(entry.rect)
		entry.onDraw(c.Canvas, entry.rect, focused)
		c.Canvas.PopClip()
		c.SetStreamingDirtyRect(entry.rect)
	}
}

// SetDebugDirtyRect включает или отключает отладочную рамку DirtyRect без искажения расчёта diff.
func (c *Context) SetDebugDirtyRect(enabled bool) {
	if c.debugDirtyRect == enabled {
		return
	}
	c.debugDirtyRect = enabled
	c.paintRequested = true
	c.RequestRedraw()
}

func (c *Context) presentFrame() {
	w, h := c.Canvas.Width, c.Canvas.Height
	need := w * h * 4
	wakeOnly := c.wakeOnlyFrame
	c.wakeOnlyFrame = false

	forceFull := c.paintRequested || c.prevPresentW != w || c.prevPresentH != h || len(c.prevPresentPixels) != need
	c.paintRequested = false

	// Fast-path для потокового обновления InteractiveCanvas (например, видеокадр по EventWakeUp):
	// пропускаем computeDiffRect и копирование в prevPresentPixels, отправляя известную область напрямую.
	if !forceFull && wakeOnly && !c.streamingDirty.Empty() && !c.debugDirtyRect &&
		!c.Overlays.HasActive() && (c.Animator == nil || !c.Animator.HasActive()) {
		sr := c.streamingDirty.Intersect(render.Rect{X: 0, Y: 0, W: w, H: h})
		if !sr.Empty() {
			c.prevPresentStale = true
			c.lastDirtyRects = append(c.lastDirtyRects[:0], sr)
			if dp, ok := c.Win.(platform.DirtyPresenter); ok && (sr.W < w || sr.H < h) {
				dp.PresentRect(c.Canvas.Pixels, w, h, sr.X, sr.Y, sr.W, sr.H)
			} else {
				c.Win.Present(c.Canvas.Pixels, w, h)
			}
			c.Canvas.ResetDirtyRects()
			return
		}
	}

	if c.prevPresentStale {
		forceFull = true
		c.prevPresentStale = false
	}

	if forceFull {
		if cap(c.prevPresentPixels) < need {
			c.prevPresentPixels = make([]byte, need)
		} else {
			c.prevPresentPixels = c.prevPresentPixels[:need]
		}
		copy(c.prevPresentPixels, c.Canvas.Pixels[:need])
		c.prevPresentW = w
		c.prevPresentH = h
		c.lastDirtyRects = append(c.lastDirtyRects[:0], render.Rect{X: 0, Y: 0, W: w, H: h})
		c.prevDebugDirty = render.Rect{}
		c.Win.Present(c.Canvas.Pixels, w, h)
		c.Canvas.ResetDirtyRects()
		return
	}

	diff := computeDiffRect(c.prevPresentPixels, c.Canvas.Pixels, w, h)
	if diff.Empty() && c.prevDebugDirty.Empty() {
		c.lastDirtyRects = c.lastDirtyRects[:0]
		c.Canvas.ResetDirtyRects()
		return
	}

	if !diff.Empty() {
		rowBytes := diff.W * 4
		for y := diff.Y; y < diff.Y+diff.H; y++ {
			off := (y*w + diff.X) * 4
			copy(c.prevPresentPixels[off:off+rowBytes], c.Canvas.Pixels[off:off+rowBytes])
		}
		c.lastDirtyRects = append(c.lastDirtyRects[:0], diff)
	} else {
		c.lastDirtyRects = c.lastDirtyRects[:0]
	}

	sendRect := diff
	if !c.prevDebugDirty.Empty() {
		if sendRect.Empty() {
			sendRect = c.prevDebugDirty
		} else {
			sendRect = sendRect.Union(c.prevDebugDirty)
		}
	}

	if c.debugDirtyRect && !diff.Empty() {
		c.Canvas.DrawRoundedBox(diff, 0, render.RGBA(0, 0, 0, 0), 2.0, c.Theme.Warning)
		c.prevDebugDirty = diff
	} else {
		c.prevDebugDirty = render.Rect{}
	}

	if dp, ok := c.Win.(platform.DirtyPresenter); ok && sendRect.W*sendRect.H < (w*h*3)/4 {
		dp.PresentRect(c.Canvas.Pixels, w, h, sendRect.X, sendRect.Y, sendRect.W, sendRect.H)
	} else {
		c.Win.Present(c.Canvas.Pixels, w, h)
	}
	c.Canvas.ResetDirtyRects()
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
	c.nextCaptureKeys = false
	c.layoutSeq = 0
	c.idScopeStack = c.idScopeStack[:0]
	c.idSeqStack = append(c.idSeqStack[:0], 0)
	c.nextFocusOrder = c.nextFocusOrder[:0]
	c.nextHitRegions = c.nextHitRegions[:0]
	c.nextDirectCanvases = c.nextDirectCanvases[:0]
	c.nextStreamingCanvases = c.nextStreamingCanvases[:0]
	c.streamingDirty = render.Rect{}
	c.layers.lastFrameBlitted = false
	c.layers.activeLayer = LayerDynamic
	if c.Theme != c.lastTheme || c.DpiScale != c.lastScale {
		c.layers.InvalidateStatic()
		c.lastTheme = c.Theme
		c.lastScale = c.DpiScale
	}

	if c.Animator != nil {
		c.Animator.BeginFrame(c.Now())
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
	ovIdx := 0
	c.Overlays.FlushWithHook(func(bounds render.Rect) {
		if !bounds.Empty() {
			c.registerHitRegion("__overlay:"+strconv.Itoa(ovIdx), bounds, platform.CursorArrow, false)
		}
		ovIdx++
	})
	c.drawToasts()
	c.focusOrder, c.nextFocusOrder = c.nextFocusOrder, c.focusOrder[:0]
	c.hitRegions, c.nextHitRegions = c.nextHitRegions, c.hitRegions[:0]
	c.directCanvases, c.nextDirectCanvases = c.nextDirectCanvases, c.directCanvases[:0]
	c.streamingCanvases, c.nextStreamingCanvases = c.nextStreamingCanvases, c.streamingCanvases[:0]
	c.lastHotID = c.HotID
	c.lastHitID, _, _, _ = c.hitTest(c.MouseX, c.MouseY)
	if !c.needsRedraw {
		c.layers.staticDirty = false
	}

	if (c.MouseReleased || c.RightMouseReleased || c.MiddleMouseReleased) &&
		!c.MouseDown && !c.RightMouseDown && !c.MiddleMouseDown {
		c.ActiveID = ""
	}
	if c.nextCursor != c.lastCursor {
		c.Win.SetCursor(c.nextCursor)
		c.lastCursor = c.nextCursor
	}
	c.captureKeysFocus = c.nextCaptureKeys
	c.prevFocusID = c.FocusID
	c.MousePressed = false
	c.MouseReleased = false
	c.WheelX = 0
	c.WheelY = 0
	c.RightMousePressed = false
	c.RightMouseReleased = false
	c.MiddleMousePressed = false
	c.MiddleMouseReleased = false
	c.TypedRunes = c.TypedRunes[:0]
	c.KeyEvents = c.KeyEvents[:0]
	c.rawKeyEvents = c.rawKeyEvents[:0]
	c.rawMouseEvents = c.rawMouseEvents[:0]
}

func formatIDKey(key any) string {
	switch v := key.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	default:
		return fmt.Sprint(v)
	}
}

// PushID добавляет префикс пространства имён в стек ID виджетов.
func (c *Context) PushID(key any) {
	part := formatIDKey(key)
	if len(c.idScopeStack) > 0 {
		part = c.idScopeStack[len(c.idScopeStack)-1] + "/" + part
	}
	c.idScopeStack = append(c.idScopeStack, part)
	c.idSeqStack = append(c.idSeqStack, 0)
}

// PopID удаляет верхний префикс из стека ID виджетов.
func (c *Context) PopID() {
	if len(c.idScopeStack) > 0 {
		c.idScopeStack = c.idScopeStack[:len(c.idScopeStack)-1]
	}
	if len(c.idSeqStack) > 1 {
		c.idSeqStack = c.idSeqStack[:len(c.idSeqStack)-1]
	}
}

// IDScope задаёт изолированное пространство имён ID на время выполнения fn.
// Удобно использовать в циклах и переиспользуемых компонентах.
func (c *Context) IDScope(key any, fn func()) {
	if fn == nil {
		return
	}
	c.PushID(key)
	defer c.PopID()
	fn()
}

// nextLocalID возвращает локальный ID без префикса IDScope.
// Используется композитными контролами, которые делегируют рендер в дочерние виджеты.
func (c *Context) nextLocalID(kind, id string) string {
	if id != "" {
		return id
	}
	if len(c.idSeqStack) == 0 {
		c.idSeqStack = append(c.idSeqStack, 0)
	}
	idx := len(c.idSeqStack) - 1
	seq := c.idSeqStack[idx]
	c.idSeqStack[idx] = seq + 1
	if kind == "" {
		kind = "w"
	}
	return kind + "#" + strconv.Itoa(seq)
}

// ResolveID возвращает итоговый уникальный ID виджета с учётом активного IDScope.
// Если передана пустая строка id == "", генерирует детерминированный ID по счётчику в текущем скоупе.
func (c *Context) ResolveID(kind, id string) string {
	id = c.nextLocalID(kind, id)
	if len(c.idScopeStack) > 0 {
		return c.idScopeStack[len(c.idScopeStack)-1] + "/" + id
	}
	return id
}


