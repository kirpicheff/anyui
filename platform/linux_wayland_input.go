//go:build linux

package platform

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	// wl_seat capabilities
	wlSeatCapPointer  uint32 = 1
	wlSeatCapKeyboard uint32 = 2

	// wl_seat requests
	wlSeatGetPointer  uint16 = 0
	wlSeatGetKeyboard uint16 = 1

	// wl_seat events
	wlSeatEventCapabilities uint16 = 0

	// wl_pointer requests
	wlPointerSetCursor uint16 = 0

	// wl_pointer events
	wlPointerEventEnter        uint16 = 0
	wlPointerEventLeave        uint16 = 1
	wlPointerEventMotion       uint16 = 2
	wlPointerEventButton       uint16 = 3
	wlPointerEventAxis         uint16 = 4
	wlPointerEventFrame        uint16 = 5
	wlPointerEventAxisValue120 uint16 = 9

	// wl_keyboard events
	wlKeyboardEventKeymap     uint16 = 0
	wlKeyboardEventEnter      uint16 = 1
	wlKeyboardEventLeave      uint16 = 2
	wlKeyboardEventKey        uint16 = 3
	wlKeyboardEventModifiers  uint16 = 4
	wlKeyboardEventRepeatInfo uint16 = 5

	// wp_cursor_shape_manager_v1 / wp_cursor_shape_device_v1
	wpCursorShapeMgrGetPointer uint16 = 1
	wpCursorShapeDevDestroy    uint16 = 0
	wpCursorShapeDevSetShape   uint16 = 1

	// linux/input-event-codes.h кнопки мыши
	evBtnLeft   uint32 = 0x110
	evBtnRight  uint32 = 0x111
	evBtnMiddle uint32 = 0x112
)

type wlSeatState struct {
	disp *wlDisplay

	seatID       uint32
	pointerID    uint32
	keyboardID   uint32
	cursorDevID  uint32
	hasCursorMgr bool

	mu                 sync.Mutex
	pointerSurfaceID   uint32
	pointerEnterSerial uint32
	lastInputSerial    uint32
	lastPointerX       float64
	lastPointerY       float64
	pendingMotion      bool
	pendingAxisY       float32
	pendingAxisX       float32
	lastCursorApplied  CursorType
	lastCursorSerial   uint32

	keyboardSurfaceID uint32
	mods              Modifiers
	capsLock          bool
	xkbGroup          uint32
	keymap            *wlKeymap

	repeatRate  int
	repeatDelay int
	repeatStop  chan struct{}
}

func newWLSeatState(d *wlDisplay, seatID uint32) *wlSeatState {
	s := &wlSeatState{
		disp:        d,
		seatID:      seatID,
		repeatRate:  25,
		repeatDelay: 400,
	}
	d.conn.setHandler(seatID, s.handleSeatEvent)
	return s
}

func (s *wlSeatState) handleSeatEvent(ev wlEvent) {
	if ev.Opcode != wlSeatEventCapabilities {
		return
	}
	r := wlReader{data: ev.Payload}
	caps, ok := r.uint32()
	if !ok {
		return
	}

	if caps&wlSeatCapPointer != 0 && s.pointerID == 0 {
		s.pointerID = s.disp.conn.allocID(s.handlePointerEvent)
		var w wlWriter
		w.uint32(s.pointerID)
		_ = s.disp.conn.send(s.seatID, wlSeatGetPointer, w.bytes())

		if s.disp.cursorShapeMgrID != 0 {
			s.cursorDevID = s.disp.conn.allocID(nil)
			var cw wlWriter
			cw.uint32(s.cursorDevID)
			cw.uint32(s.pointerID)
			if err := s.disp.conn.send(s.disp.cursorShapeMgrID, wpCursorShapeMgrGetPointer, cw.bytes()); err == nil {
				s.hasCursorMgr = true
			}
		}
	}

	if caps&wlSeatCapKeyboard != 0 && s.keyboardID == 0 {
		s.keyboardID = s.disp.conn.allocID(s.handleKeyboardEvent)
		s.disp.conn.registerFDEvent(s.keyboardID, wlKeyboardEventKeymap)
		var w wlWriter
		w.uint32(s.keyboardID)
		_ = s.disp.conn.send(s.seatID, wlSeatGetKeyboard, w.bytes())
	}
}

func (s *wlSeatState) handlePointerEvent(ev wlEvent) {
	r := wlReader{data: ev.Payload}

	switch ev.Opcode {
	case wlPointerEventEnter:
		serial, ok1 := r.uint32()
		surfID, ok2 := r.uint32()
		sx, ok3 := r.fixed()
		sy, ok4 := r.fixed()
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return
		}
		s.mu.Lock()
		s.pointerSurfaceID = surfID
		s.pointerEnterSerial = serial
		s.lastPointerX = sx
		s.lastPointerY = sy
		mods := s.mods
		s.mu.Unlock()

		if win := s.disp.windowBySurface(surfID); win != nil {
			s.applyCursor(win.currentCursor())
			px, py := win.logicalToPhys(sx, sy)
			if !win.handleCSDMouseMove(px, py) {
				cy := py - win.csdHeightPx()
				if cy >= 0 {
					win.emit(Event{
						Type: EventMouseMove,
						X:    px,
						Y:    cy,
						Mods: mods,
					})
				}
			}
		}

	case wlPointerEventLeave:
		_, _ = r.uint32() // serial
		surfID, ok := r.uint32()
		if !ok {
			return
		}
		s.mu.Lock()
		if s.pointerSurfaceID == surfID {
			s.pointerSurfaceID = 0
		}
		s.mu.Unlock()

	case wlPointerEventMotion:
		_, ok1 := r.uint32() // time
		sx, ok2 := r.fixed()
		sy, ok3 := r.fixed()
		if !ok1 || !ok2 || !ok3 {
			return
		}
		s.mu.Lock()
		s.lastPointerX = sx
		s.lastPointerY = sy
		s.pendingMotion = true
		s.mu.Unlock()

	case wlPointerEventButton:
		serial, ok1 := r.uint32()
		_, ok2 := r.uint32() // time
		btnCode, ok3 := r.uint32()
		state, ok4 := r.uint32()
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return
		}
		btn := mapWaylandMouseButton(btnCode)
		if btn == 0 {
			return
		}

		s.mu.Lock()
		s.lastInputSerial = serial
		surfID := s.pointerSurfaceID
		sx := s.lastPointerX
		sy := s.lastPointerY
		mods := s.mods
		s.mu.Unlock()

		win := s.disp.windowBySurface(surfID)
		if win == nil {
			return
		}
		px, py := win.logicalToPhys(sx, sy)

		if state == 1 {
			if win.handleCSDMouseDown(px, py, btn, s.seatID, serial) {
				return
			}
			cy := py - win.csdHeightPx()
			if btn == MouseLeft && win.tryFramelessMoveResize(px, cy, s.seatID, serial) {
				return
			}
			win.emit(Event{
				Type:   EventMouseDown,
				X:      px,
				Y:      cy,
				Button: btn,
				Mods:   mods,
			})
		} else {
			cy := py - win.csdHeightPx()
			win.emit(Event{
				Type:   EventMouseUp,
				X:      px,
				Y:      cy,
				Button: btn,
				Mods:   mods,
			})
		}

	case wlPointerEventAxis:
		_, ok1 := r.uint32() // time
		axis, ok2 := r.uint32()
		val, ok3 := r.fixed()
		if !ok1 || !ok2 || !ok3 {
			return
		}
		// В Wayland положительное значение по вертикали — прокрутка вниз, в GoUI WheelY > 0 — вверх.
		delta := float32(-val / 15.0)
		s.mu.Lock()
		if axis == 0 {
			s.pendingAxisY += delta
		} else if axis == 1 {
			s.pendingAxisX += float32(val / 15.0)
		}
		s.mu.Unlock()

	case wlPointerEventAxisValue120:
		axis, ok1 := r.uint32()
		v120, ok2 := r.int32()
		if !ok1 || !ok2 {
			return
		}
		s.mu.Lock()
		if axis == 0 {
			s.pendingAxisY = float32(-v120) / 120.0
		} else if axis == 1 {
			s.pendingAxisX = float32(v120) / 120.0
		}
		s.mu.Unlock()

	case wlPointerEventFrame:
		s.mu.Lock()
		moved := s.pendingMotion
		s.pendingMotion = false
		axX := s.pendingAxisX
		axY := s.pendingAxisY
		s.pendingAxisX = 0
		s.pendingAxisY = 0
		surfID := s.pointerSurfaceID
		sx := s.lastPointerX
		sy := s.lastPointerY
		mods := s.mods
		s.mu.Unlock()

		win := s.disp.windowBySurface(surfID)
		if win == nil {
			return
		}
		px, py := win.logicalToPhys(sx, sy)
		if moved {
			if !win.handleCSDMouseMove(px, py) {
				cy := py - win.csdHeightPx()
				win.emit(Event{
					Type: EventMouseMove,
					X:    px,
					Y:    cy,
					Mods: mods,
				})
			}
		}
		if axX != 0 || axY != 0 {
			cy := py - win.csdHeightPx()
			win.emit(Event{
				Type:   EventMouseWheel,
				X:      px,
				Y:      cy,
				WheelX: axX,
				WheelY: axY,
				Mods:   mods,
			})
		}
	}
}

func (s *wlSeatState) applyCursor(cursor CursorType) {
	s.mu.Lock()
	serial := s.pointerEnterSerial
	hasMgr := s.hasCursorMgr
	devID := s.cursorDevID
	ptrID := s.pointerID
	if serial == 0 || (s.lastCursorSerial == serial && s.lastCursorApplied == cursor) {
		s.mu.Unlock()
		return
	}
	s.lastCursorSerial = serial
	s.lastCursorApplied = cursor
	s.mu.Unlock()

	if cursor == CursorNone {
		if ptrID != 0 {
			// В Wayland скрытие курсора выполняется передачей nil (0) surface в wl_pointer.set_cursor.
			var pw wlWriter
			pw.uint32(serial)
			pw.uint32(0)
			pw.int32(0)
			pw.int32(0)
			_ = s.disp.conn.send(ptrID, wlPointerSetCursor, pw.bytes())
		}
		return
	}

	if hasMgr && devID != 0 {
		shape := waylandCursorShape(cursor)
		var w wlWriter
		w.uint32(serial)
		w.uint32(shape)
		_ = s.disp.conn.send(devID, wpCursorShapeDevSetShape, w.bytes())
		return
	}

	if s.disp.cursorCache != nil && ptrID != 0 {
		s.disp.cursorCache.setCursor(ptrID, serial, cursor)
	}
}

func waylandCursorShape(cursor CursorType) uint32 {
	switch cursor {
	case CursorHand:
		return 4 // pointer
	case CursorIBeam:
		return 9 // text
	case CursorResizeH:
		return 26 // ew_resize
	case CursorResizeV:
		return 27 // ns_resize
	case CursorResizeNWSE:
		return 29 // nwse_resize
	case CursorResizeNESW:
		return 28 // nesw_resize
	case CursorNotAllowed:
		return 15 // not_allowed
	case CursorCrosshair:
		return 8 // crosshair
	case CursorMove:
		return 13 // move
	default:
		return 1 // default
	}
}

func mapWaylandMouseButton(code uint32) MouseButton {
	switch code {
	case evBtnLeft:
		return MouseLeft
	case evBtnRight:
		return MouseRight
	case evBtnMiddle:
		return MouseMiddle
	default:
		return 0
	}
}

func (s *wlSeatState) handleKeyboardEvent(ev wlEvent) {
	r := wlReader{data: ev.Payload}

	switch ev.Opcode {
	case wlKeyboardEventKeymap:
		format, ok1 := r.uint32()
		size, ok2 := r.uint32()
		if !ok1 || !ok2 || len(ev.FDs) == 0 {
			return
		}
		fd := ev.FDs[0]
		ev.FDs[0] = -1
		defer func() { _ = unix.Close(fd) }()

		if format == 1 && size > 0 && size < 4*1024*1024 {
			data, err := unix.Mmap(fd, 0, int(size), unix.PROT_READ, unix.MAP_PRIVATE)
			if err == nil {
				km := parseXKBKeymap(string(data))
				_ = unix.Munmap(data)
				s.mu.Lock()
				s.keymap = km
				s.mu.Unlock()
			}
		}

	case wlKeyboardEventEnter:
		serial, _ := r.uint32()
		surfID, ok := r.uint32()
		if !ok {
			return
		}
		s.mu.Lock()
		s.keyboardSurfaceID = surfID
		if serial != 0 {
			s.lastInputSerial = serial
		}
		s.mu.Unlock()

	case wlKeyboardEventLeave:
		_, _ = r.uint32() // serial
		surfID, ok := r.uint32()
		if !ok {
			return
		}
		s.stopRepeat()
		s.mu.Lock()
		wasFocused := s.keyboardSurfaceID == surfID
		if wasFocused {
			s.keyboardSurfaceID = 0
		}
		s.mu.Unlock()
		if wasFocused {
			if win := s.disp.windowBySurface(surfID); win != nil {
				win.emit(Event{Type: EventFocusLost})
			}
		}

	case wlKeyboardEventKey:
		serial, ok1 := r.uint32()
		_, ok2 := r.uint32() // time
		key, ok3 := r.uint32()
		state, ok4 := r.uint32()
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return
		}

		s.mu.Lock()
		if serial != 0 {
			s.lastInputSerial = serial
		}
		surfID := s.keyboardSurfaceID
		mods := s.mods
		caps := s.capsLock
		group := s.xkbGroup
		km := s.keymap
		s.mu.Unlock()

		win := s.disp.windowBySurface(surfID)
		if win == nil {
			return
		}

		vk, ch := translateEvdevKey(key, mods, caps, group, km)
		// Приводим evdev keycode к X11/XKB keycode (+8), чтобы ScanCode на Linux
		// совпадал между бэкендами Wayland и X11.
		scanCode := uint16(key + 8)
		ext := evdevIsExtended(key)
		var keyChar rune
		if ch != 0 && !mods.Ctrl && !mods.Alt {
			keyChar = ch
		}
		if state == 1 || state == 2 {
			if vk != 0 {
				win.emit(Event{
					Type:     EventKeyDown,
					KeyCode:  vk,
					ScanCode: scanCode,
					Extended: ext,
					Char:     keyChar,
					Mods:     mods,
				})
			}
			if keyChar != 0 {
				win.emit(Event{
					Type: EventChar,
					Char: keyChar,
					Mods: mods,
				})
			}
			if state == 1 {
				s.startRepeat(surfID, key)
			}
		} else {
			s.stopRepeat()
			if vk != 0 {
				win.emit(Event{
					Type:     EventKeyUp,
					KeyCode:  vk,
					ScanCode: scanCode,
					Extended: ext,
					Mods:     mods,
				})
			}
		}

	case wlKeyboardEventModifiers:
		_, ok1 := r.uint32() // serial
		depressed, ok2 := r.uint32()
		_, ok3 := r.uint32() // latched
		locked, ok4 := r.uint32()
		group, ok5 := r.uint32()
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
			return
		}
		// Стандартные биты XKB модификаторов: Shift=0, Lock=1, Control=2, Mod1(Alt)=3.
		s.mu.Lock()
		s.mods = Modifiers{
			Shift: depressed&(1<<0) != 0,
			Ctrl:  depressed&(1<<2) != 0,
			Alt:   depressed&(1<<3) != 0,
		}
		s.capsLock = locked&(1<<1) != 0
		s.xkbGroup = group
		s.mu.Unlock()

	case wlKeyboardEventRepeatInfo:
		rate, ok1 := r.int32()
		delay, ok2 := r.int32()
		if !ok1 || !ok2 {
			return
		}
		s.mu.Lock()
		s.repeatRate = int(rate)
		s.repeatDelay = int(delay)
		s.mu.Unlock()
	}
}

func (s *wlSeatState) startRepeat(surfID uint32, key uint32) {
	s.stopRepeat()

	s.mu.Lock()
	rate := s.repeatRate
	delay := s.repeatDelay
	if rate <= 0 || delay <= 0 {
		s.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	s.repeatStop = stop
	s.mu.Unlock()

	interval := time.Second / time.Duration(rate)
	go func() {
		timer := time.NewTimer(time.Duration(delay) * time.Millisecond)
		defer timer.Stop()

		select {
		case <-stop:
			return
		case <-timer.C:
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				s.mu.Lock()
				curSurf := s.keyboardSurfaceID
				mods := s.mods
				caps := s.capsLock
				group := s.xkbGroup
				km := s.keymap
				s.mu.Unlock()

				if curSurf != surfID {
					return
				}
				win := s.disp.windowBySurface(surfID)
				if win == nil {
					return
				}
				vk, ch := translateEvdevKey(key, mods, caps, group, km)
				var keyChar rune
				if ch != 0 && !mods.Ctrl && !mods.Alt {
					keyChar = ch
				}
				if vk != 0 {
					win.emit(Event{
						Type:     EventKeyDown,
						KeyCode:  vk,
						ScanCode: uint16(key + 8),
						Extended: evdevIsExtended(key),
						Char:     keyChar,
						Mods:     mods,
					})
				}
				if keyChar != 0 {
					win.emit(Event{
						Type: EventChar,
						Char: keyChar,
						Mods: mods,
					})
				}
				win.WakeUp()
			}
		}
	}()
}

func (s *wlSeatState) stopRepeat() {
	s.mu.Lock()
	if s.repeatStop != nil {
		close(s.repeatStop)
		s.repeatStop = nil
	}
	s.mu.Unlock()
}

// wlKeymap хранит упрощённое отображение evdev keycode -> символы по группам раскладки XKB.
type wlKeymap struct {
	keycodes map[string]uint32   // "<AD01>" -> evdev keycode (XKB keycode - 8)
	symbols  map[uint32][][]rune // evdev keycode -> [group][level]
}

// parseXKBKeymap извлекает keycodes и символы из текстового XKB keymap без CGO.
// Поддерживает как однострочную, так и многострочную форму GNOME Mutter / libxkbcommon.
func parseXKBKeymap(text string) *wlKeymap {
	km := &wlKeymap{
		keycodes: make(map[string]uint32),
		symbols:  make(map[uint32][][]rune),
	}

	inKeycodes := false
	inSymbols := false
	inKeyBlock := false
	var keyName string
	var keyBuf strings.Builder

	flushKeyBlock := func() {
		if keyName != "" {
			if evCode, ok := km.keycodes[keyName]; ok {
				groups := parseXKBSymbolGroups(keyBuf.String())
				if len(groups) > 0 {
					km.symbols[evCode] = groups
				}
			}
		}
		inKeyBlock = false
		keyName = ""
		keyBuf.Reset()
	}

	for _, rawLine := range strings.Split(text, "\n") {
		line := strings.TrimSpace(rawLine)
		if !inKeyBlock {
			if strings.HasPrefix(line, "xkb_keycodes") {
				inKeycodes = true
				inSymbols = false
				continue
			}
			if strings.HasPrefix(line, "xkb_symbols") {
				inSymbols = true
				inKeycodes = false
				continue
			}
			if line == "};" {
				inKeycodes = false
				inSymbols = false
				continue
			}
		}

		if inKeycodes && strings.HasPrefix(line, "<") {
			// Формат: <AD01> = 24;
			eq := strings.IndexByte(line, '=')
			if eq > 0 {
				name := strings.TrimSpace(line[:eq])
				valStr := strings.TrimSpace(strings.TrimSuffix(line[eq+1:], ";"))
				if kc, err := strconv.ParseUint(valStr, 10, 32); err == nil && kc >= 8 {
					km.keycodes[name] = uint32(kc - 8)
				}
			}
			continue
		}

		if inSymbols {
			if inKeyBlock {
				keyBuf.WriteString(line)
				keyBuf.WriteByte('\n')
				if strings.Contains(line, "};") {
					flushKeyBlock()
				}
				continue
			}

			keyLine := line
			if strings.HasPrefix(keyLine, "override ") {
				keyLine = strings.TrimSpace(strings.TrimPrefix(keyLine, "override "))
			}
			if strings.HasPrefix(keyLine, "key <") {
				gt := strings.IndexByte(keyLine, '>')
				if gt <= 5 {
					continue
				}
				keyName = keyLine[4 : gt+1]
				rest := keyLine[gt+1:]
				keyBuf.Reset()
				keyBuf.WriteString(rest)
				keyBuf.WriteByte('\n')
				if strings.Contains(rest, "};") {
					flushKeyBlock()
				} else {
					inKeyBlock = true
				}
			}
		}
	}

	if len(km.symbols) == 0 {
		return nil
	}
	return km
}

func parseXKBSymbolGroups(block string) [][]rune {
	lower := strings.ToLower(block)
	// Формат GNOME Mutter / libxkbcommon: symbols[Group1]= [ q, Q ], symbols[Group2]= [ Cyrillic_shorti, Cyrillic_SHORTI ]
	if strings.Contains(lower, "symbols[group") {
		var groups [][]rune
		idx := 0
		for idx < len(lower) {
			pos := strings.Index(lower[idx:], "symbols[group")
			if pos < 0 {
				break
			}
			numStart := idx + pos + len("symbols[group")
			rb := strings.IndexByte(block[numStart:], ']')
			if rb <= 0 {
				break
			}
			groupNum, err := strconv.Atoi(strings.TrimSpace(block[numStart : numStart+rb]))
			afterRB := numStart + rb + 1
			open := strings.IndexByte(block[afterRB:], '[')
			if open < 0 {
				break
			}
			listStart := afterRB + open + 1
			closeIdx := strings.IndexByte(block[listStart:], ']')
			if closeIdx < 0 {
				break
			}
			if err == nil && groupNum >= 1 && groupNum <= 8 {
				for len(groups) < groupNum {
					groups = append(groups, nil)
				}
				groups[groupNum-1] = parseXKBSymbolList(block[listStart : listStart+closeIdx])
			}
			idx = listStart + closeIdx + 1
		}
		if len(groups) > 0 {
			return groups
		}
	}

	// Короткая форма: [ q, Q ], [ Cyrillic_shorti, Cyrillic_SHORTI ]
	var groups [][]rune
	idx := 0
	for idx < len(block) {
		open := strings.IndexByte(block[idx:], '[')
		if open < 0 {
			break
		}
		openAbs := idx + open
		start := openAbs + 1
		closeIdx := strings.IndexByte(block[start:], ']')
		if closeIdx < 0 {
			break
		}
		// Пропускаем скобки вида type[group1] или actions[group1], где перед '[' стоит буква/цифра.
		if openAbs > 0 && isXKBIdentChar(block[openAbs-1]) {
			idx = start + closeIdx + 1
			continue
		}
		groups = append(groups, parseXKBSymbolList(block[start:start+closeIdx]))
		idx = start + closeIdx + 1
	}
	return groups
}

func parseXKBSymbolList(content string) []rune {
	parts := strings.Split(content, ",")
	levels := make([]rune, 0, len(parts))
	for _, p := range parts {
		symName := strings.TrimSpace(p)
		levels = append(levels, xkbSymNameToRune(symName))
	}
	return levels
}

func isXKBIdentChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func translateEvdevKey(evCode uint32, mods Modifiers, caps bool, group uint32, km *wlKeymap) (uint32, rune) {
	vk := evdevToVK(evCode)

	if km != nil {
		if groups, ok := km.symbols[evCode]; ok && len(groups) > 0 {
			g := int(group)
			if g >= len(groups) {
				g = 0
			}
			levels := groups[g]
			if len(levels) > 0 {
				base := levels[0]
				shift := base
				if len(levels) > 1 && levels[1] != 0 {
					shift = levels[1]
				}
				upper := mods.Shift
				if caps && isLetterRune(base) {
					upper = !upper
				}
				ch := base
				if upper {
					ch = shift
				}
				if ch != 0 {
					return vk, ch
				}
			}
		}
	}

	ch := evdevToFallbackRune(evCode, mods.Shift, caps)
	return vk, ch
}

func isLetterRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'а' && r <= 'я') || r == 'ё'
}

func evdevIsExtended(code uint32) bool {
	switch code {
	case 96, // KP_Enter
		97,  // Right Ctrl
		98,  // KP_Slash
		100, // Right Alt
		102, // Home
		103, // Up
		104, // PageUp
		105, // Left
		106, // Right
		107, // End
		108, // Down
		109, // PageDown
		110, // Insert
		111, // Delete
		125, // Left Meta/Win
		126, // Right Meta/Win
		127: // Apps/Menu
		return true
	default:
		return false
	}
}

func evdevToVK(code uint32) uint32 {
	switch code {
	case 1:
		return 0x1B // Escape
	case 2, 3, 4, 5, 6, 7, 8, 9, 10:
		return uint32('1' + (code - 2))
	case 11:
		return '0'
	case 12:
		return 0xBD // VK_OEM_MINUS ('-')
	case 13:
		return 0xBB // VK_OEM_PLUS ('=')
	case 14:
		return 0x08 // BackSpace
	case 15:
		return 0x09 // Tab
	case 16:
		return 'Q'
	case 17:
		return 'W'
	case 18:
		return 'E'
	case 19:
		return 'R'
	case 20:
		return 'T'
	case 21:
		return 'Y'
	case 22:
		return 'U'
	case 23:
		return 'I'
	case 24:
		return 'O'
	case 25:
		return 'P'
	case 26:
		return 0xDB // VK_OEM_4 ('[' / 'х')
	case 27:
		return 0xDD // VK_OEM_6 (']' / 'ъ')
	case 28, 96:
		return 0x0D // Return / KP_Enter
	case 29, 97:
		return 0x11 // VK_CONTROL (Left / Right)
	case 30:
		return 'A'
	case 31:
		return 'S'
	case 32:
		return 'D'
	case 33:
		return 'F'
	case 34:
		return 'G'
	case 35:
		return 'H'
	case 36:
		return 'J'
	case 37:
		return 'K'
	case 38:
		return 'L'
	case 39:
		return 0xBA // VK_OEM_1 (';' / 'ж')
	case 40:
		return 0xDE // VK_OEM_7 ('\'' / 'э')
	case 41:
		return 0xC0 // VK_OEM_3 ('`' / 'ё')
	case 42, 54:
		return 0x10 // VK_SHIFT (Left / Right)
	case 43:
		return 0xDC // VK_OEM_5 ('\\')
	case 44:
		return 'Z'
	case 45:
		return 'X'
	case 46:
		return 'C'
	case 47:
		return 'V'
	case 48:
		return 'B'
	case 49:
		return 'N'
	case 50:
		return 'M'
	case 51:
		return 0xBC // VK_OEM_COMMA (',' / 'б')
	case 52:
		return 0xBE // VK_OEM_PERIOD ('.' / 'ю')
	case 53:
		return 0xBF // VK_OEM_2 ('/' / '.')
	case 55:
		return 0x6A // VK_MULTIPLY (KP_*)
	case 56, 100:
		return 0x12 // VK_MENU (Left / Right Alt)
	case 57:
		return 0x20 // Space
	case 58:
		return 0x14 // VK_CAPITAL (CapsLock)
	case 59, 60, 61, 62, 63, 64, 65, 66, 67, 68:
		return 0x70 + (code - 59) // F1..F10
	case 69:
		return 0x90 // VK_NUMLOCK
	case 70:
		return 0x91 // VK_SCROLL
	case 71:
		return 0x67 // VK_NUMPAD7
	case 72:
		return 0x68 // VK_NUMPAD8
	case 73:
		return 0x69 // VK_NUMPAD9
	case 74:
		return 0x6D // VK_SUBTRACT (KP_-)
	case 75:
		return 0x64 // VK_NUMPAD4
	case 76:
		return 0x65 // VK_NUMPAD5
	case 77:
		return 0x66 // VK_NUMPAD6
	case 78:
		return 0x6B // VK_ADD (KP_+)
	case 79:
		return 0x61 // VK_NUMPAD1
	case 80:
		return 0x62 // VK_NUMPAD2
	case 81:
		return 0x63 // VK_NUMPAD3
	case 82:
		return 0x60 // VK_NUMPAD0
	case 83:
		return 0x6E // VK_DECIMAL (KP_.)
	case 87:
		return 0x7A // F11
	case 88:
		return 0x7B // F12
	case 98:
		return 0x6F // VK_DIVIDE (KP_/)
	case 102:
		return 0x24 // Home
	case 103:
		return 0x26 // Up
	case 104:
		return 0x21 // PageUp
	case 105:
		return 0x25 // Left
	case 106:
		return 0x27 // Right
	case 107:
		return 0x23 // End
	case 108:
		return 0x28 // Down
	case 109:
		return 0x22 // PageDown
	case 110:
		return 0x2D // Insert
	case 111:
		return 0x2E // Delete
	case 125:
		return 0x5B // VK_LWIN
	case 126:
		return 0x5C // VK_RWIN
	case 127:
		return 0x5D // VK_APPS
	default:
		return 0
	}
}

func evdevToFallbackRune(code uint32, shift, caps bool) rune {
	upper := shift
	if caps {
		upper = !upper
	}
	switch code {
	case 2:
		if shift {
			return '!'
		}
		return '1'
	case 3:
		if shift {
			return '@'
		}
		return '2'
	case 4:
		if shift {
			return '#'
		}
		return '3'
	case 5:
		if shift {
			return '$'
		}
		return '4'
	case 6:
		if shift {
			return '%'
		}
		return '5'
	case 7:
		if shift {
			return '^'
		}
		return '6'
	case 8:
		if shift {
			return '&'
		}
		return '7'
	case 9:
		if shift {
			return '*'
		}
		return '8'
	case 10:
		if shift {
			return '('
		}
		return '9'
	case 11:
		if shift {
			return ')'
		}
		return '0'
	case 12:
		if shift {
			return '_'
		}
		return '-'
	case 13:
		if shift {
			return '+'
		}
		return '='
	case 26:
		if shift {
			return '{'
		}
		return '['
	case 27:
		if shift {
			return '}'
		}
		return ']'
	case 39:
		if shift {
			return ':'
		}
		return ';'
	case 40:
		if shift {
			return '"'
		}
		return '\''
	case 41:
		if shift {
			return '~'
		}
		return '`'
	case 43:
		if shift {
			return '|'
		}
		return '\\'
	case 51:
		if shift {
			return '<'
		}
		return ','
	case 52:
		if shift {
			return '>'
		}
		return '.'
	case 53:
		if shift {
			return '?'
		}
		return '/'
	case 57:
		return ' '
	}

	letters := map[uint32]rune{
		16: 'q', 17: 'w', 18: 'e', 19: 'r', 20: 't', 21: 'y', 22: 'u', 23: 'i', 24: 'o', 25: 'p',
		30: 'a', 31: 's', 32: 'd', 33: 'f', 34: 'g', 35: 'h', 36: 'j', 37: 'k', 38: 'l',
		44: 'z', 45: 'x', 46: 'c', 47: 'v', 48: 'b', 49: 'n', 50: 'm',
	}
	if ch, ok := letters[code]; ok {
		if upper {
			return ch - ('a' - 'A')
		}
		return ch
	}
	return 0
}

func xkbSymNameToRune(name string) rune {
	if len(name) == 1 {
		r := rune(name[0])
		if r >= 0x20 && r <= 0x7E {
			return r
		}
	}
	if strings.HasPrefix(name, "U") && len(name) >= 5 {
		if v, err := strconv.ParseUint(name[1:], 16, 32); err == nil {
			return rune(v)
		}
	}
	if r, ok := xkbNamedSyms[name]; ok {
		return r
	}
	return 0
}

var xkbNamedSyms = map[string]rune{
	"space":           ' ',
	"exclam":          '!',
	"quotedbl":        '"',
	"numbersign":      '#',
	"dollar":          '$',
	"percent":         '%',
	"ampersand":       '&',
	"apostrophe":      '\'',
	"parenleft":       '(',
	"parenright":      ')',
	"asterisk":        '*',
	"plus":            '+',
	"comma":           ',',
	"minus":           '-',
	"period":          '.',
	"slash":           '/',
	"colon":           ':',
	"semicolon":       ';',
	"less":            '<',
	"equal":           '=',
	"greater":         '>',
	"question":        '?',
	"at":              '@',
	"bracketleft":     '[',
	"backslash":       '\\',
	"bracketright":    ']',
	"asciicircum":     '^',
	"underscore":      '_',
	"grave":           '`',
	"braceleft":       '{',
	"bar":             '|',
	"braceright":      '}',
	"asciitilde":      '~',
	"numerosign":      '№',
	"Cyrillic_io":     'ё',
	"Cyrillic_IO":     'Ё',
	"Cyrillic_a":      'а',
	"Cyrillic_A":      'А',
	"Cyrillic_be":     'б',
	"Cyrillic_BE":     'Б',
	"Cyrillic_ve":     'в',
	"Cyrillic_VE":     'В',
	"Cyrillic_ghe":    'г',
	"Cyrillic_GHE":    'Г',
	"Cyrillic_de":     'д',
	"Cyrillic_DE":     'Д',
	"Cyrillic_ie":     'е',
	"Cyrillic_IE":     'Е',
	"Cyrillic_zhe":    'ж',
	"Cyrillic_ZHE":    'Ж',
	"Cyrillic_ze":     'з',
	"Cyrillic_ZE":     'З',
	"Cyrillic_i":      'и',
	"Cyrillic_I":      'И',
	"Cyrillic_shorti": 'й',
	"Cyrillic_SHORTI": 'Й',
	"Cyrillic_ka":     'к',
	"Cyrillic_KA":     'К',
	"Cyrillic_el":     'л',
	"Cyrillic_EL":     'Л',
	"Cyrillic_em":     'м',
	"Cyrillic_EM":     'М',
	"Cyrillic_en":     'н',
	"Cyrillic_EN":     'Н',
	"Cyrillic_o":      'о',
	"Cyrillic_O":      'О',
	"Cyrillic_pe":     'п',
	"Cyrillic_PE":     'П',
	"Cyrillic_er":     'р',
	"Cyrillic_ER":     'Р',
	"Cyrillic_es":     'с',
	"Cyrillic_ES":     'С',
	"Cyrillic_te":     'т',
	"Cyrillic_TE":     'Т',
	"Cyrillic_u":      'у',
	"Cyrillic_U":      'У',
	"Cyrillic_ef":     'ф',
	"Cyrillic_EF":     'Ф',
	"Cyrillic_ha":     'х',
	"Cyrillic_HA":     'Х',
	"Cyrillic_tse":    'ц',
	"Cyrillic_TSE":    'Ц',
	"Cyrillic_che":    'ч',
	"Cyrillic_CHE":    'Ч',
	"Cyrillic_sha":    'ш',
	"Cyrillic_SHA":    'Ш',
	"Cyrillic_shcha":  'щ',
	"Cyrillic_SHCHA":  'Щ',
	"Cyrillic_hardsign": 'ъ',
	"Cyrillic_HARDSIGN": 'Ъ',
	"Cyrillic_yeru":   'ы',
	"Cyrillic_YERU":   'Ы',
	"Cyrillic_softsign": 'ь',
	"Cyrillic_SOFTSIGN": 'Ь',
	"Cyrillic_e":      'э',
	"Cyrillic_E":      'Э',
	"Cyrillic_yu":     'ю',
	"Cyrillic_YU":     'Ю',
	"Cyrillic_ya":     'я',
	"Cyrillic_YA":     'Я',
}

func (w *waylandWindow) logicalToPhys(lx, ly float64) (int, int) {
	w.mu.Lock()
	scale := w.scale
	w.mu.Unlock()
	if scale <= 0 {
		scale = 1.0
	}
	return int(math.Round(lx * float64(scale))), int(math.Round(ly * float64(scale)))
}

func (w *waylandWindow) tryFramelessMoveResize(px, py int, seatID, serial uint32) bool {
	topID := w.currentToplevelID()
	if !w.cfg.Frameless || topID == 0 || seatID == 0 {
		return false
	}
	w.mu.Lock()
	scale := w.scale
	ww, wh := w.width, w.height
	maxed := w.maximized
	w.mu.Unlock()

	borderPx := int(6 * scale)
	titleH := w.cfg.TitleBarHeight
	if titleH <= 0 {
		titleH = 36
	}
	titlePx := int(float32(titleH) * scale)
	hit := ComputeFramelessHitTest(px, py, ww, wh, borderPx, titlePx, maxed, w.cfg.HitTest)

	if hit == HitCaption {
		var wr wlWriter
		wr.uint32(seatID)
		wr.uint32(serial)
		_ = w.disp.conn.send(topID, xdgToplevelMove, wr.bytes())
		return true
	}

	if edge := hitTestToXDGRegionEdge(hit); edge != 0 {
		var wr wlWriter
		wr.uint32(seatID)
		wr.uint32(serial)
		wr.uint32(edge)
		_ = w.disp.conn.send(topID, xdgToplevelResize, wr.bytes())
		return true
	}
	return false
}

func hitTestToXDGRegionEdge(hit HitTestResult) uint32 {
	// xdg_toplevel.resize_edge: none=0, top=1, bottom=2, left=4, top_left=5, bottom_left=6, right=8, top_right=9, bottom_right=10
	switch hit {
	case HitTop:
		return 1
	case HitBottom:
		return 2
	case HitLeft:
		return 4
	case HitTopLeft:
		return 5
	case HitBottomLeft:
		return 6
	case HitRight:
		return 8
	case HitTopRight:
		return 9
	case HitBottomRight:
		return 10
	default:
		return 0
	}
}

// Заглушка для fmt, если не используется напрямую
var _ = fmt.Sprintf
