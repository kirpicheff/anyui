# AnyUI (`anyui`)

Лёгкий кроссплатформенный UI-фреймворк на Go с программным 2D-рендерингом (без WebView2 и без зависимости от OpenGL/GPU).

## Ключевые особенности

- Единый корневой пакет `anyui` для построения интерфейса (`App`, `Window`, `UI`, `Theme`, `Option`).
- Нулевая нагрузка на CPU в простое: цикл событий ОС пробуждается только при пользовательских событиях, `App.Dispatch` или активных анимациях.
- Программный 2D-рендерер (`render`): сглаженные SDF-примитивы, градиенты, тени, кеш глифов и SVG, частичная перерисовка (dirty rectangles).
- Темы оформления (`DeepNavyTheme`, `DarkTheme`, `LightTheme`, `SystemTheme`) и функциональные опции стилизации (`Primary`, `Width`, `FontSize` и др.).
- Навигация фокуса (`Tab` / `Shift+Tab`), горячие клавиши, оверлеи, системный трей, диалоги и защита от повторного запуска.

## Быстрый старт

```go
package main

import (
	"log"

	"anyui"
)

func main() {
	app := anyui.NewApp(anyui.Config{
		Title:  "My App",
		Width:  960,
		Height: 600,
		Theme:  anyui.DeepNavyTheme(),
	})

	name := ""
	enabled := true

	err := app.Run(func(ui *anyui.UI) {
		ui.Padding(20, func() {
			ui.Column(12, func() {
				ui.Label("Настройки", anyui.FontSize(18))
				ui.TextInput("inp_name", &name, "Введите имя", anyui.Width(260))
				ui.Toggle("tgl_on", "Включить уведомления", &enabled)
				if ui.Button("btn_save", "Сохранить", anyui.Primary()) {
					log.Println("Сохранено:", name, enabled)
				}
			})
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

## Документация

- [Обзор и архитектура](docs/README.md)
- [Виджеты и раскладка](docs/widgets.md)
- [Графика и 2D-рендеринг](docs/graphics.md)
- [Темы и стилизация](docs/theming.md)
- [Системная интеграция (окна, трей, диалоги)](docs/system.md)

Для просмотра справочника API из терминала используйте стандартную команду:

```sh
go doc .
go doc -all .
```

