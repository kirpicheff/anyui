[English](README.md) | [Русский](../ru/README.md)

# AnyUI (`anyui`)

A lightweight, embeddable cross-platform Go UI framework with software 2D rendering (no WebView2 and no OpenGL/GPU dependency).

## Key Features
- Single root package `anyui` for building interfaces.
- Zero CPU usage when idle (the OS event loop wakes up only on OS events, `Dispatch`, or active animations).
- Smooth animation engine (`Animator`) that automatically goes to sleep once transitions finish.
- Theme support (`NavyTheme`, `DarkTheme`, `LightTheme`) and functional styling options (`Option`).
- Keyboard focus navigation (`Tab` / `Shift+Tab`), confirmation via `Enter`, dismissal via `Escape`, and global shortcuts (`Shortcut`).
- Overlay layer (`PushOverlay`) blocking clicks beneath floating panels and popups.

## Quick Start

```go
package main

import (
	"log"

	"github.com/kirpicheff/anyui"
)

func main() {
	app := anyui.NewApp(anyui.Config{
		Title:  "My App",
		Width:  960,
		Height: 600,
		Theme:  anyui.NavyTheme(),
	})

	name := ""
	enabled := true

	err := app.Run(func(ui *anyui.UI) {
		ui.Padding(20, func() {
			ui.Column(12, func() {
				ui.Label("Settings", anyui.FontSize(18))
				ui.TextInput("inp_name", &name, "Enter name", anyui.Width(260))
				ui.Toggle("tgl_on", "Enable notifications", &enabled)
				if ui.Button("btn_save", "Save", anyui.Primary()) {
					log.Println("Saved:", name, enabled)
				}
			})
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

## Architecture

- `anyui` (module root) — public facade (`App`, `Window`, `UI`, `Config`, `Theme`, `Option`).
- `platform/` — cross-platform `Window` interface and native OS backends (Windows Win32 with Per-Monitor V2 DPI, Linux X11, Linux Wayland).
- `render/` — software 2D BGRA renderer, anti-aliased SDF primitives, SVG rasterizer, and font glyph cache.
- `ui/` — Immediate Mode UI context, layout engine (`Column`, `Row`, `Padding`, `Grid`, `Flow`), animation engine (`Animator`), overlays (`OverlayLayer`), focus management, and widgets.

## Concurrency (`Dispatch` and `Remember`)

State read inside `drawFn` must be updated from background goroutines via `app.Dispatch(fn)`. The callback runs on the UI thread, and the window is automatically woken up and redrawn:

```go
go func() {
	result := fetchData()
	app.Dispatch(func() {
		status = result
	})
}()
```

For typical async data loading directly inside `drawFn` without manual goroutine management, use `anyui.Remember`:

```go
users, loading, err := anyui.Remember(ui, "users_list", func() ([]User, error) {
	return api.FetchUsers()
})
if loading {
	ui.Spinner("", 20)
} else if err != nil {
	ui.Label("Error: "+err.Error(), anyui.TextColor(ui.Theme().Danger))
}
```
You can invalidate the cache by key using `ui.InvalidateRemember("users_list")`.

## Widget IDs (`ID` and `IDScope`)

Interactive widgets accept:
- an explicit unique `id` (recommended for text inputs and scrollable lists when preceding elements appear or disappear dynamically);
- an empty string `""` — the framework generates a deterministic ID within the current scope;
- `ui.IDScope(key, func() { ... })` (or `ui.PushID(key)` / `ui.PopID()`) — isolates the ID namespace in loops and reusable components.

## Multi-Window Support

`App` manages a single OS event loop. Additional independent or child windows can be opened with `app.OpenWindow(cfg, drawFn)`:

```go
win, _ := app.OpenWindow(anyui.Config{
	Title:  "Logs",
	Width:  640,
	Height: 400,
}, func(ui *anyui.UI) {
	ui.Padding(16, func() {
		ui.Label("Log window")
	})
})
```

To wake up and redraw only a specific window from a background goroutine (for example, when streaming frames to an `InteractiveCanvas` with `Streaming: true`), call `win.WakeUp()` instead of `app.Dispatch(...)` — this avoids waking other windows and preserves the `EventWakeUp` fast-path: `Canvas.Clear` and full widget tree reconstruction are skipped (only the streaming canvas `OnDraw` and `PresentRect` run, rendering directly into the Windows `DIBSection` + `BitBlt`). When surrounding UI elements (such as a status-bar FPS counter once per second) need updating, call `ctx.RequestRedraw()` (or `win.Invalidate()`).


