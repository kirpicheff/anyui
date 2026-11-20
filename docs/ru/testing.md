[English](../en/testing.md) | [Русский](testing.md)

# Тестирование и Headless-рантайм (`anyui`)

Пакет `anyui` предоставляет встроенный headless-рантайм для модульного, интеграционного и визуального регрессионного тестирования интерфейса без создания физических окон ОС и без обращения к оконному менеджеру (WinAPI, X11, Wayland, Cocoa).

---

## Ключевые возможности

- **100% изоляция от ОС:** тесты выполняются в чистом Go без оконных серверов и без CGO, в том числе в headless-окружениях Linux CI (GitHub Actions, GitLab CI, Docker).
- **Детерминированные шрифты:** обязательное указание встроенных шрифтов (`anyui.DefaultTestFonts()`) исключает расхождения в рендеринге текста на разных платформах.
- **Пошаговое управление кадрами (`h.Step()`):** синхронная отрисовка ровно одного кадра по требованию теста с гарантированной обработкой очереди `app.Dispatch()`.
- **Виртуальный системный клок:** управление виртуальным временем (`h.AdvanceTime()`, `h.SetTime()`) позволяет мгновенно тестировать анимации и таймеры (например, время жизни `ShowToast`) без нестабильных задержек и `time.Sleep`.
- **Синтетическая инъекция ввода:** эмуляция действий пользователя (мышь, колесо, клики, текст, клавиши клавиатуры, порядок табуляции).
- **Снимки экрана и Golden-тесты:** выгрузка кадра в `image.Image` / PNG и попиксельное сравнение с эталонными снимками (`AssertGolden` с флагом `go test -update`).

---

## Быстрый старт

```go
package myapp_test

import (
	"testing"

	"github.com/kirpicheff/anyui"
)

func TestButtonIncrement(t *testing.T) {
	reg, bold, mono := anyui.DefaultTestFonts()
	h, err := anyui.NewHeadless(anyui.Config{
		Width:       400,
		Height:      300,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("ошибка инициализации headless: %v", err)
	}
	defer h.Close()

	clicks := 0
	h.SetDraw(func(ui *anyui.UI) {
		ui.Padding(16, func() {
			if ui.Button("counter_btn", "Нажми меня", anyui.Width(120)) {
				clicks++
			}
		})
	})

	// 1. Первый шаг строит начальную геометрию виджетов
	h.Step()

	// 2. Эмулируем клик по кнопке (координаты X: 50, Y: 30)
	h.InjectMouseClick(50, 30, anyui.MouseLeft)
	h.Step()

	if clicks != 1 {
		t.Fatalf("ожидался 1 клик, получено: %d", clicks)
	}
}
```

---

## Детерминированные шрифты

Рендеринг текста стандартными системными шрифтами ОС (`Segoe UI`, `SF Pro`, `DejaVu Sans`) неизбежно приводит к разнице в метриках глифов между Windows, Linux и macOS.

Для гарантированной воспроизводимости golden-тестов конструктор `anyui.NewHeadless` **требует** явного указания `Config.FontRegular`. Если параметр не передан, конструктор вернёт ошибку.

Функция `anyui.DefaultTestFonts()` возвращает встроенные шрифты GoFont (`goregular.TTF`, `gobold.TTF`, `gomono.TTF`), гарантирующие идентичный рендеринг пиксель-в-пиксель на любой платформе:

```go
reg, bold, mono := anyui.DefaultTestFonts()
cfg := anyui.Config{
	FontRegular: reg,
	FontBold:    bold,
	FontMono:    mono,
}
```

---

## Пошаговое управление кадрами (`Step`)

Метод `h.Step()`:
1. Выполняет все отложенные задачи из очереди `app.Dispatch(fn)`.
2. Доставляет накопленные события ввода в контекст интерфейса.
3. Синхронно отрисовывает ровно один кадр и сохраняет его в готовый пиксельный буфер.

```go
h.App().Dispatch(func() {
	userState.Loaded = true
})

// Dispatch гарантированно выполняется при следующем вызове Step
h.Step()
```

---

## Виртуальный клок и таймеры

Все временные механизмы AnyUI (движок `Animator`, время жизни всплывающих уведомлений `ShowToast`, вычисление двойного клика, контролы даты/времени) привязаны к функции `c.Now()`, которая опрашивает виртуальный клок Headless-рантайма.

### Управление временем

- `h.Time() time.Time` — получить текущее виртуальное время.
- `h.SetTime(t time.Time)` — установить конкретный момент времени.
- `h.AdvanceTime(d time.Duration)` — перемотать виртуальное время вперёд на `d`.

### Пример: тестирование исчезновения тоста

```go
func TestToastExpiration(t *testing.T) {
	reg, bold, mono := anyui.DefaultTestFonts()
	h, _ := anyui.NewHeadless(anyui.Config{
		Width:       600,
		Height:      400,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	defer h.Close()

	h.SetDraw(func(ui *anyui.UI) {
		ui.ShowToast("Успех", "Операция выполнена", anyui.ToastSuccess, 2*time.Second)
	})

	h.Step() // Кадр создания тоста

	// Перематываем время на 1 секунду вперёд — тост всё ещё активен
	h.AdvanceTime(1 * time.Second)
	h.Step()

	// Перематываем ещё на 1.5 секунды (суммарно 2.5 сек) — тост истёк и удалён
	h.AdvanceTime(1500 * time.Millisecond)
	h.Step()
}
```
Тест выполняется мгновенно (за доли миллисекунды) без вызовов `time.Sleep` и без риска нестабильных падений из-за таймингов операционной системы.

---

## Синтетическая инъекция ввода

### Мышь и колесо
- `h.InjectMouseMove(x, y int)` — переместить курсор в координаты `(x, y)`.
- `h.InjectMouseDown(x, y int, btn anyui.MouseButton)` — зажать кнопку мыши (`MouseLeft`, `MouseRight`, `MouseMiddle`).
- `h.InjectMouseUp(x, y int, btn anyui.MouseButton)` — отпустить кнопку мыши.
- `h.InjectMouseClick(x, y int, btn anyui.MouseButton)` — полный цикл клика (`MouseMove` -> `MouseDown` -> `MouseUp`).
- `h.InjectMouseWheel(dx, dy int)` — прокрутить колесо мыши (`dy > 0` — вверх, `dy < 0` — вниз).

### Клавиатура и ввод текста
- `h.InjectText(text string)` — эмуляция ввода символов Unicode (`EventChar`) в сфокусированное поле.
- `h.InjectKey(key anyui.Key, mod anyui.Modifiers)` — полный цикл нажатия и отпускания клавиши.
- `h.InjectKeyDown(key, mod)` / `h.InjectKeyUp(key, mod)` — раздельные события нажатия и отпускания.

Константы виртуальных клавиш `anyui.Key`:
`KeyTab`, `KeyEnter`, `KeyEscape`, `KeyBackspace`, `KeySpace`, `KeyDelete`, `KeyLeft`, `KeyUp`, `KeyRight`, `KeyDown`, `KeyHome`, `KeyEnd`, `KeyPageUp`, `KeyPageDown`, `KeyA`..`KeyZ`.

### Фокус и навигация по табуляции
- `h.InjectFocus(id string)` — программно установить фокус на виджет по его `id`.
- `h.FocusID() string` — узнать идентификатор текущего виджета в фокусе.
- `h.FocusOrder() []string` — получить срез идентификаторов виджетов в порядке обхода по `Tab`.

```go
// Проверка перехода фокуса по Tab
h.InjectFocus("first_input")
h.Step()

h.InjectKey(anyui.KeyTab, anyui.Modifiers{})
h.Step()
if h.FocusID() != "second_input" {
	t.Fatalf("ожидался фокус на second_input, получен: %s", h.FocusID())
}

// Переход назад по Shift+Tab
h.InjectKey(anyui.KeyTab, anyui.Modifiers{Shift: true})
h.Step()
if h.FocusID() != "first_input" {
	t.Fatalf("ожидался фокус на first_input, получен: %s", h.FocusID())
}
```

---

## Снимки экрана и Golden-тесты

### Выгрузка снимков кадра
- `h.CaptureImage() image.Image` — возвращает текущий кадр в виде стандартного объекта `*image.NRGBA`.
- `h.CapturePNG(path string) error` — сохраняет текущий кадр в PNG-файл (автоматически создавая родительские директории).

### Инфраструктура регрессионных проверок (`AssertGolden`)

Функция `anyui.AssertGolden(t, h, goldenName)` сравнивает текущий кадр с эталонным файлом `testdata/<goldenName>.png`.

```go
func TestSettingsVisualGolden(t *testing.T) {
	reg, bold, mono := anyui.DefaultTestFonts()
	h, err := anyui.NewHeadless(anyui.Config{
		Width:       480,
		Height:      320,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
		Theme:       anyui.NavyTheme(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	h.SetDraw(func(ui *anyui.UI) {
		ui.Padding(20, func() {
			ui.Label("Настройки AnyUI", anyui.FontSize(18), anyui.Bold())
			ui.Button("btn_save", "Сохранить", anyui.Primary())
		})
	})

	h.Step()

	// Сравнение с эталоном testdata/golden_settings.png
	anyui.AssertGolden(t, h, "golden_settings")
}
```

### Создание и обновление эталонов (`-update`)

В тестовом файле зарегистрируйте флаг `-update`:

```go
var _ = flag.Bool("update", false, "обновить эталонные golden-файлы")
```

- **Первичная запись или обновление:**
  ```bash
  go test -v -run TestSettingsVisualGolden -update
  ```
  Команда создаст или перезапишет эталонный файл `testdata/golden_settings.png`.
- **Обычная проверка в CI:**
  ```bash
  go test ./...
  ```
  Тест сверяет каждый пиксель. При расхождении:
  1. Выводится количество несовпадающих пикселей и координаты первого отклонения.
  2. Фактический ошибочный снимок автоматически сохраняется в `testdata/<goldenName>.failed.png` для визуального анализа.

---

## Бенчмарки кадрового бюджета

С помощью `Headless` и `testing.B` можно измерять точное время отрисовки кадра и нагрузку на память для сложных экранов:

```go
func BenchmarkFrameBudget1000Widgets(b *testing.B) {
	reg, bold, mono := anyui.DefaultTestFonts()
	h, err := anyui.NewHeadless(anyui.Config{
		Width:       1280,
		Height:      960,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()

	h.SetDraw(func(ui *anyui.UI) {
		ui.ScrollView("bench_scroll", 1280, 960, func() {
			for i := 0; i < 200; i++ {
				ui.Row(8, 30, func() {
					ui.Label("Элемент")
					ui.Button("btn", "Клик")
					val := true
					ui.Checkbox("chk", "Выбор", &val)
					str := "Текст"
					ui.TextInput("txt", &str, "Подсказка")
				})
			}
		})
	})

	// Прогревочный шаг
	h.Step()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Step()
	}
}
```
Запуск бенчмарка:
```bash
go test -bench BenchmarkFrameBudget -benchmem -run ^$
```
Позволяет убедиться, что время кадра остаётся в пределах бюджета 16.6 мс (60 FPS), а аллокации не создают избыточного давления на сборщик мусора Go.

