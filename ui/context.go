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
	id        string
	rect      render.Rect
	cb        CanvasCallbacks
	inOverlay bool
}

type tabBarRegion struct {
	id        string
	barRect   render.Rect
	tabsRect  render.Rect
	vertical  bool
	tabCount  int
	tabSizePx int
	gapPx     int
	scrollPx  int
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
	MouseX              int
	MouseY              int
	MouseDown           bool
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

	// Признак кадра с реальными событиями пользовательского ввода (мышь/клавиатура)
	isInputFrame bool

	// Быстрый путь локальной перерисовки Toast без полного Render
	toastMu           sync.Mutex
	toastTimerActive  bool
	toastStopCh       chan struct{}
	toastUnderlay     []byte
	toastUnderlayRect render.Rect

	// Внутреннее состояние для текстовых полей и контролов
	caretPos        map[string]int
	textSelStart    map[string]int
	textScroll      map[string]int
	textPrefX       map[string]int
	textHistory     map[string]*textUndoState
	textMenuChanged map[string]bool
	passwordVisible map[string]bool
	scrollState     map[string]*scrollInfo
	svStack         []scrollScope
	svChildOffsets  map[string]map[string]render.Rect
	openState       map[string]bool
	floatPos        map[string][2]int
	floatCollapsed  map[string]bool
	tabScroll       map[string]int

	// Состояние контролов Фазы 5
	tableColWidths     map[string][]int
	tableResizeCol     map[string]int
	tableResizeStartX  map[string]int
	tableResizeStartW  map[string]int
	tabDragID          string
	tabDragIndex       int
	tabDragStartX      int
	tabDragStartY      int
	tabDragging        bool
	tabBars            []tabBarRegion
	extTabHighlightID  string
	extTabInsertIdx    int
	extTabPendingBarID string
	extTabPendingID    string
	extTabPendingIdx   int
	reorderWidgetID    string
	reorderSourceIdx   int
	reorderSourceKey   string
	reorderStartX      int
	reorderStartY      int
	reorderActive      bool
	reorderSourceType  string
	reorderSourceFn    func(index int, key string) DragPayload
	reorderPayload     *DragPayload
	reorderScrollAccum float32
	numEditText        map[string]string
	colorHexText       map[string]string
	colorHSV           map[string][3]float32
	dateViewMonth      map[string]time.Time
	spinnerAngle       map[string]float32
	toasts             []*ToastMessage
	nextToastID        int
	toastPos           ToastPosition
	toastOffsetXDp     Dp
	toastOffsetYDp     Dp
	toastHasOffset     bool

	focusOrder     []string
	nextFocusOrder []string
	shortcuts      []Shortcut

	layoutStack   []layoutBox
	idScopeStack  []string
	idSeqStack    []int
	rememberMu    sync.Mutex
	rememberStore map[string]*RememberEntry
	autoHeights   map[string]int
	justifySizes  map[string][]int
	layoutSeq     int
	nextCursor       platform.CursorType
	lastCursor       platform.CursorType
	nextCustomCursor *platform.CustomCursor
	lastCustomCursor *platform.CustomCursor
	needsRedraw   bool
	inRender      bool
	wakePending   atomic.Bool

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
	layers                LayerManager
	lastTheme             Theme
	lastScale             float32
	hitRegions            []hitRegion
	nextHitRegions        []hitRegion
	lastHotID             string
	lastHitID             string
	smartMouseMove        bool
	skippedMouseMove      uint64
	lastDirtyRects        []render.Rect
	prevPresentPixels     []byte
	prevPresentW          int
	prevPresentH          int
	paintRequested        bool
	debugDirtyRect        bool
	prevDebugDirty        render.Rect
	prevPresentStale      bool
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
		Win:             win,
		Canvas:          canvas,
		Font:            fm,
		Theme:           NavyTheme(),
		DpiScale:        scale,
		Animator:        NewAnimator(),
		Strings:         RussianStrings(),
		caretPos:        make(map[string]int),
		textScroll:      make(map[string]int),
		passwordVisible: make(map[string]bool),
		layers:          newLayerManager(),
		smartMouseMove:  true,
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

// DrawText рисует строку текста в физических координатах xPx, yPx (для кастомного рисования и ListView).
func (c *Context) DrawText(xPx, yPx Px, text string, sizeDp Dp, col render.Color) {
	if c.Font == nil || text == "" {
		return
	}
	c.Font.DrawText(c.Canvas, xPx.Int(), yPx.Int(), text, c.Scale(sizeDp).Int(), col)
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

// PushNonBlockingOverlay регистрирует отложенную отрисовку всплывающего элемента поверх основного UI,
// не блокируя ввод и клики по нижележащему интерфейсу.
func (c *Context) PushNonBlockingOverlay(bounds render.Rect, drawFn func()) {
	c.Overlays.PushNonBlocking(bounds, drawFn)
}

// HasActiveOverlays возвращает true, если в текущем или предыдущем кадре есть активные оверлеи
// (Modal, Popover, FloatingPanel, Drawer, ContextMenu, Select, Tooltip, PushOverlay и др.).
func (c *Context) HasActiveOverlays() bool {
	if c == nil {
		return false
	}
	return c.Overlays.HasActive()
}

// ActiveOverlayRects возвращает прямоугольники всех активных оверлеев (включая неблокирующие
// вроде Tooltip и PushNonBlockingOverlay) и уведомлений Toast в физических пикселях (Px).
func (c *Context) ActiveOverlayRects() []render.Rect {
	if c == nil {
		return nil
	}
	rects := c.Overlays.ActiveRects()
	if len(c.toasts) > 0 {
		if tr := c.toastsBoundingRect(); !tr.Empty() {
			rects = append(rects, tr)
		}
	}
	return rects
}

func (c *Context) registerTabBar(r tabBarRegion) {
	if c == nil {
		return
	}
	c.tabBars = append(c.tabBars, r)
}

// TabBarAtClient проверяет, попадает ли клиентская координата (cx, cy) в область одного из TabBar,
// и возвращает идентификатор TabBar и индекс вставки [0..tabCount].
func (c *Context) TabBarAtClient(cx, cy int) (id string, insertIdx int, ok bool) {
	if c == nil {
		return "", -1, false
	}
	for i := len(c.tabBars) - 1; i >= 0; i-- {
		tb := c.tabBars[i]
		if !tb.barRect.Contains(cx, cy) {
			continue
		}
		step := tb.tabSizePx + tb.gapPx
		idx := tb.tabCount
		if step > 0 && tb.tabCount > 0 {
			if tb.vertical {
				rel := cy - tb.tabsRect.Y + tb.scrollPx
				idx = (rel + step/2) / step
			} else {
				rel := cx - tb.tabsRect.X + tb.scrollPx
				idx = (rel + step/2) / step
			}
			if idx < 0 {
				idx = 0
			}
			if idx > tb.tabCount {
				idx = tb.tabCount
			}
		} else {
			idx = 0
		}
		return tb.id, idx, true
	}
	return "", -1, false
}

// SetExternalTabInsert включает или сбрасывает визуальную подсветку вставки внешней вкладки в TabBar.
func (c *Context) SetExternalTabInsert(tabBarID string, insertIdx int) {
	if c == nil {
		return
	}
	if c.extTabHighlightID == tabBarID && c.extTabInsertIdx == insertIdx {
		return
	}
	c.extTabHighlightID = tabBarID
	c.extTabInsertIdx = insertIdx
	c.RequestRedraw()
}

// EnqueueExternalTabAttach регистрирует прикрепление внешней вкладки tabID к TabBar tabBarID на позицию insertIdx.
func (c *Context) EnqueueExternalTabAttach(tabBarID, tabID string, insertIdx int) {
	if c == nil || tabBarID == "" || tabID == "" {
		return
	}
	c.extTabPendingBarID = tabBarID
	c.extTabPendingID = tabID
	c.extTabPendingIdx = insertIdx
	c.extTabHighlightID = ""
	c.extTabInsertIdx = -1
	c.RequestRedraw()
}

func (c *Context) consumeExternalTabAttach(tabBarID string) (tabID string, insertIdx int, ok bool) {
	if c == nil || c.extTabPendingBarID != tabBarID || c.extTabPendingID == "" {
		return "", -1, false
	}
	tabID = c.extTabPendingID
	insertIdx = c.extTabPendingIdx
	c.extTabPendingBarID = ""
	c.extTabPendingID = ""
	c.extTabPendingIdx = -1
	return tabID, insertIdx, true
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
			if !e.inOverlay && c.Overlays.IsBlockedCurrent(x, y) {
				return nil
			}
			return e
		}
	}
	return nil
}

// tryDirectCanvasEvent пытается доставить событие напрямую в InteractiveCanvas с DirectInput=true
// без запуска полного цикла Render(). Возвращает (handled, needRedraw).
func (c *Context) tryDirectCanvasEvent(ev platform.Event) (bool, bool) {
	if len(c.directCanvases) == 0 {
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
			c.isInputFrame = true
		}
		return needRedraw && c.Canvas.Width > 0 && c.Canvas.Height > 0
	}

	redraw := false

	switch ev.Type {
	case platform.EventMouseMove, platform.EventMouseDown, platform.EventMouseUp, platform.EventMouseWheel,
		platform.EventKeyDown, platform.EventKeyUp, platform.EventChar:
		c.isInputFrame = true
	case platform.EventWakeUp:
		c.wakePending.Store(false)
		c.wakeOnlyFrame = true
		c.isInputFrame = false
		redraw = true
	default:
		c.isInputFrame = false
	}

	switch ev.Type {
	case platform.EventClose:
		c.stopAnimTimer()
		c.stopToastTimer()
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
		// wakeOnlyFrame и redraw уже выставлены в первом switch выше
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
		c.reorderActive = false
		c.reorderWidgetID = ""
		c.reorderSourceKey = ""
		c.reorderScrollAccum = 0
		redraw = true
	case platform.EventFocusLost:
		c.FocusID = ""
		c.MouseDown = false
		c.RightMouseDown = false
		c.MiddleMouseDown = false
		c.ActiveID = ""
		c.reorderActive = false
		c.reorderWidgetID = ""
		c.reorderSourceKey = ""
		c.reorderScrollAccum = 0
		redraw = true
	case platform.EventKeyDown:
		if (c.reorderActive || c.reorderWidgetID != "") && ev.KeyCode == KeyEscape {
			c.ResetReorder()
			redraw = true
		}
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
			c.presentFrame(true)
			c.syncAnimTimer()
			c.syncToastTimer()
			return
		}
		c.wakeOnlyFrame = false
	}

	// Быстрый путь для фонового кадра уведомлений Toast (EventWakeUp + Toasts):
	// перерисовываем только полоску времени активных тостов в локальной области toastsBoundingRect
	// без вызова drawFn(c) всего дерева виджетов окна 2K/4K.
	if c.canSkipFullFrameForToasts() {
		c.renderToastsOnly()
		if !c.needsRedraw {
			c.syncAnimTimer()
			c.syncToastTimer()
			return
		}
		c.wakeOnlyFrame = false
	}

	// Максимум 4 прохода: событие на 1-м проходе может изменить состояние и породить
	// новые виджеты с двухпроходным расчётом (Card AutoHeight, Justify, ScrollView).
	for pass := 0; pass < 4; pass++ {
		c.beginFrame()
		drawFn(c)
		c.endFrame()

		if !c.needsRedraw {
			break
		}
	}

	c.presentFrame(false)
	c.syncAnimTimer()
	c.syncToastTimer()
}

func (c *Context) canSkipFullFrameForToasts() bool {
	if !c.wakeOnlyFrame || c.paintRequested || len(c.toasts) == 0 {
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
	if c.Animator != nil && c.Animator.HasActive() {
		return false
	}
	if len(c.streamingCanvases) > 0 {
		return false
	}
	now := c.Now()
	for _, t := range c.toasts {
		if now.Sub(t.CreatedAt) >= t.Duration {
			return false
		}
	}
	if c.toastUnderlayRect.Empty() || len(c.toastUnderlay) == 0 {
		return false
	}
	return true
}

func (c *Context) renderToastsOnly() {
	c.wakeOnlyFrame = false
	c.restoreToastUnderlay()
	c.drawToasts()

	w, h := c.Canvas.Width, c.Canvas.Height
	sendRect := c.toastUnderlayRect.Intersect(render.Rect{X: 0, Y: 0, W: w, H: h})
	if sendRect.Empty() {
		return
	}

	if len(c.prevPresentPixels) == w*h*4 {
		dst := c.prevPresentPixels
		src := c.Canvas.Pixels
		bytesPerRow := sendRect.W * 4
		for y := 0; y < sendRect.H; y++ {
			off := ((sendRect.Y + y) * w + sendRect.X) * 4
			copy(dst[off:off+bytesPerRow], src[off:off+bytesPerRow])
		}
	}

	c.lastDirtyRects = append(c.lastDirtyRects[:0], sendRect)
	if dp, ok := c.Win.(platform.DirtyPresenter); ok && (sendRect.W < w || sendRect.H < h) {
		dp.PresentRect(c.Canvas.Pixels, w, h, sendRect.X, sendRect.Y, sendRect.W, sendRect.H)
	} else {
		c.Win.Present(c.Canvas.Pixels, w, h)
	}
	c.Canvas.ResetDirtyRects()
}

func (c *Context) saveToastUnderlay(r render.Rect) {
	if c.Canvas == nil {
		return
	}
	w, h := c.Canvas.Width, c.Canvas.Height
	clamped := r.Intersect(render.Rect{X: 0, Y: 0, W: w, H: h})
	if clamped.Empty() {
		c.toastUnderlayRect = render.Rect{}
		c.toastUnderlay = nil
		return
	}
	c.toastUnderlayRect = clamped
	bytesPerRow := clamped.W * 4
	need := bytesPerRow * clamped.H
	if cap(c.toastUnderlay) < need {
		c.toastUnderlay = make([]byte, need)
	} else {
		c.toastUnderlay = c.toastUnderlay[:need]
	}
	src := c.Canvas.Pixels
	for y := 0; y < clamped.H; y++ {
		srcOff := ((clamped.Y + y) * w + clamped.X) * 4
		dstOff := y * bytesPerRow
		copy(c.toastUnderlay[dstOff:dstOff+bytesPerRow], src[srcOff:srcOff+bytesPerRow])
	}
}

func (c *Context) restoreToastUnderlay() {
	if c.Canvas == nil || c.toastUnderlayRect.Empty() || len(c.toastUnderlay) == 0 {
		return
	}
	w := c.Canvas.Width
	clamped := c.toastUnderlayRect
	bytesPerRow := clamped.W * 4
	dst := c.Canvas.Pixels
	for y := 0; y < clamped.H; y++ {
		dstOff := ((clamped.Y + y) * w + clamped.X) * 4
		srcOff := y * bytesPerRow
		copy(dst[dstOff:dstOff+bytesPerRow], c.toastUnderlay[srcOff:srcOff+bytesPerRow])
	}
}

func (c *Context) syncToastTimer() {
	c.toastMu.Lock()
	defer c.toastMu.Unlock()
	need := len(c.toasts) > 0
	if need && !c.toastTimerActive {
		stopCh := make(chan struct{})
		c.toastStopCh = stopCh
		c.toastTimerActive = true
		win := c.Win
		go func() {
			ticker := time.NewTicker(50 * time.Millisecond) // 20 FPS
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
	} else if !need && c.toastTimerActive {
		close(c.toastStopCh)
		c.toastStopCh = nil
		c.toastTimerActive = false
	}
}

func (c *Context) stopToastTimer() {
	c.toastMu.Lock()
	defer c.toastMu.Unlock()
	if c.toastTimerActive {
		close(c.toastStopCh)
		c.toastStopCh = nil
		c.toastTimerActive = false
	}
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
	if len(c.toasts) > 0 {
		now := c.Now()
		for _, t := range c.toasts {
			if now.Sub(t.CreatedAt) >= t.Duration {
				// Просроченный тост требует полного кадра, чтобы стереть его область вне видеохолста.
				return false
			}
		}
	}
	if c.Animator != nil && c.Animator.HasActive() {
		return false
	}
	return true
}

func (c *Context) renderStreamingOnly() {
	c.streamingDirty = render.Rect{}
	shadowPad := c.Scale(32).Int()
	for _, entry := range c.streamingCanvases {
		if entry.onDraw == nil || entry.rect.Empty() {
			continue
		}
		focused := c.FocusID == entry.id
		c.Canvas.ResetDirtyRects()
		c.Canvas.PushClip(entry.rect)
		entry.onDraw(c.Canvas, entry.rect, focused)
		c.Canvas.PopClip()

		// Берём реально перезаписанную видеокадром область (например, fitRect без letterbox-полос),
		// чтобы не накладывать полупрозрачные тени/backdrop повторно на неперезаписанные полосы.
		drawnRect := c.Canvas.DirtyBounds().Intersect(entry.rect)
		if drawnRect.Empty() {
			drawnRect = entry.rect
		}

		needOverlays := c.Overlays.IntersectsAny(drawnRect, shadowPad)
		needToasts := len(c.toasts) > 0
		if needOverlays || needToasts {
			// Сохраняем временные буферы кадра, чтобы повторная отрисовка оверлеев/тостов
			// не накапливала записи между полными кадрами при длительном стриминге.
			savedHitLen := len(c.nextHitRegions)
			savedFocusLen := len(c.nextFocusOrder)
			savedDirectLen := len(c.nextDirectCanvases)
			savedStreamLen := len(c.nextStreamingCanvases)
			savedCursor := c.nextCursor
			savedCustomCursor := c.nextCustomCursor
			savedHotID := c.HotID
			savedLayoutSeq := c.layoutSeq
			savedScopeLen := len(c.idScopeStack)
			savedSeqLen := len(c.idSeqStack)
			var savedSeqTop int
			if savedSeqLen > 0 {
				savedSeqTop = c.idSeqStack[savedSeqLen-1]
			}

			c.Canvas.PushClip(drawnRect)
			if needOverlays {
				c.Overlays.RedrawIntersecting(c.Canvas, drawnRect, shadowPad, func(layoutSeq, idSeqTop int) {
					c.layoutSeq = layoutSeq
					c.idScopeStack = c.idScopeStack[:0]
					if len(c.idSeqStack) == 0 {
						c.idSeqStack = append(c.idSeqStack, idSeqTop)
					} else {
						c.idSeqStack = c.idSeqStack[:1]
						c.idSeqStack[0] = idSeqTop
					}
				})
			}
			if needToasts {
				c.drawToasts()
			}
			c.Canvas.PopClip()

			c.nextHitRegions = c.nextHitRegions[:savedHitLen]
			c.nextFocusOrder = c.nextFocusOrder[:savedFocusLen]
			c.nextDirectCanvases = c.nextDirectCanvases[:savedDirectLen]
			c.nextStreamingCanvases = c.nextStreamingCanvases[:savedStreamLen]
			c.nextCursor = savedCursor
			c.nextCustomCursor = savedCustomCursor
			c.HotID = savedHotID
			c.layoutSeq = savedLayoutSeq
			c.idScopeStack = c.idScopeStack[:savedScopeLen]
			c.idSeqStack = c.idSeqStack[:savedSeqLen]
			if savedSeqLen > 0 {
				c.idSeqStack[savedSeqLen-1] = savedSeqTop
			}
		}

		c.SetStreamingDirtyRect(drawnRect)
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

func (c *Context) presentFrame(streamingOnly bool) {
	w, h := c.Canvas.Width, c.Canvas.Height
	need := w * h * 4
	wakeOnly := c.wakeOnlyFrame
	c.wakeOnlyFrame = false

	forceFull := c.paintRequested || c.prevPresentW != w || c.prevPresentH != h || len(c.prevPresentPixels) != need
	c.paintRequested = false

	// Fast-path для потокового обновления InteractiveCanvas (например, видеокадр по EventWakeUp):
	// пропускаем computeDiffRect и копирование в prevPresentPixels, отправляя известную область напрямую.
	// При streamingOnly == true оверлеи поверх streamingDirty уже перерисованы, поэтому PresentRect
	// безопасен и при c.Overlays.HasActive().
	if !forceFull && wakeOnly && !c.streamingDirty.Empty() && !c.debugDirtyRect &&
		(streamingOnly || !c.Overlays.HasActive()) && (c.Animator == nil || !c.Animator.HasActive()) {
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
	if !c.wakeOnlyFrame && !c.MouseDown && !c.MouseReleased {
		c.reorderActive = false
		c.reorderWidgetID = ""
		c.reorderSourceKey = ""
		c.reorderScrollAccum = 0
	}
	c.HotID = ""
	c.nextCursor = platform.CursorArrow
	c.nextCustomCursor = nil
	c.needsRedraw = false
	c.nextCaptureKeys = false
	c.layoutSeq = 0
	c.idScopeStack = c.idScopeStack[:0]
	c.idSeqStack = append(c.idSeqStack[:0], 0)
	c.nextFocusOrder = c.nextFocusOrder[:0]
	c.nextHitRegions = c.nextHitRegions[:0]
	c.nextDirectCanvases = c.nextDirectCanvases[:0]
	c.nextStreamingCanvases = c.nextStreamingCanvases[:0]
	c.tabBars = c.tabBars[:0]
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
	c.Overlays.FlushWithStateAndCache(func(bounds render.Rect, nonBlocking bool) (int, int) {
		if !bounds.Empty() && !nonBlocking {
			c.registerHitRegion("__overlay:"+strconv.Itoa(ovIdx), bounds, platform.CursorArrow, false)
			if bounds.Contains(c.MouseX, c.MouseY) {
				c.nextCustomCursor = nil
			}
		}
		ovIdx++
		idSeqTop := 0
		if len(c.idSeqStack) > 0 {
			idSeqTop = c.idSeqStack[len(c.idSeqStack)-1]
		}
		return c.layoutSeq, idSeqTop
	}, func(it *overlayItem, bounds render.Rect) {
		mainCanvas := c.Canvas
		if mainCanvas != nil && mainCanvas.Width > 0 && mainCanvas.Height > 0 {
			cache := c.Overlays.AllocCache(mainCanvas.Width, mainCanvas.Height)
			if cache != nil {
				c.Canvas = cache
				it.draw()
				c.Canvas = mainCanvas
				it.cache = cache
				mainCanvas.CompositeFrom(cache, render.Rect{X: 0, Y: 0, W: mainCanvas.Width, H: mainCanvas.Height})
				return
			}
		}
		it.draw()
	})
	c.drawDragGhostBadge()
	if len(c.toasts) > 0 {
		c.saveToastUnderlay(c.toastsBoundingRect())
	} else {
		c.toastUnderlayRect = render.Rect{}
		c.toastUnderlay = nil
	}
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

	if !c.wakeOnlyFrame {
		if (c.MouseReleased || c.RightMouseReleased || c.MiddleMouseReleased) &&
			!c.MouseDown && !c.RightMouseDown && !c.MiddleMouseDown {
			c.ActiveID = ""
		}
		if c.MouseReleased || !c.MouseDown {
			c.reorderActive = false
			c.reorderWidgetID = ""
			c.reorderSourceIdx = -1
			c.reorderSourceKey = ""
			c.reorderScrollAccum = 0
			c.reorderSourceType = ""
			c.reorderSourceFn = nil
			c.reorderPayload = nil
		}
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
		c.isInputFrame = false
	} else {
		// На кадре EventWakeUp сбрасываем нажатие только если кнопка мыши физически не зажата
		if !c.MouseDown {
			c.MousePressed = false
		}
		if !c.RightMouseDown {
			c.RightMousePressed = false
		}
		if !c.MiddleMouseDown {
			c.MiddleMousePressed = false
		}
	}
	if c.nextCursor != c.lastCursor {
		c.Win.SetCursor(c.nextCursor)
		c.lastCursor = c.nextCursor
	}
	if c.nextCustomCursor != nil {
		c.Win.SetCustomCursor(c.nextCustomCursor)
		c.lastCustomCursor = c.nextCustomCursor
	} else if c.lastCustomCursor != nil {
		c.Win.SetCustomCursor(nil)
		c.lastCustomCursor = nil
	}
	c.captureKeysFocus = c.nextCaptureKeys
	c.prevFocusID = c.FocusID
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

// ArmReorder инициирует отслеживание возможного перетаскивания (Reorder) элемента при нажатии кнопки мыши.
func (c *Context) ArmReorder(widgetID string, fromIdx int, fromKey string) {
	c.reorderWidgetID = widgetID
	c.reorderSourceIdx = fromIdx
	c.reorderSourceKey = fromKey
	c.reorderStartX = c.MouseX
	c.reorderStartY = c.MouseY
	c.reorderActive = false
	c.reorderScrollAccum = 0
	c.reorderSourceType = ""
	c.reorderSourceFn = nil
	c.reorderPayload = nil
}

// ArmDrag инициирует отслеживание перетаскивания (внутреннего Reorder и/или внешнего DragSource).
func (c *Context) ArmDrag(widgetID string, fromIdx int, fromKey string, st WidgetStyle) {
	c.ArmReorder(widgetID, fromIdx, fromKey)
	c.reorderSourceType = st.DragSourceType
	c.reorderSourceFn = st.DragSourceFn
}

func (c *Context) updateReorderThreshold() {
	if c.reorderWidgetID == "" || !c.MouseDown || c.reorderActive {
		return
	}
	dx := c.MouseX - c.reorderStartX
	dy := c.MouseY - c.reorderStartY
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	threshold := c.Scale(4).Int()
	if dx > threshold || dy > threshold {
		c.reorderActive = true
		if c.reorderSourceType != "" && c.reorderPayload == nil {
			var p DragPayload
			if c.reorderSourceFn != nil {
				p = c.reorderSourceFn(c.reorderSourceIdx, c.reorderSourceKey)
				if p.Type == "" {
					p.Type = c.reorderSourceType
				}
			} else {
				p = DragPayload{Type: c.reorderSourceType}
			}
			c.reorderPayload = &p
		}
	}
}

// UpdateReorderDrag проверяет превышение порога движения (4 Dp) при зажатой кнопке мыши и активирует режим drag.
// Если перетаскивание активно, выставляет курсор platform.CursorHand и возвращает true.
func (c *Context) UpdateReorderDrag(widgetID string) bool {
	if c.reorderWidgetID == widgetID {
		c.updateReorderThreshold()
	}
	if c.reorderActive && c.reorderWidgetID == widgetID {
		c.nextCursor = platform.CursorHand
		return true
	}
	return false
}

// ReorderSource возвращает исходный индекс, ключ и признак активности drag для указанного виджета.
// Если перетаскивание не активно или принадлежит другому виджету, active равен false.
func (c *Context) ReorderSource(widgetID string) (fromIdx int, fromKey string, active bool) {
	if !c.reorderActive || c.reorderWidgetID != widgetID {
		return -1, "", false
	}
	return c.reorderSourceIdx, c.reorderSourceKey, true
}

// ResetReorder сбрасывает состояние перетаскивания (Reorder/Drop), восстанавливает курсор и запрашивает перерисовку.
func (c *Context) ResetReorder() {
	c.reorderActive = false
	c.reorderWidgetID = ""
	c.reorderSourceIdx = -1
	c.reorderSourceKey = ""
	c.reorderScrollAccum = 0
	c.reorderSourceType = ""
	c.reorderSourceFn = nil
	c.reorderPayload = nil
	c.nextCursor = platform.CursorArrow
	c.RequestRedraw()
}

func (c *Context) acceptsExternalDrop(targetID string, acceptTypes []string) (*DragPayload, bool) {
	if len(acceptTypes) == 0 || c.reorderWidgetID == "" || c.reorderWidgetID == targetID {
		return nil, false
	}
	c.updateReorderThreshold()
	if !c.reorderActive || c.reorderPayload == nil || c.reorderPayload.Type == "" {
		return nil, false
	}
	for _, t := range acceptTypes {
		if t == c.reorderPayload.Type {
			return c.reorderPayload, true
		}
	}
	return nil, false
}

func (c *Context) drawDragGhostBadge() {
	if !c.reorderActive || c.reorderPayload == nil || c.Font == nil {
		return
	}
	label := c.reorderPayload.Label
	if label == "" {
		label = c.reorderPayload.Type
	}
	if label == "" {
		return
	}
	pxSize := c.Scale(12).Int()
	tw, th := c.Font.MeasureText(label, pxSize)
	padX := c.Scale(8).Int()
	padY := c.Scale(4).Int()
	bw := tw + padX*2
	bh := th + padY*2
	bx := c.MouseX + c.Scale(12).Int()
	by := c.MouseY + c.Scale(12).Int()
	if bx+bw > c.Canvas.Width-4 {
		bx = c.Canvas.Width - bw - 4
	}
	if by+bh > c.Canvas.Height-4 {
		by = c.Canvas.Height - bh - 4
	}
	r := render.Rect{X: bx, Y: by, W: bw, H: bh}
	c.Canvas.DrawRoundedBox(r, c.ScaleF(6), c.Theme.Surface.WithAlpha(230), 1.0, c.Theme.Accent)
	c.Font.DrawText(c.Canvas, r.X+padX, r.Y+padY, label, pxSize, c.Theme.TextPrimary)
}

// DrawDropLine отрисовывает горизонтальную линию-индикатор места вставки Accent толщиной 2 Dp, центрированную по yPx.
func (c *Context) DrawDropLine(xPx, yPx, wPx int) {
	lineH := c.Scale(2).Int()
	if lineH < 1 {
		lineH = 1
	}
	lineY := yPx - lineH/2
	c.Canvas.FillRect(render.Rect{
		X: xPx,
		Y: lineY,
		W: wPx,
		H: lineH,
	}, c.Theme.Accent)
}

// DrawDropInside отрисовывает подсветку и рамку целевой строки цветом Accent при вложении (DropInside).
func (c *Context) DrawDropInside(r render.Rect) {
	c.Canvas.DrawRoundedBox(r, c.ScaleF(4), c.Theme.Accent.WithAlpha(35), c.ScaleF(1.5), c.Theme.Accent)
}

// ApplyReorderAutoScroll выполняет плавный авто-скролл контейнера при нахождении курсора у верхней или нижней границы.
func (c *Context) ApplyReorderAutoScroll(viewY, viewH int, si *scrollInfo, maxSc int) {
	if si == nil {
		return
	}
	if maxSc < 0 {
		maxSc = 0
	}
	if maxSc == 0 {
		si.y = 0
		c.reorderScrollAccum = 0
		return
	}
	edgeZone := c.Scale(24).Int()
	inTopZone := c.MouseY >= viewY && c.MouseY < viewY+edgeZone
	inBottomZone := c.MouseY > viewY+viewH-edgeZone && c.MouseY <= viewY+viewH

	dtSeconds := float32(1.0 / 60.0)
	if c.Animator != nil {
		if d := c.Animator.Delta(); d > 0 {
			dtSeconds = d
		}
	}
	if dtSeconds > 0.1 {
		dtSeconds = 0.1 // Защита от телепортации прокрутки при паузах
	}

	speed := c.ScaleF(160) * dtSeconds
	if inTopZone && si.y > 0 {
		c.reorderScrollAccum -= speed
		if c.reorderScrollAccum <= -1.0 {
			step := int(c.reorderScrollAccum)
			si.y += step
			c.reorderScrollAccum -= float32(step)
			c.RequestRedraw()
		}
	} else if inBottomZone && si.y < maxSc {
		c.reorderScrollAccum += speed
		if c.reorderScrollAccum >= 1.0 {
			step := int(c.reorderScrollAccum)
			si.y += step
			c.reorderScrollAccum -= float32(step)
			c.RequestRedraw()
		}
	} else {
		c.reorderScrollAccum = 0
	}

	// Обязательный clamp si.y
	if si.y < 0 {
		si.y = 0
		c.reorderScrollAccum = 0
	}
	if si.y > maxSc {
		si.y = maxSc
		c.reorderScrollAccum = 0
	}
}

// RowDropTarget рассчитывает строку-цель, зону сброса и ToIndex (индекс после удаления)
// для линейных списков, таблиц и плоского представления дерева.
// При allowInside == false строка делится пополам (DropAbove / DropBelow).
// При allowInside == true строка делится на 25% (DropAbove) / 50% (DropInside) / 25% (DropBelow).
// Является чистой детерминированной функцией без графических зависимостей.
func (c *Context) RowDropTarget(from, cursorY, rowsTop, itemH, count int, allowInside bool) (targetRow int, zone DropZone, toIdx int, noop bool) {
	if count <= 0 || itemH <= 0 {
		return 0, DropBelow, from, true
	}
	relY := cursorY - rowsTop
	targetRow = relY / itemH
	if targetRow < 0 {
		targetRow = 0
	}
	if targetRow >= count {
		targetRow = count - 1
	}
	targetRowY := rowsTop + targetRow*itemH
	if allowInside {
		q := itemH / 4
		if cursorY < targetRowY+q {
			zone = DropAbove
		} else if cursorY >= targetRowY+itemH-q {
			zone = DropBelow
		} else {
			zone = DropInside
		}
	} else {
		zone = DropBelow
		if cursorY < targetRowY+itemH/2 {
			zone = DropAbove
		}
	}
	if zone == DropInside {
		return targetRow, DropInside, targetRow, targetRow == from
	}
	insertIdx := targetRow
	if zone == DropBelow {
		insertIdx = targetRow + 1
	}
	toIdx = insertIdx
	if from < insertIdx {
		toIdx = insertIdx - 1
	}
	noop = (toIdx == from)
	return targetRow, zone, toIdx, noop
}

// HandleLinearReorder выполняет авто-скролл, подсчет целевой строки, отрисовку drop-линии и генерацию DragEvent при MouseReleased.
// Возвращает true, если произошел сброс (Drop) элемента.
func (c *Context) HandleLinearReorder(
	widgetID string,
	viewRect render.Rect,
	lineX, lineW int,
	itemH int,
	count int,
	si *scrollInfo,
	maxSc int,
	onReorder func(DragEvent),
	onDrop func(),
) bool {
	if !c.UpdateReorderDrag(widgetID) {
		return false
	}
	fromIdx, _, active := c.ReorderSource(widgetID)
	if !active || fromIdx < 0 || fromIdx >= count {
		c.ResetReorder()
		return false
	}

	c.ApplyReorderAutoScroll(viewRect.Y, viewRect.H, si, maxSc)

	if !c.IsInputBlocked() && viewRect.Contains(c.MouseX, c.MouseY) && c.MouseX < lineX+lineW {
		targetRow, zone, toIdx, noop := c.RowDropTarget(fromIdx, c.MouseY, viewRect.Y-si.y, itemH, count, false)
		targetRowY := viewRect.Y + targetRow*itemH - si.y
		lineY := targetRowY
		if zone == DropBelow {
			lineY += itemH
		}
		c.DrawDropLine(lineX, lineY, lineW)

		if c.MouseReleased {
			if onDrop != nil {
				onDrop()
			}
			if !noop && onReorder != nil {
				onReorder(DragEvent{
					FromWidget: widgetID,
					ToWidget:   widgetID,
					FromIndex:  fromIdx,
					ToIndex:    toIdx,
					Drop:       zone,
					Payload:    c.reorderPayload,
				})
			}
			c.ResetReorder()
			return true
		}
	}
	return false
}

// HandleTreeReorder выполняет авто-скролл, расчёт целевого узла и зоны сброса (через RowDropTarget),
// проверку правил сиблингов / защиту от циклов, отрисовку индикатора (линия или DropInside) и генерацию DragEvent.
// Возвращает true, если произошёл валидный сброс (Drop) узла.
func (c *Context) HandleTreeReorder(
	widgetID string,
	viewRect render.Rect,
	contentW int,
	itemH int,
	flat []FlatTreeItem,
	si *scrollInfo,
	maxSc int,
	allowReparent bool,
	onReorder func(DragEvent),
	onDrop func(),
) bool {
	if !c.UpdateReorderDrag(widgetID) {
		return false
	}
	_, fromKey, active := c.ReorderSource(widgetID)
	if !active {
		c.ResetReorder()
		return false
	}
	sourceIdx := -1
	for idx, it := range flat {
		if it.Node != nil && it.Node.ID == fromKey {
			sourceIdx = idx
			break
		}
	}
	if sourceIdx < 0 {
		c.ResetReorder()
		return false
	}
	sourceItem := flat[sourceIdx]

	c.ApplyReorderAutoScroll(viewRect.Y, viewRect.H, si, maxSc)

	if !c.IsInputBlocked() && viewRect.Contains(c.MouseX, c.MouseY) && c.MouseX < viewRect.X+contentW {
		targetIdx, zone, _, _ := c.RowDropTarget(sourceIdx, c.MouseY, viewRect.Y-si.y, itemH, len(flat), allowReparent)
		if targetIdx >= 0 && targetIdx < len(flat) {
			targetItem := flat[targetIdx]
			targetNode := targetItem.Node
			targetRowY := viewRect.Y + targetIdx*itemH - si.y

			var isValidDrop bool
			if allowReparent {
				// Смена родителя и DropInside с защитой от циклов и no-op на текущем родителе
				isValidDrop = targetNode.ID != sourceItem.Node.ID &&
					!isTreeDescendant(sourceItem.Node, targetNode.ID) &&
					!(zone == DropInside && targetNode.ID == sourceItem.ParentID)
			} else {
				// Без AllowReparent перетаскивание узлов разрешено строго между сиблингами одного родителя
				isValidDrop = targetItem.ParentID == sourceItem.ParentID && targetNode.ID != sourceItem.Node.ID
			}

			if isValidDrop {
				if zone == DropInside {
					c.DrawDropInside(render.Rect{
						X: viewRect.X,
						Y: targetRowY,
						W: contentW,
						H: itemH,
					})
				} else {
					lineY := targetRowY
					if zone == DropBelow {
						lineY += itemH
					}
					treeIndentPx := c.Scale(TreeIndent).Int()
					treeInsetPx := c.Scale(TreeIndicatorInset).Int()
					targetIndent := targetItem.Depth*treeIndentPx + treeInsetPx
					lineX := viewRect.X + targetIndent
					lineW := contentW - targetIndent
					if lineW > 0 {
						c.DrawDropLine(lineX, lineY, lineW)
					}
				}

				if c.MouseReleased {
					if onDrop != nil {
						onDrop()
					}
					dragEv := DragEvent{
						FromWidget: widgetID,
						ToWidget:   widgetID,
						FromKey:    sourceItem.Node.ID,
						ToKey:      targetNode.ID,
						FromIndex:  -1,
						ToIndex:    -1,
						Drop:       zone,
						Payload:    c.reorderPayload,
					}
					if onReorder != nil {
						onReorder(dragEv)
					}
					c.ResetReorder()
					return true
				}
			} else if c.MouseReleased {
				c.ResetReorder()
			}
		}
	}
	return false
}

// HandleLinearDrop обрабатывает внешний кросс-виджетный перенос в список или таблицу (ListView, ListViewEx, Table, TableEx).
func (c *Context) HandleLinearDrop(
	widgetID string,
	viewRect render.Rect,
	lineX, lineW int,
	itemH int,
	count int,
	si *scrollInfo,
	maxSc int,
	acceptTypes []string,
	onDrop func(DragEvent),
) bool {
	payload, ok := c.acceptsExternalDrop(widgetID, acceptTypes)
	if !ok {
		return false
	}
	if c.IsInputBlocked() || !viewRect.Contains(c.MouseX, c.MouseY) || c.MouseX >= lineX+lineW {
		return false
	}
	c.nextCursor = platform.CursorHand
	c.ApplyReorderAutoScroll(viewRect.Y, viewRect.H, si, maxSc)

	var (
		toIdx int
		zone  DropZone
	)
	if count <= 0 {
		zone = DropInside
		toIdx = 0
		c.DrawDropInside(viewRect)
	} else {
		targetRow, z, _, _ := c.RowDropTarget(-1, c.MouseY, viewRect.Y-si.y, itemH, count, false)
		zone = z
		toIdx = targetRow
		if zone == DropBelow {
			toIdx = targetRow + 1
		}
		targetRowY := viewRect.Y + targetRow*itemH - si.y
		lineY := targetRowY
		if zone == DropBelow {
			lineY += itemH
		}
		c.DrawDropLine(lineX, lineY, lineW)
	}

	if c.MouseReleased {
		pCopy := *payload
		ev := DragEvent{
			FromWidget: c.reorderWidgetID,
			ToWidget:   widgetID,
			FromKey:    c.reorderSourceKey,
			FromIndex:  c.reorderSourceIdx,
			ToIndex:    toIdx,
			Drop:       zone,
			Payload:    &pCopy,
		}
		if onDrop != nil {
			onDrop(ev)
		}
		c.ResetReorder()
		return true
	}
	return false
}

// HandleTreeDrop обрабатывает внешний кросс-виджетный перенос в дерево (TreeView, TreeViewEx).
func (c *Context) HandleTreeDrop(
	widgetID string,
	viewRect render.Rect,
	contentW int,
	itemH int,
	flat []FlatTreeItem,
	si *scrollInfo,
	maxSc int,
	allowReparent bool,
	acceptTypes []string,
	onDrop func(DragEvent),
) bool {
	payload, ok := c.acceptsExternalDrop(widgetID, acceptTypes)
	if !ok {
		return false
	}
	if c.IsInputBlocked() || !viewRect.Contains(c.MouseX, c.MouseY) || c.MouseX >= viewRect.X+contentW {
		return false
	}
	c.nextCursor = platform.CursorHand
	c.ApplyReorderAutoScroll(viewRect.Y, viewRect.H, si, maxSc)

	var (
		toKey string
		zone  DropZone
	)
	if len(flat) == 0 {
		zone = DropInside
		c.DrawDropInside(viewRect)
	} else {
		targetIdx, z, _, _ := c.RowDropTarget(-1, c.MouseY, viewRect.Y-si.y, itemH, len(flat), allowReparent)
		if targetIdx < 0 || targetIdx >= len(flat) {
			return false
		}
		targetItem := flat[targetIdx]
		toKey = targetItem.Node.ID
		zone = z
		targetRowY := viewRect.Y + targetIdx*itemH - si.y
		if zone == DropInside {
			c.DrawDropInside(render.Rect{
				X: viewRect.X,
				Y: targetRowY,
				W: contentW,
				H: itemH,
			})
		} else {
			lineY := targetRowY
			if zone == DropBelow {
				lineY += itemH
			}
			treeIndentPx := c.Scale(TreeIndent).Int()
			treeInsetPx := c.Scale(TreeIndicatorInset).Int()
			targetIndent := targetItem.Depth*treeIndentPx + treeInsetPx
			lineX := viewRect.X + targetIndent
			lineW := contentW - targetIndent
			if lineW > 0 {
				c.DrawDropLine(lineX, lineY, lineW)
			}
		}
	}

	if c.MouseReleased {
		pCopy := *payload
		ev := DragEvent{
			FromWidget: c.reorderWidgetID,
			ToWidget:   widgetID,
			FromKey:    c.reorderSourceKey,
			ToKey:      toKey,
			FromIndex:  c.reorderSourceIdx,
			ToIndex:    -1,
			Drop:       zone,
			Payload:    &pCopy,
		}
		if onDrop != nil {
			onDrop(ev)
		}
		c.ResetReorder()
		return true
	}
	return false
}
