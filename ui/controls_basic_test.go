package ui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

type mockWindow struct {
	w, h  int
	scale float32
	clip  string
}

func (m *mockWindow) Run(func(platform.Event)) error  { return nil }
func (m *mockWindow) Open(func(platform.Event)) error { return nil }
func (m *mockWindow) Present([]byte, int, int)        {}
func (m *mockWindow) Invalidate()                     {}
func (m *mockWindow) WakeUp()                         {}
func (m *mockWindow) Show()                           {}
func (m *mockWindow) Hide()                           {}
func (m *mockWindow) Close()                          {}
func (m *mockWindow) SetTitle(string)                 {}
func (m *mockWindow) SetCursor(platform.CursorType)   {}
func (m *mockWindow) Size() (int, int)                { return m.w, m.h }
func (m *mockWindow) Scale() float32                  { return m.scale }
func (m *mockWindow) ClipboardText() (string, error)  { return m.clip, nil }
func (m *mockWindow) SetClipboardText(s string) error { m.clip = s; return nil }
func (m *mockWindow) Minimize()                       {}
func (m *mockWindow) Maximize()                       {}
func (m *mockWindow) Restore()                        {}
func (m *mockWindow) IsMaximized() bool               { return false }
func (m *mockWindow) IsMinimized() bool               { return false }
func (m *mockWindow) SetFullscreen(bool)              {}
func (m *mockWindow) IsFullscreen() bool              { return false }
func (m *mockWindow) Position() (int, int)            { return 0, 0 }
func (m *mockWindow) SetPosition(int, int)            {}
func (m *mockWindow) SetSize(int, int)                {}
func (m *mockWindow) SetOnClose(func() bool)          {}
func (m *mockWindow) NativeHandle() uintptr           { return 0 }
func (m *mockWindow) SetCapture()                     {}
func (m *mockWindow) ReleaseCapture()                 {}
func (m *mockWindow) DoubleClickTimeMs() int          { return 500 }

func newTestContext(w, h int) *Context {
	win := &mockWindow{w: w, h: h, scale: 1.0}
	return NewContext(win, nil)
}

func TestComputeVisibleRange(t *testing.T) {
	s, e := ComputeVisibleRange(0, 200, 40, 1000)
	if s != 0 || e != 6 {
		t.Fatalf("ожидался диапазон [0, 6), получено [%d, %d)", s, e)
	}
	s, e = ComputeVisibleRange(400, 200, 40, 1000)
	if s != 10 || e != 16 {
		t.Fatalf("ожидался диапазон [10, 16), получено [%d, %d)", s, e)
	}
	s, e = ComputeVisibleRange(0, 200, 40, 0)
	if s != 0 || e != 0 {
		t.Fatalf("для пустого списка ожидался [0, 0), получено [%d, %d)", s, e)
	}
}

func TestTextSelectionAndFilter(t *testing.T) {
	ctx := newTestContext(400, 200)
	val := "12345"

	// Фокусируем поле и жмём Ctrl+A + Backspace
	ctx.FocusID = "inp"
	ctx.caretPos["inp"] = 5
	ctx.KeyEvents = []KeyPress{
		{Code: KeyA, Mods: platform.Modifiers{Ctrl: true}},
		{Code: KeyBackspace},
	}
	ctx.beginFrame()
	changed := ctx.TextInputOpts("inp", &val, "", Width(200))
	ctx.endFrame()

	if !changed || val != "" {
		t.Fatalf("ожидалось удаление всего текста по Ctrl+A + Backspace, получено %q (changed=%v)", val, changed)
	}

	// Ввод с фильтром (только цифры)
	ctx.FocusID = "inp"
	ctx.TypedRunes = []rune{'a', '7', 'b', '9'}
	ctx.beginFrame()
	onlyDigits := func(r rune) bool { return r >= '0' && r <= '9' }
	changed = ctx.TextInputOpts("inp", &val, "", Width(200), InputFilter(onlyDigits))
	ctx.endFrame()

	if !changed || val != "79" {
		t.Fatalf("ожидалось значение \"79\", получено %q", val)
	}

	// ReadOnly не должен менять значение
	ctx.FocusID = "inp"
	ctx.TypedRunes = []rune{'5'}
	ctx.beginFrame()
	changed = ctx.TextInputOpts("inp", &val, "", Width(200), ReadOnly(true))
	ctx.endFrame()
	if changed || val != "79" {
		t.Fatalf("ReadOnly поле не должно изменяться, получено %q", val)
	}
}

func TestCheckboxAndRadioGroup(t *testing.T) {
	ctx := newTestContext(400, 300)
	checked := false

	// Клик в область чекбокса (0,0)-(18,18)
	ctx.MouseX, ctx.MouseY = 8, 8
	ctx.MousePressed = true
	ctx.beginFrame()
	ctx.Checkbox("chk", "", &checked)
	ctx.endFrame()

	ctx.MouseReleased = true
	ctx.beginFrame()
	changed := ctx.Checkbox("chk", "", &checked)
	ctx.endFrame()

	if !changed || !checked {
		t.Fatalf("ожидалось переключение Checkbox в true (changed=%v, checked=%v)", changed, checked)
	}

	// Disabled чекбокс не должен реагировать
	ctx.MousePressed = true
	ctx.beginFrame()
	ctx.Checkbox("chk", "", &checked, Disabled(true))
	ctx.endFrame()
	ctx.MouseReleased = true
	ctx.beginFrame()
	changed = ctx.Checkbox("chk", "", &checked, Disabled(true))
	ctx.endFrame()
	if changed {
		t.Fatal("Disabled Checkbox не должен меняться")
	}

	// Проверка RadioButton с выбором по строковому ключу
	radioSel := "opt1"
	ctx.MouseX, ctx.MouseY = 8, 8
	ctx.MousePressed = true
	ctx.beginFrame()
	ctx.RadioButton("rb2", "Опция 2", &radioSel, "opt2")
	ctx.endFrame()
	ctx.MouseReleased = true
	ctx.beginFrame()
	changed = ctx.RadioButton("rb2", "Опция 2", &radioSel, "opt2")
	ctx.endFrame()
	if !changed || radioSel != "opt2" {
		t.Fatalf("ожидалось radioSel=opt2, получено %q (changed=%v)", radioSel, changed)
	}

	// Проверка RadioGroup с выбором по строковому ключу
	rgSel := "val1"
	rgItems := []SelectItem{
		{Key: "val1", Label: "Label 1"},
		{Key: "val2", Label: "Label 2"},
	}
	ctx.beginFrame()
	ctx.RadioGroup("rg", rgItems, &rgSel)
	ctx.endFrame()
	if rgSel != "val1" {
		t.Fatalf("ожидалось rgSel=val1, получено %q", rgSel)
	}
}

func TestTabBarEvents(t *testing.T) {
	ctx := newTestContext(600, 200)
	tabs := []TabItem{
		{ID: "t1", Title: "Первая", Closable: true},
		{ID: "t2", Title: "Вторая", Closable: true},
	}
	active := "t1"

	// Клик по второй вкладке (tabW = 160 -> вторая вкладка начинается примерно с x=162)
	ctx.MouseX, ctx.MouseY = 200, 15
	ctx.MousePressed = true
	ctx.beginFrame()
	ev := ctx.TabBar("tb", tabs, &active, Width(600))
	ctx.endFrame()

	if ev.SelectedID != "t2" || active != "t2" {
		t.Fatalf("ожидался выбор вкладки t2, получено SelectedID=%q active=%q", ev.SelectedID, active)
	}

	// Клик по кнопке '+' (правый край: 600 - 16 = 584)
	ctx.MouseX, ctx.MouseY = 584, 15
	ctx.MousePressed = true
	ctx.beginFrame()
	ev = ctx.TabBar("tb", tabs, &active, Width(600))
	ctx.endFrame()

	if !ev.Added {
		t.Fatal("ожидалось событие Added=true при клике на '+'")
	}
}

func TestTabBarHideAddButton(t *testing.T) {
	ctx := newTestContext(600, 200)
	tabs := []TabItem{
		{ID: "t1", Title: "Первая"},
		{ID: "t2", Title: "Вторая"},
	}
	active := "t1"

	// Клик в область правого края, где обычно расположена кнопка '+'
	ctx.MouseX, ctx.MouseY = 584, 15
	ctx.MousePressed = true
	ctx.beginFrame()
	ev := ctx.TabBar("tb_no_add", tabs, &active, Width(600), HideAddButton())
	ctx.endFrame()

	if ev.Added {
		t.Fatal("при HideAddButton() кнопка '+' не должна срабатывать")
	}
}

func TestTabBarVertical(t *testing.T) {
	ctx := newTestContext(400, 300)
	tabs := []TabItem{
		{ID: "general", Title: "Общие", Closable: true},
		{ID: "network", Title: "Сеть", Closable: true},
		{ID: "security", Title: "Безопасность"},
	}
	active := "general"

	// Клик по второй вкладке (tabH = 34, gap = 2 -> вторая вкладка по y в [36..70))
	ctx.MouseX, ctx.MouseY = 60, 50
	ctx.MousePressed = true
	ctx.beginFrame()
	ev := ctx.TabBar("tb_vert", tabs, &active, TabOrientation(Vertical), Width(180), Height(200))
	ctx.endFrame()

	if ev.SelectedID != "network" || active != "network" {
		t.Fatalf("ожидался выбор вкладки network, получено SelectedID=%q active=%q", ev.SelectedID, active)
	}

	// Клик по кнопке '+' внизу вертикальной панели (y в [168..200))
	ctx.MouseX, ctx.MouseY = 90, 184
	ctx.MousePressed = true
	ctx.beginFrame()
	ev = ctx.TabBar("tb_vert", tabs, &active, TabOrientation(Vertical), Width(180), Height(200))
	ctx.endFrame()

	if !ev.Added {
		t.Fatal("ожидалось событие Added=true при клике на '+' внизу вертикального TabBar")
	}

	// С HideAddButton() клик по нижней области не генерирует Added
	ctx.MouseX, ctx.MouseY = 90, 184
	ctx.MousePressed = true
	ctx.beginFrame()
	ev = ctx.TabBar("tb_vert_static", tabs, &active, TabOrientation(Vertical), HideAddButton(), Width(180), Height(200))
	ctx.endFrame()

	if ev.Added {
		t.Fatal("при HideAddButton() в вертикальном TabBar не должно быть события Added")
	}
}

func TestScrollViewWheel(t *testing.T) {
	ctx := newTestContext(400, 300)

	// Первый кадр вычисляет contentH (например, 10 строк по 30px)
	ctx.beginFrame()
	ctx.ScrollView("sv", func() {
		for i := 0; i < 10; i++ {
			ctx.allocRect(100, 30)
		}
	}, Width(300), Height(100))
	ctx.endFrame()

	// Прокрутка колесом вниз (WheelY < 0)
	ctx.MouseX, ctx.MouseY = 50, 50
	ctx.WheelY = -2
	ctx.beginFrame()
	ctx.ScrollView("sv", func() {
		for i := 0; i < 10; i++ {
			ctx.allocRect(100, 30)
		}
	}, Width(300), Height(100))
	ctx.endFrame()

	si := ctx.getScrollInfo("sv:sv")
	if si.y <= 0 {
		t.Fatalf("ожидалось смещение скролла > 0, получено %d (contentH=%d)", si.y, si.contentH)
	}
	_ = render.Rect{}
}

func TestModalAutoHeight(t *testing.T) {
	ctx := newTestContext(800, 600)
	open := true

	var bodyRect render.Rect
	renderModal := func(contentH int, opts ...Option) {
		ctx.beginFrame()
		ctx.Modal("auto_m", &open, "Заголовок", func() {
			bodyRect = ctx.CurrentArea()
			ctx.allocRect(200, contentH)
		}, opts...)
		ctx.endFrame()
	}

	// Первый кадр с AutoHeight(): измеряет контент (120px) -> 120 + header(42) + pad*2(36) = 198px
	renderModal(120, AutoHeight())
	// Второй кадр применяет измеренную высоту
	renderModal(120, AutoHeight())
	if bodyRect.H != 120 {
		t.Fatalf("ожидалась высота тела модалки 120px при AutoHeight(), получено %d", bodyRect.H)
	}

	// Увеличение контента до 220px автоматически расширяет модалку
	renderModal(220, Height(Auto))
	renderModal(220, Height(Auto))
	if bodyRect.H != 220 {
		t.Fatalf("ожидалась высота тела модалки 220px при Height(Auto), получено %d", bodyRect.H)
	}

	// MaxHeight ограничивает итоговую высоту модалки (200px -> тело 200 - 42 - 36 = 122px)
	renderModal(300, AutoHeight(), MaxHeight(200))
	renderModal(300, AutoHeight(), MaxHeight(200))
	if bodyRect.H != 122 {
		t.Fatalf("ожидалась высота тела модалки 122px при MaxHeight(200), получено %d", bodyRect.H)
	}
}
