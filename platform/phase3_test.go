//go:build windows

package platform

import (
	"testing"
	"unicode/utf16"
)

func TestComputeFramelessHitTest(t *testing.T) {
	// Углы и края при обычном окне
	if got := ComputeFramelessHitTest(2, 2, 800, 600, 6, 36, false, nil); got != HitTopLeft {
		t.Fatalf("ожидался HitTopLeft, получено %v", got)
	}
	if got := ComputeFramelessHitTest(798, 2, 800, 600, 6, 36, false, nil); got != HitTopRight {
		t.Fatalf("ожидался HitTopRight, получено %v", got)
	}
	if got := ComputeFramelessHitTest(2, 598, 800, 600, 6, 36, false, nil); got != HitBottomLeft {
		t.Fatalf("ожидался HitBottomLeft, получено %v", got)
	}
	if got := ComputeFramelessHitTest(798, 598, 800, 600, 6, 36, false, nil); got != HitBottomRight {
		t.Fatalf("ожидался HitBottomRight, получено %v", got)
	}
	if got := ComputeFramelessHitTest(3, 100, 800, 600, 6, 36, false, nil); got != HitLeft {
		t.Fatalf("ожидался HitLeft, получено %v", got)
	}
	if got := ComputeFramelessHitTest(100, 20, 800, 600, 6, 36, false, nil); got != HitCaption {
		t.Fatalf("ожидался HitCaption, получено %v", got)
	}
	if got := ComputeFramelessHitTest(100, 100, 800, 600, 6, 36, false, nil); got != HitClient {
		t.Fatalf("ожидался HitClient, получено %v", got)
	}

	// В развёрнутом окне ресайз по краям отключён, работает шапка
	if got := ComputeFramelessHitTest(2, 2, 800, 600, 6, 36, true, nil); got != HitCaption {
		t.Fatalf("в развёрнутом окне ожидался HitCaption, получено %v", got)
	}

	// Кастомный предикат (например, кнопка закрытия в шапке остаётся HitClient)
	custom := func(x, y int) HitTestResult {
		if x >= 750 && y < 36 {
			return HitClient
		}
		if y < 36 {
			return HitCaption
		}
		return HitClient
	}
	if got := ComputeFramelessHitTest(770, 15, 800, 600, 6, 36, false, custom); got != HitClient {
		t.Fatalf("для кнопки в шапке ожидался HitClient, получено %v", got)
	}
}

func TestBuildWin32Filter(t *testing.T) {
	if got := BuildWin32Filter(nil); got != nil {
		t.Fatalf("для пустого списка фильтров ожидался nil, получено %v", got)
	}

	buf := BuildWin32Filter([]FileFilter{
		{Name: "Go files", Pattern: "*.go"},
		{Name: "All", Pattern: "*.*"},
	})
	if len(buf) < 2 || buf[len(buf)-1] != 0 || buf[len(buf)-2] != 0 {
		t.Fatalf("ожидался двойной терминатор нуля в конце фильтра")
	}
	decoded := string(utf16.Decode(buf))
	want := "Go files\x00*.go\x00All\x00*.*\x00\x00"
	if decoded != want {
		t.Fatalf("ожидалось %q, получено %q", want, decoded)
	}
}

func TestParseOpenFileNames(t *testing.T) {
	// Одиночный файл
	single := utf16.Encode([]rune(`C:\work\test.go` + "\x00\x00"))
	gotSingle := ParseOpenFileNames(single)
	if len(gotSingle) != 1 || gotSingle[0] != `C:\work\test.go` {
		t.Fatalf("ожидался одиночный файл, получено %v", gotSingle)
	}

	// Множественный выбор
	multi := utf16.Encode([]rune(`C:\work` + "\x00" + "a.go" + "\x00" + "b.go" + "\x00\x00"))
	gotMulti := ParseOpenFileNames(multi)
	if len(gotMulti) != 2 || gotMulti[0] != `C:\work\a.go` || gotMulti[1] != `C:\work\b.go` {
		t.Fatalf("ожидалось 2 файла из C:\\work, получено %v", gotMulti)
	}
}

func TestCreateHIconFromRGBA(t *testing.T) {
	rgba, w, h := defaultTrayIconRGBA()
	hIcon, err := createHIconFromRGBA(rgba, w, h)
	if err != nil {
		t.Fatalf("ошибка создания HICON: %v", err)
	}
	if hIcon == 0 {
		t.Fatal("получен нулевой HICON")
	}
	procDestroyIcon.Call(uintptr(hIcon))
}

func TestWin32CursorResourceID(t *testing.T) {
	cases := []struct {
		cursor CursorType
		want   uintptr
	}{
		{CursorArrow, idcArrow},
		{CursorHand, idcHand},
		{CursorIBeam, idcIBeam},
		{CursorResizeH, idcSizeWE},
		{CursorResizeV, idcSizeNS},
		{CursorResizeNWSE, idcSizeNWSE},
		{CursorResizeNESW, idcSizeNESW},
		{CursorNotAllowed, idcNo},
		{CursorCrosshair, idcCross},
		{CursorMove, idcSizeAll},
	}
	for _, tc := range cases {
		if got := win32CursorResourceID(tc.cursor); got != tc.want {
			t.Fatalf("для курсора %v ожидался id %d, получено %d", tc.cursor, tc.want, got)
		}
	}
}

