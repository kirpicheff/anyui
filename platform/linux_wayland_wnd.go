//go:build linux

package platform

import (
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// wl_compositor requests
	wlCompositorCreateSurface uint16 = 0

	// wl_shm requests & formats
	wlShmCreatePool   uint16 = 0
	wlShmFormatARGB88 uint32 = 0
	wlShmFormatXRGB88 uint32 = 1

	// wl_shm_pool requests
	wlShmPoolCreateBuffer uint16 = 0
	wlShmPoolDestroy      uint16 = 1

	// wl_buffer requests & events
	wlBufferDestroy      uint16 = 0
	wlBufferEventRelease uint16 = 0

	// wl_surface requests & events
	wlSurfaceDestroy              uint16 = 0
	wlSurfaceAttach               uint16 = 1
	wlSurfaceDamage               uint16 = 2
	wlSurfaceFrame                uint16 = 3
	wlSurfaceCommit               uint16 = 6
	wlSurfaceSetBufferScale       uint16 = 8
	wlSurfaceDamageBuffer         uint16 = 9
	wlSurfaceEventEnter           uint16 = 0
	wlSurfaceEventLeave           uint16 = 1
	wlSurfaceEventPrefBufferScale uint16 = 2

	// wl_output events
	wlOutputEventScale uint16 = 3

	// xdg_wm_base requests & events
	xdgWmBaseGetXdgSurface uint16 = 2
	xdgWmBasePong          uint16 = 3
	xdgWmBaseEventPing     uint16 = 0

	// xdg_surface requests & events
	xdgSurfaceDestroy          uint16 = 0
	xdgSurfaceGetToplevel      uint16 = 1
	xdgSurfaceSetWindowGeom    uint16 = 3
	xdgSurfaceAckConfigure     uint16 = 4
	xdgSurfaceEventConfigure   uint16 = 0

	// xdg_toplevel requests & events
	xdgToplevelDestroy        uint16 = 0
	xdgToplevelSetParent      uint16 = 1
	xdgToplevelSetTitle       uint16 = 2
	xdgToplevelSetAppID       uint16 = 3
	xdgToplevelMove           uint16 = 5
	xdgToplevelResize         uint16 = 6
	xdgToplevelSetMaxSize     uint16 = 7
	xdgToplevelSetMinSize     uint16 = 8
	xdgToplevelSetMaximized   uint16 = 9
	xdgToplevelUnsetMaximized uint16 = 10
	xdgToplevelSetFullscreen  uint16 = 11
	xdgToplevelUnsetFull      uint16 = 12
	xdgToplevelSetMinimized   uint16 = 13
	xdgToplevelEventConfigure uint16 = 0
	xdgToplevelEventClose     uint16 = 1

	// xdg_toplevel states
	xdgToplevelStateMaximized  uint32 = 1
	xdgToplevelStateFullscreen uint32 = 2
	xdgToplevelStateActivated  uint32 = 4

	// zxdg_decoration_manager_v1 / zxdg_toplevel_decoration_v1
	zxdgDecMgrGetToplevelDec uint16 = 1
	zxdgToplevelDecDestroy   uint16 = 0
	zxdgToplevelDecSetMode   uint16 = 1
	zxdgDecModeClientSide    uint32 = 1
	zxdgDecModeServerSide    uint32 = 2

	// wp_fractional_scale_manager_v1 / wp_fractional_scale_v1
	wpFracMgrGetScale       uint16 = 1
	wpFracScaleDestroy      uint16 = 0
	wpFracScaleEventPref    uint16 = 0

	// wp_viewporter / wp_viewport
	wpViewporterGetViewport uint16 = 1
	wpViewportDestroy       uint16 = 0
	wpViewportSetDest       uint16 = 2
)

type wlDisplay struct {
	conn       *wlConn
	registryID uint32

	compositorID  uint32
	compositorVer uint32
	shmID         uint32
	xdgWmBaseID   uint32
	seatID        uint32
	decorMgrID    uint32
	fracScaleID   uint32
	viewporterID  uint32
	cursorShapeMgrID uint32
	dataDevMgrID  uint32

	seat        *wlSeatState
	dataDev     *wlDataDeviceState
	cursorCache *wlCursorCache

	mu          sync.Mutex
	windows     map[uint32]*waylandWindow // surfaceID -> window
	outputs     map[uint32]int32          // outputID -> scale
	wakePending atomic.Bool
	running     atomic.Bool
}

var (
	wlDispOnce sync.Once
	wlDisp     *wlDisplay
	wlDispErr  error
)

func getWaylandDisplay() (*wlDisplay, error) {
	wlDispOnce.Do(func() {
		conn, err := connectWayland()
		if err != nil {
			wlDispErr = err
			return
		}

		d := &wlDisplay{
			conn:    conn,
			windows: make(map[uint32]*waylandWindow),
			outputs: make(map[uint32]int32),
		}

		regID := conn.allocID(func(ev wlEvent) {
			if ev.Opcode == wlRegistryEventGlobal {
				r := wlReader{data: ev.Payload}
				name, ok1 := r.uint32()
				iface, ok2 := r.string()
				ver, ok3 := r.uint32()
				if ok1 && ok2 && ok3 {
					conn.globalsMu.Lock()
					conn.globals[iface] = wlGlobal{Name: name, Interface: iface, Version: ver}
					conn.globalsMu.Unlock()
					if iface == "wl_output" {
						d.bindOutput(name, ver)
					}
				}
			}
		})
		d.registryID = regID

		var w wlWriter
		w.uint32(regID)
		if err := conn.send(wlDisplayID, wlDisplayGetRegistry, w.bytes()); err != nil {
			conn.Close()
			wlDispErr = err
			return
		}
		if err := conn.roundtrip(); err != nil {
			conn.Close()
			wlDispErr = err
			return
		}

		if d.compositorID, d.compositorVer, err = conn.bindGlobal(regID, "wl_compositor", 6, nil); err != nil {
			conn.Close()
			wlDispErr = err
			return
		}
		if d.shmID, _, err = conn.bindGlobal(regID, "wl_shm", 1, nil); err != nil {
			conn.Close()
			wlDispErr = err
			return
		}
		if d.xdgWmBaseID, _, err = conn.bindGlobal(regID, "xdg_wm_base", 4, func(ev wlEvent) {
			if ev.Opcode == xdgWmBaseEventPing {
				r := wlReader{data: ev.Payload}
				if serial, ok := r.uint32(); ok {
					var pw wlWriter
					pw.uint32(serial)
					_ = conn.send(d.xdgWmBaseID, xdgWmBasePong, pw.bytes())
				}
			}
		}); err != nil {
			conn.Close()
			wlDispErr = err
			return
		}

		if conn.hasGlobal("wp_cursor_shape_manager_v1") {
			d.cursorShapeMgrID, _, _ = conn.bindGlobal(regID, "wp_cursor_shape_manager_v1", 1, nil)
		}
		if conn.hasGlobal("zxdg_decoration_manager_v1") {
			d.decorMgrID, _, _ = conn.bindGlobal(regID, "zxdg_decoration_manager_v1", 1, nil)
		}
		if conn.hasGlobal("wp_fractional_scale_manager_v1") {
			d.fracScaleID, _, _ = conn.bindGlobal(regID, "wp_fractional_scale_manager_v1", 1, nil)
		}
		if conn.hasGlobal("wp_viewporter") {
			d.viewporterID, _, _ = conn.bindGlobal(regID, "wp_viewporter", 1, nil)
		}
		if conn.hasGlobal("wl_data_device_manager") {
			d.dataDevMgrID, _, _ = conn.bindGlobal(regID, "wl_data_device_manager", 3, nil)
		}
		if conn.hasGlobal("wl_seat") {
			if seatID, _, bindErr := conn.bindGlobal(regID, "wl_seat", 8, nil); bindErr == nil {
				d.seatID = seatID
				d.seat = newWLSeatState(d, seatID)
				d.dataDev = newWLDataDeviceState(d)
			}
		}

		if d.cursorShapeMgrID == 0 {
			d.cursorCache = newWLCursorCache(d)
		}
		// Второй roundtrip, чтобы получить capabilities wl_seat и scale wl_output
		_ = conn.roundtrip()
		wlDisp = d
	})
	return wlDisp, wlDispErr
}

func (d *wlDisplay) bindOutput(name, ver uint32) {
	if ver > 4 {
		ver = 4
	}
	var outID uint32
	outID = d.conn.allocID(func(ev wlEvent) {
		if ev.Opcode == wlOutputEventScale {
			r := wlReader{data: ev.Payload}
			if factor, ok := r.int32(); ok && factor > 0 {
				d.mu.Lock()
				d.outputs[outID] = factor
				d.mu.Unlock()
			}
		}
	})
	var w wlWriter
	w.uint32(name)
	w.string("wl_output")
	w.uint32(ver)
	w.uint32(outID)
	_ = d.conn.send(d.registryID, wlRegistryBind, w.bytes())
}

func (d *wlDisplay) maxOutputScale() float32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	var maxS int32 = 1
	for _, s := range d.outputs {
		if s > maxS {
			maxS = s
		}
	}
	return float32(maxS)
}

func (d *wlDisplay) windowBySurface(surfID uint32) *waylandWindow {
	d.mu.Lock()
	w := d.windows[surfID]
	d.mu.Unlock()
	return w
}

func (d *wlDisplay) anyWindow() *waylandWindow {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, w := range d.windows {
		return w
	}
	return nil
}

func (d *wlDisplay) flushPaints() {
	d.mu.Lock()
	wins := make([]*waylandWindow, 0, len(d.windows))
	for _, w := range d.windows {
		wins = append(wins, w)
	}
	d.mu.Unlock()

	for _, w := range wins {
		if !w.framePending.Load() && w.needsPaint.Swap(false) && w.handler != nil && !w.closed.Load() && w.configured.Load() {
			w.handler(Event{Type: EventPaint})
		}
	}
}

func runWaylandEventLoop(onStart func() error) error {
	d, err := getWaylandDisplay()
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

		if err := d.conn.readAndDispatch(false); err != nil {
			return err
		}
		d.flushPaints()

		wlReady, wakeReady, err := d.conn.waitEvents(-1)
		if err != nil {
			return err
		}
		if wakeReady {
			d.conn.drainWakePipe()
			if d.wakePending.Swap(false) {
				if w := d.anyWindow(); w != nil && w.handler != nil {
					w.handler(Event{Type: EventWakeUp})
				}
			}
		}
		if wlReady {
			if err := d.conn.readAndDispatch(false); err != nil {
				return err
			}
		}
		d.flushPaints()
	}
}

type wlShmBuffer struct {
	poolID   uint32
	bufferID uint32
	fd       int
	data     []byte
	width    int
	height   int
	stride   int
	size     int
	busy     atomic.Bool
}

func createWLShmBuffer(d *wlDisplay, width, height int) (*wlShmBuffer, error) {
	stride := width * 4
	size := stride * height
	if size <= 0 {
		return nil, errors.New("некорректный размер shm-буфера")
	}

	fd, err := createAnonymousShmFile(size)
	if err != nil {
		return nil, err
	}

	data, err := unix.Mmap(fd, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("mmap shm: %w", err)
	}

	poolID := d.conn.allocID(nil)
	var pw wlWriter
	pw.uint32(poolID)
	pw.int32(int32(size))
	if err := d.conn.sendWithFDs(d.shmID, wlShmCreatePool, pw.bytes(), []int{fd}); err != nil {
		_ = unix.Munmap(data)
		_ = unix.Close(fd)
		d.conn.removeHandler(poolID)
		return nil, err
	}

	buf := &wlShmBuffer{
		poolID: poolID,
		fd:     fd,
		data:   data,
		width:  width,
		height: height,
		stride: stride,
		size:   size,
	}

	buf.bufferID = d.conn.allocID(func(ev wlEvent) {
		if ev.Opcode == wlBufferEventRelease {
			buf.busy.Store(false)
		}
	})

	var bw wlWriter
	bw.uint32(buf.bufferID)
	bw.int32(0)
	bw.int32(int32(width))
	bw.int32(int32(height))
	bw.int32(int32(stride))
	bw.uint32(wlShmFormatARGB88)
	if err := d.conn.send(poolID, wlShmPoolCreateBuffer, bw.bytes()); err != nil {
		buf.destroy(d)
		return nil, err
	}

	return buf, nil
}

func (b *wlShmBuffer) destroy(d *wlDisplay) {
	if b == nil {
		return
	}
	if b.bufferID != 0 {
		_ = d.conn.send(b.bufferID, wlBufferDestroy, nil)
		d.conn.removeHandler(b.bufferID)
		b.bufferID = 0
	}
	if b.poolID != 0 {
		_ = d.conn.send(b.poolID, wlShmPoolDestroy, nil)
		d.conn.removeHandler(b.poolID)
		b.poolID = 0
	}
	if b.data != nil {
		_ = unix.Munmap(b.data)
		b.data = nil
	}
	if b.fd >= 0 {
		_ = unix.Close(b.fd)
		b.fd = -1
	}
}

func createAnonymousShmFile(size int) (int, error) {
	fd, err := unix.MemfdCreate("anyui-shm", unix.MFD_CLOEXEC)
	if err == nil {
		if truncErr := unix.Ftruncate(fd, int64(size)); truncErr == nil {
			return fd, nil
		}
		_ = unix.Close(fd)
	}

	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = "/dev/shm"
	}
	f, err := os.CreateTemp(dir, "anyui-shm-*")
	if err != nil {
		return -1, err
	}
	path := f.Name()
	_ = os.Remove(path)
	if err := f.Truncate(int64(size)); err != nil {
		_ = f.Close()
		return -1, err
	}
	dupFD, err := unix.FcntlInt(f.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	_ = f.Close()
	if err != nil {
		return -1, err
	}
	return dupFD, nil
}

type waylandWindow struct {
	disp *wlDisplay
	cfg  WindowConfig

	surfaceID    uint32
	xdgSurfaceID uint32
	toplevelID   uint32
	decorID      uint32
	fracScaleID  uint32
	viewportID   uint32

	handler func(Event)
	onClose func() bool

	mu         sync.Mutex
	logicalW   int
	logicalH   int
	width      int // физические пиксели
	height     int // физические пиксели
	scale      float32
	cursor     CursorType
	maximized  bool
	fullscreen bool
	pendingW   int
	pendingH   int
	pendingMax bool
	pendingFS  bool
	buffers    [2]*wlShmBuffer
	bufIdx     int

	csdEnabled   bool
	csdHoverBtn  int
	csdTitle     string
	csdLastClick time.Time

	framePending atomic.Bool
	configured   atomic.Bool
	needsPaint   atomic.Bool
	closed       atomic.Bool
}

func newWaylandWindow(cfg WindowConfig) (Window, error) {
	d, err := getWaylandDisplay()
	if err != nil {
		return nil, err
	}

	scale := d.maxOutputScale()
	if scale <= 0 {
		scale = 1.0
	}

	logW := cfg.Width
	logH := cfg.Height
	if logW <= 0 {
		logW = 800
	}
	if logH <= 0 {
		logH = 600
	}
	if cfg.InitialState != nil && isValidWindowState(*cfg.InitialState) {
		logW = int(math.Round(float64(cfg.InitialState.Width) / float64(scale)))
		logH = int(math.Round(float64(cfg.InitialState.Height) / float64(scale)))
		if logW <= 0 {
			logW = 800
		}
		if logH <= 0 {
			logH = 600
		}
	}

	w := &waylandWindow{
		disp:       d,
		cfg:        cfg,
		logicalW:   logW,
		logicalH:   logH,
		width:      int(math.Round(float64(logW) * float64(scale))),
		height:     int(math.Round(float64(logH) * float64(scale))),
		scale:      scale,
		cursor:     CursorArrow,
		csdEnabled: !cfg.Frameless && d.decorMgrID == 0,
		csdTitle:   cfg.Title,
	}
	if w.csdEnabled {
		w.logicalH += csdLogicalHeight
		w.height = int(math.Round(float64(w.logicalH) * float64(scale)))
	}

	w.surfaceID = d.conn.allocID(w.handleSurfaceEvent)
	var cw wlWriter
	cw.uint32(w.surfaceID)
	if err := d.conn.send(d.compositorID, wlCompositorCreateSurface, cw.bytes()); err != nil {
		d.conn.removeHandler(w.surfaceID)
		return nil, err
	}

	d.mu.Lock()
	d.windows[w.surfaceID] = w
	d.mu.Unlock()

	if d.fracScaleID != 0 && d.viewporterID != 0 {
		w.fracScaleID = d.conn.allocID(w.handleFracScaleEvent)
		var fw wlWriter
		fw.uint32(w.fracScaleID)
		fw.uint32(w.surfaceID)
		_ = d.conn.send(d.fracScaleID, wpFracMgrGetScale, fw.bytes())

		w.viewportID = d.conn.allocID(nil)
		var vw wlWriter
		vw.uint32(w.viewportID)
		vw.uint32(w.surfaceID)
		_ = d.conn.send(d.viewporterID, wpViewporterGetViewport, vw.bytes())
	} else if d.compositorVer >= 3 {
		var sw wlWriter
		sw.int32(int32(math.Round(float64(scale))))
		_ = d.conn.send(w.surfaceID, wlSurfaceSetBufferScale, sw.bytes())
	}

	w.xdgSurfaceID = d.conn.allocID(w.handleXDGSurfaceEvent)
	var xw wlWriter
	xw.uint32(w.xdgSurfaceID)
	xw.uint32(w.surfaceID)
	if err := d.conn.send(d.xdgWmBaseID, xdgWmBaseGetXdgSurface, xw.bytes()); err != nil {
		w.destroyObjects()
		return nil, err
	}

	w.toplevelID = d.conn.allocID(w.handleToplevelEvent)
	var tw wlWriter
	tw.uint32(w.toplevelID)
	if err := d.conn.send(w.xdgSurfaceID, xdgSurfaceGetToplevel, tw.bytes()); err != nil {
		w.destroyObjects()
		return nil, err
	}

	if cfg.Title != "" {
		w.SetTitle(cfg.Title)
	}
	var aw wlWriter
	aw.string("anyui")
	_ = d.conn.send(w.toplevelID, xdgToplevelSetAppID, aw.bytes())

	if cfg.MinWidth > 0 || cfg.MinHeight > 0 {
		var mw wlWriter
		mw.int32(int32(cfg.MinWidth))
		mw.int32(int32(cfg.MinHeight))
		_ = d.conn.send(w.toplevelID, xdgToplevelSetMinSize, mw.bytes())
	}

	if parent, ok := cfg.Parent.(*waylandWindow); ok && parent != nil && parent.toplevelID != 0 {
		var pw wlWriter
		pw.uint32(parent.toplevelID)
		_ = d.conn.send(w.toplevelID, xdgToplevelSetParent, pw.bytes())
	}

	if d.decorMgrID != 0 {
		w.decorID = d.conn.allocID(func(ev wlEvent) {
			if ev.Opcode == 0 && !cfg.Frameless {
				r := wlReader{data: ev.Payload}
				if mode, ok := r.uint32(); ok && mode == zxdgDecModeClientSide {
					w.mu.Lock()
					w.csdEnabled = true
					w.mu.Unlock()
					w.needsPaint.Store(true)
				}
			}
		})
		var dw wlWriter
		dw.uint32(w.decorID)
		dw.uint32(w.toplevelID)
		_ = d.conn.send(d.decorMgrID, zxdgDecMgrGetToplevelDec, dw.bytes())

		mode := zxdgDecModeServerSide
		if cfg.Frameless {
			mode = zxdgDecModeClientSide
		}
		var sm wlWriter
		sm.uint32(mode)
		_ = d.conn.send(w.decorID, zxdgToplevelDecSetMode, sm.bytes())
	}

	if cfg.InitialState != nil && cfg.InitialState.Maximized {
		w.Maximize()
	}

	// Первичный commit без буфера — Wayland требует его для получения xdg_surface.configure.
	_ = d.conn.send(w.surfaceID, wlSurfaceCommit, nil)
	_ = d.conn.roundtrip()

	return w, nil
}

func (w *waylandWindow) handleSurfaceEvent(ev wlEvent) {
	if ev.Opcode == wlSurfaceEventPrefBufferScale && w.fracScaleID == 0 {
		r := wlReader{data: ev.Payload}
		if factor, ok := r.int32(); ok && factor > 0 {
			w.applyNewScale(float32(factor))
		}
	}
}

func (w *waylandWindow) handleFracScaleEvent(ev wlEvent) {
	if ev.Opcode == wpFracScaleEventPref {
		r := wlReader{data: ev.Payload}
		if scale120, ok := r.uint32(); ok && scale120 > 0 {
			w.applyNewScale(float32(scale120) / 120.0)
		}
	}
}

func (w *waylandWindow) applyNewScale(newScale float32) {
	if newScale <= 0 {
		return
	}
	w.mu.Lock()
	if math.Abs(float64(w.scale-newScale)) < 0.01 {
		w.mu.Unlock()
		return
	}
	w.scale = newScale
	physW := int(math.Round(float64(w.logicalW) * float64(newScale)))
	physH := int(math.Round(float64(w.logicalH) * float64(newScale)))
	if physW < 1 {
		physW = 1
	}
	if physH < 1 {
		physH = 1
	}
	w.width = physW
	w.height = physH
	w.mu.Unlock()

	if w.fracScaleID == 0 && w.disp.compositorVer >= 3 {
		var sw wlWriter
		sw.int32(int32(math.Round(float64(newScale))))
		_ = w.disp.conn.send(w.surfaceID, wlSurfaceSetBufferScale, sw.bytes())
	}

	w.emit(Event{Type: EventDPIChange, Scale: newScale})
	cw, ch := w.Size()
	w.emit(Event{Type: EventResize, Width: cw, Height: ch})
	w.needsPaint.Store(true)
}

func (w *waylandWindow) handleXDGSurfaceEvent(ev wlEvent) {
	if ev.Opcode != xdgSurfaceEventConfigure {
		return
	}
	r := wlReader{data: ev.Payload}
	serial, ok := r.uint32()
	if !ok {
		return
	}

	var aw wlWriter
	aw.uint32(serial)
	_ = w.disp.conn.send(w.xdgSurfaceID, xdgSurfaceAckConfigure, aw.bytes())

	w.mu.Lock()
	pw, ph := w.pendingW, w.pendingH
	if pw > 0 {
		w.logicalW = pw
	}
	if ph > 0 {
		w.logicalH = ph
	}
	w.maximized = w.pendingMax
	w.fullscreen = w.pendingFS

	physW := int(math.Round(float64(w.logicalW) * float64(w.scale)))
	physH := int(math.Round(float64(w.logicalH) * float64(w.scale)))
	if physW < 1 {
		physW = 1
	}
	if physH < 1 {
		physH = 1
	}
	resized := physW != w.width || physH != w.height
	w.width = physW
	w.height = physH
	logW, logH := w.logicalW, w.logicalH
	w.mu.Unlock()

	if w.viewportID != 0 {
		var vw wlWriter
		vw.int32(int32(logW))
		vw.int32(int32(logH))
		_ = w.disp.conn.send(w.viewportID, wpViewportSetDest, vw.bytes())
	}

	var gw wlWriter
	gw.int32(0)
	gw.int32(0)
	gw.int32(int32(logW))
	gw.int32(int32(logH))
	_ = w.disp.conn.send(w.xdgSurfaceID, xdgSurfaceSetWindowGeom, gw.bytes())

	first := !w.configured.Swap(true)
	if (resized || first) && w.handler != nil {
		cw, ch := w.Size()
		w.handler(Event{Type: EventResize, Width: cw, Height: ch})
	}
	w.needsPaint.Store(true)
}

func (w *waylandWindow) handleToplevelEvent(ev wlEvent) {
	switch ev.Opcode {
	case xdgToplevelEventConfigure:
		r := wlReader{data: ev.Payload}
		cw, ok1 := r.int32()
		ch, ok2 := r.int32()
		states, ok3 := r.array()
		if !ok1 || !ok2 || !ok3 {
			return
		}
		var isMax, isFS bool
		for i := 0; i+4 <= len(states); i += 4 {
			st := uint32(states[i]) | uint32(states[i+1])<<8 | uint32(states[i+2])<<16 | uint32(states[i+3])<<24
			if st == xdgToplevelStateMaximized {
				isMax = true
			} else if st == xdgToplevelStateFullscreen {
				isFS = true
			}
		}
		w.mu.Lock()
		w.pendingW = int(cw)
		w.pendingH = int(ch)
		w.pendingMax = isMax
		w.pendingFS = isFS
		w.mu.Unlock()

	case xdgToplevelEventClose:
		w.Close()
	}
}

func (w *waylandWindow) emit(ev Event) {
	if w.handler != nil && !w.closed.Load() {
		w.handler(ev)
	}
}

func (w *waylandWindow) currentCursor() CursorType {
	w.mu.Lock()
	c := w.cursor
	w.mu.Unlock()
	return c
}

func (w *waylandWindow) Run(handler func(Event)) error {
	w.handler = handler
	w.needsPaint.Store(true)
	return runWaylandEventLoop(nil)
}

func (w *waylandWindow) Open(handler func(Event)) error {
	w.handler = handler
	w.needsPaint.Store(true)
	w.WakeUp()
	return nil
}

func (w *waylandWindow) Present(pixels []byte, width, height int) {
	w.PresentRect(pixels, width, height, 0, 0, width, height)
}

func (w *waylandWindow) PresentRect(pixels []byte, width, height, rx, ry, rw, rh int) {
	if w.closed.Load() || !w.configured.Load() || width <= 0 || height <= 0 || len(pixels) < width*height*4 {
		return
	}

	barH := w.csdHeightPx()
	totalW := width
	totalH := height + barH

	w.mu.Lock()
	idx := w.bufIdx
	if w.buffers[idx] != nil && w.buffers[idx].busy.Load() {
		alt := 1 - idx
		if w.buffers[alt] == nil || !w.buffers[alt].busy.Load() {
			idx = alt
		}
	}
	w.bufIdx = 1 - idx

	buf := w.buffers[idx]
	if buf == nil || buf.width != totalW || buf.height != totalH {
		if buf != nil {
			buf.destroy(w.disp)
		}
		newBuf, err := createWLShmBuffer(w.disp, totalW, totalH)
		if err != nil {
			w.mu.Unlock()
			return
		}
		w.buffers[idx] = newBuf
		buf = newBuf
	}
	w.mu.Unlock()

	if barH > 0 {
		w.renderCSDTitleBar(buf.data, totalW, barH)
		copy(buf.data[barH*totalW*4:], pixels[:width*height*4])
	} else {
		copy(buf.data, pixels[:width*height*4])
	}
	buf.busy.Store(true)

	var aw wlWriter
	aw.uint32(buf.bufferID)
	aw.int32(0)
	aw.int32(0)
	_ = w.disp.conn.send(w.surfaceID, wlSurfaceAttach, aw.bytes())

	if rw <= 0 || rh <= 0 {
		rx, ry, rw, rh = 0, 0, totalW, totalH
	} else if barH > 0 {
		if rx == 0 && ry == 0 && rw == width && rh == height {
			rh = totalH
		} else {
			ry += barH
		}
	}
	if w.disp.compositorVer >= 4 {
		var dw wlWriter
		dw.int32(int32(rx))
		dw.int32(int32(ry))
		dw.int32(int32(rw))
		dw.int32(int32(rh))
		_ = w.disp.conn.send(w.surfaceID, wlSurfaceDamageBuffer, dw.bytes())
	} else {
		var dw wlWriter
		dw.int32(0)
		dw.int32(0)
		dw.int32(int32(w.logicalW))
		dw.int32(int32(w.logicalH))
		_ = w.disp.conn.send(w.surfaceID, wlSurfaceDamage, dw.bytes())
	}

	var cbID uint32
	cbID = w.disp.conn.allocID(func(ev wlEvent) {
		if ev.Opcode == wlCallbackEventDone {
			w.framePending.Store(false)
			w.disp.conn.removeHandler(cbID)
		}
	})
	var fw wlWriter
	fw.uint32(cbID)
	if err := w.disp.conn.send(w.surfaceID, wlSurfaceFrame, fw.bytes()); err == nil {
		w.framePending.Store(true)
	} else {
		w.disp.conn.removeHandler(cbID)
	}

	_ = w.disp.conn.send(w.surfaceID, wlSurfaceCommit, nil)
}

func (w *waylandWindow) Invalidate() {
	if w.closed.Load() {
		return
	}
	w.needsPaint.Store(true)
	w.WakeUp()
}

func (w *waylandWindow) WakeUp() {
	if w.closed.Load() {
		return
	}
	w.disp.wakePending.Store(true)
	w.disp.conn.wakeUp()
}

func (w *waylandWindow) Show() {
	if w.closed.Load() {
		return
	}
	_ = w.disp.conn.send(w.surfaceID, wlSurfaceCommit, nil)
	w.Invalidate()
}

func (w *waylandWindow) Hide() {
	if w.closed.Load() {
		return
	}
	var aw wlWriter
	aw.uint32(0)
	aw.int32(0)
	aw.int32(0)
	_ = w.disp.conn.send(w.surfaceID, wlSurfaceAttach, aw.bytes())
	_ = w.disp.conn.send(w.surfaceID, wlSurfaceCommit, nil)
}

func (w *waylandWindow) Close() {
	if w.closed.Load() {
		return
	}
	if w.onClose != nil && !w.onClose() {
		return
	}
	if w.closed.Swap(true) {
		return
	}
	if w.handler != nil {
		w.handler(Event{Type: EventClose})
	}
	w.destroyObjects()
	w.disp.conn.wakeUp()
}

func (w *waylandWindow) destroyObjects() {
	w.disp.mu.Lock()
	delete(w.disp.windows, w.surfaceID)
	w.disp.mu.Unlock()

	w.mu.Lock()
	for i := range w.buffers {
		if w.buffers[i] != nil {
			w.buffers[i].destroy(w.disp)
			w.buffers[i] = nil
		}
	}
	w.mu.Unlock()

	if w.decorID != 0 {
		_ = w.disp.conn.send(w.decorID, zxdgToplevelDecDestroy, nil)
		w.disp.conn.removeHandler(w.decorID)
		w.decorID = 0
	}
	if w.toplevelID != 0 {
		_ = w.disp.conn.send(w.toplevelID, xdgToplevelDestroy, nil)
		w.disp.conn.removeHandler(w.toplevelID)
		w.toplevelID = 0
	}
	if w.xdgSurfaceID != 0 {
		_ = w.disp.conn.send(w.xdgSurfaceID, xdgSurfaceDestroy, nil)
		w.disp.conn.removeHandler(w.xdgSurfaceID)
		w.xdgSurfaceID = 0
	}
	if w.fracScaleID != 0 {
		_ = w.disp.conn.send(w.fracScaleID, wpFracScaleDestroy, nil)
		w.disp.conn.removeHandler(w.fracScaleID)
		w.fracScaleID = 0
	}
	if w.viewportID != 0 {
		_ = w.disp.conn.send(w.viewportID, wpViewportDestroy, nil)
		w.disp.conn.removeHandler(w.viewportID)
		w.viewportID = 0
	}
	if w.surfaceID != 0 {
		_ = w.disp.conn.send(w.surfaceID, wlSurfaceDestroy, nil)
		w.disp.conn.removeHandler(w.surfaceID)
		w.surfaceID = 0
	}
}

func (w *waylandWindow) SetTitle(title string) {
	w.mu.Lock()
	w.csdTitle = title
	csd := w.csdEnabled
	w.mu.Unlock()
	if csd {
		w.needsPaint.Store(true)
	}
	if w.toplevelID == 0 {
		return
	}
	var tw wlWriter
	tw.string(title)
	_ = w.disp.conn.send(w.toplevelID, xdgToplevelSetTitle, tw.bytes())
}

func (w *waylandWindow) SetCursor(cursor CursorType) {
	w.mu.Lock()
	w.cursor = cursor
	w.mu.Unlock()
	if w.disp.seat != nil {
		w.disp.seat.applyCursor(cursor)
	}
}

func (w *waylandWindow) SetCustomCursor(cur *CustomCursor) {
	// Кастомный BGRA-курсор на Wayland пока не реализован; откатываемся к текущему системному курсору.
	_ = cur
	w.SetCursor(w.currentCursor())
}

func (w *waylandWindow) Size() (int, int) {
	barH := w.csdHeightPx()
	w.mu.Lock()
	ww, wh := w.width, w.height
	w.mu.Unlock()
	ch := wh - barH
	if ch < 1 {
		ch = 1
	}
	return ww, ch
}

func (w *waylandWindow) Scale() float32 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.scale
}

func (w *waylandWindow) Minimize() {
	if w.toplevelID != 0 {
		_ = w.disp.conn.send(w.toplevelID, xdgToplevelSetMinimized, nil)
	}
}

func (w *waylandWindow) Maximize() {
	if w.toplevelID != 0 {
		_ = w.disp.conn.send(w.toplevelID, xdgToplevelSetMaximized, nil)
	}
}

func (w *waylandWindow) Restore() {
	if w.toplevelID == 0 {
		return
	}
	if w.IsFullscreen() {
		_ = w.disp.conn.send(w.toplevelID, xdgToplevelUnsetFull, nil)
	}
	if w.IsMaximized() {
		_ = w.disp.conn.send(w.toplevelID, xdgToplevelUnsetMaximized, nil)
	}
}

func (w *waylandWindow) IsMaximized() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.maximized
}

func (w *waylandWindow) IsMinimized() bool {
	return false
}

func (w *waylandWindow) SetFullscreen(fullscreen bool) {
	if w.toplevelID == 0 {
		return
	}
	if fullscreen {
		var fw wlWriter
		fw.uint32(0)
		_ = w.disp.conn.send(w.toplevelID, xdgToplevelSetFullscreen, fw.bytes())
	} else {
		_ = w.disp.conn.send(w.toplevelID, xdgToplevelUnsetFull, nil)
	}
}

func (w *waylandWindow) IsFullscreen() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.fullscreen
}

func (w *waylandWindow) Position() (int, int) {
	// Протокол Wayland намеренно не раскрывает глобальную позицию toplevel-окна.
	return 0, 0
}

func (w *waylandWindow) SetPosition(x, y int) {
	// Протокол Wayland не позволяет клиентским окнам задавать экранные координаты.
	_, _ = x, y
}

func (w *waylandWindow) SetSize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	w.mu.Lock()
	w.logicalW = width
	w.logicalH = height
	physW := int(math.Round(float64(width) * float64(w.scale)))
	physH := int(math.Round(float64(height) * float64(w.scale)))
	w.width = physW
	w.height = physH
	w.mu.Unlock()

	if w.viewportID != 0 {
		var vw wlWriter
		vw.int32(int32(width))
		vw.int32(int32(height))
		_ = w.disp.conn.send(w.viewportID, wpViewportSetDest, vw.bytes())
	}
	w.emit(Event{Type: EventResize, Width: physW, Height: physH})
	w.Invalidate()
}

func (w *waylandWindow) SetOnClose(fn func() bool) {
	w.onClose = fn
}

func (w *waylandWindow) NativeHandle() uintptr {
	return uintptr(w.surfaceID)
}

func (w *waylandWindow) SetCapture() {
	// В Wayland захват мыши при нажатой кнопке выполняется композитором автоматически.
}

func (w *waylandWindow) ReleaseCapture() {}

func (w *waylandWindow) DoubleClickTimeMs() int {
	return 400
}

func (w *waylandWindow) State() WindowState {
	w.mu.Lock()
	defer w.mu.Unlock()
	return WindowState{
		Width:     w.width,
		Height:    w.height,
		Maximized: w.maximized,
	}
}

func (w *waylandWindow) RestoreState(state WindowState) {
	if !isValidWindowState(state) {
		return
	}
	w.mu.Lock()
	scale := w.scale
	w.mu.Unlock()
	if scale <= 0 {
		scale = 1.0
	}
	w.SetSize(int(math.Round(float64(state.Width)/float64(scale))), int(math.Round(float64(state.Height)/float64(scale))))
	if state.Maximized {
		w.Maximize()
	} else {
		w.Restore()
	}
}

func (w *waylandWindow) ClipboardText() (string, error) {
	if w.disp.dataDev == nil {
		return "", errors.New("wl_data_device_manager недоступен")
	}
	return w.disp.dataDev.readText()
}

func (w *waylandWindow) SetClipboardText(text string) error {
	if w.disp.dataDev == nil {
		return errors.New("wl_data_device_manager недоступен")
	}
	return w.disp.dataDev.writeText(text)
}

func (w *waylandWindow) ClipboardImage() (image.Image, error) {
	if w.disp.dataDev == nil {
		return nil, errors.New("wl_data_device_manager недоступен")
	}
	return w.disp.dataDev.readImage()
}

func (w *waylandWindow) SetClipboardImage(img image.Image) error {
	if w.disp.dataDev == nil {
		return errors.New("wl_data_device_manager недоступен")
	}
	return w.disp.dataDev.writeImage(img)
}
