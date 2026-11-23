package ui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

func TestCrossWidgetDrop_TreeViewToTable(t *testing.T) {
	ctx := newTestContext(700, 400)
	roots := []*TreeNode{
		{ID: "node_a", Label: "Сервер Alpha"},
		{ID: "node_b", Label: "Сервер Beta"},
	}
	selTree := ""
	selRow := -1
	cols := []TableColumn{{Title: "Name", WidthDp: 200}}
	rowCount := 2

	var gotDrop *DragEvent
	var gotTableEv TableEvent

	renderAll := func() {
		ctx.Render(func(c *Context) {
			c.Row(10, 220, func() {
				c.TreeView("tv_src", roots, &selTree, Width(240), Height(200),
					DragSource("server_node", func(_ int, key string) DragPayload {
						return DragPayload{Type: "server_node", Data: "payload:" + key, Label: key}
					}),
				)
				tev := c.Table("tbl_dst", cols, rowCount, &selRow, nil,
					func(row, col int, area render.Rect, isSel bool) {},
					Width(300), Height(200),
					AcceptDrop("server_node"),
					OnDrop(func(ev DragEvent) {
						gotDrop = &ev
					}),
				)
				if tev.Drag != nil {
					gotTableEv = tev
				}
			})
		})
	}

	// 1. Зажимаем node_a в TreeView (x=40, y=14)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 14})
	renderAll()

	// 2. Перетаскиваем в Table (x=280, y=70 -> строка 1, header=32, rowH=30)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 280, Y: 70})
	renderAll()
	if !ctx.reorderActive || ctx.reorderPayload == nil || ctx.reorderPayload.Type != "server_node" {
		t.Fatalf("ожидался активный DragPayload типа server_node, active=%v payload=%+v", ctx.reorderActive, ctx.reorderPayload)
	}

	// 3. Отпускаем над Table
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 280, Y: 70})
	renderAll()

	if gotDrop == nil || gotTableEv.Drag == nil {
		t.Fatalf("ожидалось событие OnDrop и TableEvent.Drag, gotDrop=%v tableDrag=%v", gotDrop, gotTableEv.Drag)
	}
	if gotDrop.FromWidget != "tv_src" || gotDrop.ToWidget != "tbl_dst" {
		t.Fatalf("неверные виджеты: %+v", gotDrop)
	}
	if gotDrop.FromKey != "node_a" || gotDrop.FromIndex != -1 {
		t.Fatalf("неверный источник: FromKey=%q FromIndex=%d", gotDrop.FromKey, gotDrop.FromIndex)
	}
	if gotDrop.Payload == nil || gotDrop.Payload.Data != "payload:node_a" || gotDrop.Payload.Label != "node_a" {
		t.Fatalf("неверный payload: %+v", gotDrop.Payload)
	}
	if gotDrop.ToIndex < 0 || gotDrop.ToIndex > rowCount {
		t.Fatalf("ToIndex вне допустимого диапазона [0..%d]: %d", rowCount, gotDrop.ToIndex)
	}
}

func TestCrossWidgetDrop_TypeMismatchAndDisabledAndEmptyTarget(t *testing.T) {
	ctx := newTestContext(700, 400)
	selList := -1
	selRow := -1
	cols := []TableColumn{{Title: "Col", WidthDp: 200}}

	// 1. Несовпадающий тип AcceptDrop игнорируется
	dropCount := 0
	renderMismatch := func() {
		ctx.Render(func(c *Context) {
			c.Row(10, 200, func() {
				c.ListView("lv_src", 2, 30, &selList, nil, Width(200), Height(180),
					DragSource("type_a", nil),
				)
				c.Table("tbl_empty", cols, 0, &selRow, nil, nil, Width(240), Height(180),
					AcceptDrop("type_b"),
					OnDrop(func(ev DragEvent) { dropCount++ }),
				)
			})
		})
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 15})
	renderMismatch()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 260, Y: 60})
	renderMismatch()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 260, Y: 60})
	renderMismatch()
	if dropCount != 0 {
		t.Fatalf("при несовпадении типа OnDrop не должен вызываться, получено %d", dropCount)
	}

	// 2. Совпадающий тип в пустую Table (count == 0) и обратный порядок отрисовки (приёмник раньше источника)
	var emptyDrop *DragEvent
	renderEmptyReverse := func() {
		ctx.Render(func(c *Context) {
			c.Row(10, 200, func() {
				c.Table("tbl_empty", cols, 0, &selRow, nil, nil, Width(240), Height(180),
					AcceptDrop("type_a"),
					OnDrop(func(ev DragEvent) { emptyDrop = &ev }),
				)
				c.ListView("lv_src", 2, 30, &selList, nil, Width(200), Height(180),
					DragSource("type_a", func(idx int, _ string) DragPayload {
						return DragPayload{Type: "type_a", Data: idx, Label: "Item"}
					}),
				)
			})
		})
	}
	// Источник ListView теперь справа: x ~ 270, y ~ 15; приёмник Table слева: x ~ 80, y ~ 70
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 270, Y: 15})
	renderEmptyReverse()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 80, Y: 70})
	renderEmptyReverse()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 80, Y: 70})
	renderEmptyReverse()

	if emptyDrop == nil {
		t.Fatal("ожидался drop в пустую Table при обратном порядке отрисовки")
	}
	if emptyDrop.Drop != DropInside || emptyDrop.ToIndex != 0 || emptyDrop.FromIndex != 0 {
		t.Fatalf("ожидались DropInside, ToIndex=0, FromIndex=0; получено %+v", emptyDrop)
	}

	// 3. Disabled-приёмник не принимает внешний drop
	disabledDrops := 0
	renderDisabled := func() {
		ctx.Render(func(c *Context) {
			c.Row(10, 200, func() {
				c.ListView("lv_src2", 2, 30, &selList, nil, Width(200), Height(180),
					DragSource("type_a", nil),
				)
				c.TreeView("tv_dis_dst", nil, nil, Width(240), Height(180),
					Disabled(),
					AcceptDrop("type_a"),
					OnDrop(func(ev DragEvent) { disabledDrops++ }),
				)
			})
		})
	}
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 15})
	renderDisabled()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 260, Y: 60})
	renderDisabled()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 260, Y: 60})
	renderDisabled()
	if disabledDrops != 0 {
		t.Fatalf("Disabled-приёмник не должен принимать внешний drop, получено %d", disabledDrops)
	}
}

func TestCrossWidgetDrop_ListViewToTreeViewInside(t *testing.T) {
	ctx := newTestContext(700, 400)
	selList := -1
	selTree := ""
	roots := []*TreeNode{
		{ID: "folder_1", Label: "Папка 1"},
	}

	var gotDrop *DragEvent
	renderAll := func() {
		ctx.Render(func(c *Context) {
			c.Row(10, 200, func() {
				c.ListView("lv_items", 2, 30, &selList, nil, Width(200), Height(180),
					DragSource("item", func(idx int, _ string) DragPayload {
						return DragPayload{Type: "item", Data: idx}
					}),
				)
				c.TreeView("tv_folders", roots, &selTree, Width(240), Height(180),
					AllowReparent(),
					AcceptDrop("item"),
					OnDrop(func(ev DragEvent) { gotDrop = &ev }),
				)
			})
		})
	}

	// Зажимаем строку 1 в ListView (y=45), переносим в середину folder_1 в TreeView (x=260, y=18 -> DropInside)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 45})
	renderAll()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 260, Y: 18})
	renderAll()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 260, Y: 18})
	renderAll()

	if gotDrop == nil {
		t.Fatal("ожидался drop из ListView в TreeView")
	}
	if gotDrop.FromWidget != "lv_items" || gotDrop.ToWidget != "tv_folders" {
		t.Fatalf("неверные виджеты: %+v", gotDrop)
	}
	if gotDrop.FromIndex != 1 || gotDrop.ToKey != "folder_1" || gotDrop.Drop != DropInside {
		t.Fatalf("ожидались FromIndex=1, ToKey=folder_1, DropInside; получено %+v", gotDrop)
	}
	if gotDrop.Payload == nil || gotDrop.Payload.Data != 1 {
		t.Fatalf("неверный payload: %+v", gotDrop.Payload)
	}
}

func TestCrossWidgetDrop_TableExInsideModal(t *testing.T) {
	ctx := newTestContext(800, 600)
	cols := []TableColumn{{Title: "Name", WidthDp: 120}}
	open := true

	var gotDrop *DragEvent
	var gotTableEv TableEvent

	renderModal := func() {
		ctx.Render(func(c *Context) {
			c.Modal("test_modal", &open, "Диалог переноса", func() {
				c.Row(10, 200, func() {
					c.TableEx("src_table", cols, 3, nil, nil, nil,
						Width(180), Height(160),
						DragSource("transfer_item", func(idx int, _ string) DragPayload {
							return DragPayload{Type: "transfer_item", Data: idx, Label: "File" + string(rune('A'+idx))}
						}),
					)
					tev := c.TableEx("dst_table", cols, 2, nil, nil, nil,
						Width(180), Height(160),
						AcceptDrop("transfer_item"),
						OnDrop(func(ev DragEvent) {
							gotDrop = &ev
						}),
					)
					if tev.Drag != nil {
						gotTableEv = tev
					}
				})
			}, Width(500), Height(350))
		})
	}

	// 1. Зажимаем строку 0 в src_table внутри Modal
	// Модалка центрирована: box.X = (800 - 500) / 2 = 150, box.Y = (600 - 350) / 2 = 125.
	// body: box.X + 16 = 166, box.Y + header(40) + 16 = 181.
	// src_table: x ~ 170..340, y ~ 181..340.
	// Заголовок таблицы TableEx: 32 px. Строка 0: y ~ 181 + 32 + 10 = 223.
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 200, Y: 225})
	renderModal()

	// 2. Перетаскиваем в dst_table: x ~ 166 + 180 + 10 + 20 = 376, y ~ 225.
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 380, Y: 225})
	renderModal()

	if !ctx.reorderActive || ctx.reorderPayload == nil || ctx.reorderPayload.Type != "transfer_item" {
		t.Fatalf("ожидался активный DragPayload transfer_item, active=%v payload=%+v", ctx.reorderActive, ctx.reorderPayload)
	}
	if ctx.reorderPayload.Label != "FileA" {
		t.Fatalf("ожидался Label 'FileA', получено %q", ctx.reorderPayload.Label)
	}

	// 3. Отпускаем ЛКМ над dst_table
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 380, Y: 225})
	renderModal()

	if gotDrop == nil || gotTableEv.Drag == nil {
		t.Fatalf("ожидалось успешное срабатывание OnDrop в TableEx внутри Modal, gotDrop=%v, tableDrag=%v", gotDrop, gotTableEv.Drag)
	}
	if gotDrop.FromWidget != "src_table" || gotDrop.ToWidget != "dst_table" {
		t.Fatalf("неверные виджеты: From=%q, To=%q", gotDrop.FromWidget, gotDrop.ToWidget)
	}
	if gotDrop.FromIndex != 0 {
		t.Fatalf("ожидался FromIndex=0, получено %d", gotDrop.FromIndex)
	}
	if gotDrop.Payload == nil || gotDrop.Payload.Label != "FileA" {
		t.Fatalf("неверный payload: %+v", gotDrop.Payload)
	}
}
