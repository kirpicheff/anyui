package ui

import (
	"testing"
)

func TestListViewScrollToIndex(t *testing.T) {
	ctx := newTestContext(400, 300)

	var sel int = -1
	count := 100
	itemH := Dp(30)

	// Кадр 1: рендер с ScrollToIndex(50)
	ctx.beginFrame()
	ctx.ListView("lv", count, itemH, &sel, nil, ScrollToIndex(50))
	ctx.endFrame()

	si := ctx.getScrollInfo("lv:lv")
	expectedY := 50 * 30
	maxSc := count*30 - 300
	if expectedY > maxSc {
		expectedY = maxSc
	}
	if si.y != expectedY {
		t.Fatalf("ожидалось смещение si.y=%d, получено %d", expectedY, si.y)
	}

	// Кадр 2: свободная прокрутка колесом мыши при неизменном ScrollToIndex(50)
	ctx.MouseX, ctx.MouseY = 50, 50
	ctx.WheelY = 1.0 // крутим вверх
	ctx.beginFrame()
	ctx.ListView("lv", count, itemH, &sel, nil, ScrollToIndex(50))
	ctx.endFrame()

	if si.y >= expectedY {
		t.Fatalf("ожидалось уменьшение si.y при скролле колесом вверх, получено %d", si.y)
	}

	// Кадр 3: переход к новому индексу ScrollToIndex(10)
	ctx.WheelY = 0
	ctx.beginFrame()
	ctx.ListView("lv", count, itemH, &sel, nil, ScrollToIndex(10))
	ctx.endFrame()

	expectedY10 := 10 * 30
	if si.y != expectedY10 {
		t.Fatalf("ожидалось смещение si.y=%d, получено %d", expectedY10, si.y)
	}

	// Кадр 4: сброс через ScrollToIndex(-1)
	ctx.beginFrame()
	ctx.ListView("lv", count, itemH, &sel, nil, ScrollToIndex(-1))
	ctx.endFrame()

	if si.lastScrollIndex != -1 {
		t.Fatalf("ожидался сброс lastScrollIndex=-1, получено %d", si.lastScrollIndex)
	}
}

func TestTableScrollToIndex(t *testing.T) {
	ctx := newTestContext(500, 300)

	cols := []TableColumn{
		{Title: "ID", WidthDp: 100},
		{Title: "Name", WidthDp: 200},
	}
	rowCount := 100
	var sel int = -1

	// Кадр 1: рендер таблицы с ScrollToIndex(25)
	ctx.beginFrame()
	ctx.Table("tbl", cols, rowCount, &sel, nil, nil, ScrollToIndex(25))
	ctx.endFrame()

	si := ctx.getScrollInfo("tbl:tbl")
	expectedY := 25 * 30
	headerH := 32
	viewH := 300 - headerH - 2
	maxSc := rowCount*30 - viewH
	if expectedY > maxSc {
		expectedY = maxSc
	}
	if si.y != expectedY {
		t.Fatalf("ожидалось смещение таблицы si.y=%d, получено %d", expectedY, si.y)
	}
}

func TestScrollViewScrollToID(t *testing.T) {
	ctx := newTestContext(400, 200)

	drawContent := func() {
		for i := 0; i < 20; i++ {
			ctx.allocRect(200, 30)
		}
		ctx.Anchor("section_marker")
		ctx.Button("btn_target", "Целевая кнопка", false, 120)
		for i := 0; i < 20; i++ {
			ctx.allocRect(200, 30)
		}
	}

	// Кадр 1: первичный рендер без скролла
	ctx.beginFrame()
	ctx.ScrollView("sv", 400, 200, drawContent)
	ctx.endFrame()

	// Кадр 2: декларативная прокрутка к якорю section_marker
	ctx.beginFrame()
	ctx.ScrollView("sv", 400, 200, drawContent, ScrollToID("section_marker"))
	ctx.endFrame()

	si := ctx.getScrollInfo("sv:sv")
	if si.y <= 0 {
		t.Fatalf("ожидалось смещение ScrollView si.y > 0 при ScrollToID, получено %d", si.y)
	}

	savedY := si.y

	// Кадр 3: ScrollToID к кнопке btn_target (или короткому id)
	ctx.beginFrame()
	ctx.ScrollView("sv", 400, 200, drawContent, ScrollToID("btn_target"))
	ctx.endFrame()

	if si.y < savedY {
		t.Fatalf("ожидалось смещение к кнопке btn_target >= savedY (%d), получено %d", savedY, si.y)
	}
}
