package ui

import (
	"testing"
	"time"

	"github.com/kirpicheff/anyui/platform"
)

func TestTableSortAndSelect(t *testing.T) {
	ctx := newTestContext(600, 400)
	cols := []TableColumn{
		{Title: "ID", WidthDp: 80, Sortable: true},
		{Title: "Name", WidthDp: 200, Sortable: true},
	}
	selected := -1
	sort := TableSort{Column: 0, Desc: false}
	var sortChanged bool
	var rowClicked int = -1

	// Клик по заголовку второй колонки (x ~ 120, y ~ 16)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 120, Y: 16})
	ctx.Render(func(c *Context) {
		ev := c.Table("tbl", cols, 50, &selected, &sort, nil, Width(400), Height(200))
		if ev.SortChanged {
			sortChanged = true
		}
	})
	if !sortChanged || sort.Column != 1 || sort.Desc {
		t.Fatalf("expected sort changed to col=1 asc, got sortChanged=%v sort=%+v", sortChanged, sort)
	}

	// Клик по второй строке тела таблицы (header=32, row0=[32..62], row1=[62..92])
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 120, Y: 16})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 120, Y: 70})
	ctx.Render(func(c *Context) {
		ev := c.Table("tbl", cols, 50, &selected, &sort, nil, Width(400), Height(200))
		if ev.RowClicked >= 0 {
			rowClicked = ev.RowClicked
		}
	})
	if rowClicked != 1 || selected != 1 {
		t.Fatalf("expected row 1 clicked and selected, got rowClicked=%d selected=%d", rowClicked, selected)
	}
}

func TestTreeViewFlattenAndToggle(t *testing.T) {
	roots := []*TreeNode{
		{
			ID:       "r1",
			Label:    "Root",
			Expanded: false,
			Children: []*TreeNode{
				{ID: "c1", Label: "Child 1"},
			},
		},
	}
	if got := len(FlattenTree(roots)); got != 1 {
		t.Fatalf("expected 1 visible node when collapsed, got %d", got)
	}
	roots[0].Expanded = true
	flat := FlattenTree(roots)
	if len(flat) != 2 || flat[1].Depth != 1 || flat[1].Node.ID != "c1" {
		t.Fatalf("unexpected flattened tree: %+v", flat)
	}

	ctx := newTestContext(400, 300)
	sel := ""
	// Клик по стрелке раскрытия первого узла (x ~ 12, y ~ 18)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 12, Y: 18})
	changed := false
	ctx.Render(func(c *Context) {
		if c.TreeView("tree", roots, &sel, Width(300), Height(200)) {
			changed = true
		}
	})
	if !changed || roots[0].Expanded {
		t.Fatalf("expected root to collapse on toggle click, changed=%v expanded=%v", changed, roots[0].Expanded)
	}
}

func TestSplitPaneAndAccordion(t *testing.T) {
	ctx := newTestContext(500, 400)
	ratio := float32(0.5)
	// Разделитель при width=400, ratio=0.5 находится примерно на x=200
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 202, Y: 100})
	ctx.Render(func(c *Context) {
		c.SplitPane("sp", SplitHorizontal, &ratio, func() {}, func() {}, Width(400), Height(200))
	})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 300, Y: 100})
	ctx.Render(func(c *Context) {
		c.SplitPane("sp", SplitHorizontal, &ratio, func() {}, func() {}, Width(400), Height(200))
	})
	if ratio <= 0.6 {
		t.Fatalf("expected split ratio to increase after dragging right, got %v", ratio)
	}

	openIdx := 0
	items := []AccordionItem{
		{ID: "a1", Title: "One", Draw: func() {}},
		{ID: "a2", Title: "Two", Draw: func() {}},
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 0, Y: 0})
	// Заголовок 1 занимает y=[0..34], контент ~ пустой (padding 16px => y=[34..50]), заголовок 2 начинается около y=50
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 60})
	ctx.Render(func(c *Context) {
		c.Accordion("acc", items, &openIdx, Width(300))
	})
	if openIdx != 1 {
		t.Fatalf("expected accordion openIdx=1, got %d", openIdx)
	}
}

func TestTabBarDragReorderAndDetach(t *testing.T) {
	ctx := newTestContext(600, 300)
	tabs := []TabItem{
		{ID: "t1", Title: "One", Closable: true},
		{ID: "t2", Title: "Two", Closable: true},
	}
	active := 0
	movedFrom, movedTo, detached := -1, -1, -1

	// Нажимаем на первую вкладку (x=40, y=18)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 18})
	ctx.Render(func(c *Context) {
		c.TabBar("tb", tabs, &active)
	})

	// Тянем вправо на вторую вкладку (x=200, y=18)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 200, Y: 18})
	ctx.Render(func(c *Context) {
		ev := c.TabBar("tb", tabs, &active)
		if ev.MovedFrom >= 0 {
			movedFrom, movedTo = ev.MovedFrom, ev.MovedTo
		}
	})
	if movedFrom != 0 || movedTo != 1 {
		t.Fatalf("expected MovedFrom=0 MovedTo=1, got from=%d to=%d", movedFrom, movedTo)
	}

	// Тянем далеко вниз (y=150) и отпускаем -> Detach
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 200, Y: 150})
	ctx.Render(func(c *Context) {
		c.TabBar("tb", tabs, &active)
	})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 200, Y: 150})
	ctx.Render(func(c *Context) {
		ev := c.TabBar("tb", tabs, &active)
		if ev.Detached >= 0 {
			detached = ev.Detached
		}
	})
	if detached != 1 {
		t.Fatalf("expected Detached=1, got %d", detached)
	}
}

func TestNumberInputSliderAndRangeSlider(t *testing.T) {
	ctx := newTestContext(400, 300)
	val := 10.0
	// Кнопка '+' в NumberInput (width=160, height=34 => '+' в x=[126..160])
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 145, Y: 16})
	ctx.Render(func(c *Context) {
		c.NumberInput("num", &val, 0, 20, 5, 0, Width(160))
	})
	if val != 15.0 {
		t.Fatalf("expected NumberInput val=15, got %v", val)
	}

	sVal := float32(0)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 110, Y: 14})
	ctx.Render(func(c *Context) {
		c.Slider("sld", &sVal, 0, 100, Width(220))
	})
	if sVal < 40 || sVal > 60 {
		t.Fatalf("expected Slider near 50, got %v", sVal)
	}

	low, high := float32(20), float32(80)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 0, Y: 0})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 20, Y: 14})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 200, Y: 14})
	ctx.Render(func(c *Context) {
		c.RangeSlider("rs", &low, &high, 0, 100, Width(220))
	})
	if low > high {
		t.Fatalf("expected low <= high, got low=%v high=%v", low, high)
	}
}

func TestColorPickerHelpers(t *testing.T) {
	col, ok := ParseHexColor("#3B82F6")
	if !ok {
		t.Fatalf("failed to parse valid hex color")
	}
	if FormatHexColor(col) != "#3B82F6" {
		t.Fatalf("unexpected formatted hex: %s", FormatHexColor(col))
	}
	h, s, v := rgbToHSV(col)
	roundtrip := hsvToRGB(h, s, v)
	if absInt(int(roundtrip.R)-int(col.R)) > 2 || absInt(int(roundtrip.G)-int(col.G)) > 2 || absInt(int(roundtrip.B)-int(col.B)) > 2 {
		t.Fatalf("HSV roundtrip mismatch: %+v vs %+v", col, roundtrip)
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func TestToastAndSpinner(t *testing.T) {
	ctx := newTestContext(500, 400)
	ctx.ShowToast("Test", "Hello", ToastSuccess, 30*time.Millisecond)
	if ctx.ToastsCount() != 1 {
		t.Fatalf("expected 1 active toast, got %d", ctx.ToastsCount())
	}
	ctx.Render(func(c *Context) {
		c.Spinner("sp", 24)
	})
	if !ctx.Animator.HasActive() {
		t.Fatalf("expected Animator to be active when Spinner/Toast is rendered")
	}
	time.Sleep(45 * time.Millisecond)
	ctx.Render(func(c *Context) {})
	if ctx.ToastsCount() != 0 {
		t.Fatalf("expected expired toast to be removed, got %d", ctx.ToastsCount())
	}
}

func TestDatePickerRender(t *testing.T) {
	ctx := newTestContext(500, 400)
	d := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 16})
	ctx.Render(func(c *Context) {
		c.DatePicker("dp", &d, Width(180))
	})
	if !ctx.openState["dp"] {
		t.Fatalf("expected DatePicker popup to open on click")
	}
}
