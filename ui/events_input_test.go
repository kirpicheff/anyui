package ui

import (
	"testing"
	"time"

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
	active := "t1"

	// Первый кадр для раскладки
	ctx.Render(func(c *Context) {
		c.TabBar("tb", tabs, &active)
	})

	// Средний клик по первой (незакрываемой) вкладке -> ClosedID == ""
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseMiddle, X: 40, Y: 18})
	var ev1 TabBarEvent
	ctx.Render(func(c *Context) {
		ev1 = c.TabBar("tb", tabs, &active)
	})
	if ev1.ClosedID != "" {
		t.Fatalf("незакрываемая вкладка не должна закрываться по СКМ, получено ClosedID=%q", ev1.ClosedID)
	}

	// Средний клик по второй (закрываемой) вкладке -> ClosedID == "t2"
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseMiddle, X: 40, Y: 18})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseMiddle, X: 200, Y: 18})
	closedID := ""
	ctx.Render(func(c *Context) {
		ev2 := c.TabBar("tb", tabs, &active)
		if ev2.ClosedID != "" {
			closedID = ev2.ClosedID
		}
	})
	if closedID != "t2" {
		t.Fatalf("ожидалось закрытие вкладки t2 по СКМ, получено ClosedID=%q", closedID)
	}
}

func TestScrollViewHorizontalScroll(t *testing.T) {
	ctx := newTestContext(400, 300)

	draw := func(c *Context) {
		c.ScrollView("sv", func() {
			c.Button("wide_btn", "Очень широкая кнопка", false, 500)
		}, Width(200), Height(150))
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
	selKey := "one"
	items := []SelectItem{
		{Key: "one", Label: "Один"},
		{Key: "two", Label: "Два"},
		{Key: "three", Label: "Три"},
	}

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

	// Select: фокус, KeyDown меняет выбранный ключ, KeySpace открывает
	ctx.FocusID = "sel"
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyDown})
	ctx.Render(func(c *Context) {
		c.Select("sel", items, &selKey, Width(200))
	})
	if selKey != "two" {
		t.Fatalf("Select: ожидалось selKey=two после KeyDown, получено %q", selKey)
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
		c.ScrollView("sv_cap", func() {
			for i := 0; i < 15; i++ {
				c.Spacer(30)
			}
		}, Width(200), Height(100))
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
			c.InteractiveCanvas("rdp", CanvasCallbacks{
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
			}, Width(200), Height(100))
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

func TestInteractiveCanvasDirectInputSkipsRender(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 400, h: 300, scale: 1.0}}
	ctx := NewContext(win, nil)

	var moves, downs, ups, wheels, keys int
	var lastMoveX, lastMoveY int
	var requestOnDown bool
	var showOverlay bool

	draw := func(c *Context) {
		c.Padding(20, func() {
			c.InteractiveCanvas("rdp", CanvasCallbacks{
				DirectInput: true,
				OnMouseMove: func(x, y int, mods platform.Modifiers) {
					moves++
					lastMoveX, lastMoveY = x, y
				},
				OnMouseDown: func(x, y int, button platform.MouseButton, mods platform.Modifiers) {
					downs++
					if requestOnDown {
						c.RequestRedraw()
					}
				},
				OnMouseUp: func(x, y int, button platform.MouseButton, mods platform.Modifiers) {
					ups++
				},
				OnMouseWheel: func(x, y int, wheelX, wheelY float32, mods platform.Modifiers) {
					wheels++
				},
				OnKey: func(ev RawKeyEvent) {
					keys++
				},
			}, Width(200), Height(100))
		})
		if showOverlay {
			c.PushOverlay(render.Rect{X: 0, Y: 0, W: 100, H: 100}, func() {})
		}
	}

	// Стартовый рендер
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)

	// Вход курсора в холст и первый клик (установка фокуса) требуют Render()
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 50}) {
		t.Fatal("вход курсора в холст должен запросить один Render для обновления hit/cursor")
	}
	ctx.Render(draw)
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 50}) {
		t.Fatal("первый клик для захвата фокуса должен запросить Render")
	}
	ctx.Render(draw)
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 50}) {
		// Холст уже в фокусе — MouseUp обрабатывается напрямую без Render
	} else {
		t.Fatal("MouseUp в сфокусированном холсте DirectInput не должен требовать Render")
	}

	movesBefore := moves
	downsBefore := downs
	upsBefore := ups

	// Движения мыши внутри холста должны доставляться напрямую БЕЗ Render()
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 70, Y: 60}) {
		t.Fatal("MouseMove внутри DirectInput-холста не должен вызывать Render")
	}
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 80, Y: 65}) {
		t.Fatal("повторный MouseMove внутри DirectInput-холста не должен вызывать Render")
	}
	if moves != movesBefore+2 || lastMoveX != 60 || lastMoveY != 45 {
		t.Fatalf("ожидалось +2 прямых OnMouseMove и координаты (60,45), получено moves=%d (%d,%d)", moves, lastMoveX, lastMoveY)
	}

	// Клик, колесо, клавиши и EventChar внутри сфокусированного холста — без Render()
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 80, Y: 65}) {
		t.Fatal("MouseDown в сфокусированном DirectInput-холсте не должен вызывать Render")
	}
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 80, Y: 65}) {
		t.Fatal("MouseUp в сфокусированном DirectInput-холсте не должен вызывать Render")
	}
	if ctx.HandleEvent(platform.Event{Type: platform.EventMouseWheel, WheelY: 1.0}) {
		t.Fatal("MouseWheel в сфокусированном DirectInput-холсте не должен вызывать Render")
	}
	if ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: 'A', ScanCode: 0x1E}) {
		t.Fatal("KeyDown в сфокусированном DirectInput-холсте не должен вызывать Render")
	}
	if ctx.HandleEvent(platform.Event{Type: platform.EventChar, Char: 'a'}) {
		t.Fatal("EventChar в сфокусированном DirectInput-холсте не должен вызывать Render")
	}
	if ctx.HandleEvent(platform.Event{Type: platform.EventKeyUp, KeyCode: 'A', ScanCode: 0x1E}) {
		t.Fatal("KeyUp в сфокусированном DirectInput-холсте не должен вызывать Render")
	}

	if downs != downsBefore+1 || ups != upsBefore+1 || wheels != 1 || keys != 2 {
		t.Fatalf("некорректная доставка прямых событий: downs=%d ups=%d wheels=%d keys=%d", downs, ups, wheels, keys)
	}

	// Если коллбек явно зовёт RequestRedraw(), HandleEvent должен вернуть true
	requestOnDown = true
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 80, Y: 65}) {
		t.Fatal("при вызове RequestRedraw() из коллбека HandleEvent должен вернуть true")
	}
	requestOnDown = false

	// При активном оверлее DirectInput должен отключиться
	showOverlay = true
	ctx.Render(draw)
	if !ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 80, Y: 65}) {
		t.Fatal("при открытом оверлее клик не должен перехватываться DirectInput без Render")
	}
}

func TestInteractiveCanvasStreamingFastPath(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 400, h: 300, scale: 1.0}}
	ctx := NewContext(win, nil)

	frameColor := render.RGB(10, 20, 30)
	draw := func(c *Context) {
		c.Padding(20, func() {
			c.InteractiveCanvas("video", CanvasCallbacks{
				Streaming: true,
				OnDraw: func(canvas *render.Canvas, area render.Rect, focused bool) {
					canvas.FillRect(area, frameColor)
				},
			}, Width(200), Height(100))
		})
	}

	// Первый кадр (Paint) -> полный Present
	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)
	if win.fullPresents != 1 || win.rectPresents != 0 {
		t.Fatalf("первый кадр должен идти через full Present, получено full=%d rect=%d", win.fullPresents, win.rectPresents)
	}

	// Фоновое обновление видеокадра по EventWakeUp -> fast-path PresentRect ровно по области холста
	frameColor = render.RGB(200, 100, 50)
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	ctx.Render(draw)
	if win.fullPresents != 1 || win.rectPresents != 1 {
		t.Fatalf("стриминговый кадр должен идти через PresentRect, получено full=%d rect=%d", win.fullPresents, win.rectPresents)
	}
	wantRect := render.Rect{X: 20, Y: 20, W: 200, H: 100}
	if win.lastRect != wantRect {
		t.Fatalf("ожидался PresentRect %+v, получено %+v", wantRect, win.lastRect)
	}
	if !ctx.prevPresentStale {
		t.Fatal("после стримингового fast-path флаг prevPresentStale должен быть установлен")
	}

	// Последующее обычное событие (например, клик вне холста) должно синхронизировать prevPresentPixels
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 350, Y: 250})
	ctx.Render(draw)
	if win.fullPresents != 2 {
		t.Fatalf("обычный кадр после стриминга должен синхронизировать prevPresentPixels через full Present, получено full=%d", win.fullPresents)
	}
	if ctx.prevPresentStale {
		t.Fatal("флаг prevPresentStale должен быть сброшен после полной синхронизации")
	}
}

func TestStreamingWakeOnlySkipsFullRedraw(t *testing.T) {
	win := &dirtyMockWindow{mockWindow: mockWindow{w: 320, h: 240, scale: 1.0}}
	ctx := NewContext(win, nil)

	drawCalls := 0
	canvasCalls := 0
	frameColor := render.RGB(10, 20, 30)

	draw := func(c *Context) {
		drawCalls++
		c.Padding(16, func() {
			c.InteractiveCanvas("video", CanvasCallbacks{
				Streaming: true,
				OnDraw: func(canvas *render.Canvas, area render.Rect, focused bool) {
					canvasCalls++
					canvas.FillRect(area, frameColor)
				},
			}, Width(160), Height(120))
		})
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventPaint})
	ctx.Render(draw)
	if drawCalls != 1 || canvasCalls != 1 {
		t.Fatalf("после первого Paint ожидались drawCalls=1 canvasCalls=1, получено %d и %d", drawCalls, canvasCalls)
	}

	// Повторяем несколько EventWakeUp подряд: draw() и Canvas.Clear не должны вызываться
	for i := 0; i < 3; i++ {
		frameColor = render.RGB(uint8(50+i*40), 100, 150)
		ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
		ctx.Render(draw)
	}
	if drawCalls != 1 {
		t.Fatalf("на EventWakeUp полное дерево draw() не должно перестраиваться: drawCalls=%d", drawCalls)
	}
	if canvasCalls != 4 {
		t.Fatalf("ожидалось 4 вызова OnDraw потокового холста, получено %d", canvasCalls)
	}
	if win.rectPresents != 3 || win.fullPresents != 1 {
		t.Fatalf("ожидалось full=1 rect=3, получено full=%d rect=%d", win.fullPresents, win.rectPresents)
	}
}

type backBufferMockWindow struct {
	dirtyMockWindow
	backBuf []byte
}

func (b *backBufferMockWindow) BackBuffer(width, height int) []byte {
	need := width * height * 4
	if need <= 0 {
		return nil
	}
	if len(b.backBuf) != need {
		b.backBuf = make([]byte, need)
	}
	return b.backBuf
}

func TestBackBufferProviderZeroCopy(t *testing.T) {
	win := &backBufferMockWindow{dirtyMockWindow: dirtyMockWindow{mockWindow: mockWindow{w: 200, h: 100, scale: 1.0}}}
	ctx := NewContext(win, nil)

	if len(ctx.Canvas.Pixels) == 0 || &ctx.Canvas.Pixels[0] != &win.backBuf[0] {
		t.Fatal("ожидалась прямая привязка Canvas.Pixels к BackBuffer окна")
	}

	// Ресайз должен обновить привязку к новому буферу
	ctx.HandleEvent(platform.Event{Type: platform.EventResize, Width: 300, Height: 150})
	if len(ctx.Canvas.Pixels) != 300*150*4 || &ctx.Canvas.Pixels[0] != &win.backBuf[0] {
		t.Fatal("после Resize Canvas.Pixels должен указывать на обновлённый BackBuffer")
	}

	// Закрытие должно отвязать срез от платформенной памяти
	ctx.HandleEvent(platform.Event{Type: platform.EventClose})
	if ctx.Canvas.Pixels != nil {
		t.Fatal("после EventClose Canvas.Pixels должен быть отвязан (nil)")
	}
}

func TestWakeUpDoesNotDropInputAndDragState(t *testing.T) {
	ctx := newTestContext(400, 300)

	// Нажатие кнопки мыши на строке виджета
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 50})
	ctx.ArmDrag("test_widget", 2, "row_2", WidgetStyle{})
	if ctx.reorderWidgetID != "test_widget" || ctx.reorderSourceIdx != 2 {
		t.Fatalf("ArmDrag не выставил состояние: id=%q idx=%d", ctx.reorderWidgetID, ctx.reorderSourceIdx)
	}

	// Серия тиков EventWakeUp (анимации, таймеры, фоновые задачи)
	for i := 0; i < 5; i++ {
		ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
		ctx.Render(func(c *Context) {
			// Внутри кадра проверяем сохранность состояния
			if c.reorderWidgetID != "test_widget" {
				t.Fatalf("reorderWidgetID сбросился на тике EventWakeUp #%d", i)
			}
		})
	}

	if ctx.reorderWidgetID != "test_widget" || !ctx.MouseDown {
		t.Fatalf("состояние перетаскивания потеряно после серии EventWakeUp: id=%q down=%v",
			ctx.reorderWidgetID, ctx.MouseDown)
	}

	// Перемещение мыши с превышением порога (4 dp)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 70})
	if !ctx.UpdateReorderDrag("test_widget") {
		t.Fatal("UpdateReorderDrag должен активировать перетаскивание при движении мыши")
	}
	if !ctx.reorderActive {
		t.Fatal("ожидалось reorderActive == true")
	}

	// Отпускание кнопки мыши завершает drag
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 70})
	ctx.Render(func(c *Context) {})
	if ctx.reorderActive || ctx.reorderWidgetID != "" {
		t.Fatalf("перетаскивание не сбросилось после MouseUp: active=%v id=%q", ctx.reorderActive, ctx.reorderWidgetID)
	}
}

func TestPushNonBlockingOverlayDoesNotBlockInput(t *testing.T) {
	ctx := newTestContext(400, 300)

	// Регистрируем неблокирующий оверлей
	ctx.Render(func(c *Context) {
		c.PushNonBlockingOverlay(render.Rect{X: 50, Y: 50, W: 150, H: 100}, func() {})
	})
	// Второй кадр переносит nextBlocks в prevBlocks
	ctx.Render(func(c *Context) {})

	// Координаты внутри оверлея не должны блокировать ввод
	if ctx.Overlays.IsBlocked(100, 80) {
		t.Fatal("PushNonBlockingOverlay не должен блокировать ввод под своими границами")
	}
	ctx.MouseX = 100
	ctx.MouseY = 80
	if ctx.IsInputBlocked() {
		t.Fatal("IsInputBlocked() должен возвращать false под неблокирующим оверлеем")
	}

	// Сравнение со стандартным PushOverlay (который обязан блокировать)
	ctx.Render(func(c *Context) {
		c.PushOverlay(render.Rect{X: 50, Y: 50, W: 150, H: 100}, func() {})
	})
	// Второй кадр переносит nextBlocks в prevBlocks
	ctx.Render(func(c *Context) {})

	if !ctx.Overlays.IsBlocked(100, 80) {
		t.Fatal("стандартный PushOverlay должен блокировать ввод под своими границами")
	}
}

func TestToastFastPathAndTimerLifecycle(t *testing.T) {
	ctx := newTestContext(500, 400)

	now := time.Now()
	ctx.Clock = func() time.Time { return now }

	ctx.ShowToast("Заголовок", "Текст сообщения", ToastInfo, 200*time.Millisecond)
	if ctx.ToastsCount() != 1 {
		t.Fatalf("ожидался 1 активный тост, получено %d", ctx.ToastsCount())
	}
	bounds := ctx.toastsBoundingRect()
	if bounds.Empty() {
		t.Fatal("toastsBoundingRect() не должен быть пустым при наличии активных тостов")
	}

	// Первый полный кадр для отрисовки тоста и сохранения подложки
	ctx.Render(func(c *Context) {})
	if ctx.toastUnderlayRect.Empty() || len(ctx.toastUnderlay) == 0 {
		t.Fatal("toastUnderlay не сохранился после первого рендера тоста")
	}

	// Кадр таймера EventWakeUp должен использовать быстрый путь перерисовки
	ctx.HandleEvent(platform.Event{Type: platform.EventWakeUp})
	if !ctx.canSkipFullFrameForToasts() {
		t.Fatal("canSkipFullFrameForToasts() должен возвращать true для живого тоста на EventWakeUp")
	}

	// Прокручиваем время за пределы жизни тоста
	now = now.Add(300 * time.Millisecond)
	if ctx.canSkipFullFrameForToasts() {
		t.Fatal("canSkipFullFrameForToasts() должен возвращать false после истечения срока жизни тоста для полного кадра очистки")
	}

	// Полный рендер очищает просроченный тост
	ctx.Render(func(c *Context) {})
	if ctx.ToastsCount() != 0 {
		t.Fatalf("просроченный тост должен быть удалён, осталось %d", ctx.ToastsCount())
	}
}
