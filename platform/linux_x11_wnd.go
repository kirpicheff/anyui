//go:build linux

package platform

import (
	"encoding/binary"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xproto"
	"golang.org/x/sys/unix"
)

const (
	motifHintsProp     = "_MOTIF_WM_HINTS"
	netWMStrut         = "_NET_WM_STATE"
	netWMStateMaxHorz  = "_NET_WM_STATE_MAXIMIZED_HORZ"
	netWMStateMaxVert  = "_NET_WM_STATE_MAXIMIZED_VERT"
	netWMStateFull     = "_NET_WM_STATE_FULLSCREEN"
	netWMStateHidden   = "_NET_WM_STATE_HIDDEN"
	netWMMoveResize    = "_NET_WM_MOVERESIZE"
	netWMIcon          = "_NET_WM_ICON"
	netActiveWindow    = "_NET_ACTIVE_WINDOW"
	gtkThemeVariant    = "_GTK_THEME_VARIANT"
	anyuiWakeupAtom    = "_ANYUI_WAKEUP"
	wmProtocolsAtom    = "WM_PROTOCOLS"
	wmDeleteWindowAtom = "WM_DELETE_WINDOW"
)

// x11Display инкапсулирует соединение с X11-сервером и общий цикл событий.
type x11Display struct {
	mu      sync.Mutex
	conn    *xgb.Conn
	screen  *xproto.ScreenInfo
	windows map[xproto.Window]*x11Window

	atomsMu sync.Mutex
	atoms   map[string]xproto.Atom

	shmAvail bool
	wakeWin  xproto.Window
	wakeAtom xproto.Atom
	running  atomic.Bool

	cursors    map[CursorType]xproto.Cursor
	cursorFont xproto.Font

	keyMap *x11KeyMap
	clip   *x11Clipboard
	tray   *x11Tray
}

var (
	x11DispOnce sync.Once
	x11Disp     *x11Display
	x11DispErr  error
)

func getX11Display() (*x11Display, error) {
	x11DispOnce.Do(func() {
		conn, err := xgb.NewConn()
		if err != nil {
			x11DispErr = err
			return
		}
		setup := xproto.Setup(conn)
		screen := setup.DefaultScreen(conn)

		d := &x11Display{
			conn:    conn,
			screen:  screen,
			windows: make(map[xproto.Window]*x11Window),
			atoms:   make(map[string]xproto.Atom),
			cursors: make(map[CursorType]xproto.Cursor),
		}

		// Проверяем доступность MIT-SHM
		if err := shm.Init(conn); err == nil {
			if _, err := shm.QueryVersion(conn).Reply(); err == nil {
				d.shmAvail = true
			}
		}

		// Скрытое служебное окно для пробуждения (WakeUp) и буфера обмена
		wakeWin, err := xproto.NewWindowId(conn)
		if err == nil {
			_ = xproto.CreateWindowChecked(
				conn,
				xproto.WindowClassCopyFromParent,
				wakeWin,
				screen.Root,
				-10, -10, 1, 1, 0,
				xproto.WindowClassInputOnly,
				screen.RootVisual,
				0, nil,
			).Check()
			d.wakeWin = wakeWin
		}
		d.wakeAtom = d.internAtom(anyuiWakeupAtom)
		d.initCursors()
		d.keyMap = newX11KeyMap(conn, setup)
		d.clip = newX11Clipboard(d)

		x11Disp = d
	})
	return x11Disp, x11DispErr
}

func (d *x11Display) internAtom(name string) xproto.Atom {
	d.atomsMu.Lock()
	if atom, ok := d.atoms[name]; ok {
		d.atomsMu.Unlock()
		return atom
	}
	d.atomsMu.Unlock()

	reply, err := xproto.InternAtom(d.conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0
	}

	d.atomsMu.Lock()
	d.atoms[name] = reply.Atom
	d.atomsMu.Unlock()
	return reply.Atom
}

func (d *x11Display) initCursors() {
	fontID, err := xproto.NewFontId(d.conn)
	if err != nil {
		return
	}
	if err := xproto.OpenFontChecked(d.conn, fontID, uint16(len("cursor")), "cursor").Check(); err != nil {
		return
	}
	d.cursorFont = fontID

	// Стандартные индексы глифов из X11/cursorfont.h
	glyphs := map[CursorType]uint16{
		CursorArrow:      68,  // XC_left_ptr
		CursorHand:       60,  // XC_hand2
		CursorIBeam:      152, // XC_xterm
		CursorResizeH:    108, // XC_sb_h_double_arrow
		CursorResizeV:    116, // XC_sb_v_double_arrow
		CursorResizeNWSE: 14,  // XC_bottom_right_corner
		CursorResizeNESW: 12,  // XC_bottom_left_corner
		CursorNotAllowed: 88,  // XC_pirate
		CursorCrosshair:  34,  // XC_crosshair
		CursorMove:       52,  // XC_fleur
	}

	for ct, glyph := range glyphs {
		cid, err := xproto.NewCursorId(d.conn)
		if err != nil {
			continue
		}
		err = xproto.CreateGlyphCursorChecked(
			d.conn, cid, fontID, fontID,
			glyph, glyph+1,
			0, 0, 0, // foreground black
			65535, 65535, 65535, // background white
		).Check()
		if err == nil {
			d.cursors[ct] = cid
		}
	}
}

func (d *x11Display) wakeUp() {
	if d.wakeWin == 0 || d.wakeAtom == 0 {
		return
	}
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: d.wakeWin,
		Type:   d.wakeAtom,
		Data:   xproto.ClientMessageDataUnionData32New([]uint32{0, 0, 0, 0, 0}),
	}
	_ = xproto.SendEventChecked(d.conn, false, d.wakeWin, 0, string(ev.Bytes())).Check()
}

// RunEventLoop запускает главный цикл событий X11.
func RunEventLoop(onStart func() error) error {
	d, err := getX11Display()
	if err != nil {
		return err
	}
	d.running.Store(true)
	defer d.running.Store(false)

	if onStart != nil {
		if err := onStart(); err != nil {
			return err
		}
	}

	for {
		d.mu.Lock()
		winCount := len(d.windows)
		d.mu.Unlock()
		if winCount == 0 {
			return nil
		}

		ev, err := d.conn.WaitForEvent()
		if err != nil {
			return err
		}
		if ev == nil {
			return nil
		}
		d.dispatchEvent(ev)

		// Вычитываем все накопившиеся события без блокировки перед отрисовкой
		for {
			nextEv, pollErr := d.conn.PollForEvent()
			if pollErr != nil || nextEv == nil {
				break
			}
			d.dispatchEvent(nextEv)
		}

		// Отрисовываем окна, которым требуется перерисовка
		d.flushPaints()
	}
}

func (d *x11Display) flushPaints() {
	d.mu.Lock()
	wins := make([]*x11Window, 0, len(d.windows))
	for _, w := range d.windows {
		wins = append(wins, w)
	}
	d.mu.Unlock()

	for _, w := range wins {
		if w.needsPaint.Swap(false) && w.handler != nil && !w.closed.Load() {
			w.handler(Event{Type: EventPaint})
		}
	}
}

func (d *x11Display) dispatchEvent(ev xgb.Event) {
	// Сначала пробуем отдать событие буферу обмена (SelectionRequest / SelectionNotify)
	if d.clip != nil && d.clip.handleEvent(ev) {
		return
	}
	// Затем системному трею, если он активен
	if d.tray != nil && d.tray.handleEvent(ev) {
		return
	}

	switch e := ev.(type) {
	case xproto.MappingNotifyEvent:
		d.keyMap.refresh(d.conn, xproto.Setup(d.conn))
		return

	case xproto.ExposeEvent:
		if w := d.windowByID(e.Window); w != nil && e.Count == 0 {
			w.needsPaint.Store(true)
		}

	case xproto.ConfigureNotifyEvent:
		if w := d.windowByID(e.Window); w != nil {
			w.handleConfigure(e)
		}

	case xproto.ClientMessageEvent:
		if e.Window == d.wakeWin && e.Type == d.wakeAtom {
			if w := d.anyWindow(); w != nil && w.handler != nil {
				w.handler(Event{Type: EventWakeUp})
			}
			return
		}
		if w := d.windowByID(e.Window); w != nil {
			w.handleClientMessage(e)
		}

	case xproto.FocusOutEvent:
		if w := d.windowByID(e.Event); w != nil && w.handler != nil {
			w.handler(Event{Type: EventFocusLost})
		}

	case xproto.MotionNotifyEvent:
		if w := d.windowByID(e.Event); w != nil && w.handler != nil {
			w.handler(Event{
				Type: EventMouseMove,
				X:    int(e.EventX),
				Y:    int(e.EventY),
				Mods: modifiersFromX11State(e.State),
			})
		}

	case xproto.ButtonPressEvent:
		if w := d.windowByID(e.Event); w != nil {
			w.handleButtonPress(e)
		}

	case xproto.ButtonReleaseEvent:
		if w := d.windowByID(e.Event); w != nil {
			w.handleButtonRelease(e)
		}

	case xproto.KeyPressEvent:
		if w := d.windowByID(e.Event); w != nil {
			w.handleKeyPress(e)
		}

	case xproto.KeyReleaseEvent:
		if w := d.windowByID(e.Event); w != nil {
			w.handleKeyRelease(e)
		}

	case xproto.SelectionNotifyEvent:
		if w := d.windowByID(e.Requestor); w != nil {
			w.handleXDNDSelectionNotify(e)
		}

	case xproto.DestroyNotifyEvent:
		if w := d.windowByID(e.Window); w != nil {
			w.markClosed()
		}
	}
}

func (d *x11Display) windowByID(id xproto.Window) *x11Window {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.windows[id]
}

func (d *x11Display) anyWindow() *x11Window {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, w := range d.windows {
		return w
	}
	return nil
}

// x11Window реализует platform.Window для X11.
type x11Window struct {
	disp    *x11Display
	cfg     WindowConfig
	win     xproto.Window
	gc      xproto.Gcontext
	depth   byte
	handler func(Event)

	mu         sync.Mutex
	width      int
	height     int
	posX       int
	posY       int
	scale      float32
	maximized  bool
	minimized  bool
	fullscreen bool
	onClose    func() bool

	needsPaint atomic.Bool
	closed     atomic.Bool

	// MIT-SHM буфер
	shmSeg  shm.Seg
	shmID   int
	shmData []byte
	shmW    int
	shmH    int

	// XDND состояние
	xdndSource xproto.Window
	xdndTypes  []xproto.Atom
}

// NewWindow создаёт новое нативное окно X11.
func NewWindow(cfg WindowConfig) (Window, error) {
	d, err := getX11Display()
	if err != nil {
		return nil, err
	}

	scale := detectX11Scale(d)
	w := &x11Window{
		disp:  d,
		cfg:   cfg,
		depth: d.screen.RootDepth,
		scale: scale,
	}

	physW := int(math.Round(float64(cfg.Width) * float64(scale)))
	physH := int(math.Round(float64(cfg.Height) * float64(scale)))
	if physW <= 0 {
		physW = 800
	}
	if physH <= 0 {
		physH = 600
	}
	posX, posY := 100, 100
	if cfg.InitialState != nil && isValidWindowState(*cfg.InitialState) {
		posX = cfg.InitialState.X
		posY = cfg.InitialState.Y
		physW = cfg.InitialState.Width
		physH = cfg.InitialState.Height
	}
	w.width = physW
	w.height = physH
	w.posX = posX
	w.posY = posY

	winID, err := xproto.NewWindowId(d.conn)
	if err != nil {
		return nil, err
	}
	w.win = winID

	eventMask := uint32(
		xproto.EventMaskExposure |
			xproto.EventMaskStructureNotify |
			xproto.EventMaskKeyPress |
			xproto.EventMaskKeyRelease |
			xproto.EventMaskButtonPress |
			xproto.EventMaskButtonRelease |
			xproto.EventMaskPointerMotion |
			xproto.EventMaskFocusChange,
	)

	err = xproto.CreateWindowChecked(
		d.conn,
		w.depth,
		winID,
		d.screen.Root,
		int16(posX), int16(posY),
		uint16(physW), uint16(physH),
		0,
		xproto.WindowClassInputOutput,
		d.screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{d.screen.BlackPixel, eventMask},
	).Check()
	if err != nil {
		return nil, err
	}

	gcID, err := xproto.NewGcontextId(d.conn)
	if err != nil {
		_ = xproto.DestroyWindowChecked(d.conn, winID).Check()
		return nil, err
	}
	if err := xproto.CreateGCChecked(d.conn, gcID, xproto.Drawable(winID), 0, nil).Check(); err != nil {
		_ = xproto.DestroyWindowChecked(d.conn, winID).Check()
		return nil, err
	}
	w.gc = gcID

	// Регистрируем WM_DELETE_WINDOW
	wmProt := d.internAtom(wmProtocolsAtom)
	wmDel := d.internAtom(wmDeleteWindowAtom)
	if wmProt != 0 && wmDel != 0 {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(wmDel))
		_ = xproto.ChangePropertyChecked(d.conn, xproto.PropModeReplace, winID, wmProt, xproto.AtomAtom, 32, 1, buf).Check()
	}

	// Заголовок окна
	if cfg.Title != "" {
		w.SetTitle(cfg.Title)
	}

	// Минимальный размер (WM_NORMAL_HINTS)
	if cfg.MinWidth > 0 || cfg.MinHeight > 0 {
		w.applyMinSizeHints()
	}

	// Безрамочный режим (_MOTIF_WM_HINTS)
	if cfg.Frameless {
		w.applyFramelessHints()
	}

	// Тёмный заголовок (GTK/GNOME)
	if cfg.DarkTitle {
		w.SetDarkTitle(true)
	}

	// Иконка окна (_NET_WM_ICON)
	if len(cfg.IconRGBA) > 0 && cfg.IconWidth > 0 && cfg.IconHeight > 0 {
		_ = w.SetIcon(cfg.IconRGBA, cfg.IconWidth, cfg.IconHeight)
	}

	// Включаем XDND если разрешено перетаскивание файлов
	if cfg.AcceptDropFiles {
		w.enableXDND()
	}

	// Привязка к родительскому окну (WM_TRANSIENT_FOR)
	if cfg.Parent != nil && cfg.Parent.NativeHandle() != 0 {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, uint32(cfg.Parent.NativeHandle()))
		_ = xproto.ChangePropertyChecked(d.conn, xproto.PropModeReplace, winID, xproto.AtomWmTransientFor, xproto.AtomWindow, 32, 1, buf).Check()
	}

	d.mu.Lock()
	d.windows[winID] = w
	d.mu.Unlock()

	if cfg.InitialState != nil && cfg.InitialState.Maximized {
		w.Maximize()
	}

	return w, nil
}

func (w *x11Window) Run(handler func(Event)) error {
	w.handler = handler
	w.Show()
	w.needsPaint.Store(true)
	return RunEventLoop(nil)
}

func (w *x11Window) Open(handler func(Event)) error {
	w.handler = handler
	w.Show()
	w.needsPaint.Store(true)
	w.WakeUp()
	return nil
}

func (w *x11Window) Present(pixels []byte, width, height int) {
	w.PresentRect(pixels, width, height, 0, 0, width, height)
}

func (w *x11Window) PresentRect(pixels []byte, width, height, rx, ry, rw, rh int) {
	if w.closed.Load() || width <= 0 || height <= 0 || len(pixels) < width*height*4 {
		return
	}
	if rx < 0 {
		rw += rx
		rx = 0
	}
	if ry < 0 {
		rh += ry
		ry = 0
	}
	if rx+rw > width {
		rw = width - rx
	}
	if ry+rh > height {
		rh = height - ry
	}
	if rw <= 0 || rh <= 0 {
		return
	}

	if w.disp.shmAvail && w.ensureShmBuffer(width, height) {
		stride := width * 4
		for y := ry; y < ry+rh; y++ {
			srcOff := y*stride + rx*4
			copy(w.shmData[srcOff:srcOff+rw*4], pixels[srcOff:srcOff+rw*4])
		}
		shm.PutImage(
			w.disp.conn,
			xproto.Drawable(w.win),
			w.gc,
			uint16(width), uint16(height),
			uint16(rx), uint16(ry),
			uint16(rw), uint16(rh),
			int16(rx), int16(ry),
			w.depth,
			xproto.ImageFormatZPixmap,
			0,
			w.shmSeg,
			0,
		)
		return
	}

	// Fallback на стандартный PutImage (полосами, чтобы не превысить лимит запроса X11)
	maxBytes := int(xproto.Setup(w.disp.conn).MaximumRequestLength)*4 - 64
	if maxBytes < 4096 {
		maxBytes = 4096
	}
	rowBytes := rw * 4
	maxRows := maxBytes / rowBytes
	if maxRows < 1 {
		maxRows = 1
	}

	srcStride := width * 4
	strip := make([]byte, maxRows*rowBytes)
	for y := ry; y < ry+rh; y += maxRows {
		rows := maxRows
		if y+rows > ry+rh {
			rows = ry + rh - y
		}
		for row := 0; row < rows; row++ {
			srcOff := (y+row)*srcStride + rx*4
			copy(strip[row*rowBytes:(row+1)*rowBytes], pixels[srcOff:srcOff+rowBytes])
		}
		xproto.PutImage(
			w.disp.conn,
			xproto.ImageFormatZPixmap,
			xproto.Drawable(w.win),
			w.gc,
			uint16(rw), uint16(rows),
			int16(rx), int16(y),
			0,
			w.depth,
			strip[:rows*rowBytes],
		)
	}
}

func (w *x11Window) ensureShmBuffer(width, height int) bool {
	if w.shmSeg != 0 && w.shmW == width && w.shmH == height {
		return true
	}
	w.freeShmBuffer()

	size := width * height * 4
	shmID, err := unix.SysvShmGet(0, size, unix.IPC_CREAT|unix.IPC_EXCL|0600)
	if err != nil {
		return false
	}
	data, err := unix.SysvShmAttach(shmID, 0, 0)
	if err != nil {
		_, _ = unix.SysvShmCtl(shmID, unix.IPC_RMID, nil)
		return false
	}
	segID, err := shm.NewSegId(w.disp.conn)
	if err != nil {
		_ = unix.SysvShmDetach(data)
		_, _ = unix.SysvShmCtl(shmID, unix.IPC_RMID, nil)
		return false
	}
	if err := shm.AttachChecked(w.disp.conn, segID, uint32(shmID), false).Check(); err != nil {
		_ = unix.SysvShmDetach(data)
		_, _ = unix.SysvShmCtl(shmID, unix.IPC_RMID, nil)
		return false
	}
	// Помечаем сегмент для автоматического удаления после отключения обоих процессов
	_, _ = unix.SysvShmCtl(shmID, unix.IPC_RMID, nil)

	w.shmSeg = segID
	w.shmID = shmID
	w.shmData = data
	w.shmW = width
	w.shmH = height
	return true
}

func (w *x11Window) freeShmBuffer() {
	if w.shmSeg != 0 {
		_ = shm.DetachChecked(w.disp.conn, w.shmSeg).Check()
		w.shmSeg = 0
	}
	if w.shmData != nil {
		_ = unix.SysvShmDetach(w.shmData)
		w.shmData = nil
	}
	w.shmID = 0
	w.shmW = 0
	w.shmH = 0
}

func (w *x11Window) Invalidate() {
	w.needsPaint.Store(true)
	w.WakeUp()
}

func (w *x11Window) WakeUp() {
	w.disp.wakeUp()
}

func (w *x11Window) Show() {
	_ = xproto.MapWindowChecked(w.disp.conn, w.win).Check()
}

func (w *x11Window) Hide() {
	_ = xproto.UnmapWindowChecked(w.disp.conn, w.win).Check()
}

func (w *x11Window) Close() {
	if w.closed.Load() {
		return
	}
	if w.onClose != nil && !w.onClose() {
		return
	}
	w.markClosed()
	if w.handler != nil {
		w.handler(Event{Type: EventClose})
	}
}

func (w *x11Window) markClosed() {
	if w.closed.Swap(true) {
		return
	}
	w.freeShmBuffer()
	if w.gc != 0 {
		_ = xproto.FreeGCChecked(w.disp.conn, w.gc).Check()
		w.gc = 0
	}
	if w.win != 0 {
		_ = xproto.DestroyWindowChecked(w.disp.conn, w.win).Check()
		w.disp.mu.Lock()
		delete(w.disp.windows, w.win)
		w.disp.mu.Unlock()
		w.win = 0
	}
}

func (w *x11Window) SetTitle(title string) {
	b := []byte(title)
	_ = xproto.ChangePropertyChecked(w.disp.conn, xproto.PropModeReplace, w.win, xproto.AtomWmName, xproto.AtomString, 8, uint32(len(b)), b).Check()
	netName := w.disp.internAtom("_NET_WM_NAME")
	utf8 := w.disp.internAtom("UTF8_STRING")
	if netName != 0 && utf8 != 0 {
		_ = xproto.ChangePropertyChecked(w.disp.conn, xproto.PropModeReplace, w.win, netName, utf8, 8, uint32(len(b)), b).Check()
	}
}

func (w *x11Window) SetCursor(cursor CursorType) {
	if cid, ok := w.disp.cursors[cursor]; ok {
		_ = xproto.ChangeWindowAttributesChecked(w.disp.conn, w.win, xproto.CwCursor, []uint32{uint32(cid)}).Check()
	}
}

func (w *x11Window) Size() (int, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.width, w.height
}

func (w *x11Window) Scale() float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.scale
}

func (w *x11Window) Minimize() {
	// ICCCM: отправка WM_CHANGE_STATE (IconicState = 3)
	atom := w.disp.internAtom("WM_CHANGE_STATE")
	if atom == 0 {
		return
	}
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: w.win,
		Type:   atom,
		Data:   xproto.ClientMessageDataUnionData32New([]uint32{3, 0, 0, 0, 0}),
	}
	_ = xproto.SendEventChecked(
		w.disp.conn, false, w.disp.screen.Root,
		xproto.EventMaskSubstructureRedirect|xproto.EventMaskSubstructureNotify,
		string(ev.Bytes()),
	).Check()
	w.mu.Lock()
	w.minimized = true
	w.mu.Unlock()
}

func (w *x11Window) Maximize() {
	w.sendNetWMState(1, netWMStateMaxHorz, netWMStateMaxVert)
	w.mu.Lock()
	w.maximized = true
	w.mu.Unlock()
}

func (w *x11Window) Restore() {
	w.mu.Lock()
	wasMin := w.minimized
	w.minimized = false
	w.maximized = false
	w.fullscreen = false
	w.mu.Unlock()

	if wasMin {
		w.Show()
	}
	w.sendNetWMState(0, netWMStateMaxHorz, netWMStateMaxVert)
	w.sendNetWMState(0, netWMStateFull, "")
}

func (w *x11Window) IsMaximized() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.maximized
}

func (w *x11Window) IsMinimized() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.minimized
}

func (w *x11Window) SetFullscreen(fullscreen bool) {
	action := uint32(0)
	if fullscreen {
		action = 1
	}
	w.sendNetWMState(action, netWMStateFull, "")
	w.mu.Lock()
	w.fullscreen = fullscreen
	w.mu.Unlock()
}

func (w *x11Window) IsFullscreen() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.fullscreen
}

func (w *x11Window) Position() (int, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.posX, w.posY
}

func (w *x11Window) SetPosition(x, y int) {
	_ = xproto.ConfigureWindowChecked(
		w.disp.conn,
		w.win,
		xproto.ConfigWindowX|xproto.ConfigWindowY,
		[]uint32{uint32(int32(x)), uint32(int32(y))},
	).Check()
}

func (w *x11Window) SetSize(width, height int) {
	physW := int(math.Round(float64(width) * float64(w.scale)))
	physH := int(math.Round(float64(height) * float64(w.scale)))
	if physW < 1 {
		physW = 1
	}
	if physH < 1 {
		physH = 1
	}
	_ = xproto.ConfigureWindowChecked(
		w.disp.conn,
		w.win,
		xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
		[]uint32{uint32(physW), uint32(physH)},
	).Check()
}

func (w *x11Window) SetOnClose(fn func() bool) {
	w.onClose = fn
}

func (w *x11Window) NativeHandle() uintptr {
	return uintptr(w.win)
}

func (w *x11Window) SetCapture() {
	mask := uint16(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion)
	_, _ = xproto.GrabPointer(w.disp.conn, false, w.win, mask, xproto.GrabModeAsync, xproto.GrabModeAsync, 0, 0, xproto.TimeCurrentTime).Reply()
}

func (w *x11Window) ReleaseCapture() {
	_ = xproto.UngrabPointerChecked(w.disp.conn, xproto.TimeCurrentTime).Check()
}

func (w *x11Window) DoubleClickTimeMs() int {
	return 400
}

func (w *x11Window) State() WindowState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return WindowState{
		X:         w.posX,
		Y:         w.posY,
		Width:     w.width,
		Height:    w.height,
		Maximized: w.maximized,
	}
}

func (w *x11Window) RestoreState(state WindowState) {
	if !isValidWindowState(state) {
		return
	}
	_ = xproto.ConfigureWindowChecked(
		w.disp.conn,
		w.win,
		xproto.ConfigWindowX|xproto.ConfigWindowY|xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
		[]uint32{uint32(int32(state.X)), uint32(int32(state.Y)), uint32(state.Width), uint32(state.Height)},
	).Check()
	if state.Maximized {
		w.Maximize()
	}
}

func (w *x11Window) BringToFront() {
	atom := w.disp.internAtom(netActiveWindow)
	if atom != 0 {
		ev := xproto.ClientMessageEvent{
			Format: 32,
			Window: w.win,
			Type:   atom,
			Data:   xproto.ClientMessageDataUnionData32New([]uint32{1, uint32(time.Now().UnixMilli()), 0, 0, 0}),
		}
		_ = xproto.SendEventChecked(
			w.disp.conn, false, w.disp.screen.Root,
			xproto.EventMaskSubstructureRedirect|xproto.EventMaskSubstructureNotify,
			string(ev.Bytes()),
		).Check()
	}
	_ = xproto.ConfigureWindowChecked(w.disp.conn, w.win, xproto.ConfigWindowStackMode, []uint32{xproto.StackModeAbove}).Check()
}

func (w *x11Window) SetDarkTitle(dark bool) {
	atom := w.disp.internAtom(gtkThemeVariant)
	if atom == 0 {
		return
	}
	val := "light"
	if dark {
		val = "dark"
	}
	utf8 := w.disp.internAtom("UTF8_STRING")
	_ = xproto.ChangePropertyChecked(w.disp.conn, xproto.PropModeReplace, w.win, atom, utf8, 8, uint32(len(val)), []byte(val)).Check()
}

func (w *x11Window) SetTitleBarColors(_, _, _ *WindowColor) {
	// X11 не предоставляет стандартного способа перекраски системной шапки окна произвольным цветом.
}

func (w *x11Window) SetIcon(rgba []byte, width, height int) error {
	if width <= 0 || height <= 0 || len(rgba) < width*height*4 {
		return errors.New("некорректный буфер иконки")
	}
	atom := w.disp.internAtom(netWMIcon)
	if atom == 0 {
		return errors.New("атом _NET_WM_ICON недоступен")
	}
	// Формат _NET_WM_ICON: width, height, затем ARGB пиксели (uint32)
	total := 2 + width*height
	buf := make([]byte, total*4)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(width))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(height))
	for i := 0; i < width*height; i++ {
		r := uint32(rgba[i*4+0])
		g := uint32(rgba[i*4+1])
		b := uint32(rgba[i*4+2])
		a := uint32(rgba[i*4+3])
		argb := (a << 24) | (r << 16) | (g << 8) | b
		binary.LittleEndian.PutUint32(buf[(2+i)*4:], argb)
	}
	return xproto.ChangePropertyChecked(w.disp.conn, xproto.PropModeReplace, w.win, atom, xproto.AtomCardinal, 32, uint32(total), buf).Check()
}

func (w *x11Window) ClipboardText() (string, error) {
	return w.disp.clip.readText(w.win)
}

func (w *x11Window) SetClipboardText(text string) error {
	return w.disp.clip.writeText(w.win, text)
}

func (w *x11Window) sendNetWMState(action uint32, prop1, prop2 string) {
	stateAtom := w.disp.internAtom(netWMStrut)
	if stateAtom == 0 {
		return
	}
	var a1, a2 uint32
	if prop1 != "" {
		a1 = uint32(w.disp.internAtom(prop1))
	}
	if prop2 != "" {
		a2 = uint32(w.disp.internAtom(prop2))
	}
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: w.win,
		Type:   stateAtom,
		Data:   xproto.ClientMessageDataUnionData32New([]uint32{action, a1, a2, 1, 0}),
	}
	_ = xproto.SendEventChecked(
		w.disp.conn, false, w.disp.screen.Root,
		xproto.EventMaskSubstructureRedirect|xproto.EventMaskSubstructureNotify,
		string(ev.Bytes()),
	).Check()
}

func (w *x11Window) applyFramelessHints() {
	atom := w.disp.internAtom(motifHintsProp)
	if atom == 0 {
		return
	}
	// struct MotifWmHints { flags, functions, decorations, inputMode, status } (5 x uint32)
	buf := make([]byte, 20)
	binary.LittleEndian.PutUint32(buf[0:4], 2) // MWM_HINTS_DECORATIONS
	binary.LittleEndian.PutUint32(buf[8:12], 0)
	_ = xproto.ChangePropertyChecked(w.disp.conn, xproto.PropModeReplace, w.win, atom, atom, 32, 5, buf).Check()
}

func (w *x11Window) applyMinSizeHints() {
	minW := int(math.Round(float64(w.cfg.MinWidth) * float64(w.scale)))
	minH := int(math.Round(float64(w.cfg.MinHeight) * float64(w.scale)))
	// WM_SIZE_HINTS (18 x uint32 = 72 байта), PMinSize = 1 << 4
	buf := make([]byte, 72)
	binary.LittleEndian.PutUint32(buf[0:4], 1<<4)
	binary.LittleEndian.PutUint32(buf[20:24], uint32(minW))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(minH))
	_ = xproto.ChangePropertyChecked(w.disp.conn, xproto.PropModeReplace, w.win, xproto.AtomWmNormalHints, xproto.AtomWmSizeHints, 32, 18, buf).Check()
}

func (w *x11Window) handleConfigure(e xproto.ConfigureNotifyEvent) {
	w.mu.Lock()
	oldW, oldH := w.width, w.height
	newW, newH := int(e.Width), int(e.Height)
	w.posX = int(e.X)
	w.posY = int(e.Y)
	w.width = newW
	w.height = newH
	w.mu.Unlock()

	if (newW != oldW || newH != oldH) && w.handler != nil {
		w.handler(Event{
			Type:   EventResize,
			Width:  newW,
			Height: newH,
		})
		w.needsPaint.Store(true)
	}
}

func (w *x11Window) handleClientMessage(e xproto.ClientMessageEvent) {
	if len(e.Data.Data32) > 0 {
		wmDel := w.disp.internAtom(wmDeleteWindowAtom)
		if xproto.Atom(e.Data.Data32[0]) == wmDel {
			w.Close()
			return
		}
	}
	// Проверяем XDND сообщения
	w.handleXDNDClientMessage(e)
}

func detectX11Scale(d *x11Display) float32 {
	// Читаем Xft.dpi из RESOURCE_MANAGER на корневом окне
	prop, err := xproto.GetProperty(d.conn, false, d.screen.Root, xproto.AtomResourceManager, xproto.AtomString, 0, 65536).Reply()
	if err == nil && len(prop.Value) > 0 {
		if scale := parseXftDPIScale(string(prop.Value)); scale > 0 {
			return scale
		}
	}
	return 1.0
}

func parseXftDPIScale(resources string) float32 {
	for _, line := range strings.Split(resources, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Xft.dpi:") {
			valStr := strings.TrimSpace(strings.TrimPrefix(line, "Xft.dpi:"))
			if dpi, err := strconv.ParseFloat(valStr, 64); err == nil && dpi >= 48 && dpi <= 480 {
				return float32(dpi / 96.0)
			}
		}
	}
	return 0
}
