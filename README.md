**English** | [Русский](README.ru.md)

# AnyUI (`anyui`)

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](go.mod)
[![CGO](https://img.shields.io/badge/CGO-Disabled_(Win%2FLinux)-success.svg)](#platform-support)

A lightweight, embeddable cross-platform Go UI framework powered by a pure-Go software 2D renderer — **no WebView2, no OpenGL/GPU, and no CGO** on Windows and Linux.

Designed for desktop tools, system utilities, background services with a tray icon, and environments without hardware acceleration (servers, VMs, RDP, SSH X11 forwarding).

---

## Key Features

- **Zero Idle CPU Usage:** the OS event loop sleeps until woken by user input, `App.Dispatch`, or active animations.
- **Software 2D Renderer (`render`):** anti-aliased SDF primitives, linear/radial gradients, soft shadows (`BoxShadow`), backdrop blur, SVG icon rasterization with tinting, LRU glyph/image caches, and dirty-rectangle presentation.
- **Pure Go on Windows & Linux (`CGO_ENABLED=0`):** builds compact standalone binaries (3–5 MB) with zero external runtime dependencies.
- **Rich Widget Set (`ui`):** Immediate Mode API with `Button`, `TextInput`, `MaskedInput`, `TextArea`, `Select`, `ComboBox`, `MultiSelect`, `Table`, `TreeView`, `ListView`, `TabBar`, `MenuBar`, `Toolbar`, `StatusBar`, `LogView`, charts (`Sparkline`, `LineChart`, `BarChart`), `DatePicker`, `TimePicker`, `ColorPicker`, `Modal`, `Drawer`, `SplitPane`, and more.
- **Flexible Layout & Styling:** `Row`, `Column`, `RowFlex`, `ColumnFlex`, `Grid`, `FormLayout`, `Flow`, `ScrollView`, `Card`, `Page`, built-in themes (`DeepNavyTheme`, `DarkTheme`, `LightTheme`, `SystemTheme`), and functional `Option` modifiers.
- **System Integration (`platform`):** system tray, native notifications, file/folder/message dialogs, text & image clipboard, file Drag & Drop, `Frameless` / custom `TitleBar`, `SingleInstance`, `WindowState` persistence, multi-window support, and `i18n`.

---

## Platform Support

| Platform | Backend | CGO | Status |
| --- | --- | --- | --- |
| **Windows** (7 / 10 / 11 / Server) | Win32 + GDI + DWM | `CGO_ENABLED=0` | Supported |
| **Linux (X11)** | `xgb` + MIT-SHM / PutImage | `CGO_ENABLED=0` | Supported |
| **Linux (Wayland)** | Native Wayland (`wl_shm`, `xdg_shell`) | `CGO_ENABLED=0` | Supported |
| **macOS** | Cocoa / AppKit | Planned (`CGO`) | Planned |

---

## Quick Start

```bash
go get git.gtw.su/kirpich/anyui
```

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
				ui.Label("Settings", anyui.FontSize(18), anyui.Bold())
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

To run the interactive widget showcase:

```bash
go run ./cmd/demo
```

---

## Documentation

- [Overview & Architecture](docs/en/README.md)
- [Widgets & Layout](docs/en/widgets.md)
- [Graphics & 2D Rendering](docs/en/graphics.md)
- [Themes & Styling](docs/en/theming.md)
- [System Integration (Windows, Tray, Dialogs, Linux)](docs/en/system.md)

You can also inspect the Go API directly from the terminal:

```bash
go doc .
go doc -all .
```

---

## License

Licensed under the [Apache License 2.0](LICENSE). See [NOTICE](NOTICE) for attribution details.
