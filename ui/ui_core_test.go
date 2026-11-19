package ui

import (
	"testing"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

func TestAnimatorConvergence(t *testing.T) {
	anim := NewAnimator()
	now := time.Unix(1000, 0)
	anim.BeginFrame(now)

	// Первое значение сразу равно target
	v0 := anim.Float("btn", 0.0, 0.15)
	if v0 != 0.0 {
		t.Fatalf("ожидалось начальное значение 0, получено %v", v0)
	}
	if anim.HasActive() {
		t.Fatal("не должно быть активной анимации в начальном кадре")
	}

	// Запускаем переход к 1.0
	for i := 1; i <= 60; i++ {
		anim.BeginFrame(now.Add(time.Duration(i*16) * time.Millisecond))
		v := anim.Float("btn", 1.0, 0.15)
		if i == 1 && (v <= 0 || v >= 1) {
			t.Fatalf("ожидалась промежуточная интерполяция, получено %v", v)
		}
	}

	// Через 1 секунду анимация должна сойтись к 1.0 и завершиться
	anim.BeginFrame(now.Add(2 * time.Second))
	vEnd := anim.Float("btn", 1.0, 0.15)
	if vEnd != 1.0 {
		t.Fatalf("ожидалось схождение к 1.0, получено %v", vEnd)
	}
	if anim.HasActive() {
		t.Fatal("анимация должна была завершиться")
	}
}

func TestOverlayLayerBlocking(t *testing.T) {
	var ol OverlayLayer
	ol.BeginFrame()

	drawn := false
	bounds := render.Rect{X: 10, Y: 10, W: 100, H: 50}
	ol.Push(bounds, func() {
		drawn = true
	})

	// На этом же кадре отрисовка откладывается до Flush
	if drawn {
		t.Fatal("оверлей не должен рисоваться до Flush")
	}
	ol.Flush()
	if !drawn {
		t.Fatal("оверлей должен быть отрисован в Flush")
	}

	// На следующем кадре область bounds блокирует ввод основного слоя
	ol.BeginFrame()
	if !ol.IsBlocked(50, 30) {
		t.Fatal("точка внутри оверлея должна блокироваться")
	}
	if ol.IsBlocked(5, 5) {
		t.Fatal("точка вне оверлея не должна блокироваться")
	}
}

func TestFocusTabAndShortcuts(t *testing.T) {
	ctx := &Context{
		focusOrder: []string{"a", "b", "c"},
		FocusID:    "a",
	}

	triggered := false
	ctx.RegisterShortcut(KeyA, platform.Modifiers{Ctrl: true}, func() {
		triggered = true
	})

	// Tab вперёд: a -> b
	ctx.KeyEvents = []KeyPress{{Code: KeyTab}}
	ctx.processFocusAndShortcuts()
	if ctx.FocusID != "b" {
		t.Fatalf("ожидался фокус 'b', получено %q", ctx.FocusID)
	}

	// Shift+Tab назад: b -> a
	ctx.KeyEvents = []KeyPress{{Code: KeyTab, Mods: platform.Modifiers{Shift: true}}}
	ctx.processFocusAndShortcuts()
	if ctx.FocusID != "a" {
		t.Fatalf("ожидался фокус 'a', получено %q", ctx.FocusID)
	}

	// Ctrl+A триггерит горячую клавишу
	ctx.KeyEvents = []KeyPress{{Code: KeyA, Mods: platform.Modifiers{Ctrl: true}}}
	ctx.processFocusAndShortcuts()
	if !triggered {
		t.Fatal("горячая клавиша Ctrl+A не сработала")
	}
}

func TestApplyOptions(t *testing.T) {
	c := render.Hex(0x112233)
	st := ApplyOptions([]Option{
		Primary(),
		Bg(c),
		Width(240),
		FontSize(16),
		Disabled(true),
	})

	if st.Variant != VariantPrimary {
		t.Fatalf("ожидался VariantPrimary, получено %v", st.Variant)
	}
	if st.Bg == nil || *st.Bg != c {
		t.Fatal("цвет Bg не применён")
	}
	if st.WidthDp == nil || *st.WidthDp != 240 {
		t.Fatal("ширина Width не применена")
	}
	if st.FontSizeDp == nil || *st.FontSizeDp != 16 {
		t.Fatal("размер шрифта FontSize не применён")
	}
	if !st.Disabled {
		t.Fatal("флаг Disabled не применён")
	}
}

func TestNewStylingOptionsAndButtonTransparentBg(t *testing.T) {
	hoverCol := render.Hex(0xAABBCC)
	focusCol := render.RGBA(0, 0, 0, 0)
	shadowCol := render.RGBA(10, 20, 30, 128)

	st := ApplyOptions([]Option{
		HoverTextColor(hoverCol),
		FocusBorderColor(focusCol),
		TextAlignCenter(),
		LetterSpacing(1.5),
		TextShadow(1, 2, 4, shadowCol),
		Bg(render.RGBA(0, 0, 0, 0)),
	})

	if st.HoverTextColor == nil || *st.HoverTextColor != hoverCol {
		t.Fatal("HoverTextColor не применён")
	}
	if st.FocusBorderColor == nil || *st.FocusBorderColor != focusCol {
		t.Fatal("FocusBorderColor не применён")
	}
	if st.TextAlign != AlignTextCenter {
		t.Fatalf("ожидался AlignTextCenter, получено %v", st.TextAlign)
	}
	if st.LetterSpacing != 1.5 {
		t.Fatalf("ожидался LetterSpacing 1.5, получено %v", st.LetterSpacing)
	}
	if st.TextShadow == nil || st.TextShadow.Blur != 4 || st.TextShadow.Color != shadowCol {
		t.Fatal("TextShadow не применён")
	}

	ctx := NewContext(&dirtyMockWindow{}, nil)
	bg, hoverBg, downBg, _, _ := ctx.resolveButtonColors(st)
	if bg.A != 0 || hoverBg.A != 0 || downBg.A != 0 {
		t.Fatalf("для полностью прозрачного Bg подложка при наведении/нажатии должна оставаться прозрачной: bg=%+v hover=%+v down=%+v", bg, hoverBg, downBg)
	}
}

func TestTextInputAlignCenterAndFocusBorderColor(t *testing.T) {
	fm, err := render.LoadDefaultSystemFont()
	if err != nil {
		t.Skip("системный шрифт недоступен в тестовой среде")
	}
	win := &dirtyMockWindow{}
	ctx := NewContext(win, fm)
	ctx.Canvas = render.NewCanvas(320, 120)
	ctx.Theme = DarkTheme()

	val := ""
	ctx.Render(func(c *Context) {
		c.TextInputOpts("inp_center", &val, "Код сессии",
			Width(220),
			Height(40),
			TextAlignCenter(),
			BorderColor(render.RGBA(0, 0, 0, 0)),
			FocusBorderColor(render.RGBA(0, 0, 0, 0)),
		)
	})

	// Переводим фокус на поле и рендерим кадр — проверяем отсутствие рамки фокуса
	ctx.FocusID = ctx.ResolveID("inp", "inp_center")
	ctx.Canvas.Clear(render.RGB(10, 10, 10))
	ctx.Render(func(c *Context) {
		c.TextInputOpts("inp_center", &val, "Код сессии",
			Width(220),
			Height(40),
			TextAlignCenter(),
			Bg(render.RGB(10, 10, 10)),
			BorderColor(render.RGBA(0, 0, 0, 0)),
			FocusBorderColor(render.RGBA(0, 0, 0, 0)),
		)
	})

	// Верхняя граница поля (y=0) не должна окраситься в Theme.BorderFocus
	topPx := ctx.Canvas.Pixels[10*4:]
	if topPx[2] == ctx.Theme.BorderFocus.R && topPx[1] == ctx.Theme.BorderFocus.G && topPx[0] == ctx.Theme.BorderFocus.B {
		t.Fatal("внутренняя рамка фокуса не должна рисоваться при FocusBorderColor(transparent)")
	}
}

func TestLayoutRectInsidePushOverlayAndHoverOptions(t *testing.T) {
	ctx := newTestContext(600, 400)
	hBg := render.Hex(0x223344)
	hBr := render.Hex(0x556677)
	st := ApplyOptions([]Option{
		Bg(render.RGBA(0, 0, 0, 0)),
		HoverBg(hBg),
		HoverBorderColor(hBr),
	})
	_, hoverBg, _, _, _ := ctx.resolveButtonColors(st)
	if hoverBg != hBg {
		t.Fatalf("ожидался HoverBg=%+v, получено %+v", hBg, hoverBg)
	}
	if st.HoverBorderColor == nil || *st.HoverBorderColor != hBr {
		t.Fatal("HoverBorderColor не применён")
	}

	var overlayArea render.Rect
	var dockArea render.Rect
	panelRect := render.Rect{X: 300, Y: 20, W: 280, H: 360}
	ctx.Render(func(c *Context) {
		// Сдвигаем основной layout вниз
		c.allocRect(100, 200)
		c.PushOverlay(panelRect, func() {
			c.LayoutRect(panelRect, func() {
				overlayArea = c.CurrentArea()
				c.DockTop(0, 56, func() {
					dockArea = c.CurrentArea()
				}, func() {})
			})
		})
	})

	if overlayArea != panelRect {
		t.Fatalf("ожидалась область раскладки %+v, получено %+v", panelRect, overlayArea)
	}
	wantDock := render.Rect{X: 300, Y: 20, W: 280, H: 56}
	if dockArea != wantDock {
		t.Fatalf("ожидалась область DockTop %+v, получено %+v", wantDock, dockArea)
	}
}

func TestWrapTextLinesWordBoundaryAndLabelAlignCenter(t *testing.T) {
	fm, err := render.LoadDefaultSystemFont()
	if err != nil {
		t.Skip("системный шрифт недоступен в тестовой среде")
	}
	win := &dirtyMockWindow{}
	ctx := NewContext(win, fm)
	ctx.Canvas = render.NewCanvas(400, 300)
	ctx.Theme = DarkTheme()

	pxSize := ctx.S(14)
	wPart1, _ := fm.MeasureTextStyled("слово ", pxSize, render.FontRegular)
	wPart2, _ := fm.MeasureTextStyled("слово э", pxSize, render.FontRegular)

	// Задаём ширину между "слово " и "слово э", чтобы "это" не помещалось даже на одну букву
	maxW := (wPart1 + wPart2) / 2
	if maxW <= wPart1 {
		maxW = wPart1 + 1
	}

	text := "слово это"
	runes := []rune(text)
	lines := ctx.WrapTextLines(runes, maxW, pxSize, render.FontRegular)
	if len(lines) != 2 {
		t.Fatalf("ожидалось 2 строки, получено %d", len(lines))
	}

	line1 := string(runes[lines[0].start:lines[0].end])
	line2 := string(runes[lines[1].start:lines[1].end])
	if line1 != "слово " {
		t.Fatalf("ожидалась первая строка 'слово ', получено %q", line1)
	}
	if line2 != "это" {
		t.Fatalf("слово 'это' разорвалось: ожидалась вторая строка 'это', получено %q", line2)
	}

	// Проверяем, что длинное непрерывное слово без пробелов переносится посимвольно без паники
	longWord := "оченьдлинноесловобезпробелов"
	linesLong := ctx.WrapTextLines([]rune(longWord), maxW, pxSize, render.FontRegular)
	if len(linesLong) < 2 {
		t.Fatalf("длинное слово должно было быть перенесено посимвольно, строк %d", len(linesLong))
	}

	// Проверяем рендеринг многострочного текста с TextAlignCenter внутри карточки
	rendered := false
	ctx.Render(func(c *Context) {
		c.Card(200, 100, 10, func() {
			c.LabelOpts("строка раз это два", WordWrap(true), TextAlignCenter(), Width(180))
			rendered = true
		})
	})
	if !rendered {
		t.Fatal("виджет LabelOpts не был отрендерен")
	}
}

