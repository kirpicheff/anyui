[English](widgets.md) | [Русский](../ru/widgets.md)

# Widget & Layout Reference (`anyui`)

All widgets are invoked inside `drawFn(ui *anyui.UI)` in Immediate Mode. Use functional `anyui.Option` values to customize appearance and behavior (see [theming.md](theming.md)).

---

## Buttons & Toggles

### `Button` and `IconButton`
- **Signatures:**
  - `ui.Button(id, label string, opts ...Option) bool`
  - `ui.IconButton(id, svg string, opts ...Option) bool`
- **Options:** `Primary(enabled ...bool)`, `Secondary()`, `Danger()`, `Ghost(enabled ...bool)`, `Bg(c)`, `HoverBg(c)`, `BorderColor(c)`, `HoverBorderColor(c)`, `TextColor(c)`, `HoverTextColor(c)`, `LetterSpacing(dp)`, `TextShadow(ox, oy, blur, c)`, `Icon(svg)`, `Width(dp)`, `Height(dp)`, `Radius(dp)`, `Disabled(enabled ...bool)`. When `Bg` is fully transparent (`A == 0`), hover and pressed backgrounds stay transparent (unless `HoverBg` is specified).
- **Example:**
```go
if ui.Button("save", "Save", anyui.Primary(), anyui.Icon(saveSVG)) {
	// handle click
}
if ui.IconButton("close", closeSVG, anyui.Ghost()) {
	// icon clicked
}
```

### `Checkbox`
- **Signature:** `ui.Checkbox(id, label string, checked *bool, opts ...Option) bool`
- **Options:** `Indeterminate(enabled ...bool)`, `Disabled(enabled ...bool)`, `FontSize(dp)`, `TextColor(c)`, `Radius(dp)`.
- **Example:**
```go
ui.Checkbox("agree", "I accept the terms", &agreed)
```

### `RadioButton` and `RadioGroup`
- **Signatures:**
  - `ui.RadioButton(id, label string, selected *string, value string, opts ...Option) bool`
  - `ui.RadioGroup(id string, options []string, selected *int, opts ...Option) bool`
- **Example:**
```go
// Key-based individual radio buttons
ui.RadioButton("mode_perf", "Performance", &mode, "perf")
ui.RadioButton("mode_eco", "Power saving", &mode, "eco")

// Radio group
ui.RadioGroup("theme", []string{"Navy", "Dark", "Light"}, &themeIdx)
```

### `Toggle`
- **Signature:** `ui.Toggle(id, label string, value *bool, opts ...Option) bool`
- **Example:**
```go
ui.Toggle("autostart", "Launch on startup", &autoStart)
```

---

## Text Input

### `TextInput`
- **Signature:** `ui.TextInput(id string, value *string, placeholder string, opts ...Option) bool`
- **Features:** accurate click-to-position cursor, mouse selection, `Shift + Arrows`, word navigation/deletion (`Ctrl + Left/Right`, `Ctrl + Shift + Left/Right`, `Ctrl + Backspace/Delete`), double-click (word) and triple-click (all text), `Ctrl+A / C / V / X`, `Undo` / `Redo` (`Ctrl+Z` / `Ctrl+Y` / `Ctrl+Shift+Z`), built-in right-click context menu.
- **Options:** `Password()`, `PasswordToggle(visible ...*bool)`, `ReadOnly(enabled ...bool)`, `Clearable(enabled ...bool)`, `Icon(svg)`, `IconRight(svg)`, `InputFilter(func(rune) bool)`, `Error(string)`, `FocusBorderColor(c)`, `TextAlignCenter()`, `TextAlignLeft()`, `TextAlignRight()`, `Disabled(enabled ...bool)`, `Width(dp)`.
- **Built-in Password Visibility Toggle (PasswordToggle):**
  - `anyui.PasswordToggle()` or `anyui.PasswordToggle(&showPassword)` renders a clickable eye icon **inside** the right edge of the input field (automatically reserving inner `rightOffset`).
  - Clicking the eye toggles symbol masking (`••••••••` / plain text) and switches the icon between eye and eye-off.
  - Allows login and password inputs to share the exact same width without external buttons.
- **Example:**
```go
// Password input with built-in eye toggle (visibility state controlled by showPassword):
ui.TextInput("pass", &password, "Enter password",
	anyui.PasswordToggle(&showPassword),
	anyui.Clearable(true),
	anyui.Width(280),
)

// Or with automatic internal state management:
ui.TextInput("pass_auto", &password, "Password",
	anyui.PasswordToggle(),
	anyui.Width(280),
)

ui.TextInput("email", &email, "Email...",
	anyui.Width(280),
	anyui.Error(emailErr),
)
```

### `MaskedInput`
- **Signature:** `ui.MaskedInput(id string, value *string, mask string, opts ...Option) bool`
- **Purpose:** single-line masked input (`0`/`9` — digit, `A` — letter, `*` — alphanumeric, other characters — fixed literals). Supports `Undo`/`Redo`, right-click context menu, and `Error(string)`.
- **Example:**
```go
ui.MaskedInput("phone", &phone, "+1 (000) 000-0000", anyui.Width(200), anyui.Mono())
ui.MaskedInput("date", &dateStr, "00.00.0000", anyui.Width(140), anyui.Mono())
ui.MaskedInput("ip", &ipStr, "000.000.000.000", anyui.Width(170), anyui.Mono())
```

### `TextArea`
- **Signature:** `ui.TextArea(id string, value *string, placeholder string, opts ...Option) bool`
- **Features:** multi-line editing with wrapping, vertical scroll and caret auto-scroll, `Up`/`Down`/`Home`/`End`/`Ctrl+Home`/`Ctrl+End`, word navigation (`Ctrl + Arrows`), multi-line mouse selection, double/triple-click, `Undo`/`Redo`, optional line numbers (`LineNumbers(true)`), right-click context menu, and `Error(string)`.
- **Example:**
```go
ui.TextArea("notes", &notes, "Comment...",
	anyui.Height(140),
	anyui.LineNumbers(true),
)
```

### `Label`, `SelectableLabel`, and `Hyperlink`
- **Signatures:**
  - `ui.Label(text string, opts ...Option)` — static text with `WordWrap(enabled ...bool)` (word boundary wrapping by spaces), `Ellipsis(true)`, text alignment (`TextAlignCenter()`, `TextAlignLeft()`, `TextAlignRight()`), `LetterSpacing(dp)`, and `TextShadow(offsetX, offsetY, blur, col)` options.
  - `ui.SelectableLabel(id, text string, opts ...Option)` — selectable static text (including double/triple-click), `Ctrl+C`/`Ctrl+A`, and right-click context menu.
  - `ui.Hyperlink(id, text string, opts ...Option) bool` — clickable link with hover underline and hand cursor.
- **Example:**
```go
ui.Label("Long text with word boundary wrapping...", anyui.WordWrap(true))
ui.Label("Centered card text", anyui.WordWrap(true), anyui.TextAlignCenter(), anyui.FillWidth())
ui.Label("Truncated heading", anyui.Width(220), anyui.Ellipsis(true))
ui.SelectableLabel("session_id", "ID: 8f4c-2a19-b70e | IP: 10.24.18.5", anyui.Mono())
if ui.Hyperlink("docs_link", "Open Documentation") {
	// handle click
}
```

---

## Lists & Navigation

### `Select` and `ComboBox`
- **Item Model:** `anyui.SelectItem{Key string, Label string, Icon string}` (or convenience helper `anyui.SelectItems("Item 1", "Item 2")`).
- **Signatures:**
  - `ui.Select(id string, items []anyui.SelectItem, selected *string, opts ...Option) bool`
  - `ui.ComboBox(id string, items []anyui.SelectItem, selected *string, opts ...Option) bool` (with search/filter input)
- **Example:**
```go
countries := anyui.SelectItems("United States", "Germany", "Japan")
ui.ComboBox("country", countries, &countryKey, anyui.Width(260))
```

### `ListView`
- **Signature:** `ui.ListView(id string, count int, itemHeightDp int, selected *int, drawItem func(index int, area Rect, selected bool), opts ...Option) bool`
- **Features:** virtualized rendering (only visible rows are drawn), mouse wheel scrolling.
- **Example:**
```go
ui.ListView("files", len(files), 36, &selectedIdx, func(i int, area anyui.Rect, sel bool) {
	// draw row i inside area
}, anyui.Height(240))
```

### `Tabs` and `TabBar`
- **Signatures:**
  - `ui.Tabs(id string, items []string, active *int, orient anyui.Orientation, opts ...Option) bool`
  - `ui.TabBar(id string, tabs []TabItem, active *int, opts ...Option) TabBarEvent`
- **`TabBar` Example:**
```go
ev := ui.TabBar("editor_tabs", tabs, &activeTab)
if ev.Added {
	// add new tab
}
if ev.Closed >= 0 {
	// close tab ev.Closed
}
```

---

## Containers, Indicators & Overlays

### `ScrollView`
- **Signature:** `ui.ScrollView(id string, fn func(), opts ...Option)`
- **Features:** vertical and horizontal scrolling via mouse wheel and draggable scrollbars.
- **Auto-Fill:** by default `ScrollView` fills the remaining container space automatically; pass `anyui.Width(...)`, `anyui.Height(...)`, `anyui.FillWidth()`, or `anyui.FillHeight()` to customize dimensions.

### Container Coordinates (`CurrentArea` vs `CurrentAreaPx`)
- **`ui.CurrentArea() Rect`:** returns the container's rectangle strictly in logical pixels (`Dp`).
- **`ui.CurrentAreaPx() Rect`:** returns the container's rectangle in physical pixels (`Px`), used for low-level `ui.Canvas()` operations.

### Modal Dialogs, Overlays, and Floating Panels

AnyUI provides a dedicated set of containers for overlays and floating UI elements.

#### `Modal` (Modal Dialog Windows)
- **Signature:** `ui.Modal(id string, open *bool, title string, fn func(), opts ...Option)`
- **Use case:** confirmation dialogs, login forms, alert popups, settings dialogs.
- **Behavior:**
  - Automatically centers the dialog box and dims the backdrop across the rest of the window (`rgba(0, 0, 0, 140)`).
  - Blocks clicks and keyboard focus through the backdrop onto widgets beneath it, while all input inside the modal (`fn`) remains fully active.
  - Includes a header with title and close button `×`, automatically closes on `Escape` key press (`*open = false`).
  - Options: `anyui.Width(wDp)` (default 400 dp), `anyui.Height(hDp)` (default 300 dp).
  - **Important:** always use `ui.Modal` for modal dialogs. Do not construct homemade modals with `PushOverlay(fullRect)` — this marks the entire screen as input-blocked on subsequent frames.

```go
var showLoginModal bool

if ui.Button("btn_open_login", "Log In") {
    showLoginModal = true
}

ui.Modal("modal_login", &showLoginModal, "Authentication", func() {
    ui.Column(12, func() {
        ui.Label("Enter your credentials:")
        ui.TextInput("login_email", &email, "Email", anyui.FillWidth())
        ui.TextInput("login_pass", &pass, "Password", anyui.Password(), anyui.FillWidth())
        ui.RowWith(anyui.AlignCenter, anyui.JustifyEnd, 8, 36,
            func() {
                if ui.Button("btn_cancel", "Cancel", anyui.Secondary()) {
                    showLoginModal = false
                }
            },
            func() {
                if ui.Button("btn_submit", "Log In", anyui.Primary()) {
                    showLoginModal = false
                }
            },
        )
    })
}, anyui.Width(420), anyui.Height(280))
```

#### `Popover` (Floating Card Anchored to a Widget)
- **Signature:** `ui.Popover(id string, open *bool, anchor Rect, fn func(), opts ...Option)`
- **Use case:** profile dropdowns, filter menus, quick action popups.
- **Behavior:**
  - Anchors to the given `anchor` rectangle (e.g. directly below a button).
  - Automatically flips upward or shifts horizontally if it would overflow the window boundary.
  - Closes on outside click or `Escape`.

```go
var popoverOpen bool

btnArea := ui.CurrentArea()
if ui.Button("btn_filters", "Filters") {
    popoverOpen = !popoverOpen
    btnArea = ui.CurrentArea()
}

ui.Popover("pop_filters", &popoverOpen, btnArea, func() {
    ui.Padding(12, func() {
        ui.Column(8, func() {
            ui.Toggle("tog_active", "Active only", &onlyActive)
            ui.Toggle("tog_archived", "Include archived", &includeArchived)
        })
    })
}, anyui.Width(220), anyui.Height(100))
```

#### `FloatingPanel` (Movable Floating Window)
- **Signature:** `ui.FloatingPanel(id, title string, open *bool, fn func(), opts ...Option)`
- **Use case:** floating toolbars, color palettes, property inspectors.
- **Behavior:**
  - Can be dragged by its header using the mouse (`CursorMove`).
  - Supports collapsing the panel body via header toggle and closing via `×`.
  - Persists dragged window coordinates between frames by `id`.
  - Options: `anyui.Width(wDp)`, `anyui.Height(hDp)`, `anyui.Pos(xDp, yDp)`.

#### `Drawer` (Slide-Out Panel from Any Screen Edge)
- **Signature:** `ui.Drawer(id string, open *bool, side DrawerSide, fn func(), opts ...Option)`
- **Use case:** side navigation menus, shopping carts, item details, top banners, bottom sheets.
- **Sides:**
  - `anyui.DrawerLeft` — slides out from the left along full window height (default width 280 dp, option `anyui.Width(wDp)`).
  - `anyui.DrawerRight` — slides out from the right along full window height (default width 280 dp, option `anyui.Width(wDp)`).
  - `anyui.DrawerTop` — slides down from the top across full window width (default height 240 dp, option `anyui.Height(hDp)`). If `anyui.Width(wDp)` is set, horizontally centered.
  - `anyui.DrawerBottom` — slides up from the bottom (bottom sheet, default height 240 dp, option `anyui.Height(hDp)`). If `anyui.Width(wDp)` is set, horizontally centered.
- **Behavior:**
  - Smooth slide animation.
  - Dims the remaining screen area and automatically closes when clicking the dimmed backdrop outside the panel or pressing `Escape`.

#### `ContextMenu` (Right-Click Popup Menu)
- **Signature:** `ui.ContextMenu(id string, anchor Rect, items []ContextMenuItem)`
- **Behavior:** opens at click location or anchor rect, supports separators (`Separator: true`), icons, `Disabled` items, keyboard shortcuts, and submenus. Closes on outside click or `Escape`.

#### `Tooltip` (Hover Tooltip)
- **Signature:** `ui.Tooltip(text string, anchor Rect)`
- **Behavior:** renders a rounded floating card with shadow above the UI near `anchor` on hover, automatically sized to content.

#### `PushOverlay` and `LayoutRect` (Low-Level Overlays)
- **`ui.PushOverlay(bounds Rect, drawFn func())`**:
  - Defers `drawFn` execution to the end of the frame (`endFrame`), ensuring the element renders **on top of all other UI** without being clipped by parent containers (`ScrollView`, `Card`).
  - The `bounds` rectangle is recorded to block input clicks on the underlying layer on the next frame (`IsInputBlocked()`).
  - **Rule:** use `PushOverlay` strictly for custom floating controls (custom dropdowns, pickers, menus). Never pass a full-window `fullRect` to emulate a modal dialog — use `ui.Modal()` instead.
- **`ui.LayoutRect(rect Rect, fn func())`**:
  - Creates an isolated vertical layout box strictly matching `rect` with clipping enabled (`PushClip`).
  - Ideal for arranging widgets inside custom `PushOverlay` callbacks, floating windows, or custom cards.

### `Separator`, `Badge`, `StatusDot`, `ProgressBar`
```go
ui.Separator()
ui.Badge("BETA", anyui.Primary())
ui.StatusDot(theme.Success, "Connected")
ui.ProgressBar(0.65, anyui.Width(240))
```

---

## Advanced Controls

### `Table`
Table with column headers, resizable columns, sorting, and virtualized rows:
```go
cols := []anyui.TableColumn{
	{Title: "ID", WidthDp: 64, MinWidthDp: 48, Sortable: true},
	{Title: "Name", WidthDp: 200, Sortable: true},
}
ev := ui.Table("tbl", cols, len(rows), &selectedRow, &sortState, func(row, col int, area anyui.Rect, sel bool) {
	ui.DrawText(area.X+ui.Scale(8).Int(), area.Y+ui.Scale(7).Int(), rows[row].Name, 12, ui.Theme().TextPrimary)
}, anyui.Height(240), anyui.Reorderable(), anyui.OnReorder(func(ev anyui.DragEvent) {
	// ev.FromWidget, ev.ToWidget, ev.FromKey, ev.ToKey, ev.Drop (DropAbove / DropBelow)
}))
```

### `TreeView`
Expandable hierarchical tree with virtualization. Supports node reordering (`Reorderable`), drop zones `DropAbove`, `DropBelow`, `DropInside` (nesting inside folders with row highlight), visual drop insertion indicator, translucent ghost drag preview, and cross-widget drag & drop:
```go
ui.TreeView("tree", roots, &selectedNodeID, anyui.Height(220), anyui.Reorderable(), anyui.OnReorder(func(ev anyui.DragEvent) {
	// ev.FromWidget, ev.ToWidget — source and target widget IDs
	// ev.FromKey, ev.ToKey — dragged node ID and target node ID
	// ev.Drop — anyui.DropAbove, anyui.DropBelow, or anyui.DropInside (nested inside folder)
}))
```

### `SplitPane`
Splits an area into 2 panes (`SplitHorizontal` or `SplitVertical`) with a draggable divider:
```go
ui.SplitPane("split", anyui.SplitHorizontal, &ratio, func() {
	// First pane
}, func() {
	// Second pane
}, anyui.Height(260))
```

### `Collapsible` and `Accordion`
Collapsible sections:
```go
ui.Collapsible("sec1", "Advanced", &expanded, func() {
	ui.Label("Section content")
})

// Accordion with map of opened section IDs (supports opening multiple sections simultaneously)
openSections := map[string]bool{"sec1": true}
ui.Accordion("acc", items, openSections)
```

### `TabBar` Drag & Detach
`TabBarEvent` returns `MovedFrom`, `MovedTo` when reordering tabs via drag, and `Detached` when a tab is dragged out of the bar (for opening in a separate window via `app.OpenWindow`).

### `NumberInput`, `Slider`, `RangeSlider`
```go
ui.NumberInput("qty", &qty, 0, 100, 1, 0, anyui.Width(150))
ui.Slider("vol", &vol, 0, 100, anyui.Width(240))
ui.RangeSlider("rng", &low, &high, 0, 100, anyui.Width(240))
```

### `ColorPicker` and `DatePicker`
```go
ui.ColorPicker("cp", &colorVal, anyui.Width(150))
ui.DatePicker("dp", &dateVal, anyui.Width(170))
```

### `Toast` and `Spinner`
- **Toast positioning:** `ui.SetToastPosition(pos)` (`ToastBottomRight`, `ToastBottomLeft`, `ToastTopRight`, `ToastTopLeft`, `ToastBottomCenter`, `ToastTopCenter`) and `ui.SetToastOffset(xDp, yDp)` (window edge offsets in `dp`, defaults to `16, 16`).
- Toast notifications are rendered on top of the UI directly without blocking mouse clicks or focus in other parts of the screen.
```go
ui.Spinner("loader", 28)
ui.SetToastPosition(anyui.ToastBottomRight)
ui.SetToastOffset(16, 48) // lift above a footer or status bar
ui.ShowToast("Done", "Data saved", anyui.ToastSuccess)
```

---

## Advanced Layout System

### `RowFlex`, `ColumnFlex`, and `Expand`
Proportional space distribution via `FlexItem{Weight, FixedDp, Fn}`:
```go
ui.RowFlex(10, 38,
	anyui.FlexItem{FixedDp: 100, Fn: func() { ui.Button("b1", "Fixed") }},
	anyui.FlexItem{Weight: 1, Fn: func() { ui.TextInput("in1", &s1, "1x") }},
	anyui.FlexItem{Weight: 2, Fn: func() { ui.TextInput("in2", &s2, "2x") }},
)
ui.ColumnFlex(10,
	anyui.FlexItem{Weight: 1, Fn: func() { /* top */ }},
	anyui.FlexItem{Weight: 2, Fn: func() { /* bottom */ }},
)
ui.Expand(func() {
	// fills all remaining space in the current container
})
```

### `RowWith` and `ColumnWith`
Extended alignment (`AlignStart`, `AlignCenter`, `AlignEnd`, `AlignStretch`) and justification (`JustifyStart`, `JustifyCenter`, `JustifyEnd`, `JustifySpaceBetween`, `JustifySpaceAround`):
```go
ui.RowWith(anyui.AlignCenter, anyui.JustifySpaceBetween, 10, 40,
	func() { ui.Label("Title", anyui.Bold()) },
	func() { ui.Button("act", "Action", anyui.Primary()) },
)
```

### `Grid` and `FormLayout` / `FormRow`
`cols`-column grid and aligned "Label — Field" form layout:
```go
ui.Grid(3, 12, 12, 44, func() {
	// child elements are distributed across 3 columns automatically
})

ui.FormLayout(120, 12, 8, func() {
	ui.FormRow("Host:", func() { ui.TextInput("host", &host, "localhost") })
	ui.FormRow("Port:", func() { ui.TextInput("port", &port, "8080") })
})
```

### `Flow` and Auto-Height `Card`
`Flow` wraps items onto the next line when width is exceeded, and `Card` with `anyui.AutoHeight()` sizes itself to its content:
```go
ui.Card(func() {
	ui.Flow(8, 8, func() {
		ui.Badge("Go", anyui.Primary())
		ui.Badge("Win32")
		ui.Badge("Software Renderer")
	})
}, anyui.AutoHeight(), anyui.Padding(12))
```

---

## Extended Mouse, Keyboard & Focus Events

### `DoubleClick` and `MiddleClick`
- `UI` and `ui.Context` expose `MousePosDp() (int, int)` (cursor position in logical dp units), `MousePosPx() (int, int)` (cursor position in physical window pixels), `MousePressed() bool` (LMB pressed this frame), `MouseDown() bool` (LMB held down), `DoubleClicked() bool`, and `MiddleClicked() bool` (`MiddleMousePressed` / `MiddleMouseDown` / `MiddleMouseReleased` on `ui.Context`).
- `TableEvent` returns `RowDoubleClicked` and `RowMiddleClicked` (`-1` if none):
```go
ev := ui.Table("tbl", cols, len(rows), &sel, &sort, drawCell)
if ev.RowDoubleClicked >= 0 {
	// Open item on double click
}
if ev.RowMiddleClicked >= 0 {
	// Action on middle click
}
```
- In `TabBar`, middle-clicking a tab with `Closable: true` closes it (`ev.Closed`).
- In `TreeView`, double-clicking a parent node toggles `Expanded`.

### Horizontal Scrolling (`WheelX` and `Shift + WheelY`)
- Supports horizontal mouse wheel (`WheelX`) and `Shift + WheelY`.
- `ScrollView`, `Table`, and `TabBar` all support horizontal wheel scrolling.

### Mouse Capture (`SetCapture` / `ReleaseCapture`)
- Mouse capture ensures dragging `Slider`, `RangeSlider`, `SplitPane`, scrollbars, `Table` column dividers, `TabBar` tabs, and text selection continues smoothly even when the cursor leaves the window.
- On window focus loss (`Alt+Tab`), `EventFocusLost` and `EventCaptureLost` reset `FocusID`, `MouseDown`, and `ActiveID` cleanly.

### Keyboard Navigation (`Navigable`)
`ListView`, `Table`, `TreeView`, and `Select`/`ComboBox` implement `ui.Navigable` for consistent keyboard navigation (`Up`, `Down`, `Left`, `Right`, `PageUp`, `PageDown`, `Home`, `End`, `Space`, `Enter`).

### Declarative Programmatic Navigation (`ScrollToIndex`, `ScrollToID`, `Anchor`)
AnyUI provides declarative scrolling directly within the frame rendering pipeline without deferred queues or async channels:
- **`anyui.ScrollToIndex(idx int)`** for `ListView` and `Table`:
  - Scrolls virtualized list or table so that the item at `idx` appears at the start of the visible viewport (clamped to `maxSc`).
  - Evaluated on index changes while allowing continued manual mouse wheel scrolling without jitter. Pass `-1` to reset.
  ```go
  ui.ListView("files", len(files), 36, &selectedIdx, drawItem, anyui.ScrollToIndex(50))
  ui.Table("tbl", cols, len(rows), &selectedRow, &sort, drawCell, anyui.ScrollToIndex(120))
  ```
- **`anyui.ScrollToID(id string)`** and **`ui.Anchor(id string)`** for `ScrollView`:
  - `anyui.ScrollToID(id)` scrolls `ScrollView` directly to the child widget with the given identifier (e.g., `ui.Button("buy_btn", ...)` or an anchor `ui.Anchor("pricing_section")`).
  - `ui.Anchor(id string)` creates a zero-size anchor marker anywhere in the layout for quick navigation to headings or sections.
  ```go
  ui.ScrollView("page", func() {
      ui.Label("Page Header", anyui.FontSize(18))
      // ...
      ui.Anchor("pricing_section")
      ui.Label("Pricing", anyui.FontSize(16), anyui.Bold())
      // ...
      ui.Button("buy_btn", "Buy Now", anyui.Primary())
  }, anyui.ScrollToID("pricing_section"))
  ```


---

## Desktop Controls, Charts & Multi-Selection

### `MenuBar`, `Toolbar`, `StatusBar`, `Breadcrumbs`, `Pagination`
```go
ui.MenuBar("mb", []anyui.MenuBarItem{
	{Label: "File", Items: []anyui.MenuBarEntry{
		{Label: "New", Shortcut: "Ctrl+N", OnClick: func() {}},
		{Separator: true},
		{Label: "Export", SubItems: []anyui.MenuBarEntry{{Label: "PNG"}}},
	}},
})
ui.Toolbar("tb", []anyui.ToolbarItem{
	{ID: "save", Label: "Save", Tooltip: "Save file", OnClick: func() {}},
})
clickedID := ui.Breadcrumbs("bc", []anyui.BreadcrumbItem{
	{ID: "home", Label: "Home"},
	{ID: "section", Label: "Section"},
})
_ = clickedID
ui.Pagination("pg", &page, 10)
ui.StatusBar("sb", []anyui.StatusBarSection{{Text: "Ready", Flex: 1}, {Text: "UTF-8", WidthDp: 70}})
```

### `LogView` and `VirtualTextView`
- `ui.LogView(id, entries, &followTail, opts...)`: virtualized log viewer with log levels (`LogInfo`, `LogWarn`, `LogError`, `LogDebug`), built-in filter, `Follow tail` mode, selection (`Click`, `Shift+Click`, `Ctrl+A`), and copy (`Ctrl+C` or Copy button).
- `ui.VirtualTextView(id, lines, opts...)`: virtualized line array viewer.

### Charts (`Sparkline`, `LineChart`, `BarChart`)
- `ui.Sparkline(id, values, opts...)`: compact trend sparkline with hover tooltip.
- `ui.LineChart(id, labels, series, opts...)` and `ui.BarChart(id, labels, series, opts...)`: charts with axes, grid, legend, and hover tooltips.

### Extended `DatePicker`, `TimePicker`, `DateTimePicker`
- `DatePicker`: clicking the month or year in the header opens a 12-month or 12-year picker grid; includes a "Today" button; double-clicking the field enables manual `YYYY-MM-DD` input.
- `ui.TimePicker(id, &t, opts...)` and `ui.DateTimePicker(id, &dt, opts...)`: `HH:MM:SS` time selection with `+`/`-` buttons and mouse wheel support.

### `MultiSelect` / `CheckComboBox` and `TagInput`
- `ui.MultiSelect(id, items, selectedIDs, opts...)` (alias `CheckComboBox`): dropdown with checkboxes using `map[string]bool`, search, "Select All / Clear", and tag/summary display. Accepts `[]anyui.SelectItem`.
- `ui.TagInput(id, &tags, placeholder, opts...)`: tag editor (add with `Enter` or `,`, remove with `x` or `Backspace`).

### Multi-Selection Variants: `ListViewEx`, `TreeViewEx`, `TableEx`
- `ui.ListViewEx`: multi-select (`Ctrl+Click`, `Shift+Click`, `Ctrl+A`) and optional checkboxes.
- `ui.TreeViewEx` + `anyui.SetTreeNodeChecked` / `anyui.SyncTreeCheckState`: multi-select and cascading checkboxes with automatic `Indeterminate` parent states.
- `ui.TableEx`: multi-row selection (`Ctrl+Click`, `Shift+Click`, `Ctrl+A`), checkbox column with "Select All" header checkbox, `FrozenCols`, inline cell editing on double-click, and `AutoFitOnDoubleClick`.

---

## Custom TitleBar and Control Localization

### `TitleBar` and `TitleBarHitTest`
Ready-to-use custom title bar widget for `Frameless: true` windows (consistent across Windows 7/10/11/Server, supports dragging, double-click to maximize, icon, custom slots, and `Minimize` / `Maximize` / `Close` buttons):

```go
var currentUI *anyui.UI

app := anyui.NewApp(anyui.Config{
	Title:     "Custom App",
	Frameless: true,
	HitTest: func(x, y int) anyui.HitTestResult {
		if currentUI != nil {
			return currentUI.TitleBarHitTest(x, y)
		}
		return anyui.HitClient
	},
})

app.Run(func(ui *anyui.UI) {
	currentUI = ui
	ui.TitleBar("main_title", "Custom App", anyui.TitleBarConfig{
		IconSVG: appIconSVG,
		LeftContent: func() {
			ui.Button("menu_btn", "File", anyui.Ghost(), anyui.Height(26))
		},
	})
})
```

### Control String Localization (`Strings`)
All built-in strings (`DatePicker`, `TimePicker`, `MultiSelect`, `Select`, `LogView`, text context menus) can be customized via `anyui.Strings` (`RussianStrings()`, `EnglishStrings()`):

```go
ui.SetStrings(anyui.EnglishStrings())
```

---

## Layout Helpers & State (`IDScope`, `Card`, `Page`, `Remember`)

### Automatic IDs and `IDScope`
Pass `""` instead of a manual `id` to any widget for automatic counter-based ID generation. Use `ui.IDScope(key, func() { ... })` inside loops and reusable components:

```go
for _, item := range items {
	ui.IDScope(item.ID, func() {
		ui.Row(8, 36, func() {
			ui.Label(item.Name)
			if ui.Button("", "Delete", anyui.Danger()) {
				deleteItem(item.ID)
			}
		})
	})
}
```

### `Card` and `Page`
- `ui.Card(fn, opts...)` sizes its height to content by default (`AutoHeight`). Pass `anyui.FillHeight()` to stretch it to the container height.
- `ui.Page(cfg, headerFn, bodyFn, footerFn)` builds a standard layout with a fixed header, scrollable `ScrollView` body, and fixed footer:

```go
ui.Page(anyui.PageConfig{HeaderDp: 44, FooterDp: 48},
	func() { ui.Label("Settings", anyui.FontSize(18), anyui.Bold()) },
	func() { /* scrollable form */ },
	func() { ui.Button("", "Save", anyui.Primary()) },
)
```

### Background Loading Cache (`Remember`)
`anyui.Remember(ui, key, loader)` runs `loader` in a background goroutine on first access or when `key` changes, and automatically triggers a redraw via `app.Dispatch`. Reset the cache with `ui.InvalidateRemember(key)` or `ui.ClearRemember()`.

