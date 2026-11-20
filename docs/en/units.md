[English](units.md) | [Русский](../ru/units.md)

# Unit Typing Tracker (Dp / Px)

This document tracks the status of strict scalar typing in the AnyUI library according to Phase 18b.

## Units Contract
- **`Dp` (`type Dp int`)** — density-independent logical pixels. Used across the entire high-level public API for layouts, containers, options, and widgets.
- **`Px` (`type Px int`)** — device physical pixels of the render buffer. Used in low-level drawing (`render.Canvas`), low-level callbacks (`CanvasCallbacks`), system window metrics, and screen presentation.
- **DPI Scaling**: `Scale(dp Dp) Px` and `ScaleF(dp float32) float32` convert logical units to physical device pixels. Reverse conversion is performed via `PxToDp(px Px) Dp`.
- **No mixed overloads**: exactly one form per concept (parallel overloads like `Width(dp int)` alongside `WidthDp(dp Dp)` are prohibited).

---

## Signature Tracker

| Signature | int | Dp | Note |
|---|:---:|:---:|---|
| **Conversions and Coordinates** | | | |
| `func (u *UI) Scale(dp Dp) Px` | | ✓ | Converts logical pixels to physical device pixels |
| `func (u *UI) ScaleF(dp float32) float32` | | ✓ | Scales floating-point logical values |
| `func (u *UI) PxToDp(px Px) Dp` | | ✓ | Converts physical pixels to logical pixels |
| `func (u *UI) CurrentArea() Rect` | | ✓ | Current container bounds in Dp |
| `func (u *UI) CurrentAreaPx() Rect` | | ✓ | Current container bounds in physical Px |
| `func (u *UI) MousePosDp() (Dp, Dp)` | | ✓ | Window mouse cursor coordinates in Dp |
| `func (u *UI) MousePosPx() (int, int)` | ✓ | | Kept intentionally: OS mouse coordinates in physical pixels |
| `func Auto() Dp` / `const AutoSize Dp = -1` | | ✓ | Sentinel value for automatic sizing |
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
| `CanvasView(wDp, hDp Dp, drawFn func(canvas *render.Canvas, area Rect))` | | ✓ | Custom canvas drawing area |
| `InteractiveCanvas(id string, wDp, hDp Dp, cb CanvasCallbacks, opts ...Option)` | | ✓ | Interactive custom canvas |
| `ScrollView(id string, wDp, hDp Dp, fn func(), opts ...Option)` | | ✓ | Scrollable container |
| `ListView(id string, count int, itemHeightDp Dp, selected *int, drawItem func(...), opts ...Option)` | | ✓ | Virtualized list (count, selected — int indices) |
| `ListViewEx(id string, count int, itemHeightDp Dp, selected map[int]bool, checked []bool, drawItem func(...), opts ...Option)` | | ✓ | Extended list (count — int item count) |
| `TableColumn.WidthDp Dp` | | ✓ | Table column base width |
| `TableColumn.MinWidthDp Dp` | | ✓ | Table column min width |
| `StatusBarSection.WidthDp Dp` | | ✓ | Status bar section width |
| `StatusBarSection.Flex float32` | ✓ | | Kept intentionally: dimensionless flex factor |
| `TitleBarConfig.HeightDp Dp` | | ✓ | Custom title bar height |
| `SetToastOffset(xDp, yDp Dp)` | | ✓ | Toast notification offsets |
| `Spinner(id string, sizeDp Dp, opts ...Option)` | | ✓ | Loading spinner size |
| `Pagination(id string, currentPage *int, totalPages int, opts ...Option) bool` | ✓ | | Kept intentionally: currentPage, totalPages — page numbers |

