package render

import "math"

// PointF описывает точку с вещественными координатами.
type PointF struct {
	X float32
	Y float32
}

type polySegment struct {
	x1, y1, vx, vy float32
	invLenSq       float32
	minX, maxX     float32
	minY, maxY     float32
}

func (s *polySegment) distSq(px, py float32) float32 {
	if s.invLenSq <= 0 {
		dx := px - s.x1
		dy := py - s.y1
		return dx*dx + dy*dy
	}
	t := ((px-s.x1)*s.vx + (py-s.y1)*s.vy) * s.invLenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	cx := s.x1 + s.vx*t
	cy := s.y1 + s.vy*t
	dx := px - cx
	dy := py - cy
	return dx*dx + dy*dy
}

func distToSegment(px, py, x1, y1, x2, y2 float32) float32 {
	vx := x2 - x1
	vy := y2 - y1
	lenSq := vx*vx + vy*vy
	if lenSq <= 1e-6 {
		dx := px - x1
		dy := py - y1
		return float32(math.Sqrt(float64(dx*dx + dy*dy)))
	}
	t := ((px-x1)*vx + (py-y1)*vy) / lenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	cx := x1 + vx*t
	cy := y1 + vy*t
	dx := px - cx
	dy := py - cy
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// DrawLine рисует сглаженный отрезок от (x1, y1) до (x2, y2) с заданной толщиной.
func (c *Canvas) DrawLine(x1, y1, x2, y2, thickness float32, col Color) {
	if col.A == 0 {
		return
	}
	if thickness <= 0 {
		thickness = 1
	}
	halfW := thickness * 0.5
	pad := halfW + 1.5

	minX, maxX := x1, x2
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	minY, maxY := y1, y2
	if minY > maxY {
		minY, maxY = maxY, minY
	}

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
	c.AddDirtyRect(area)

	vx := x2 - x1
	vy := y2 - y1
	lenSq := vx*vx + vy*vy
	maxDist := halfW + 0.5
	maxDistSq := maxDist * maxDist
	innerDist := halfW - 0.5
	innerDistSq := innerDist * innerDist

	if lenSq <= 1e-6 {
		for y := area.Y; y < area.Y+area.H; y++ {
			py := float32(y) + 0.5
			dy := py - y1
			dySq := dy * dy
			for x := area.X; x < area.X+area.W; x++ {
				px := float32(x) + 0.5
				dx := px - x1
				dSq := dx*dx + dySq
				if dSq >= maxDistSq {
					continue
				}
				var cov float32
				if innerDist > 0 && dSq <= innerDistSq {
					cov = 1.0
				} else {
					cov = clamp01(0.5 - (float32(math.Sqrt(float64(dSq))) - halfW))
				}
				if cov > 0 {
					c.blendPixelUnchecked(x, y, col, uint8(cov*255+0.5))
				}
			}
		}
		return
	}

	invLenSq := 1.0 / lenSq
	absVy := float32(math.Abs(float64(vy)))
	lineLen := float32(math.Sqrt(float64(lenSq)))

	for y := area.Y; y < area.Y+area.H; y++ {
		py := float32(y) + 0.5
		dy1 := py - y1
		vyDy1 := vy * dy1

		rowMinX := area.X
		rowMaxX := area.X + area.W

		if absVy > 1e-3 {
			xMid := x1 + dy1*(vx/vy)
			span := float32(math.Ceil(float64(maxDist * lineLen / absVy)))
			if span < pad {
				span = pad
			}
			sMin := int(math.Floor(float64(xMid - span)))
			sMax := int(math.Ceil(float64(xMid + span))) + 1

			if py < minY+pad || py > maxY-pad {
				c1Min := int(math.Floor(float64(x1 - pad)))
				c1Max := int(math.Ceil(float64(x1 + pad))) + 1
				c2Min := int(math.Floor(float64(x2 - pad)))
				c2Max := int(math.Ceil(float64(x2 + pad))) + 1
				if c1Min < sMin {
					sMin = c1Min
				}
				if c1Max > sMax {
					sMax = c1Max
				}
				if c2Min < sMin {
					sMin = c2Min
				}
				if c2Max > sMax {
					sMax = c2Max
				}
			}

			if sMin > rowMinX {
				rowMinX = sMin
			}
			if sMax < rowMaxX {
				rowMaxX = sMax
			}
		}

		for x := rowMinX; x < rowMaxX; x++ {
			px := float32(x) + 0.5
			dx1 := px - x1
			t := (dx1*vx + vyDy1) * invLenSq
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			cx := x1 + vx*t
			cy := y1 + vy*t
			dx := px - cx
			dy := py - cy
			dSq := dx*dx + dy*dy
			if dSq >= maxDistSq {
				continue
			}
			var cov float32
			if innerDist > 0 && dSq <= innerDistSq {
				cov = 1.0
			} else {
				cov = clamp01(0.5 - (float32(math.Sqrt(float64(dSq))) - halfW))
			}
			if cov > 0 {
				c.blendPixelUnchecked(x, y, col, uint8(cov*255+0.5))
			}
		}
	}
}

// DrawPolyline рисует сглаженную ломаную линию по точкам pts без наложения альфы на стыках.
func (c *Canvas) DrawPolyline(pts []PointF, thickness float32, col Color, closed bool) {
	if len(pts) < 2 || col.A == 0 {
		return
	}
	if len(pts) == 2 && !closed {
		c.DrawLine(pts[0].X, pts[0].Y, pts[1].X, pts[1].Y, thickness, col)
		return
	}
	if thickness <= 0 {
		thickness = 1
	}
	halfW := thickness * 0.5
	pad := halfW + 1.5

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
	c.AddDirtyRect(area)

	segCount := len(pts) - 1
	if closed && len(pts) > 2 {
		segCount = len(pts)
	}

	var inlineSegs [64]polySegment
	var segs []polySegment
	if segCount <= len(inlineSegs) {
		segs = inlineSegs[:segCount]
	} else {
		segs = make([]polySegment, segCount)
	}

	for i := 0; i < segCount; i++ {
		a := pts[i]
		b := pts[(i+1)%len(pts)]
		vx := b.X - a.X
		vy := b.Y - a.Y
		lenSq := vx*vx + vy*vy
		invLenSq := float32(0)
		if lenSq > 1e-6 {
			invLenSq = 1.0 / lenSq
		}
		sMinX, sMaxX := a.X, b.X
		if sMinX > sMaxX {
			sMinX, sMaxX = sMaxX, sMinX
		}
		sMinY, sMaxY := a.Y, b.Y
		if sMinY > sMaxY {
			sMinY, sMaxY = sMaxY, sMinY
		}
		segs[i] = polySegment{
			x1:       a.X,
			y1:       a.Y,
			vx:       vx,
			vy:       vy,
			invLenSq: invLenSq,
			minX:     sMinX - pad,
			maxX:     sMaxX + pad,
			minY:     sMinY - pad,
			maxY:     sMaxY + pad,
		}
	}

	maxDist := halfW + 0.5
	maxDistSq := maxDist * maxDist
	innerDist := halfW - 0.5
	innerDistSq := innerDist * innerDist

	for y := area.Y; y < area.Y+area.H; y++ {
		py := float32(y) + 0.5
		for x := area.X; x < area.X+area.W; x++ {
			px := float32(x) + 0.5
			minDistSq := float32(math.MaxFloat32)
			for i := 0; i < segCount; i++ {
				seg := &segs[i]
				if px < seg.minX || px > seg.maxX || py < seg.minY || py > seg.maxY {
					continue
				}
				dSq := seg.distSq(px, py)
				if dSq < minDistSq {
					minDistSq = dSq
				}
			}
			if minDistSq >= maxDistSq {
				continue
			}
			var cov float32
			if innerDist > 0 && minDistSq <= innerDistSq {
				cov = 1.0
			} else {
				cov = clamp01(0.5 - (float32(math.Sqrt(float64(minDistSq))) - halfW))
			}
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
