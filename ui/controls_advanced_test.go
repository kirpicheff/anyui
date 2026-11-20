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

	openIDs := map[string]bool{"a1": true}
	items := []AccordionItem{
		{ID: "a1", Title: "One", Draw: func() {}},
		{ID: "a2", Title: "Two", Draw: func() {}},
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 0, Y: 0})
	// Заголовок 1 занимает y=[0..34], контент ~ пустой (padding 16px => y=[34..50]), заголовок 2 начинается около y=50
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 60})
	ctx.Render(func(c *Context) {
		c.Accordion("acc", items, openIDs, Width(300))
	})
	if !openIDs["a2"] {
		t.Fatalf("expected accordion openIDs[a2]=true, got %v", openIDs)
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

func TestToastPositionAndOffset(t *testing.T) {
	ctx := newTestContext(800, 600)

	// По умолчанию: правый нижний угол с отступом 16, 16
	rx, ry, ok := ctx.calcToastRectPos(300, 64, 10, 16, 16, 8, 0)
	if !ok || rx != 800-300-16 || ry != 600-16-64 {
		t.Fatalf("неожиданная позиция по умолчанию: rx=%d ry=%d ok=%v", rx, ry, ok)
	}

	// Верхний центр с кастомным отступом
	ctx.SetToastPosition(ToastTopCenter)
	ctx.SetToastOffset(24, 56)
	if !ctx.toastHasOffset || ctx.toastOffsetXDp != 24 || ctx.toastOffsetYDp != 56 {
		t.Fatalf("отступы тоста не сохранились: %+v %d %d", ctx.toastHasOffset, ctx.toastOffsetXDp, ctx.toastOffsetYDp)
	}
	rx, ry, ok = ctx.calcToastRectPos(300, 64, 10, ctx.toastOffsetXDp.Int(), ctx.toastOffsetYDp.Int(), 8, 0)
	if !ok || rx != (800-300)/2 || ry != 56 {
		t.Fatalf("неожиданная позиция ToastTopCenter: rx=%d ry=%d ok=%v", rx, ry, ok)
	}

	// Левый нижний угол с отрицательным отступом (должен зажаться к 0)
	ctx.SetToastPosition(ToastBottomLeft)
	ctx.SetToastOffset(-10, -5)
	if ctx.toastOffsetXDp != 0 || ctx.toastOffsetYDp != 0 {
		t.Fatalf("отрицательные отступы должны зажиматься к 0: x=%d y=%d", ctx.toastOffsetXDp, ctx.toastOffsetYDp)
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

func TestToastDoesNotBlockOverlayOrInput(t *testing.T) {
	ctx := newTestContext(800, 600)
	ctx.ShowToast("Notice", "Something happened", ToastInfo, 5*time.Second)

	ctx.Render(func(c *Context) {
		// Обычный кадр с тостом
	})

	// Тосты не должны регистрироваться в оверлеях
	if ctx.Overlays.HasActive() {
		t.Fatalf("expected Overlays.HasActive() == false when only toast is present")
	}

	// Тосты не должны блокировать ввод
	ctx.MouseX = 700
	ctx.MouseY = 550
	if ctx.IsInputBlocked() {
		t.Fatalf("expected IsInputBlocked() == false under toast area")
	}
}

func TestDrawerSides(t *testing.T) {
	ctx := newTestContext(800, 600)
	open := true

	// Проверяем DrawerTop
	ctx.Render(func(c *Context) {
		c.Drawer("drw_top", &open, DrawerTop, func() {
			c.Label("Top content", 14, c.Theme.TextPrimary)
		}, Height(150))
	})
	if !ctx.Overlays.HasActive() {
		t.Fatalf("expected active overlay for DrawerTop")
	}

	// Проверяем DrawerBottom
	ctx.Render(func(c *Context) {
		c.Drawer("drw_bottom", &open, DrawerBottom, func() {
			c.Label("Bottom content", 14, c.Theme.TextPrimary)
		}, Height(200))
	})
	if !ctx.Overlays.HasActive() {
		t.Fatalf("expected active overlay for DrawerBottom")
	}

	// Проверяем закрытие по клику вне панели DrawerTop
	openTop := true
	ctx.MouseX = 400
	ctx.MouseY = 500
	ctx.MousePressed = true
	ctx.Render(func(c *Context) {
		c.Drawer("drw_top_close", &openTop, DrawerTop, func() {}, Height(150))
	})
	if openTop {
		t.Fatalf("expected DrawerTop to close on outside click")
	}
}

func TestKeyBasedStateControls(t *testing.T) {
	ctx := newTestContext(600, 500)

	// 1. SelectItems helper
	items := SelectItems("Альфа", "Бета", "Гамма")
	if len(items) != 3 || items[0].Key != "Альфа" || items[0].Label != "Альфа" {
		t.Fatalf("некорректный результат SelectItems: %+v", items)
	}

	// 2. Select и ComboBox с выбором по ключу
	selKey := "k2"
	customItems := []SelectItem{
		{Key: "k1", Label: "Ключ 1", Icon: "<svg></svg>"},
		{Key: "k2", Label: "Ключ 2"},
		{Key: "k3", Label: "Ключ 3"},
	}

	// Nil selected
	if ctx.Select("sel_nil", customItems, nil) {
		t.Fatal("Select с nil selected должен возвращать false")
	}

	// Рендерим начальный кадр
	ctx.Render(func(c *Context) {
		c.Select("sel_test", customItems, &selKey, Width(200))
	})

	// Навигация клавишей KeyDown при закрытом списке: k2 -> k3
	ctx.FocusID = "sel_test"
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyDown})
	var changed bool
	ctx.Render(func(c *Context) {
		if c.Select("sel_test", customItems, &selKey, Width(200)) {
			changed = true
		}
	})
	if !changed || selKey != "k3" {
		t.Fatalf("ожидался выбор selKey=k3 после KeyDown, получено %q (changed=%v)", selKey, changed)
	}

	// Навигация KeyHome: k3 -> k1
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyHome})
	var changedHome bool
	ctx.Render(func(c *Context) {
		if c.Select("sel_test", customItems, &selKey, Width(200)) {
			changedHome = true
		}
	})
	if !changedHome || selKey != "k1" {
		t.Fatalf("ожидался выбор selKey=k1 после KeyHome, получено %q", selKey)
	}

	// 3. MultiSelect и CheckComboBox на map[string]bool
	selMap := map[string]bool{"k1": true}
	if ctx.MultiSelect("ms_nil", customItems, nil) {
		t.Fatal("MultiSelect с nil selectedIDs должен возвращать false")
	}

	ctx.Render(func(c *Context) {
		c.MultiSelect("ms_test", customItems, selMap, Width(200))
		c.CheckComboBox("ccb_test", customItems, selMap, Width(200))
	})
	if !selMap["k1"] {
		t.Fatal("ожидалось сохранение k1=true в selMap")
	}

	// 4. Accordion с поддержкой одновременного раскрытия секций
	accMap := map[string]bool{"s1": true, "s2": true}
	s1Drawn := false
	s2Drawn := false
	accItems := []AccordionItem{
		{ID: "s1", Title: "Секция 1", Draw: func() { s1Drawn = true }},
		{ID: "s2", Title: "Секция 2", Draw: func() { s2Drawn = true }},
	}
	ctx.Render(func(c *Context) {
		c.Accordion("acc_test", accItems, accMap)
	})
	if !s1Drawn || !s2Drawn {
		t.Fatalf("в Accordion с map[string]bool должны одновременно отрисовываться обе раскрытые секции (s1=%v, s2=%v)", s1Drawn, s2Drawn)
	}

	// 5. RadioButton с выбором по строковому ключу
	radioVal := "optA"
	if ctx.RadioButton("rad_nil", "Лейбл", nil, "optB") {
		t.Fatal("RadioButton с nil selected должен возвращать false")
	}

	// Клик по радиокнопке
	ctx.MouseX = 10
	ctx.MouseY = 10
	ctx.MousePressed = true
	ctx.Render(func(c *Context) {
		c.RadioButton("rad_test", "Опция B", &radioVal, "optB")
	})
	ctx.MouseReleased = true
	var changedRadio bool
	ctx.Render(func(c *Context) {
		if c.RadioButton("rad_test", "Опция B", &radioVal, "optB") {
			changedRadio = true
		}
	})
	if !changedRadio || radioVal != "optB" {
		t.Fatalf("ожидался выбор radioVal=optB, получено %q (changed=%v)", radioVal, changedRadio)
	}
}



