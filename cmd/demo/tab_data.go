package main

import (
	"fmt"
	"sort"

	"github.com/kirpicheff/anyui"
)

type demoTableRow struct {
	ID      int
	Name    string
	Status  string
	Latency int
}

func makeDemoTableRows() []demoTableRow {
	statuses := []string{"Online", "Idle", "Syncing", "Offline"}
	rows := make([]demoTableRow, 200)
	for i := range rows {
		rows[i] = demoTableRow{
			ID:      i + 1,
			Name:    fmt.Sprintf("Узел cluster-msk-%03d", i+1),
			Status:  statuses[i%len(statuses)],
			Latency: 5 + ((i * 17) % 140),
		}
	}
	return rows
}

func makeDemoTree() []*anyui.TreeNode {
	return []*anyui.TreeNode{
		{
			ID:       "root_proj",
			Label:    "AnyUI Workspace",
			IconSVG:  svgGear,
			Expanded: true,
			Children: []*anyui.TreeNode{
				{
					ID:       "dir_ui",
					Label:    "ui/",
					Expanded: true,
					Children: []*anyui.TreeNode{
						{ID: "f_ctx", Label: "context.go"},
						{ID: "f_data", Label: "controls_data.go"},
						{ID: "f_adv", Label: "controls_advanced_input.go"},
					},
				},
				{
					ID:       "dir_render",
					Label:    "render/",
					Expanded: false,
					Children: []*anyui.TreeNode{
						{ID: "f_canvas", Label: "canvas.go"},
						{ID: "f_svg", Label: "svg.go"},
					},
				},
			},
		},
		{
			ID:       "root_docs",
			Label:    "docs/",
			Expanded: false,
			Children: []*anyui.TreeNode{
				{ID: "f_widgets", Label: "widgets.md"},
				{ID: "f_sys", Label: "system.md"},
			},
		},
	}
}

func (st *demoState) sortDemoTable() {
	col := st.tableSort.Column
	desc := st.tableSort.Desc
	sort.SliceStable(st.tableRows, func(i, j int) bool {
		a, b := st.tableRows[i], st.tableRows[j]
		var less bool
		switch col {
		case 0:
			less = a.ID < b.ID
		case 1:
			less = a.Name < b.Name
		case 2:
			less = a.Status < b.Status
		case 3:
			less = a.Latency < b.Latency
		}
		if desc {
			return !less
		}
		return less
	})
}

func findAndRemoveNode(nodes []*anyui.TreeNode, id string) (*anyui.TreeNode, []*anyui.TreeNode) {
	for i, n := range nodes {
		if n.ID == id {
			res := make([]*anyui.TreeNode, 0, len(nodes)-1)
			res = append(res, nodes[:i]...)
			res = append(res, nodes[i+1:]...)
			return n, res
		}
		if len(n.Children) > 0 {
			if found, rem := findAndRemoveNode(n.Children, id); found != nil {
				n.Children = rem
				return found, nodes
			}
		}
	}
	return nil, nodes
}

func insertNode(nodes []*anyui.TreeNode, targetID string, item *anyui.TreeNode, drop anyui.DropZone) []*anyui.TreeNode {
	if item == nil {
		return nodes
	}
	if targetID == "" {
		res := make([]*anyui.TreeNode, len(nodes)+1)
		copy(res, nodes)
		res[len(nodes)] = item
		return res
	}
	for i, n := range nodes {
		if n.ID == targetID {
			switch drop {
			case anyui.DropAbove:
				res := make([]*anyui.TreeNode, 0, len(nodes)+1)
				res = append(res, nodes[:i]...)
				res = append(res, item)
				res = append(res, nodes[i:]...)
				return res
			case anyui.DropBelow:
				if n.Expanded && len(n.Children) > 0 {
					res := make([]*anyui.TreeNode, 0, len(n.Children)+1)
					res = append(res, item)
					res = append(res, n.Children...)
					n.Children = res
					return nodes
				}
				res := make([]*anyui.TreeNode, 0, len(nodes)+1)
				res = append(res, nodes[:i+1]...)
				res = append(res, item)
				res = append(res, nodes[i+1:]...)
				return res
			case anyui.DropInside:
				if n.ID != item.ID {
					n.Children = append(n.Children, item)
					n.Expanded = true
				}
				return nodes
			}
		}
		if len(n.Children) > 0 {
			n.Children = insertNode(n.Children, targetID, item, drop)
		}
	}
	return nodes
}

func (st *demoState) drawTabData(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_data", func() {
		ui.Label("Table (Drag-Reorder строк, сортировка, ресайз, скролл, DoubleClick/MiddleClick)", anyui.FontSize(15), anyui.Bold())
		cols := []anyui.TableColumn{
			{Title: "ID", WidthDp: 64, MinWidthDp: 48, Sortable: true},
			{Title: "Сервер", WidthDp: 220, MinWidthDp: 100, Sortable: true},
			{Title: "Статус", WidthDp: 120, MinWidthDp: 70, Sortable: true},
			{Title: "Пинг (мс)", WidthDp: 110, MinWidthDp: 70, Sortable: true},
			{Title: "Зона / Примечание", WidthDp: 240, MinWidthDp: 100, Sortable: false},
		}

		handleTableReorder := func(ev anyui.DragEvent) {
			if ev.FromIndex >= 0 && ev.FromIndex < len(st.tableRows) && ev.ToIndex >= 0 && ev.ToIndex < len(st.tableRows) {
				row := st.tableRows[ev.FromIndex]
				tmp := make([]demoTableRow, 0, len(st.tableRows)-1)
				tmp = append(tmp, st.tableRows[:ev.FromIndex]...)
				tmp = append(tmp, st.tableRows[ev.FromIndex+1:]...)

				res := make([]demoTableRow, 0, len(st.tableRows))
				res = append(res, tmp[:ev.ToIndex]...)
				res = append(res, row)
				res = append(res, tmp[ev.ToIndex:]...)
				st.tableRows = res
				st.statusMsg = fmt.Sprintf("Строка перенесена: %d -> %d", ev.FromIndex, ev.ToIndex)
				ui.ShowToast("Таблица: Reorder", st.statusMsg, anyui.ToastInfo)
			}
		}

		tev := ui.Table("tbl_servers", cols, len(st.tableRows), &st.tableSelected, &st.tableSort, func(row, col int, area anyui.Rect, sel bool) {
			r := st.tableRows[row]
			var txt string
			txtCol := th.TextPrimary
			switch col {
			case 0:
				txt = fmt.Sprintf("#%d", r.ID)
				txtCol = th.TextMuted
			case 1:
				txt = r.Name
			case 2:
				txt = r.Status
				if r.Status == "Online" {
					txtCol = th.Success
				} else if r.Status == "Offline" {
					txtCol = th.Danger
				} else if r.Status == "Syncing" {
					txtCol = th.Warning
				}
			case 3:
				txt = fmt.Sprintf("%d ms", r.Latency)
			case 4:
				txt = "msk-dc1 / основной кластер"
				txtCol = th.TextMuted
			}
			ui.DrawText(area.X+ui.Scale(8).Int(), area.Y+ui.Scale(7).Int(), txt, 12, txtCol)
		}, anyui.Height(210), anyui.Reorderable(), anyui.OnReorder(handleTableReorder))
		if tev.SortChanged {
			st.sortDemoTable()
			st.statusMsg = fmt.Sprintf("Таблица отсортирована по колонке %d", st.tableSort.Column)
		}
		if tev.RowClicked >= 0 {
			st.statusMsg = fmt.Sprintf("Выбрана строка: %s", st.tableRows[tev.RowClicked].Name)
		}
		if tev.RowDoubleClicked >= 0 {
			row := st.tableRows[tev.RowDoubleClicked]
			st.statusMsg = fmt.Sprintf("DoubleClick по строке: %s (%d ms)", row.Name, row.Latency)
			ui.ShowToast("Двойной клик", st.statusMsg, anyui.ToastInfo)
		}
		if tev.RowMiddleClicked >= 0 {
			row := st.tableRows[tev.RowMiddleClicked]
			st.statusMsg = fmt.Sprintf("MiddleClick по строке: %s", row.Name)
			ui.ShowToast("Средняя кнопка", st.statusMsg, anyui.ToastSuccess)
		}

		handleTreeDrop := func(ev anyui.DragEvent) {
			var moved *anyui.TreeNode
			if ev.FromWidget == "tree_proj" {
				moved, st.treeRoots = findAndRemoveNode(st.treeRoots, ev.FromKey)
			} else if ev.FromWidget == "tree_fav" {
				moved, st.treeFavRoots = findAndRemoveNode(st.treeFavRoots, ev.FromKey)
			}
			if moved != nil {
				if ev.ToWidget == "tree_proj" {
					st.treeRoots = insertNode(st.treeRoots, ev.ToKey, moved, ev.Drop)
				} else if ev.ToWidget == "tree_fav" {
					st.treeFavRoots = insertNode(st.treeFavRoots, ev.ToKey, moved, ev.Drop)
				}
				zoneName := "внутрь"
				if ev.Drop == anyui.DropAbove {
					zoneName = "выше"
				} else if ev.Drop == anyui.DropBelow {
					zoneName = "ниже"
				}
				st.statusMsg = fmt.Sprintf("Drag&Drop: %s (%s -> %s, %s)", moved.Label, ev.FromWidget, ev.ToWidget, zoneName)
				ui.ShowToast("Drag & Drop", st.statusMsg, anyui.ToastSuccess)
			}
		}

		ui.Spacer(12)
		ui.Label("SplitPane + TreeView (Drag-Reorder, вложение DropInside и смена родителя) + Collapsible & Accordion", anyui.FontSize(16), anyui.Bold())
		ui.SplitPane("split_main", anyui.Horizontal, &st.splitRatio, func() {
			ui.Label("TreeView Проект (AllowReparent: смена родителя и вложение DropInside)", anyui.FontSize(13), anyui.Bold())
			if ui.TreeView("tree_proj", st.treeRoots, &st.treeSelected, anyui.Height(170), anyui.Reorderable(), anyui.AllowReparent(), anyui.OnReorder(handleTreeDrop)) {
				st.statusMsg = fmt.Sprintf("TreeView: %s", st.treeSelected)
			}
			ui.Spacer(6)
			ui.Label("TreeView Избранное (перестановка корневых узлов)", anyui.FontSize(12), anyui.Bold())
			if ui.TreeView("tree_fav", st.treeFavRoots, &st.treeFavSelected, anyui.Height(100), anyui.Reorderable(), anyui.OnReorder(handleTreeDrop)) {
				st.statusMsg = fmt.Sprintf("TreeView Избранное: %s", st.treeFavSelected)
			}
		}, func() {
			ui.Collapsible("coll_info", "Collapsible: Свойства узла", &st.collapsibleOpen, func() {
				ui.Label(fmt.Sprintf("Выбран в проекте: %s", st.treeSelected), anyui.FontSize(12), anyui.TextColor(th.TextMuted))
				ui.Label(fmt.Sprintf("Выбран в избранном: %s", st.treeFavSelected), anyui.FontSize(12), anyui.TextColor(th.TextMuted))
				ui.Label(fmt.Sprintf("Split ratio: %.0f%%", st.splitRatio*100), anyui.FontSize(12), anyui.TextColor(th.TextMuted))
			})
			ui.Spacer(6)
			ui.Accordion("acc_demo", []anyui.AccordionItem{
				{
					ID:    "sec1",
					Title: "Accordion 1: Сеть и порты",
					Draw: func() {
						ui.Label("TCP 443 / QUIC 8443 активны", anyui.FontSize(12), anyui.TextColor(th.TextMuted))
					},
				},
				{
					ID:    "sec2",
					Title: "Accordion 2: Ресурсы",
					Draw: func() {
						ui.Label("CPU ~0% в простое, Pure Go", anyui.FontSize(12), anyui.TextColor(th.TextMuted))
					},
				},
			}, st.accordionOpen)
		}, anyui.Height(330))
	})
}
