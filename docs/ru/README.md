[English](../en/README.md) | [Русский](README.md)

# AnyUI (`anyui`)

Лёгкий, встраиваемый кроссплатформенный UI-фреймворк на Go с программным рендерингом (без WebView2 и без зависимости от OpenGL/GPU).

## Ключевые особенности
- Один корневой пакет `anyui` для построения интерфейса.
- Нулевая нагрузка на CPU в простое (цикл сообщений `GetMessageW` пробуждается только при событиях ОС, `Dispatch` или активных анимациях).
- Плавный движок анимаций (`Animator`) с автоматическим переходом в сон после завершения переходов.
- Поддержка тем (`DeepNavyTheme`, `DarkTheme`, `LightTheme`) и функциональных опций стилизации (`Option`).
- Навигация фокуса по `Tab` / `Shift+Tab`, подтверждение по `Enter`, сброс по `Escape`, горячие клавиши (`Shortcut`).
- Слой оверлеев (`PushOverlay`) с блокировкой кликов под всплывающими панелями.

## Быстрый старт

```go
package main

import (
	"log"

	"git.gtw.su/kirpich/anyui"
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

## Архитектура

- `anyui` (корень модуля) — публичный фасад (`App`, `Window`, `UI`, `Config`, `Theme`, `Option`).
- `platform/` — кроссплатформенный интерфейс `Window` и нативный Win32-бэкенд (DPI Per-Monitor V2, цикл событий, буфер обмена).
- `render/` — программный 2D-рендерер BGRA, SDF-примитивы со сглаживанием, кеш глифов шрифтов.
- `ui/` — Immediate Mode UI-контекст, раскладка (`Column`, `Row`, `Padding`), движок анимаций (`Animator`), оверлеи (`OverlayLayer`), фокус и виджеты.

## Многопоточность (`Dispatch` и `Remember`)

Обновлять состояние, которое читается в `drawFn`, из фоновых горутин нужно через `app.Dispatch(fn)`. Переданная функция выполнится в главном потоке UI, а окно будет автоматически разбужено и перерисовано:

```go
go func() {
	result := fetchData()
	app.Dispatch(func() {
		status = result
	})
}()
```

Для типовой загрузки данных прямо внутри `drawFn` без ручного управления горутинами используйте `anyui.Remember`:

```go
users, loading, err := anyui.Remember(ui, "users_list", func() ([]User, error) {
	return api.FetchUsers()
})
if loading {
	ui.Spinner("", 20)
} else if err != nil {
	ui.Label("Ошибка: "+err.Error(), anyui.TextColor(ui.Theme().Danger))
}
```
Сбросить кеш по ключу можно через `ui.InvalidateRemember("users_list")`.

## Идентификаторы виджетов (`ID` и `IDScope`)

Интерактивным виджетам можно передавать:
- явный уникальный `id` (рекомендуется для полей ввода и прокручиваемых списков, если элементы выше по дереву появляются/скрываются динамически);
- пустую строку `""` — фреймворк автоматически сгенерирует детерминированный ID в текущем скоупе;
- `ui.IDScope(key, func() { ... })` (или пару `ui.PushID(key)` / `ui.PopID()`) — изолирует пространство имён ID в циклах и переиспользуемых компонентах.

## Многооконность

`App` управляет единым циклом событий ОС. Дополнительные независимые окна открываются через `app.OpenWindow(cfg, drawFn)`:

```go
_, _ = app.OpenWindow(anyui.Config{
	Title:  "Логи",
	Width:  640,
	Height: 400,
}, func(ui *anyui.UI) {
	ui.Padding(16, func() {
		ui.Label("Окно логов")
	})
})
```

