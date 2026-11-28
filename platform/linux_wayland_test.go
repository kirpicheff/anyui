//go:build linux

package platform

import (
	"encoding/binary"
	"math"
	"testing"

	"golang.org/x/sys/unix"
)

func TestDetectLinuxBackend(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want linuxBackend
	}{
		{
			name: "wayland_display",
			env:  map[string]string{"WAYLAND_DISPLAY": "wayland-0", "DISPLAY": ":0"},
			want: linuxBackendWayland,
		},
		{
			name: "x11_only",
			env:  map[string]string{"DISPLAY": ":0"},
			want: linuxBackendX11,
		},
		{
			name: "override_x11",
			env:  map[string]string{"WAYLAND_DISPLAY": "wayland-0", "ANYUI_BACKEND": "x11"},
			want: linuxBackendX11,
		},
		{
			name: "override_wayland",
			env:  map[string]string{"ANYUI_BACKEND": "wayland"},
			want: linuxBackendWayland,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := detectLinuxBackend(func(k string) string { return tc.env[k] })
			if got != tc.want {
				t.Fatalf("detectLinuxBackend() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWaylandWireWriterReader(t *testing.T) {
	var w wlWriter
	w.uint32(42)
	w.int32(-17)
	w.fixed(12.5)
	w.string("hello")
	w.string("")
	w.array([]byte{1, 2, 3, 4, 5})

	r := wlReader{data: w.bytes()}
	u, ok := r.uint32()
	if !ok || u != 42 {
		t.Fatalf("uint32 = %d (%v), want 42", u, ok)
	}
	i, ok := r.int32()
	if !ok || i != -17 {
		t.Fatalf("int32 = %d (%v), want -17", i, ok)
	}
	fx, ok := r.fixed()
	if !ok || math.Abs(fx-12.5) > 0.01 {
		t.Fatalf("fixed = %v (%v), want 12.5", fx, ok)
	}
	s1, ok := r.string()
	if !ok || s1 != "hello" {
		t.Fatalf("string1 = %q (%v), want hello", s1, ok)
	}
	s2, ok := r.string()
	if !ok || s2 != "" {
		t.Fatalf("string2 = %q (%v), want empty", s2, ok)
	}
	arr, ok := r.array()
	if !ok || len(arr) != 5 || arr[4] != 5 {
		t.Fatalf("array = %v (%v), want [1 2 3 4 5]", arr, ok)
	}
}

func TestParseXKBKeymapAndTranslate(t *testing.T) {
	sample := `
xkb_keymap {
xkb_keycodes "evdev" {
    <AD01> = 24;
};
xkb_symbols "pc+us+ru:2" {
    key <AD01> { [ q, Q ], [ Cyrillic_shorti, Cyrillic_SHORTI ] };
};
};
`
	km := parseXKBKeymap(sample)
	if km == nil {
		t.Fatal("parseXKBKeymap returned nil")
	}

	// evdev keycode 24 - 8 = 16 ('Q')
	vk, ch := translateEvdevKey(16, Modifiers{}, false, 0, km)
	if vk != 'Q' || ch != 'q' {
		t.Fatalf("group 0 lower: vk=%X ch=%q, want 'Q' 'q'", vk, ch)
	}

	_, chShift := translateEvdevKey(16, Modifiers{Shift: true}, false, 0, km)
	if chShift != 'Q' {
		t.Fatalf("group 0 shift: ch=%q, want 'Q'", chShift)
	}

	_, chRu := translateEvdevKey(16, Modifiers{}, false, 1, km)
	if chRu != 'й' {
		t.Fatalf("group 1 lower: ch=%q, want 'й'", chRu)
	}

	_, chRuCaps := translateEvdevKey(16, Modifiers{}, true, 1, km)
	if chRuCaps != 'Й' {
		t.Fatalf("group 1 caps: ch=%q, want 'Й'", chRuCaps)
	}
}

func TestWaylandCursorShapeAndHitTest(t *testing.T) {
	if got := waylandCursorShape(CursorHand); got != 4 {
		t.Fatalf("CursorHand = %d, want 4", got)
	}
	if got := waylandCursorShape(CursorIBeam); got != 9 {
		t.Fatalf("CursorIBeam = %d, want 9", got)
	}
	if got := hitTestToXDGRegionEdge(HitBottomRight); got != 10 {
		t.Fatalf("HitBottomRight = %d, want 10", got)
	}
}

func TestPickTextMIME(t *testing.T) {
	got := pickTextMIME([]string{"text/plain", "text/plain;charset=utf-8"})
	if got != "text/plain;charset=utf-8" {
		t.Fatalf("pickTextMIME = %q, want text/plain;charset=utf-8", got)
	}
	if empty := pickTextMIME([]string{"image/png"}); empty != "" {
		t.Fatalf("expected empty string, got %q", empty)
	}
}

func TestRGBAToSNIPixmapAndDBusMenu(t *testing.T) {
	rgba := []byte{0x11, 0x22, 0x33, 0xFF} // R=11, G=22, B=33, A=FF
	pix := rgbaToSNIPixmap(rgba, 1, 1)
	if pix.Width != 1 || pix.Height != 1 || len(pix.Data) != 4 {
		t.Fatalf("unexpected pixmap dimensions: %+v", pix)
	}
	// Ожидается ARGB32 big-endian: A, R, G, B
	if pix.Data[0] != 0xFF || pix.Data[1] != 0x11 || pix.Data[2] != 0x22 || pix.Data[3] != 0x33 {
		t.Fatalf("unexpected ARGB bytes: % X", pix.Data)
	}

	props := buildDBusMenuProps(TrayMenuItem{Label: "Открыть", Checked: true})
	if props["label"].Value() != "Открыть" {
		t.Fatalf("unexpected label: %v", props["label"].Value())
	}
	if props["toggle-state"].Value() != int32(1) {
		t.Fatalf("unexpected toggle-state: %v", props["toggle-state"].Value())
	}
}

// compile-time проверка, что waylandWindow реализует WindowActivator.
var _ WindowActivator = (*waylandWindow)(nil)

type testWLMsg struct {
	objID   uint32
	opcode  uint16
	payload []byte
}

func newTestWaylandPair(t *testing.T) (*wlConn, int) {
	t.Helper()
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatalf("Socketpair: %v", err)
	}
	var pipeFDs [2]int
	if err := unix.Pipe2(pipeFDs[:], unix.O_NONBLOCK|unix.O_CLOEXEC); err != nil {
		_ = unix.Close(fds[0])
		_ = unix.Close(fds[1])
		t.Fatalf("Pipe2: %v", err)
	}
	conn := &wlConn{
		fd:       fds[0],
		nextID:   100,
		handlers: make(map[uint32]wlEventHandler),
		globals:  make(map[string]wlGlobal),
		readBuf:  make([]byte, 0, 4096),
		wakeR:    pipeFDs[0],
		wakeW:    pipeFDs[1],
	}
	t.Cleanup(func() {
		conn.Close()
		_ = unix.Close(fds[1])
	})
	return conn, fds[1]
}

func readAllServerWLMsgs(t *testing.T, serverFD int) []testWLMsg {
	t.Helper()
	var raw []byte
	buf := make([]byte, 4096)
	for {
		n, _, _, _, err := unix.Recvmsg(serverFD, buf, nil, unix.MSG_DONTWAIT)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				break
			}
			t.Fatalf("Recvmsg: %v", err)
		}
		if n == 0 {
			break
		}
		raw = append(raw, buf[:n]...)
	}
	var out []testWLMsg
	for len(raw) >= 8 {
		objID := binary.LittleEndian.Uint32(raw[0:4])
		sizeOp := binary.LittleEndian.Uint32(raw[4:8])
		msgSize := int(sizeOp >> 16)
		opcode := uint16(sizeOp & 0xFFFF)
		if msgSize < 8 || len(raw) < msgSize {
			t.Fatalf("обрезанное сообщение Wayland: size=%d len=%d", msgSize, len(raw))
		}
		payload := make([]byte, msgSize-8)
		copy(payload, raw[8:msgSize])
		out = append(out, testWLMsg{objID: objID, opcode: opcode, payload: payload})
		raw = raw[msgSize:]
	}
	return out
}

func sendServerWLEvent(t *testing.T, serverFD int, objID uint32, opcode uint16, payload []byte) {
	t.Helper()
	totalLen := 8 + len(payload)
	pkt := make([]byte, totalLen)
	binary.LittleEndian.PutUint32(pkt[0:4], objID)
	binary.LittleEndian.PutUint32(pkt[4:8], (uint32(totalLen)<<16)|uint32(opcode))
	copy(pkt[8:], payload)
	if _, err := unix.Write(serverFD, pkt); err != nil {
		t.Fatalf("Write server event: %v", err)
	}
}

func TestWaylandXDGActivation(t *testing.T) {
	conn, serverFD := newTestWaylandPair(t)
	disp := &wlDisplay{
		conn:         conn,
		xdgWmBaseID:  10,
		activationID: 20,
		windows:      make(map[uint32]*waylandWindow),
	}
	w := &waylandWindow{
		disp:         disp,
		surfaceID:    30,
		xdgSurfaceID: 31,
		toplevelID:   32,
	}
	w.configured.Store(true)

	// Последовательный вызов Show + Restore + BringToFront (сценарий клика по трею)
	// должен отправить ровно один запрос токена активации.
	w.Show()
	w.Restore()
	w.BringToFront()

	msgs := readAllServerWLMsgs(t, serverFD)
	if len(msgs) != 3 {
		t.Fatalf("получено %d сообщений, ожидалось 3 (get_activation_token, set_app_id, commit): %+v", len(msgs), msgs)
	}
	if msgs[0].objID != disp.activationID || msgs[0].opcode != xdgActivationGetToken {
		t.Fatalf("msg[0] = (%d, %d), ожидалось (%d, %d)", msgs[0].objID, msgs[0].opcode, disp.activationID, xdgActivationGetToken)
	}
	r := wlReader{data: msgs[0].payload}
	tokenID, ok := r.uint32()
	if !ok || tokenID == 0 {
		t.Fatalf("некорректный tokenID: %d (%v)", tokenID, ok)
	}
	if msgs[1].objID != tokenID || msgs[1].opcode != xdgActivationTokenSetAppID {
		t.Fatalf("msg[1] = (%d, %d), ожидалось set_app_id для токена %d", msgs[1].objID, msgs[1].opcode, tokenID)
	}
	if msgs[2].objID != tokenID || msgs[2].opcode != xdgActivationTokenCommit {
		t.Fatalf("msg[2] = (%d, %d), ожидалось commit для токена %d", msgs[2].objID, msgs[2].opcode, tokenID)
	}

	// Композитор возвращает токен через xdg_activation_token_v1.done
	var dw wlWriter
	dw.string("gnome-token-42")
	sendServerWLEvent(t, serverFD, tokenID, xdgActivationTokenEventDone, dw.bytes())

	if err := conn.readAndDispatch(false); err != nil {
		t.Fatalf("readAndDispatch: %v", err)
	}
	if w.activationPending.Load() {
		t.Fatal("activationPending должен сброситься после done")
	}

	reply := readAllServerWLMsgs(t, serverFD)
	if len(reply) != 2 {
		t.Fatalf("получено %d ответных сообщений, ожидалось 2 (destroy, activate): %+v", len(reply), reply)
	}
	if reply[0].objID != tokenID || reply[0].opcode != xdgActivationTokenDestroy {
		t.Fatalf("reply[0] = (%d, %d), ожидалось destroy токена %d", reply[0].objID, reply[0].opcode, tokenID)
	}
	if reply[1].objID != disp.activationID || reply[1].opcode != xdgActivationActivate {
		t.Fatalf("reply[1] = (%d, %d), ожидалось activate на %d", reply[1].objID, reply[1].opcode, disp.activationID)
	}
	ar := wlReader{data: reply[1].payload}
	gotToken, ok1 := ar.string()
	gotSurf, ok2 := ar.uint32()
	if !ok1 || !ok2 || gotToken != "gnome-token-42" || gotSurf != w.surfaceID {
		t.Fatalf("activate(%q, %d), ожидалось (%q, %d)", gotToken, gotSurf, "gnome-token-42", w.surfaceID)
	}
}

func TestWaylandHideShowRemap(t *testing.T) {
	conn, serverFD := newTestWaylandPair(t)
	disp := &wlDisplay{
		conn:         conn,
		xdgWmBaseID:  10,
		activationID: 20,
		windows:      make(map[uint32]*waylandWindow),
	}
	w := &waylandWindow{
		disp:         disp,
		surfaceID:    30,
		xdgSurfaceID: 31,
		toplevelID:   32,
		csdTitle:     "Test Window",
	}
	w.configured.Store(true)
	w.framePending.Store(true)

	w.Hide()
	if !w.hidden.Load() || w.configured.Load() || w.framePending.Load() {
		t.Fatalf("после Hide(): hidden=%v configured=%v framePending=%v", w.hidden.Load(), w.configured.Load(), w.framePending.Load())
	}
	if w.currentToplevelID() != 0 || w.currentXDGSurfaceID() != 0 {
		t.Fatalf("role-объекты должны быть уничтожены после Hide(): xdg=%d top=%d", w.currentXDGSurfaceID(), w.currentToplevelID())
	}
	hideMsgs := readAllServerWLMsgs(t, serverFD)
	if len(hideMsgs) < 4 {
		t.Fatalf("ожидались attach(0), commit, toplevel.destroy, xdg_surface.destroy, получено: %+v", hideMsgs)
	}

	w.Show()
	if w.hidden.Load() {
		t.Fatal("после Show() флаг hidden должен быть false")
	}
	if !w.pendingActivate.Load() {
		t.Fatal("после Show() после Hide() должен быть выставлен pendingActivate")
	}
	if w.currentToplevelID() == 0 || w.currentXDGSurfaceID() == 0 {
		t.Fatal("после Show() role-объекты должны быть пересозданы")
	}
	showMsgs := readAllServerWLMsgs(t, serverFD)
	var hasGetXdg, hasGetTop, hasCommit bool
	for _, m := range showMsgs {
		if m.objID == disp.xdgWmBaseID && m.opcode == xdgWmBaseGetXdgSurface {
			hasGetXdg = true
		}
		if m.objID == w.currentXDGSurfaceID() && m.opcode == xdgSurfaceGetToplevel {
			hasGetTop = true
		}
		if m.objID == w.surfaceID && m.opcode == wlSurfaceCommit {
			hasCommit = true
		}
	}
	if !hasGetXdg || !hasGetTop || !hasCommit {
		t.Fatalf("при Show() не отправлена цепочка remap: get_xdg=%v get_top=%v commit=%v", hasGetXdg, hasGetTop, hasCommit)
	}
}

func TestWaylandKeyboardScanCodeX11Offset(t *testing.T) {
	disp := &wlDisplay{
		windows: make(map[uint32]*waylandWindow),
	}
	var got []Event
	w := &waylandWindow{
		disp:      disp,
		surfaceID: 42,
		handler: func(ev Event) {
			got = append(got, ev)
		},
	}
	disp.windows[w.surfaceID] = w

	seat := &wlSeatState{
		disp:              disp,
		keyboardSurfaceID: w.surfaceID,
		repeatRate:        0, // отключаем фоновый автоповтор в тесте
		repeatDelay:       0,
	}

	// Клавиша 'P': evdev keycode 25 -> X11/XKB keycode 33 (25 + 8).
	var down wlWriter
	down.uint32(1)  // serial
	down.uint32(10) // time
	down.uint32(25) // evdev KEY_P
	down.uint32(1)  // pressed
	seat.handleKeyboardEvent(wlEvent{Opcode: wlKeyboardEventKey, Payload: down.bytes()})

	var up wlWriter
	up.uint32(2)  // serial
	up.uint32(20) // time
	up.uint32(25) // evdev KEY_P
	up.uint32(0)  // released
	seat.handleKeyboardEvent(wlEvent{Opcode: wlKeyboardEventKey, Payload: up.bytes()})

	var keyDown, keyUp *Event
	for i := range got {
		switch got[i].Type {
		case EventKeyDown:
			keyDown = &got[i]
		case EventKeyUp:
			keyUp = &got[i]
		}
	}
	if keyDown == nil || keyUp == nil {
		t.Fatalf("ожидались EventKeyDown и EventKeyUp, получено: %+v", got)
	}
	if keyDown.KeyCode != 'P' || keyDown.ScanCode != 33 {
		t.Fatalf("KeyDown: KeyCode=%X ScanCode=%d, ожидалось 'P' и 33", keyDown.KeyCode, keyDown.ScanCode)
	}
	if keyUp.KeyCode != 'P' || keyUp.ScanCode != 33 {
		t.Fatalf("KeyUp: KeyCode=%X ScanCode=%d, ожидалось 'P' и 33", keyUp.KeyCode, keyUp.ScanCode)
	}
}


