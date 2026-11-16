# Справочник основных контролов (`anyui`)

Все контролы вызываются внутри `drawFn(ui *anyui.UI)` в Immediate Mode. Для настройки внешнего вида и поведения используются функциональные опции `anyui.Option` (см. [theming.md](theming.md)).

---

## Кнопки и переключатели

### `Button` и `IconButton`
- **Сигнатуры:**
  - `ui.Button(id, label string, opts ...Option) bool`
  - `ui.IconButton(id, svg string, opts ...Option) bool`
- **Опции:** `Primary()`, `Secondary()`, `Danger()`, `Ghost()`, `Icon(svg)`, `Width(dp)`, `Height(dp)`, `Radius(dp)`, `Disabled(bool)`.
- **Пример:**
```go
if ui.Button("save", "Сохранить", anyui.Primary(), anyui.Icon(saveSVG)) {
	// обработка клика
}
if ui.IconButton("close", closeSVG, anyui.Ghost()) {
	// клик по иконке
}
```

### `Checkbox`
- **Сигнатура:** `ui.Checkbox(id, label string, checked *bool, opts ...Option) bool`
- **Опции:** `Indeterminate(bool)`, `Disabled(bool)`, `FontSize(dp)`, `TextColor(c)`, `Radius(dp)`.
- **Пример:**
```go
ui.Checkbox("agree", "Принимаю условия", &agreed)
```

### `RadioButton` и `RadioGroup`
- **Сигнатуры:**
  - `ui.RadioButton(id, label string, selected bool, opts ...Option) bool`
  - `ui.RadioGroup(id string, options []string, selected *int, opts ...Option) bool`
- **Пример:**
```go
ui.RadioGroup("theme", []string{"Deep Navy", "Dark", "Light"}, &themeIdx)
```

### `Toggle`
- **Сигнатура:** `ui.Toggle(id, label string, value *bool, opts ...Option) bool`
- **Пример:**
```go
ui.Toggle("autostart", "Запускать при старте ОС", &autoStart)
```

---

## Ввод текста

### `TextInput`
- **Сигнатура:** `ui.TextInput(id string, value *string, placeholder string, opts ...Option) bool`
- **Возможности:** точный клик курсора, выделение мышью и `Shift + Стрелки`, `Ctrl+A / C / V / X`.
- **Опции:** `Password()`, `ReadOnly(bool)`, `Clearable(bool)`, `Icon(svg)`, `IconRight(svg)`, `InputFilter(func(rune) bool)`, `Disabled(bool)`, `Width(dp)`.
- **Пример:**
```go
ui.TextInput("pass", &password, "Введите пароль",
	anyui.Password(),
	anyui.Clearable(true),
	anyui.Width(280),
)
```

### `TextArea`
- **Сигнатура:** `ui.TextArea(id string, value *string, placeholder string, opts ...Option) bool`
- **Возможности:** многострочный ввод с переносами строк и вертикальной прокруткой колесом мыши.
- **Пример:**
```go
ui.TextArea("notes", &notes, "Комментарий...", anyui.Height(140))
```

---

## Списки и навигация

### `Select` и `ComboBox`
- **Сигнатуры:**
  - `ui.Select(id string, items []string, selected *int, opts ...Option) bool`
  - `ui.ComboBox(id string, items []string, selected *int, opts ...Option) bool` (с полем фильтрации/поиска)
- **Пример:**
```go
ui.ComboBox("country", countries, &countryIdx, anyui.Width(260))
```

### `ListView`
- **Сигнатура:** `ui.ListView(id string, count int, itemHeightDp int, selected *int, drawItem func(index int, area Rect, selected bool), opts ...Option) bool`
- **Возможности:** виртуализация (рисуются только видимые строки), прокрутка колесом мыши.
- **Пример:**
```go
ui.ListView("files", len(files), 36, &selectedIdx, func(i int, area anyui.Rect, sel bool) {
	// отрисовка строки i внутри area
}, anyui.Height(240))
```

### `Tabs` и `TabBar`
- **Сигнатуры:**
  - `ui.Tabs(id string, items []string, active *int, vertical bool, opts ...Option) bool`
  - `ui.TabBar(id string, tabs []TabItem, active *int, opts ...Option) TabBarEvent`
- **Пример `TabBar`:**
```go
ev := ui.TabBar("editor_tabs", tabs, &activeTab)
if ev.Added {
	// добавить новую вкладку
}
if ev.Closed >= 0 {
	// закрыть вкладку ev.Closed
}
```

---

## Контейнеры, индикаторы и оверлеи

### `ScrollView`
- **Сигнатура:** `ui.ScrollView(id string, wDp, hDp int, fn func(), opts ...Option)`
- **Возможности:** вертикальная прокрутка колесом мыши и перетаскиванием скроллбара.

### `Modal`, `Popover`, `FloatingPanel`, `Drawer`
- `ui.Modal(id string, open *bool, title string, fn func(), opts ...Option)` — модальное окно с затемнением и блокировкой кликов.
- `ui.Popover(id string, open *bool, anchor Rect, fn func(), opts ...Option)` — выпадающая панель у элемента-якоря.
- `ui.FloatingPanel(id, title string, open *bool, fn func(), opts ...Option)` — перетаскиваемая сворачиваемая панель.
- `ui.Drawer(id string, open *bool, side DrawerSide, fn func(), opts ...Option)` — выезжающая панель (`DrawerLeft`, `DrawerRight`).

### `ContextMenu` и `Tooltip`
- `ui.ContextMenu(id string, anchor Rect, items []ContextMenuItem)` — меню по правому клику.
- `ui.Tooltip(text string, anchor Rect)` — всплывающая подсказка при наведении.

### `Separator`, `Badge`, `StatusDot`, `ProgressBar`
```go
ui.Separator()
ui.Badge("BETA", anyui.Primary())
ui.StatusDot(theme.Success, "Подключено")
ui.ProgressBar(0.65, anyui.Width(240))
```


---

## Дополнительные контролы (Фаза 5)

### `Table`
Таблица с заголовками колонок, ресайзом ширины колонок, сортировкой и виртуализацией строк.
```go
cols := []anyui.TableColumn{
	{Title: "ID", WidthDp: 64, MinWidthDp: 48, Sortable: true},
	{Title: "Имя", WidthDp: 200, Sortable: true},
}
ev := ui.Table("tbl", cols, len(rows), &selectedRow, &sortState, func(row, col int, area anyui.Rect, sel bool) {
	ui.DrawText(area.X+ui.S(8), area.Y+ui.S(7), rows[row].Name, 12, ui.Theme().TextPrimary)
}, anyui.Height(240))
```

### `TreeView`
Древовидный список с раскрытием узлов и виртуализацией.
```go
ui.TreeView("tree", roots, &selectedNodeID, anyui.Height(220))
```

### `SplitPane`
Разделение области на 2 панели (`SplitHorizontal` или `SplitVertical`) с перетаскиваемым разделителем:
```go
ui.SplitPane("split", anyui.SplitHorizontal, &ratio, func() {
	// Первая панель
}, func() {
	// Вторая панель
}, anyui.Height(260))
```

### `Collapsible` и `Accordion`
Сворачиваемые секции:
```go
ui.Collapsible("sec1", "Дополнительно", &expanded, func() {
	ui.Label("Содержимое секции")
})
ui.Accordion("acc", items, &openIndex)
```

### Расширение `TabBar` (Drag & Detach)
`TabBarEvent` возвращает `MovedFrom`, `MovedTo` при перетаскивании вкладок и `Detached` при вытягивании вкладки за пределы панели (для открытия в отдельном окне через `app.OpenWindow`).

### `NumberInput`, `Slider`, `RangeSlider`
```go
ui.NumberInput("qty", &qty, 0, 100, 1, 0, anyui.Width(150))
ui.Slider("vol", &vol, 0, 100, anyui.Width(240))
ui.RangeSlider("rng", &low, &high, 0, 100, anyui.Width(240))
```

### `ColorPicker` и `DatePicker`
```go
ui.ColorPicker("cp", &colorVal, anyui.Width(150))
ui.DatePicker("dp", &dateVal, anyui.Width(170))
```

### `Toast` и `Spinner`
```go
ui.Spinner("loader", 28)
ui.ShowToast("Готово", "Данные сохранены", anyui.ToastSuccess)
```
