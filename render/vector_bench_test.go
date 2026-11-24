package render

import (
	"testing"
)

func BenchmarkDrawLine_Diagonal(b *testing.B) {
	c := NewCanvas(800, 600)
	col := RGB(255, 0, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.DrawLine(50, 50, 250, 250, 2.0, col)
	}
}

func BenchmarkDrawLine_Horizontal(b *testing.B) {
	c := NewCanvas(800, 600)
	col := RGB(0, 255, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.DrawLine(50, 100, 250, 100, 2.0, col)
	}
}

func BenchmarkDrawLine_Vertical(b *testing.B) {
	c := NewCanvas(800, 600)
	col := RGB(0, 0, 255)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.DrawLine(100, 50, 100, 250, 2.0, col)
	}
}

func BenchmarkDrawLine_1000Microsegments(b *testing.B) {
	c := NewCanvas(800, 600)
	col := RGB(255, 128, 0)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 1000; j++ {
			x := float32(50 + (j%50)*10)
			y := float32(50 + (j/50)*10)
			c.DrawLine(x, y, x+6, y+4, 1.5, col)
		}
	}
}

func BenchmarkDrawPolyline_ManyPoints(b *testing.B) {
	c := NewCanvas(800, 600)
	col := RGB(255, 255, 0)
	pts := make([]PointF, 50)
	for i := range pts {
		pts[i] = PointF{X: float32(50 + i*10), Y: float32(100 + (i%5)*20)}
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c.DrawPolyline(pts, 2.0, col, false)
	}
}
