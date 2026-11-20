package anyui

import (
	"testing"

	"github.com/kirpicheff/anyui/render"
)

// TestCurrentAreaAndMousePosDPI проверяет контракт систем координат (Dp vs Px)
// для CurrentArea(), CurrentAreaPx() и координат мыши при масштабах 100%, 150% и 200% DPI.
func TestCurrentAreaAndMousePosDPI(t *testing.T) {
	scales := []float32{1.0, 1.5, 2.0}

	for _, sc := range scales {
		reg, bold, mono := DefaultTestFonts()
		// Физический размер окна 900x600 пикселей
		h, err := NewHeadless(Config{
			Width:       900,
			Height:      600,
			FontRegular: reg,
			FontBold:    bold,
			FontMono:    mono,
		})
		if err != nil {
			t.Fatalf("scale %.1f: ошибка NewHeadless: %v", sc, err)
		}

		h.SetScale(sc)

		var capturedAreaDp Rect
		var capturedAreaPx Rect
		var mouseDpX, mouseDpY Dp
		var mousePxX, mousePxY int

		h.SetDraw(func(ui *UI) {
			capturedAreaDp = ui.CurrentArea()
			capturedAreaPx = ui.CurrentAreaPx()

			ui.Padding(20, func() {
				// Вложенный контейнер
				innerDp := ui.CurrentArea()
				expectedInnerW := int(float32(900)/sc+0.5) - 40
				expectedInnerH := int(float32(600)/sc+0.5) - 40
				if innerDp.W != expectedInnerW || innerDp.H != expectedInnerH {
					t.Errorf("scale %.1f: неверный размер вложенного CurrentArea(): got (%d, %d), want (%d, %d)",
						sc, innerDp.W, innerDp.H, expectedInnerW, expectedInnerH)
				}
			})

			mouseDpX, mouseDpY = ui.MousePosDp()
			mousePxX, mousePxY = ui.MousePosPx()
		})

		// Имитируем физические координаты мыши (300, 150)
		h.InjectMouseMove(300, 150)
		h.Step()

		// Проверяем CurrentAreaPx: строго физические пиксели окна
		if capturedAreaPx.W != 900 || capturedAreaPx.H != 600 {
			t.Errorf("scale %.1f: CurrentAreaPx не соответствует размеру окна (900, 600): got (%d, %d)",
				sc, capturedAreaPx.W, capturedAreaPx.H)
		}

		// Проверяем CurrentArea: строго логические пиксели Dp
		expectedDpW := int(float32(900)/sc + 0.5)
		expectedDpH := int(float32(600)/sc + 0.5)
		if capturedAreaDp.W != expectedDpW || capturedAreaDp.H != expectedDpH {
			t.Errorf("scale %.1f: CurrentArea() не в Dp: got (%d, %d), want (%d, %d)",
				sc, capturedAreaDp.W, capturedAreaDp.H, expectedDpW, expectedDpH)
		}

		// Проверяем координаты мыши
		if mousePxX != 300 || mousePxY != 150 {
			t.Errorf("scale %.1f: MousePosPx() неверен: got (%d, %d), want (300, 150)", sc, mousePxX, mousePxY)
		}
		expectedMouseDpX := Dp(float32(300)/sc + 0.5)
		expectedMouseDpY := Dp(float32(150)/sc + 0.5)
		if mouseDpX != expectedMouseDpX || mouseDpY != expectedMouseDpY {
			t.Errorf("scale %.1f: MousePosDp() неверен: got (%d, %d), want (%d, %d)",
				sc, mouseDpX, mouseDpY, expectedMouseDpX, expectedMouseDpY)
		}
	}
}

// TestAdaptiveContainersAutoFill проверяет автозаполнение ScrollView, ListView и CanvasView
// с использованием sentinel-значений Auto() и -1 / <= 0.
func TestAdaptiveContainersAutoFill(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       800,
		Height:      600,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("ошибка NewHeadless: %v", err)
	}

	var svArea Rect
	var cvArea Rect
	var lvCountDrawn int

	h.SetDraw(func(ui *UI) {
		ui.Column(0, func() {
			// 1. Верхняя фиксированная панель 100 dp
			ui.Card(func() {
				ui.Label("Верхняя панель")
			}, Height(100), FillWidth())

			// 2. ScrollView с Auto(), должен занять оставшиеся 500 dp
			ui.ScrollView("sv_auto", Auto(), Auto(), func() {
				svArea = ui.CurrentArea()
				ui.Label("Контент внутри ScrollView")
			})
		})
	})
	h.Step()

	if svArea.H < 490 || svArea.H > 500 {
		t.Errorf("ScrollView с Auto() должен занять остаток высоты ~500dp, получено %d", svArea.H)
	}

	// Проверяем CanvasView с sentinel-значением Auto()
	h.SetDraw(func(ui *UI) {
		ui.Column(0, func() {
			ui.Card(func() {
				ui.Label("Шапка")
			}, Height(50), FillWidth())

			ui.CanvasView(Auto(), Auto(), func(canvas *render.Canvas, area Rect) {
				cvArea = area
			})
		})
	})
	h.Step()

	if cvArea.H < 545 || cvArea.H > 550 {
		t.Errorf("CanvasView с Auto() должен занять остаток высоты ~550px, получено %d", cvArea.H)
	}

	// Проверяем ListView с Auto()
	selected := 0
	h.SetDraw(func(ui *UI) {
		ui.Column(0, func() {
			ui.Card(func() {
				ui.Label("Шапка")
			}, Height(120), FillWidth())

			ui.ListView("lv_auto", 50, 30, &selected, func(index int, area Rect, isSelected bool) {
				lvCountDrawn++
			}, Height(Auto()))
		})
	})
	h.Step()

	if lvCountDrawn == 0 {
		t.Error("ListView с Height(Auto()) не отрисовал ни одного элемента")
	}
}

// TestNestedContainersScrollViewInsideSplitPaneInsidePage проверяет сложную вложенность:
// ScrollView внутри SplitPane внутри Page при различных масштабах DPI (100%, 150%, 200%).
func TestNestedContainersScrollViewInsideSplitPaneInsidePage(t *testing.T) {
	scales := []float32{1.0, 1.5, 2.0}

	for _, sc := range scales {
		reg, bold, mono := DefaultTestFonts()
		h, err := NewHeadless(Config{
			Width:       1200,
			Height:      800,
			FontRegular: reg,
			FontBold:    bold,
			FontMono:    mono,
		})
		if err != nil {
			t.Fatalf("scale %.1f: ошибка NewHeadless: %v", sc, err)
		}
		h.SetScale(sc)

		var pageBodyH int
		var sv1Area Rect
		var sv2Area Rect
		ratio := float32(0.4)

		h.SetDraw(func(ui *UI) {
			ui.Page(PageConfig{
				HeaderDp: 60,
				FooterDp: 40,
				GapDp:    10,
				NoScroll: true,
			}, func() {
				ui.Label("Шапка Page")
			}, func() {
				bodyArea := ui.CurrentArea()
				pageBodyH = bodyArea.H

				// SplitPane внутри тела Page с заполнением доступной высоты
				ui.SplitPane("test_sp", SplitHorizontal, &ratio, func() {
					// Первая панель: ScrollView с Auto()
					ui.ScrollView("sv_panel1", Auto(), Auto(), func() {
						sv1Area = ui.CurrentArea()
						for i := 0; i < 20; i++ {
							ui.Label("Панель 1 Строка")
						}
					})
				}, func() {
					// Вторая панель: ScrollView с Auto()
					ui.ScrollView("sv_panel2", Auto(), Auto(), func() {
						sv2Area = ui.CurrentArea()
						for i := 0; i < 20; i++ {
							ui.Label("Панель 2 Строка")
						}
					})
				}, FillHeight())
			}, func() {
				ui.Label("Футер Page")
			})
		})

		h.Step()

		// Окно 800px / sc в Dp
		windowDpH := int(float32(800)/sc + 0.5)
		// Ожидаемая высота тела Page: windowDpH - Header(60) - Footer(40) - 2*Gap(10) = windowDpH - 120
		expectedBodyH := windowDpH - 120
		if pageBodyH < expectedBodyH-2 || pageBodyH > expectedBodyH+2 {
			t.Errorf("scale %.1f: неверная высота тела Page: got %d, want ~%d", sc, pageBodyH, expectedBodyH)
		}

		// Проверяем, что оба ScrollView получили корректную ненулевую высоту без схлопывания и без убегания в бесконечность
		if sv1Area.H <= 0 || sv1Area.W <= 0 {
			t.Errorf("scale %.1f: sv1Area схлопнулся: %+v", sc, sv1Area)
		}
		if sv2Area.H <= 0 || sv2Area.W <= 0 {
			t.Errorf("scale %.1f: sv2Area схлопнулся: %+v", sc, sv2Area)
		}

		if sv1Area.H > pageBodyH || sv2Area.H > pageBodyH {
			t.Errorf("scale %.1f: ScrollView вышел за границы родителя: sv1.H=%d, sv2.H=%d, bodyH=%d",
				sc, sv1Area.H, sv2Area.H, pageBodyH)
		}
	}
}

