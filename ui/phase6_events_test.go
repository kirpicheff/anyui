package ui

import (
	"testing"

	"anyui/platform"
	"anyui/render"
)

func TestContextDoubleAndMiddleClickAndCaptureLost(t *testing.T) {
	ctx := newTestContext(600, 400)

	// Первый клик ЛКМ
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 20, Y: 20})
	if ctx.DoubleClicked() {
		t.Fatal("первый клик не должен быть двойным")
	}
	ctx.endFrame()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 20, Y: 20})
	ctx.endFrame()

	// Второй клик ЛКМ рядом в пределах 500мс -> DoubleClicked
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 21, Y: 21})
	if !ctx.DoubleClicked() {
		t.Fatal("ожидался DoubleClicked() == true на втором клике")
	}
	ctx.endFrame()

	// Средняя кнопка мыши
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseMiddle, X: 50, Y: 60})
	if !ctx.MiddleMousePressed || !ctx.MiddleMouseDown || ctx.MiddleMouseX != 50 || ctx.MiddleMouseY != 60 {
		t.Fatalf("некорректное состояние средней кнопки при нажатии: pressed=%v down=%v", ctx.MiddleMousePressed, ctx.MiddleMouseDown)
	}
	ctx.endFrame()
	if ctx.MiddleMousePressed {
		t.Fatal("MiddleMousePressed должен сбрасываться в endFrame")
	}

	// Потеря захвата мыши сбрасывает зажатые кнопки и ActiveID
	ctx.ActiveID = "drag_slider"
	ctx.HandleEvent(platform.Event{Type: platform.EventCaptureLost})
	if ctx.MouseDown || ctx.MiddleMouseDown || ctx.ActiveID != "" {
		t.Fatalf("EventCaptureLost должен сбрасывать состояние захвата: MouseDown=%v MiddleDown=%v ActiveID=%q",
			ctx.MouseDown, ctx.MiddleMouseDown, ctx.ActiveID)
	}
}

func TestTabBarMiddleClickCloseAndWheelX(t *testing.T) {
	ctx := newTestContext(500, 200)
	tabs := []TabItem{
		{ID: "t1", Title: "Pinned", Closable: false},
		{ID: "t2", Title: "Closable", Closable: true},
	}
	active := 0

	// Первый кадр для раскладки
	ctx.Render(func(c *Context) {
		c.TabBar("tb", tabs, &active)
	})

	// Средний клик по первой (незакрываемой) вкладке -> Closed == -1
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseMiddle, X: 40, Y: 18})
	var ev1 TabBarEvent
	ctx.Render(func(c *Context) {
		ev1 = c.TabBar("tb", tabs, &active)
	})
	if ev1.Closed != -1 {
		t.Fatalf("незакрываемая вкладка не должна закрываться по СКМ, получено Closed=%d", ev1.Closed)
	}

	// Средний клик по второй (закрываемой) вкладке -> Closed == 1
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseMiddle, X: 40, Y: 18})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseMiddle, X: 200, Y: 18})
	closedIdx := -1
	ctx.Render(func(c *Context) {
		ev2 := c.TabBar("tb", tabs, &active)
		if ev2.Closed >= 0 {
			closedIdx = ev2.Closed
		}
	})
	if closedIdx != 1 {
		t.Fatalf("ожидалось закрытие вкладки 1 по СКМ, получено Closed=%d", closedIdx)
	}
}

func TestScrollViewHorizontalScroll(t *testing.T) {
	ctx := newTestContext(400, 300)

	draw := func(c *Context) {
		c.ScrollView("sv", 200, 150, func() {
			c.Button("wide_btn", "Очень широкая кнопка", false, 500)
		})
	}
	ctx.Render(draw)

	// Наводим курсор внутрь ScrollView и крутим колесо по горизонтали (WheelX)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 50})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseWheel, WheelX: 1.0})
	ctx.Render(draw)

	si := ctx.getScrollInfo("sv:sv")
	if si.x <= 0 {
		t.Fatalf("ожидалась горизонтальная прокрутка по WheelX, si.x=%d", si.x)
	}
	prevX := si.x

	// Прокрутка через Shift + WheelY вниз (-1.0) должна прокрутить ещё правее
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseWheel, WheelY: -1.0, Mods: platform.Modifiers{Shift: true}})
	ctx.Render(draw)
	if si.x <= prevX {
		t.Fatalf("ожидалась горизонтальная прокрутка по Shift+WheelY, prevX=%d, si.x=%d", prevX, si.x)
	}
}

func TestTableEventsAndKeyboardNav(t *testing.T) {
	ctx := newTestContext(500, 400)
	cols := []TableColumn{
		{Title: "Col1", WidthDp: 200},
		{Title: "Col2", WidthDp: 200},
		{Title: "Col3", WidthDp: 200},
	}
	sel := 0
	dblRow := -1
	midRow := -1

	draw := func(c *Context) {
		ev := c.Table("tbl", cols, 20, &sel, nil, func(row, col int, area render.Rect, isSelected bool) {}, Width(300), Height(200))
		if ev.RowDoubleClicked >= 0 {
			dblRow = ev.RowDoubleClicked
		}
		if ev.RowMiddleClicked >= 0 {
			midRow = ev.RowMiddleClicked
		}
	}
	ctx.Render(draw)

	// Двойной клик по первой строке (y ~ 32+15 = 47)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 45})
	ctx.Render(draw)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 40, Y: 45})
	ctx.Render(draw)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 45})
	ctx.Render(draw)
	if dblRow != 0 {
		t.Fatalf("ожидался RowDoubleClicked=0, получено %d", dblRow)
	}

	// Средний клик по второй строке (y ~ 32+30+15 = 77)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseMiddle, X: 40, Y: 75})
	ctx.Render(draw)
	if midRow != 1 || sel != 1 {
		t.Fatalf("ожидался RowMiddleClicked=1 и sel=1, получено RowMiddleClicked=%d sel=%d", midRow, sel)
	}

	// Клавиатурная навигация в фокусе (Down, End, Home, Right)
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyDown})
	ctx.Render(draw)
	if sel != 2 {
		t.Fatalf("ожидалось sel=2 после KeyDown, получено %d", sel)
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEnd})
	ctx.Render(draw)
	if sel != 19 {
		t.Fatalf("ожидалось sel=19 после KeyEnd, получено %d", sel)
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyRight})
	ctx.Render(draw)
	if ctx.getScrollInfo("tbl:tbl").x <= 0 {
		t.Fatal("ожидался горизонтальный скролл таблицы после KeyRight")
	}
}

func TestListViewTreeViewSelectKeyboardNav(t *testing.T) {
	ctx := newTestContext(600, 500)
	lvSel := 0
	treeSel := "root"
	roots := []*TreeNode{
		{
			ID:       "root",
			Label:    "Root",
			Expanded: false,
			Children: []*TreeNode{{ID: "child1", Label: "Child 1"}},
		},
	}
	selIdx := 0
	items := []string{"Один", "Два", "Три"}

	// ListView: фокус и KeyDown
	ctx.FocusID = "lv"
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyDown})
	ctx.Render(func(c *Context) {
		c.ListView("lv", 10, 30, &lvSel, nil, Width(200), Height(150))
	})
	if lvSel != 1 {
		t.Fatalf("ListView: ожидалось lvSel=1 после KeyDown, получено %d", lvSel)
	}

	// TreeView: фокус и KeyRight раскрывает узел, следующий KeyDown переходит на child1
	ctx.FocusID = "tv"
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyRight})
	ctx.Render(func(c *Context) {
		c.TreeView("tv", roots, &treeSel, Width(200), Height(150))
	})
	if !roots[0].Expanded {
		t.Fatal("TreeView: KeyRight должен раскрывать свёрнутый узел")
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyDown})
	ctx.Render(func(c *Context) {
		c.TreeView("tv", roots, &treeSel, Width(200), Height(150))
	})
	if treeSel != "child1" {
		t.Fatalf("TreeView: ожидалось treeSel=child1, получено %q", treeSel)
	}

	// Select: фокус, KeyDown меняет выбранный индекс, KeySpace открывает
	ctx.FocusID = "sel"
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyDown})
	ctx.Render(func(c *Context) {
		c.Select("sel", items, &selIdx, Width(200))
	})
	if selIdx != 1 {
		t.Fatalf("Select: ожидалось selIdx=1 после KeyDown, получено %d", selIdx)
	}
}

func TestSliderCaptureOutsideWindow(t *testing.T) {
	ctx := newTestContext(300, 200)
	val := float32(0.5)

	draw := func(c *Context) {
		c.Slider("sl", &val, 0, 1, Width(200))
	}
	ctx.Render(draw)

	// Нажимаем на слайдер
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 100, Y: 14})
	ctx.Render(draw)
	if ctx.ActiveID != "sl" {
		t.Fatalf("ожидался ActiveID=sl, получено %q", ctx.ActiveID)
	}

	// Уводим мышь далеко за левую границу окна (X = -100) при зажатой кнопке
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: -100, Y: -50})
	ctx.Render(draw)
	if val != 0 {
		t.Fatalf("ожидалось val=0 при уводе курсора влево за окно, получено %v", val)
	}
}

type testNav struct {
	count    int
	selected int
	page     int
	lastCall int
}

func (n *testNav) Count() int         { return n.count }
func (n *testNav) SelectedIndex() int { return n.selected }
func (n *testNav) PageSize() int      { return n.page }
func (n *testNav) OnSelect(idx int) {
	n.selected = idx
	n.lastCall = idx
}

func TestNavigableInterfaceHelper(t *testing.T) {
	ctx := newTestContext(300, 200)
	ctx.FocusID = "custom_nav"
	nav := &testNav{count: 10, selected: 1, page: 4, lastCall: -1}

	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyPageDown})
	if !ctx.handleNavigable("custom_nav", nav, nil, nil) || nav.selected != 5 {
		t.Fatalf("ожидался selected=5 после PageDown, получено %d", nav.selected)
	}
	ctx.endFrame()

	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEnd})
	if !ctx.handleNavigable("custom_nav", nav, nil, nil) || nav.selected != 9 {
		t.Fatalf("ожидался selected=9 после End, получено %d", nav.selected)
	}
	ctx.endFrame()

	activated := -1
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeySpace})
	if !ctx.handleNavigable("custom_nav", nav, func(idx int) { activated = idx }, nil) || activated != 9 {
		t.Fatalf("ожидался вызов onActivate(9) по Space, получено %d", activated)
	}
}

func TestFocusLostAndScrollbarCapture(t *testing.T) {
	ctx := newTestContext(300, 200)
	ctx.FocusID = "txt_input"
	ctx.ActiveID = "sv_v:sv"
	ctx.MouseDown = true

	// Потеря фокуса окном (WM_KILLFOCUS -> EventFocusLost) должна сбрасывать и FocusID, и захват мыши
	ctx.HandleEvent(platform.Event{Type: platform.EventFocusLost})
	if ctx.FocusID != "" || ctx.ActiveID != "" || ctx.MouseDown {
		t.Fatalf("EventFocusLost должен сбрасывать FocusID, ActiveID и MouseDown: focus=%q active=%q down=%v",
			ctx.FocusID, ctx.ActiveID, ctx.MouseDown)
	}

	// Перетаскивание вертикального скроллбара ScrollView продолжается за пределами окна
	draw := func(c *Context) {
		c.ScrollView("sv_cap", 200, 100, func() {
			for i := 0; i < 15; i++ {
				c.Spacer(30)
			}
		})
	}
	ctx.Render(draw)

	// Нажимаем на бегунок скроллбара (справа внутри ScrollView)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 194, Y: 10})
	ctx.Render(draw)

	// Уводим курсор далеко вниз за границу окна (X = 350, Y = 500)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 350, Y: 500})
	ctx.Render(draw)
	si := ctx.getScrollInfo("sv:sv_cap")
	if si.y <= 0 {
		t.Fatalf("ожидалась прокрутка ScrollView при перетаскивании скроллбара за пределы окна, si.y=%d", si.y)
	}
}
