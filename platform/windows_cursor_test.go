//go:build windows

package platform

import "testing"

func makeTestCursorBGRA(w, h int, b, g, r, a byte) []byte {
	buf := make([]byte, w*h*4)
	for i := 0; i < len(buf); i += 4 {
		buf[i+0] = b
		buf[i+1] = g
		buf[i+2] = r
		buf[i+3] = a
	}
	return buf
}

func TestWin32CustomCursorLifecycleAndCache(t *testing.T) {
	w := &win32Window{scale: 1.0}
	defer w.clearCustomCursor(false)

	cur1 := &CustomCursor{
		BGRA:   makeTestCursorBGRA(32, 32, 0, 128, 255, 200),
		Width:  32,
		Height: 32,
		HotX:   4,
		HotY:   6,
	}

	// Создание нового HCURSOR
	w.SetCustomCursor(cur1)
	if w.customHCursor == 0 {
		t.Fatal("ожидался ненулевой HCURSOR после SetCustomCursor")
	}
	firstHandle := w.customHCursor

	// Повторная передача идентичного курсора (другой экземпляр структуры с теми же байтами)
	// не должна пересоздавать HCURSOR
	cur1Copy := &CustomCursor{
		BGRA:   append([]byte(nil), cur1.BGRA...),
		Width:  32,
		Height: 32,
		HotX:   4,
		HotY:   6,
	}
	w.SetCustomCursor(cur1Copy)
	if w.customHCursor != firstHandle {
		t.Fatalf("ожидалось переиспользование кэшированного HCURSOR (%v), получено %v", firstHandle, w.customHCursor)
	}

	// Изменение HotX должно пересоздать HCURSOR
	curHotChanged := &CustomCursor{
		BGRA:   append([]byte(nil), cur1.BGRA...),
		Width:  32,
		Height: 32,
		HotX:   10,
		HotY:   6,
	}
	w.SetCustomCursor(curHotChanged)
	if w.customHCursor == 0 {
		t.Fatal("ожидался ненулевой HCURSOR после смены HotX")
	}
	secondHandle := w.customHCursor

	// Изменение содержимого пикселей должно обновить хэш и пересоздать HCURSOR
	curPixChanged := &CustomCursor{
		BGRA:   makeTestCursorBGRA(32, 32, 255, 0, 0, 255),
		Width:  32,
		Height: 32,
		HotX:   10,
		HotY:   6,
	}
	w.SetCustomCursor(curPixChanged)
	if w.customHCursor == 0 {
		t.Fatal("ожидался ненулевой HCURSOR после смены пикселей")
	}
	if w.customCursorHash == customCursorDigest(cur1.BGRA) {
		t.Fatal("ожидалось изменение хэша кастомного курсора")
	}
	_ = secondHandle

	// Невалидные размеры/буфер должны сбрасывать кастомный курсор
	w.SetCustomCursor(&CustomCursor{BGRA: []byte{1, 2, 3}, Width: 32, Height: 32})
	if w.customHCursor != 0 {
		t.Fatalf("ожидался сброс HCURSOR при невалидном буфере, получено %v", w.customHCursor)
	}

	// Повторное создание и сброс через nil
	w.SetCustomCursor(cur1)
	if w.customHCursor == 0 {
		t.Fatal("ожидался ненулевой HCURSOR после повторной установки")
	}
	w.SetCustomCursor(nil)
	if w.customHCursor != 0 {
		t.Fatalf("ожидался сброс HCURSOR при SetCustomCursor(nil), получено %v", w.customHCursor)
	}
}

func TestWin32CursorNone(t *testing.T) {
	w := &win32Window{scale: 1.0}
	w.SetCursor(CursorNone)
	if w.cursor != CursorNone {
		t.Fatalf("ожидался w.cursor == CursorNone, получено %v", w.cursor)
	}
	w.SetCursor(CursorArrow)
	if w.cursor != CursorArrow {
		t.Fatalf("ожидался возврат w.cursor == CursorArrow, получено %v", w.cursor)
	}
}

func TestPremultiplyBGRA(t *testing.T) {
	src := []byte{
		// a = 0: цветные каналы должны обнулиться
		200, 150, 100, 0,
		// a = 128: (c*128 + 127) / 255
		255, 100, 0, 128,
		// a = 255: без изменений
		12, 34, 56, 255,
	}
	dst := make([]byte, len(src))
	premultiplyBGRA(dst, src)

	want := []byte{
		0, 0, 0, 0,
		128, 50, 0, 128,
		12, 34, 56, 255,
	}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("индекс %d: ожидалось %d, получено %d (dst=%v)", i, want[i], dst[i], dst)
		}
	}
}

