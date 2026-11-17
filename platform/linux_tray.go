//go:build linux

package platform

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	systemTrayRequestDock = 0
	xembedMapped          = 1 << 0
)

type x11Tray struct {
	disp *x11Display
	cfg  TrayConfig

	mu      sync.Mutex
	win     xproto.Window
	gc      xproto.Gcontext
	width   int
	height  int
	rgba    []byte
	iconW   int
	iconH   int
	menu    []TrayMenuItem
	closed  bool
	lastClk time.Time
}

type noopLinuxTray struct{}

func (noopLinuxTray) SetTooltip(string) error               { return nil }
func (noopLinuxTray) SetIconRGBA([]byte, int, int) error    { return nil }
func (noopLinuxTray) SetMenu([]TrayMenuItem)                {}
func (noopLinuxTray) ShowNotification(n TrayNotification) error { return sendLinuxNotification(n) }
func (noopLinuxTray) Close() error                          { return nil }

// NewTray создаёт иконку приложения в области уведомлений X11 (протокол XEmbed / _NET_SYSTEM_TRAY).
// Если системный трей X11 недоступен (например, в XWayland/WSLg или GNOME без расширений),
// возвращает безопасную заглушку, чтобы не блокировать запуск приложения.
func NewTray(cfg TrayConfig) (Tray, error) {
	d, err := getX11Display()
	if err != nil {
		return nil, err
	}

	traySelAtom := d.internAtom(fmt.Sprintf("_NET_SYSTEM_TRAY_S%d", d.conn.DefaultScreen))
	if traySelAtom == 0 {
		return noopLinuxTray{}, nil
	}

	ownerReply, err := xproto.GetSelectionOwner(d.conn, traySelAtom).Reply()
	if err != nil || ownerReply == nil || ownerReply.Owner == 0 {
		return noopLinuxTray{}, nil
	}

	winID, err := xproto.NewWindowId(d.conn)
	if err != nil {
		return nil, err
	}

	const defaultSize = 24
	eventMask := uint32(
		xproto.EventMaskExposure |
			xproto.EventMaskStructureNotify |
			xproto.EventMaskButtonPress,
	)

	err = xproto.CreateWindowChecked(
		d.conn,
		d.screen.RootDepth,
		winID,
		d.screen.Root,
		0, 0, defaultSize, defaultSize, 0,
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
	_ = xproto.CreateGCChecked(d.conn, gcID, xproto.Drawable(winID), 0, nil).Check()

	// Устанавливаем _XEMBED_INFO: версия 0, флаг XEMBED_MAPPED
	xembedAtom := d.internAtom("_XEMBED_INFO")
	if xembedAtom != 0 {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint32(buf[0:4], 0)
		binary.LittleEndian.PutUint32(buf[4:8], xembedMapped)
		_ = xproto.ChangePropertyChecked(d.conn, xproto.PropModeReplace, winID, xembedAtom, xembedAtom, 32, 2, buf).Check()
	}

	t := &x11Tray{
		disp:   d,
		cfg:    cfg,
		win:    winID,
		gc:     gcID,
		width:  defaultSize,
		height: defaultSize,
		rgba:   append([]byte(nil), cfg.IconRGBA...),
		iconW:  cfg.IconWidth,
		iconH:  cfg.IconHeight,
		menu:   append([]TrayMenuItem(nil), cfg.Menu...),
	}
	if len(t.rgba) == 0 || t.iconW <= 0 || t.iconH <= 0 {
		t.rgba, t.iconW, t.iconH = defaultLinuxTrayIconRGBA()
	}

	d.mu.Lock()
	d.tray = t
	d.mu.Unlock()

	if cfg.Tooltip != "" {
		_ = t.SetTooltip(cfg.Tooltip)
	}

	// Отправляем запрос на докинг в системный трей
	opcodeAtom := d.internAtom("_NET_SYSTEM_TRAY_OPCODE")
	if opcodeAtom != 0 {
		ev := xproto.ClientMessageEvent{
			Format: 32,
			Window: ownerReply.Owner,
			Type:   opcodeAtom,
			Data:   xproto.ClientMessageDataUnionData32New([]uint32{uint32(xproto.TimeCurrentTime), systemTrayRequestDock, uint32(winID), 0, 0}),
		}
		_ = xproto.SendEventChecked(d.conn, false, ownerReply.Owner, 0, string(ev.Bytes())).Check()
	}

	return t, nil
}

func (t *x11Tray) SetTooltip(tooltip string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed || t.win == 0 {
		return nil
	}
	b := []byte(tooltip)
	_ = xproto.ChangePropertyChecked(t.disp.conn, xproto.PropModeReplace, t.win, xproto.AtomWmName, xproto.AtomString, 8, uint32(len(b)), b).Check()
	netName := t.disp.internAtom("_NET_WM_NAME")
	utf8 := t.disp.internAtom("UTF8_STRING")
	if netName != 0 && utf8 != 0 {
		return xproto.ChangePropertyChecked(t.disp.conn, xproto.PropModeReplace, t.win, netName, utf8, 8, uint32(len(b)), b).Check()
	}
	return nil
}

func (t *x11Tray) SetIconRGBA(rgba []byte, width, height int) error {
	if width <= 0 || height <= 0 || len(rgba) < width*height*4 {
		return errors.New("некорректный буфер иконки трея")
	}
	t.mu.Lock()
	t.rgba = append([]byte(nil), rgba[:width*height*4]...)
	t.iconW = width
	t.iconH = height
	t.mu.Unlock()

	t.redraw()
	return nil
}

func (t *x11Tray) SetMenu(items []TrayMenuItem) {
	t.mu.Lock()
	t.menu = append([]TrayMenuItem(nil), items...)
	t.mu.Unlock()
}

func (t *x11Tray) ShowNotification(n TrayNotification) error {
	return sendLinuxNotification(n)
}

func (t *x11Tray) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	win := t.win
	gc := t.gc
	t.win = 0
	t.gc = 0
	t.mu.Unlock()

	t.disp.mu.Lock()
	if t.disp.tray == t {
		t.disp.tray = nil
	}
	t.disp.mu.Unlock()

	if gc != 0 {
		_ = xproto.FreeGCChecked(t.disp.conn, gc).Check()
	}
	if win != 0 {
		_ = xproto.DestroyWindowChecked(t.disp.conn, win).Check()
	}
	return nil
}

func (t *x11Tray) handleEvent(ev xgb.Event) bool {
	t.mu.Lock()
	win := t.win
	t.mu.Unlock()
	if win == 0 {
		return false
	}

	switch e := ev.(type) {
	case xproto.ExposeEvent:
		if e.Window == win {
			if e.Count == 0 {
				t.redraw()
			}
			return true
		}

	case xproto.ConfigureNotifyEvent:
		if e.Window == win {
			t.mu.Lock()
			if e.Width > 0 && e.Height > 0 {
				t.width = int(e.Width)
				t.height = int(e.Height)
			}
			t.mu.Unlock()
			t.redraw()
			return true
		}

	case xproto.ButtonPressEvent:
		if e.Event == win {
			switch e.Detail {
			case 1: // Левый клик
				now := time.Now()
				t.mu.Lock()
				isDbl := !t.lastClk.IsZero() && now.Sub(t.lastClk) <= 400*time.Millisecond
				if isDbl {
					t.lastClk = time.Time{}
				} else {
					t.lastClk = now
				}
				t.mu.Unlock()
				if isDbl && t.cfg.OnDoubleClick != nil {
					t.cfg.OnDoubleClick()
				} else if t.cfg.OnClick != nil {
					t.cfg.OnClick()
				}
			case 3: // Правый клик
				if t.cfg.OnRightClick != nil {
					t.cfg.OnRightClick()
				}
			}
			return true
		}
	}
	return false
}

func (t *x11Tray) redraw() {
	t.mu.Lock()
	win := t.win
	gc := t.gc
	w, h := t.width, t.height
	rgba := t.rgba
	iw, ih := t.iconW, t.iconH
	t.mu.Unlock()

	if win == 0 || gc == 0 || w <= 0 || h <= 0 || iw <= 0 || ih <= 0 || len(rgba) < iw*ih*4 {
		return
	}

	bgra := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		sy := y * ih / h
		for x := 0; x < w; x++ {
			sx := x * iw / w
			sOff := (sy*iw + sx) * 4
			dOff := (y*w + x) * 4
			r := rgba[sOff+0]
			g := rgba[sOff+1]
			b := rgba[sOff+2]
			a := rgba[sOff+3]
			// Простой альфа-блендинг на тёмно-серый фон панели
			bg := uint16(36)
			alpha := uint16(a)
			inv := 255 - alpha
			bgra[dOff+0] = byte((uint16(b)*alpha + bg*inv) / 255)
			bgra[dOff+1] = byte((uint16(g)*alpha + bg*inv) / 255)
			bgra[dOff+2] = byte((uint16(r)*alpha + bg*inv) / 255)
			bgra[dOff+3] = 255
		}
	}

	_ = xproto.PutImageChecked(
		t.disp.conn,
		xproto.ImageFormatZPixmap,
		xproto.Drawable(win),
		gc,
		uint16(w), uint16(h),
		0, 0,
		0,
		t.disp.screen.RootDepth,
		bgra,
	).Check()
}

func defaultLinuxTrayIconRGBA() ([]byte, int, int) {
	const size = 16
	buf := make([]byte, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			off := (y*size + x) * 4
			dx := float64(x) - 7.5
			dy := float64(y) - 7.5
			if dx*dx+dy*dy <= 49 {
				buf[off+0] = 66  // R
				buf[off+1] = 133 // G
				buf[off+2] = 244 // B
				buf[off+3] = 255 // A
			}
		}
	}
	return buf, size, size
}

