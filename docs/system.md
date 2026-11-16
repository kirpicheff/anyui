# Системная интеграция (`anyui`)

В этом разделе описаны возможности платформенного уровня: системный трей, нативные диалоги, управление окном, безрамочный режим (`Frameless`), Drag & Drop файлов, буфер обмена, многооконность и нюансы совместимости со старыми версиями Windows.

---

## Системный трей (`TrayConfig`)

Иконка в области уведомлений создаётся при старте `App.Run`, если в `Config` передан `Tray`:

```go
app := anyui.NewApp(anyui.Config{
	Title:       "My Service",
	HideOnClose: true, // При нажатии на крестик окно сворачивается в трей
	Tray: &anyui.TrayConfig{
		Tooltip: "My Service — работает",
		OnDoubleClick: func(a *anyui.App) {
			a.Show()
		},
		Menu: []anyui.MenuItem{
			{Label: "Показать окно", OnClick: func(a *anyui.App) { a.Show() }},
			{Label: "Автозапуск", Checked: true, OnClick: func(a *anyui.App) { /* ... */ }},
			{Separator: true},
			{Label: "Неактивный пункт", Disabled: true},
			{Label: "Выход", OnClick: func(a *anyui.App) { a.Quit() }},
		},
	},
})
```

Доступно динамическое обновление из главного потока (или через `app.Dispatch`):
- `app.SetTrayTooltip(text)` — изменить текст подсказки.
- `app.SetTrayIcon(rgba, width, height)` — заменить иконку из RGBA-буфера (`width * height * 4` байт).
- `app.SetTrayMenu(items)` — обновить пункты контекстного меню.
- Если `IconRGBA` не задан, используется встроенная иконка по умолчанию.
- При вызове `app.Quit()` иконка автоматически удаляется из трея, а все окна закрываются (даже если у них включён `HideOnClose`).

---

## Системные диалоги

Диалоги работают поверх нативных API ОС (`comdlg32.dll`, `shell32.dll`, `user32.dll`):

```go
// Выбор одного файла
path, ok, err := anyui.OpenFileDialog(anyui.FileDialogOptions{
	Title: "Открыть конфигурацию",
	Filters: []anyui.FileFilter{
		{Name: "JSON (*.json)", Pattern: "*.json"},
		{Name: "Все файлы (*.*)", Pattern: "*.*"},
	},
	Owner: ui.Window(),
})

// Выбор нескольких файлов
files, ok, err := anyui.OpenFilesDialog(anyui.FileDialogOptions{
	Title: "Выберите файлы",
	Owner: ui.Window(),
})

// Сохранение файла
savePath, ok, err := anyui.SaveFileDialog(anyui.FileDialogOptions{
	Title:       "Сохранить отчёт",
	DefaultName: "report.txt",
	Owner:       ui.Window(),
})

// Выбор каталога
dir, ok, err := anyui.SelectFolderDialog(anyui.FolderDialogOptions{
	Title: "Выберите папку проекта",
	Owner: ui.Window(),
})

// Сообщение
res := anyui.MessageBox(
	ui.Window(),
	"Подтверждение",
	"Удалить выбранный элемент?",
	anyui.MsgQuestion,
	anyui.MsgYesNo,
)
if res == anyui.ResYes {
	// ...
}
```

---

## Управление окном и режим `Frameless`

Методы `*anyui.Window`:
- `Minimize()`, `Maximize()`, `Restore()`, `IsMaximized()`, `IsMinimized()`
- `SetFullscreen(bool)`, `IsFullscreen()`
- `Position() (x, y int)`, `SetPosition(x, y int)`, `SetSize(width, height int)`
- `SetHideOnClose(bool)` и `SetOnClose(func(w *Window) bool)` (возврат `false` отменяет закрытие)

Для создания окна с кастомной шапкой (без системной рамки, но с сохранением теней DWM и Aero Snap):

```go
app := anyui.NewApp(anyui.Config{
	Title:          "Custom Chrome",
	Frameless:      true,
	TitleBarHeight: 38, // Область перетаскивания (dp), если не задан HitTest
	HitTest: func(x, y int) anyui.HitTestResult {
		// Например, оставляем правый верхний угол кликабельным для кнопок управления
		if y < 38 && x < 760 {
			return anyui.HitCaption
		}
		return anyui.HitClient
	},
})
```

---

## Drag & Drop файлов

Включите `AcceptDropFiles: true` в `Config` и передайте обработчик `OnDropFiles`:

```go
app := anyui.NewApp(anyui.Config{
	Title:           "Drop Target",
	AcceptDropFiles: true,
	OnDropFiles: func(files []string, x, y int) {
		// files содержит пути к перетащенным файлам
	},
})
```

---

## Буфер обмена

Работа с буфером обмена доступна через платформенный слой окна (`ClipboardText()` / `SetClipboardText(text)`), а также автоматически используется в `TextInput` по `Ctrl+C` / `Ctrl+V`.

---

## Многооконность

Из одного `App` можно открывать дополнительные независимые или дочерние окна:

```go
child, err := app.OpenWindow(anyui.Config{
	Title:  "Настройки",
	Width:  500,
	Height: 400,
	Parent: ui.Window(), // Привязка к родительскому окну
}, func(cui *anyui.UI) {
	cui.Label("Окно настроек")
})
```

- Закрытие дочернего окна не завершает `App`, пока открыто хотя бы одно окно (или приложение работает через трей).
- `app.Dispatch(fn)` безопасно обновляет данные из фоновых горутин и пробуждает активные окна.

---

## Нюансы сборки под старые версии Windows

- Фреймворк использует только стандартные системные DLL (`user32.dll`, `gdi32.dll`, `kernel32.dll`, `shell32.dll`, `comdlg32.dll`, `ole32.dll`, `dwmapi.dll`) без зависимости от WebView2, OpenGL или DirectX.
- Функции, появившиеся в новых версиях Windows (например, `SetProcessDpiAwarenessContext`, `GetDpiForWindow`, `DwmSetWindowAttribute`), загружаются лениво и имеют автоматический откат (fallback) для Windows 7 / Windows Server 2008 R2 / 2012.
- Для сборки GUI-приложения без консольного окна используйте флаг линковщика:
  ```bat
  go build -ldflags="-H windowsgui" .
  ```

