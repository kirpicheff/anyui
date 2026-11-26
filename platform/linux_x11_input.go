//go:build linux

package platform

import (
	"encoding/binary"
	"net/url"
	"strings"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type x11KeyMap struct {
	mu           sync.RWMutex
	minKeycode   xproto.Keycode
	maxKeycode   xproto.Keycode
	symsPerCode  int
	keysyms      []xproto.Keysym
}

func newX11KeyMap(conn *xgb.Conn, setup *xproto.SetupInfo) *x11KeyMap {
	km := &x11KeyMap{}
	km.refresh(conn, setup)
	return km
}

func (km *x11KeyMap) refresh(conn *xgb.Conn, setup *xproto.SetupInfo) {
	count := byte(setup.MaxKeycode - setup.MinKeycode + 1)
	reply, err := xproto.GetKeyboardMapping(conn, setup.MinKeycode, count).Reply()
	if err != nil || reply == nil {
		return
	}
	km.mu.Lock()
	km.minKeycode = setup.MinKeycode
	km.maxKeycode = setup.MaxKeycode
	km.symsPerCode = int(reply.KeysymsPerKeycode)
	km.keysyms = reply.Keysyms
	km.mu.Unlock()
}

func (km *x11KeyMap) lookup(code xproto.Keycode, state uint16) (xproto.Keysym, xproto.Keysym) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if km.symsPerCode <= 0 || code < km.minKeycode || code > km.maxKeycode {
		return 0, 0
	}
	idx := int(code-km.minKeycode) * km.symsPerCode
	if idx >= len(km.keysyms) {
		return 0, 0
	}

	latinBase := km.keysyms[idx]

	// Активная XKB-группа раскладки хранится в битах 13..14 поля state.
	group := int((state >> 13) & 0x03)
	groupIdx := idx + group*2
	if group > 0 && (group*2 >= km.symsPerCode || groupIdx >= len(km.keysyms) || km.keysyms[groupIdx] == 0) {
		groupIdx = idx
	}

	baseSym := km.keysyms[groupIdx]
	shiftSym := baseSym
	if groupIdx+1 < idx+km.symsPerCode && groupIdx+1 < len(km.keysyms) && km.keysyms[groupIdx+1] != 0 {
		shiftSym = km.keysyms[groupIdx+1]
	}

	shift := (state & xproto.ModMaskShift) != 0
	caps := (state & xproto.ModMaskLock) != 0

	if shift {
		return shiftSym, latinBase
	}
	if caps && isLowerLetterKeysym(baseSym) {
		return shiftSym, latinBase
	}
	return baseSym, latinBase
}

func isLowerLetterKeysym(sym xproto.Keysym) bool {
	return (sym >= 'a' && sym <= 'z') || sym == 0x06A3 || (sym >= 0x06C0 && sym <= 0x06DF) || (sym >= 0x01000430 && sym <= 0x0100044F)
}

func modifiersFromX11State(state uint16) Modifiers {
	return Modifiers{
		Shift: (state & xproto.ModMaskShift) != 0,
		Ctrl:  (state & xproto.ModMaskControl) != 0,
		Alt:   (state & xproto.ModMask1) != 0,
	}
}

func (w *x11Window) handleButtonPress(e xproto.ButtonPressEvent) {
	if w.handler == nil {
		return
	}
	mods := modifiersFromX11State(e.State)
	x, y := int(e.EventX), int(e.EventY)

	switch e.Detail {
	case 4: // Колесо вверх
		w.handler(Event{Type: EventMouseWheel, X: x, Y: y, WheelY: 1, Mods: mods})
		return
	case 5: // Колесо вниз
		w.handler(Event{Type: EventMouseWheel, X: x, Y: y, WheelY: -1, Mods: mods})
		return
	case 6: // Колесо влево
		w.handler(Event{Type: EventMouseWheel, X: x, Y: y, WheelX: -1, Mods: mods})
		return
	case 7: // Колесо вправо
		w.handler(Event{Type: EventMouseWheel, X: x, Y: y, WheelX: 1, Mods: mods})
		return
	}

	btn := mapX11MouseButton(e.Detail)
	if btn == 0 {
		return
	}

	// Обработка перетаскивания и ресайза в режиме Frameless
	if btn == MouseLeft && w.cfg.Frameless {
		borderPx := int(6 * w.scale)
		titleH := w.cfg.TitleBarHeight
		if titleH <= 0 {
			titleH = 36
		}
		titlePx := int(float32(titleH) * w.scale)
		ww, wh := w.Size()
		hit := ComputeFramelessHitTest(x, y, ww, wh, borderPx, titlePx, w.IsMaximized(), w.cfg.HitTest)
		if dir, ok := hitTestToNetWMMoveResize(hit); ok {
			_ = xproto.UngrabPointerChecked(w.disp.conn, xproto.TimeCurrentTime).Check()
			w.sendNetWMMoveResize(int(e.RootX), int(e.RootY), dir)
			return
		}
	}

	w.handler(Event{
		Type:   EventMouseDown,
		X:      x,
		Y:      y,
		Button: btn,
		Mods:   mods,
	})
}

func (w *x11Window) handleButtonRelease(e xproto.ButtonReleaseEvent) {
	if w.handler == nil {
		return
	}
	if e.Detail >= 4 && e.Detail <= 7 {
		return
	}
	btn := mapX11MouseButton(e.Detail)
	if btn == 0 {
		return
	}
	w.handler(Event{
		Type:   EventMouseUp,
		X:      int(e.EventX),
		Y:      int(e.EventY),
		Button: btn,
		Mods:   modifiersFromX11State(e.State),
	})
}

func mapX11MouseButton(detail xproto.Button) MouseButton {
	switch detail {
	case 1:
		return MouseLeft
	case 2:
		return MouseMiddle
	case 3:
		return MouseRight
	default:
		return 0
	}
}

func hitTestToNetWMMoveResize(r HitTestResult) (uint32, bool) {
	switch r {
	case HitTopLeft:
		return 0, true
	case HitTop:
		return 1, true
	case HitTopRight:
		return 2, true
	case HitRight:
		return 3, true
	case HitBottomRight:
		return 4, true
	case HitBottom:
		return 5, true
	case HitBottomLeft:
		return 6, true
	case HitLeft:
		return 7, true
	case HitCaption:
		return 8, true
	default:
		return 0, false
	}
}

func (w *x11Window) sendNetWMMoveResize(rootX, rootY int, direction uint32) {
	atom := w.disp.internAtom(netWMMoveResize)
	if atom == 0 {
		return
	}
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: w.win,
		Type:   atom,
		Data:   xproto.ClientMessageDataUnionData32New([]uint32{uint32(rootX), uint32(rootY), direction, 1, 1}),
	}
	_ = xproto.SendEventChecked(
		w.disp.conn, false, w.disp.screen.Root,
		xproto.EventMaskSubstructureRedirect|xproto.EventMaskSubstructureNotify,
		string(ev.Bytes()),
	).Check()
}

func (w *x11Window) handleKeyPress(e xproto.KeyPressEvent) {
	if w.handler == nil {
		return
	}
	mods := modifiersFromX11State(e.State)
	sym, baseSym := w.disp.keyMap.lookup(e.Detail, e.State)
	vk := x11KeysymToVK(baseSym)
	if vk == 0 {
		vk = x11KeysymToVK(sym)
	}

	if vk != 0 {
		w.handler(Event{
			Type:     EventKeyDown,
			KeyCode:  vk,
			ScanCode: uint16(e.Detail),
			Mods:     mods,
		})
	}

	if !mods.Ctrl && !mods.Alt {
		if ch := x11KeysymToRune(sym); ch != 0 {
			w.handler(Event{
				Type: EventChar,
				Char: ch,
				Mods: mods,
			})
		}
	}
}

func (w *x11Window) handleKeyRelease(e xproto.KeyReleaseEvent) {
	if w.handler == nil {
		return
	}
	mods := modifiersFromX11State(e.State)
	sym, baseSym := w.disp.keyMap.lookup(e.Detail, e.State)
	vk := x11KeysymToVK(baseSym)
	if vk == 0 {
		vk = x11KeysymToVK(sym)
	}
	if vk != 0 {
		w.handler(Event{
			Type:     EventKeyUp,
			KeyCode:  vk,
			ScanCode: uint16(e.Detail),
			Mods:     mods,
		})
	}
}

// x11KeysymToVK преобразует X11 KeySym в виртуальный код клавиши (совместимый с VK_* в ui/).
func x11KeysymToVK(sym xproto.Keysym) uint32 {
	if sym >= 'a' && sym <= 'z' {
		return uint32(sym - 'a' + 'A')
	}
	if sym >= 'A' && sym <= 'Z' {
		return uint32(sym)
	}
	if sym >= '0' && sym <= '9' {
		return uint32(sym)
	}
	switch sym {
	case 0xFF08: // BackSpace
		return 0x08
	case 0xFF09, 0xFE20: // Tab, ISO_Left_Tab
		return 0x09
	case 0xFF0D, 0xFF8D: // Return, KP_Enter
		return 0x0D
	case 0xFF1B: // Escape
		return 0x1B
	case 0x0020: // space
		return 0x20
	case 0xFF55: // Page_Up
		return 0x21
	case 0xFF56: // Page_Down
		return 0x22
	case 0xFF57: // End
		return 0x23
	case 0xFF50: // Home
		return 0x24
	case 0xFF51: // Left
		return 0x25
	case 0xFF52: // Up
		return 0x26
	case 0xFF53: // Right
		return 0x27
	case 0xFF54: // Down
		return 0x28
	case 0xFF63: // Insert
		return 0x2D
	case 0xFFFF: // Delete
		return 0x2E
	}
	// F1..F12
	if sym >= 0xFFBE && sym <= 0xFFC9 {
		return 0x70 + uint32(sym-0xFFBE)
	}
	return 0
}

// x11KeysymToRune преобразует X11 KeySym в Unicode-символ для EventChar.
func x11KeysymToRune(sym xproto.Keysym) rune {
	if sym >= 0x0020 && sym <= 0x007E {
		return rune(sym)
	}
	if sym >= 0x00A0 && sym <= 0x00FF {
		return rune(sym)
	}
	if sym >= 0x01000000 && sym <= 0x0110FFFF {
		return rune(sym - 0x01000000)
	}
	// Исторические X11 KeySym для кириллицы (0x06A1..0x06FF)
	if r := cyrillicKeysymToRune(sym); r != 0 {
		return r
	}
	return 0
}

var cyrillicKeysymTable = map[xproto.Keysym]rune{
	0x06A3: 'ё', 0x06B3: 'Ё',
	0x06C1: 'а', 0x06C2: 'б', 0x06D7: 'в', 0x06C7: 'г', 0x06C4: 'д',
	0x06C5: 'е', 0x06D6: 'ж', 0x06DA: 'з', 0x06C9: 'и', 0x06CA: 'й',
	0x06CB: 'к', 0x06CC: 'л', 0x06CD: 'м', 0x06CE: 'н', 0x06CF: 'о',
	0x06D0: 'п', 0x06D2: 'р', 0x06D3: 'с', 0x06D4: 'т', 0x06D5: 'у',
	0x06C6: 'ф', 0x06C8: 'х', 0x06C3: 'ц', 0x06DE: 'ч', 0x06DB: 'ш',
	0x06DD: 'щ', 0x06DF: 'ъ', 0x06D9: 'ы', 0x06D8: 'ь', 0x06DC: 'э',
	0x06C0: 'ю', 0x06D1: 'я',
	0x06E1: 'А', 0x06E2: 'Б', 0x06F7: 'В', 0x06E7: 'Г', 0x06E4: 'Д',
	0x06E5: 'Е', 0x06F6: 'Ж', 0x06FA: 'З', 0x06E9: 'И', 0x06EA: 'Й',
	0x06EB: 'К', 0x06EC: 'Л', 0x06ED: 'М', 0x06EE: 'Н', 0x06EF: 'О',
	0x06F0: 'П', 0x06F2: 'Р', 0x06F3: 'С', 0x06F4: 'Т', 0x06F5: 'У',
	0x06E6: 'Ф', 0x06E8: 'Х', 0x06E3: 'Ц', 0x06FE: 'Ч', 0x06FB: 'Ш',
	0x06FD: 'Щ', 0x06FF: 'Ъ', 0x06F9: 'Ы', 0x06F8: 'Ь', 0x06FC: 'Э',
	0x06E0: 'Ю', 0x06F1: 'Я',
}

func cyrillicKeysymToRune(sym xproto.Keysym) rune {
	return cyrillicKeysymTable[sym]
}

// --- XDND (Drag & Drop файлов) ---

func (w *x11Window) enableXDND() {
	aware := w.disp.internAtom("XdndAware")
	if aware == 0 {
		return
	}
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 5) // XDND version 5
	_ = xproto.ChangePropertyChecked(w.disp.conn, xproto.PropModeReplace, w.win, aware, xproto.AtomAtom, 32, 1, buf).Check()
}

func (w *x11Window) handleXDNDClientMessage(e xproto.ClientMessageEvent) {
	if len(e.Data.Data32) == 0 {
		return
	}
	switch e.Type {
	case w.disp.internAtom("XdndEnter"):
		w.xdndSource = xproto.Window(e.Data.Data32[0])

	case w.disp.internAtom("XdndPosition"):
		src := xproto.Window(e.Data.Data32[0])
		w.xdndSource = src
		action := w.disp.internAtom("XdndActionCopy")
		status := w.disp.internAtom("XdndStatus")
		ev := xproto.ClientMessageEvent{
			Format: 32,
			Window: src,
			Type:   status,
			Data:   xproto.ClientMessageDataUnionData32New([]uint32{uint32(w.win), 1, 0, 0, uint32(action)}),
		}
		_ = xproto.SendEventChecked(w.disp.conn, false, src, 0, string(ev.Bytes())).Check()

	case w.disp.internAtom("XdndDrop"):
		src := xproto.Window(e.Data.Data32[0])
		w.xdndSource = src
		ts := xproto.Timestamp(e.Data.Data32[2])
		sel := w.disp.internAtom("XdndSelection")
		target := w.disp.internAtom("text/uri-list")
		prop := w.disp.internAtom("_ANYUI_XDND_DATA")
		_ = xproto.ConvertSelectionChecked(w.disp.conn, w.win, sel, target, prop, ts).Check()
	}
}

func (w *x11Window) handleXDNDSelectionNotify(e xproto.SelectionNotifyEvent) {
	sel := w.disp.internAtom("XdndSelection")
	if e.Selection != sel {
		return
	}
	defer func() {
		if w.xdndSource != 0 {
			finished := w.disp.internAtom("XdndFinished")
			action := w.disp.internAtom("XdndActionCopy")
			ev := xproto.ClientMessageEvent{
				Format: 32,
				Window: w.xdndSource,
				Type:   finished,
				Data:   xproto.ClientMessageDataUnionData32New([]uint32{uint32(w.win), 1, uint32(action), 0, 0}),
			}
			_ = xproto.SendEventChecked(w.disp.conn, false, w.xdndSource, 0, string(ev.Bytes())).Check()
		}
	}()

	if e.Property == 0 {
		return
	}
	reply, err := xproto.GetProperty(w.disp.conn, true, w.win, e.Property, xproto.GetPropertyTypeAny, 0, 1<<20).Reply()
	if err != nil || reply == nil || len(reply.Value) == 0 {
		return
	}
	files := parseXDNDURIList(string(reply.Value))
	if len(files) > 0 && w.handler != nil {
		w.handler(Event{
			Type:  EventDropFiles,
			Files: files,
		})
	}
}

func parseXDNDURIList(raw string) []string {
	var files []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if u, err := url.Parse(line); err == nil && u.Scheme == "file" {
			if u.Path != "" {
				files = append(files, u.Path)
			}
		}
	}
	return files
}

