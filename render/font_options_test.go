package render

import "testing"

func TestMeasureAndDrawTextWithOptions(t *testing.T) {
	fm, err := LoadDefaultSystemFont()
	if err != nil {
		t.Skip("системный шрифт недоступен в тестовой среде")
	}

	baseW, baseH := fm.MeasureTextWithOptions("ABCD", 16, TextRenderOptions{})
	spacedW, spacedH := fm.MeasureTextWithOptions("ABCD", 16, TextRenderOptions{LetterSpacingPx: 4})
	if spacedW <= baseW {
		t.Fatalf("ожидалась большая ширина с LetterSpacingPx: base=%d spaced=%d", baseW, spacedW)
	}
	if spacedH != baseH {
		t.Fatalf("высота строки не должна меняться: base=%d spaced=%d", baseH, spacedH)
	}

	// Одиночный символ не должен получать trailing letter-spacing
	singleBase, _ := fm.MeasureTextWithOptions("A", 16, TextRenderOptions{})
	singleSpaced, _ := fm.MeasureTextWithOptions("A", 16, TextRenderOptions{LetterSpacingPx: 10})
	if singleSpaced != singleBase {
		t.Fatalf("для одного символа ширина не должна меняться: base=%d spaced=%d", singleBase, singleSpaced)
	}

	// Проверяем мягкое свечение TextShadow
	c := NewCanvas(160, 60)
	c.Clear(RGB(0, 0, 0))
	fm.DrawTextWithOptions(c, 20, 16, "Glow", 18, RGB(255, 255, 255), TextRenderOptions{
		LetterSpacingPx: 2,
		Shadow: &TextShadowStyle{
			OffsetX: 0,
			OffsetY: 0,
			Blur:    6,
			Color:   RGBA(0, 180, 255, 200),
		},
	})

	// Вокруг текста (например, чуть левее/выше начала x=20,y=16) должны появиться синие пиксели свечения
	glowFound := false
	for y := 10; y < 16; y++ {
		for x := 14; x < 40; x++ {
			off := (y*c.Width + x) * 4
			if c.Pixels[off+0] > 0 || c.Pixels[off+1] > 0 {
				glowFound = true
				break
			}
		}
		if glowFound {
			break
		}
	}
	if !glowFound {
		t.Fatal("ожидались пиксели мягкого свечения вокруг текста при Blur > 0")
	}
}

func TestDrawTextOnTransparentOverlayCompositeNoHalo(t *testing.T) {
	fm, err := LoadDefaultSystemFont()
	if err != nil {
		t.Skip("системный шрифт недоступен в тестовой среде")
	}

	// Рисуем белый текст на полностью прозрачный оверлей (straight alpha)
	overlay := NewCanvas(120, 48)
	overlay.ClearTransparent()
	fm.DrawTextWithOptions(overlay, 10, 10, "HELLO", 20, RGB(255, 255, 255), TextRenderOptions{
		Shadow: &TextShadowStyle{
			OffsetX: 1,
			OffsetY: 1,
			Blur:    2,
			Color:   RGBA(0, 0, 0, 120),
		},
	})

	// На краях букв (без тени) должны быть полупрозрачные пиксели со straight-цветом без чёрного ореола;
	// проверим чистый текст без тени на отдельном оверлее.
	textOnly := NewCanvas(120, 48)
	textOnly.ClearTransparent()
	fm.DrawText(textOnly, 10, 10, "HELLO", 20, RGB(255, 255, 255))

	antiAliasedFound := false
	for i := 0; i < len(textOnly.Pixels); i += 4 {
		a := textOnly.Pixels[i+3]
		if a > 0 && a < 255 {
			antiAliasedFound = true
			// В straight alpha для белого текста все каналы должны быть близки к 255 (без чёрного ореола)
			if textOnly.Pixels[i+0] < 240 || textOnly.Pixels[i+1] < 240 || textOnly.Pixels[i+2] < 240 {
				t.Fatalf("ожидался straight белый цвет на сглаженном пикселе (a=%d), получено BGRA=(%d,%d,%d,%d)",
					a, textOnly.Pixels[i+0], textOnly.Pixels[i+1], textOnly.Pixels[i+2], a)
			}
		}
	}
	if !antiAliasedFound {
		t.Fatal("ожидались полупрозрачные пиксели сглаживания текста на прозрачном холсте")
	}

	// Композитинг поверх светлого фона не должен оставлять тёмных пикселей по краям белых букв
	bg := NewCanvas(120, 48)
	bg.Clear(RGB(220, 220, 220))
	bg.CompositeFrom(textOnly, Rect{X: 0, Y: 0, W: 120, H: 48})
	for i := 0; i < len(bg.Pixels); i += 4 {
		if bg.Pixels[i+0] < 220 || bg.Pixels[i+1] < 220 || bg.Pixels[i+2] < 220 {
			t.Fatalf("обнаружен тёмный ореол при композитинге белого текста на светлый фон: BGRA=(%d,%d,%d,%d)",
				bg.Pixels[i+0], bg.Pixels[i+1], bg.Pixels[i+2], bg.Pixels[i+3])
		}
	}

	// Композитинг текста с тенью тоже не должен падать или оставлять полупрозрачную альфу в окне
	bg.Clear(RGB(40, 48, 64))
	bg.CompositeFrom(overlay, Rect{X: 0, Y: 0, W: 120, H: 48})
	for i := 0; i < len(bg.Pixels); i += 4 {
		if bg.Pixels[i+3] != 255 {
			t.Fatalf("ожидалась альфа 255 после CompositeFrom, получено %d", bg.Pixels[i+3])
		}
	}
}

