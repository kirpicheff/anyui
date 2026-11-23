package anyui

import (
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kirpicheff/anyui/render"
)

var _ = flag.Bool("update", false, "обновить эталонные golden-файлы")

func TestHeadlessDeterministicFontRequirement(t *testing.T) {
	// Без шрифтов NewHeadless обязан возвращать ошибку для исключения расхождений в CI
	_, err := NewHeadless(Config{
		Width:  400,
		Height: 300,
	})
	if err == nil {
		t.Fatal("ожидалась ошибка при создании Headless без Config.FontRegular")
	}

	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       400,
		Height:      300,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("ошибка создания Headless с детерминированными шрифтами: %v", err)
	}
	defer h.Close()
}

func TestHeadlessStepAndDispatch(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       400,
		Height:      300,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	renderedFrames := 0
	h.SetDraw(func(ui *UI) {
		renderedFrames++
		ui.Label("Hello Headless")
	})

	h.Step()
	if renderedFrames != 1 {
		t.Fatalf("ожидался ровно 1 кадр, получено: %d", renderedFrames)
	}

	dispatched := false
	h.App().Dispatch(func() {
		dispatched = true
	})

	// Dispatch выполняется синхронно во время h.Step()
	h.Step()
	if !dispatched {
		t.Fatal("задача из App.Dispatch не выполнилась во время h.Step()")
	}
	if renderedFrames != 2 {
		t.Fatalf("ожидалось 2 кадра, получено: %d", renderedFrames)
	}
}

func TestHeadlessVirtualClockAndToasts(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       600,
		Height:      400,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	toastTriggered := false
	h.SetDraw(func(ui *UI) {
		if !toastTriggered {
			ui.ShowToast("Уведомление", "Тестовое сообщение", ToastSuccess, 2*time.Second)
			toastTriggered = true
		}
	})

	h.Step()
	if h.win.ctx.ToastsCount() != 1 {
		t.Fatalf("ожидался 1 активный тост, получено: %d", h.win.ctx.ToastsCount())
	}

	// Перематываем время на 1 секунду вперед (тост еще жив при длительности 2 секунды)
	h.AdvanceTime(1 * time.Second)
	h.Step()
	if h.win.ctx.ToastsCount() != 1 {
		t.Fatalf("тост не должен был исчезнуть через 1 секунду, осталось: %d", h.win.ctx.ToastsCount())
	}

	// Перематываем еще на 1.5 секунды (суммарно 2.5 сек) — тост должен истечь и удалиться
	h.AdvanceTime(1500 * time.Millisecond)
	h.Step()
	if h.win.ctx.ToastsCount() != 0 {
		t.Fatalf("просроченный тост должен быть удален, осталось: %d", h.win.ctx.ToastsCount())
	}
}

func TestHeadlessSyntheticInputInjection(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       400,
		Height:      300,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	clicks := 0
	text := ""
	lastWheel := float32(0)
	shortcutFired := false

	h.SetDraw(func(ui *UI) {
		ui.Shortcut(KeyA, Modifiers{Ctrl: true}, func() {
			shortcutFired = true
		})

		ui.Padding(10, func() {
			if ui.Button("btn", "Нажми меня", Width(120)) {
				clicks++
			}
			ui.TextInput("inp", &text, "Подсказка", Width(200))
			lastWheel = ui.ctx.WheelY
		})
	})

	// 1. Первый кадр для построения геометрии виджетов
	h.Step()

	// 2. Клик мышью по кнопке (кнопка в пределах X: 10..130, Y: 10..46)
	h.InjectMouseClick(50, 25, MouseLeft)
	h.Step()
	if clicks != 1 {
		t.Fatalf("ожидался 1 клик, получено: %d", clicks)
	}

	// 3. Ввод текста в поле
	h.InjectFocus("inp")
	h.InjectText("Привет AnyUI")
	h.Step()
	if text != "Привет AnyUI" {
		t.Fatalf("ожидался введенный текст 'Привет AnyUI', получено: %q", text)
	}

	// 4. Прокрутка колеса мыши
	h.InjectMouseWheel(0, -120)
	h.Step()
	if lastWheel != -120 {
		t.Fatalf("ожидалось значение колеса -120, получено: %v", lastWheel)
	}

	// 5. Горячая клавиша Ctrl+A
	h.InjectKey(KeyA, Modifiers{Ctrl: true})
	h.Step()
	if !shortcutFired {
		t.Fatal("горячая клавиша Ctrl+A не сработала")
	}
}

func TestHeadlessFocusAndTabOrder(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       400,
		Height:      300,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	txt1 := "A"
	txt2 := "B"
	h.SetDraw(func(ui *UI) {
		ui.Column(10, func() {
			ui.TextInput("field1", &txt1, "")
			ui.TextInput("field2", &txt2, "")
			ui.Button("btn1", "Кнопка 1")
		})
	})

	h.Step()

	order := h.FocusOrder()
	expectedOrder := []string{"field1", "field2", "btn1"}
	if len(order) != len(expectedOrder) {
		t.Fatalf("неожиданный порядок фокуса: ожидалось %v, получено %v", expectedOrder, order)
	}
	for i := range order {
		if order[i] != expectedOrder[i] {
			t.Fatalf("несовпадение порядка фокуса на позиции %d: ожидалось %s, получено %s", i, expectedOrder[i], order[i])
		}
	}

	// Программная установка фокуса
	h.InjectFocus("field1")
	h.Step()
	if h.FocusID() != "field1" {
		t.Fatalf("ожидался фокус на field1, текущий: %s", h.FocusID())
	}

	// Переход по Tab вперед на field2
	h.InjectKey(KeyTab, Modifiers{})
	h.Step()
	if h.FocusID() != "field2" {
		t.Fatalf("после Tab ожидался фокус на field2, текущий: %s", h.FocusID())
	}

	// Переход по Tab вперед на btn1
	h.InjectKey(KeyTab, Modifiers{})
	h.Step()
	if h.FocusID() != "btn1" {
		t.Fatalf("после Tab ожидался фокус на btn1, текущий: %s", h.FocusID())
	}

	// Переход по Shift+Tab назад на field2
	h.InjectKey(KeyTab, Modifiers{Shift: true})
	h.Step()
	if h.FocusID() != "field2" {
		t.Fatalf("после Shift+Tab ожидался фокус на field2, текущий: %s", h.FocusID())
	}
}

func TestHeadlessGoldenRendering(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       480,
		Height:      320,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
		Theme:       NavyTheme(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	text := "AnyUI Golden Test"
	checked := true
	h.SetDraw(func(ui *UI) {
		ui.Padding(24, func() {
			ui.Column(16, func() {
				ui.Label("Тест детерминированного рендеринга", FontSize(18), Bold())
				ui.Row(12, 36, func() {
					ui.Button("btn_primary", "Сохранить", Primary())
					ui.Button("btn_ghost", "Отмена", Ghost())
				})
				ui.TextInput("input_name", &text, "Имя")
				ui.Checkbox("chk_active", "Включить модуль", &checked)
				ui.StatusDot(render.RGB(34, 197, 94), "Готов к работе")
			})
		})
	})

	// Отрисовываем кадр
	h.Step()

	// Создаём директорию testdata, если её нет
	_ = os.MkdirAll("testdata", 0755)

	// Если файл ещё не существует, сохраняем эталон
	goldenFile := "testdata/golden_basic_controls.png"
	if _, err := os.Stat(goldenFile); os.IsNotExist(err) {
		if err := h.CapturePNG(goldenFile); err != nil {
			t.Fatalf("не удалось записать первоначальный эталон: %v", err)
		}
		t.Logf("создан первоначальный эталон: %s", goldenFile)
	}

	// Проверяем пиксель-в-пиксель
	AssertGolden(t, h, "golden_basic_controls")
}

func BenchmarkFrameBudget1000Widgets(b *testing.B) {
	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       1280,
		Height:      960,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		b.Fatal(err)
	}
	defer h.Close()

	// 200 строк по 5 виджетов = 1000 виджетов
	h.SetDraw(func(ui *UI) {
		ui.ScrollView("bench_scroll", func() {
			for i := 0; i < 200; i++ {
				ui.Row(8, 30, func() {
					ui.Label(fmt.Sprintf("Item %d", i))
					ui.Button(fmt.Sprintf("btn_%d", i), "Click")
					val := i%2 == 0
					ui.Checkbox(fmt.Sprintf("chk_%d", i), "Check", &val)
					str := "Test"
					ui.TextInput(fmt.Sprintf("txt_%d", i), &str, "Placeholder")
					ui.StatusDot(render.RGB(0, 200, 0), "OK")
				})
			}
		}, Width(1280), Height(960))
	})

	// Прогревочный шаг
	h.Step()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Step()
	}
}

type mockHandleWindow struct {
	headlessWindow
	handle uintptr
}

func (m *mockHandleWindow) NativeHandle() uintptr { return m.handle }

func TestWindowNativeHandle(t *testing.T) {
	var nilWin *Window
	if got := nilWin.NativeHandle(); got != 0 {
		t.Fatalf("ожидалось NativeHandle() == 0 для nil-окна, получено %d", got)
	}

	emptyWin := &Window{}
	if got := emptyWin.NativeHandle(); got != 0 {
		t.Fatalf("ожидалось NativeHandle() == 0 для неинициализированного окна, получено %d", got)
	}

	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       400,
		Height:      300,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("ошибка создания Headless: %v", err)
	}
	defer h.Close()

	if got := h.Window().NativeHandle(); got != 0 {
		t.Fatalf("в Headless ожидалось NativeHandle() == 0, получено %d", got)
	}

	const wantHandle uintptr = 0x1234ABCD
	winWithHandle := &Window{platWin: &mockHandleWindow{handle: wantHandle}}
	if got := winWithHandle.NativeHandle(); got != wantHandle {
		t.Fatalf("ожидалось NativeHandle() == 0x%X, получено 0x%X", wantHandle, got)
	}
}

func TestHeadlessHasActiveOverlays(t *testing.T) {
	var nilUI *UI
	if nilUI.HasActiveOverlays() {
		t.Fatal("ожидалось HasActiveOverlays() == false для nil *UI")
	}

	reg, bold, mono := DefaultTestFonts()
	h, err := NewHeadless(Config{
		Width:       640,
		Height:      480,
		FontRegular: reg,
		FontBold:    bold,
		FontMono:    mono,
	})
	if err != nil {
		t.Fatalf("ошибка создания Headless: %v", err)
	}
	defer h.Close()

	modalOpen := false
	pushOverlay := false
	seenBeforeOverlay := false
	seenAfterOverlay := false

	h.SetDraw(func(ui *UI) {
		seenBeforeOverlay = ui.HasActiveOverlays()
		ui.Label("Базовый экран")
		if modalOpen {
			ui.Modal("dlg_test", &modalOpen, "Подтверждение", func() {
				ui.Label("Содержимое модалки")
			})
		}
		if pushOverlay {
			ui.PushOverlay(Rect{X: 20, Y: 20, W: 160, H: 80}, func() {
				ui.Label("Оверлей")
			})
		}
		seenAfterOverlay = ui.HasActiveOverlays()
	})

	// 1. Без оверлеев — false
	h.Step()
	if seenBeforeOverlay || seenAfterOverlay || h.UI().HasActiveOverlays() {
		t.Fatalf("без оверлеев ожидалось false, получено before=%v after=%v post=%v",
			seenBeforeOverlay, seenAfterOverlay, h.UI().HasActiveOverlays())
	}

	// 2. Только Toast — не является блокирующим оверлеем
	h.UI().ShowToast("Уведомление", "Проверка", ToastInfo, 3*time.Second)
	h.Step()
	if seenBeforeOverlay || seenAfterOverlay || h.UI().HasActiveOverlays() {
		t.Fatalf("для Toast ожидалось HasActiveOverlays() == false, получено before=%v after=%v post=%v",
			seenBeforeOverlay, seenAfterOverlay, h.UI().HasActiveOverlays())
	}

	// 3. Активный Modal — true сразу после вызова в кадре, а на следующем кадре и до вызова (prevBlocks)
	modalOpen = true
	h.Step()
	if !seenAfterOverlay || !h.UI().HasActiveOverlays() {
		t.Fatalf("при открытом Modal ожидалось HasActiveOverlays() == true, получено after=%v post=%v",
			seenAfterOverlay, h.UI().HasActiveOverlays())
	}
	h.Step()
	if !seenBeforeOverlay || !seenAfterOverlay {
		t.Fatalf("на втором кадре с Modal ожидалось true до и после вызова, получено before=%v after=%v",
			seenBeforeOverlay, seenAfterOverlay)
	}

	// 4. После закрытия Modal — на кадре закрытия ещё сохраняется prevBlocks, на следующем кадре false
	modalOpen = false
	h.Step()
	h.Step()
	if seenBeforeOverlay || seenAfterOverlay || h.UI().HasActiveOverlays() {
		t.Fatalf("после закрытия Modal ожидалось false, получено before=%v after=%v post=%v",
			seenBeforeOverlay, seenAfterOverlay, h.UI().HasActiveOverlays())
	}

	// 5. PushOverlay также переводит HasActiveOverlays в true и затем сбрасывается
	pushOverlay = true
	h.Step()
	if !seenAfterOverlay || !h.UI().HasActiveOverlays() {
		t.Fatalf("при активном PushOverlay ожидалось true, получено after=%v post=%v",
			seenAfterOverlay, h.UI().HasActiveOverlays())
	}
	pushOverlay = false
	h.Step()
	h.Step()
	if h.UI().HasActiveOverlays() {
		t.Fatal("после снятия PushOverlay ожидалось HasActiveOverlays() == false")
	}
}
