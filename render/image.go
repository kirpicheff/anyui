package render

import (
	"image"
	"math"
)

// ImageScaleMode задаёт режим масштабирования изображения внутри целевого прямоугольника.
type ImageScaleMode uint8

const (
	// ScaleFit вписывает изображение целиком с сохранением пропорций.
	ScaleFit ImageScaleMode = iota
	// ScaleFill заполняет весь прямоугольник с сохранением пропорций и обрезкой лишнего.
	ScaleFill
	// ScaleOneToOne выводит изображение в исходном масштабе 1:1 по центру области.
	ScaleOneToOne
)

// DrawBGRA рисует внешний буфер пикселей в формате BGRA (4 байта на пиксель).
func (c *Canvas) DrawBGRA(dst Rect, src []byte, srcW, srcH int, mode ImageScaleMode) {
	c.blitRawBuffer(dst, src, srcW, srcH, true, mode)
}

// DrawRGBA рисует внешний буфер пикселей в формате RGBA (4 байта на пиксель).
func (c *Canvas) DrawRGBA(dst Rect, src []byte, srcW, srcH int, mode ImageScaleMode) {
	c.blitRawBuffer(dst, src, srcW, srcH, false, mode)
}

// DrawImage рисует стандартный image.Image внутри прямоугольника dst.
func (c *Canvas) DrawImage(dst Rect, img image.Image, mode ImageScaleMode) {
	if img == nil || dst.Empty() {
		return
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return
	}

	if nrgba, ok := img.(*image.NRGBA); ok && nrgba.Stride == w*4 && b.Min.X == 0 && b.Min.Y == 0 {
		c.DrawRGBA(dst, nrgba.Pix, w, h, mode)
		return
	}

	buf := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			off := (y*w + x) * 4
			buf[off+0] = uint8(r >> 8)
			buf[off+1] = uint8(g >> 8)
			buf[off+2] = uint8(bl >> 8)
			buf[off+3] = uint8(a >> 8)
		}
	}
	c.DrawRGBA(dst, buf, w, h, mode)
}

func (c *Canvas) blitRawBuffer(dst Rect, src []byte, srcW, srcH int, isBGRA bool, mode ImageScaleMode) {
	if c == nil || dst.Empty() || srcW <= 0 || srcH <= 0 || len(src) < srcW*srcH*4 {
		return
	}

	target := computeTargetRect(dst, srcW, srcH, mode)
	if target.Empty() {
		return
	}

	clip := c.CurrentClip().Intersect(dst)
	vis := clip.Intersect(target)
	if vis.Empty() {
		return
	}

	// Быстрая ветка 1:1 без масштабирования
	if target.W == srcW && target.H == srcH {
		for y := vis.Y; y < vis.Y+vis.H; y++ {
			sy := y - target.Y
			for x := vis.X; x < vis.X+vis.W; x++ {
				sx := x - target.X
				col := readPixel(src, (sy*srcW+sx)*4, isBGRA)
				if col.A == 255 {
					off := (y*c.Width + x) * 4
					c.Pixels[off+0] = col.B
					c.Pixels[off+1] = col.G
					c.Pixels[off+2] = col.R
					c.Pixels[off+3] = 255
				} else if col.A > 0 {
					c.blendPixelUnchecked(x, y, col, 255)
				}
			}
		}
		return
	}

	// Билинейная фильтрация при масштабировании
	scaleX := float32(srcW) / float32(target.W)
	scaleY := float32(srcH) / float32(target.H)

	for y := vis.Y; y < vis.Y+vis.H; y++ {
		fy := (float32(y-target.Y)+0.5)*scaleY - 0.5
		if fy < 0 {
			fy = 0
		}
		y0 := int(math.Floor(float64(fy)))
		y1 := y0 + 1
		if y1 >= srcH {
			y1 = srcH - 1
		}
		wy := fy - float32(y0)

		for x := vis.X; x < vis.X+vis.W; x++ {
			fx := (float32(x-target.X)+0.5)*scaleX - 0.5
			if fx < 0 {
				fx = 0
			}
			x0 := int(math.Floor(float64(fx)))
			x1 := x0 + 1
			if x1 >= srcW {
				x1 = srcW - 1
			}
			wx := fx - float32(x0)

			c00 := readPixel(src, (y0*srcW+x0)*4, isBGRA)
			c10 := readPixel(src, (y0*srcW+x1)*4, isBGRA)
			c01 := readPixel(src, (y1*srcW+x0)*4, isBGRA)
			c11 := readPixel(src, (y1*srcW+x1)*4, isBGRA)

			top := LerpColor(c00, c10, wx)
			bot := LerpColor(c01, c11, wx)
			col := LerpColor(top, bot, wy)
			if col.A > 0 {
				c.blendPixelUnchecked(x, y, col, 255)
			}
		}
	}
}

func computeTargetRect(dst Rect, srcW, srcH int, mode ImageScaleMode) Rect {
	if mode == ScaleOneToOne {
		return Rect{
			X: dst.X + (dst.W-srcW)/2,
			Y: dst.Y + (dst.H-srcH)/2,
			W: srcW,
			H: srcH,
		}
	}

	rw := float32(dst.W) / float32(srcW)
	rh := float32(dst.H) / float32(srcH)
	scale := rw
	if mode == ScaleFit {
		if rh < scale {
			scale = rh
		}
	} else {
		if rh > scale {
			scale = rh
		}
	}

	tw := int(float32(srcW)*scale + 0.5)
	th := int(float32(srcH)*scale + 0.5)
	return Rect{
		X: dst.X + (dst.W-tw)/2,
		Y: dst.Y + (dst.H-th)/2,
		W: tw,
		H: th,
	}
}

func readPixel(src []byte, off int, isBGRA bool) Color {
	if isBGRA {
		return Color{B: src[off+0], G: src[off+1], R: src[off+2], A: src[off+3]}
	}
	return Color{R: src[off+0], G: src[off+1], B: src[off+2], A: src[off+3]}
}

