package render

// Color представляет цвет в формате RGBA.
type Color struct {
	R, G, B, A uint8
}

// RGB создаёт непрозрачный цвет.
func RGB(r, g, b uint8) Color {
	return Color{R: r, G: g, B: b, A: 255}
}

// RGBA создаёт цвет с заданной прозрачностью.
func RGBA(r, g, b, a uint8) Color {
	return Color{R: r, G: g, B: b, A: a}
}

// Hex создаёт непрозрачный цвет из 24-битного значения 0xRRGGBB.
func Hex(rgb uint32) Color {
	return Color{
		R: uint8((rgb >> 16) & 0xFF),
		G: uint8((rgb >> 8) & 0xFF),
		B: uint8(rgb & 0xFF),
		A: 255,
	}
}

// WithAlpha возвращает копию цвета с новой прозрачностью.
func (c Color) WithAlpha(a uint8) Color {
	c.A = a
	return c
}

// LerpColor линейно интерполирует между цветами a и b с коэффициентом t в [0..1].
func LerpColor(a, b Color, t float32) Color {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	inv := 1 - t
	return Color{
		R: uint8(float32(a.R)*inv + float32(b.R)*t + 0.5),
		G: uint8(float32(a.G)*inv + float32(b.G)*t + 0.5),
		B: uint8(float32(a.B)*inv + float32(b.B)*t + 0.5),
		A: uint8(float32(a.A)*inv + float32(b.A)*t + 0.5),
	}
}

// Rect описывает прямоугольник в пиксельных координатах.
type Rect struct {
	X, Y, W, H int
}

// Contains проверяет, попадает ли точка (px, py) внутрь прямоугольника.
func (r Rect) Contains(px, py int) bool {
	return px >= r.X && px < r.X+r.W && py >= r.Y && py < r.Y+r.H
}

// Intersect возвращает пересечение двух прямоугольников.
func (r Rect) Intersect(other Rect) Rect {
	x0 := maxInt(r.X, other.X)
	y0 := maxInt(r.Y, other.Y)
	x1 := minInt(r.X+r.W, other.X+other.W)
	y1 := minInt(r.Y+r.H, other.Y+other.H)
	if x1 <= x0 || y1 <= y0 {
		return Rect{}
	}
	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// Empty возвращает true, если площадь прямоугольника равна нулю.
func (r Rect) Empty() bool {
	return r.W <= 0 || r.H <= 0
}

// Intersects возвращает true, если два прямоугольника пересекаются.
func (r Rect) Intersects(other Rect) bool {
	return !r.Intersect(other).Empty()
}

// Union возвращает минимальный прямоугольник, охватывающий оба прямоугольника.
func (r Rect) Union(other Rect) Rect {
	if r.Empty() {
		return other
	}
	if other.Empty() {
		return r
	}
	x0 := minInt(r.X, other.X)
	y0 := minInt(r.Y, other.Y)
	x1 := maxInt(r.X+r.W, other.X+other.W)
	y1 := maxInt(r.Y+r.H, other.Y+other.H)
	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

const maxDirtyRects = 16

// Canvas — программный 2D-буфер в формате BGRA.
type Canvas struct {
	Pixels     []byte
	Width      int
	Height     int
	clips      []Rect
	dirtyRects []Rect
}

// NewCanvas создаёт новый пустой холст.
func NewCanvas(w, h int) *Canvas {
	c := &Canvas{}
	c.Resize(w, h)
	return c
}

// Resize меняет размер холста, переиспользуя память буфера при возможности.
func (c *Canvas) Resize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	c.Width = w
	c.Height = h
	need := w * h * 4
	if cap(c.Pixels) < need {
		c.Pixels = make([]byte, need)
	} else {
		c.Pixels = c.Pixels[:need]
	}
	c.clips = c.clips[:0]
	c.clips = append(c.clips, Rect{X: 0, Y: 0, W: w, H: h})
	c.MarkFullDirty()
}

// ResizeWithBuffer меняет размер холста и привязывает внешний буфер пикселей (если он передан),
// позволяя рисовать напрямую в память платформы (например, Windows DIBSection) без копирования.
func (c *Canvas) ResizeWithBuffer(w, h int, external []byte) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	need := w * h * 4
	if need > 0 && len(external) >= need {
		c.Width = w
		c.Height = h
		c.Pixels = external[:need]
		c.clips = c.clips[:0]
		c.clips = append(c.clips, Rect{X: 0, Y: 0, W: w, H: h})
		c.MarkFullDirty()
		return
	}
	c.Resize(w, h)
}

// Clear заливает весь буфер указанным цветом.
func (c *Canvas) Clear(col Color) {
	if len(c.Pixels) == 0 {
		return
	}
	// Заполняем первую строку, затем копируем её экспоненциально по буферу
	stride := c.Width * 4
	for x := 0; x < stride; x += 4 {
		c.Pixels[x+0] = col.B
		c.Pixels[x+1] = col.G
		c.Pixels[x+2] = col.R
		c.Pixels[x+3] = col.A
	}
	for filled := stride; filled < len(c.Pixels); filled *= 2 {
		copy(c.Pixels[filled:], c.Pixels[:filled])
	}
	c.MarkFullDirty()
}

// ClearTransparent быстро очищает буфер полностью прозрачными пикселями (нули).
func (c *Canvas) ClearTransparent() {
	if len(c.Pixels) == 0 {
		return
	}
	clear(c.Pixels)
	c.MarkFullDirty()
}

// AddDirtyRect регистрирует изменённую область холста для оптимизации перерисовки и блиттинга.
func (c *Canvas) AddDirtyRect(r Rect) {
	if c == nil || c.Width <= 0 || c.Height <= 0 {
		return
	}
	bounds := Rect{X: 0, Y: 0, W: c.Width, H: c.Height}
	clipped := bounds.Intersect(r)
	if clipped.Empty() {
		return
	}
	for i := range c.dirtyRects {
		existing := c.dirtyRects[i]
		if existing.X <= clipped.X && existing.Y <= clipped.Y &&
			existing.X+existing.W >= clipped.X+clipped.W &&
			existing.Y+existing.H >= clipped.Y+clipped.H {
			return
		}
		if existing.Intersects(clipped) {
			c.dirtyRects[i] = existing.Union(clipped)
			return
		}
	}
	if len(c.dirtyRects) < maxDirtyRects {
		c.dirtyRects = append(c.dirtyRects, clipped)
		return
	}
	merged := clipped
	for _, dr := range c.dirtyRects {
		merged = merged.Union(dr)
	}
	c.dirtyRects[0] = merged
	c.dirtyRects = c.dirtyRects[:1]
}

// MarkFullDirty помечает весь холст как изменённый.
func (c *Canvas) MarkFullDirty() {
	if c == nil || c.Width <= 0 || c.Height <= 0 {
		c.dirtyRects = c.dirtyRects[:0]
		return
	}
	full := Rect{X: 0, Y: 0, W: c.Width, H: c.Height}
	if len(c.dirtyRects) == 0 {
		c.dirtyRects = append(c.dirtyRects, full)
	} else {
		c.dirtyRects[0] = full
		c.dirtyRects = c.dirtyRects[:1]
	}
}

// DirtyRects возвращает список накопленных изменённых областей текущего кадра.
func (c *Canvas) DirtyRects() []Rect {
	if c == nil {
		return nil
	}
	return c.dirtyRects
}

// DirtyBounds возвращает единый прямоугольник, охватывающий все изменённые области.
func (c *Canvas) DirtyBounds() Rect {
	if c == nil || len(c.dirtyRects) == 0 {
		return Rect{}
	}
	res := c.dirtyRects[0]
	for _, dr := range c.dirtyRects[1:] {
		res = res.Union(dr)
	}
	return res
}

// ResetDirtyRects сбрасывает список изменённых областей после вывода кадра.
func (c *Canvas) ResetDirtyRects() {
	if c != nil {
		c.dirtyRects = c.dirtyRects[:0]
	}
}

// BlitFrom быстро копирует прямоугольную область area из src в текущий холст (без альфа-смешивания).
func (c *Canvas) BlitFrom(src *Canvas, area Rect) {
	if c == nil || src == nil || len(c.Pixels) == 0 || len(src.Pixels) == 0 {
		return
	}
	vis := c.CurrentClip().Intersect(area)
	vis = vis.Intersect(Rect{X: 0, Y: 0, W: src.Width, H: src.Height})
	if vis.Empty() {
		return
	}
	// Быстрая ветка: копирование всего буфера целиком при совпадении размеров
	if vis.X == 0 && vis.Y == 0 && vis.W == c.Width && vis.H == c.Height &&
		c.Width == src.Width && c.Height == src.Height {
		copy(c.Pixels, src.Pixels)
		c.AddDirtyRect(vis)
		return
	}
	rowBytes := vis.W * 4
	for y := vis.Y; y < vis.Y+vis.H; y++ {
		dstOff := (y*c.Width + vis.X) * 4
		srcOff := (y*src.Width + vis.X) * 4
		copy(c.Pixels[dstOff:dstOff+rowBytes], src.Pixels[srcOff:srcOff+rowBytes])
	}
	c.AddDirtyRect(vis)
}

// CompositeFrom накладывает прямоугольную область area из src поверх текущего холста с учётом альфа-канала.
func (c *Canvas) CompositeFrom(src *Canvas, area Rect) {
	if c == nil || src == nil || len(c.Pixels) == 0 || len(src.Pixels) == 0 {
		return
	}
	vis := c.CurrentClip().Intersect(area)
	vis = vis.Intersect(Rect{X: 0, Y: 0, W: src.Width, H: src.Height})
	if vis.Empty() {
		return
	}
	for y := vis.Y; y < vis.Y+vis.H; y++ {
		dstRow := (y*c.Width + vis.X) * 4
		srcRow := (y*src.Width + vis.X) * 4
		for x := 0; x < vis.W; x++ {
			so := srcRow + x*4
			a := uint32(src.Pixels[so+3])
			if a == 0 {
				continue
			}
			do := dstRow + x*4
			if a == 255 {
				c.Pixels[do+0] = src.Pixels[so+0]
				c.Pixels[do+1] = src.Pixels[so+1]
				c.Pixels[do+2] = src.Pixels[so+2]
				c.Pixels[do+3] = 255
				continue
			}
			invA := 255 - a
			c.Pixels[do+0] = uint8((uint32(src.Pixels[so+0])*a + uint32(c.Pixels[do+0])*invA + 127) / 255)
			c.Pixels[do+1] = uint8((uint32(src.Pixels[so+1])*a + uint32(c.Pixels[do+1])*invA + 127) / 255)
			c.Pixels[do+2] = uint8((uint32(src.Pixels[so+2])*a + uint32(c.Pixels[do+2])*invA + 127) / 255)
			c.Pixels[do+3] = 255
		}
	}
	c.AddDirtyRect(vis)
}

// CurrentClip возвращает текущую область отсечения.
func (c *Canvas) CurrentClip() Rect {
	if len(c.clips) == 0 {
		return Rect{X: 0, Y: 0, W: c.Width, H: c.Height}
	}
	return c.clips[len(c.clips)-1]
}

// PushClip ограничивает дальнейшую отрисовку пересечением текущего клипа и r.
func (c *Canvas) PushClip(r Rect) {
	clipped := c.CurrentClip().Intersect(r)
	c.clips = append(c.clips, clipped)
}

// PopClip снимает последнее ограничение области отсечения.
func (c *Canvas) PopClip() {
	if len(c.clips) > 1 {
		c.clips = c.clips[:len(c.clips)-1]
	}
}

// FillRect рисует обычный прямоугольник с учётом текущего клипа и прозрачности.
func (c *Canvas) FillRect(r Rect, col Color) {
	if col.A == 0 {
		return
	}
	area := c.CurrentClip().Intersect(r)
	if area.Empty() {
		return
	}
	c.AddDirtyRect(area)

	if col.A == 255 {
		for y := area.Y; y < area.Y+area.H; y++ {
			row := (y*c.Width + area.X) * 4
			for x := 0; x < area.W; x++ {
				i := row + x*4
				c.Pixels[i+0] = col.B
				c.Pixels[i+1] = col.G
				c.Pixels[i+2] = col.R
				c.Pixels[i+3] = 255
			}
		}
		return
	}

	for y := area.Y; y < area.Y+area.H; y++ {
		for x := area.X; x < area.X+area.W; x++ {
			c.blendPixelUnchecked(x, y, col, 255)
		}
	}
}

// BlendPixel смешивает один пиксель с учётом клипа и покрытия cov (0..255).
func (c *Canvas) BlendPixel(x, y int, col Color, cov uint8) {
	clip := c.CurrentClip()
	if x < clip.X || x >= clip.X+clip.W || y < clip.Y || y >= clip.Y+clip.H {
		return
	}
	c.blendPixelUnchecked(x, y, col, cov)
}

func (c *Canvas) blendPixelUnchecked(x, y int, col Color, cov uint8) {
	if cov == 0 || col.A == 0 {
		return
	}
	// Итоговая альфа = (col.A * cov) / 255
	a := (uint32(col.A)*uint32(cov) + 127) / 255
	if a == 0 {
		return
	}
	i := (y*c.Width + x) * 4
	if a >= 255 {
		c.Pixels[i+0] = col.B
		c.Pixels[i+1] = col.G
		c.Pixels[i+2] = col.R
		c.Pixels[i+3] = 255
		return
	}

	invA := 255 - a
	c.Pixels[i+0] = uint8((uint32(col.B)*a + uint32(c.Pixels[i+0])*invA + 127) / 255)
	c.Pixels[i+1] = uint8((uint32(col.G)*a + uint32(c.Pixels[i+1])*invA + 127) / 255)
	c.Pixels[i+2] = uint8((uint32(col.R)*a + uint32(c.Pixels[i+2])*invA + 127) / 255)
	c.Pixels[i+3] = 255
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

