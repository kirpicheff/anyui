package ui

import (
	"testing"
	"time"

	"github.com/kirpicheff/anyui/render"
	"golang.org/x/image/font/gofont/goregular"
)

func TestTableAndTreeViewDisabledBlocksMouseAndFocus(t *testing.T) {
	ctx := newTestContext(500, 400)
	cols := []TableColumn{{Title: "Name", WidthDp: 120, Sortable: true}}
	selRow := -1
	sort := TableSort{Column: 0}

	// Клик по заблокированной таблице не должен ставить фокус, выбирать строку или менять сортировку
	ctx.beginFrame()
	ctx.MouseX, ctx.MouseY = 40, 50
	ctx.MousePressed = true
	ev := ctx.Table("tbl_dis", cols, 5, &selRow, &sort, nil, Height(120), Disabled(), Bg(render.RGB(10, 20, 30)))
	ctx.endFrame()

	if ctx.FocusID == "tbl_dis" {
		t.Fatalf("заблокированная Table не должна получать фокус")
	}
	if selRow != -1 || ev.RowClicked != -1 || ev.SortChanged {
		t.Fatalf("заблокированная Table не должна реагировать на клик: selRow=%d ev=%+v", selRow, ev)
	}

	// Клик по заблокированному дереву не должен ставить фокус или выбирать узел
	roots := []*TreeNode{{ID: "r1", Label: "Root"}}
	selNode := ""
	ctx.beginFrame()
	ctx.MouseX, ctx.MouseY = 40, 20
	ctx.MousePressed = true
	changed := ctx.TreeView("tv_dis", roots, &selNode, Height(120), Disabled(), Bg(render.RGB(10, 20, 30)))
	ctx.endFrame()

	if ctx.FocusID == "tv_dis" {
		t.Fatalf("заблокированный TreeView не должен получать фокус")
	}
	if changed || selNode != "" {
		t.Fatalf("заблокированный TreeView не должен реагировать на клик: changed=%v selNode=%q", changed, selNode)
	}
}

func TestTitleBarResolveIDInScope(t *testing.T) {
	ctx := newTestContext(500, 200)
	ctx.beginFrame()
	ctx.IDScope("winA", func() {
		ctx.TitleBar("tb", "Заголовок", TitleBarConfig{})
	})
	ctx.endFrame()

	found := false
	for _, hr := range ctx.hitRegions {
		if hr.id == "winA/tb:close" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("ожидалась регистрация hit-области winA/tb:close с учётом IDScope")
	}
}

func TestTextWidgetsWidthHeightAndTextAreaFocusBorder(t *testing.T) {
	ctx := newTestContext(600, 400)
	fm, err := render.NewFontFromBytes(goregular.TTF)
	if err != nil {
		t.Fatalf("не удалось загрузить тестовый шрифт: %v", err)
	}
	ctx.Font = fm

	// Проверка явных Width/Height у Label (без WordWrap), StatusDot, SelectableLabel и Hyperlink
	ctx.beginFrame()
	var afterLabelY, afterDotY, afterSelY, afterLinkY int
	ctx.Column(0, func() {
		ctx.LabelOpts("Текст", Width(200), Height(40), TextAlignCenter())
		afterLabelY = ctx.topLayout().cursorY

		ctx.StatusDot(render.RGB(0, 200, 0), "OK", Width(150), Height(30))
		afterDotY = ctx.topLayout().cursorY

		ctx.SelectableLabel("sl1", "Выделяемый", Width(180), Height(36))
		afterSelY = ctx.topLayout().cursorY

		ctx.Hyperlink("hl1", "Ссылка", Width(160), Height(28))
		afterLinkY = ctx.topLayout().cursorY
	})
	ctx.endFrame()

	if afterLabelY != 40 {
		t.Fatalf("ожидалась высота Label 40 px, получено %d", afterLabelY)
	}
	if afterDotY-afterLabelY != 30 {
		t.Fatalf("ожидалась высота StatusDot 30 px, получено %d", afterDotY-afterLabelY)
	}
	if afterSelY-afterDotY != 36 {
		t.Fatalf("ожидалась высота SelectableLabel 36 px, получено %d", afterSelY-afterDotY)
	}
	if afterLinkY-afterSelY != 28 {
		t.Fatalf("ожидалась высота Hyperlink 28 px, получено %d", afterLinkY-afterSelY)
	}

	// Проверка FocusBorderColor у TextArea
	val := "hello"
	focusCol := render.RGB(255, 10, 20)
	ctx.beginFrame()
	ctx.FocusID = "ta_focus"
	ctx.TextArea("ta_focus", &val, "", Width(200), Height(80), FocusBorderColor(focusCol))
	ctx.endFrame()

	// Верхняя граница прямоугольника (x=100, y=0) должна быть отрисована цветом focusCol
	off := (0*ctx.Canvas.Width + 100) * 4
	gotB, gotG, gotR := ctx.Canvas.Pixels[off], ctx.Canvas.Pixels[off+1], ctx.Canvas.Pixels[off+2]
	if gotR != focusCol.R || gotG != focusCol.G || gotB != focusCol.B {
		t.Fatalf("ожидался цвет рамки TextArea %v, получено RGB(%d,%d,%d)", focusCol, gotR, gotG, gotB)
	}
}

func TestDateTimePickerWidthAndMultiSelectStyles(t *testing.T) {
	ctx := newTestContext(600, 300)
	dt := time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC)
	sel := map[string]bool{"a": true}
	items := SelectItems("a", "b")

	ctx.beginFrame()
	ctx.Column(8, func() {
		ctx.DateTimePicker("dtp1", &dt, Width(320))
		ctx.MultiSelect("ms1", items, sel, Width(260), Bg(render.RGB(15, 25, 35)), Radius(8), Bold())
		var tgl bool
		ctx.ToggleOpts("tg1", "Переключатель", &tgl, Bold())
		rad := "a"
		ctx.RadioButton("rb1", "Вариант", &rad, "a", TextColor(render.RGB(200, 100, 50)))
	})
	ctx.endFrame()
}

func TestFocusBorderColorListsTablesAndButton(t *testing.T) {
	ctx := newTestContext(400, 300)
	focusCol := render.RGB(255, 40, 50)

	// 1. Button с FocusBorderColor
	ctx.beginFrame()
	ctx.FocusID = "btn_f"
	ctx.ButtonOpts("btn_f", "Кнопка", Width(100), Height(30), FocusBorderColor(focusCol))
	ctx.endFrame()
	off := (0*ctx.Canvas.Width + 50) * 4
	if ctx.Canvas.Pixels[off+2] != focusCol.R || ctx.Canvas.Pixels[off+1] != focusCol.G || ctx.Canvas.Pixels[off] != focusCol.B {
		t.Fatalf("ожидался цвет рамки Button %v, получено RGB(%d,%d,%d)", focusCol, ctx.Canvas.Pixels[off+2], ctx.Canvas.Pixels[off+1], ctx.Canvas.Pixels[off])
	}

	// 2. Table с FocusBorderColor
	selRow := -1
	cols := []TableColumn{{Title: "Col1", WidthDp: 80}}
	ctx.beginFrame()
	ctx.FocusID = "tbl_f"
	ctx.Table("tbl_f", cols, 2, &selRow, nil, nil, Width(100), Height(60), FocusBorderColor(focusCol))
	ctx.endFrame()
	off = (0*ctx.Canvas.Width + 50) * 4
	if ctx.Canvas.Pixels[off+2] != focusCol.R || ctx.Canvas.Pixels[off+1] != focusCol.G || ctx.Canvas.Pixels[off] != focusCol.B {
		t.Fatalf("ожидался цвет рамки Table %v, получено RGB(%d,%d,%d)", focusCol, ctx.Canvas.Pixels[off+2], ctx.Canvas.Pixels[off+1], ctx.Canvas.Pixels[off])
	}

	// 3. ListView с FocusBorderColor
	selIdx := -1
	ctx.beginFrame()
	ctx.FocusID = "lv_f"
	ctx.ListView("lv_f", 2, 25, &selIdx, nil, Width(100), Height(50), FocusBorderColor(focusCol))
	ctx.endFrame()
	off = (0*ctx.Canvas.Width + 50) * 4
	if ctx.Canvas.Pixels[off+2] != focusCol.R || ctx.Canvas.Pixels[off+1] != focusCol.G || ctx.Canvas.Pixels[off] != focusCol.B {
		t.Fatalf("ожидался цвет рамки ListView %v, получено RGB(%d,%d,%d)", focusCol, ctx.Canvas.Pixels[off+2], ctx.Canvas.Pixels[off+1], ctx.Canvas.Pixels[off])
	}

	// 4. TableEx с FocusBorderColor
	cfg := &TableExConfig{}
	ctx.beginFrame()
	ctx.FocusID = "tblex_f"
	ctx.TableEx("tblex_f", cols, 2, cfg, nil, nil, Width(100), Height(60), FocusBorderColor(focusCol))
	ctx.endFrame()
	off = (0*ctx.Canvas.Width + 50) * 4
	if ctx.Canvas.Pixels[off+2] != focusCol.R || ctx.Canvas.Pixels[off+1] != focusCol.G || ctx.Canvas.Pixels[off] != focusCol.B {
		t.Fatalf("ожидался цвет рамки TableEx %v, получено RGB(%d,%d,%d)", focusCol, ctx.Canvas.Pixels[off+2], ctx.Canvas.Pixels[off+1], ctx.Canvas.Pixels[off])
	}
}
