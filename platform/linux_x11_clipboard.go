//go:build linux

package platform

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/png"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type x11Clipboard struct {
	disp *x11Display

	mu       sync.Mutex
	ownerWin xproto.Window
	text     string
	pngData  []byte

	pendingMu     sync.Mutex
	pendingActive bool
	pendingProp   xproto.Atom
	pendingDone   bool
	pendingResult []byte
}

func newX11Clipboard(d *x11Display) *x11Clipboard {
	return &x11Clipboard{disp: d}
}

func (c *x11Clipboard) readText(win xproto.Window) (string, error) {
	utf8 := c.disp.internAtom("UTF8_STRING")
	data, err := c.readSelection(win, utf8)
	if err == nil {
		return string(data), nil
	}
	// Fallback на STRING
	data, err = c.readSelection(win, xproto.AtomString)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *x11Clipboard) writeText(win xproto.Window, text string) error {
	c.mu.Lock()
	c.ownerWin = win
	c.text = text
	c.pngData = nil
	c.mu.Unlock()

	clipAtom := c.disp.internAtom("CLIPBOARD")
	if clipAtom == 0 {
		return errors.New("атом CLIPBOARD недоступен")
	}
	return xproto.SetSelectionOwnerChecked(c.disp.conn, win, clipAtom, xproto.TimeCurrentTime).Check()
}

func (c *x11Clipboard) readImage(win xproto.Window) (image.Image, error) {
	pngAtom := c.disp.internAtom("image/png")
	if pngAtom == 0 {
		return nil, errors.New("атом image/png недоступен")
	}
	data, err := c.readSelection(win, pngAtom)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}

func (c *x11Clipboard) writeImage(win xproto.Window, img image.Image) error {
	if img == nil {
		return errors.New("пустое изображение")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}

	c.mu.Lock()
	c.ownerWin = win
	c.text = ""
	c.pngData = buf.Bytes()
	c.mu.Unlock()

	clipAtom := c.disp.internAtom("CLIPBOARD")
	if clipAtom == 0 {
		return errors.New("атом CLIPBOARD недоступен")
	}
	return xproto.SetSelectionOwnerChecked(c.disp.conn, win, clipAtom, xproto.TimeCurrentTime).Check()
}

func (c *x11Clipboard) readSelection(win xproto.Window, target xproto.Atom) ([]byte, error) {
	clipAtom := c.disp.internAtom("CLIPBOARD")
	propAtom := c.disp.internAtom("_ANYUI_CLIP_DATA")
	if clipAtom == 0 || propAtom == 0 {
		return nil, errors.New("атомы буфера обмена недоступны")
	}

	// Если мы сами владеем буфером обмена, возвращаем данные напрямую без X11 roundtrip
	ownerReply, err := xproto.GetSelectionOwner(c.disp.conn, clipAtom).Reply()
	if err == nil && ownerReply != nil && ownerReply.Owner != 0 {
		c.mu.Lock()
		isSelf := ownerReply.Owner == c.ownerWin
		txt := c.text
		pngBytes := c.pngData
		c.mu.Unlock()
		if isSelf {
			if target == c.disp.internAtom("image/png") {
				if len(pngBytes) > 0 {
					return pngBytes, nil
				}
				return nil, errors.New("в буфере нет изображения")
			}
			return []byte(txt), nil
		}
	}

	c.pendingMu.Lock()
	c.pendingActive = true
	c.pendingProp = propAtom
	c.pendingDone = false
	c.pendingResult = nil
	c.pendingMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		c.pendingActive = false
		c.pendingMu.Unlock()
	}()

	if err := xproto.ConvertSelectionChecked(c.disp.conn, win, clipAtom, target, propAtom, xproto.TimeCurrentTime).Check(); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		c.pendingMu.Lock()
		done := c.pendingDone
		res := c.pendingResult
		c.pendingMu.Unlock()
		if done {
			if res == nil {
				return nil, errors.New("буфер обмена пуст или формат не поддерживается")
			}
			return res, nil
		}

		ev, pollErr := c.disp.conn.PollForEvent()
		if pollErr != nil {
			return nil, pollErr
		}
		if ev != nil {
			c.disp.dispatchEvent(ev)
			continue
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil, errors.New("таймаут чтения буфера обмена X11")
}

func (c *x11Clipboard) handleEvent(ev xgb.Event) bool {
	switch e := ev.(type) {
	case xproto.SelectionNotifyEvent:
		clipAtom := c.disp.internAtom("CLIPBOARD")
		if e.Selection != clipAtom {
			return false
		}
		c.pendingMu.Lock()
		active := c.pendingActive
		prop := c.pendingProp
		c.pendingMu.Unlock()
		if !active {
			return false
		}

		var result []byte
		if e.Property != 0 && e.Property == prop {
			if reply, err := xproto.GetProperty(c.disp.conn, true, e.Requestor, e.Property, xproto.GetPropertyTypeAny, 0, 16<<20).Reply(); err == nil && reply != nil {
				result = append([]byte(nil), reply.Value...)
			}
		}
		c.pendingMu.Lock()
		c.pendingResult = result
		c.pendingDone = true
		c.pendingMu.Unlock()
		return true

	case xproto.SelectionRequestEvent:
		clipAtom := c.disp.internAtom("CLIPBOARD")
		if e.Selection != clipAtom {
			return false
		}
		c.handleSelectionRequest(e)
		return true

	case xproto.SelectionClearEvent:
		clipAtom := c.disp.internAtom("CLIPBOARD")
		if e.Selection == clipAtom {
			c.mu.Lock()
			c.ownerWin = 0
			c.text = ""
			c.pngData = nil
			c.mu.Unlock()
			return true
		}
	}
	return false
}

func (c *x11Clipboard) handleSelectionRequest(e xproto.SelectionRequestEvent) {
	c.mu.Lock()
	txt := c.text
	pngBytes := c.pngData
	c.mu.Unlock()

	targetsAtom := c.disp.internAtom("TARGETS")
	utf8Atom := c.disp.internAtom("UTF8_STRING")
	pngAtom := c.disp.internAtom("image/png")

	prop := e.Property
	if prop == 0 {
		prop = e.Target
	}

	replyProp := xproto.Atom(0)
	switch e.Target {
	case targetsAtom:
		var atoms []xproto.Atom
		atoms = append(atoms, targetsAtom)
		if len(pngBytes) > 0 {
			atoms = append(atoms, pngAtom)
		} else {
			atoms = append(atoms, utf8Atom, xproto.AtomString)
		}
		buf := make([]byte, len(atoms)*4)
		for i, a := range atoms {
			binary.LittleEndian.PutUint32(buf[i*4:], uint32(a))
		}
		if xproto.ChangePropertyChecked(c.disp.conn, xproto.PropModeReplace, e.Requestor, prop, xproto.AtomAtom, 32, uint32(len(atoms)), buf).Check() == nil {
			replyProp = prop
		}

	case utf8Atom, xproto.AtomString:
		if len(pngBytes) == 0 {
			b := []byte(txt)
			if xproto.ChangePropertyChecked(c.disp.conn, xproto.PropModeReplace, e.Requestor, prop, e.Target, 8, uint32(len(b)), b).Check() == nil {
				replyProp = prop
			}
		}

	case pngAtom:
		if len(pngBytes) > 0 {
			if xproto.ChangePropertyChecked(c.disp.conn, xproto.PropModeReplace, e.Requestor, prop, pngAtom, 8, uint32(len(pngBytes)), pngBytes).Check() == nil {
				replyProp = prop
			}
		}
	}

	notify := xproto.SelectionNotifyEvent{
		Time:      e.Time,
		Requestor: e.Requestor,
		Selection: e.Selection,
		Target:    e.Target,
		Property:  replyProp,
	}
	_ = xproto.SendEventChecked(c.disp.conn, false, e.Requestor, 0, string(notify.Bytes())).Check()
}

// ClipboardImage реализует platform.ClipboardImager для X11.
func (w *x11Window) ClipboardImage() (image.Image, error) {
	return w.disp.clip.readImage(w.win)
}

// SetClipboardImage реализует platform.ClipboardImager для X11.
func (w *x11Window) SetClipboardImage(img image.Image) error {
	return w.disp.clip.writeImage(w.win, img)
}

