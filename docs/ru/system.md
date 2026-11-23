[English](../en/system.md) | [Русский](system.md)

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
- `SetAlwaysOnTop(bool)`, `IsAlwaysOnTop()` (также задаётся на старте через `Config.AlwaysOnTop` — закрепление окна поверх всех окон ОС)
- `Position() (x, y int)`, `SetPosition(x, y int)`, `SetSize(width, height int)`
- `SetHideOnClose(bool)` и `SetOnClose(func(w *Window) bool)` (возврат `false` отменяет закрытие)
- `SetCustomCursor(*CustomCursor)` — установка нативного кастомного курсора из BGRA-буфера (`nil` возвращает системный `CursorType`)
- `NativeHandle() uintptr` — нативный дескриптор окна ОС (`HWND` на Windows, `XID` на X11, ID `wl_surface` на Wayland, `NSWindow*` на macOS; `0` в `Headless` или до создания окна)

Для создания окна с кастомной шапкой (без системной рамки, но с сохранением теней DWM и Aero Snap):

```go
app := anyui.NewApp(anyui.Config{
	Title:          "Custom Chrome",
	Frameless:      true,
	TitleBarHeight: 38, // Область перетаскивания (px), если не задан HitTest
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

Работа с буфером обмена ОС доступна на уровне окна `*anyui.Window` и UI-контекста `*anyui.UI`:
- Текст: `win.ClipboardText() (string, error)` / `win.SetClipboardText(text string) error` (также доступно напрямую через `ui.ClipboardText()` / `ui.SetClipboardText()`; автоматически используется в `TextInput` / `TextArea` по `Ctrl+C` / `Ctrl+V`).
- Изображения: `win.ClipboardImage() (image.Image, error)` / `win.SetClipboardImage(img image.Image) error` (на Windows работает через `CF_DIB` 24/32 bpp, на Linux через PNG).

---

## Системные уведомления (`ShowNotification`)

Если настроен `Tray`, можно показывать нативные уведомления ОС (включая режим, когда окно свёрнуто в трей):

```go
err := app.ShowNotification(anyui.Notification{
	Title:   "Синхронизация завершена",
	Message: "Все файлы успешно обновлены",
	Icon:    anyui.NotifyInfo, // NotifyNone, NotifyInfo, NotifyWarning, NotifyError
	Silent:  false,
})
```

Для реакции на клик по уведомлению задайте `OnNotificationClick` в `TrayConfig`.

---

## Защита от повторного запуска (`SingleInstance`)

Чтобы запретить запуск второй копии приложения и передать аргументы командной строки уже работающему экземпляру:

```go
app := anyui.NewApp(anyui.Config{
	Title: "My App",
	SingleInstance: &anyui.SingleInstanceConfig{
		ID: "com.example.myapp",
		OnSecondInstance: func(a *anyui.App, args []string) {
			// Главное окно поднимается автоматически (если не задан SkipActivate: true)
		},
	},
})
if err := app.Run(drawFn); errors.Is(err, anyui.ErrAlreadyRunning) {
	return
}
```

Для ранней проверки до инициализации шрифтов и окон можно также вызвать `guard, already, err := anyui.AcquireSingleInstance("com.example.myapp", os.Args[1:])` и передать `guard` в `SingleInstanceConfig.Guard`.

---

## Сохранение и восстановление геометрии окна (`WindowState`)

```go
state, _ := anyui.LoadWindowState("window.json")
app := anyui.NewApp(anyui.Config{
	Title:        "My App",
	InitialState: &state,
	OnClose: func(w *anyui.Window) bool {
		_ = anyui.SaveWindowState("window.json", w.State())
		return true
	},
})
```

Если сохранённый монитор был отключён, окно автоматически переносится в видимую рабочую область ближайшего экрана.

---

## Системная тема и цвета рамки окна (DWM)

- `anyui.IsSystemDarkTheme() bool` и `anyui.SystemTheme() Theme` определяют текущую тему ОС.
- При `FollowSystemTheme: true` приложение автоматически стартует в теме ОС, реагирует на её смену на лету (`WM_SETTINGCHANGE`) и обновляет `OnSystemThemeChange(app, dark)`.
- На Windows 11 поддерживается настройка произвольного цвета системной шапки, текста заголовка и рамки окна через `Config` (`TitleBarColor`, `TitleTextColor`, `BorderColor`) или динамически методом `win.SetTitleBarColors(caption, text, border)`.
- При вызове `win.SetTheme(theme)` или `ui.SetTheme(theme)` атрибут `DWMWA_USE_IMMERSIVE_DARK_MODE` обновляется автоматически. Для полного совпадения шапки с темой на Windows 7/10/Server используйте `Frameless: true` и виджет `ui.TitleBar`.

---

## Платформенные утилиты и локализация (`i18n`)

- `anyui.OpenURL("https://example.com")` — безопасное открытие ссылок (`http`, `https`, `mailto`) в браузере по умолчанию.
- `anyui.RevealInFileManager(path)` — открыть Проводник и выделить указанный файл.
- Локализация встроенных строк контролов (контекстное меню текста, `DatePicker`, `TimePicker`, `MultiSelect`, `LogView`): передайте `Strings` в `Config` или вызывайте `app.SetStrings(anyui.EnglishStrings())` / `ui.SetStrings(...)` (доступны пресеты `RussianStrings()` и `EnglishStrings()`).

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

## Точки расширения для нативных интеграций

Ядро `anyui` остаётся полностью софтверным (без GPU/D3D11/OpenGL), но предоставляет два минимальных хука для внешних нативных интеграций (например, дочерней поверхности `HWND`/D3D11 поверх `InteractiveCanvas` в прикладном пакете):

- `win.NativeHandle() uintptr` — возвращает нативный дескриптор окна ОС (`HWND` на Windows, `XID` на Linux X11, ID `wl_surface` на Linux Wayland, указатель `NSWindow*` на macOS; `0` в `Headless` или до создания окна).
- `ui.HasActiveOverlays() bool` — возвращает `true`, если в текущем или предыдущем кадре присутствуют активные оверлеи (`Modal`, `Popover`, `FloatingPanel`, `Drawer`, `ContextMenu`, `Select`, `MultiSelect`, `Tooltip`, `PushOverlay`; всплывающие `Toast` не считаются блокирующими оверлеями). Позволяет внешней нативной поверхности корректно уступать z-order (например, скрываться и переключаться на софтверный фолбэк `Canvas.DrawBGRAFast`).

---

## Нюансы сборки под старые версии Windows

- Фреймворк использует только стандартные системные DLL (`user32.dll`, `gdi32.dll`, `kernel32.dll`, `shell32.dll`, `comdlg32.dll`, `ole32.dll`, `dwmapi.dll`) без зависимости от WebView2, OpenGL или DirectX.
- Гарантирована сборка без CGO (`CGO_ENABLED=0`), все системные вызовы выполняются через `golang.org/x/sys/windows` и `syscall`.
- Функции, появившиеся в новых версиях Windows (например, `SetProcessDpiAwarenessContext`, `GetDpiForWindow`, `DwmSetWindowAttribute`), загружаются лениво и имеют автоматический откат (fallback) для Windows 7 / Windows Server 2008 R2 / 2012.
- Для сборки компактного GUI-приложения (3–5 МБ) без консольного окна используйте:
  ```bat
  set CGO_ENABLED=0
  go build -trimpath -ldflags="-s -w -H windowsgui" .
  ```

---

## Linux (X11 и Wayland)

- Бэкенды Linux X11 и Wayland полностью реализованы на чистом Go без CGO (`CGO_ENABLED=0`), что позволяет кросс-компилировать бинарник с Windows (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`).
- При запуске бэкенд выбирается автоматически (`WAYLAND_DISPLAY` → Wayland, иначе X11, с автоматическим откатом на X11 и переопределением через переменную окружения `ANYUI_BACKEND`).
- **X11:** используется `MIT-SHM` (`ShmPutImage`) с автоматическим переключением на `PutImage` (например, при работе по SSH / X11-forwarding), `Xft.dpi`, системные курсоры, `Frameless` (`_MOTIF_WM_HINTS` + `_NET_WM_MOVERESIZE`), Drag & Drop файлов (`XDND`), захват мыши (`GrabPointer`) и буфер обмена `CLIPBOARD` (текст UTF-8 и изображения `image/png`).
- **Wayland:** используются `wl_compositor`, `wl_shm`, `xdg_shell`, `wl_seat` / `wl_pointer` / `wl_keyboard`, `wp_cursor_shape_manager_v1`, `wp_fractional_scale_manager_v1` / `wp_viewporter` для HiDPI и `wl_data_device_manager` для буфера обмена и Drag & Drop.
- **Системная интеграция:**
  - Диалоги (`OpenFileDialog`, `OpenFilesDialog`, `SaveFileDialog`, `SelectFolderDialog`, `MessageBox`) работают через `zenity`, `yad` или `kdialog` (автоопределение в `PATH`).
  - Системный трей (`Config.Tray`) поддерживает `StatusNotifierItem` (D-Bus) и `XEmbed` (`_NET_SYSTEM_TRAY_S0`), уведомления — через `notify-send`.
  - `SingleInstance` использует `flock` и UNIX domain socket в `$XDG_RUNTIME_DIR`.
  - `IsSystemDarkTheme` проверяет `gsettings` (`color-scheme` / `gtk-theme`), `GTK_THEME` и `~/.config/kdeglobals`.
  - `OpenURL` использует `xdg-open`, а `RevealInFileManager` — D-Bus `org.freedesktop.FileManager1.ShowItems` с откатом на `xdg-open`.

---

## macOS (AppKit / Cocoa)

- Для работы GUI-окон, системного трея и цикла событий на macOS требуется сборка с CGO (`CGO_ENABLED=1`), так как используется компактный нативный Objective-C мост к Cocoa/AppKit. Режим `CGO_ENABLED=0` поддерживается только в виде заглушек для кросс-компиляции и тестов без macOS SDK.
- **Рендеринг:** вывод пиксельного софтверного буфера через `CALayer.contents = CGImageRef` с автоматическим масштабированием под Retina (`backingScaleFactor`).
- **Окна и цикл событий:** нативный `NSApplication` в главном потоке, поддержка `Frameless` (`mouseDownCanMoveWindow`), маршалинг `app.Dispatch(fn)` через главный `dispatch_queue`.
- **Системная интеграция:**
  - Системный трей (`NSStatusItem` с иконкой и нативным `NSMenu`).
  - Системные уведомления через `UNUserNotificationCenter` с автоматическим откатом на `osascript` (`display notification`).
  - Нативные диалоги файлов `NSOpenPanel` / `NSSavePanel` / `NSAlert`.
  - Буфер обмена `NSPasteboard` (текст и PNG/TIFF), `SingleInstance` через UNIX-сокет.
  - Определение темы macOS через `AppleInterfaceStyle` (`Dark` / `Light`).


