package ui

import (
	"testing"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
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
		c.SplitPane("sp", Horizontal, &ratio, func() {}, func() {}, Width(400), Height(200))
	})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 300, Y: 100})
	ctx.Render(func(c *Context) {
		c.SplitPane("sp", Horizontal, &ratio, func() {}, func() {}, Width(400), Height(200))
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
	active := "t1"
	movedFrom, movedTo := -1, -1
	detached := ""

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
		if ev.DetachedID != "" {
			detached = ev.DetachedID
		}
	})
	if detached != "t2" {
		t.Fatalf("expected DetachedID=t2, got %s", detached)
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

	// 3. MultiSelect на map[string]bool
	selMap := map[string]bool{"k1": true}
	if ctx.MultiSelect("ms_nil", customItems, nil) {
		t.Fatal("MultiSelect с nil selectedIDs должен возвращать false")
	}

	ctx.Render(func(c *Context) {
		c.MultiSelect("ms_test", customItems, selMap, Width(200))
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

func TestTableReorder(t *testing.T) {
	ctx := newTestContext(600, 400)
	cols := []TableColumn{
		{Title: "ID", WidthDp: 80},
		{Title: "Title", WidthDp: 200},
	}
	items := []string{"Item 0", "Item 1", "Item 2", "Item 3"}
	selected := -1

	var receivedDrag *DragEvent
	reorderCallback := func(ev DragEvent) {
		receivedDrag = &ev
		// Перемещение элемента в модели данных силами приложения по контракту ToIndex (после удаления)
		fromIdx := ev.FromIndex
		toIdx := ev.ToIndex
		item := items[fromIdx]
		items = append(items[:fromIdx], items[fromIdx+1:]...)
		items = append(items[:toIdx], append([]string{item}, items[toIdx:]...)...)
	}

	renderTable := func() TableEvent {
		var result TableEvent
		ctx.Render(func(c *Context) {
			ev := c.Table("reorder_tbl", cols, len(items), &selected, nil, func(row, col int, area render.Rect, isSel bool) {},
				Width(400), Height(300), Reorderable(), OnReorder(reorderCallback))
			if ev.Drag != nil || result.Drag == nil {
				result = ev
			}
		})
		return result
	}

	// 1. Зажатие мыши на строке 1 (y: 63..93, возьмем y=75)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 75})
	renderTable()

	// 2. Движение мыши вниз к строке 3 (y: 123..153, возьмем y=145 для DropBelow)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 145})
	renderTable()

	// 3. Отпускание кнопки мыши на строке 3
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 145})
	ev := renderTable()

	if ev.Drag == nil {
		t.Fatal("ожидалось событие ev.Drag после завершения перетаскивания строки")
	}
	if ev.Drag.FromIndex != 1 || ev.Drag.ToIndex != 3 || ev.Drag.Drop != DropBelow {
		t.Fatalf("неверные параметры события Drag: %+v", ev.Drag)
	}
	if ev.Drag.FromKey != "" || ev.Drag.ToKey != "" {
		t.Fatalf("ожидались пустые FromKey и ToKey для Table, получено: %+v", ev.Drag)
	}
	if receivedDrag == nil {
		t.Fatal("ожидался вызов коллбека OnReorder")
	}
	if items[len(items)-1] != "Item 1" {
		t.Fatalf("ожидалось перемещение Item 1 в конец списка, получено: %v", items)
	}
}

func TestTreeViewReorder(t *testing.T) {
	ctx := newTestContext(600, 400)
	roots := []*TreeNode{
		{ID: "node0", Label: "Узел 0"},
		{ID: "node1", Label: "Узел 1"},
		{ID: "node2", Label: "Узел 2"},
	}
	selectedID := ""

	var receivedDrag *DragEvent
	reorderCallback := func(ev DragEvent) {
		receivedDrag = &ev
		// Перемещение узла в дереве
		var fromIdx, toIdx = -1, -1
		for i, n := range roots {
			if n.ID == ev.FromKey {
				fromIdx = i
			}
			if n.ID == ev.ToKey {
				toIdx = i
			}
		}
		if fromIdx >= 0 && toIdx >= 0 {
			moved := roots[fromIdx]
			roots = append(roots[:fromIdx], roots[fromIdx+1:]...)
			insertIdx := toIdx
			if fromIdx < toIdx {
				insertIdx = toIdx - 1
			}
			if ev.Drop == DropBelow {
				insertIdx++
			}
			roots = append(roots[:insertIdx], append([]*TreeNode{moved}, roots[insertIdx:]...)...)
		}
	}

	renderTree := func() bool {
		var anyChanged bool
		ctx.Render(func(c *Context) {
			if c.TreeView("reorder_tree", roots, &selectedID, Width(300), Height(200), Reorderable(), OnReorder(reorderCallback)) {
				anyChanged = true
			}
		})
		return anyChanged
	}

	// 1. Зажатие мыши на node0 (y: 4..32, берем y=18)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 18})
	renderTree()

	// 2. Движение мыши вниз к node2 (y: 60..88, берем y=65 для DropAbove)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 65})
	renderTree()

	// 3. Отпускание кнопки мыши над node2
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 65})
	changed := renderTree()

	if !changed {
		t.Fatal("ожидалось changed=true после успешного Reorder в TreeView")
	}
	if receivedDrag == nil {
		t.Fatal("ожидался вызов OnReorder для TreeView")
	}
	if receivedDrag.FromKey != "node0" || receivedDrag.ToKey != "node2" || receivedDrag.Drop != DropAbove {
		t.Fatalf("неверные параметры события Drag для TreeView: %+v", receivedDrag)
	}
	if roots[1].ID != "node0" {
		t.Fatalf("ожидалось перемещение node0 перед node2 (индекс 1), текущий порядок: %s, %s, %s", roots[0].ID, roots[1].ID, roots[2].ID)
	}
}

func TestTreeViewDropInside(t *testing.T) {
	t.Skip("перенос внутрь (DropInside) зарезервирован для Фазы 20b")
	ctx := newTestContext(600, 400)
	roots := []*TreeNode{
		{ID: "node0", Label: "Узел 0"},
		{ID: "folder1", Label: "Каталог 1", Expanded: true, Children: []*TreeNode{}},
	}
	selectedID := ""

	var receivedDrag *DragEvent
	reorderCallback := func(ev DragEvent) {
		receivedDrag = &ev
		if ev.Drop == DropInside && ev.ToKey == "folder1" {
			// Перемещение node0 внутрь folder1
			roots[1].Children = append(roots[1].Children, roots[0])
			roots = roots[1:]
		}
	}

	renderTree := func() bool {
		var anyChanged bool
		ctx.Render(func(c *Context) {
			if c.TreeView("tree_nested", roots, &selectedID, Width(300), Height(200), Reorderable(), OnReorder(reorderCallback)) {
				anyChanged = true
			}
		})
		return anyChanged
	}

	// 1. Зажатие мыши на node0 (y: 4..32, берем y=18)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 18})
	renderTree()

	// 2. Движение мыши в центр строки folder1 (y: 32..60, центр y=46)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 46})
	renderTree()

	// Проверяем, что во время перетаскивания активно ghost-превью в оверлее
	if !ctx.Overlays.HasActive() {
		t.Fatal("ожидалось активное ghost-превью в оверлее во время перетаскивания")
	}

	// 3. Отпускание кнопки мыши в центре строки folder1
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 46})
	changed := renderTree()

	if !changed {
		t.Fatal("ожидалось changed=true после перемещения внутрь каталога")
	}
	if receivedDrag == nil {
		t.Fatal("ожидался вызов OnReorder")
	}
	if receivedDrag.FromKey != "node0" || receivedDrag.ToKey != "folder1" || receivedDrag.Drop != DropInside {
		t.Fatalf("ожидался сброс DropInside в folder1, получено: %+v", receivedDrag)
	}
	if receivedDrag.FromWidget != "tree_nested" || receivedDrag.ToWidget != "tree_nested" {
		t.Fatalf("неверные FromWidget/ToWidget: %+v", receivedDrag)
	}
	if len(roots) != 1 || len(roots[0].Children) != 1 || roots[0].Children[0].ID != "node0" {
		t.Fatalf("ожидалось, что node0 теперь внутри folder1, структура: %+v", roots)
	}
}

func TestCrossWidgetReorderTree(t *testing.T) {
	t.Skip("кросс-виджетный перенос узлов зарезервирован для Фазы 20b")
	ctx := newTestContext(800, 400)
	treeA := []*TreeNode{
		{ID: "src_0", Label: "Источник 0"},
		{ID: "src_1", Label: "Источник 1"},
	}
	treeB := []*TreeNode{
		{ID: "dst_0", Label: "Приёмник 0"},
	}
	selA := ""
	selB := ""

	var receivedDrag *DragEvent
	reorderCallbackB := func(ev DragEvent) {
		receivedDrag = &ev
		if ev.FromWidget == "tree_a" && ev.ToWidget == "tree_b" {
			// Переносим элемент из A в B
			for i, n := range treeA {
				if n.ID == ev.FromKey {
					treeA = append(treeA[:i], treeA[i+1:]...)
					treeB = append(treeB, n)
					break
				}
			}
		}
	}

	renderAll := func() {
		ctx.Render(func(c *Context) {
			c.Row(0, 200, func() {
				c.TreeView("tree_a", treeA, &selA, Width(200), Height(200), Reorderable())
				c.TreeView("tree_b", treeB, &selB, Width(200), Height(200), Reorderable(), OnReorder(reorderCallbackB))
			})
		})
	}

	// 1. Зажатие мыши на первом элементе tree_a (X=50, Y=18)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 18})
	renderAll()

	// 2. Перемещение мыши в область tree_b (X=250, Y=18)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 250, Y: 18})
	renderAll()

	// 3. Отпускание кнопки мыши над tree_b
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 250, Y: 18})
	renderAll()

	if receivedDrag == nil {
		t.Fatal("ожидался вызов OnReorder на дереве tree_b при кросс-виджетном переносе")
	}
	if receivedDrag.FromWidget != "tree_a" || receivedDrag.ToWidget != "tree_b" {
		t.Fatalf("неверные виджеты источника/приемника: %+v", receivedDrag)
	}
	if receivedDrag.FromKey != "src_0" || receivedDrag.ToKey != "dst_0" {
		t.Fatalf("неверные ключи элементов: %+v", receivedDrag)
	}
	if len(treeA) != 1 || len(treeB) != 2 {
		t.Fatalf("ожидалось перемещение элемента: len(treeA)=%d, len(treeB)=%d", len(treeA), len(treeB))
	}
}

func TestToastClickAfterReorder(t *testing.T) {
	ctx := newTestContext(800, 600)
	roots := []*TreeNode{
		{ID: "node0", Label: "Узел 0"},
		{ID: "node1", Label: "Узел 1"},
	}
	selectedID := ""
	reorderCalled := false
	buttonClicked := false

	reorderCallback := func(ev DragEvent) {
		reorderCalled = true
		ctx.ShowToast("Reorder", "Перемещено", ToastSuccess, 3*time.Second)
	}

	renderUI := func() {
		ctx.Render(func(c *Context) {
			c.TreeView("my_tree", roots, &selectedID, Width(300), Height(200), Reorderable(), OnReorder(reorderCallback))
			if c.Button("test_btn", "Тестовая кнопка", false, 150) {
				buttonClicked = true
			}
		})
	}

	// 1. Зажатие мыши на node0
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 18})
	renderUI()

	// 2. Движение к node1
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 46})
	renderUI()

	// 3. Отпускание над node1
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 46})
	renderUI()

	if !reorderCalled {
		t.Fatal("reorder не вызвался")
	}
	if ctx.ToastsCount() == 0 {
		t.Fatal("тост не создан")
	}

	// 4. Теперь кликаем по тестовой кнопке (она находится ниже дерева, например Y=220, X=50)
	// Имитируем тики анимации EventWakeUp между нажатием и отпусканием
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 220})
	renderUI()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 220})
	renderUI()
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	renderUI()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 220})
	renderUI()

	if !buttonClicked {
		t.Fatal("клик по кнопке не прошел пока висит тост!")
	}

	// 5. Теперь кликаем по node0 в самом TreeView
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 18})
	renderUI()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 18})
	renderUI()
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	renderUI()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 18})
	renderUI()

	if selectedID != "node0" {
		t.Fatalf("клик по узлу node0 не выбрал его пока висит тост: selectedID=%s", selectedID)
	}

	// 6. Теперь пробуем снова начать drag на node0 и проверить сброс при EventCaptureLost
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 18})
	renderUI()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 46})
	renderUI()
	if !ctx.reorderActive {
		t.Fatal("повторный drag не активировался!")
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventCaptureLost})
	renderUI()
	if ctx.reorderActive {
		t.Fatal("reorderActive должен быть сброшен при EventCaptureLost!")
	}
}

func TestSelectSyncClickAndKeyboard(t *testing.T) {
	ctx := newTestContext(400, 300)
	items := []SelectItem{
		{Key: "a", Label: "Alpha"},
		{Key: "b", Label: "Beta"},
		{Key: "c", Label: "Gamma"},
	}
	selected := "a"

	// 1. Открываем список кликом по кнопке Select (высота 38, y=[0..38])
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 18})
	changed := false
	ctx.Render(func(c *Context) {
		if c.Select("sel1", items, &selected, Width(200)) {
			changed = true
		}
	})
	if changed {
		t.Fatal("открытие списка не должно возвращать changed=true")
	}
	if !ctx.openState["sel1"] {
		t.Fatal("список должен быть открыт после клика по кнопке")
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 18})

	// 2. Клик по второму пункту ("Beta"): dropRect.Y = 38 + 4 = 42, listY = 46, item0=[46..78], item1=[78..110]
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 90})
	changed = false
	ctx.Render(func(c *Context) {
		if c.Select("sel1", items, &selected, Width(200)) {
			changed = true
		}
	})
	if !changed {
		t.Fatalf("клик по пункту Select должен синхронно вернуть true в том же кадре (selected=%q, open=%v, mouse=(%d,%d))", selected, ctx.openState["sel1"], ctx.MouseX, ctx.MouseY)
	}
	if selected != "b" {
		t.Fatalf("ожидался ключ 'b', получено %q", selected)
	}
	if ctx.openState["sel1"] {
		t.Fatal("список должен закрыться после выбора пункта кликом")
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 90})

	// 3. Повторно открываем список и выбираем следующий пункт клавиатурой (KeyDown + KeyEnter)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 18})
	ctx.Render(func(c *Context) {
		c.Select("sel1", items, &selected, Width(200))
	})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 18})

	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyDown})
	changed = false
	ctx.Render(func(c *Context) {
		if c.Select("sel1", items, &selected, Width(200)) {
			changed = true
		}
	})
	if !changed || selected != "c" {
		t.Fatalf("KeyDown в открытом Select должен синхронно вернуть true и выбрать 'c', got changed=%v selected=%q", changed, selected)
	}
	if !ctx.openState["sel1"] {
		t.Fatal("KeyDown не должен закрывать список")
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEnter})
	ctx.Render(func(c *Context) {
		c.Select("sel1", items, &selected, Width(200))
	})
	if ctx.openState["sel1"] {
		t.Fatal("KeyEnter должен закрыть открытый список")
	}
}

func TestMultiSelectSyncClickAndButtons(t *testing.T) {
	ctx := newTestContext(400, 300)
	items := []SelectItem{
		{Key: "k1", Label: "One"},
		{Key: "k2", Label: "Two"},
		{Key: "k3", Label: "Three"},
	}
	selectedIDs := map[string]bool{}

	// 1. Открываем MultiSelect кликом по основному полю (rect: y=[0..34])
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 16})
	changed := false
	ctx.Render(func(c *Context) {
		if c.MultiSelect("ms1", items, selectedIDs, Width(240)) {
			changed = true
		}
	})
	if changed {
		t.Fatal("открытие MultiSelect не должно возвращать changed=true")
	}
	if !ctx.openState["ms1"] {
		t.Fatal("MultiSelect должен быть открыт")
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 16})

	// 2. Клик по второму пункту ("Two"): listRect.Y = 104, item0=[104..132], item1=[132..160]
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 146})
	changed = false
	ctx.Render(func(c *Context) {
		if c.MultiSelect("ms1", items, selectedIDs, Width(240)) {
			changed = true
		}
	})
	if !changed || !selectedIDs["k2"] {
		t.Fatalf("клик по пункту MultiSelect должен синхронно вернуть true и отметить k2, got changed=%v selectedIDs=%v", changed, selectedIDs)
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 146})

	// 3. Клик по кнопке "Выбрать все" (allR: x=[8..117], y=[78..100])
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 60, Y: 89})
	changed = false
	ctx.Render(func(c *Context) {
		if c.MultiSelect("ms1", items, selectedIDs, Width(240)) {
			changed = true
		}
	})
	if !changed || !selectedIDs["k1"] || !selectedIDs["k2"] || !selectedIDs["k3"] {
		t.Fatalf("кнопка 'Выбрать все' должна синхронно вернуть true и выбрать все пункты, got changed=%v selectedIDs=%v", changed, selectedIDs)
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 60, Y: 89})

	// 4. Клик по кнопке "Сбросить" (clrR: x=[123..232], y=[78..100])
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 175, Y: 89})
	changed = false
	ctx.Render(func(c *Context) {
		if c.MultiSelect("ms1", items, selectedIDs, Width(240)) {
			changed = true
		}
	})
	if !changed || len(selectedIDs) != 0 {
		t.Fatalf("кнопка 'Сбросить' должна синхронно вернуть true и очистить выбор, got changed=%v selectedIDs=%v", changed, selectedIDs)
	}
}

func TestDatePickerSyncClick(t *testing.T) {
	ctx := newTestContext(400, 400)
	fixedNow := time.Date(2025, 6, 20, 12, 0, 0, 0, time.UTC)
	ctx.Clock = func() time.Time { return fixedNow }

	date := time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC)

	// 1. Открываем календарь кликом по полю (rect: x=[0..180], y=[0..34])
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 16})
	changed := false
	ctx.Render(func(c *Context) {
		if c.DatePicker("dp1", &date, Width(180)) {
			changed = true
		}
	})
	if changed || !ctx.openState["dp1"] {
		t.Fatalf("открытие DatePicker должно открыть попап без changed, got changed=%v open=%v", changed, ctx.openState["dp1"])
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 16})

	// 2. Клик по 15 мая 2025: 1 мая — четверг (offset=3), день 15 -> slot=17 (col=3, row=2), центр ~(122, 176)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 122, Y: 176})
	changed = false
	ctx.Render(func(c *Context) {
		if c.DatePicker("dp1", &date, Width(180)) {
			changed = true
		}
	})
	if !changed {
		t.Fatal("клик по числу в DatePicker должен синхронно вернуть true в том же кадре")
	}
	if date.Year() != 2025 || date.Month() != time.May || date.Day() != 15 {
		t.Fatalf("ожидалась дата 2025-05-15, получено %s", date.Format("2006-01-02"))
	}
	if ctx.openState["dp1"] {
		t.Fatal("календарь должен закрыться после выбора числа")
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 122, Y: 176})

	// 3. Повторно открываем и жмём кнопку "Сегодня" (todayR: y=[288..312], центр (120, 300))
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 16})
	ctx.Render(func(c *Context) {
		c.DatePicker("dp1", &date, Width(180))
	})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 16})

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 120, Y: 300})
	changed = false
	ctx.Render(func(c *Context) {
		if c.DatePicker("dp1", &date, Width(180)) {
			changed = true
		}
	})
	if !changed {
		t.Fatal("клик по 'Сегодня' в DatePicker должен синхронно вернуть true в том же кадре")
	}
	if date.Year() != 2025 || date.Month() != time.June || date.Day() != 20 {
		t.Fatalf("ожидалась дата 2025-06-20, получено %s", date.Format("2006-01-02"))
	}
	if ctx.openState["dp1"] {
		t.Fatal("календарь должен закрыться после клика по 'Сегодня'")
	}
}

func TestTimePickerAndDateTimePickerSyncClick(t *testing.T) {
	ctx := newTestContext(400, 400)
	fixedNow := time.Date(2025, 6, 20, 14, 30, 45, 0, time.UTC)
	ctx.Clock = func() time.Time { return fixedNow }

	// Часть 1: TimePicker
	tm := time.Date(2025, 5, 10, 10, 15, 20, 0, time.UTC)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 16})
	changed := false
	ctx.Render(func(c *Context) {
		if c.TimePicker("tp1", &tm, Width(140)) {
			changed = true
		}
	})
	if changed || !ctx.openState["tp1"] {
		t.Fatalf("открытие TimePicker не должно менять значение, got changed=%v open=%v", changed, ctx.openState["tp1"])
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 16})

	// Клик по кнопке "+" часов: upR col0 = {18, 50, 50, 26}, центр (43, 63)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 43, Y: 63})
	changed = false
	ctx.Render(func(c *Context) {
		if c.TimePicker("tp1", &tm, Width(140)) {
			changed = true
		}
	})
	if !changed || tm.Hour() != 11 {
		t.Fatalf("клик '+' в TimePicker должен синхронно вернуть true и увеличить час до 11, got changed=%v hour=%d", changed, tm.Hour())
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 43, Y: 63})

	// Клик по кнопке "Сейчас": nowR = {12, 150, 186, 24}, центр (100, 162)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 100, Y: 162})
	changed = false
	ctx.Render(func(c *Context) {
		if c.TimePicker("tp1", &tm, Width(140)) {
			changed = true
		}
	})
	if !changed || tm.Hour() != 14 || tm.Minute() != 30 || tm.Second() != 45 {
		t.Fatalf("клик 'Сейчас' в TimePicker должен синхронно вернуть true и установить 14:30:45, got changed=%v time=%s", changed, tm.Format("15:04:05"))
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 100, Y: 162})

	// Часть 2: составной DateTimePicker
	ctx2 := newTestContext(400, 400)
	ctx2.Clock = func() time.Time { return fixedNow }
	dt := time.Date(2025, 5, 10, 8, 0, 0, 0, time.UTC)

	// Открываем DatePicker внутри DateTimePicker (x=[0..150], y=[0..34])
	ctx2.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 16})
	ctx2.Render(func(c *Context) {
		c.DateTimePicker("dtp1", &dt)
	})
	ctx2.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 40, Y: 16})

	// Выбираем 15 мая 2025 (центр ячейки (122, 176))
	ctx2.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 122, Y: 176})
	changed = false
	ctx2.Render(func(c *Context) {
		if c.DateTimePicker("dtp1", &dt) {
			changed = true
		}
	})
	if !changed || dt.Day() != 15 || dt.Hour() != 8 {
		t.Fatalf("DateTimePicker должен синхронно вернуть true и обновить дату до 15-го (сохранив час 8), got changed=%v dt=%s", changed, dt.Format(time.RFC3339))
	}
	ctx2.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 122, Y: 176})

	// Открываем TimePicker внутри DateTimePicker (x=[158..278], y=[0..34])
	ctx2.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 180, Y: 16})
	ctx2.Render(func(c *Context) {
		c.DateTimePicker("dtp1", &dt)
	})
	ctx2.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 180, Y: 16})

	// Жмём "+" часов (popX=158, baseX=170, upR col0={176, 50, 50, 26}, центр (201, 63))
	ctx2.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 201, Y: 63})
	changed = false
	ctx2.Render(func(c *Context) {
		if c.DateTimePicker("dtp1", &dt) {
			changed = true
		}
	})
	if !changed || dt.Day() != 15 || dt.Hour() != 9 {
		t.Fatalf("DateTimePicker должен синхронно вернуть true и обновить час до 9 (сохранив день 15), got changed=%v dt=%s", changed, dt.Format(time.RFC3339))
	}
}

func TestColorPickerSyncClick(t *testing.T) {
	ctx := newTestContext(400, 400)
	col := render.Hex(0x000000)

	// 1. Открываем ColorPicker кликом по полю (rect: x=[0..160], y=[0..34])
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 16})
	changed := false
	ctx.Render(func(c *Context) {
		if c.ColorPicker("cp1", &col, Width(160)) {
			changed = true
		}
	})
	if changed || !ctx.openState["cp1"] {
		t.Fatalf("открытие ColorPicker не должно возвращать changed=true, got changed=%v open=%v", changed, ctx.openState["cp1"])
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 40, Y: 16})

	// 2. Клик по второму быстрому свотчу (0x22C55E): swY=216, sr1={38, 216, 20, 20}, центр (48, 226)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 48, Y: 226})
	changed = false
	ctx.Render(func(c *Context) {
		if c.ColorPicker("cp1", &col, Width(160)) {
			changed = true
		}
	})
	if !changed || col != render.Hex(0x22C55E) {
		t.Fatalf("клик по свотчу ColorPicker должен синхронно вернуть true и установить #22C55E, got changed=%v col=%s", changed, FormatHexColor(col))
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 48, Y: 226})

	// 3. Клик по SV-квадрату (svR={12, 50, 212, 130}), правый верхний угол -> (220, 52)
	prevCol := col
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 220, Y: 52})
	changed = false
	ctx.Render(func(c *Context) {
		if c.ColorPicker("cp1", &col, Width(160)) {
			changed = true
		}
	})
	if !changed || col == prevCol {
		t.Fatalf("клик по SV-квадрату ColorPicker должен синхронно вернуть true и изменить цвет, got changed=%v col=%s", changed, FormatHexColor(col))
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 220, Y: 52})

	// 4. Клик по HEX-полю (hexBox={12, 246, 212, 30}) и замена последней цифры
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 100, Y: 260})
	ctx.Render(func(c *Context) {
		c.ColorPicker("cp1", &col, Width(160))
	})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 100, Y: 260})

	// Стираем 6 символов и вводим "FF0000"
	for i := 0; i < 6; i++ {
		ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyBackspace})
	}
	for _, r := range "FF0000" {
		ctx.HandleEvent(platform.Event{Type: platform.EventChar, Char: r})
	}
	changed = false
	ctx.Render(func(c *Context) {
		if c.ColorPicker("cp1", &col, Width(160)) {
			changed = true
		}
	})
	if !changed || col != render.Hex(0xFF0000) {
		t.Fatalf("ввод HEX в ColorPicker должен синхронно вернуть true и установить #FF0000, got changed=%v col=%s", changed, FormatHexColor(col))
	}
}
