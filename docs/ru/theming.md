[English](../en/theming.md) | [Русский](theming.md)

# Темы и стилизация (`Option`)

## Предустановленные темы

В `anyui` доступны готовые темы:
- `anyui.NavyTheme()` — тёмно-синяя тема по умолчанию.
- `anyui.DarkTheme()` — нейтральная тёмная тема.
- `anyui.LightTheme()` — светлая тема.
- `anyui.SystemTheme()` — автоматический выбор `DarkTheme()` или `LightTheme()` по текущей теме ОС.

Тему можно задать при создании приложения в `anyui.Config{Theme: ...}` или переключить на лету через `ui.SetTheme(theme)` / `win.SetTheme(theme)`.

## Создание своей темы

Структура `anyui.Theme` содержит все базовые цвета и радиусы:

```go
custom := anyui.NavyTheme()
custom.Accent = anyui.Hex(0x10B981)
custom.AccentHover = anyui.Hex(0x059669)
custom.RadiusSm = 6
```

## Функциональные опции виджетов (`Option`)

Каждый виджет принимает вариативный список `opts ...anyui.Option` для локального переопределения внешнего вида и размеров.

> [!NOTE]
> Булевы модификаторы (`Bold`, `Primary`, `Ghost`, `Mono`, `Disabled`, `ReadOnly`, `Clearable`, `Indeterminate`, `WordWrap`) имеют сигнатуру `(enabled ...bool)` со значением `true` по умолчанию. Их можно вызывать как без аргументов (`anyui.Bold()`), так и передавать булево условие напрямую (`anyui.Disabled(isBusy)`).

| Опция | Назначение | Пример |
| --- | --- | --- |
| `anyui.Primary(enabled ...bool)` | Акцентный стиль кнопки | `ui.Button("ok", "ОК", anyui.Primary())` |
| `anyui.Secondary(enabled ...bool)` | Второстепенный стиль кнопки (по умолчанию) | `ui.Button("cancel", "Отмена", anyui.Secondary())` |
| `anyui.Danger(enabled ...bool)` | Стиль опасного действия | `ui.Button("del", "Удалить", anyui.Danger())` |
| `anyui.Ghost(enabled ...bool)` | Прозрачный фон в покое | `ui.Button("more", "...", anyui.Ghost())` |
| `anyui.Bold(enabled ...bool)` | Полужирное начертание шрифта | `ui.Label("Заголовок", anyui.Bold())` |
| `anyui.Mono(enabled ...bool)` | Моноширинный шрифт | `ui.Label("127.0.0.1", anyui.Mono())` |
| `anyui.Bg(c)` | Переопределение цвета фона | `anyui.Bg(anyui.Hex(0x1E293B))` |
| `anyui.HoverBg(c)` | Цвет фона при наведении (`Button`) | `anyui.HoverBg(anyui.Hex(0x334155))` |
| `anyui.TextColor(c)` | Переопределение цвета текста | `anyui.TextColor(ui.Theme().TextMuted)` |
| `anyui.HoverTextColor(c)` | Цвет текста и иконки при наведении (`Button`) | `anyui.HoverTextColor(ui.Theme().Accent)` |
| `anyui.BorderColor(c)` | Переопределение цвета рамки | `anyui.BorderColor(anyui.Hex(0x475569))` |
| `anyui.HoverBorderColor(c)` | Цвет рамки при наведении (`Button`) | `anyui.HoverBorderColor(ui.Theme().Accent)` |
| `anyui.FocusBorderColor(c)` | Цвет рамки при фокусе (`TextInput`, `MaskedInput`, `TextArea`) | `anyui.FocusBorderColor(ui.Theme().Accent)` |
| `anyui.Radius(dp)` | Радиус скругления в `dp` | `anyui.Radius(10)` |
| `anyui.Width(dp)` | Фиксированная ширина в `dp` | `anyui.Width(220)` |
| `anyui.Height(dp)` | Фиксированная высота в `dp` | `anyui.Height(42)` |
| `anyui.FontSize(dp)` | Размер шрифта в `dp` | `anyui.FontSize(16)` |
| `anyui.LetterSpacing(dp)` | Межбуквенный интервал в `dp` | `anyui.LetterSpacing(1.5)` |
| `anyui.TextShadow(dx, dy, blur, c)` | Тень или свечение текста | `anyui.TextShadow(0, 1, 4, col)` |
| `anyui.TextAlignLeft/Center/Right(enabled ...bool)` | Выравнивание текста (`Label`, `TextInput`) | `anyui.TextAlignCenter()` |
| `anyui.Padding(dp)` | Внутренний отступ в `dp` | `anyui.Padding(12)` |
| `anyui.Icon(svg)` | SVG-иконка слева | `anyui.Icon(saveSVG)` |
| `anyui.IconRight(svg)` | SVG-иконка справа (`TextInput`) | `anyui.IconRight(searchSVG)` |
| `anyui.Disabled(enabled ...bool)` | Блокировка ввода и приглушённый вид | `anyui.Disabled(isBusy)` |
| `anyui.ReadOnly(enabled ...bool)` | Запрет редактирования с сохранением выделения | `anyui.ReadOnly()` |
| `anyui.Password(enabled ...bool)` | Маскировка ввода пароля (`TextInput`) | `anyui.Password()` |
| `anyui.PasswordToggle(visible ...*bool)` | Режим пароля с кнопкой-глазиком (`TextInput`) | `anyui.PasswordToggle()` |
| `anyui.Clearable(enabled ...bool)` | Кнопка быстрой очистки поля ввода | `anyui.Clearable()` |
| `anyui.InputFilter(fn)` | Фильтр допустимых символов (`TextInput`, `TextArea`) | `anyui.InputFilter(unicode.IsDigit)` |
| `anyui.Searchable(enabled ...bool)` | Строка поиска в `Select` | `anyui.Searchable()` |
| `anyui.Indeterminate(enabled ...bool)` | Промежуточное состояние чекбокса | `anyui.Indeterminate()` |
| `anyui.MinWidth(dp)` | Минимальная ширина виджета в `dp` | `anyui.MinWidth(120)` |
| `anyui.MaxWidth(dp)` | Максимальная ширина виджета в `dp` | `anyui.MaxWidth(400)` |
| `anyui.MinHeight(dp)` | Минимальная высота виджета в `dp` | `anyui.MinHeight(80)` |
| `anyui.MaxHeight(dp)` | Максимальная высота виджета в `dp` | `anyui.MaxHeight(300)` |
| `anyui.FillWidth(enabled ...bool)` | Растянуть виджет на всю доступную ширину | `anyui.FillWidth()` |
| `anyui.FillHeight(enabled ...bool)` | Растянуть виджет на всю доступную высоту | `anyui.FillHeight()` |
| `anyui.AutoHeight(enabled ...bool)` | Автоматическая высота контейнера (`Card`) по содержимому | `anyui.AutoHeight()` |
| `anyui.WordWrap(enabled ...bool)` | Перенос текста по ширине (`Label`, `SelectableLabel`) | `anyui.WordWrap()` |
| `anyui.Ellipsis(enabled ...bool)` | Обрезка текста с `...` при нехватке ширины (`Label`) | `anyui.Ellipsis()` |
| `anyui.LineNumbers(enabled ...bool)` | Нумерация строк в `TextArea` | `anyui.LineNumbers()` |
| `anyui.Error(text)` | Состояние ошибки валидации и подпись под полем ввода | `anyui.Error("Неверный формат")` |
| `anyui.ScrollToIndex(idx)` | Программная прокрутка к строке (`ListView*`, `Table*`) | `anyui.ScrollToIndex(targetIdx)` |
| `anyui.ScrollToID(id)` | Программная прокрутка к якорю (`ScrollView`) | `anyui.ScrollToID("sec_2")` |
| `anyui.Reorderable(enabled ...bool)` | Включить перетаскивание элементов (`ListView*`, `Table*`, `TreeView*`) | `anyui.Reorderable()` |
| `anyui.AllowReparent(enabled ...bool)` | Разрешить смену родителя и `DropInside` в `TreeView*` | `anyui.AllowReparent()` |
| `anyui.OnReorder(fn)` | Обработчик события перестановки `DragEvent` | `anyui.OnReorder(func(ev anyui.DragEvent) { ... })` |
| `anyui.DragSource(typeName, fn)` | Источник кросс-виджетного переноса (`DragPayload`) | `anyui.DragSource("node", fn)` |
| `anyui.AcceptDrop(types...)` | Разрешить приём внешнего переноса указанных типов | `anyui.AcceptDrop("node")` |
| `anyui.OnDrop(fn)` | Обработчик внешнего кросс-виджетного сброса `DragEvent` | `anyui.OnDrop(func(ev anyui.DragEvent) { ... })` |

