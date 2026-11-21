package anyui

import (
	"image"
	"testing"
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
		var mousePxX, mousePxY Px

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
		h.InjectMouseMovePx(300, 150)
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

			// 2. ScrollView без явных размеров (Auto по умолчанию) должен занять оставшиеся 500 dp
			ui.ScrollView("sv_auto", func() {
				svArea = ui.CurrentArea()
				ui.Label("Контент внутри ScrollView")
			})
		})
	})
	h.Step()

	if svArea.H < 490 || svArea.H > 500 {
		t.Errorf("ScrollView с Auto должен занять остаток высоты ~500dp, получено %d", svArea.H)
	}

	// Проверяем CanvasView без явных размеров (Auto по умолчанию)
	h.SetDraw(func(ui *UI) {
		ui.Column(0, func() {
			ui.Card(func() {
				ui.Label("Шапка")
			}, Height(50), FillWidth())

			ui.CanvasView(func(canvas *Canvas, area Rect) {
				cvArea = area
			})
		})
	})
	h.Step()

	if cvArea.H < 545 || cvArea.H > 550 {
		t.Errorf("CanvasView с Auto должен занять остаток высоты ~550px, получено %d", cvArea.H)
	}

	// Проверяем ListView с Auto
	selected := 0
	h.SetDraw(func(ui *UI) {
		ui.Column(0, func() {
			ui.Card(func() {
				ui.Label("Шапка")
			}, Height(120), FillWidth())

			ui.ListView("lv_auto", 50, 30, &selected, func(index int, area Rect, isSelected bool) {
				lvCountDrawn++
			}, Height(Auto))
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
				ui.SplitPane("test_sp", Horizontal, &ratio, func() {
					// Первая панель: ScrollView с Auto по умолчанию
					ui.ScrollView("sv_panel1", func() {
						sv1Area = ui.CurrentArea()
						for i := 0; i < 20; i++ {
							ui.Label("Панель 1 Строка")
						}
					})
				}, func() {
					// Вторая панель: ScrollView с Auto по умолчанию
					ui.ScrollView("sv_panel2", func() {
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

// TestOverlaysAndLayoutRectDPI проверяет контракт передачи Dp в LayoutRect, PushOverlay, Tooltip, Popover и RegisterHoverArea
// при масштабе 150% DPI.
func TestOverlaysAndLayoutRectDPI(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       900,
		Height:      600,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("NewHeadless error: %v", err)
	}

	const sc = float32(1.5)
	h.SetScale(sc)

	var layoutInnerRect Rect
	popOpen := true

	h.SetDraw(func(ui *UI) {
		dpArea := ui.CurrentArea()
		if dpArea.W != 600 || dpArea.H != 400 {
			t.Fatalf("expected CurrentArea 600x400 at 1.5x DPI, got %+v", dpArea)
		}

		// 1. LayoutRect с передачей Dp (200x100 Dp должно дать 300x150 Px в контексте)
		customBoxDp := Rect{X: 50, Y: 40, W: 200, H: 100}
		ui.LayoutRect(customBoxDp, func() {
			layoutInnerRect = ui.CurrentArea()
		})

		// 2. PushOverlay с Dp
		ui.PushOverlay(Rect{X: 100, Y: 100, W: 150, H: 80}, func() {
			ui.Label("Overlay text")
		})

		// 3. Tooltip и Popover с Dp
		ui.Tooltip("Подсказка", Rect{X: 10, Y: 10, W: 100, H: 30})
		ui.Popover("test_pop", &popOpen, Rect{X: 20, Y: 50, W: 120, H: 40}, func() {
			ui.Label("Popover content")
		})

		// 4. RegisterHoverArea с Dp
		ui.RegisterHoverArea("test_hover", Rect{X: 200, Y: 200, W: 80, H: 40}, CursorHand, false)
	})

	h.Step()

	// Внутри LayoutRect CurrentArea() в Dp должна отражать переданные 200x100
	if layoutInnerRect.W != 200 || layoutInnerRect.H != 100 {
		t.Errorf("LayoutRect inner area Dp mismatch: got (%d, %d), want (200, 100)", layoutInnerRect.W, layoutInnerRect.H)
	}
}

// TestPushOverlayDPI2xCoveringArea проверяет, что PushOverlay(ui.CurrentArea(), ...) при 2.0x DPI
// масштабирует логическую область 400x300 Dp на полные 800x600 Px и блокирует ввод по всему экрану,
// а InjectMouseClick(xDp, yDp) автоматически масштабируется из Dp в Px.
func TestPushOverlayDPI2xCoveringArea(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       800,
		Height:      600,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("NewHeadless error: %v", err)
	}

	h.SetScale(2.0)

	btnClicked := false
	overlayDrawn := false

	h.SetDraw(func(ui *UI) {
		// Кнопка в правом нижнем углу: в Dp она лежит в интервале X=[250, 350], Y=[200, 250]
		// В физических пикселях (scale=2.0) это X=[500, 700], Y=[400, 500]
		ui.LayoutRect(Rect{X: 250, Y: 200, W: 100, H: 50}, func() {
			if ui.Button("btn_test", "Click", Primary(), Width(100)) {
				btnClicked = true
			}
		})

		// Оверлей накрывает всю область окна в Dp (CurrentArea: 400x300)
		fullDp := ui.CurrentArea()
		if fullDp.W != 400 || fullDp.H != 300 {
			t.Fatalf("expected CurrentArea 400x300 at 2.0x, got %+v", fullDp)
		}
		ui.PushOverlay(fullDp, func() {
			overlayDrawn = true
		})
	})

	// Кадр 1: регистрируется оверлей
	h.Step()
	if !overlayDrawn {
		t.Fatal("оверлей не был отрисован")
	}

	// Кадр 2: кликаем в кнопку через InjectMouseClick в координатах Dp (300, 225).
	// Внутри оверлей должен заблокировать этот клик, так как накрывает 400x300 Dp = 800x600 Px.
	h.InjectMouseClick(300, 225, MouseLeft)
	h.Step()

	if btnClicked {
		t.Fatal("кнопка под оверлеем сработала! Оверлей при 2.0x DPI не накрыл область в правом нижнем углу")
	}
}

// TestUnifiedOptionsAndSizing проверяет работу opts ...Option и задание размеров через опции
// для StatusDot, Image, CanvasView, InteractiveCanvas, ScrollView, ContextMenu и Tooltip.
func TestUnifiedOptionsAndSizing(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       600,
		Height:      500,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("NewHeadless error: %v", err)
	}
	defer h.Close()

	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	var cvArea, icArea, svArea Rect
	menuClicked := false

	h.SetDraw(func(u *UI) {
		u.Column(4, func() {
			u.StatusDot(RGB(0, 200, 0), "Активен", FontSize(14), Bold(), TextColor(RGB(255, 255, 255)))
			u.Image(img, ScaleFit, Width(64), Height(48))
			u.CanvasView(func(c *Canvas, areaPx Rect) {
				cvArea = areaPx
			}, Width(180), Height(60))
			u.InteractiveCanvas("ic_opt", CanvasCallbacks{
				OnDraw: func(c *Canvas, areaPx Rect, focused bool) {
					icArea = areaPx
				},
			}, Width(150), Height(70))
			u.ScrollView("sv_opt", func() {
				svArea = u.CurrentArea()
				u.Label("Содержимое")
			}, Width(220), Height(90))
		})

		anchor := Rect{X: 10, Y: 10, W: 100, H: 40}
		u.Tooltip("Подсказка с опциями", anchor, Bg(RGB(20, 20, 30)), TextColor(RGB(240, 240, 240)), Padding(8), FontSize(13))
		u.ContextMenu("cm_opt", anchor, []ContextMenuItem{
			{Label: "Действие", OnClick: func() { menuClicked = true }},
		}, Width(180), Bg(RGB(25, 25, 35)), BorderColor(RGB(80, 80, 100)))
	})

	h.Step()

	if cvArea.W != 180 || cvArea.H != 60 {
		t.Errorf("CanvasView: ожидалось 180x60, получено %dx%d", cvArea.W, cvArea.H)
	}
	if icArea.W != 150 || icArea.H != 70 {
		t.Errorf("InteractiveCanvas: ожидалось 150x70, получено %dx%d", icArea.W, icArea.H)
	}
	if svArea.W < 200 || svArea.W > 220 || svArea.H < 75 || svArea.H > 90 {
		t.Errorf("ScrollView: ожидалась внутренняя область около 220x90 (с учётом отступов скроллбара), получено %dx%d", svArea.W, svArea.H)
	}

	// Наведение на anchor активирует Tooltip, правый клик открывает ContextMenu
	h.InjectMouseMove(30, 20)
	h.Step()
	h.InjectMouseClick(30, 20, MouseRight)
	h.Step()
	// Клик по первому пункту открытого ContextMenu (открывается в точке (30, 20), первый пункт y ~ 24..54)
	h.InjectMouseClick(50, 36, MouseLeft)
	h.Step()
	if !menuClicked {
		t.Error("ContextMenu с опциями не обработал клик по пункту")
	}
}

// TestUIClipboardImage проверяет методы UI.SetClipboardImage и UI.ClipboardImage в Headless-режиме.
func TestUIClipboardImage(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       200,
		Height:      200,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("NewHeadless error: %v", err)
	}
	defer h.Close()

	src := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var got image.Image
	var setErr, getErr error

	h.SetDraw(func(u *UI) {
		setErr = u.SetClipboardImage(src)
		got, getErr = u.ClipboardImage()
	})
	h.Step()

	if setErr != nil || getErr != nil {
		t.Fatalf("ошибки буфера обмена изображений: setErr=%v getErr=%v", setErr, getErr)
	}
	if got == nil || got.Bounds() != src.Bounds() {
		t.Fatalf("ожидалось изображение %v, получено %v", src.Bounds(), got)
	}
}
