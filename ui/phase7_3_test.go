package ui

import (
	"testing"

	"git.gtw.su/kirpich/anyui/platform"
)

type titleBarSpyWindow struct {
	mockWindow
	minimized int
	maximized int
	restored  int
	closed    int
	isMax     bool
}

func (s *titleBarSpyWindow) Minimize() { s.minimized++ }
func (s *titleBarSpyWindow) Maximize() {
	s.maximized++
	s.isMax = true
}
func (s *titleBarSpyWindow) Restore() {
	s.restored++
	s.isMax = false
}
func (s *titleBarSpyWindow) IsMaximized() bool { return s.isMax }
func (s *titleBarSpyWindow) Close()            { s.closed++ }

func TestTitleBarAndHitTest(t *testing.T) {
	win := &titleBarSpyWindow{mockWindow: mockWindow{w: 800, h: 600, scale: 1.0}}
	ctx := NewContext(win, nil)

	ctx.Render(func(c *Context) {
		c.TitleBar("tb", "AnyUI Demo", TitleBarConfig{
			HeightDp: 36,
			LeftContent: func() {
				c.ButtonOpts("tb_btn", "Меню", Width(60), Height(26))
			},
		})
	})

	// Свободная область шапки должна возвращать HitCaption
	if res := ctx.TitleBarHitTest(400, 18); res != platform.HitCaption {
		t.Fatalf("ожидалось HitCaption в свободной части шапки, получено %v", res)
	}
	// Область кнопки закрытия (правые 44px) должна возвращать HitClient
	if res := ctx.TitleBarHitTest(785, 18); res != platform.HitClient {
		t.Fatalf("ожидалось HitClient над кнопкой Close, получено %v", res)
	}
	// Область ниже шапки должна возвращать HitClient
	if res := ctx.TitleBarHitTest(400, 80); res != platform.HitClient {
		t.Fatalf("ожидалось HitClient ниже шапки, получено %v", res)
	}

	// Клик по кнопке Minimize (3-я справа: 800 - 44*3 .. 800 - 44*2)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseMove, X: 690, Y: 18})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 690, Y: 18})
	ctx.Render(func(c *Context) {
		c.TitleBar("tb", "AnyUI Demo", TitleBarConfig{HeightDp: 36})
	})
	if win.minimized != 1 {
		t.Fatalf("ожидался 1 вызов Minimize, получено %d", win.minimized)
	}

	// Клик по кнопке Maximize (2-я справа: 800 - 44*2 .. 800 - 44)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 734, Y: 18})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 734, Y: 18})
	ctx.Render(func(c *Context) {
		c.TitleBar("tb", "AnyUI Demo", TitleBarConfig{HeightDp: 36})
	})
	if win.maximized != 1 {
		t.Fatalf("ожидался 1 вызов Maximize, получено %d", win.maximized)
	}

	// Клик по кнопке Close (1-я справа: 800 - 44 .. 800)
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseUp, Button: platform.MouseLeft, X: 780, Y: 18})
	ctx.HandleEvent(platform.Event{Type: platform.EventMouseDown, Button: platform.MouseLeft, X: 780, Y: 18})
	ctx.Render(func(c *Context) {
		c.TitleBar("tb", "AnyUI Demo", TitleBarConfig{HeightDp: 36})
	})
	if win.closed != 1 {
		t.Fatalf("ожидался 1 вызов Close, получено %d", win.closed)
	}
}

func TestLocalizationStrings(t *testing.T) {
	ru := RussianStrings()
	en := EnglishStrings()
	if ru.Today == "" || en.Today == "" || ru.Today == en.Today {
		t.Fatalf("ожидались разные непустые переводы Today: ru=%q en=%q", ru.Today, en.Today)
	}
	if len(en.Months) != 12 || len(en.Weekdays) != 7 {
		t.Fatal("некорректная длина массивов месяцев или дней недели")
	}

	ctx := newTestContext(600, 400)
	// Частичное переопределение должно подтягивать значения по умолчанию для незаполненных полей
	ctx.Strings = Strings{Cut: "Обрезать"}
	got := ctx.str()
	if got.Cut != "Обрезать" {
		t.Fatalf("ожидалось пользовательское Cut, получено %q", got.Cut)
	}
	if got.Copy != ru.Copy || got.Months[0] != ru.Months[0] {
		t.Fatal("пустые поля локализации не откатились к значениям по умолчанию")
	}
}
