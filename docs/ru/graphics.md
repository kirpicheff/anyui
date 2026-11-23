[English](../en/graphics.md) | [Русский](graphics.md)

# Графика, эффекты, SVG и изображения

В `anyui` вся графика строится на программном рендерере `render.Canvas` в формате BGRA, что гарантирует одинаковый вид на любых системах без зависимости от GPU.

## Разные радиусы по углам (`CornerRadii`)

```go
canvas.DrawRoundedBoxCorners(
	area,
	anyui.CornerRadii{TL: 12, TR: 12, BR: 0, BL: 0},
	ui.Theme().Card,
	1.0,
	ui.Theme().Border,
)
```

## Градиенты (`LinearGradient`, `RadialGradient`)

Координаты градиента задаются в нормализованном диапазоне `[0..1]` относительно прямоугольника:

```go
canvas.FillLinearGradient(area, anyui.UniformRadii(12), anyui.LinearGradient{
	StartX: 0, StartY: 0,
	EndX: 1, EndY: 1,
	Stops: []anyui.GradientStop{
		{Pos: 0, Color: anyui.Hex(0x3B82F6)},
		{Pos: 1, Color: anyui.Hex(0x1D4ED8)},
	},
})
```

## Мягкие тени (`BoxShadow`) и матовое стекло (`BackdropBlur`)

```go
canvas.DrawBoxShadow(cardRect, anyui.BoxShadow{
	OffsetY: 6,
	Blur:    18,
	Color:   anyui.RGBA(0, 0, 0, 90),
	Radii:   anyui.UniformRadii(12),
})

// Размытие фона под полупрозрачной панелью
canvas.BackdropBlur(panelRect, 8, anyui.UniformRadii(12))
```

## Векторные примитивы (`DrawLine`, `DrawPolyline`, `DrawArc`)

```go
ui.CanvasView(func(canvas *anyui.Canvas, area anyui.Rect) {
	pts := []anyui.PointF{
		{X: float32(area.X), Y: float32(area.Y + 60)},
		{X: float32(area.X + 80), Y: float32(area.Y + 20)},
		{X: float32(area.X + 160), Y: float32(area.Y + 45)},
		{X: float32(area.X + 240), Y: float32(area.Y + 10)},
	}
	canvas.DrawPolyline(pts, 2.0, ui.Theme().Accent, false)
}, anyui.Width(240), anyui.Height(80))
```

## SVG-иконки

SVG-иконки парсятся на чистом Go, кешируются по `(svgHash, sizePx)` и красятся в произвольный цвет (`tint`):

```go
const saveSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
  <path d="M5 13l4 4L19 7"/>
</svg>`

ui.Button("btn_save", "Сохранить", anyui.Icon(saveSVG), anyui.Primary())
```

## Вывод изображений и видеокадров

- `ui.Image(img, anyui.ScaleFit, anyui.Width(wDp), anyui.Height(hDp))` — вывод `image.Image` (`ScaleFit`, `ScaleFill`, `ScaleOneToOne`).
- Для высокочастотных видеопотоков (например, удалённый рабочий стол) используйте `canvas.DrawBGRAFast(area, pixels, srcW, srcH, anyui.ScaleFit)` (без аллокаций и кеша) внутри `ui.InteractiveCanvas` или `ui.CanvasView`.
- Для интерактивного холста с фокусом, захватом мыши при перетаскивании и сырым вводом клавиатуры (`ScanCode`, `Extended`, `KeyDown`/`KeyUp`) и мыши/колеса используйте `ui.InteractiveCanvas`:

```go
ui.InteractiveCanvas("rdp_view", anyui.CanvasCallbacks{
	OnDraw: func(canvas *anyui.Canvas, area anyui.Rect, focused bool) {
		canvas.DrawBGRAFast(area, frameBGRA, frameW, frameH, anyui.ScaleFit)
	},
	OnKey: func(ev anyui.KeyEvent) {
		// ev.Pressed, ev.KeyCode, ev.ScanCode, ev.Extended, ev.Mods
	},
	OnMouseMove: func(x, y int, mods anyui.Modifiers) {
		// x, y — физические пиксели относительно area.X, area.Y
	},
	OnMouseDown: func(x, y int, btn anyui.MouseButton, mods anyui.Modifiers) {},
	OnMouseUp:   func(x, y int, btn anyui.MouseButton, mods anyui.Modifiers) {},
	OnMouseWheel: func(x, y int, wx, wy float32, mods anyui.Modifiers) {},
	OnFocusLost: func() {
		// Сброс зажатых модификаторов при уходе фокуса
	},
	Cursor:       anyui.CursorNone, // Скрыть системный курсор над холстом (или передать CustomCursor)
	CustomCursor: remoteCursor,     // Опционально: нативный аппаратный курсор из BGRA-буфера
})
```
- Для обмена изображениями с ОС используйте `ui.ClipboardImage()` / `ui.SetClipboardImage(img)` или методы окна `win.ClipboardImage()` / `win.SetClipboardImage(img)` (поддерживается формат `CF_DIB` 24/32 bpp).

## Начертания шрифтов и fallback

`FontManager` автоматически загружает системные начертания `FontRegular`, `FontBold` и `FontMono`, использует субпиксельное позиционирование по X (4 фазы) и учитывает кернинг. Для подключения собственных `.ttf`/`.otf` шрифтов (например, через `//go:embed`) передайте их байты в `anyui.Config`: `FontRegular`, `FontBold`, `FontMono` (при этом сохраняются системные fallback-шрифты для редких символов). В виджетах используйте опции `anyui.Bold()`, `anyui.Mono()`, `anyui.LetterSpacing(dp)` и `anyui.TextShadow(offsetX, offsetY, blur, col)`:

```go
ui.Label("Код: 000 000", anyui.Mono(), anyui.FontSize(18), anyui.LetterSpacing(1.5))
ui.Label("Заголовок", anyui.Bold(), anyui.FontSize(20), anyui.TextShadow(0, 0, 8, anyui.RGBA(80, 180, 255, 160)))
```

## Оптимизация рендеринга, слои и кеши

- **Слои (`StaticLayer` / `Dynamic` / `Overlay`) и `Dirty Rectangles`:**
  Тяжёлый статичный контент (фон, описания, заголовки) можно обернуть в `ui.StaticLayer("id", func() { ... })`. Он растеризуется в отдельный буфер и на последующих кадрах (при неизменных координатах, размере, теме и DPI) копируется через `BlitFrom` без повторной отрисовки. Для принудительного сброса вызывайте `ui.InvalidateStatic()`. При выводе кадра в окно вычисляется минимальный `DirtyRect`, и через `platform.DirtyPresenter` (`PresentRect`) обновляется только изменённая область экрана.
- **Потоковый fast-path (`Streaming` + `DirectInput`) и прямой рендер (`DIBSection` + `BitBlt`):**
  При установке `Streaming: true` в `CanvasCallbacks` фоновое обновление по `win.WakeUp()` (`EventWakeUp`) пропускает `Canvas.Clear`, перестроение дерева виджетов и расчёт diff — вызывается только `OnDraw` потокового холста поверх готового буфера, после чего реально перезаписанная область (`DirtyBounds() ∩ rect`, либо весь `rect`) отправляется в `PresentRect`. Fast-path сохраняется и при активных оверлеях: если оверлей не пересекается с перезаписанной областью, он не перерисовывается, а пересекающиеся оверлеи повторно рисуются поверх неё под клипом без повторного альфа-наложения на letterbox-полосы. Для обновления окружающего UI (например, FPS в статус-баре раз в секунду) вызывайте `ui.RequestRedraw()` (переключает `Streaming` fast-path на полный проход `drawFn` в рамках текущего цикла `Render()`) или `win.Invalidate()`. На Windows пиксельный буфер окна выделяется через `CreateDIBSection` (`platform.BackBufferProvider`), поэтому `Canvas` и `DrawBGRAFast` пишут напрямую в память GDI без промежуточного копирования, а вывод выполняется через `BitBlt`. При `DirectInput: true` события мыши и клавиатуры доставляются в коллбеки холста без запуска `Render()`.
- **Умная фильтрация `EventMouseMove`:**
  Контекст сравнивает `HotID` и активный курсор по зарегистрированным областям (`RegisterHoverArea` для кастомных зон) и пропускает лишние перерисовки при движении мыши внутри одного виджета или по пустому фону.
- **LRU-кеш глифов и изображений:**
  Внутренний менеджер шрифтов кеширует цветные битмапы глифов по ключу `(style, rune, sizePx, color)` (статистика и сброс доступны через `ui.GlyphCacheStats`, `ui.ClearGlyphCache`), а холст `anyui.Canvas` кеширует масштабированные изображения (`ui.ScaledImageCacheStats`, `ui.ClearScaledImageCache`).
- **Системные и кастомные курсоры:**
  Поддерживаются `CursorArrow`, `CursorHand`, `CursorIBeam`, `CursorResizeH`, `CursorResizeV`, `CursorResizeNWSE`, `CursorResizeNESW`, `CursorNotAllowed`, `CursorCrosshair`, `CursorMove` и `CursorNone` (скрытие курсора над клиентской областью) через `ui.SetCursor(...)` или `CanvasCallbacks.Cursor`. Для аппаратного курсора ОС из BGRA-буфера (например, в клиенте удалённого рабочего стола без перерисовки на каждый `mousemove`) используйте `anyui.CustomCursor` в `CanvasCallbacks.CustomCursor` или `win.SetCustomCursor(cur)` (на Windows создаётся нативный 32-битный `HCURSOR` с альфа-каналом и кэшированием по содержимому).


