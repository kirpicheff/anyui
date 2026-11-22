[English](theming.md) | [Русский](../ru/theming.md)

# Themes and Styling (`Option`)

## Built-in Themes

Built-in themes available in `anyui`:
- `anyui.NavyTheme()` — default navy theme.
- `anyui.DarkTheme()` — neutral dark theme.
- `anyui.LightTheme()` — light theme.
- `anyui.SystemTheme()` — automatically picks `DarkTheme()` or `LightTheme()` based on the OS theme.

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
| `anyui.Secondary(enabled ...bool)` | Secondary button style (default) | `ui.Button("cancel", "Cancel", anyui.Secondary())` |
| `anyui.Danger(enabled ...bool)` | Destructive action style | `ui.Button("del", "Delete", anyui.Danger())` |
| `anyui.Ghost(enabled ...bool)` | Transparent background at rest | `ui.Button("more", "...", anyui.Ghost())` |
| `anyui.Bold(enabled ...bool)` | Bold font weight | `ui.Label("Heading", anyui.Bold())` |
| `anyui.Mono(enabled ...bool)` | Monospace font | `ui.Label("127.0.0.1", anyui.Mono())` |
| `anyui.Bg(c)` | Override background color | `anyui.Bg(anyui.Hex(0x1E293B))` |
| `anyui.HoverBg(c)` | Background color on hover (`Button`) | `anyui.HoverBg(anyui.Hex(0x334155))` |
| `anyui.TextColor(c)` | Override text color | `anyui.TextColor(ui.Theme().TextMuted)` |
| `anyui.HoverTextColor(c)` | Text and icon color on hover (`Button`) | `anyui.HoverTextColor(ui.Theme().Accent)` |
| `anyui.BorderColor(c)` | Override border color | `anyui.BorderColor(anyui.Hex(0x475569))` |
| `anyui.HoverBorderColor(c)` | Border color on hover (`Button`) | `anyui.HoverBorderColor(ui.Theme().Accent)` |
| `anyui.FocusBorderColor(c)` | Border color when focused (`TextInput`, `MaskedInput`, `TextArea`) | `anyui.FocusBorderColor(ui.Theme().Accent)` |
| `anyui.Radius(dp)` | Corner radius in `dp` | `anyui.Radius(10)` |
| `anyui.Width(dp)` | Fixed width in `dp` | `anyui.Width(220)` |
| `anyui.Height(dp)` | Fixed height in `dp` | `anyui.Height(42)` |
| `anyui.FontSize(dp)` | Font size in `dp` | `anyui.FontSize(16)` |
| `anyui.LetterSpacing(dp)` | Letter spacing in `dp` | `anyui.LetterSpacing(1.5)` |
| `anyui.TextShadow(dx, dy, blur, c)` | Text shadow or soft glow | `anyui.TextShadow(0, 1, 4, col)` |
| `anyui.TextAlignLeft/Center/Right(enabled ...bool)` | Horizontal text alignment (`Label`, `TextInput`) | `anyui.TextAlignCenter()` |
| `anyui.Padding(dp)` | Inner padding in `dp` | `anyui.Padding(12)` |
| `anyui.Icon(svg)` | Leading SVG icon | `anyui.Icon(saveSVG)` |
| `anyui.IconRight(svg)` | Trailing SVG icon (`TextInput`) | `anyui.IconRight(searchSVG)` |
| `anyui.Disabled(enabled ...bool)` | Disable input with muted look | `anyui.Disabled(isBusy)` |
| `anyui.ReadOnly(enabled ...bool)` | Disallow editing while allowing selection/copy | `anyui.ReadOnly()` |
| `anyui.Password(enabled ...bool)` | Mask password input (`TextInput`) | `anyui.Password()` |
| `anyui.PasswordToggle(visible ...*bool)` | Password mode with visibility toggle (`TextInput`) | `anyui.PasswordToggle()` |
| `anyui.Clearable(enabled ...bool)` | Quick clear button inside input | `anyui.Clearable()` |
| `anyui.InputFilter(fn)` | Allowed rune filter (`TextInput`, `TextArea`) | `anyui.InputFilter(unicode.IsDigit)` |
| `anyui.Searchable(enabled ...bool)` | Search box inside `Select` | `anyui.Searchable()` |
| `anyui.Indeterminate(enabled ...bool)` | Indeterminate checkbox state | `anyui.Indeterminate()` |
| `anyui.MinWidth(dp)` | Minimum widget width in `dp` | `anyui.MinWidth(120)` |
| `anyui.MaxWidth(dp)` | Maximum widget width in `dp` | `anyui.MaxWidth(400)` |
| `anyui.MinHeight(dp)` | Minimum widget height in `dp` | `anyui.MinHeight(80)` |
| `anyui.MaxHeight(dp)` | Maximum widget height in `dp` | `anyui.MaxHeight(300)` |
| `anyui.FillWidth(enabled ...bool)` | Stretch widget to full available width | `anyui.FillWidth()` |
| `anyui.FillHeight(enabled ...bool)` | Stretch widget to full available height | `anyui.FillHeight()` |
| `anyui.AutoHeight(enabled ...bool)` | Auto-size container (`Card`) height to content | `anyui.AutoHeight()` |
| `anyui.WordWrap(enabled ...bool)` | Wrap text to width (`Label`, `SelectableLabel`) | `anyui.WordWrap()` |
| `anyui.Ellipsis(enabled ...bool)` | Truncate text with `...` (`Label`) | `anyui.Ellipsis()` |
| `anyui.LineNumbers(enabled ...bool)` | Show line numbers in `TextArea` | `anyui.LineNumbers()` |
| `anyui.Error(text)` | Validation error state and caption below input | `anyui.Error("Invalid format")` |
| `anyui.ScrollToIndex(idx)` | Programmatic scroll to row (`ListView*`, `Table*`) | `anyui.ScrollToIndex(targetIdx)` |
| `anyui.ScrollToID(id)` | Programmatic scroll to anchor (`ScrollView`) | `anyui.ScrollToID("sec_2")` |
| `anyui.Reorderable(enabled ...bool)` | Enable drag-and-drop reorder (`ListView*`, `Table*`, `TreeView*`) | `anyui.Reorderable()` |
| `anyui.AllowReparent(enabled ...bool)` | Allow reparenting and `DropInside` in `TreeView*` | `anyui.AllowReparent()` |
| `anyui.OnReorder(fn)` | Drag-and-drop `DragEvent` handler | `anyui.OnReorder(func(ev anyui.DragEvent) { ... })` |

