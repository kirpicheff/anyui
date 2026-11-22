[English](system.md) | [Русский](../ru/system.md)

# System Integration (`anyui`)

This guide covers platform-level features: system tray, native dialogs, window management, frameless mode (`Frameless`), file Drag & Drop, clipboard, multi-window support, and platform-specific build details.

---

## System Tray (`TrayConfig`)

A notification area icon is created on `App.Run` when `Tray` is provided in `Config`:

```go
app := anyui.NewApp(anyui.Config{
	Title:       "My Service",
	HideOnClose: true, // Minimize to tray when closing the window
	Tray: &anyui.TrayConfig{
		Tooltip: "My Service — running",
		OnDoubleClick: func(a *anyui.App) {
			a.Show()
		},
		Menu: []anyui.MenuItem{
			{Label: "Show Window", OnClick: func(a *anyui.App) { a.Show() }},
			{Label: "Autostart", Checked: true, OnClick: func(a *anyui.App) { /* ... */ }},
			{Separator: true},
			{Label: "Disabled Item", Disabled: true},
			{Label: "Quit", OnClick: func(a *anyui.App) { a.Quit() }},
		},
	},
})
```

Dynamic updates from the main thread (or via `app.Dispatch`):
- `app.SetTrayTooltip(text)` — update the tooltip text.
- `app.SetTrayIcon(rgba, width, height)` — replace the icon from an RGBA buffer (`width * height * 4` bytes).
- `app.SetTrayMenu(items)` — update context menu items.
- If `IconRGBA` is omitted, a built-in default icon is used.
- Calling `app.Quit()` removes the tray icon and closes all windows (even if `HideOnClose` is enabled).

---

## Native Dialogs

Dialogs use native OS APIs:

```go
// Single file selection
path, ok, err := anyui.OpenFileDialog(anyui.FileDialogOptions{
	Title: "Open Configuration",
	Filters: []anyui.FileFilter{
		{Name: "JSON (*.json)", Pattern: "*.json"},
		{Name: "All Files (*.*)", Pattern: "*.*"},
	},
	Owner: ui.Window(),
})

// Multiple file selection
files, ok, err := anyui.OpenFilesDialog(anyui.FileDialogOptions{
	Title: "Select Files",
	Owner: ui.Window(),
})

// Save file
savePath, ok, err := anyui.SaveFileDialog(anyui.FileDialogOptions{
	Title:       "Save Report",
	DefaultName: "report.txt",
	Owner:       ui.Window(),
})

// Folder selection
dir, ok, err := anyui.SelectFolderDialog(anyui.FolderDialogOptions{
	Title: "Select Project Folder",
	Owner: ui.Window(),
})

// Message box
res := anyui.MessageBox(
	ui.Window(),
	"Confirmation",
	"Delete the selected item?",
	anyui.MsgQuestion,
	anyui.MsgYesNo,
)
if res == anyui.ResYes {
	// ...
}
```

---

## Window Management and `Frameless` Mode

`*anyui.Window` methods:
- `Minimize()`, `Maximize()`, `Restore()`, `IsMaximized()`, `IsMinimized()`
- `SetFullscreen(bool)`, `IsFullscreen()`
- `SetAlwaysOnTop(bool)`, `IsAlwaysOnTop()` (also configurable at creation via `Config.AlwaysOnTop` — keeps the window above all OS windows)
- `Position() (x, y int)`, `SetPosition(x, y int)`, `SetSize(width, height int)`
- `SetHideOnClose(bool)` and `SetOnClose(func(w *Window) bool)` (returning `false` cancels closing)

To build a custom title bar window (without OS chrome, preserving DWM shadows and Aero Snap on Windows):

```go
app := anyui.NewApp(anyui.Config{
	Title:          "Custom Chrome",
	Frameless:      true,
	TitleBarHeight: 38, // Drag area height (px) if HitTest is not provided
	HitTest: func(x, y int) anyui.HitTestResult {
		// Leave the top-right corner clickable for window control buttons
		if y < 38 && x < 760 {
			return anyui.HitCaption
		}
		return anyui.HitClient
	},
})
```

---

## File Drag & Drop

Enable `AcceptDropFiles: true` in `Config` and provide an `OnDropFiles` handler:

```go
app := anyui.NewApp(anyui.Config{
	Title:           "Drop Target",
	AcceptDropFiles: true,
	OnDropFiles: func(files []string, x, y int) {
		// files contains paths to the dropped files
	},
})
```

---

## Clipboard

Clipboard access is available on `*anyui.Window` and `*anyui.UI`:
- Text: `win.ClipboardText() (string, error)` / `win.SetClipboardText(text string) error` (also directly accessible via `ui.ClipboardText()` / `ui.SetClipboardText()`; used automatically by `TextInput` / `TextArea` on `Ctrl+C` / `Ctrl+V`).
- Images: `win.ClipboardImage() (image.Image, error)` / `win.SetClipboardImage(img image.Image) error` (on Windows uses `CF_DIB` 24/32 bpp; on Linux uses PNG).

---

## System Notifications (`ShowNotification`)

When `Tray` is configured, you can display native OS notifications (even when the window is minimized to the tray):

```go
err := app.ShowNotification(anyui.Notification{
	Title:   "Sync Complete",
	Message: "All files have been updated",
	Icon:    anyui.NotifyInfo, // NotifyNone, NotifyInfo, NotifyWarning, NotifyError
	Silent:  false,
})
```

Set `OnNotificationClick` in `TrayConfig` to handle notification clicks.

---

## Single Instance (`SingleInstance`)

Prevent launching a second copy of your application and forward CLI arguments to the running instance:

```go
app := anyui.NewApp(anyui.Config{
	Title: "My App",
	SingleInstance: &anyui.SingleInstanceConfig{
		ID: "com.example.myapp",
		OnSecondInstance: func(a *anyui.App, args []string) {
			// Main window is activated automatically (unless SkipActivate: true)
		},
	},
})
if err := app.Run(drawFn); errors.Is(err, anyui.ErrAlreadyRunning) {
	return
}
```

For an early check before initializing fonts and windows, call `guard, already, err := anyui.AcquireSingleInstance("com.example.myapp", os.Args[1:])` and pass `guard` into `SingleInstanceConfig.Guard`.

---

## Saving and Restoring Window Geometry (`WindowState`)

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

If the previously used monitor is disconnected, the window is automatically moved into the visible work area of the nearest screen.

---

## System Theme and Window Frame Colors (DWM)

- `anyui.IsSystemDarkTheme() bool` and `anyui.SystemTheme() Theme` detect the current OS theme.
- With `FollowSystemTheme: true`, the app starts in the OS theme, reacts to live theme changes (`WM_SETTINGCHANGE`), and invokes `OnSystemThemeChange(app, dark)`.
- On Windows 11, custom title bar, title text, and border colors are supported via `Config` (`TitleBarColor`, `TitleTextColor`, `BorderColor`) or dynamically with `win.SetTitleBarColors(caption, text, border)`.
- Calling `win.SetTheme(theme)` or `ui.SetTheme(theme)` automatically updates `DWMWA_USE_IMMERSIVE_DARK_MODE`. For 100% consistent styling on Windows 7/10/Server, use `Frameless: true` with `ui.TitleBar`.

---

## Platform Utilities and Localization (`i18n`)

- `anyui.OpenURL("https://example.com")` — safely opens `http`, `https`, or `mailto` links in the default browser.
- `anyui.RevealInFileManager(path)` — opens the system file manager and selects the specified file.
- Built-in control strings (text context menu, `DatePicker`, `TimePicker`, `MultiSelect`, `LogView`): pass `Strings` in `Config` or call `app.SetStrings(anyui.EnglishStrings())` / `ui.SetStrings(...)` (`RussianStrings()` and `EnglishStrings()` presets are included).

---

## Multi-Window Support

You can open additional independent or child windows from a single `App`:

```go
child, err := app.OpenWindow(anyui.Config{
	Title:  "Settings",
	Width:  500,
	Height: 400,
	Parent: ui.Window(), // Bind to parent window
}, func(cui *anyui.UI) {
	cui.Label("Settings window")
})
```

- Closing a child window does not exit `App` as long as at least one window remains open (or the app is running in tray mode).
- `app.Dispatch(fn)` safely updates state from background goroutines and wakes up active windows.

---

## Building for Legacy Windows Versions

- The framework uses only standard system DLLs (`user32.dll`, `gdi32.dll`, `kernel32.dll`, `shell32.dll`, `comdlg32.dll`, `ole32.dll`, `dwmapi.dll`) with no dependency on WebView2, OpenGL, or DirectX.
- Pure Go build without CGO (`CGO_ENABLED=0`) via `golang.org/x/sys/windows` and `syscall`.
- Newer Windows APIs (`SetProcessDpiAwarenessContext`, `GetDpiForWindow`, `DwmSetWindowAttribute`) are loaded lazily with automatic fallbacks for Windows 7 / Windows Server 2008 R2 / 2012.
- To build a compact GUI binary (3–5 MB) without a console window:
  ```bat
  set CGO_ENABLED=0
  go build -trimpath -ldflags="-s -w -H windowsgui" .
  ```

---

## Linux (X11 & Wayland)

- Both Linux X11 and Wayland backends are implemented in pure Go without CGO (`CGO_ENABLED=0`), enabling direct cross-compilation from Windows (`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build`).
- At runtime, the backend is selected automatically (`WAYLAND_DISPLAY` → Wayland, otherwise X11, with fallback to X11 and `ANYUI_BACKEND` override).
- **X11:** uses `MIT-SHM` (`ShmPutImage`) with automatic fallback to `PutImage` (e.g., over SSH / X11-forwarding), `Xft.dpi`, system cursors, `Frameless`, `XDND`, `GrabPointer`, and `CLIPBOARD` (UTF-8 text and PNG images).
- **Wayland:** uses `wl_compositor`, `wl_shm`, `xdg_shell`, `wl_seat` / `wl_pointer` / `wl_keyboard`, `wp_cursor_shape_manager_v1`, `wp_fractional_scale_manager_v1` / `wp_viewporter` for HiDPI, and `wl_data_device_manager` for clipboard and Drag & Drop.
- **System integration:**
  - Dialogs (`OpenFileDialog`, `OpenFilesDialog`, `SaveFileDialog`, `SelectFolderDialog`, `MessageBox`) use `zenity`, `yad`, or `kdialog` (auto-detected in `PATH`).
  - System tray (`Config.Tray`) supports `StatusNotifierItem` (D-Bus) and `XEmbed` (`_NET_SYSTEM_TRAY_S0`); notifications use `notify-send`.
  - `SingleInstance` uses `flock` and a UNIX domain socket in `$XDG_RUNTIME_DIR`.
  - `IsSystemDarkTheme` checks `gsettings` (`color-scheme` / `gtk-theme`), `GTK_THEME`, and `~/.config/kdeglobals`.
  - `OpenURL` uses `xdg-open`; `RevealInFileManager` uses D-Bus `org.freedesktop.FileManager1.ShowItems` with fallback to `xdg-open`.

---

## macOS (AppKit / Cocoa)

- GUI windows, system tray, and the event loop on macOS require CGO (`CGO_ENABLED=1`) via a lightweight native Objective-C bridge to Cocoa/AppKit. Building with `CGO_ENABLED=0` is supported only as a stub fallback for cross-compilation and CI testing without the macOS SDK.
- **Rendering:** software pixel buffer presented via `CALayer.contents = CGImageRef` with automatic Retina scaling (`backingScaleFactor`).
- **Window & Lifecycle:** native `NSApplication` on the main thread, `Frameless` support (`mouseDownCanMoveWindow`), and thread marshaling for `app.Dispatch(fn)` via the main dispatch queue.
- **System Integration:**
  - System tray (`NSStatusItem` with template icon and native `NSMenu`).
  - Native system notifications via `UNUserNotificationCenter` with automatic fallback to `osascript` (`display notification`).
  - Native file dialogs (`NSOpenPanel` / `NSSavePanel` / `NSAlert`).
  - Pasteboard (`NSPasteboard` for UTF-8 text and PNG/TIFF images), `SingleInstance` via domain sockets.
  - Automatic system dark/light theme detection via `AppleInterfaceStyle`.


