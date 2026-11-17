package ui

import "github.com/kirpicheff/anyui/render"

type overlayItem struct {
	bounds render.Rect
	draw   func()
}

// OverlayLayer управляет отложенной отрисовкой всплывающих элементов поверх основного UI
// и блокирует клики по виджетам под активными оверлеями.
type OverlayLayer struct {
	prevBlocks   []render.Rect
	nextBlocks   []render.Rect
	items        []overlayItem
	inOverlay    bool
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

