[English](../en/widgets.md) | [Русский](widgets.md)

# Справочник основных контролов (`anyui`)

Все контролы вызываются внутри `drawFn(ui *anyui.UI)` в Immediate Mode. Для настройки внешнего вида и поведения используются функциональные опции `anyui.Option` (см. [theming.md](theming.md)).

---

## Кнопки и переключатели

### `Button` и `IconButton`
- **Сигнатуры:**
  - `ui.Button(id, label string, opts ...Option) bool`
  - `ui.IconButton(id, svg string, opts ...Option) bool`
- **Опции:** `Primary()`, `Secondary()`, `Danger()`, `Ghost()`, `Bg(c)`, `HoverBg(c)`, `BorderColor(c)`, `HoverBorderColor(c)`, `TextColor(c)`, `HoverTextColor(c)`, `LetterSpacing(dp)`, `TextShadow(ox, oy, blur, c)`, `Icon(svg)`, `Width(dp)`, `Height(dp)`, `Radius(dp)`, `Disabled(bool)`. При полностью прозрачном `Bg` (`A == 0`) подложка при наведении и нажатии остаётся прозрачной (если не задан `HoverBg`).
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
- **Возможности:** точный клик курсора, выделение мышью, `Shift + Стрелки`, навигация/удаление по словам (`Ctrl + Left/Right`, `Ctrl + Shift + Left/Right`, `Ctrl + Backspace/Delete`), двойной клик (слово) и тройной клик (весь текст), `Ctrl+A / C / V / X`, история правок `Undo` / `Redo` (`Ctrl+Z` / `Ctrl+Y` / `Ctrl+Shift+Z`), встроенное контекстное меню по ПКМ.
- **Опции:** `Password()`, `ReadOnly(bool)`, `Clearable(bool)`, `Icon(svg)`, `IconRight(svg)`, `InputFilter(func(rune) bool)`, `Error(string)`, `FocusBorderColor(c)`, `TextAlignCenter()`, `TextAlignLeft()`, `TextAlignRight()`, `Disabled(bool)`, `Width(dp)`.
- **Пример:**
```go
ui.TextInput("pass", &password, "Введите пароль",
	anyui.Password(),
	anyui.Clearable(true),
	anyui.Width(280),
)
ui.TextInput("email", &email, "Email...",
	anyui.Width(260),
	anyui.Error(emailErr),
)
```

### `MaskedInput`
- **Сигнатура:** `ui.MaskedInput(id string, value *string, mask string, opts ...Option) bool`
- **Назначение:** однострочный ввод по шаблону маски (`0`/`9` — цифра, `A` — буква, `*` — буква или цифра, остальные символы — фиксированные разделители). Поддерживает `Undo`/`Redo`, контекстное меню по ПКМ и `Error(string)`.
- **Пример:**
```go
ui.MaskedInput("phone", &phone, "+7 (000) 000-00-00", anyui.Width(200), anyui.Mono())
ui.MaskedInput("date", &dateStr, "00.00.0000", anyui.Width(140), anyui.Mono())
ui.MaskedInput("ip", &ipStr, "000.000.000.000", anyui.Width(170), anyui.Mono())
```

### `TextArea`
- **Сигнатура:** `ui.TextArea(id string, value *string, placeholder string, opts ...Option) bool`
- **Возможности:** многострочный ввод с переносами строк, вертикальной прокруткой и автоскроллом за кареткой, навигацией `Up`/`Down`/`Home`/`End`/`Ctrl+Home`/`Ctrl+End`, навигацией по словам (`Ctrl + Стрелки`), выделением мышью по нескольким строкам, двойным/тройным кликом, `Undo`/`Redo`, опциональной нумерацией строк (`LineNumbers(true)`), контекстным меню по ПКМ и `Error(string)`.
- **Пример:**
```go
ui.TextArea("notes", &notes, "Комментарий...",
	anyui.Height(140),
	anyui.LineNumbers(true),
)
```

### `Label`, `SelectableLabel` и `Hyperlink`
- **Сигнатуры:**
  - `ui.Label(text string, opts ...Option)` — вывод текста с опциями `WordWrap(true)` (с аккуратным переносом по пробелам между словами), `Ellipsis(true)`, выравнивания (`TextAlignCenter()`, `TextAlignLeft()`, `TextAlignRight()`), `LetterSpacing(dp)` и `TextShadow(offsetX, offsetY, blur, col)`.
  - `ui.SelectableLabel(id, text string, opts ...Option)` — статический текст с выделением мышью (включая двойной/тройной клик), `Ctrl+C`/`Ctrl+A` и контекстным меню по ПКМ.
  - `ui.Hyperlink(id, text string, opts ...Option) bool` — кликабельная ссылка с подчёркиванием при наведении и курсором руки.
- **Пример:**
```go
ui.Label("Длинный текст с автопереносом по словам...", anyui.WordWrap(true))
ui.Label("Текст по центру карточки", anyui.WordWrap(true), anyui.TextAlignCenter(), anyui.FillWidth())
ui.Label("Заголовок, обрезаемый по ширине", anyui.Width(220), anyui.Ellipsis(true))
ui.SelectableLabel("session_id", "ID: 8f4c-2a19-b70e | IP: 10.24.18.5", anyui.Mono())
if ui.Hyperlink("docs_link", "Открыть документацию") {
	// обработка клика
}
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

## Дополнительные контролы

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
- **Позиционирование тостов:** `ui.SetToastPosition(pos)` (`ToastBottomRight`, `ToastBottomLeft`, `ToastTopRight`, `ToastTopLeft`, `ToastBottomCenter`, `ToastTopCenter`) и `ui.SetToastOffset(xDp, yDp)` (отступы от краёв окна в `dp`, по умолчанию `16, 16`).
- Всплывающие уведомления рисуются поверх интерфейса напрямую и не блокируют клики, фокус и мышь в других частях экрана.
```go
ui.Spinner("loader", 28)
ui.SetToastPosition(anyui.ToastBottomRight)
ui.SetToastOffset(16, 48) // приподнять над футером/статус-баром
ui.ShowToast("Готово", "Данные сохранены", anyui.ToastSuccess)
```

---

## Продвинутая система раскладки

### `RowFlex`, `ColumnFlex` и `Expand`
Пропорциональное деление доступного пространства по весам `FlexItem{Weight, FixedDp, Fn}`:
```go
ui.RowFlex(10, 38,
	anyui.FlexItem{FixedDp: 100, Fn: func() { ui.Button("b1", "Фикс") }},
	anyui.FlexItem{Weight: 1, Fn: func() { ui.TextInput("in1", &s1, "1x") }},
	anyui.FlexItem{Weight: 2, Fn: func() { ui.TextInput("in2", &s2, "2x") }},
)
ui.ColumnFlex(10,
	anyui.FlexItem{Weight: 1, Fn: func() { /* верхняя часть */ }},
	anyui.FlexItem{Weight: 2, Fn: func() { /* нижняя часть */ }},
)
ui.Expand(func() {
	// занимает всё оставшееся пространство в текущем контейнере
})
```

### `RowWith` и `ColumnWith`
Расширенные режимы выравнивания (`AlignStart`, `AlignCenter`, `AlignEnd`, `AlignStretch`) и распределения (`JustifyStart`, `JustifyCenter`, `JustifyEnd`, `JustifySpaceBetween`, `JustifySpaceAround`):
```go
ui.RowWith(anyui.AlignCenter, anyui.JustifySpaceBetween, 10, 40,
	func() { ui.Label("Заголовок", anyui.Bold()) },
	func() { ui.Button("act", "Действие", anyui.Primary()) },
)
```

### `Grid` и `FormLayout` / `FormRow`
Сетка из `cols` колонок и табличная вёрстка форм «Подпись — Поле» с единой шириной колонки меток:
```go
ui.Grid(3, 12, 12, 44, func() {
	// дочерние элементы автоматически распределяются по 3 колонкам
})

ui.FormLayout(120, 12, 8, func() {
	ui.FormRow("Хост:", func() { ui.TextInput("host", &host, "localhost") })
	ui.FormRow("Порт:", func() { ui.TextInput("port", &port, "8080") })
})
```

### `Flow` и авто-высота `Card`
Контейнер `Flow` автоматически переносит элементы на следующую строку при нехватке ширины, а `Card` с опцией `anyui.AutoHeight()` подбирает высоту по содержимому:
```go
ui.Card(func() {
	ui.Flow(8, 8, func() {
		ui.Badge("Go 1.24", anyui.Primary())
		ui.Badge("Win32")
		ui.Badge("Software Renderer")
	})
}, anyui.AutoHeight(), anyui.Padding(12))
```

## Расширенные события мыши, клавиатуры и фокуса

### Двойной клик (`DoubleClick`) и средняя кнопка мыши (`MiddleClick`)
- В `UI` и `ui.Context` доступны методы `MousePos() (int, int)`, `MousePressed() bool` (клик ЛКМ в текущем кадре), `MouseDown() bool` (зажата ли ЛКМ), `DoubleClicked() bool` и `MiddleClicked() bool` (в `ui.Context` — `MiddleMousePressed` / `MiddleMouseDown` / `MiddleMouseReleased`).
- В `Table` структура `TableEvent` возвращает поля `RowDoubleClicked` и `RowMiddleClicked` (`-1`, если события не было):
```go
ev := ui.Table("tbl", cols, len(rows), &sel, &sort, drawCell)
if ev.RowDoubleClicked >= 0 {
	// Открытие элемента по двойному клику
}
if ev.RowMiddleClicked >= 0 {
	// Действие по средней кнопке мыши
}
```
- В `TabBar` клик средней кнопкой мыши по вкладке с `Closable: true` закрывает её (`ev.Closed`).
- В `TreeView` двойной клик по узлу с дочерними элементами переключает его состояние `Expanded`.

### Горизонтальная прокрутка (`WheelX` и `Shift + WheelY`)
- Поддерживаются горизонтальное колесо мыши (`WM_MOUSEHWHEEL` -> `WheelX`) и вертикальное колесо с зажатым `Shift` (`Shift + WheelY`).
- `ScrollView` автоматически отображает горизонтальный скроллбар при переполнении ширины содержимого и прокручивается по горизонтали.
- `Table` поддерживает горизонтальный скролл колонок (колесом `WheelX` / `Shift + WheelY`, перетаскиванием горизонтального скроллбара и клавишами `Left`/`Right`).
- `TabBar` прокручивает вкладки как по `WheelX`, так и по `WheelY`.

### Захват мыши за пределами окна (`SetCapture` / `ReleaseCapture`)
- При нажатии кнопки мыши окно захватывает ввод (`SetCapture`) и освобождает его при отпускании (`ReleaseCapture`). При потере фокуса окна (`Alt+Tab`, `WM_KILLFOCUS`) генерируется `EventFocusLost` (сбрасывает `FocusID` в текстовых полях, `MouseDown` и `ActiveID`), а при отмене захвата (`WM_CAPTURECHANGED`, `WM_CANCELMODE`) — `EventCaptureLost` (сбрасывает `MouseDown` и `ActiveID`).
- Интервал двойного клика берётся из системной настройки Windows (`GetDoubleClickTime`).
- Перетаскивание `Slider`, `RangeSlider`, `SplitPane`, скроллбаров (`ScrollView`, `Table`, `TreeView`), ресайз колонок `Table`, перетаскивание вкладок `TabBar` и выделение текста не обрываются при выходе курсора за границы окна.

### Клавиатурная навигация в списках, таблицах, деревьях и селектах
Для `ListView`, `Table` и `TreeView` используется единый интерфейс `ui.Navigable` (`Count`, `SelectedIndex`, `PageSize`, `OnSelect`) и общий обработчик навигации. При наличии фокуса (`Tab` / `Shift+Tab` или клик по виджету) работают следующие клавиши:
- **`ListView`**: `Up`, `Down`, `PageUp`, `PageDown`, `Home`, `End`, `Space` (с автопрокруткой к выбранному элементу).
- **`Table`**: `Up`, `Down`, `PageUp`, `PageDown`, `Home`, `End`, `Left`, `Right` (горизонтальный скролл), `Space` / `Enter`.
- **`TreeView`**: `Up`, `Down`, `PageUp`, `PageDown`, `Home`, `End`, `Right` (раскрыть узел или перейти к дочернему), `Left` (свернуть узел или перейти к родителю), `Space` / `Enter` (переключить `Expanded`).
- **`Select` / `ComboBox`**: `Space` / `Enter` (открыть/закрыть список), `Up`, `Down`, `PageUp`, `PageDown`, `Home`, `End` (выбор элемента как в закрытом, так и в открытом состоянии).

---

## Десктоп-контролы, графики и мультивыбор

### `MenuBar`, `Toolbar`, `StatusBar`, `Breadcrumbs`, `Pagination`
```go
ui.MenuBar("mb", []anyui.MenuBarItem{
    {Label: "Файл", Items: []anyui.BarMenuItem{
        {Label: "Новый", Shortcut: "Ctrl+N", OnClick: func() {}},
        {Separator: true},
        {Label: "Экспорт", SubItems: []anyui.BarMenuItem{{Label: "PNG"}}},
    }},
})
ui.Toolbar("tb", []anyui.ToolbarItem{
    {ID: "save", Label: "Сохранить", Tooltip: "Сохранить файл", OnClick: func() {}},
})
ui.Breadcrumbs("bc", []anyui.BreadcrumbItem{{Label: "Главная"}, {Label: "Раздел"}})
ui.Pagination("pg", &page, 10)
ui.StatusBar("sb", []anyui.StatusBarSection{{Text: "Готово", Flex: 1}, {Text: "UTF-8", WidthDp: 70}})
```

### `LogView` и `VirtualTextView`
- `ui.LogView(id, entries, &followTail, opts...)`: виртуализированный просмотрщик логов с уровнями (`LogInfo`, `LogWarn`, `LogError`, `LogDebug`), встроенным фильтром, режимом `Follow tail`, выделением (`Клик`, `Shift+Клик`, `Ctrl+A`) и копированием (`Ctrl+C` или кнопка «Копировать»).
- `ui.VirtualTextView(id, lines, opts...)`: виртуализированное отображение массива строк.

### Графики (`Sparkline`, `LineChart`, `BarChart`)
- `ui.Sparkline(id, values, opts...)`: мини-график тренда с тултипом при наведении.
- `ui.LineChart(id, labels, series, opts...)` и `ui.BarChart(id, labels, series, opts...)`: графики с осями, сеткой, легендой и всплывающей подсказкой.

### Расширенный `DatePicker`, `TimePicker`, `DateTimePicker`
- `DatePicker`: клик по названию месяца или году в шапке открывает сетку 12 месяцев или 12 лет; кнопка «Сегодня» внизу календаря; двойной клик по полю включает ручной ввод даты (`YYYY-MM-DD`).
- `ui.TimePicker(id, &t, opts...)` и `ui.DateTimePicker(id, &dt, opts...)`: выбор времени `ЧЧ:ММ:СС` с кнопками `+`/`-` и колесом мыши.

### `MultiSelect` / `CheckComboBox` и `TagInput`
- `ui.MultiSelect(id, items, &selected, opts...)` (алиас `CheckComboBox`): выпадающий список с чекбоксами, поиском, кнопками «Выбрать все / Сбросить» и отображением тегов/сводки.
- `ui.TagInput(id, &tags, placeholder, opts...)`: ввод тегов (добавление по `Enter` или `,`, удаление по `x` или `Backspace`).

### Расширенный мультивыбор: `ListViewEx`, `TreeViewEx`, `TableEx`
- `ui.ListViewEx`: мультивыбор (`Ctrl+Клик`, `Shift+Клик`, `Ctrl+A`) и опциональные чекбоксы.
- `ui.TreeViewEx` + `anyui.SetTreeNodeChecked` / `anyui.SyncTreeCheckState`: мультивыбор узлов и каскадные чекбоксы с автоматическим `Indeterminate` у родителей.
- `ui.TableEx`: мультивыбор строк (`Ctrl+Клик`, `Shift+Клик`, `Ctrl+A`), колонка чекбоксов с «Выбрать все» в шапке, закреплённые колонки (`FrozenCols`), inline-редактирование ячейки по двойному клику и автоподбор ширины колонки (`AutoFitOnDoubleClick`).

---

## Кастомная шапка окна и локализация контролов

### `TitleBar` и `TitleBarHitTest`
Готовый виджет кастомной шапки для режима `Frameless: true` (одинаково выглядит на Windows 7/10/11/Server, поддерживает перетаскивание, двойной клик для разворачивания, иконку, пользовательские слоты и кнопки `Minimize` / `Maximize` / `Close`):

```go
var currentUI *anyui.UI

app := anyui.NewApp(anyui.Config{
	Title:     "Custom App",
	Frameless: true,
	HitTest: func(x, y int) anyui.HitTestResult {
		if currentUI != nil {
			return currentUI.TitleBarHitTest(x, y)
		}
		return anyui.HitClient
	},
})

app.Run(func(ui *anyui.UI) {
	currentUI = ui
	ui.TitleBar("main_title", "Custom App", anyui.TitleBarConfig{
		IconSVG: appIconSVG,
		LeftContent: func() {
			ui.Button("menu_btn", "Файл", anyui.Ghost(), anyui.Height(26))
		},
	})
})
```

### Локализация встроенных строк (`Strings`)
Все встроенные подписи (`DatePicker`, `TimePicker`, `MultiSelect`, `Select`, `LogView`, контекстное меню текстовых полей) настраиваются через `anyui.Strings` (`RussianStrings()`, `EnglishStrings()`):

```go
ui.SetStrings(anyui.EnglishStrings())
```

---

## Упрощение вёрстки и состояния (`IDScope`, `Card`, `Page`, `Remember`)

### Автоматические ID и `IDScope`
В любом виджете можно передать пустую строку `""` вместо ручного `id` — фреймворк сгенерирует его автоматически по счётчику в текущем скоупе. Для циклов и повторяющихся компонентов используйте `ui.IDScope(key, func() { ... })`:

```go
for _, item := range items {
	ui.IDScope(item.ID, func() {
		ui.Row(8, 36, func() {
			ui.Label(item.Name)
			if ui.Button("", "Удалить", anyui.Danger()) {
				deleteItem(item.ID)
			}
		})
	})
}
```

### `Card` и `Page`
- `ui.Card(fn, opts...)` по умолчанию подбирает высоту по содержимому (`AutoHeight`). Чтобы растянуть карточку на всю доступную высоту контейнера, передайте `anyui.FillHeight()`.
- `ui.Page(cfg, headerFn, bodyFn, footerFn)` строит типовой экран: фиксированная шапка сверху, прокручиваемое тело (`ScrollView`) в центре и фиксированный футер снизу.

```go
ui.Page(anyui.PageConfig{HeaderDp: 44, FooterDp: 48},
	func() { ui.Label("Настройки", anyui.FontSize(18), anyui.Bold()) },
	func() { /* длинная форма с автопрокруткой */ },
	func() { ui.Button("", "Сохранить", anyui.Primary()) },
)
```

### Кеширование фоновой загрузки (`Remember`)
`anyui.Remember(ui, key, loader)` запускает `loader` в фоне при первом обращении или смене ключа `key` и автоматически перерисовывает окно через `app.Dispatch`. Для сброса кеша вызовите `ui.InvalidateRemember(key)` или `ui.ClearRemember()`.
