[English](testing.md) | [Русский](../ru/testing.md)

# Testing & Headless Runtime (`anyui`)

The `anyui` package provides a built-in headless runtime for unit, integration, and visual regression testing of your UI without opening physical OS windows or relying on window managers (WinAPI, X11, Wayland, Cocoa).

---

## Key Features

- **100% OS Isolation:** Tests run in pure Go with no window servers and zero CGO, including headless Linux CI environments (GitHub Actions, GitLab CI, Docker).
- **Deterministic Fonts:** Mandatory embedded fonts (`anyui.DefaultTestFonts()`) guarantee identical glyph rasterization across Windows, Linux, and macOS.
- **Step-by-step Frame Control (`h.Step()`):** Synchronous single-frame rendering on demand with guaranteed flushing of the `app.Dispatch()` task queue.
- **Deterministic Virtual Clock:** Control virtual time (`h.AdvanceTime()`, `h.SetTime()`) to test transitions and timers (such as `ShowToast` lifespans) without flaky delays or `time.Sleep`.
- **Synthetic Input Injection:** Emulate user actions: mouse moves, clicks, scroll wheel, unicode text typing, keyboard keys, and Tab navigation order.
- **Screen Captures & Golden Tests:** Export frames to `image.Image` / PNG and compare pixels against reference screenshots (`AssertGolden` with `go test -update`).

---

## Quick Start

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
		t.Fatalf("failed to initialize headless: %v", err)
	}
	defer h.Close()

	clicks := 0
	h.SetDraw(func(ui *anyui.UI) {
		ui.Padding(16, func() {
			if ui.Button("counter_btn", "Click me", anyui.Width(120)) {
				clicks++
			}
		})
	})

	// 1. Initial step builds layout geometry
	h.Step()

	// 2. Emulate mouse click on button (coordinates X: 50, Y: 30)
	h.InjectMouseClick(50, 30, anyui.MouseLeft)
	h.Step()

	if clicks != 1 {
		t.Fatalf("expected 1 click, got: %d", clicks)
	}
}
```

---

## Deterministic Fonts

Rendering text using host system fonts (`Segoe UI`, `SF Pro`, `DejaVu Sans`) leads to subtle glyph metric variations across Windows, Linux, and macOS.

To guarantee pixel-perfect visual tests, `anyui.NewHeadless` **requires** `Config.FontRegular`. If omitted, the constructor returns an error.

The helper function `anyui.DefaultTestFonts()` provides embedded GoFont typefaces (`goregular.TTF`, `gobold.TTF`, `gomono.TTF`), ensuring 100% deterministic cross-platform rendering:

```go
reg, bold, mono := anyui.DefaultTestFonts()
cfg := anyui.Config{
	FontRegular: reg,
	FontBold:    bold,
	FontMono:    mono,
}
```

---

## Step-by-Step Frame Control (`Step`)

Calling `h.Step()`:
1. Executes all pending tasks from the `app.Dispatch(fn)` queue.
2. Flushes queued input events to the UI context.
3. Synchronously renders exactly one frame and saves it to the frame buffer.

```go
h.App().Dispatch(func() {
	userState.Loaded = true
})

// Dispatch is guaranteed to execute during the next Step() call
h.Step()
```

---

## Virtual Clock & Timers

All time-dependent features in AnyUI (the `Animator` engine, `ShowToast` message durations, double-click detection, date/time pickers) rely on `c.Now()`, which queries the Headless runtime virtual clock.

### Time Control

- `h.Time() time.Time` — retrieve the current virtual time.
- `h.SetTime(t time.Time)` — set an explicit virtual point in time.
- `h.AdvanceTime(d time.Duration)` — advance virtual time forward by duration `d`.

### Example: Testing Toast Expiration

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
		ui.ShowToast("Notice", "Operation complete", anyui.ToastSuccess, 2*time.Second)
	})

	h.Step() // Frame where toast is shown

	// Advance time by 1 second — toast remains visible
	h.AdvanceTime(1 * time.Second)
	h.Step()

	// Advance by another 1.5 seconds (2.5s total) — toast expires and is removed
	h.AdvanceTime(1500 * time.Millisecond)
	h.Step()
}
```
This test runs in sub-milliseconds without calling `time.Sleep` and with zero flakiness.

---

## Synthetic Input Injection

### Mouse and Scroll Wheel
- `h.InjectMouseMove(x, y int)` — move virtual cursor to `(x, y)`.
- `h.InjectMouseDown(x, y int, btn anyui.MouseButton)` — press a mouse button (`MouseLeft`, `MouseRight`, `MouseMiddle`).
- `h.InjectMouseUp(x, y int, btn anyui.MouseButton)` — release a mouse button.
- `h.InjectMouseClick(x, y int, btn anyui.MouseButton)` — full click sequence (`MouseMove` -> `MouseDown` -> `MouseUp`).
- `h.InjectMouseWheel(dx, dy int)` — rotate wheel (`dy > 0` scrolls up, `dy < 0` scrolls down).

### Keyboard and Text
- `h.InjectText(text string)` — emulate typing Unicode runes (`EventChar`) into the focused widget.
- `h.InjectKey(key anyui.Key, mod anyui.Modifiers)` — full key press and release cycle.
- `h.InjectKeyDown(key, mod)` / `h.InjectKeyUp(key, mod)` — separate down and up events.

Virtual key constants `anyui.Key`:
`KeyTab`, `KeyEnter`, `KeyEscape`, `KeyBackspace`, `KeySpace`, `KeyDelete`, `KeyLeft`, `KeyUp`, `KeyRight`, `KeyDown`, `KeyHome`, `KeyEnd`, `KeyPageUp`, `KeyPageDown`, `KeyA`..`KeyZ`.

### Focus & Tab Navigation
- `h.InjectFocus(id string)` — focus a widget programmatically by `id`.
- `h.FocusID() string` — return the current focused widget `id`.
- `h.FocusOrder() []string` — return widget IDs in Tab traversal order.

```go
// Test Tab focus cycling
h.InjectFocus("first_input")
h.Step()

h.InjectKey(anyui.KeyTab, anyui.Modifiers{})
h.Step()
if h.FocusID() != "second_input" {
	t.Fatalf("expected focus on second_input, got: %s", h.FocusID())
}

// Cycle backward using Shift+Tab
h.InjectKey(anyui.KeyTab, anyui.Modifiers{Shift: true})
h.Step()
if h.FocusID() != "first_input" {
	t.Fatalf("expected focus on first_input, got: %s", h.FocusID())
}
```

---

## Screen Captures & Golden Tests

### Frame Captures
- `h.CaptureImage() image.Image` — returns the rendered frame as a standard `*image.NRGBA`.
- `h.CapturePNG(path string) error` — saves the rendered frame to a PNG file (creating parent directories as needed).

### Regression Testing (`AssertGolden`)

The `anyui.AssertGolden(t, h, goldenName)` helper compares the current frame against a golden baseline at `testdata/<goldenName>.png`.

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
			ui.Label("Settings", anyui.FontSize(18), anyui.Bold())
			ui.Button("btn_save", "Save", anyui.Primary())
		})
	})

	h.Step()

	// Compare with testdata/golden_settings.png
	anyui.AssertGolden(t, h, "golden_settings")
}
```

### Baseline Creation & Updates (`-update`)

Register the `-update` flag in your test file:

```go
var _ = flag.Bool("update", false, "update golden files")
```

- **Generate or Update Baseline:**
  ```bash
  go test -v -run TestSettingsVisualGolden -update
  ```
- **Regular CI Test:**
  ```bash
  go test ./...
  ```
  Compares pixels individually. On discrepancy:
  1. Reports differing pixel count and first coordinate mismatch.
  2. Saves actual output to `testdata/<goldenName>.failed.png` for visual inspection.

---

## Frame Budget Benchmarks

Combine `Headless` with `testing.B` to monitor frame rendering times, memory allocations, and GC pressure:

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
					ui.Label("Item")
					ui.Button("btn", "Click")
					val := true
					ui.Checkbox("chk", "Check", &val)
					str := "Text"
					ui.TextInput("txt", &str, "Placeholder")
				})
			}
		})
	})

	// Warmup frame
	h.Step()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Step()
	}
}
```

Run benchmark:
```bash
go test -bench BenchmarkFrameBudget -benchmem -run ^$
```
This ensures your frame budget remains under 16.6 ms (60 FPS) and memory allocations stay predictable.

