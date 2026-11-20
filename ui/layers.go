package ui

import (
	"bytes"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

// LayerKind определяет логический слой отрисовки интерфейса.
type LayerKind uint8

const (
	// LayerStatic — кешируемый слой статического фона и неизменяемых элементов.
	LayerStatic LayerKind = iota
	// LayerDynamic — основной слой интерактивных контролов.
	LayerDynamic
	// LayerOverlay — верхний слой всплывающих меню, тултипов, модальных окон и уведомлений.
	LayerOverlay
)

type hitRegion struct {
	id        string
	rect      render.Rect
	cursor    platform.CursorType
	trackMove bool
}

type staticBlockCache struct {
	dir            layoutDir
	startX         int
	startY         int
	availW         int
	availH         int
	deltaX         int
	deltaY         int
	maxUsedX       int
	maxUsedY       int
	endGridCellIdx int
	endFlowRowH    int
	rect           render.Rect
	valid          bool
}

// LayerManager управляет буферами слоёв (Static / Dynamic / Overlay) и кешем статических блоков.
type LayerManager struct {
	staticCanvas     *render.Canvas
	staticDirty      bool
	blocks           map[string]*staticBlockCache
	activeLayer      LayerKind
	lastFrameBlitted bool
}

func newLayerManager() LayerManager {
	return LayerManager{
		staticDirty: true,
		blocks:      make(map[string]*staticBlockCache),
		activeLayer: LayerDynamic,
	}
}

// InvalidateStatic помечает статический слой как требующий полной перерисовки.
func (lm *LayerManager) InvalidateStatic() {
	lm.staticDirty = true
}

func (lm *LayerManager) ensureStaticCanvas(w, h int) {
	if lm.staticCanvas == nil {
		lm.staticCanvas = render.NewCanvas(w, h)
		lm.staticDirty = true
		return
	}
	if lm.staticCanvas.Width != w || lm.staticCanvas.Height != h {
		lm.staticCanvas.Resize(w, h)
		lm.staticDirty = true
		for _, b := range lm.blocks {
			b.valid = false
		}
	}
}

// CurrentLayer возвращает текущий активный слой отрисовки.
func (c *Context) CurrentLayer() LayerKind {
	if c.Overlays.inOverlay {
		return LayerOverlay
	}
	return c.layers.activeLayer
}

// InvalidateStatic помечает кеш статического слоя как устаревший и запрашивает перерисовку.
func (c *Context) InvalidateStatic() {
	c.layers.InvalidateStatic()
	c.needsRedraw = true
}

// LastFrameStaticBlitted возвращает true, если в последнем кадре хотя бы один статический блок
// был скопирован из кеша Static Layer без повторного вызова пользовательской отрисовки.
func (c *Context) LastFrameStaticBlitted() bool {
	return c.layers.lastFrameBlitted
}

// LastDirtyRects возвращает список изменённых областей последнего отрендеренного кадра.
func (c *Context) LastDirtyRects() []render.Rect {
	return c.lastDirtyRects
}

// StaticLayer кеширует отрисовку статического блока в отдельный Static Layer.
// При повторных кадрах (например, при движении мыши или взаимодействии с кнопками)
// при неизменных координатах, размерах, теме и DPI содержимое копируется в буфер без повторной отрисовки fn.
func (c *Context) StaticLayer(id string, fn func()) {
	if fn == nil || c.Canvas == nil || len(c.layoutStack) == 0 {
		return
	}
	id = c.ResolveID("static", id)
	// Защита от вложенных вызовов StaticLayer внутри уже активного статического слоя
	if c.layers.staticCanvas != nil && c.Canvas == c.layers.staticCanvas {
		fn()
		return
	}

	c.layers.ensureStaticCanvas(c.Canvas.Width, c.Canvas.Height)
	if c.layers.blocks == nil {
		c.layers.blocks = make(map[string]*staticBlockCache)
	}

	box := c.topLayout()
	blk := c.layers.blocks[id]

	if !c.layers.staticDirty && blk != nil && blk.valid &&
		blk.dir == box.dir &&
		blk.startX == box.cursorX && blk.startY == box.cursorY &&
		blk.availW == box.rect.W && blk.availH == box.rect.H {
		if !blk.rect.Empty() {
			c.Canvas.BlitFrom(c.layers.staticCanvas, blk.rect)
		}
		box.cursorX += blk.deltaX
		box.cursorY += blk.deltaY
		if blk.maxUsedX > box.maxUsedX {
			box.maxUsedX = blk.maxUsedX
		}
		if blk.maxUsedY > box.maxUsedY {
			box.maxUsedY = blk.maxUsedY
		}
		box.gridCellIdx = blk.endGridCellIdx
		box.flowRowH = blk.endFlowRowH
		c.layers.lastFrameBlitted = true
		return
	}

	mainCanvas := c.Canvas
	startX := box.cursorX
	startY := box.cursorY
	startGridIdx := box.gridCellIdx
	startFlowH := box.flowRowH

	// Копируем актуальный фон области из основного буфера для корректного сглаживания
	copyArea := box.rect
	if copyArea.Empty() {
		copyArea = render.Rect{X: 0, Y: 0, W: mainCanvas.Width, H: mainCanvas.Height}
	}
	c.layers.staticCanvas.BlitFrom(mainCanvas, copyArea)
	c.layers.staticCanvas.ResetDirtyRects()
	c.layers.staticCanvas.PushClip(mainCanvas.CurrentClip())

	c.Canvas = c.layers.staticCanvas
	prevLayer := c.layers.activeLayer
	c.layers.activeLayer = LayerStatic
	fn()
	c.layers.activeLayer = prevLayer
	c.layers.staticCanvas.PopClip()
	c.Canvas = mainCanvas

	drawn := c.layers.staticCanvas.DirtyBounds()
	if !drawn.Empty() {
		mainCanvas.BlitFrom(c.layers.staticCanvas, drawn)
	}

	// Берём свежий указатель на верхний элемент стека (на случай реаллокации внутри fn)
	box = c.topLayout()
	if blk == nil {
		blk = &staticBlockCache{}
		c.layers.blocks[id] = blk
	}
	blk.dir = box.dir
	blk.startX = startX
	blk.startY = startY
	blk.availW = box.rect.W
	blk.availH = box.rect.H
	blk.deltaX = box.cursorX - startX
	blk.deltaY = box.cursorY - startY
	blk.maxUsedX = box.maxUsedX
	blk.maxUsedY = box.maxUsedY
	blk.endGridCellIdx = box.gridCellIdx
	blk.endFlowRowH = box.flowRowH
	blk.rect = drawn
	blk.valid = true

	// Сохраняем эти переменные от неиспользования, если понадобится отладка
	_ = startGridIdx
	_ = startFlowH
}

// RegisterHit регистрирует интерактивную область виджета для текущего кадра
// (используется для умной фильтрации EventMouseMove).
func (c *Context) RegisterHit(id string, rect render.Rect, cursor platform.CursorType) {
	c.registerHitRegion(id, rect, cursor, false)
}

// RegisterHitTrack регистрирует область, движение мыши внутри которой требует перерисовки
// (например, графики с отслеживанием ближайшей точки или элементы с тултипом).
func (c *Context) RegisterHitTrack(id string, rect render.Rect, cursor platform.CursorType) {
	c.registerHitRegion(id, rect, cursor, true)
}

func (c *Context) registerHitRegion(id string, rect render.Rect, cursor platform.CursorType, trackMove bool) {
	if id == "" || rect.Empty() {
		return
	}
	c.recordScrollChild(id, rect)
	if cursor != platform.CursorArrow && !c.IsInputBlocked() && rect.Contains(c.MouseX, c.MouseY) {
		c.nextCursor = cursor
	}
	c.nextHitRegions = append(c.nextHitRegions, hitRegion{
		id:        id,
		rect:      rect,
		cursor:    cursor,
		trackMove: trackMove,
	})
}

func (c *Context) hitTest(x, y int) (id string, cursor platform.CursorType, trackMove bool, found bool) {
	for i := len(c.hitRegions) - 1; i >= 0; i-- {
		reg := c.hitRegions[i]
		if reg.rect.Contains(x, y) {
			return reg.id, reg.cursor, reg.trackMove, true
		}
	}
	return "", platform.CursorArrow, false, false
}

// SetSmartMouseMove включает или отключает умную фильтрацию EventMouseMove по HotID.
func (c *Context) SetSmartMouseMove(enabled bool) {
	c.smartMouseMove = enabled
}

// SkippedMouseMoves возвращает количество отфильтрованных событий EventMouseMove без перерисовки.
func (c *Context) SkippedMouseMoves() uint64 {
	return c.skippedMouseMove
}

func (c *Context) shouldRedrawOnMouseMove(x, y int) bool {
	if !c.smartMouseMove {
		return true
	}
	if c.MouseDown || c.RightMouseDown || c.MiddleMouseDown || c.ActiveID != "" || c.tabDragging || c.reorderActive {
		return true
	}
	if c.Animator != nil && c.Animator.HasActive() {
		return true
	}

	id, cursor, trackMove, found := c.hitTest(x, y)
	if trackMove {
		return true
	}
	if id != c.lastHitID {
		return true
	}
	if found && cursor != c.lastCursor {
		return true
	}
	if !found && c.lastCursor != platform.CursorArrow {
		return true
	}
	return false
}

// computeDiffRect быстро находит минимальный прямоугольник, охватывающий все отличия между prev и cur.
func computeDiffRect(prev, cur []byte, w, h int) render.Rect {
	if w <= 0 || h <= 0 || len(prev) != len(cur) || len(cur) < w*h*4 {
		return render.Rect{X: 0, Y: 0, W: w, H: h}
	}
	stride := w * 4
	minY := -1
	maxY := -1
	minX := w
	maxX := -1

	for y := 0; y < h; y++ {
		off := y * stride
		if bytes.Equal(prev[off:off+stride], cur[off:off+stride]) {
			continue
		}
		if minY == -1 {
			minY = y
		}
		maxY = y

		// Находим первый изменённый пиксель в строке
		for x := 0; x < minX; x++ {
			po := off + x*4
			if prev[po] != cur[po] || prev[po+1] != cur[po+1] || prev[po+2] != cur[po+2] || prev[po+3] != cur[po+3] {
				minX = x
				break
			}
		}
		// Находим последний изменённый пиксель в строке
		for x := w - 1; x > maxX; x-- {
			po := off + x*4
			if prev[po] != cur[po] || prev[po+1] != cur[po+1] || prev[po+2] != cur[po+2] || prev[po+3] != cur[po+3] {
				maxX = x
				break
			}
		}
	}

	if minY == -1 || maxX < minX {
		return render.Rect{}
	}
	return render.Rect{
		X: minX,
		Y: minY,
		W: maxX - minX + 1,
		H: maxY - minY + 1,
	}
}
