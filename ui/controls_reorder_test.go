package ui

import (
	"testing"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

// setupReorderTableHelper настраивает тестовый контекст и обёртку для шагов рендера таблицы.
func setupReorderTableHelper(t *testing.T, count int, scale float32, reorderable bool, onReorder func(DragEvent)) (*Context, *[]string, func(x, y int, down, up bool, key uint32) TableEvent) {
	t.Helper()
	win := &mockWindow{w: 600, h: 400, scale: scale}
	ctx := NewContext(win, nil)
	ctx.DpiScale = scale

	items := make([]string, count)
	for i := 0; i < count; i++ {
		items[i] = "Item " + string(rune('0'+i))
	}
	selected := -1

	cols := []TableColumn{
		{Title: "ID", WidthDp: 80},
		{Title: "Title", WidthDp: 200},
	}

	step := func(x, y int, down, up bool, key uint32) TableEvent {
		if key != 0 {
			ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: key})
		}
		if down {
			ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: x, Y: y})
		} else if up {
			ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: x, Y: y})
		} else if x != 0 || y != 0 {
			ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: x, Y: y})
		}

		var result TableEvent
		ctx.Render(func(c *Context) {
			opts := []Option{Width(400), Height(300)}
			if reorderable {
				opts = append(opts, Reorderable())
			}
			if onReorder != nil {
				opts = append(opts, OnReorder(onReorder))
			}
			ev := c.Table("tbl_reorder", cols, len(items), &selected, nil, func(row, col int, area render.Rect, isSel bool) {}, opts...)
			if ev.Drag != nil || result.Drag == nil {
				result = ev
			}
		})
		return result
	}

	return ctx, &items, step
}

func TestTableReorder_HappyPathDown(t *testing.T) {
	var receivedDrag *DragEvent
	callCount := 0
	_, _, step := setupReorderTableHelper(t, 5, 1.0, true, func(ev DragEvent) {
		receivedDrag = &ev
		callCount++
	})

	// Строка 1: Y 64..93. Середина 75.
	step(50, 75, true, false, 0)

	// Тащим вниз на строку 3: Y 124..153. Нижняя половина 145 (DropBelow).
	step(50, 145, false, false, 0)

	// Отпускаем мышь на строке 3
	ev := step(50, 145, false, true, 0)

	if ev.Drag == nil {
		t.Fatal("ожидалось событие ev.Drag")
	}
	if callCount != 1 {
		t.Fatalf("ожидался ровно 1 вызов OnReorder, получено: %d", callCount)
	}
	if receivedDrag == nil {
		t.Fatal("колбэк OnReorder не был вызван")
	}
	if receivedDrag.FromIndex != 1 || receivedDrag.ToIndex != 3 {
		t.Fatalf("неверные индексы: FromIndex=%d, ToIndex=%d (ожидалось 1 и 3)", receivedDrag.FromIndex, receivedDrag.ToIndex)
	}
	if receivedDrag.Drop != DropBelow {
		t.Fatalf("ожидалась зона DropBelow, получено: %v", receivedDrag.Drop)
	}
	if receivedDrag.FromKey != "" || receivedDrag.ToKey != "" {
		t.Fatalf("ожидались пустые FromKey и ToKey, получено: FromKey=%q, ToKey=%q", receivedDrag.FromKey, receivedDrag.ToKey)
	}
}

func TestTableReorder_HappyPathUp(t *testing.T) {
	var receivedDrag *DragEvent
	callCount := 0
	_, _, step := setupReorderTableHelper(t, 5, 1.0, true, func(ev DragEvent) {
		receivedDrag = &ev
		callCount++
	})

	// Строка 3: Y 124..153. Нажатие на Y=140.
	step(50, 140, true, false, 0)

	// Тащим вверх на строку 1: Y 64..93. Верхняя половина 70 (DropAbove).
	step(50, 70, false, false, 0)

	// Отпускаем мышь на строке 1
	ev := step(50, 70, false, true, 0)

	if ev.Drag == nil {
		t.Fatal("ожидалось событие ev.Drag")
	}
	if callCount != 1 {
		t.Fatalf("ожидался ровно 1 вызов OnReorder, получено: %d", callCount)
	}
	if receivedDrag.FromIndex != 3 || receivedDrag.ToIndex != 1 {
		t.Fatalf("неверные индексы: FromIndex=%d, ToIndex=%d (ожидалось 3 и 1)", receivedDrag.FromIndex, receivedDrag.ToIndex)
	}
	if receivedDrag.Drop != DropAbove {
		t.Fatalf("ожидалась зона DropAbove, получено: %v", receivedDrag.Drop)
	}
	if receivedDrag.FromKey != "" || receivedDrag.ToKey != "" {
		t.Fatalf("ожидались пустые FromKey и ToKey, получено: FromKey=%q, ToKey=%q", receivedDrag.FromKey, receivedDrag.ToKey)
	}
}

func TestTableReorder_NoOpSelfDrop(t *testing.T) {
	callCount := 0
	_, _, step := setupReorderTableHelper(t, 5, 1.0, true, func(ev DragEvent) {
		callCount++
	})

	// Строка 1: Y 64..93. Нажатие на 75.
	step(50, 75, true, false, 0)
	// Смещение внутри строки 1 ниже центра (DropBelow на саму себя -> ToIndex = 1)
	step(50, 85, false, false, 0)
	ev := step(50, 85, false, true, 0)

	if ev.Drag != nil {
		t.Fatalf("ожидалось отсутствие ev.Drag при no-op, получено: %+v", ev.Drag)
	}
	if callCount != 0 {
		t.Fatalf("колбэк OnReorder не должен вызываться при дропе на саму себя, вызовов: %d", callCount)
	}
}

func TestTableReorder_Threshold1x(t *testing.T) {
	ctx, _, step := setupReorderTableHelper(t, 5, 1.0, true, nil)

	// Нажатие на строке 1 (50, 75)
	step(50, 75, true, false, 0)

	// Смещение 2 px (меньше порога 4 Dp)
	step(50, 77, false, false, 0)
	if ctx.reorderActive {
		t.Fatal("drag не должен активироваться при смещении 2 px")
	}

	// Смещение ровно 4 px (порог строгий >)
	step(50, 79, false, false, 0)
	if ctx.reorderActive {
		t.Fatal("drag не должен активироваться при смещении ровно 4 px (> строгое)")
	}

	// Смещение 5 px (> 4)
	step(50, 80, false, false, 0)
	if !ctx.reorderActive {
		t.Fatal("drag должен активироваться при смещении 5 px (> 4 Dp)")
	}
}

func TestTableReorder_Threshold2x(t *testing.T) {
	ctx, _, step := setupReorderTableHelper(t, 5, 2.0, true, nil)

	// При scale=2.0: headerH = 64, bodyRect.Y = 66, rowH = 60.
	// Строка 0: Y 66..125. Кликаем на (50, 95).
	step(50, 95, true, false, 0)

	// Смещение 4 px (2 Dp)
	step(50, 99, false, false, 0)
	if ctx.reorderActive {
		t.Fatal("drag не должен активироваться при смещении 4 px на 2.0x DPI")
	}

	// Смещение ровно 8 px (4 Dp)
	step(50, 103, false, false, 0)
	if ctx.reorderActive {
		t.Fatal("drag не должен активироваться при смещении ровно 8 px на 2.0x DPI (> строгое)")
	}

	// Смещение 9 px
	step(50, 104, false, false, 0)
	if !ctx.reorderActive {
		t.Fatal("drag должен активироваться при смещении 9 px на 2.0x DPI (> 8 px)")
	}
}

func TestTableReorder_CancelEscape(t *testing.T) {
	callCount := 0
	ctx, _, step := setupReorderTableHelper(t, 5, 1.0, true, func(ev DragEvent) {
		callCount++
	})

	// Нажатие на строке 1
	step(50, 75, true, false, 0)
	// Тащим вниз
	step(50, 140, false, false, 0)
	if !ctx.reorderActive {
		t.Fatal("drag должен быть активен")
	}

	// Нажимаем Escape
	step(50, 140, false, false, KeyEscape)
	if ctx.reorderActive || ctx.reorderWidgetID != "" {
		t.Fatal("состояние drag должно быть сброшено по Escape")
	}

	// Отпускание мыши после отмены
	ev := step(50, 140, false, true, 0)
	if ev.Drag != nil || callCount != 0 {
		t.Fatal("событие drag не должно генерироваться после отмены по Escape")
	}

	// Проверяем, что следующий drag работает штатно
	step(50, 75, true, false, 0)
	step(50, 145, false, false, 0)
	ev2 := step(50, 145, false, true, 0)
	if ev2.Drag == nil || callCount != 1 {
		t.Fatal("повторный drag после отмены по Escape должен работать штатно")
	}
}

func TestTableReorder_ReleaseOutside(t *testing.T) {
	callCount := 0
	_, _, step := setupReorderTableHelper(t, 5, 1.0, true, func(ev DragEvent) {
		callCount++
	})

	// Нажатие на строке 1
	step(50, 75, true, false, 0)
	// Тащим вниз
	step(50, 140, false, false, 0)

	// Уводим мышь за пределы таблицы и отпускаем
	ev := step(550, 380, false, true, 0)
	if ev.Drag != nil || callCount != 0 {
		t.Fatal("при отпускании за пределами таблицы событие не должно генерироваться")
	}
}

func TestTableReorder_WithoutReorderableOption(t *testing.T) {
	callCount := 0
	ctx, _, step := setupReorderTableHelper(t, 5, 1.0, false, func(ev DragEvent) {
		callCount++
	})

	// Нажатие и смещение при наличии только OnReorder, но без Reorderable()
	step(50, 75, true, false, 0)
	step(50, 140, false, false, 0)
	if ctx.reorderActive {
		t.Fatal("drag не должен активироваться без опции Reorderable()")
	}

	ev := step(50, 140, false, true, 0)
	if ev.Drag != nil || callCount != 0 {
		t.Fatal("событие не должно генерироваться без Reorderable()")
	}
}

func TestTableReorder_ClickOnHeaderOrScrollbar(t *testing.T) {
	// 20 строк создают необходимость в вертикальном скроллбаре
	ctx, _, step := setupReorderTableHelper(t, 20, 1.0, true, nil)

	// 1. Клик и таскание на заголовке таблицы (Y = 15)
	step(50, 15, true, false, 0)
	step(50, 50, false, false, 0)
	if ctx.reorderActive || ctx.reorderWidgetID != "" {
		t.Fatal("drag строк не должен начинаться при клике на заголовок таблицы")
	}
	step(50, 50, false, true, 0)

	// 2. Клик и таскание на скроллбаре таблицы (X = 395)
	step(395, 100, true, false, 0)
	step(395, 150, false, false, 0)
	if ctx.reorderActive || ctx.reorderWidgetID != "" {
		t.Fatal("drag строк не должен начинаться при клике на скроллбар")
	}
	step(395, 150, false, true, 0)
}

func TestTableReorder_DataShrunkUnderDrag(t *testing.T) {
	callCount := 0
	ctx := newTestContext(600, 400)
	items := []string{"0", "1", "2", "3", "4"}
	selected := -1

	cols := []TableColumn{
		{Title: "ID", WidthDp: 80},
		{Title: "Title", WidthDp: 200},
	}

	renderTable := func() TableEvent {
		var res TableEvent
		ctx.Render(func(c *Context) {
			ev := c.Table("tbl_shrink", cols, len(items), &selected, nil, func(row, col int, area render.Rect, isSel bool) {},
				Width(400), Height(300), Reorderable(), OnReorder(func(ev DragEvent) {
					callCount++
				}))
			if ev.Drag != nil {
				res = ev
			}
		})
		return res
	}

	// Зажимаем строку 4 (Y ~ 160)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 160})
	renderTable()

	// Тащим вниз
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 190})
	renderTable()
	if !ctx.reorderActive {
		t.Fatal("drag должен быть активен")
	}

	// Данные внезапно уменьшаются до 2 элементов (строка 4 исчезает)
	items = items[:2]

	// Движение и отпускание мыши не должны приводить к панике или вызову колбэка
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 80})
	renderTable()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 80})
	ev := renderTable()

	if ev.Drag != nil || callCount != 0 {
		t.Fatal("событие не должно генерироваться при изменении данных под drag")
	}
	if ctx.reorderActive {
		t.Fatal("состояние drag должно быть сброшено")
	}
}

func TestTableReorder_DropBelowLastRow(t *testing.T) {
	var receivedDrag *DragEvent
	_, _, step := setupReorderTableHelper(t, 4, 1.0, true, func(ev DragEvent) {
		receivedDrag = &ev
	})

	// Зажимаем строку 0 (Y ~ 45)
	step(50, 45, true, false, 0)

	// Тащим ниже последней строки (например, Y = 250 внутри bodyRect, строк всего 4)
	step(50, 250, false, false, 0)
	ev := step(50, 250, false, true, 0)

	if ev.Drag == nil || receivedDrag == nil {
		t.Fatal("ожидался дроп ниже последней строки")
	}
	// При перемещении 0 ниже последней строки (3), ToIndex должен быть равен 3 (вставка в конец)
	if receivedDrag.ToIndex != 3 {
		t.Fatalf("ожидался ToIndex=3 при дропе в конец, получено: %d", receivedDrag.ToIndex)
	}
}

func TestTableReorder_AutoScrollWithTime(t *testing.T) {
	ctx, _, step := setupReorderTableHelper(t, 30, 1.0, true, nil)
	ctx.Animator = NewAnimator()

	// Зажимаем строку 10 (Y ~ 150)
	step(50, 150, true, false, 0)
	// Тащим в зону верхнего края (bodyRect.Y = 34, edgeZone = 24 -> Y = 38)
	step(50, 38, false, false, 0)

	si := ctx.getScrollInfo("tbl:tbl_reorder")
	si.y = 100 // Задаем начальный скролл

	// Симулируем несколько кадров со временем 50ms
	ctx.Animator.BeginFrame(time.Now())
	ctx.Animator.dt = 0.050 // 50 мс
	step(50, 38, false, false, 0)

	ctx.Animator.BeginFrame(time.Now().Add(50 * time.Millisecond))
	ctx.Animator.dt = 0.050
	step(50, 38, false, false, 0)

	if si.y >= 100 {
		t.Fatalf("ожидалось уменьшение si.y при авто-скролле вверх, текущее si.y=%d", si.y)
	}

	// Отпускаем мышь для завершения теста
	step(50, 38, false, true, 0)
}

func TestTableReorder_VisualGeometryAt1_5x(t *testing.T) {
	win := &mockWindow{w: 600, h: 400, scale: 1.5}
	ctx := NewContext(win, nil)
	ctx.DpiScale = 1.5

	items := []string{"Item 0", "Item 1", "Item 2"}
	selected := -1
	cols := []TableColumn{
		{Title: "Title", WidthDp: 200},
	}

	renderTable := func() {
		ctx.Render(func(c *Context) {
			c.Table("tbl_geo", cols, len(items), &selected, nil, func(row, col int, area render.Rect, isSel bool) {},
				Width(400), Height(300), Reorderable())
		})
	}

	// 1. Зажимаем строку 0
	// При scale=1.5: headerH = Scale(32).Int() = 48px, rowH = Scale(30).Int() = 45px.
	// bodyRect.Y = 50.
	// Строка 0: Y 50..95. Центр 72.
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 72})
	renderTable()

	// 2. Тащим на строку 1 (Y 95..140), верхняя половина 105 (DropAbove)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 105})
	renderTable()

	if !ctx.reorderActive {
		t.Fatal("drag должен быть активен")
	}

	// Линия индикатора Accent должна быть отрисована на границе строки:
	// targetRow = 1, DropAbove -> lineY = targetRowY - lineH/2 = 95 - 1 = 94.
	// Проверяем пиксель в canvas по координатам (50, 95): цвет должен быть Accent (в BGRA)
	accentCol := ctx.Theme.Accent
	idx := (95*ctx.Canvas.Width + 50) * 4
	pixR := ctx.Canvas.Pixels[idx+2]
	pixG := ctx.Canvas.Pixels[idx+1]
	pixB := ctx.Canvas.Pixels[idx]
	if pixR != accentCol.R || pixG != accentCol.G || pixB != accentCol.B {
		t.Fatalf("ожидался цвет Accent линии-индикатора на границе строки, получено: R=%d, G=%d, B=%d, ожидалось: R=%d, G=%d, B=%d", pixR, pixG, pixB, accentCol.R, accentCol.G, accentCol.B)
	}

	// Завершаем drag
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 105})
	renderTable()
}

func TestTableReorder_AutoScrollClampLimits(t *testing.T) {
	ctx, _, step := setupReorderTableHelper(t, 40, 1.0, true, nil)
	ctx.Animator = NewAnimator()

	// 1. Скролл вниз до упора
	step(50, 100, true, false, 0)
	step(50, 290, false, false, 0) // Зона нижнего края (высота 300)

	si := ctx.getScrollInfo("tbl:tbl_reorder")
	totalH := 40 * 30
	// viewH = 300 - 32 (header) - 2 (border) = 266
	maxSc := totalH - 266
	si.y = maxSc - 15 // Устанавливаем вблизи нижней границы

	// Шагаем временем вперед до упора
	for i := 0; i < 10; i++ {
		ctx.Animator.BeginFrame(time.Now().Add(time.Duration(i*50) * time.Millisecond))
		ctx.Animator.dt = 0.050
		step(50, 290, false, false, 0)
	}

	if si.y != maxSc {
		t.Fatalf("ожидался строгий clamp на нижней границе si.y == maxSc (%d), получено: %d", maxSc, si.y)
	}

	// 2. Скролл вверх до упора
	step(50, 38, false, false, 0) // Зона верхнего края
	si.y = 15                     // Устанавливаем вблизи верхней границы
	for i := 0; i < 10; i++ {
		ctx.Animator.BeginFrame(time.Now().Add(time.Duration((10+i)*50) * time.Millisecond))
		ctx.Animator.dt = 0.050
		step(50, 38, false, false, 0)
	}

	if si.y != 0 {
		t.Fatalf("ожидался строгий clamp на верхней границе si.y == 0, получено: %d", si.y)
	}

	step(50, 38, false, true, 0)
}

func TestTreeViewReorder_HappyPath(t *testing.T) {
	ctx := newTestContext(600, 400)
	roots := []*TreeNode{
		{ID: "node0", Label: "Узел 0"},
		{ID: "node1", Label: "Узел 1"},
		{ID: "node2", Label: "Узел 2"},
	}
	selectedID := ""

	var receivedDrag *DragEvent
	callCount := 0
	renderTree := func() {
		ctx.Render(func(c *Context) {
			c.TreeView("tree_hp", roots, &selectedID, Width(300), Height(200), Reorderable(), OnReorder(func(ev DragEvent) {
				receivedDrag = &ev
				callCount++
			}))
		})
	}

	// Высота узла 28 px. inner.Y = 4 + 4 = 8 px.
	// node2: Y ~ 8 + 2*28 = 64..92. Середина ~78.
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 78})
	renderTree()

	// Тащим вверх на node0: Y ~ 8..36. Верхняя половина 15 (DropAbove).
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 15})
	renderTree()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 15})
	renderTree()

	if callCount != 1 || receivedDrag == nil {
		t.Fatalf("ожидался 1 вызов OnReorder для TreeView, вызовов: %d", callCount)
	}
	if receivedDrag.FromKey != "node2" || receivedDrag.ToKey != "node0" {
		t.Fatalf("неверные ключи: FromKey=%q, ToKey=%q", receivedDrag.FromKey, receivedDrag.ToKey)
	}
	if receivedDrag.FromIndex != -1 || receivedDrag.ToIndex != -1 {
		t.Fatalf("индексы для TreeView должны быть равны -1, получено: FromIndex=%d, ToIndex=%d", receivedDrag.FromIndex, receivedDrag.ToIndex)
	}
	if receivedDrag.Drop != DropAbove {
		t.Fatalf("ожидалась зона DropAbove, получено: %v", receivedDrag.Drop)
	}
}

func TestTreeViewReorder_SameParentOnly(t *testing.T) {
	ctx := newTestContext(600, 400)
	roots := []*TreeNode{
		{
			ID: "rootA", Label: "Папка A", Expanded: true,
			Children: []*TreeNode{
				{ID: "childA1", Label: "Файл A1"},
				{ID: "childA2", Label: "Файл A2"},
			},
		},
		{
			ID: "rootB", Label: "Папка B", Expanded: true,
			Children: []*TreeNode{
				{ID: "childB1", Label: "Файл B1"},
			},
		},
	}
	selectedID := ""

	callCount := 0
	renderTree := func() {
		ctx.Render(func(c *Context) {
			c.TreeView("tree_parent", roots, &selectedID, Width(300), Height(200), Reorderable(), OnReorder(func(ev DragEvent) {
				callCount++
			}))
		})
	}

	// childA1 находится на позиции 1 (Y ~ 8 + 28 = 36..64). Середина 50.
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 50})
	renderTree()

	// Тащим на childB1 (позиция 4, Y ~ 8 + 4*28 = 120..148). Середина 130.
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 130})
	renderTree()

	// Отпускаем мышь: родители разные (rootA vs rootB) -> событие отбрасывается
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 130})
	renderTree()

	if callCount != 0 {
		t.Fatalf("перенос между разными родителями запрещён в Фазе 20a, получено вызовов: %d", callCount)
	}
}

func TestTreeViewReorder_NoOpSelfDrop(t *testing.T) {
	ctx := newTestContext(600, 400)
	roots := []*TreeNode{
		{ID: "root0", Label: "Узел 0"},
		{ID: "root1", Label: "Узел 1"},
	}
	selectedID := ""

	callCount := 0
	renderTree := func() {
		ctx.Render(func(c *Context) {
			c.TreeView("tree_self", roots, &selectedID, Width(300), Height(200), Reorderable(), OnReorder(func(ev DragEvent) {
				callCount++
			}))
		})
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 20})
	renderTree()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 30})
	renderTree()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 30})
	renderTree()

	if callCount != 0 {
		t.Fatalf("дроп узла на самого себя должен быть no-op, вызовов: %d", callCount)
	}
}

func TestTreeViewReorder_TargetShrunkUnderDrag(t *testing.T) {
	ctx := newTestContext(600, 400)
	roots := []*TreeNode{
		{ID: "n0", Label: "0"},
		{ID: "n1", Label: "1"},
		{ID: "n2", Label: "2"},
	}
	selectedID := ""

	callCount := 0
	renderTree := func() {
		ctx.Render(func(c *Context) {
			c.TreeView("tree_target_shrink", roots, &selectedID, Width(300), Height(200), Reorderable(), OnReorder(func(ev DragEvent) {
				callCount++
			}))
		})
	}

	// Зажимаем n0 (Y ~ 20)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 20})
	renderTree()

	// Тащим на n2 (Y ~ 75)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 75})
	renderTree()

	// Удаляем целевой узел n2 до отпускания
	roots = roots[:1]

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 75})
	renderTree()

	if callCount != 0 {
		t.Fatal("при исчезновении целевого узла событие не должно вызываться")
	}
	if ctx.reorderActive {
		t.Fatal("состояние drag должно быть сброшено")
	}
}

func TestTreeViewReorder_VisualGeometryIndentAt1_5x(t *testing.T) {
	win := &mockWindow{w: 600, h: 400, scale: 1.5}
	ctx := NewContext(win, nil)
	ctx.DpiScale = 1.5

	roots := []*TreeNode{
		{
			ID: "root0", Label: "Папка", Expanded: true,
			Children: []*TreeNode{
				{ID: "child1", Label: "Дочерний 1"},
				{ID: "child2", Label: "Дочерний 2"},
			},
		},
	}
	selectedID := ""

	renderTree := func() {
		ctx.Render(func(c *Context) {
			c.TreeView("tree_geo", roots, &selectedID, Width(300), Height(200), Reorderable())
		})
	}

	// При scale=1.5: itemH = Scale(28) = 42 px. inner.X = Scale(4) = 6 px. inner.Y = 6 px.
	// child1: Y = 6 + 42 = 48..90. Центр 65.
	// child2: Y = 6 + 84 = 90..132. Верхняя половина 100 (DropAbove).
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 65})
	renderTree()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 100})
	renderTree()

	if !ctx.reorderActive {
		t.Fatal("drag должен быть активен")
	}

	// child2 имеет Depth = 1.
	// targetIndent = Depth * Scale(18) + Scale(4) = 1 * 27 + 6 = 33 px.
	// lineX = inner.X + targetIndent = 6 + 33 = 39 px.
	// Линия индикатора Accent начинается с X = 39!
	// Проверяем пиксель левее индикатора (X = 20, Y = 90): там НЕ должно быть цвета Accent
	accentCol := ctx.Theme.Accent
	idxLeft := (90*ctx.Canvas.Width + 20) * 4
	if ctx.Canvas.Pixels[idxLeft+2] == accentCol.R && ctx.Canvas.Pixels[idxLeft+1] == accentCol.G && ctx.Canvas.Pixels[idxLeft] == accentCol.B {
		t.Fatal("слева от отступа глубины индикатор не должен рисоваться")
	}

	// Проверяем пиксель на индикаторе (X = 60, Y = 90): там ДОЛЖЕН быть цвет Accent
	idxOn := (90*ctx.Canvas.Width + 60) * 4
	if ctx.Canvas.Pixels[idxOn+2] != accentCol.R || ctx.Canvas.Pixels[idxOn+1] != accentCol.G || ctx.Canvas.Pixels[idxOn] != accentCol.B {
		t.Fatalf("на линии-индикаторе с отступом глубины ожидался цвет Accent, получено: R=%d, G=%d, B=%d",
			ctx.Canvas.Pixels[idxOn+2], ctx.Canvas.Pixels[idxOn+1], ctx.Canvas.Pixels[idxOn])
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 100})
	renderTree()
}

func TestListViewReorder_HappyPath(t *testing.T) {
	ctx := newTestContext(600, 400)
	items := []string{"L0", "L1", "L2", "L3", "L4"}
	selected := -1

	var receivedDrag *DragEvent
	callCount := 0
	renderList := func() {
		ctx.Render(func(c *Context) {
			c.ListView("lv_test", len(items), 30, &selected, func(index int, areaPx render.Rect, isSelected bool) {},
				Width(300), Height(200), Reorderable(), OnReorder(func(ev DragEvent) {
					receivedDrag = &ev
					callCount++
				}))
		})
	}

	// Зажимаем элемент 1 (Y ~ 45)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 45})
	renderList()

	// Тащим вниз на элемент 3: Y ~ 105 (DropBelow)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 105})
	renderList()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 105})
	renderList()

	if callCount != 1 || receivedDrag == nil {
		t.Fatalf("ожидался 1 вызов OnReorder для ListView, вызовов: %d", callCount)
	}
	if receivedDrag.FromIndex != 1 || receivedDrag.ToIndex != 3 {
		t.Fatalf("неверные индексы: FromIndex=%d, ToIndex=%d", receivedDrag.FromIndex, receivedDrag.ToIndex)
	}
	if receivedDrag.FromKey != "" || receivedDrag.ToKey != "" {
		t.Fatalf("FromKey и ToKey для ListView должны быть пустыми, получено: %q, %q", receivedDrag.FromKey, receivedDrag.ToKey)
	}
}

func TestReorder_EscapeClosesModalAndResetsDrag(t *testing.T) {
	ctx := newTestContext(600, 400)
	items := []string{"A", "B", "C"}
	selected := -1
	modalOpen := false

	renderAll := func() {
		ctx.Render(func(c *Context) {
			c.Table("tbl_esc", []TableColumn{{Title: "T", WidthDp: 100}}, len(items), &selected, nil, func(row, col int, area render.Rect, isSel bool) {},
				Width(300), Height(200), Reorderable())

			if modalOpen {
				c.Modal("test_modal", &modalOpen, "Тестовое окно", func() {
					c.Label("Содержимое модалки", 14, c.Theme.TextPrimary)
				})
			}
		})
	}

	// 1. Инициируем drag таблицы
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 75})
	renderAll()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 120})
	renderAll()

	if !ctx.reorderActive {
		t.Fatal("drag должен быть активен")
	}

	// 2. Открываем модалку поверх активного drag
	modalOpen = true
	renderAll()
	if !modalOpen {
		t.Fatal("модалка должна быть открыта")
	}

	// 3. Нажимаем Escape: должен сбросить drag и одновременно закрыть модалку
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEscape})
	renderAll()

	if ctx.reorderActive {
		t.Fatal("drag должен быть сброшен по Escape")
	}
	if modalOpen {
		t.Fatal("модалка должна быть закрыта по Escape")
	}
}

func TestTreeViewReorder_RootNodesReorder(t *testing.T) {
	ctx := newTestContext(600, 400)
	roots := []*TreeNode{
		{ID: "root_a", Label: "Проект A"},
		{ID: "root_b", Label: "Проект B"},
	}
	selectedID := ""

	var receivedDrag *DragEvent
	callCount := 0
	renderTree := func() {
		ctx.Render(func(c *Context) {
			c.TreeView("tree_roots", roots, &selectedID, Width(300), Height(200), Reorderable(), OnReorder(func(ev DragEvent) {
				receivedDrag = &ev
				callCount++
			}))
		})
	}

	// 1. Зажимаем root_a (Y ~ 20)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 20})
	renderTree()

	// 2. Тащим вниз на root_b (Y ~ 50, DropBelow)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 50})
	renderTree()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 50})
	renderTree()

	if callCount != 1 || receivedDrag == nil {
		t.Fatalf("ожидался 1 вызов OnReorder для корневых узлов, вызовов: %d", callCount)
	}
	if receivedDrag.FromKey != "root_a" || receivedDrag.ToKey != "root_b" {
		t.Fatalf("неверные ключи корневых узлов: FromKey=%q, ToKey=%q", receivedDrag.FromKey, receivedDrag.ToKey)
	}
	if receivedDrag.FromIndex != -1 || receivedDrag.ToIndex != -1 {
		t.Fatalf("индексы для TreeView должны быть равны -1, получено: FromIndex=%d, ToIndex=%d", receivedDrag.FromIndex, receivedDrag.ToIndex)
	}
	if receivedDrag.Drop != DropBelow {
		t.Fatalf("ожидалась зона DropBelow, получено: %v", receivedDrag.Drop)
	}
}

func TestReorder_EscapeAfterModalClosed(t *testing.T) {
	ctx := newTestContext(600, 400)
	modalOpen := true
	textVal := "значение"

	renderAll := func() {
		ctx.Render(func(c *Context) {
			c.TextInput("txt_field", &textVal, "плейсхолдер", 200)
			if modalOpen {
				c.Modal("modal_temp", &modalOpen, "Окно", func() {
					c.Label("Текст", 14, c.Theme.TextPrimary)
				})
			}
		})
	}

	// Кадр 1: Модалка открыта
	renderAll()
	if !modalOpen {
		t.Fatal("модалка должна быть открыта")
	}

	// Кадр 2: Закрываем модалку по Escape
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEscape})
	renderAll()
	if modalOpen {
		t.Fatal("модалка должна быть закрыта")
	}

	// Кликаем по текстовому полю, чтобы взять фокус
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 20, Y: 10})
	renderAll()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 20, Y: 10})
	renderAll()

	if ctx.FocusID == "" {
		t.Fatal("текстовое поле должно получить фокус")
	}

	// Кадр 3: Нажатие Escape должно снять фокус (оверлей закрыт, нет лага залипания)
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEscape})
	renderAll()

	if ctx.FocusID != "" {
		t.Fatalf("Escape должен сбросить фокус после закрытия модалки, текущий FocusID=%q", ctx.FocusID)
	}
}

func TestListViewExReorder_HappyPath(t *testing.T) {
	ctx := newTestContext(600, 400)
	items := []string{"Ex0", "Ex1", "Ex2", "Ex3", "Ex4"}
	selected := map[int]bool{1: true}
	checked := []bool{false, true, false, false, false} // Элемент 1 отмечен галочкой

	var receivedDrag *DragEvent
	callCount := 0
	renderListEx := func() {
		ctx.Render(func(c *Context) {
			c.ListViewEx("lvex_test", len(items), 30, selected, checked, func(index int, areaPx render.Rect, isSelected bool) {},
				Width(300), Height(200), Reorderable(), OnReorder(func(ev DragEvent) {
					receivedDrag = &ev
					callCount++

					// Правильный паттерн: синхронная перестановка данных, checked и selected
					from, to := ev.FromIndex, ev.ToIndex
					it := items[from]
					items = append(items[:from], items[from+1:]...)
					items = append(items[:to], append([]string{it}, items[to:]...)...)

					chk := checked[from]
					checked = append(checked[:from], checked[from+1:]...)
					checked = append(checked[:to], append([]bool{chk}, checked[to:]...)...)

					newSel := make(map[int]bool)
					if selected[from] {
						newSel[to] = true
					}
					selected = newSel
				}))
		})
	}

	// 1. Зажимаем строку 1 (Y ~ 45, правее чекбокса: X = 50)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 45})
	renderListEx()

	// 2. Тащим вниз на строку 3 (Y = 110 для DropBelow, так как середина строки 3 на Y = 106)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 110})
	renderListEx()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 110})
	renderListEx()

	if callCount != 1 || receivedDrag == nil {
		t.Fatalf("ожидался 1 вызов OnReorder для ListViewEx, вызовов: %d", callCount)
	}
	if receivedDrag.FromIndex != 1 || receivedDrag.ToIndex != 3 {
		t.Fatalf("неверные индексы: FromIndex=%d, ToIndex=%d", receivedDrag.FromIndex, receivedDrag.ToIndex)
	}
	// Проверяем синхронность: перенесённый Ex1 теперь на индексе 3, и его галочка с выделением тоже на индексе 3
	if items[3] != "Ex1" {
		t.Fatalf("ожидалось перемещение Ex1 на индекс 3, получено: %v", items)
	}
	if !checked[3] || checked[1] {
		t.Fatalf("галочка должна была переместиться на индекс 3 вместе с элементом, состояние checked: %v", checked)
	}
	if !selected[3] || selected[1] {
		t.Fatalf("выделение должно было переместиться на индекс 3 вместе с элементом, состояние selected: %v", selected)
	}
}

func TestListViewExReorder_ShortCheckedSliceNoPanic(t *testing.T) {
	ctx := newTestContext(600, 400)
	items := []string{"0", "1", "2", "3", "4"}
	selected := map[int]bool{}
	checked := []bool{true, false} // Длина меньше count (2 < 5)

	callCount := 0
	renderListEx := func() {
		ctx.Render(func(c *Context) {
			c.ListViewEx("lvex_short", len(items), 30, selected, checked, func(index int, areaPx render.Rect, isSelected bool) {},
				Width(300), Height(200), Reorderable(), OnReorder(func(ev DragEvent) {
					callCount++
				}))
		})
	}

	// Зажимаем строку 0, тащим вниз на строку 2
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 15})
	renderListEx()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 75})
	renderListEx()

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 75})
	renderListEx()

	if callCount != 1 {
		t.Fatalf("ожидался успешный вызов OnReorder без паники при коротком срезе checked, вызовов: %d", callCount)
	}
}

func TestLinearDropTarget_Matrix(t *testing.T) {
	c := &Context{}
	count := 5
	itemH := 20
	rowsTop := 100

	// 1. Полный перебор матрицы from (0..4) x target (0..4) x 2 зоны (DropAbove / DropBelow)
	for from := 0; from < count; from++ {
		for target := 0; target < count; target++ {
			// Зона DropAbove (верхняя четверть строки target)
			cursorYAbove := rowsTop + target*itemH + itemH/4
			targetRow, zone, toIdx, noop := c.LinearDropTarget(from, cursorYAbove, rowsTop, itemH, count)
			if targetRow != target {
				t.Fatalf("from=%d target=%d DropAbove: targetRow=%d, ожидали %d", from, target, targetRow, target)
			}
			if zone != DropAbove {
				t.Fatalf("from=%d target=%d DropAbove: zone=%v, ожидали DropAbove", from, target, zone)
			}
			expectedInsertAbove := target
			expectedToAbove := expectedInsertAbove
			if from < expectedInsertAbove {
				expectedToAbove = expectedInsertAbove - 1
			}
			if toIdx != expectedToAbove {
				t.Fatalf("from=%d target=%d DropAbove: toIdx=%d, ожидали %d", from, target, toIdx, expectedToAbove)
			}
			if noop != (toIdx == from) {
				t.Fatalf("from=%d target=%d DropAbove: noop=%v, ожидали %v", from, target, noop, toIdx == from)
			}

			// Зона DropBelow (нижняя четверть строки target)
			cursorYBelow := rowsTop + target*itemH + 3*itemH/4
			targetRowB, zoneB, toIdxB, noopB := c.LinearDropTarget(from, cursorYBelow, rowsTop, itemH, count)
			if targetRowB != target {
				t.Fatalf("from=%d target=%d DropBelow: targetRow=%d, ожидали %d", from, target, targetRowB, target)
			}
			if zoneB != DropBelow {
				t.Fatalf("from=%d target=%d DropBelow: zone=%v, ожидали DropBelow", from, target, zoneB)
			}
			expectedInsertBelow := target + 1
			expectedToBelow := expectedInsertBelow
			if from < expectedInsertBelow {
				expectedToBelow = expectedInsertBelow - 1
			}
			if toIdxB != expectedToBelow {
				t.Fatalf("from=%d target=%d DropBelow: toIdx=%d, ожидали %d", from, target, toIdxB, expectedToBelow)
			}
			if noopB != (toIdxB == from) {
				t.Fatalf("from=%d target=%d DropBelow: noop=%v, ожидали %v", from, target, noopB, toIdxB == from)
			}
		}
	}

	// 2. Граничные значения курсора выше верха и ниже низа
	// Курсор выше верха (-50px)
	trAbove, zAbove, toAbove, _ := c.LinearDropTarget(2, rowsTop-50, rowsTop, itemH, count)
	if trAbove != 0 || zAbove != DropAbove || toAbove != 0 {
		t.Fatalf("курсор выше верха: tr=%d z=%v to=%d, ожидали (0, DropAbove, 0)", trAbove, zAbove, toAbove)
	}

	// Курсор ниже низа (+500px)
	trBelow, zBelow, toBelow, _ := c.LinearDropTarget(2, rowsTop+count*itemH+500, rowsTop, itemH, count)
	if trBelow != count-1 || zBelow != DropBelow || toBelow != count-1 {
		t.Fatalf("курсор ниже низа: tr=%d z=%v to=%d, ожидали (%d, DropBelow, %d)", trBelow, zBelow, toBelow, count-1, count-1)
	}

	// 3. Вырожденные списки
	_, _, _, noopEmpty := c.LinearDropTarget(0, 50, 0, itemH, 0)
	if !noopEmpty {
		t.Fatalf("пустой список должен возвращать noop=true")
	}
	_, _, _, noopZeroH := c.LinearDropTarget(0, 50, 0, 0, count)
	if !noopZeroH {
		t.Fatalf("нулевая высота строки должна возвращать noop=true")
	}
}

func TestApplyReorderAutoScroll_ZeroAndNegativeMaxSc(t *testing.T) {
	// 1. Прямая проверка ApplyReorderAutoScroll при maxSc == 0 и maxSc < 0
	ctx := newTestContext(400, 300)
	for _, maxSc := range []int{0, -50} {
		si := &scrollInfo{y: -5}
		ctx.reorderScrollAccum = 0.75
		// Курсор в верхней и нижней зонах авто-скролла
		ctx.MouseY = 5
		ctx.ApplyReorderAutoScroll(0, 200, si, maxSc)
		if si.y != 0 {
			t.Fatalf("maxSc=%d (top): ожидали si.y=0, получено %d", maxSc, si.y)
		}
		if ctx.reorderScrollAccum != 0 {
			t.Fatalf("maxSc=%d (top): ожидали reorderScrollAccum=0, получено %v", maxSc, ctx.reorderScrollAccum)
		}

		ctx.MouseY = 195
		ctx.ApplyReorderAutoScroll(0, 200, si, maxSc)
		if si.y != 0 {
			t.Fatalf("maxSc=%d (bottom): ожидали si.y=0, получено %d", maxSc, si.y)
		}
		if ctx.reorderScrollAccum != 0 {
			t.Fatalf("maxSc=%d (bottom): ожидали reorderScrollAccum=0, получено %v", maxSc, ctx.reorderScrollAccum)
		}
	}

	// 2. Короткий ListView (5 строк по 28px = 140px целиком влезают в высоту 220px, maxSc == 0):
	// авто-скролл не срабатывает, но сам reorder обязан успешно доставить корректный ToIndex.
	ctxShort := newTestContext(400, 300)
	sel := 0
	var got *DragEvent
	renderShortList := func() {
		ctxShort.Render(func(c *Context) {
			c.ListView("lv_short_fit", 5, 28, &sel, func(index int, area render.Rect, isSel bool) {},
				Width(250), Height(220), Reorderable(), OnReorder(func(ev DragEvent) {
					e := ev
					got = &e
				}))
		})
	}

	// Захватываем строку 0 (Y=14), тащим в нижнюю часть строки 3 (Y=105 -> DropBelow строки 3 -> toIdx=3)
	ctxShort.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 50, Y: 14})
	renderShortList()

	ctxShort.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 50, Y: 105})
	renderShortList()

	siList := ctxShort.getScrollInfo("lv:lv_short_fit")
	if siList.y != 0 {
		t.Fatalf("короткий список не должен скроллиться во время drag: si.y=%d", siList.y)
	}
	if ctxShort.reorderScrollAccum != 0 {
		t.Fatalf("короткий список не должен копить reorderScrollAccum: %v", ctxShort.reorderScrollAccum)
	}

	ctxShort.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 50, Y: 105})
	renderShortList()

	if got == nil {
		t.Fatal("ожидалось событие OnReorder в коротком списке при maxSc == 0")
	}
	if got.FromIndex != 0 || got.ToIndex != 3 || got.Drop != DropBelow {
		t.Fatalf("неверные параметры Reorder в коротком списке: %+v", *got)
	}
	if siList.y != 0 || ctxShort.reorderScrollAccum != 0 {
		t.Fatalf("после дропа в коротком списке si.y=%d, accum=%v, ожидали 0/0", siList.y, ctxShort.reorderScrollAccum)
	}
}
