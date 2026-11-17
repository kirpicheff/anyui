# Темы и стилизация (`Option`)

## Предустановленные темы

В `anyui` доступны три готовые темы:
- `anyui.DeepNavyTheme()` — тёмно-синяя тема по умолчанию.
- `anyui.DarkTheme()` — нейтральная тёмная тема.
- `anyui.LightTheme()` — светлая тема.

Тему можно задать при создании приложения в `anyui.Config{Theme: ...}` или переключить на лету через `ui.SetTheme(theme)` / `win.SetTheme(theme)`.

## Создание своей темы

Структура `anyui.Theme` содержит все базовые цвета и радиусы:

```go
custom := anyui.DeepNavyTheme()
custom.Accent = anyui.Hex(0x10B981)
custom.AccentHover = anyui.Hex(0x059669)
custom.RadiusSm = 6
```

## Функциональные опции виджетов (`Option`)

Каждый виджет принимает вариативный список `opts ...anyui.Option` для локального переопределения внешнего вида и размеров:

| Опция | Назначение | Пример |
| --- | --- | --- |
| `anyui.Primary()` | Акцентный стиль кнопки | `ui.Button("ok", "ОК", anyui.Primary())` |
| `anyui.Secondary()` | Второстепенный стиль кнопки (по умолчанию) | `ui.Button("cancel", "Отмена", anyui.Secondary())` |
| `anyui.Danger()` | Стиль опасного действия | `ui.Button("del", "Удалить", anyui.Danger())` |
| `anyui.Ghost()` | Прозрачный фон в покое | `ui.Button("more", "...", anyui.Ghost())` |
| `anyui.Bg(c)` | Переопределение цвета фона | `anyui.Bg(anyui.Hex(0x1E293B))` |
| `anyui.TextColor(c)` | Переопределение цвета текста | `anyui.TextColor(ui.Theme().TextMuted)` |
| `anyui.BorderColor(c)` | Переопределение цвета рамки | `anyui.BorderColor(anyui.Hex(0x475569))` |
| `anyui.Radius(dp)` | Радиус скругления в `dp` | `anyui.Radius(10)` |
| `anyui.Width(dp)` | Фиксированная ширина в `dp` | `anyui.Width(220)` |
| `anyui.Height(dp)` | Фиксированная высота в `dp` | `anyui.Height(42)` |
| `anyui.FontSize(dp)` | Размер шрифта в `dp` | `anyui.FontSize(16)` |
| `anyui.Padding(dp)` | Внутренний отступ в `dp` | `anyui.Padding(12)` |
| `anyui.Icon(svg)` | SVG-иконка (рендеринг с Фазы 2) | `anyui.Icon(saveSVG)` |
| `anyui.Disabled(v)` | Блокировка ввода и приглушённый вид | `anyui.Disabled(isBusy)` |
| `anyui.MinWidth(dp)` | Минимальная ширина виджета в `dp` | `anyui.MinWidth(120)` |
| `anyui.MaxWidth(dp)` | Максимальная ширина виджета в `dp` | `anyui.MaxWidth(400)` |
| `anyui.MinHeight(dp)` | Минимальная высота виджета в `dp` | `anyui.MinHeight(80)` |
| `anyui.MaxHeight(dp)` | Максимальная высота виджета в `dp` | `anyui.MaxHeight(300)` |
| `anyui.FillWidth()` | Растянуть виджет на всю доступную ширину | `anyui.FillWidth()` |
| `anyui.FillHeight()` | Растянуть виджет на всю доступную высоту | `anyui.FillHeight()` |
| `anyui.AutoHeight()` | Автоматическая высота контейнера (`Card`) по содержимому | `anyui.AutoHeight()` |
| `anyui.Flex(weight)` | Пропорциональный вес виджета | `anyui.Flex(2)` |
| `anyui.WordWrap(v)` | Перенос текста по ширине (`Label`, `SelectableLabel`) | `anyui.WordWrap(true)` |
| `anyui.Ellipsis(v)` | Обрезка текста с `...` при нехватке ширины (`Label`) | `anyui.Ellipsis(true)` |
| `anyui.LineNumbers(v)` | Нумерация строк в `TextArea` | `anyui.LineNumbers(true)` |
| `anyui.Error(text)` | Состояние ошибки валидации и подпись под полем ввода | `anyui.Error("Неверный формат")` |

