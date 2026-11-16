package ui

import (
	"testing"
	"time"

	"anyui/platform"
	"anyui/render"
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

