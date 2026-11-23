package ui

import "github.com/kirpicheff/anyui/render"

type overlayItem struct {
	bounds      render.Rect
	draw        func()
	layoutSeq   int
	idSeqTop    int
	cache       *render.Canvas
	nonBlocking bool
}

// OverlayLayer управляет отложенной отрисовкой всплывающих элементов поверх основного UI
// и блокирует клики по виджетам под активными оверлеями.
type OverlayLayer struct {
	prevBlocks []render.Rect
	nextBlocks []render.Rect
	items      []overlayItem
	inOverlay  bool
	canvasPool []*render.Canvas
}

// BeginFrame подготавливает слой оверлеев к новому кадру и сбрасывает кэши оверлеев с возвратом в пул.
func (o *OverlayLayer) BeginFrame() {
	o.prevBlocks, o.nextBlocks = o.nextBlocks, o.prevBlocks[:0]
	for i := range o.items {
		if o.items[i].cache != nil {
			o.canvasPool = append(o.canvasPool, o.items[i].cache)
			o.items[i].cache = nil
		}
	}
	o.items = o.items[:0]
	o.inOverlay = false
}

// AllocCache возвращает чистый прозрачный холст заданного размера из пула либо создаёт новый.
func (o *OverlayLayer) AllocCache(w, h int) *render.Canvas {
	if w <= 0 || h <= 0 {
		return nil
	}
	for i := len(o.canvasPool) - 1; i >= 0; i-- {
		cv := o.canvasPool[i]
		if cv.Width == w && cv.Height == h {
			o.canvasPool = append(o.canvasPool[:i], o.canvasPool[i+1:]...)
			cv.Clear(render.RGBA(0, 0, 0, 0))
			return cv
		}
	}
	cv := render.NewCanvas(w, h)
	cv.Clear(render.RGBA(0, 0, 0, 0))
	return cv
}

// Push регистрирует оверлей, который будет отрисован в конце кадра поверх основного интерфейса.
// Прямоугольник bounds блокирует ввод для виджетов основного слоя на следующем кадре.
func (o *OverlayLayer) Push(bounds render.Rect, drawFn func()) {
	o.push(bounds, drawFn, false)
}

// PushNonBlocking регистрирует оверлей, который будет отрисован в конце кадра поверх основного интерфейса,
// но не блокирует ввод и клики для виджетов основного слоя под ним.
func (o *OverlayLayer) PushNonBlocking(bounds render.Rect, drawFn func()) {
	o.push(bounds, drawFn, true)
}

func (o *OverlayLayer) push(bounds render.Rect, drawFn func(), nonBlocking bool) {
	if drawFn == nil {
		return
	}
	if !bounds.Empty() && !nonBlocking {
		o.nextBlocks = append(o.nextBlocks, bounds)
	}
	o.items = append(o.items, overlayItem{
		bounds:      bounds,
		draw:        drawFn,
		nonBlocking: nonBlocking,
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

// IsBlockedCurrent возвращает true, если точка (x, y) перекрыта оверлеем из последнего отрисованного кадра.
// Используется между кадрами (в HandleEvent / tryDirectCanvasEvent), где актуальные границы лежат в nextBlocks.
func (o *OverlayLayer) IsBlockedCurrent(x, y int) bool {
	for _, b := range o.nextBlocks {
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
	o.FlushWithState(func(bounds render.Rect, _ bool) (int, int) {
		if onItem != nil {
			onItem(bounds)
		}
		return 0, 0
	})
}

// FlushWithState отрисовывает оверлеи поверх основного интерфейса, сохраняя для каждого элемента
// значения счётчиков layoutSeq и idSeqTop на момент начала его draw(), чтобы последующая
// перерисовка на wakeOnly-кадре генерировала идентичные ключи justifySizes и ResolveID.
func (o *OverlayLayer) FlushWithState(onItem func(bounds render.Rect, nonBlocking bool) (layoutSeq, idSeqTop int)) {
	o.FlushWithStateAndCache(onItem, nil)
}

// FlushWithStateAndCache отрисовывает оверлеи поверх основного интерфейса, сохраняя для каждого элемента
// значения счётчиков и вызывая renderItem (если он задан) для отрисовки элемента в отдельный кэш-буфер.
func (o *OverlayLayer) FlushWithStateAndCache(
	onItem func(bounds render.Rect, nonBlocking bool) (layoutSeq, idSeqTop int),
	renderItem func(it *overlayItem, bounds render.Rect),
) {
	if len(o.items) == 0 {
		return
	}
	o.inOverlay = true
	for i := 0; i < len(o.items); i++ {
		if onItem != nil {
			o.items[i].layoutSeq, o.items[i].idSeqTop = onItem(o.items[i].bounds, o.items[i].nonBlocking)
		}
		if renderItem != nil {
			renderItem(&o.items[i], o.items[i].bounds)
		} else {
			o.items[i].draw()
		}
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
// Если передан целевой холст target и у оверлея сохранён кэш-буфер (cache != nil),
// выполняется быстрый target.CompositeFrom(it.cache, r) поверх видеопотока.
// Иначе вызывается резервный draw() с восстановлением layoutSeq и idSeqTop.
// Сохраняет и восстанавливает длины внутренних срезов, чтобы вложенные Push на wakeOnly-кадре
// не накапливали элементы между полными кадрами.
func (o *OverlayLayer) RedrawIntersecting(target *render.Canvas, r render.Rect, padPx int, beforeDraw func(layoutSeq, idSeqTop int)) {
	n := len(o.items)
	if n == 0 || r.Empty() {
		return
	}
	savedBlocksLen := len(o.nextBlocks)
	o.inOverlay = true
	for i := 0; i < n; i++ {
		it := o.items[i]
		if overlayBoundsIntersects(it.bounds, r, padPx) {
			if target != nil && it.cache != nil {
				target.CompositeFrom(it.cache, r)
			} else {
				if beforeDraw != nil {
					beforeDraw(it.layoutSeq, it.idSeqTop)
				}
				it.draw()
			}
		}
	}
	o.inOverlay = false
	o.items = o.items[:n]
	o.nextBlocks = o.nextBlocks[:savedBlocksLen]
}
