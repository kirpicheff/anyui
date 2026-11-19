//go:build windows

package platform

import "testing"

func TestWin32DIBBackBufferAndFree(t *testing.T) {
	w := &win32Window{width: 320, height: 240}
	defer func() {
		w.mu.Lock()
		w.freeGDILocked()
		w.mu.Unlock()
	}()

	buf1 := w.BackBuffer(320, 240)
	if len(buf1) != 320*240*4 {
		t.Fatalf("ожидался буфер %d байт, получено %d", 320*240*4, len(buf1))
	}
	if w.memDC == 0 || w.hBitmap == 0 {
		t.Fatal("ожидалось создание memDC и hBitmap через CreateDIBSection")
	}

	// Повторный вызов с тем же размером должен возвращать тот же самый DIBSection без пересоздания
	oldBmp := w.hBitmap
	buf2 := w.BackBuffer(320, 240)
	if len(buf2) != len(buf1) || &buf2[0] != &buf1[0] || w.hBitmap != oldBmp {
		t.Fatal("повторный BackBuffer того же размера должен переиспользовать существующий DIBSection")
	}

	// Изменение размера должно пересоздать DIBSection
	buf3 := w.BackBuffer(400, 300)
	if len(buf3) != 400*300*4 || w.hBitmap == 0 || w.hBitmap == oldBmp {
		t.Fatal("при смене размера DIBSection должен пересоздаваться")
	}

	// Проверяем, что PresentRect с тем же срезом (zero-copy) не падает и обновляет без пересоздания
	w.PresentRect(buf3, 400, 300, 10, 10, 50, 50)
}

