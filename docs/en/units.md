[English](units.md) | [Русский](../ru/units.md)

# Unit Typing Tracker (Dp / Px)

This document tracks the status of strict scalar typing in the AnyUI library according to Phase 18b.

## Units Contract
- **`Dp` (`type Dp int`)** — density-independent logical pixels. Used across the entire high-level public API for layouts, containers, options, overlays, and widgets:
  - All public layout and overlay methods accept geometry strictly in `Dp`: `LayoutRect`, `Tooltip`, `PushOverlay`, `Popover`, `ContextMenu`, `RegisterHoverArea`. Coordinates from `ui.CurrentArea()` are in `Dp` and can be passed directly.
- **`Px` (`type Px int`)** — device physical pixels of the render buffer. Kept strictly for low-level drawing (`render.Canvas`), low-level canvas/table callbacks (`CanvasView` area, `Table.drawCell` area), system window metrics, and screen presentation (`LastDirtyRects`, `Config.HitTest`, `TitleBarHitTest`).
- **Orientation**: `Horizontal` and `Vertical` are the only valid enum constants of type `Orientation`. Legacy `SplitOrientation` is completely removed.
- **DPI Scaling & Conversion Rule**:
  - `Scale(dp Dp) Px` converts logical units to physical device pixels (frequent hot-path in application code).
  - `ScaleF(dp float32) float32` scales floating-point logical units.
  - `PxToDp(px Px) Dp` converts physical pixels to logical units (reverse projection for Canvas callbacks, hit-testing).
  - `PxToDpf(px float32) float32` reverse projection for floating-point values.
  - **Rule**: Only the canonical quartet `Scale` / `ScaleF` / `PxToDp` / `PxToDpf` is permitted. Introducing aliases such as `DpToPx` or `DpToPxf` is strictly prohibited.
  - **Suffix `F` Convention**: The `F` suffix strictly denotes a `float32` variant of the same operation.
- **Sentinel Value**: `const Auto Dp = -1` is the single canonical sentinel value for automatic sizing. Functions like `Auto()` or constants like `AutoSize` are removed.
- **No mixed overloads**: exactly one form per concept (parallel overloads like `Width(dp int)` alongside `WidthDp(dp Dp)` are prohibited).
- **Scalar Type Boundary**:
  - `int` is used only at the OS boundary (`Config`, `Window` window metrics, item counts/indices, column counts).
  - `float32` is used for subpixel and dimensionless values (`Radius`, `LetterSpacing`, `TextShadow`, `FlexWeight`, `ScaleF`/`PxToDpf`).
  - `Rect` is the single rectangle type (in `Dp` for layouts/overlays and in `Px` for low-level canvas/table callbacks).

---

## Signature Tracker

| Signature | int | Dp | Note |
|---|:---:|:---:|---|
| **Conversions and Coordinates** | | | |
| `func (u *UI) Scale(dp Dp) Px` | | ✓ | Converts logical pixels to physical device pixels |
| `func (u *UI) ScaleF(dp float32) float32` | | ✓ | Scales floating-point logical values |
| `func (u *UI) PxToDp(px Px) Dp` | | ✓ | Converts physical pixels to logical pixels |
| `func (u *UI) PxToDpf(px float32) float32` | | ✓ | Reverse projection for floating-point values |
| `func (u *UI) CurrentArea() Rect` | | ✓ | Current container bounds in Dp |
| `func (u *UI) CurrentAreaPx() Rect` | | ✓ | Current container bounds in physical Px |
| `func (u *UI) MousePosDp() (Dp, Dp)` | | ✓ | Window mouse cursor coordinates in Dp |
| `func (u *UI) MousePosPx() (Px, Px)` | | ✓ | Window mouse cursor coordinates in physical Px |
| `const Auto Dp = -1` | | ✓ | Canonical sentinel value for automatic sizing |
| **Overlays and Interaction (All in Dp)** | | | |
| `func (u *UI) PushOverlay(bounds Rect, drawFn func())` | | ✓ | Overlay bounds strictly in Dp (scaled to Px at boundary) |
| `func (u *UI) LayoutRect(rect Rect, fn func())` | | ✓ | Direct layout box strictly in Dp |
| `func (u *UI) Tooltip(text string, anchor Rect)` | | ✓ | Tooltip anchor strictly in Dp |
| `func (u *UI) Popover(id string, open *bool, anchor Rect, fn func(), opts ...Option)` | | ✓ | Popover anchor strictly in Dp |
| `func (u *UI) ContextMenu(id string, anchor Rect, items []ContextMenuItem)` | | ✓ | Context menu anchor strictly in Dp |
| `func (u *UI) RegisterHoverArea(id string, rect Rect, cursor CursorType, trackMove bool)` | | ✓ | Interactive hover area strictly in Dp |
| `func (u *UI) LastDirtyRectsPx() []Rect` | | ✓ | Window damage rectangles in device Px (do NOT pass to Dp overlay APIs) |
| **Testing (Headless)** | | | |
| `func (h *Headless) InjectMouseMove(xDp, yDp int)` | | ✓ | Mouse move event in logical Dp (auto-scaled by Scale) |
| `func (h *Headless) InjectMouseClick(xDp, yDp int, btn MouseButton)` | | ✓ | Full mouse click in logical Dp (auto-scaled by Scale) |
| `func (h *Headless) InjectMouseWheel(dxDp, dyDp int)` | | ✓ | Mouse wheel delta in logical steps (Dp) |
| `func (h *Headless) InjectMouseMovePx(xPx, yPx int)` | ✓ | | Raw mouse move event in physical device Px |
| `func (h *Headless) InjectMouseClickPx(xPx, yPx int, btn MouseButton)` | ✓ | | Raw mouse click event in physical device Px |
| `func (h *Headless) InjectMouseWheelPx(dxPx, dyPx int)` | ✓ | | Raw mouse wheel delta in physical device steps (Px) |
| **Styling Options** | | | |
| `Width(dp Dp) Option` | | ✓ | Fixed widget width |
| `Height(dp Dp) Option` | | ✓ | Fixed widget height |
| `MinWidth(dp Dp) Option` | | ✓ | Minimum widget width |
| `MaxWidth(dp Dp) Option` | | ✓ | Maximum widget width |
| `MinHeight(dp Dp) Option` | | ✓ | Minimum widget height |
| `MaxHeight(dp Dp) Option` | | ✓ | Maximum widget height |
| `FontSize(dp Dp) Option` | | ✓ | Widget text font size |
| `Padding(dp Dp) Option` | | ✓ | Widget internal padding |
| `Radius(dp float32) Option` | | ✓ | Corner radius in logical units (float32) |
| **WidgetStyle Struct** | | | |
| `WidgetStyle.WidthDp *Dp` | | ✓ | Width storage |
| `WidgetStyle.HeightDp *Dp` | | ✓ | Height storage |
| `WidgetStyle.MinWidthDp *Dp` | | ✓ | Minimum width storage |
| `WidgetStyle.MaxWidthDp *Dp` | | ✓ | Maximum width storage |
| `WidgetStyle.MinHeightDp *Dp` | | ✓ | Minimum height storage |
| `WidgetStyle.MaxHeightDp *Dp` | | ✓ | Maximum height storage |
| `WidgetStyle.FontSizeDp *Dp` | | ✓ | Font size storage |
| `WidgetStyle.PadDp *Dp` | | ✓ | Internal padding storage |
| `WidgetStyle.FlexWeight float32` | ✓ | | Kept intentionally: dimensionless proportion weight |
| **Layouts and Containers** | | | |
| `Column(gapDp Dp, fn func())` | | ✓ | Vertical column |
| `ColumnCentered(gapDp Dp, fn func())` | | ✓ | Centered vertical column |
| `Row(gapDp, heightDp Dp, fn func())` | | ✓ | Horizontal row |
| `RowCentered(gapDp, heightDp Dp, fn func())` | | ✓ | Centered horizontal row |
| `Padding(padDp Dp, fn func())` | | ✓ | Padding on all sides |
| `PaddingXY(pxDp, pyDp Dp, fn func())` | | ✓ | Separate horizontal and vertical padding |
| `Spacer(dp Dp)` | | ✓ | Fixed gap spacer |
| `FlexSpacer(reserveDp Dp)` | | ✓ | Spacer with reserved space |
| `RowFlex(gapDp, heightDp Dp, items ...FlexItem)` | | ✓ | Proportional row layout |
| `ColumnFlex(gapDp Dp, items ...FlexItem)` | | ✓ | Proportional column layout |
| `FlexItem.FixedDp Dp` | | ✓ | Fixed slot size in flex |
| `FlexItem.Weight float32` | ✓ | | Kept intentionally: dimensionless slot weight |
| `DockBottom(gapDp, bottomDp Dp, bodyFn, bottomFn func())` | | ✓ | Dock bottom panel |
| `DockTop(gapDp, topDp Dp, topFn, bodyFn func())` | | ✓ | Dock top panel |
| `RowWith(align Align, justify Justify, gapDp, heightDp Dp, items ...func())` | | ✓ | Aligned row |
| `ColumnWith(align Align, justify Justify, gapDp Dp, items ...func())` | | ✓ | Aligned column |
| `Grid(cols int, gapXDp, gapYDp, rowHeightDp Dp, fn func())` | | ✓ | Grid (cols int — kept intentionally, number of columns) |
| `FormLayout(labelWidthDp, gapXDp, gapYDp Dp, fn func())` | | ✓ | Form two-column layout |
| `Flow(gapXDp, gapYDp Dp, fn func())` | | ✓ | Flow wrap layout |
| `PageConfig.HeaderDp Dp` | | ✓ | Page header height |
| `PageConfig.FooterDp Dp` | | ✓ | Page footer height |
| `PageConfig.GapDp Dp` | | ✓ | Gap between page sections |
| **Widgets and Components** | | | |
| `Label(text string, sizeDp Dp, col Color)` | | ✓ | Text label |
| `DrawText(x, y int, text string, sizeDp Dp, col Color)` | | ✓ | Text at specific canvas coordinates |
| `DrawSVG(svgData string, x, y int, sizeDp Dp, tint Color)` | | ✓ | SVG icon with specified size |
| `Card(wDp, hDp, padDp Dp, fn func())` | | ✓ | Rounded card |
| `Button(id, label string, primary bool, widthDp Dp) bool` | | ✓ | Button with width |
| `Image(img image.Image, wDp, hDp Dp, mode ImageScaleMode)` | | ✓ | Image display |
| `CanvasView(wDp, hDp Dp, drawFn func(canvas *render.Canvas, areaPx Rect))` | | ✓ | Custom canvas drawing area (areaPx in device Px) |
| `InteractiveCanvas(id string, wDp, hDp Dp, cb CanvasCallbacks, opts ...Option)` | | ✓ | Interactive custom canvas |
| `ScrollView(id string, wDp, hDp Dp, fn func(), opts ...Option)` | | ✓ | Scrollable container |
| `ListView(id string, count int, itemHeightDp Dp, selected *int, drawItem func(idx int, areaPx Rect, sel bool), opts ...Option)` | | ✓ | Virtualized list (areaPx in device Px; count, selected — int indices) |
| `ListViewEx(id string, count int, itemHeightDp Dp, selected map[int]bool, checked []bool, drawItem func(idx int, areaPx Rect, sel bool), opts ...Option)` | | ✓ | Extended list (areaPx in device Px; count — int item count) |
| `TableColumn.WidthDp Dp` | | ✓ | Table column base width |
| `TableColumn.MinWidthDp Dp` | | ✓ | Table column min width |
| `StatusBarSection.WidthDp Dp` | | ✓ | Status bar section width |
| `StatusBarSection.Flex float32` | ✓ | | Kept intentionally: dimensionless flex factor |
| `TitleBarConfig.HeightDp Dp` | | ✓ | Custom title bar height |
| `SetToastOffset(xDp, yDp Dp)` | | ✓ | Toast notification offsets |
| `Spinner(id string, sizeDp Dp, opts ...Option)` | | ✓ | Loading spinner size |
| `Pagination(id string, currentPage *int, totalPages int, opts ...Option) bool` | ✓ | | Kept intentionally: currentPage, totalPages — page numbers |

