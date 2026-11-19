[English](graphics.md) | [Русский](../ru/graphics.md)

# Graphics, Effects, SVG, and Images

In `anyui`, all graphics are built on the software `render.Canvas` renderer in BGRA format, guaranteeing identical appearance across all systems without GPU dependencies.

## Per-Corner Radii (`CornerRadii`)

```go
canvas.DrawRoundedBoxCorners(
	area,
	anyui.CornerRadii{TL: 12, TR: 12, BR: 0, BL: 0},
	ui.Theme().Card,
	1.0,
	ui.Theme().Border,
)
```

## Gradients (`LinearGradient`, `RadialGradient`)

Gradient coordinates are specified in the normalized `[0..1]` range relative to the target rectangle:

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

## Soft Shadows (`BoxShadow`) and Frosted Glass (`BackdropBlur`)

```go
canvas.DrawBoxShadow(cardRect, anyui.BoxShadow{
	OffsetY: 6,
	Blur:    18,
	Color:   anyui.RGBA(0, 0, 0, 90),
	Radii:   anyui.UniformRadii(12),
})

// Blur background under a semi-transparent panel
canvas.BackdropBlur(panelRect, 8, anyui.UniformRadii(12))
```

## Vector Primitives (`DrawLine`, `DrawPolyline`, `DrawArc`)

```go
ui.CanvasView(240, 80, func(canvas *render.Canvas, area anyui.Rect) {
	pts := []anyui.PointF{
		{X: float32(area.X), Y: float32(area.Y + 60)},
		{X: float32(area.X + 80), Y: float32(area.Y + 20)},
		{X: float32(area.X + 160), Y: float32(area.Y + 45)},
		{X: float32(area.X + 240), Y: float32(area.Y + 10)},
	}
	canvas.DrawPolyline(pts, 2.0, ui.Theme().Accent, false)
})
```

## SVG Icons

SVG icons are parsed in pure Go, cached by `(svgHash, sizePx)`, and tinted dynamically at draw time:

```go
const saveSVG = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
  <path d="M5 13l4 4L19 7"/>
</svg>`

ui.Button("btn_save", "Save", anyui.Icon(saveSVG), anyui.Primary())
```

## Images and Video Frames

- `ui.Image(img, wDp, hDp, anyui.ScaleFit)` — render an `image.Image` (`ScaleFit`, `ScaleFill`, `ScaleOneToOne`).
- For high-frequency video streams (e.g., remote desktop), use `canvas.DrawBGRAFast(area, pixels, srcW, srcH, anyui.ScaleFit)` (zero-allocation, bypasses image cache) inside `ui.InteractiveCanvas` or `ui.CanvasView`.
- For an interactive canvas with focus, mouse drag capture, and raw keyboard (`ScanCode`, `Extended`, `KeyDown`/`KeyUp`) and mouse/wheel input, use `ui.InteractiveCanvas`:

```go
ui.InteractiveCanvas("rdp_view", 0, 0, anyui.CanvasCallbacks{
	OnDraw: func(canvas *render.Canvas, area anyui.Rect, focused bool) {
		canvas.DrawBGRAFast(area, frameBGRA, frameW, frameH, anyui.ScaleFit)
	},
	OnKey: func(ev anyui.KeyEvent) {
		// ev.Pressed, ev.KeyCode, ev.ScanCode, ev.Extended, ev.Mods
	},
	OnMouseMove: func(x, y int, mods anyui.Modifiers) {
		// x, y are physical pixels relative to area.X, area.Y
	},
	OnMouseDown:  func(x, y int, btn anyui.MouseButton, mods anyui.Modifiers) {},
	OnMouseUp:    func(x, y int, btn anyui.MouseButton, mods anyui.Modifiers) {},
	OnMouseWheel: func(x, y int, wx, wy float32, mods anyui.Modifiers) {},
	OnFocusLost: func() {
		// Release held modifier keys on blur
	},
})
```
- For OS image clipboard exchange, use `win.ClipboardImage()` and `win.SetClipboardImage(img)`.

## Font Styles and Fallback

`FontManager` automatically loads system `FontRegular`, `FontBold`, and `FontMono`, using 4-phase subpixel X positioning and font kerning. To embed custom `.ttf`/`.otf` fonts (for example via `//go:embed`), pass their bytes in `anyui.Config`: `FontRegular`, `FontBold`, `FontMono` (system fallback fonts are preserved). Use `anyui.Bold()`, `anyui.Mono()`, `anyui.LetterSpacing(dp)`, and `anyui.TextShadow(offsetX, offsetY, blur, col)` on widgets:

```go
ui.Label("Code: 000 000", anyui.Mono(), anyui.FontSize(18), anyui.LetterSpacing(1.5))
ui.Label("Heading", anyui.Bold(), anyui.FontSize(20), anyui.TextShadow(0, 0, 8, anyui.RGBA(80, 180, 255, 160)))
```

## Rendering Optimization, Layers, and Caches

- **Layers (`StaticLayer` / `Dynamic` / `Overlay`) and `Dirty Rectangles`:**
  Heavy static content (backgrounds, descriptions, headings) can be wrapped in `ui.StaticLayer("id", func() { ... })`. It is rasterized into an offscreen buffer and blitted via `BlitFrom` on subsequent frames as long as coordinates, size, theme, and DPI remain unchanged. Call `ui.InvalidateStatic()` to force a redraw. When presenting to the window, a minimal `DirtyRect` is computed and sent via `platform.DirtyPresenter` (`PresentRect`).
- **Streaming fast-path (`Streaming` + `DirectInput`) and direct GDI rendering (`DIBSection` + `BitBlt`):**
  When `Streaming: true` is set in `CanvasCallbacks`, background frame updates triggered by `win.WakeUp()` (`EventWakeUp`) skip `Canvas.Clear`, full widget tree reconstruction, and pixel diffing — only the streaming canvas `OnDraw` runs over the existing buffer and presents the canvas rectangle via `PresentRect`. Call `ui.RequestRedraw()` (which switches the `Streaming` fast-path to a full `drawFn` pass within the current `Render()` cycle) or `win.Invalidate()` when surrounding UI elements (such as a once-per-second status-bar FPS counter) need updating. On Windows, the window back buffer is allocated via `CreateDIBSection` (`platform.BackBufferProvider`), allowing `Canvas` and `DrawBGRAFast` to write directly into GDI memory without intermediate copies and blit via `BitBlt`. With `DirectInput: true`, mouse and keyboard events are delivered directly to canvas callbacks without triggering `Render()`.
- **Smart `EventMouseMove` filtering:**
  The context tracks `HotID` and active cursor across registered hover regions (`RegisterHoverArea` for custom zones) and skips redundant redraws when moving the mouse within the same widget or over empty background.
- **LRU glyph and image caches:**
  `FontManager` caches colored glyph bitmaps by `(style, rune, sizePx, color)` (configured via `SetGlyphCacheCapacity`, `GlyphCacheStats`, `ClearGlyphCache`), and `render.Canvas` caches scaled images (`ClearScaledImageCache`, `ScaledImageCacheStats`, `SetScaledImageCacheLimits`).
- **System cursors:**
  Supports `CursorArrow`, `CursorHand`, `CursorIBeam`, `CursorResizeH`, `CursorResizeV`, `CursorResizeNWSE`, `CursorResizeNESW`, `CursorNotAllowed`, `CursorCrosshair`, `CursorMove` (`ui.SetCursor(...)`).

