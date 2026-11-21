[English](../en/units.md) | [Русский](units.md)

# Трекер типизации единиц измерения (Dp / Px)

Данный документ фиксирует статус строгой скалярной типизации в библиотеке AnyUI согласно Фазе 18b (синхронизировано с [docs/en/units.md](../en/units.md)).

## Контракт единиц измерения
- **`Dp` (`type Dp int`)** — логические аппаратно-независимые пиксели (Density-independent Pixels). Используются во всем высокоуровневом публичном API раскладки, контейнеров, опций, оверлеев и виджетов:
  - Все публичные методы раскладки и оверлеев принимают геометрию строго в `Dp`: `LayoutRect`, `Tooltip`, `PushOverlay`, `Popover`, `ContextMenu`, `RegisterHoverArea`. Прямоугольники из `ui.CurrentArea()` имеют единицы `Dp` и могут передаваться напрямую без масштабирования.
- **`Px` (`type Px int`)** — физические пиксели буфера отрисовки (Device Pixels). Сохранены исключительно для низкоуровневого рисования (`render.Canvas`), низкоуровневых колбэков холста и таблиц (`CanvasView` area, `Table.drawCell` area), системных метрик окна и вывода на экран (`LastDirtyRects`, `Config.HitTest`, `TitleBarHitTest`).
- **Ориентация**: константы `Horizontal` и `Vertical` являются единственными допустимыми значениями типа `Orientation`. Устаревший `SplitOrientation` полностью удалён.
- **Правило масштабирования и конвертации**:
  - `Scale(dp Dp) Px` преобразует логические единицы в физические пиксели (частый hot-path в прикладном коде).
  - `ScaleF(dp float32) float32` масштабирует вещественные логические величины.
  - `PxToDp(px Px) Dp` преобразует физические пиксели в логические (обратная проекция для колбэков Canvas, hit-test).
  - `PxToDpf(px float32) float32` обратная проекция для вещественных значений.
  - **Правило**: Допустима только каноническая четвёрка `Scale` / `ScaleF` / `PxToDp` / `PxToDpf`. Введение имён вроде `DpToPx` или `DpToPxf` строго запрещено.
  - **Конвенция суффикса `F`**: суффикс `F` строго обозначает `float32`-вариант той же операции.
- **Sentinel-значение**: `const Auto Dp = -1` является единым каноническим sentinel-значением автоматического размера. Функции вроде `Auto()` и константы вроде `AutoSize` удалены.
- **Запрет на смешанные перегрузки**: ровно одна форма на одну концепцию (запрещено создавать параллельные `Width(dp int)` и `WidthDp(dp Dp)`).

---

## Трекер сигнатур

| Сигнатура | int | Dp | Примечание |
|---|:---:|:---:|---|
| **Преобразования и координаты** | | | |
| `func (u *UI) Scale(dp Dp) Px` | | ✓ | Перевод логических пикселей в физические |
| `func (u *UI) ScaleF(dp float32) float32` | | ✓ | Масштабирование вещественных логических величин |
| `func (u *UI) PxToDp(px Px) Dp` | | ✓ | Перевод физических пикселей в логические |
| `func (u *UI) PxToDpf(px float32) float32` | | ✓ | Обратная проекция вещественных физических пикселей |
| `func (u *UI) CurrentArea() Rect` | | ✓ | Контракт области текущего контейнера в Dp |
| `func (u *UI) CurrentAreaPx() Rect` | | ✓ | Область текущего контейнера в физических Px |
| `func (u *UI) MousePosDp() (Dp, Dp)` | | ✓ | Координаты курсора мыши в окне в Dp |
| `func (u *UI) MousePosPx() (Px, Px)` | | ✓ | Координаты курсора мыши в физических пикселях буфера |
| `const Auto Dp = -1` | | ✓ | Каноническое sentinel-значение автоматического размера |
| **Оверлеи и интерактивные зоны (все строго в Dp)** | | | |
| `func (u *UI) PushOverlay(bounds Rect, drawFn func())` | | ✓ | Границы оверлея в Dp (масштабируются в Px на границе) |
| `func (u *UI) LayoutRect(rect Rect, fn func())` | | ✓ | Область явной раскладки строго в Dp |
| `func (u *UI) Tooltip(text string, anchor Rect)` | | ✓ | Якорь всплывающей подсказки строго в Dp |
| `func (u *UI) Popover(id string, open *bool, anchor Rect, fn func(), opts ...Option)` | | ✓ | Якорь поповера строго в Dp |
| `func (u *UI) ContextMenu(id string, anchor Rect, items []ContextMenuItem)` | | ✓ | Якорь контекстного меню строго в Dp |
| `func (u *UI) RegisterHoverArea(id string, rect Rect, cursor CursorType, trackMove bool)` | | ✓ | Интерактивная зона курсора строго в Dp |
| `func (u *UI) LastDirtyRectsPx() []Rect` | | ✓ | Изменённые области окна в физических Px (не передавать в методы оверлеев Dp) |
| **Тестирование (Headless)** | | | |
| `func (h *Headless) InjectMouseMove(xDp, yDp int)` | | ✓ | Перемещение мыши в логических Dp (масштабируется по Scale) |
| `func (h *Headless) InjectMouseClick(xDp, yDp int, btn MouseButton)` | | ✓ | Полный клик мыши в логических Dp (масштабируется по Scale) |
| `func (h *Headless) InjectMouseWheel(dxDp, dyDp int)` | | ✓ | Прокрутка колеса мыши в логических шагах (Dp) |
| `func (h *Headless) InjectMouseMovePx(xPx, yPx int)` | ✓ | | Прямое перемещение мыши в физических пикселях Px |
| `func (h *Headless) InjectMouseClickPx(xPx, yPx int, btn MouseButton)` | ✓ | | Прямой клик мыши в физических пикселях Px |
| `func (h *Headless) InjectMouseWheelPx(dxPx, dyPx int)` | ✓ | | Прямая прокрутка колеса мыши в физических шагах (Px) |
| **Опции стилизации (Options)** | | | |
| `Width(dp Dp) Option` | | ✓ | Фиксированная ширина виджета |
| `Height(dp Dp) Option` | | ✓ | Фиксированная высота виджета |
| `MinWidth(dp Dp) Option` | | ✓ | Минимальная ширина виджета |
| `MaxWidth(dp Dp) Option` | | ✓ | Максимальная ширина виджета |
| `MinHeight(dp Dp) Option` | | ✓ | Минимальная высота виджета |
| `MaxHeight(dp Dp) Option` | | ✓ | Максимальная высота виджета |
| `FontSize(dp Dp) Option` | | ✓ | Размер шрифта текста виджета |
| `Padding(dp Dp) Option` | | ✓ | Внутренний отступ виджета |
| `Radius(dp float32) Option` | | ✓ | Радиус скругления в логических единицах float32 |
| **Структура WidgetStyle** | | | |
| `WidgetStyle.WidthDp *Dp` | | ✓ | Хранение ширины |
| `WidgetStyle.HeightDp *Dp` | | ✓ | Хранение высоты |
| `WidgetStyle.MinWidthDp *Dp` | | ✓ | Хранение минимальной ширины |
| `WidgetStyle.MaxWidthDp *Dp` | | ✓ | Хранение максимальной ширины |
| `WidgetStyle.MinHeightDp *Dp` | | ✓ | Хранение минимальной высоты |
| `WidgetStyle.MaxHeightDp *Dp` | | ✓ | Хранение максимальной высоты |
| `WidgetStyle.FontSizeDp *Dp` | | ✓ | Хранение размера шрифта |
| `WidgetStyle.PadDp *Dp` | | ✓ | Хранение внутреннего отступа |
| `WidgetStyle.FlexWeight float32` | ✓ | | Оставлено осознанно: безразмерный коэффициент пропорции |
| **Макеты и контейнеры (Layout)** | | | |
| `Column(gapDp Dp, fn func())` | | ✓ | Вертикальная колонка |
| `ColumnCentered(gapDp Dp, fn func())` | | ✓ | Вертикальная центрированная колонка |
| `Row(gapDp, heightDp Dp, fn func())` | | ✓ | Горизонтальная строка |
| `RowCentered(gapDp, heightDp Dp, fn func())` | | ✓ | Горизонтальная центрированная строка |
| `Padding(padDp Dp, fn func())` | | ✓ | Отступ со всех сторон |
| `PaddingXY(pxDp, pyDp Dp, fn func())` | | ✓ | Раздельные отступы по осям |
| `Spacer(dp Dp)` | | ✓ | Отступ-распорка |
| `FlexSpacer(reserveDp Dp)` | | ✓ | Распорка с резервом места |
| `RowFlex(gapDp, heightDp Dp, items ...FlexItem)` | | ✓ | Пропорциональная строка |
| `ColumnFlex(gapDp Dp, items ...FlexItem)` | | ✓ | Пропорциональная колонка |
| `FlexItem.FixedDp Dp` | | ✓ | Фиксированный размер слота flex |
| `FlexItem.Weight float32` | ✓ | | Оставлено осознанно: безразмерный вес слота |
| `DockBottom(gapDp, bottomDp Dp, bodyFn, bottomFn func())` | | ✓ | Закрепление нижней панели |
| `DockTop(gapDp, topDp Dp, topFn, bodyFn func())` | | ✓ | Закрепление верхней панели |
| `RowWith(align Align, justify Justify, gapDp, heightDp Dp, items ...func())` | | ✓ | Строка с выравниванием |
| `ColumnWith(align Align, justify Justify, gapDp Dp, items ...func())` | | ✓ | Колонка с выравниванием |
| `Grid(cols int, gapXDp, gapYDp, rowHeightDp Dp, fn func())` | | ✓ | Сетка (cols int — оставлено осознанно, число колонок) |
| `FormLayout(labelWidthDp, gapXDp, gapYDp Dp, fn func())` | | ✓ | Разметка формы |
| `Flow(gapXDp, gapYDp Dp, fn func())` | | ✓ | Потоковая разметка с переносом |
| `PageConfig.HeaderDp Dp` | | ✓ | Высота шапки страницы |
| `PageConfig.FooterDp Dp` | | ✓ | Высота футера страницы |
| `PageConfig.GapDp Dp` | | ✓ | Отступ между секциями страницы |
| **Виджеты и компоненты** | | | |
| `Label(text string, sizeDp Dp, col Color)` | | ✓ | Текстовая метка |
| `DrawText(x, y int, text string, sizeDp Dp, col Color)` | | ✓ | Текст в произвольных координатах |
| `DrawSVG(svgData string, x, y int, sizeDp Dp, tint Color)` | | ✓ | SVG-иконка заданного размера |
| `Card(wDp, hDp, padDp Dp, fn func())` | | ✓ | Скруглённая карточка |
| `Button(id, label string, primary bool, widthDp Dp) bool` | | ✓ | Кнопка с шириной |
| `Image(img image.Image, wDp, hDp Dp, mode ImageScaleMode)` | | ✓ | Вывод изображения |
| `CanvasView(wDp, hDp Dp, drawFn func(canvas *render.Canvas, areaPx Rect))` | | ✓ | Область кастомного рисования (areaPx в физических Px) |
| `InteractiveCanvas(id string, wDp, hDp Dp, cb CanvasCallbacks, opts ...Option)` | | ✓ | Интерактивный холст |
| `ScrollView(id string, wDp, hDp Dp, fn func(), opts ...Option)` | | ✓ | Прокручиваемый контейнер |
| `ListView(id string, count int, itemHeightDp Dp, selected *int, drawItem func(idx int, areaPx Rect, sel bool), opts ...Option)` | | ✓ | Виртуализированный список (areaPx в Px буфера; count, selected — int индексы) |
| `ListViewEx(id string, count int, itemHeightDp Dp, selected map[int]bool, checked []bool, drawItem func(idx int, areaPx Rect, sel bool), opts ...Option)` | | ✓ | Расширенный список (areaPx в Px буфера; count — int число элементов) |
| `TableColumn.WidthDp Dp` | | ✓ | Базовая ширина колонки таблицы |
| `TableColumn.MinWidthDp Dp` | | ✓ | Минимальная ширина колонки таблицы |
| `StatusBarSection.WidthDp Dp` | | ✓ | Ширина секции строки состояния |
| `StatusBarSection.Flex float32` | ✓ | | Оставлено осознанно: безразмерный коэффициент flex |
| `TitleBarConfig.HeightDp Dp` | | ✓ | Высота кастомного заголовка окна |
| `SetToastOffset(xDp, yDp Dp)` | | ✓ | Отступы всплывающих уведомлений |
| `Spinner(id string, sizeDp Dp, opts ...Option)` | | ✓ | Размер индикатора загрузки |
| `Pagination(id string, currentPage *int, totalPages int, opts ...Option) bool` | ✓ | | Оставлено осознанно: currentPage, totalPages — номера страниц |

