[English](theming.md) | [Русский](../ru/theming.md)

# Themes and Styling (`Option`)

## Built-in Themes

Three built-in themes are available in `anyui`:
- `anyui.NavyTheme()` — default navy theme.
- `anyui.DarkTheme()` — neutral dark theme.
- `anyui.LightTheme()` — light theme.

You can set a theme when creating the app via `anyui.Config{Theme: ...}` or switch it at runtime using `ui.SetTheme(theme)` / `win.SetTheme(theme)`.

## Custom Themes

The `anyui.Theme` struct holds all base colors and corner radii:

```go
custom := anyui.NavyTheme()
custom.Accent = anyui.Hex(0x10B981)
custom.AccentHover = anyui.Hex(0x059669)
custom.RadiusSm = 6
```

## Functional Widget Options (`Option`)

Every widget accepts variadic `opts ...anyui.Option` to override appearance and sizing locally.

> [!NOTE]
> Boolean modifiers (`Bold`, `Primary`, `Ghost`, `Mono`, `Disabled`, `ReadOnly`, `Clearable`, `Indeterminate`, `WordWrap`) accept variadic `(enabled ...bool)` defaulting to `true`. They can be called without arguments (`anyui.Bold()`) or passed a boolean condition directly (`anyui.Disabled(isBusy)`).

| Option | Purpose | Example |
| --- | --- | --- |
| `anyui.Primary(enabled ...bool)` | Primary accent button style | `ui.Button("ok", "OK", anyui.Primary())` |
| `anyui.Secondary()` | Secondary button style (default) | `ui.Button("cancel", "Cancel", anyui.Secondary())` |
| `anyui.Danger()` | Destructive action style | `ui.Button("del", "Delete", anyui.Danger())` |
| `anyui.Ghost(enabled ...bool)` | Transparent background at rest | `ui.Button("more", "...", anyui.Ghost())` |
| `anyui.Bold(enabled ...bool)` | Bold font weight | `ui.Label("Heading", anyui.Bold())` |
| `anyui.Mono(enabled ...bool)` | Monospace font | `ui.Label("127.0.0.1", anyui.Mono())` |
| `anyui.Bg(c)` | Override background color | `anyui.Bg(anyui.Hex(0x1E293B))` |
| `anyui.TextColor(c)` | Override text color | `anyui.TextColor(ui.Theme().TextMuted)` |
| `anyui.BorderColor(c)` | Override border color | `anyui.BorderColor(anyui.Hex(0x475569))` |
| `anyui.Radius(dp)` | Corner radius in `dp` | `anyui.Radius(10)` |
| `anyui.Width(dp)` | Fixed width in `dp` | `anyui.Width(220)` |
| `anyui.Height(dp)` | Fixed height in `dp` | `anyui.Height(42)` |
| `anyui.FontSize(dp)` | Font size in `dp` | `anyui.FontSize(16)` |
| `anyui.Padding(dp)` | Inner padding in `dp` | `anyui.Padding(12)` |
| `anyui.Icon(svg)` | SVG icon | `anyui.Icon(saveSVG)` |
| `anyui.Disabled(enabled ...bool)` | Disable input with muted look | `anyui.Disabled(isBusy)` |
| `anyui.ReadOnly(enabled ...bool)` | Disallow editing while allowing selection/copy | `anyui.ReadOnly()` |
| `anyui.Clearable(enabled ...bool)` | Quick clear button inside input | `anyui.Clearable()` |
| `anyui.Indeterminate(enabled ...bool)` | Indeterminate checkbox state | `anyui.Indeterminate()` |
| `anyui.MinWidth(dp)` | Minimum widget width in `dp` | `anyui.MinWidth(120)` |
| `anyui.MaxWidth(dp)` | Maximum widget width in `dp` | `anyui.MaxWidth(400)` |
| `anyui.MinHeight(dp)` | Minimum widget height in `dp` | `anyui.MinHeight(80)` |
| `anyui.MaxHeight(dp)` | Maximum widget height in `dp` | `anyui.MaxHeight(300)` |
| `anyui.FillWidth()` | Stretch widget to full available width | `anyui.FillWidth()` |
| `anyui.FillHeight()` | Stretch widget to full available height | `anyui.FillHeight()` |
| `anyui.AutoHeight()` | Auto-size container (`Card`) height to content | `anyui.AutoHeight()` |
| `anyui.Flex(weight)` | Proportional flex weight | `anyui.Flex(2)` |
| `anyui.WordWrap(enabled ...bool)` | Wrap text to width (`Label`, `SelectableLabel`) | `anyui.WordWrap()` |
| `anyui.Ellipsis(v)` | Truncate text with `...` (`Label`) | `anyui.Ellipsis(true)` |
| `anyui.LineNumbers(v)` | Show line numbers in `TextArea` | `anyui.LineNumbers(true)` |
| `anyui.Error(text)` | Validation error state and caption below input | `anyui.Error("Invalid format")` |

