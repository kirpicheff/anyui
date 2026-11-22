package render

import (
	"image"
	"image/color"
	"testing"
)

func TestCanvasDirtyRectsAndBlit(t *testing.T) {
	src := NewCanvas(100, 100)
	src.Clear(RGB(10, 20, 30))
	src.ResetDirtyRects()

	src.FillRect(Rect{X: 10, Y: 15, W: 20, H: 10}, RGB(200, 50, 50))
	src.FillRect(Rect{X: 25, Y: 20, W: 15, H: 15}, RGB(50, 200, 50))
	// Пересекающиеся прямоугольники должны объединяться
	rects := src.DirtyRects()
	if len(rects) != 1 {
		t.Fatalf("ожидалось объединение пересекающихся грязных областей в 1, получено %d: %+v", len(rects), rects)
	}
	wantBounds := Rect{X: 10, Y: 15, W: 30, H: 20}
	if rects[0] != wantBounds {
		t.Fatalf("ожидались границы %+v, получено %+v", wantBounds, rects[0])
	}

	dst := NewCanvas(100, 100)
	dst.Clear(RGB(0, 0, 0))
	dst.ResetDirtyRects()
	dst.BlitFrom(src, Rect{X: 10, Y: 15, W: 30, H: 20})

	p := pixelAt(dst, 12, 17)
	if p.R != 200 || p.G != 50 || p.B != 50 {
		t.Fatalf("некорректный цвет после BlitFrom, получено %+v", p)
	}
	// Пиксель вне блитнутой области должен остаться чёрным
	if out := pixelAt(dst, 5, 5); out.R != 0 || out.G != 0 || out.B != 0 {
		t.Fatalf("ожидался чёрный пиксель вне области BlitFrom, получено %+v", out)
	}
	if len(dst.DirtyRects()) != 1 || dst.DirtyBounds() != wantBounds {
		t.Fatalf("ожидался dirty rect %+v после BlitFrom, получено %+v", wantBounds, dst.DirtyRects())
	}
}

func TestFontGlyphLRUCache(t *testing.T) {
	fm, err := LoadDefaultSystemFont()
	if err != nil {
		t.Skipf("системный шрифт недоступен в среде тестирования: %v", err)
	}
	fm.ClearGlyphCache()
	fm.SetGlyphCacheCapacity(3)

	c := NewCanvas(200, 80)
	c.Clear(RGB(0, 0, 0))

	// Рисуем 3 разных символа — все 3 должны поместиться
	fm.DrawText(c, 10, 10, "A", 16, RGB(255, 255, 255))
	fm.DrawText(c, 30, 10, "B", 16, RGB(255, 255, 255))
	fm.DrawText(c, 50, 10, "C", 16, RGB(255, 255, 255))

	entries, bytes := fm.GlyphCacheStats()
	if entries != 3 || bytes <= 0 {
		t.Fatalf("ожидалось 3 записи в LRU-кеше и >0 байт, получено entries=%d bytes=%d", entries, bytes)
	}

	// Обращаемся к 'A', чтобы он стал самым свежим, затем добавляем 'D' (должен вытеснить 'B')
	fm.DrawText(c, 70, 10, "A", 16, RGB(255, 255, 255))
	fm.DrawText(c, 90, 10, "D", 16, RGB(255, 255, 255))

	entries, _ = fm.GlyphCacheStats()
	if entries != 3 {
		t.Fatalf("ожидалось ограничение LRU в 3 записи, получено %d", entries)
	}

	keyA := coloredGlyphKey{style: FontRegular, r: 'A', size: 16, color: RGB(255, 255, 255)}
	keyB := coloredGlyphKey{style: FontRegular, r: 'B', size: 16, color: RGB(255, 255, 255)}
	if _, ok := fm.coloredGlyphs.Get(keyA); !ok {
		t.Fatal("символ 'A' не должен был быть вытеснен после повторного обращения")
	}
	if _, ok := fm.coloredGlyphs.Get(keyB); ok {
		t.Fatal("символ 'B' должен был быть вытеснен из LRU-кеша")
	}

	// Проверяем, что разные цвета дают разные ключи в кеше
	fm.DrawText(c, 110, 10, "A", 16, RGB(255, 0, 0))
	keyARed := coloredGlyphKey{style: FontRegular, r: 'A', size: 16, color: RGB(255, 0, 0)}
	if _, ok := fm.coloredGlyphs.Get(keyARed); !ok {
		t.Fatal("ожидалась отдельная запись в кеше для красного символа 'A'")
	}

	// Проверяем, что текст действительно нарисовал яркие пиксели на холсте
	lit := 0
	for y := 10; y < 35; y++ {
		for x := 10; x < 130; x++ {
			if pixelAt(c, x, y).R > 100 {
				lit++
			}
		}
	}
	if lit == 0 {
		t.Fatal("текст через цветной LRU-кеш глифов не оставил пикселей на холсте")
	}
}

func TestScaledImageLRUCache(t *testing.T) {
	ClearScaledImageCache()
	SetScaledImageCacheLimits(2, 4<<20)
	defer SetScaledImageCacheLimits(defaultScaledImageCacheCap, defaultScaledImageCacheBytes)

	makeSolidImg := func(col color.NRGBA) *image.NRGBA {
		im := image.NewNRGBA(image.Rect(0, 0, 4, 4))
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				im.SetNRGBA(x, y, col)
			}
		}
		return im
	}

	c := NewCanvas(80, 80)
	img1 := makeSolidImg(color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	img2 := makeSolidImg(color.NRGBA{R: 0, G: 255, B: 0, A: 255})
	img3 := makeSolidImg(color.NRGBA{R: 0, G: 0, B: 255, A: 255})

	c.Clear(RGB(0, 0, 0))
	c.DrawImage(Rect{X: 0, Y: 0, W: 20, H: 20}, img1, ScaleFit)
	// Повторная отрисовка того же изображения не должна создавать новую запись
	c.DrawImage(Rect{X: 30, Y: 0, W: 20, H: 20}, img1, ScaleFit)

	entries, bytes := ScaledImageCacheStats()
	if entries != 1 || bytes != 20*20*4 {
		t.Fatalf("ожидалась 1 запись (1600 байт), получено entries=%d bytes=%d", entries, bytes)
	}
	if p := pixelAt(c, 35, 10); p.R < 240 || p.G > 10 {
		t.Fatalf("некорректный цвет при чтении из кеша изображений: %+v", p)
	}

	c.DrawImage(Rect{X: 0, Y: 30, W: 20, H: 20}, img2, ScaleFit)
	c.DrawImage(Rect{X: 30, Y: 30, W: 20, H: 20}, img3, ScaleFit)
	entries, _ = ScaledImageCacheStats()
	if entries != 2 {
		t.Fatalf("ожидалось вытеснение по LRU до 2 записей, получено %d", entries)
	}
}

func TestFontCenterTextYOpticalBalance(t *testing.T) {
	fm, err := LoadDefaultSystemFont()
	if err != nil {
		t.Skipf("системный шрифт недоступен: %v", err)
	}

	const boxH = 48
	for _, style := range []FontStyle{FontRegular, FontMono} {
		c := NewCanvas(200, boxH)
		c.Clear(RGB(0, 0, 0))
		y := fm.CenterTextY(0, boxH, 18, style)
		fm.DrawTextStyled(c, 10, y, "000 000 000", 18, RGB(255, 255, 255), style)

		minY, maxY := boxH, -1
		for py := 0; py < boxH; py++ {
			for px := 0; px < 200; px++ {
				if p := pixelAt(c, px, py); p.R > 40 {
					if py < minY {
						minY = py
					}
					if py > maxY {
						maxY = py
					}
				}
			}
		}
		if maxY < minY {
			t.Fatalf("style=%v: текст не отрисовался", style)
		}
		topMargin := minY
		bottomMargin := (boxH - 1) - maxY
		diff := topMargin - bottomMargin
		if diff < 0 {
			diff = -diff
		}
		if diff > 2 {
			t.Fatalf("style=%v: нарушен вертикальный оптический баланс: top=%d bottom=%d (minY=%d maxY=%d)", style, topMargin, bottomMargin, minY, maxY)
		}
	}
}

