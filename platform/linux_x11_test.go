//go:build linux

package platform

import (
	"reflect"
	"testing"

	"github.com/jezek/xgb/xproto"
)

func TestX11KeysymToVK(t *testing.T) {
	cases := []struct {
		sym  xproto.Keysym
		want uint32
	}{
		{'a', 'A'},
		{'Z', 'Z'},
		{'5', '5'},
		{0xFF08, 0x08}, // BackSpace
		{0xFF09, 0x09}, // Tab
		{0xFF0D, 0x0D}, // Return
		{0xFF1B, 0x1B}, // Escape
		{0x0020, 0x20}, // Space
		{0xFF51, 0x25}, // Left
		{0xFF52, 0x26}, // Up
		{0xFF53, 0x27}, // Right
		{0xFF54, 0x28}, // Down
		{0xFFFF, 0x2E}, // Delete
		{0xFFBE, 0x70}, // F1
	}
	for _, tc := range cases {
		if got := x11KeysymToVK(tc.sym); got != tc.want {
			t.Errorf("x11KeysymToVK(0x%X) = 0x%X, want 0x%X", tc.sym, got, tc.want)
		}
	}
}

func TestX11KeysymToRune(t *testing.T) {
	if got := x11KeysymToRune('x'); got != 'x' {
		t.Fatalf("expected 'x', got %q", got)
	}
	if got := x11KeysymToRune(0x06C1); got != 'а' {
		t.Fatalf("expected 'а', got %q", got)
	}
	if got := x11KeysymToRune(0x06E1); got != 'А' {
		t.Fatalf("expected 'А', got %q", got)
	}
	if got := x11KeysymToRune(0x01000416); got != 'Ж' {
		t.Fatalf("expected 'Ж', got %q", got)
	}
	if got := x11KeysymToRune(0xFF0D); got != 0 {
		t.Fatalf("expected 0 for Return, got %q", got)
	}
}

func TestParseXDNDURIList(t *testing.T) {
	raw := "# comment\r\nfile:///home/user/doc.txt\r\n\r\nfile:///tmp/a%20b.png\n"
	got := parseXDNDURIList(raw)
	want := []string{"/home/user/doc.txt", "/tmp/a b.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseXDNDURIList = %v, want %v", got, want)
	}
}

func TestParseXftDPIScale(t *testing.T) {
	res := "*customization:\t-color\nXft.dpi:\t144\nXcursor.size:\t24\n"
	if got := parseXftDPIScale(res); got != 1.5 {
		t.Fatalf("parseXftDPIScale = %v, want 1.5", got)
	}
	if got := parseXftDPIScale("Xft.dpi: invalid\n"); got != 0 {
		t.Fatalf("expected 0 for invalid DPI, got %v", got)
	}
}

func TestBuildKdialogFilterAndInitialPath(t *testing.T) {
	gotFilter := buildKdialogFilter([]FileFilter{
		{Name: "Images", Pattern: "*.png;*.jpg"},
		{Pattern: "*.*"},
	})
	wantFilter := "Images (*.png *.jpg)\n*.*"
	if gotFilter != wantFilter {
		t.Fatalf("buildKdialogFilter = %q, want %q", gotFilter, wantFilter)
	}

	if got := buildInitialPath("/tmp", "out.txt"); got != "/tmp/out.txt" {
		t.Fatalf("buildInitialPath = %q, want /tmp/out.txt", got)
	}
	if got := buildInitialPath("/tmp", ""); got != "/tmp/" {
		t.Fatalf("buildInitialPath = %q, want /tmp/", got)
	}
}

func TestHitTestToNetWMMoveResize(t *testing.T) {
	if _, ok := hitTestToNetWMMoveResize(HitClient); ok {
		t.Fatal("HitClient should not trigger _NET_WM_MOVERESIZE")
	}
	if dir, ok := hitTestToNetWMMoveResize(HitCaption); !ok || dir != 8 {
		t.Fatalf("HitCaption = (%d, %v), want (8, true)", dir, ok)
	}
	if dir, ok := hitTestToNetWMMoveResize(HitBottomRight); !ok || dir != 4 {
		t.Fatalf("HitBottomRight = (%d, %v), want (4, true)", dir, ok)
	}
}

func TestX11PropertyChunkBounds(t *testing.T) {
	// Маленькая иконка (16x16) укладывается в один запрос.
	small := x11PropertyChunkBounds((2+16*16)*4, 262076)
	if len(small) != 1 || small[0] != [2]int{0, 1032} {
		t.Fatalf("unexpected small icon chunks: %v", small)
	}

	// Крупная иконка (256x256 = 262152 байт) превышает стандартный лимит X11 и должна разбиваться на чанки, кратные 4 байтам.
	total := (2 + 256*256) * 4
	chunks := x11PropertyChunkBounds(total, 262076)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks for 256x256 icon, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != [2]int{0, 262076} || chunks[1] != [2]int{262076, total} {
		t.Fatalf("unexpected chunk bounds: %v", chunks)
	}
	for i, c := range chunks {
		if (c[1]-c[0])%4 != 0 {
			t.Fatalf("chunk %d size %d is not 4-byte aligned", i, c[1]-c[0])
		}
	}
}

func TestX11KeyMapXKBGroupLookup(t *testing.T) {
	km := &x11KeyMap{
		minKeycode:  38,
		maxKeycode:  39,
		symsPerCode: 4,
		keysyms: []xproto.Keysym{
			// Keycode 38: латиница 'a'/'A', кириллица 'ф'/'Ф'
			'a', 'A', 0x06C6, 0x06E6,
			// Keycode 39: латиница '`'/'~', кириллица 'ё'/'Ё'
			'`', '~', 0x06A3, 0x06B3,
		},
	}

	// Группа 0 (латиница)
	sym, base := km.lookup(38, 0)
	if sym != 'a' || base != 'a' {
		t.Fatalf("group 0: got (%#x, %#x), want ('a', 'a')", sym, base)
	}

	// Группа 1 (кириллица) без Shift: sym='ф', но base='a' (для горячих клавиш Ctrl+...)
	sym, base = km.lookup(38, 1<<13)
	if sym != 0x06C6 || base != 'a' || x11KeysymToRune(sym) != 'ф' {
		t.Fatalf("group 1: got sym=%#x (%q), base=%#x, want 'ф' and 'a'", sym, x11KeysymToRune(sym), base)
	}

	// Группа 1 + Shift: sym='Ф'
	sym, base = km.lookup(38, (1<<13)|xproto.ModMaskShift)
	if sym != 0x06E6 || base != 'a' || x11KeysymToRune(sym) != 'Ф' {
		t.Fatalf("group 1 + Shift: got sym=%#x (%q), base=%#x, want 'Ф' and 'a'", sym, x11KeysymToRune(sym), base)
	}

	// Группа 1 + CapsLock для 'ё' -> 'Ё'
	sym, _ = km.lookup(39, (1<<13)|xproto.ModMaskLock)
	if sym != 0x06B3 || x11KeysymToRune(sym) != 'Ё' {
		t.Fatalf("group 1 + CapsLock (ё): got sym=%#x (%q), want 'Ё'", sym, x11KeysymToRune(sym))
	}

	// Пустая группа 2 откатывается на группу 0
	sym, base = km.lookup(38, 2<<13)
	if sym != 'a' || base != 'a' {
		t.Fatalf("fallback group 2: got (%#x, %#x), want ('a', 'a')", sym, base)
	}

	// Реальная Xorg core-карта при pc105 (symsPerCode=6): группа 0 занимает 4 слота, группа 1 — со смещения 4.
	xorgSyms := []xproto.Keysym{
		'a', 'A', 'a', 'A', 0x06C6, 0x06E6,
		's', 'S', 's', 'S', 0x06D9, 0x06F9,
	}
	stride := detectX11GroupStride(38, 39, 6, xorgSyms)
	if stride != 4 {
		t.Fatalf("detectX11GroupStride = %d, want 4", stride)
	}
	kmXorg := &x11KeyMap{
		minKeycode:  38,
		maxKeycode:  39,
		symsPerCode: 6,
		groupStride: stride,
		keysyms:     xorgSyms,
	}
	sym, base = kmXorg.lookup(38, 1<<13)
	if sym != 0x06C6 || base != 'a' || x11KeysymToRune(sym) != 'ф' {
		t.Fatalf("xorg group 1: got sym=%#x (%q), base=%#x, want 'ф' and 'a'", sym, x11KeysymToRune(sym), base)
	}
	sym, base = kmXorg.lookup(39, (1<<13)|xproto.ModMaskShift)
	if sym != 0x06F9 || base != 's' || x11KeysymToRune(sym) != 'Ы' {
		t.Fatalf("xorg group 1 + Shift: got sym=%#x (%q), base=%#x, want 'Ы' and 's'", sym, x11KeysymToRune(sym), base)
	}
}
