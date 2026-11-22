package main

import (
	"fmt"
	"time"

	"github.com/kirpicheff/anyui"
)

type desktopTabState struct {
	init          bool
	page          int
	followTail    bool
	logs          []anyui.LogEntry
	dt            time.Time
	multiSel      map[string]bool
	tags          []string
	listMultiSel  map[int]bool
	listMultiChk  []bool
	treeMultiSel  map[string]bool
	treeRoots     []*anyui.TreeNode
	tableSel      map[int]bool
	tableChk      []bool
	tableSort     anyui.TableSort
	tableCells    [][]string
	activeTool    int
	lastBreadcrumb string
}

var deskTab = desktopTabState{
	page:       2,
	followTail: true,
	dt:         time.Now(),
	multiSel:   map[string]bool{"go": true, "cpp": true},
	tags:       []string{"Go", "AnyUI", "Desktop"},
	listMultiSel: map[int]bool{0: true},
	listMultiChk: []bool{true, false, true, false, false, false},
	treeMultiSel: map[string]bool{"src": true},
	tableSel:     map[int]bool{0: true},
	tableChk:     []bool{true, false, false, true},
	tableCells: [][]string{
		{"101", "Сервер API", "Активен", "12 ms"},
		{"102", "База данных", "Активен", "4 ms"},
		{"103", "Кеш Redis", "Предупр.", "1 ms"},
		{"104", "Очередь задач", "Активен", "8 ms"},
	},
	logs: []anyui.LogEntry{
		{Level: anyui.LogInfo, Timestamp: "10:00:01", Text: "Инициализация подсистемы рендеринга"},
		{Level: anyui.LogDebug, Timestamp: "10:00:02", Text: "Загружен кеш шрифтов и SVG-иконок"},
		{Level: anyui.LogWarn, Timestamp: "10:00:04", Text: "Высокая нагрузка на пул воркеров (82%)"},
		{Level: anyui.LogError, Timestamp: "10:00:05", Text: "Таймаут подключения к узлу #4"},
		{Level: anyui.LogInfo, Timestamp: "10:00:06", Text: "Повторное подключение выполнено успешно"},
	},
}

func ensureDesktopTree() []*anyui.TreeNode {
	if deskTab.treeRoots != nil {
		return deskTab.treeRoots
	}
	deskTab.treeRoots = []*anyui.TreeNode{
		{
			ID:       "src",
			Label:    "Исходники (ui/)",
			Expanded: true,
			Children: []*anyui.TreeNode{
				{ID: "desktop", Label: "controls_desktop.go", Checked: true},
				{ID: "charts", Label: "controls_charts.go"},
				{ID: "logview", Label: "controls_logview.go", Checked: true},
			},
		},
		{
			ID:       "docs",
			Label:    "Документация",
			Expanded: true,
			Children: []*anyui.TreeNode{
				{ID: "widgets_md", Label: "widgets.md"},
			},
		},
	}
	anyui.SyncTreeCheckState(deskTab.treeRoots)
	return deskTab.treeRoots
}

func (st *demoState) drawTabDesktop(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("scroll_desktop", func() {
		ui.Column(14, func() {
			ui.Label("Фаза 7.1: Десктоп-контролы, графики и мультивыбор", anyui.FontSize(16), anyui.Bold())

			// 1. MenuBar + Toolbar + StatusBar
			ui.MenuBar("demo_menubar", []anyui.MenuBarItem{
				{
					Label: "Файл",
					Items: []anyui.MenuBarEntry{
						{Label: "Новый проект", Shortcut: "Ctrl+N", OnClick: func() { st.statusMsg = "MenuBar: Новый проект" }},
						{Label: "Открыть...", Shortcut: "Ctrl+O", OnClick: func() { st.statusMsg = "MenuBar: Открыть" }},
						{Separator: true},
						{Label: "Автосохранение", Checked: true, OnClick: func() { st.statusMsg = "MenuBar: Автосохранение" }},
						{Label: "Экспорт", SubItems: []anyui.MenuBarEntry{
							{Label: "В PNG", OnClick: func() { st.statusMsg = "Экспорт в PNG" }},
							{Label: "В SVG", OnClick: func() { st.statusMsg = "Экспорт в SVG" }},
						}},
					},
				},
				{
					Label: "Вид",
					Items: []anyui.MenuBarEntry{
						{Label: "Добавить лог", OnClick: func() {
							deskTab.logs = append(deskTab.logs, anyui.LogEntry{
								Level:     anyui.LogInfo,
								Timestamp: time.Now().Format("15:04:05"),
								Text:      fmt.Sprintf("Событие #%d добавлено из меню", len(deskTab.logs)+1),
							})
						}},
					},
				},
			})

			ui.Toolbar("demo_toolbar", []anyui.ToolbarItem{
				{ID: "add", Label: "Добавить лог", IconSVG: svgPlus, Tooltip: "Добавить строку в LogView", Active: deskTab.activeTool == 0, OnClick: func() {
					deskTab.activeTool = 0
					deskTab.logs = append(deskTab.logs, anyui.LogEntry{
						Level:     anyui.LogInfo,
						Timestamp: time.Now().Format("15:04:05"),
						Text:      fmt.Sprintf("Новая запись #%d", len(deskTab.logs)+1),
					})
				}},
				{Separator: true},
				{ID: "warn", Label: "Предупреждение", IconSVG: svgStar, Tooltip: "Добавить Warn-лог", Active: deskTab.activeTool == 1, OnClick: func() {
					deskTab.activeTool = 1
					deskTab.logs = append(deskTab.logs, anyui.LogEntry{
						Level:     anyui.LogWarn,
						Timestamp: time.Now().Format("15:04:05"),
						Text:      "Предупреждение из Toolbar",
					})
				}},
			})

			// Breadcrumbs и Pagination
			bcID := ui.Breadcrumbs("demo_bc", []anyui.BreadcrumbItem{
				{ID: "home", Label: "Главная"},
				{ID: "cluster", Label: "Кластер"},
				{ID: "nodes", Label: "Узлы"},
				{ID: "monitoring", Label: "Мониторинг"},
			})
			if bcID != "" {
				deskTab.lastBreadcrumb = fmt.Sprintf("Клик по сегменту %q", bcID)
			}
			ui.RowCentered(12, 32, func() {
				ui.Pagination("demo_pag", &deskTab.page, 12, anyui.Width(340))
				ui.Label(fmt.Sprintf("Стр. %d / 12 %s", deskTab.page, deskTab.lastBreadcrumb), anyui.FontSize(12), anyui.TextColor(th.TextMuted))
			})

			// 2. Графики
			ui.Label("Графики: Sparkline, LineChart, BarChart", anyui.FontSize(13), anyui.Bold())
			ui.Sparkline("demo_spark", []float64{12, 18, 14, 26, 22, 31, 28, 39, 35, 44}, anyui.Height(42))

			chartLabels := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}
			chartSeries := []anyui.ChartSeries{
				{Name: "Трафик", Values: []float64{120, 185, 160, 240, 210, 290, 265}},
				{Name: "Ошибки", Values: []float64{12, 8, 15, 6, 9, 4, 5}},
			}
			ui.RowFlex(12, 200,
				anyui.FlexItem{Weight: 1, Fn: func() {
					ui.LineChart("demo_linechart", chartLabels, chartSeries, anyui.Height(195))
				}},
				anyui.FlexItem{Weight: 1, Fn: func() {
					ui.BarChart("demo_barchart", chartLabels, chartSeries, anyui.Height(195))
				}},
			)

			// 3. LogView
			ui.Label("LogView (поиск, Follow tail, Ctrl+A / Ctrl+C)", anyui.FontSize(13), anyui.Bold())
			ui.LogView("demo_logview", deskTab.logs, &deskTab.followTail, anyui.Height(170))

			// 4. DateTimePicker, MultiSelect и TagInput
			ui.Label("DateTimePicker, MultiSelect и TagInput", anyui.FontSize(13), anyui.Bold())
			ui.DateTimePicker("demo_dtp", &deskTab.dt)
			msItems := []anyui.SelectItem{
				{Key: "go", Label: "Go"},
				{Key: "rust", Label: "Rust"},
				{Key: "cpp", Label: "C++"},
				{Key: "ts", Label: "TypeScript"},
				{Key: "py", Label: "Python"},
			}
			ui.Row(12, 36, func() {
				ui.MultiSelect("demo_ms", msItems, deskTab.multiSel, anyui.Width(260))
				ui.TagInput("demo_tags", &deskTab.tags, "Введите тег и Enter...")
			})

			// 5. Мультивыбор в ListViewEx, TreeViewEx и TableEx
			ui.Label("Мультивыбор (Ctrl+Клик, Shift+Клик, Ctrl+A) и чекбоксы", anyui.FontSize(13), anyui.Bold())
			ui.RowFlex(12, 190,
				anyui.FlexItem{Weight: 1, Fn: func() {
					ui.ListViewEx("demo_lvex", len(deskTab.listMultiChk), 28, &anyui.ListViewExConfig{
						Selected:       deskTab.listMultiSel,
						ShowCheckboxes: true,
						Checked:        deskTab.listMultiChk,
					}, func(idx int, area anyui.Rect, isSel bool) {
						ui.DrawText(anyui.Px(area.X+6), anyui.Px(area.Y+7), fmt.Sprintf("Элемент #%d", idx+1), 12, th.TextPrimary)
					}, anyui.Height(185))
				}},
				anyui.FlexItem{Weight: 1, Fn: func() {
					ui.TreeViewEx("demo_tvex", ensureDesktopTree(), &anyui.TreeViewExConfig{
						SelectedIDs:    deskTab.treeMultiSel,
						ShowCheckboxes: true,
					}, anyui.Height(185))
				}},
			)

			cols := []anyui.TableColumn{
				{Title: "ID", WidthDp: 65, Sortable: true},
				{Title: "Сервис", WidthDp: 160, Sortable: true},
				{Title: "Статус", WidthDp: 120},
				{Title: "Пинг", WidthDp: 100},
			}
			ui.TableEx("demo_tblex", cols, len(deskTab.tableCells), &anyui.TableExConfig{
				Selected:             deskTab.tableSel,
				ShowCheckboxes:       true,
				Checked:              deskTab.tableChk,
				FrozenCols:           1,
				AutoFitOnDoubleClick: true,
				GetCellText: func(row, col int) string {
					return deskTab.tableCells[row][col]
				},
				OnCellEdit: func(row, col int, newVal string) {
					deskTab.tableCells[row][col] = newVal
				},
			}, &deskTab.tableSort, func(row, col int, area anyui.Rect, isSel bool) {
				ui.DrawText(anyui.Px(area.X+8), anyui.Px(area.Y+8), deskTab.tableCells[row][col], 12, th.TextPrimary)
			}, anyui.Height(175))

			// 6. StatusBar
			okCol := th.Success
			ui.StatusBar("demo_statusbar", []anyui.StatusBarSection{
				{Text: "Подключено", DotColor: &okCol, Flex: 1},
				{Text: fmt.Sprintf("Стр: %d", deskTab.page), WidthDp: 80},
				{Text: "UTF-8", WidthDp: 70, AlignEnd: true},
			})
		})
	})
}
