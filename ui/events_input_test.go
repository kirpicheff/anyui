package ui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
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

func TestInteractiveCanvasInputCaptureAndFocus(t *testing.T) {
	ctx := newTestContext(400, 300)

	var keys []RawKeyEvent
	var downs, moves, ups, wheels int
	var lastDownX, lastDownY int
	var lastMoveX, lastMoveY int
	var lastUpX, lastUpY int
	var lastUpBtn platform.MouseButton
	var focusLostCount int
	var lastFocused bool

	draw := func(c *Context) {
		c.Padding(20, func() {
			c.InteractiveCanvas("rdp", 200, 100, CanvasCallbacks{
				OnDraw: func(canvas *render.Canvas, area render.Rect, focused bool) {
					lastFocused = focused
				},
				OnKey: func(ev RawKeyEvent) {
					keys = append(keys, ev)
				},
				OnMouseDown: func(x, y int, button platform.MouseButton, mods platform.Modifiers) {
					downs++
					lastDownX, lastDownY = x, y
				},
				OnMouseMove: func(x, y int, mods platform.Modifiers) {
					moves++
					lastMoveX, lastMoveY = x, y
				},
				OnMouseUp: func(x, y int, button platform.MouseButton, mods platform.Modifiers) {
					ups++
					lastUpX, lastUpY = x, y
					lastUpBtn = button
				},
				OnMouseWheel: func(x, y int, wheelX, wheelY float32, mods platform.Modifiers) {
					wheels++
				},
				OnFocusLost: func() {
					focusLostCount++
				},
			})
		})
	}

	// Первый кадр: область холста находится по (20, 20, 200, 100)
	ctx.Render(draw)
	if lastFocused {
		t.Fatalf("до клика холст не должен быть в фокусе")
	}

	// Без фокуса клавиши не должны доходить до OnKey
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: 'A', ScanCode: 0x1E})
	ctx.Render(draw)
	if len(keys) != 0 {
		t.Fatalf("без фокуса OnKey не должен вызываться, получено %d событий", len(keys))
	}

	// Клик ПКМ внутри области (x=50, y=45) -> локальные координаты (30, 25), фокус устанавливается
	ctx.HandleEvent(platform.Event{
		Type:   platform.EventMouseDown,
		Button: platform.MouseRight,
		X:      50,
		Y:      45,
	})
	ctx.Render(draw)
	if !lastFocused || downs != 1 || lastDownX != 30 || lastDownY != 25 {
		t.Fatalf("ожидался фокус и локальные координаты (30,25), получено focused=%v downs=%d (%d,%d)",
			lastFocused, downs, lastDownX, lastDownY)
	}

	// Mouse Capture: уводим мышь за пределы холста (350, 250) и отпускаем ПКМ
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 350, Y: 250})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseRight, X: 350, Y: 250})
	ctx.Render(draw)
	if moves != 1 || lastMoveX != 330 || lastMoveY != 230 {
		t.Fatalf("ожидался MouseMove при захвате (330,230), получено moves=%d (%d,%d)", moves, lastMoveX, lastMoveY)
	}
	if ups != 1 || lastUpBtn != platform.MouseRight || lastUpX != 330 || lastUpY != 230 {
		t.Fatalf("ожидался MouseUp ПКМ при захвате (330,230), получено ups=%d btn=%v (%d,%d)", ups, lastUpBtn, lastUpX, lastUpY)
	}

	// Колесо мыши над холстом
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 40, Y: 40})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseWheel, WheelY: 1.0})
	ctx.Render(draw)
	if wheels != 1 {
		t.Fatalf("ожидался вызов OnMouseWheel, получено %d", wheels)
	}

	// KeyDown / KeyUp со ScanCode, Extended, Tab, Escape и VK_LWIN в фокусе
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyTab, ScanCode: 0x0F})
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyUp, KeyCode: KeyTab, ScanCode: 0x0F})
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyRight, ScanCode: 0x4D, Extended: true})
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: 0x5B, ScanCode: 0x5B, Extended: true}) // VK_LWIN
	ctx.Render(draw)
	if len(keys) != 4 {
		t.Fatalf("ожидалось 4 сырых события клавиатуры (включая Tab, KeyUp, Extended, VK_LWIN), получено %d", len(keys))
	}
	if !keys[0].Pressed || keys[0].KeyCode != KeyTab || keys[0].ScanCode != 0x0F {
		t.Fatalf("некорректное событие Tab KeyDown: %+v", keys[0])
	}
	if keys[1].Pressed || keys[1].KeyCode != KeyTab {
		t.Fatalf("некорректное событие Tab KeyUp: %+v", keys[1])
	}
	if !keys[2].Extended || keys[2].ScanCode != 0x4D {
		t.Fatalf("ожидался флаг Extended=true и ScanCode=0x4D, получено %+v", keys[2])
	}
	if keys[3].KeyCode != 0x5B || !keys[3].Extended {
		t.Fatalf("ожидалось событие VK_LWIN, получено %+v", keys[3])
	}

	// Проверяем на втором кадре, что Tab не сбросил и не переключил фокус (т.к. captureKeysFocus сработал)
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEscape, ScanCode: 0x01})
	ctx.Render(draw)
	if !lastFocused || len(keys) != 5 || keys[4].KeyCode != KeyEscape {
		t.Fatalf("Escape не должен сбрасывать фокус и должен попасть в OnKey: focused=%v len=%d", lastFocused, len(keys))
	}

	// Потеря фокуса окном (Alt+Tab / WM_KILLFOCUS) вызывает OnFocusLost
	ctx.HandleEvent(platform.Event{Type: platform.EventFocusLost})
	ctx.Render(draw)
	if lastFocused || focusLostCount != 1 {
		t.Fatalf("ожидался OnFocusLost=1 и focused=false, получено count=%d focused=%v", focusLostCount, lastFocused)
	}
}
