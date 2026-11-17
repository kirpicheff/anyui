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

func (st *demoState) drawTabData(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_data", 0, 0, func() {
		ui.Label("Table (сортировка, ресайз, горизонтальный скролл, DoubleClick/MiddleClick, навигация клавишами)", anyui.FontSize(15), anyui.Bold())
		cols := []anyui.TableColumn{
			{Title: "ID", WidthDp: 64, MinWidthDp: 48, Sortable: true},
			{Title: "Сервер", WidthDp: 220, MinWidthDp: 100, Sortable: true},
			{Title: "Статус", WidthDp: 120, MinWidthDp: 70, Sortable: true},
			{Title: "Пинг (мс)", WidthDp: 110, MinWidthDp: 70, Sortable: true},
			{Title: "Зона / Примечание", WidthDp: 240, MinWidthDp: 100, Sortable: false},
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
			ui.DrawText(area.X+ui.S(8), area.Y+ui.S(7), txt, 12, txtCol)
		}, anyui.Height(210))
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

		ui.Spacer(12)
		ui.Label("SplitPane + TreeView + Collapsible & Accordion", anyui.FontSize(16), anyui.Bold())
		ui.SplitPane("split_main", anyui.SplitHorizontal, &st.splitRatio, func() {
			ui.Label("TreeView (двойной клик и стрелки ←/→/↑/↓/Space)", anyui.FontSize(13), anyui.Bold())
			if ui.TreeView("tree_proj", st.treeRoots, &st.treeSelected, anyui.Height(220)) {
				st.statusMsg = fmt.Sprintf("TreeView: %s", st.treeSelected)
			}
		}, func() {
			ui.Collapsible("coll_info", "Collapsible: Свойства узла", &st.collapsibleOpen, func() {
				ui.Label(fmt.Sprintf("Выбран: %s", st.treeSelected), anyui.FontSize(12), anyui.TextColor(th.TextMuted))
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
			}, &st.accordionIdx)
		}, anyui.Height(280))
	})
}

