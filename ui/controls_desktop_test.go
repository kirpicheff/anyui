package ui

import (
	"testing"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

func TestMenuBarToolbarStatusBarBreadcrumbsPagination(t *testing.T) {
	ctx := newTestContext(800, 600)

	menuClicked := false
	menus := []MenuBarItem{
		{
			Label: "Файл",
			Items: []BarMenuItem{
				{Label: "Новый", Shortcut: "Ctrl+N", OnClick: func() { menuClicked = true }},
				{Separator: true},
				{Label: "Выход", Disabled: true},
			},
		},
		{Label: "Правка"},
	}

	toolClicked := false
	tools := []ToolbarItem{
		{ID: "save", Label: "Сохранить", OnClick: func() { toolClicked = true }},
		{Separator: true},
		{ID: "run", Label: "Запуск", Active: true},
	}

	statusClicked := false
	dotCol := render.RGB(34, 197, 94)
	sections := []StatusBarSection{
		{Text: "Готово", DotColor: &dotCol, Flex: 1},
		{Text: "UTF-8", WidthDp: 80, OnClick: func() { statusClicked = true }},
	}

	page := 2
	bcClicked := -1

	renderFrame := func() {
		ctx.beginFrame()
		ctx.Column(8, func() {
			ctx.MenuBar("mb1", menus)
			ctx.Toolbar("tb1", tools)
			if idx := ctx.Breadcrumbs("bc1", []BreadcrumbItem{
				{Label: "Главная"},
				{Label: "Проекты"},
				{Label: "Детали"},
			}); idx >= 0 {
				bcClicked = idx
			}
			ctx.Pagination("pg1", &page, 10)
			ctx.StatusBar("sb1", sections)
		})
		ctx.endFrame()
	}

	// Кадр 1: начальная отрисовка
	renderFrame()

	// Клик по первому пункту MenuBar ("Файл")
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 25, Y: 15})
	renderFrame()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 25, Y: 15})
	renderFrame()

	if _, ok := menuBarActiveIdx["mb1"]; !ok {
		t.Fatal("ожидалось открытие выпадающего меню MenuBar по клику на 'Файл'")
	}

	// Клик по пункту "Новый" в оверлее MenuBar
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 40, Y: 48})
	renderFrame()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 40, Y: 48})
	renderFrame()

	if !menuClicked {
		t.Fatal("ожидался вызов OnClick пункта меню 'Новый'")
	}

	// Клик по кнопке Toolbar ("Сохранить")
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 30, Y: 55})
	renderFrame()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 30, Y: 55})
	renderFrame()

	if !toolClicked {
		t.Fatal("ожидался вызов OnClick кнопки Toolbar")
	}

	// Клик по первому сегменту Breadcrumbs ("Главная", без шрифта ширина сегмента 8px)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 4, Y: 90})
	renderFrame()
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 4, Y: 90})
	renderFrame()
	if bcClicked != 0 {
		t.Fatalf("ожидался клик по сегменту 0 Breadcrumbs, получено %d", bcClicked)
	}

	// Клик по кнопке ">" в Pagination (y ≈ 124, 5-я или 6-я кнопка)
	// Проверим кнопку "<" (первая кнопка Pagination: x ≈ 16, y ≈ 124)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 16, Y: 124})
	renderFrame()
	if page != 1 {
		t.Fatalf("ожидался переход на страницу 1 по кнопке '<', получено %d", page)
	}

	// Клик по правой секции StatusBar (внутри sr: Y = 161..186)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 760, Y: 170})
	renderFrame()
	if !statusClicked {
		t.Fatal("ожидался вызов OnClick секции StatusBar")
	}
}

func TestLogViewAndChartsRender(t *testing.T) {
	ctx := newTestContext(700, 600)

	entries := []LogEntry{
		{Level: LogInfo, Timestamp: "10:00:00", Text: "Сервер запущен"},
		{Level: LogWarn, Timestamp: "10:00:01", Text: "Высокая загрузка CPU"},
		{Level: LogError, Timestamp: "10:00:02", Text: "Ошибка соединения"},
		{Level: LogDebug, Timestamp: "10:00:03", Text: "Отладочное сообщение"},
	}
	follow := true

	series := []ChartSeries{
		{Name: "RPS", Values: []float64{10, 25, 18, 40, 35}},
		{Name: "Errors", Values: []float64{1, 0, 2, 1, 0}},
	}
	labels := []string{"Пн", "Вт", "Ср", "Чт", "Пт"}

	ctx.beginFrame()
	ctx.Column(8, func() {
		ctx.LogView("lv1", entries, &follow, Height(180))
		ctx.Sparkline("sp1", []float64{5, 12, 8, 19, 14}, Height(40))
		ctx.LineChart("lc1", labels, series, Height(160))
		ctx.BarChart("bc_chart", labels, series, Height(160))
	})
	ctx.endFrame()

	// Проверка фильтрации в LogView
	lvs := getLogViewState("lv1")
	lvs.searchText = "ошибка"
	ctx.beginFrame()
	ctx.LogView("lv1", entries, &follow, Height(180))
	ctx.endFrame()
}

func TestDateTimeAndMultiSelectTagInput(t *testing.T) {
	ctx := newTestContext(600, 500)

	dt := time.Date(2026, 5, 10, 14, 30, 0, 0, time.Local)
	if parsed, ok := parseDateInput("2025-12-31", time.Local); !ok || parsed.Year() != 2025 || parsed.Month() != 12 || parsed.Day() != 31 {
		t.Fatalf("ошибка parseDateInput: ok=%v parsed=%v", ok, parsed)
	}
	if _, ok := parseDateInput("invalid", time.Local); ok {
		t.Fatal("ожидался отказ парсинга некорректной даты")
	}

	items := []string{"Go", "Rust", "TypeScript", "Python"}
	sel := []bool{true, false, false, false}
	tags := []string{"ui", "desktop"}

	ctx.beginFrame()
	ctx.Column(8, func() {
		ctx.DateTimePicker("dtp1", &dt)
		ctx.MultiSelect("ms1", items, &sel)
		ctx.TagInput("ti1", &tags, "Добавить тег...")
	})
	ctx.endFrame()

	// Симуляция добавления тега через TagInput (кадр 1: ввод текста, кадр 2: Enter)
	ctx.beginFrame()
	ctx.FocusID = "ti1"
	ctx.TypedRunes = []rune("golang")
	ctx.TagInput("ti1", &tags, "Добавить тег...")
	ctx.endFrame()

	ctx.beginFrame()
	ctx.FocusID = "ti1"
	ctx.KeyEvents = []KeyPress{{Code: KeyEnter}}
	changed := ctx.TagInput("ti1", &tags, "Добавить тег...")
	ctx.endFrame()
	if !changed || len(tags) != 3 || tags[2] != "golang" {
		t.Fatalf("ожидалось добавление тега 'golang', tags=%v, changed=%v", tags, changed)
	}
}

func TestTreeCheckCascadeAndSelectionEx(t *testing.T) {
	roots := []*TreeNode{
		{
			ID:       "root",
			Label:    "Корень",
			Expanded: true,
			Children: []*TreeNode{
				{ID: "c1", Label: "Ребёнок 1"},
				{ID: "c2", Label: "Ребёнок 2"},
			},
		},
	}

	// Отмечаем одного ребёнка -> родитель должен стать Indeterminate
	SetTreeNodeChecked(roots[0].Children[0], true)
	SyncTreeCheckState(roots)
	if roots[0].Checked || !roots[0].Indeterminate {
		t.Fatalf("ожидалось состояние Indeterminate у родителя, got Checked=%v Indeterminate=%v", roots[0].Checked, roots[0].Indeterminate)
	}

	// Отмечаем второго ребёнка -> родитель становится полностью Checked
	SetTreeNodeChecked(roots[0].Children[1], true)
	SyncTreeCheckState(roots)
	if !roots[0].Checked || roots[0].Indeterminate {
		t.Fatalf("ожидалось состояние Checked=true и Indeterminate=false, got Checked=%v Indeterminate=%v", roots[0].Checked, roots[0].Indeterminate)
	}

	// Проверка рендера ListViewEx, TreeViewEx и TableEx
	ctx := newTestContext(700, 600)
	listSel := map[int]bool{0: true}
	listChk := []bool{true, false, false}
	treeSel := map[string]bool{"root": true}
	tableSel := map[int]bool{0: true}
	tableChk := []bool{false, true, false}

	cols := []TableColumn{
		{Title: "ID", WidthDp: 60, Sortable: true},
		{Title: "Имя", WidthDp: 140, Sortable: true},
		{Title: "Статус", WidthDp: 120},
	}
	sort := TableSort{Column: 0}

	ctx.beginFrame()
	ctx.Column(8, func() {
		ctx.ListViewEx("lv_ex", 3, 28, listSel, listChk, func(i int, area render.Rect, isSel bool) {})
		ctx.TreeViewEx("tv_ex", roots, treeSel, true)
		ctx.TableEx("tbl_ex", cols, 3, &TableExConfig{
			SelectedRows:   tableSel,
			ShowCheckboxes: true,
			CheckedRows:    tableChk,
			FrozenCols:     1,
		}, &sort, func(row, col int, area render.Rect, isSel bool) {})
	})
	ctx.endFrame()

	// Проверка Ctrl+A на ListViewEx
	ctx.beginFrame()
	ctx.FocusID = "lv_ex"
	ctx.KeyEvents = []KeyPress{{Code: KeyA, Mods: platform.Modifiers{Ctrl: true}}}
	changed := ctx.ListViewEx("lv_ex", 3, 28, listSel, listChk, nil)
	ctx.endFrame()
	if !changed || len(listSel) != 3 {
		t.Fatalf("ожидалось выделение всех 3 элементов по Ctrl+A, got len=%d changed=%v", len(listSel), changed)
	}
}
