package render

import "testing"

func TestDrawBGRAFast1to1AndScaled(t *testing.T) {
	c := NewCanvas(10, 10)
	src := make([]byte, 4*4*4)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			off := (y*4 + x) * 4
			src[off+0] = uint8(x * 40) // B
			src[off+1] = uint8(y * 50) // G
			src[off+2] = 200           // R
			src[off+3] = 255           // A
		}
	}

	// Проверка 1:1 копирования
	c.DrawBGRAFast(Rect{X: 2, Y: 3, W: 4, H: 4}, src, 4, 4, ScaleFit)
	off := (4*c.Width + 5) * 4 // точка (5, 4) -> src (3, 1)
	if c.Pixels[off+0] != 120 || c.Pixels[off+1] != 50 || c.Pixels[off+2] != 200 || c.Pixels[off+3] != 255 {
		t.Fatalf("некорректный пиксель 1:1: %v", c.Pixels[off:off+4])
	}

	// Проверка масштабирования 4x4 -> 8x8 без аллокаций
	c.Clear(RGBA(0, 0, 0, 255))
	c.DrawBGRAFast(Rect{X: 1, Y: 1, W: 8, H: 8}, src, 4, 4, ScaleFit)
	offScaled := (7*c.Width + 7) * 4 // (7,7) -> (6,6) в 8x8 -> src (3,3)
	if c.Pixels[offScaled+0] != 120 || c.Pixels[offScaled+1] != 150 || c.Pixels[offScaled+2] != 200 || c.Pixels[offScaled+3] != 255 {
		t.Fatalf("некорректный масштабированный пиксель: %v", c.Pixels[offScaled:offScaled+4])
	}
}

func BenchmarkDrawBGRAFast1080p(b *testing.B) {
	c := NewCanvas(1280, 720)
	src := make([]byte, 1920*1080*4)
	dst := Rect{X: 0, Y: 0, W: 1280, H: 720}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.DrawBGRAFast(dst, src, 1920, 1080, ScaleFit)
	}
}
