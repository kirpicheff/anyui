package render

import "math"

// PointF описывает точку с вещественными координатами.
type PointF struct {
	X float32
	Y float32
}

func distToSegment(px, py, x1, y1, x2, y2 float32) float32 {
	vx := x2 - x1
	vy := y2 - y1
	lenSq := vx*vx + vy*vy
	if lenSq <= 1e-6 {
		return float32(math.Hypot(float64(px-x1), float64(py-y1)))
	}
	t := ((px-x1)*vx + (py-y1)*vy) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	cx := x1 + vx*t
	cy := y1 + vy*t
	return float32(math.Hypot(float64(px-cx), float64(py-cy)))
}

// DrawLine рисует сглаженный отрезок от (x1, y1) до (x2, y2) с заданной толщиной.
func (c *Canvas) DrawLine(x1, y1, x2, y2, thickness float32, col Color) {
	c.DrawPolyline([]PointF{{X: x1, Y: y1}, {X: x2, Y: y2}}, thickness, col, false)
}

// DrawPolyline рисует сглаженную ломаную линию по точкам pts без наложения альфы на стыках.
func (c *Canvas) DrawPolyline(pts []PointF, thickness float32, col Color, closed bool) {
	if len(pts) < 2 || col.A == 0 {
		return
	}
	if thickness <= 0 {
		thickness = 1
	}
	halfW := thickness * 0.5

	minX, minY := pts[0].X, pts[0].Y
	maxX, maxY := pts[0].X, pts[0].Y
	for _, p := range pts[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	pad := halfW + 1.5
	bounds := Rect{
		X: int(math.Floor(float64(minX - pad))),
		Y: int(math.Floor(float64(minY - pad))),
		W: int(math.Ceil(float64(maxX-minX+pad*2))) + 1,
		H: int(math.Ceil(float64(maxY-minY+pad*2))) + 1,
	}
	area := c.CurrentClip().Intersect(bounds)
	if area.Empty() {
		return
	}

	segCount := len(pts) - 1
	if closed && len(pts) > 2 {
		segCount = len(pts)
	}

	for y := area.Y; y < area.Y+area.H; y++ {
		py := float32(y) + 0.5
		for x := area.X; x < area.X+area.W; x++ {
			px := float32(x) + 0.5
			minDist := float32(math.MaxFloat32)
			for i := 0; i < segCount; i++ {
				a := pts[i]
				b := pts[(i+1)%len(pts)]
				d := distToSegment(px, py, a.X, a.Y, b.X, b.Y)
				if d < minDist {
					minDist = d
				}
			}
			cov := clamp01(0.5 - (minDist - halfW))
			if cov > 0 {
				c.blendPixelUnchecked(x, y, col, uint8(cov*255+0.5))
			}
		}
	}
}

// DrawArc рисует сглаженную дугу окружности от startAngleRad до endAngleRad.
func (c *Canvas) DrawArc(cx, cy, radius, startAngleRad, endAngleRad, thickness float32, col Color) {
	if radius <= 0 || col.A == 0 {
		return
	}
	if thickness <= 0 {
		thickness = 1
	}
	span := float64(endAngleRad - startAngleRad)
	if math.Abs(span) < 1e-4 {
		return
	}

	steps := int(math.Ceil(math.Abs(span) * float64(radius) / 2.0))
	if steps < 8 {
		steps = 8
	}
	if steps > 256 {
		steps = 256
	}

	pts := make([]PointF, 0, steps+1)
	for i := 0; i <= steps; i++ {
		ang := float64(startAngleRad) + span*float64(i)/float64(steps)
		pts = append(pts, PointF{
			X: cx + float32(math.Cos(ang))*radius,
			Y: cy + float32(math.Sin(ang))*radius,
		})
	}
	c.DrawPolyline(pts, thickness, col, false)
}

