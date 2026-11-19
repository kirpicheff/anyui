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

