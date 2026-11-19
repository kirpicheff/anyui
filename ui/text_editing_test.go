package ui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
)

func TestTextInputUndoRedoAndWordNav(t *testing.T) {
	ctx := newTestContext(400, 200)
	val := "hello world"
	draw := func(c *Context) {
		c.TextInputOpts("ti", &val, "", Width(240))
	}

	ctx.Render(draw)
	ctx.FocusID = "ti"
	ctx.caretPos["ti"] = len([]rune(val))
	ctx.textSelStart["ti"] = len([]rune(val))

	// Ctrl+Backspace удаляет слово "world"
	ctx.HandleEvent(platform.Event{
		Type:    platform.EventKeyDown,
		KeyCode: KeyBackspace,
		Mods:    platform.Modifiers{Ctrl: true},
	})
	ctx.Render(draw)
	if val != "hello " {
		t.Fatalf("expected 'hello ' after Ctrl+Backspace, got %q", val)
	}

	// Ctrl+Z отменяет удаление
	ctx.HandleEvent(platform.Event{
		Type:    platform.EventKeyDown,
		KeyCode: KeyZ,
		Mods:    platform.Modifiers{Ctrl: true},
	})
	ctx.Render(draw)
	if val != "hello world" {
		t.Fatalf("expected 'hello world' after Ctrl+Z, got %q", val)
	}

	// Ctrl+Y повторяет удаление
	ctx.HandleEvent(platform.Event{
		Type:    platform.EventKeyDown,
		KeyCode: KeyY,
		Mods:    platform.Modifiers{Ctrl: true},
	})
	ctx.Render(draw)
	if val != "hello " {
		t.Fatalf("expected 'hello ' after Ctrl+Y, got %q", val)
	}

	// Ctrl+Shift+Z после Ctrl+Z также повторяет
	ctx.HandleEvent(platform.Event{
		Type:    platform.EventKeyDown,
		KeyCode: KeyZ,
		Mods:    platform.Modifiers{Ctrl: true},
	})
	ctx.Render(draw)
	ctx.HandleEvent(platform.Event{
		Type:    platform.EventKeyDown,
		KeyCode: KeyZ,
		Mods:    platform.Modifiers{Ctrl: true, Shift: true},
	})
	ctx.Render(draw)
	if val != "hello " {
		t.Fatalf("expected 'hello ' after Ctrl+Shift+Z, got %q", val)
	}

	// Проверка навигации по словам Ctrl+Left / Ctrl+Shift+Left
	val = "one two three"
	ctx.caretPos["ti"] = len([]rune(val))
	ctx.textSelStart["ti"] = len([]rune(val))
	ctx.HandleEvent(platform.Event{
		Type:    platform.EventKeyDown,
		KeyCode: KeyLeft,
		Mods:    platform.Modifiers{Ctrl: true, Shift: true},
	})
	ctx.Render(draw)
	a, b, ok := selectionBounds(ctx.caretPos["ti"], ctx.textSelStart["ti"], len([]rune(val)))
	if !ok || string([]rune(val)[a:b]) != "three" {
		t.Fatalf("expected 'three' selected, got ok=%v %q", ok, string([]rune(val)[a:b]))
	}
}

func TestDoubleAndTripleClickCounts(t *testing.T) {
	ctx := newTestContext(400, 200)
	val := "alpha beta"
	draw := func(c *Context) {
		c.TextInputOpts("ti_click", &val, "", Width(240))
	}

	// Двойной клик
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 16, Y: 18})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 16, Y: 18})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 16, Y: 18})
	if ctx.ClickCount() != 2 {
		t.Fatalf("expected ClickCount() == 2, got %d", ctx.ClickCount())
	}
	ctx.Render(draw)
	a, b, ok := selectionBounds(ctx.caretPos["ti_click"], ctx.textSelStart["ti_click"], len([]rune(val)))
	if !ok || string([]rune(val)[a:b]) != "alpha" {
		t.Fatalf("expected 'alpha' selected on double click, got %q", string([]rune(val)[a:b]))
	}

	// Тройной клик
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 16, Y: 18})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 16, Y: 18})
	if ctx.ClickCount() != 3 {
		t.Fatalf("expected ClickCount() == 3, got %d", ctx.ClickCount())
	}
	ctx.Render(draw)
	a, b, ok = selectionBounds(ctx.caretPos["ti_click"], ctx.textSelStart["ti_click"], len([]rune(val)))
	if !ok || string([]rune(val)[a:b]) != "alpha beta" {
		t.Fatalf("expected full text selected on triple click, got %q", string([]rune(val)[a:b]))
	}
}

func TestTextAreaLineNavigationAndLineNumbers(t *testing.T) {
	ctx := newTestContext(400, 200)
	val := "first\nsecond\nthird"
	draw := func(c *Context) {
		c.TextArea("ta", &val, "", Width(300), Height(100), LineNumbers(true))
	}

	ctx.Render(draw)
	ctx.FocusID = "ta"
	ctx.caretPos["ta"] = 0
	ctx.textSelStart["ta"] = 0

	// Down переходит на вторую строку
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyDown})
	ctx.Render(draw)
	if ctx.caretPos["ta"] < 6 || ctx.caretPos["ta"] > 12 {
		t.Fatalf("expected caret on second line (6..12), got %d", ctx.caretPos["ta"])
	}

	// End переходит в конец второй строки
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEnd})
	ctx.Render(draw)
	if ctx.caretPos["ta"] != 12 {
		t.Fatalf("expected caret at end of second line (12), got %d", ctx.caretPos["ta"])
	}

	// Ctrl+End переходит в конец текста
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyEnd, Mods: platform.Modifiers{Ctrl: true}})
	ctx.Render(draw)
	if ctx.caretPos["ta"] != len([]rune(val)) {
		t.Fatalf("expected caret at end of text (%d), got %d", len([]rune(val)), ctx.caretPos["ta"])
	}
}

func TestTextContextMenuOpensOnRightClick(t *testing.T) {
	ctx := newTestContext(400, 200)
	val := "context"
	draw := func(c *Context) {
		c.TextInputOpts("ti_cm", &val, "", Width(200))
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseRight, X: 40, Y: 18})
	ctx.Render(draw)

	if !ctx.openState["cm:txt_cm:ti_cm"] {
		t.Fatalf("expected text context menu to open on right click")
	}
}

func TestSelectableLabelHyperlinkAndMaskedInput(t *testing.T) {
	ctx := newTestContext(400, 200)
	clicked := false
	maskVal := ""
	draw := func(c *Context) {
		c.SelectableLabel("sl", "copy-me-123")
		if c.Hyperlink("hl", "Click link") {
			clicked = true
		}
		c.MaskedInput("mi", &maskVal, "00.00.0000", Width(160), Error("Неверная дата"))
	}

	ctx.Render(draw)
	// SelectableLabel: Ctrl+A + Ctrl+C
	ctx.FocusID = "sl"
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyA, Mods: platform.Modifiers{Ctrl: true}})
	ctx.Render(draw)
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyC, Mods: platform.Modifiers{Ctrl: true}})
	ctx.Render(draw)
	clip, _ := ctx.Win.ClipboardText()
	if clip != "copy-me-123" {
		t.Fatalf("expected clipboard 'copy-me-123', got %q", clip)
	}

	// Hyperlink: клик
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 10, Y: 25})
	ctx.Render(draw)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 10, Y: 25})
	ctx.Render(draw)
	if !clicked {
		t.Fatalf("expected Hyperlink click to return true")
	}

	// MaskedInput: ввод цифр с пропуском разделителей
	ctx.FocusID = "mi"
	ctx.caretPos["mi"] = 0
	ctx.textSelStart["mi"] = 0
	for _, ch := range []rune{'1', '7', '1', '1', '2', '0', '2', '6'} {
		ctx.HandleEvent(platform.Event{Type: platform.EventChar, Char: ch})
	}
	ctx.Render(draw)
	if maskVal != "17.11.2026" {
		t.Fatalf("expected masked value '17.11.2026', got %q", maskVal)
	}

	// Backspace в MaskedInput перепрыгивает разделитель и удаляет последнюю цифру
	ctx.HandleEvent(platform.Event{Type: platform.EventKeyDown, KeyCode: KeyBackspace})
	ctx.Render(draw)
	if maskVal != "17.11.202" {
		t.Fatalf("expected masked value '17.11.202' after Backspace, got %q", maskVal)
	}
}

func TestTextInputPasswordToggle(t *testing.T) {
	ctx := newTestContext(400, 200)
	val := "secret123"
	showExt := false

	drawExt := func(c *Context) {
		c.TextInputOpts("ti_pass_ext", &val, "Пароль", Width(200), PasswordToggle(&showExt))
	}

	ctx.Render(drawExt)
	if showExt {
		t.Fatal("ожидалось showExt == false")
	}

	// Координата глазика внутри поля: r.X=0, r.Y=0, W=200, padX=12, iconSize=16 -> eyeX = 172..188, eyeY = 11..27
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 175, Y: 15})
	ctx.Render(drawExt)
	if !showExt {
		t.Fatal("ожидалось showExt == true после клика по глазику")
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 175, Y: 15})
	ctx.Render(drawExt)
	if showExt {
		t.Fatal("ожидалось showExt == false после повторного клика")
	}

	drawInt := func(c *Context) {
		c.TextInputOpts("ti_pass_int", &val, "Пароль", Width(200), PasswordToggle())
	}
	ctx.Render(drawInt)
	if ctx.passwordVisible["ti_pass_int"] {
		t.Fatal("ожидалось passwordVisible == false по умолчанию")
	}

	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 175, Y: 15})
	ctx.Render(drawInt)
	if !ctx.passwordVisible["ti_pass_int"] {
		t.Fatal("ожидалось passwordVisible == true после клика")
	}

	// PasswordToggle + Clearable: проверяем очистку и переключение видимости
	valBoth := "password123"
	showBoth := false
	drawBoth := func(c *Context) {
		c.TextInputOpts("ti_both", &valBoth, "Пароль", Width(200), PasswordToggle(&showBoth), Clearable(true))
	}
	ctx.Render(drawBoth)

	// Кликаем по кнопке Clear (clearX = 148..164, Y = 11..27)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 155, Y: 15})
	ctx.Render(drawBoth)
	if valBoth != "" {
		t.Fatalf("ожидалась очистка поля при клике на Clear, получено %q", valBoth)
	}

	// Кликаем по глазику (eyeX = 172..188)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 175, Y: 15})
	ctx.Render(drawBoth)
	if !showBoth {
		t.Fatal("ожидалось showBoth == true после клика по глазику")
	}
}
