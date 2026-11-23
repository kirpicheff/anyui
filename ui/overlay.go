package ui

import "github.com/kirpicheff/anyui/render"

type overlayItem struct {
	bounds render.Rect
	draw   func()
}

// OverlayLayer управляет отложенной отрисовкой всплывающих элементов поверх основного UI
// и блокирует клики по виджетам под активными оверлеями.
type OverlayLayer struct {
	prevBlocks []render.Rect
	nextBlocks []render.Rect
	items      []overlayItem
	inOverlay  bool
}

// BeginFrame подготавливает слой оверлеев к новому кадру.
func (o *OverlayLayer) BeginFrame() {
	o.prevBlocks, o.nextBlocks = o.nextBlocks, o.prevBlocks[:0]
	o.items = o.items[:0]
	o.inOverlay = false
}

// Push регистрирует оверлей, который будет отрисован в конце кадра поверх основного интерфейса.
// Прямоугольник bounds блокирует ввод для виджетов основного слоя на следующем кадре.
func (o *OverlayLayer) Push(bounds render.Rect, drawFn func()) {
	if drawFn == nil {
		return
	}
	if !bounds.Empty() {
		o.nextBlocks = append(o.nextBlocks, bounds)
	}
	o.items = append(o.items, overlayItem{
		bounds: bounds,
		draw:   drawFn,
	})
}

// IsBlocked возвращает true, если точка (x, y) перекрыта активным оверлеем с предыдущего кадра.
// Внутри самого оверлея ввод не блокируется.
func (o *OverlayLayer) IsBlocked(x, y int) bool {
	if o.inOverlay {
		return false
	}
	for _, b := range o.prevBlocks {
		if b.Contains(x, y) {
			return true
		}
	}
	return false
}

// HasActive возвращает true, если в текущем или предыдущем кадре были активные оверлеи (меню, попапы, модалки).
func (o *OverlayLayer) HasActive() bool {
	return len(o.prevBlocks) > 0 || len(o.nextBlocks) > 0 || len(o.items) > 0
}

// Flush отрисовывает все накопленные оверлеи поверх основного интерфейса.
func (o *OverlayLayer) Flush() {
	o.FlushWithHook(nil)
}

// FlushWithHook отрисовывает оверлеи поверх основного интерфейса, вызывая onItem перед каждым элементом.
func (o *OverlayLayer) FlushWithHook(onItem func(bounds render.Rect)) {
	if len(o.items) == 0 {
		return
	}
	o.inOverlay = true
	for _, it := range o.items {
		if onItem != nil {
			onItem(it.bounds)
		}
		it.draw()
	}
	o.inOverlay = false
}

// overlayBoundsIntersects проверяет пересечение прямоугольника r с областью оверлея bounds,
// расширенной на padPx (для учёта теней, выходящих за пределы bounds).
// При пустом bounds оверлей считается потенциально пересекающимся.
func overlayBoundsIntersects(bounds, r render.Rect, padPx int) bool {
	if r.Empty() {
		return false
	}
	if bounds.Empty() {
		return true
	}
	if padPx > 0 {
		bounds = render.Rect{
			X: bounds.X - padPx,
			Y: bounds.Y - padPx,
			W: bounds.W + padPx*2,
			H: bounds.H + padPx*2,
		}
	}
	return bounds.Intersects(r)
}

// IntersectsAny возвращает true, если хотя бы один оверлей с последнего полного кадра
// пересекается с прямоугольником r (с учётом запаса padPx под тень).
func (o *OverlayLayer) IntersectsAny(r render.Rect, padPx int) bool {
	for _, it := range o.items {
		if overlayBoundsIntersects(it.bounds, r, padPx) {
			return true
		}
	}
	return false
}

// RedrawIntersecting повторно отрисовывает только те оверлеи с последнего полного кадра,
// чьи области (с запасом padPx под тень) пересекаются с r.
// Сохраняет и восстанавливает длины внутренних срезов, чтобы вложенные Push на wakeOnly-кадре
// не накапливали элементы между полными кадрами.
func (o *OverlayLayer) RedrawIntersecting(r render.Rect, padPx int) {
	n := len(o.items)
	if n == 0 || r.Empty() {
		return
	}
	savedBlocksLen := len(o.nextBlocks)
	o.inOverlay = true
	for i := 0; i < n; i++ {
		it := o.items[i]
		if overlayBoundsIntersects(it.bounds, r, padPx) {
			it.draw()
		}
	}
	o.inOverlay = false
	o.items = o.items[:n]
	o.nextBlocks = o.nextBlocks[:savedBlocksLen]
}
