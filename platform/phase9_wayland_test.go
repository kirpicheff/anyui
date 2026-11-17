//go:build linux

package platform

import (
	"math"
	"testing"
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

