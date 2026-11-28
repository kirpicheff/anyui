package ui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
)

// Проверяет размещение TabBar в LeftContent заголовка: высота, ширина слота и hit-test.
func TestTitleBarTabsInLeftContent(t *testing.T) {
	win := &titleBarSpyWindow{mockWindow: mockWindow{w: 800, h: 600, scale: 1.0}}
	ctx := NewContext(win, nil)
	tabs := []TabItem{{ID: "t1", Title: "Первая"}, {ID: "t2", Title: "Вторая"}}
	active := "t1"

	render := func() TabBarEvent {
		var ev TabBarEvent
		ctx.Render(func(c *Context) {
			c.TitleBar("tb", "RDP", TitleBarConfig{
				HeightDp: 36,
				LeftContent: func() {
					ev = c.TabBar("tabs", tabs, &active)
				},
			})
		})
		return ev
	}
	render()

	if len(ctx.tabBars) == 0 {
		t.Fatal("TabBar в заголовке не зарегистрирован")
	}
	reg := ctx.tabBars[len(ctx.tabBars)-1]
	t.Logf("titleBarRect=%+v barRect=%+v tabsRect=%+v tabSize=%d", ctx.titleBarRect, reg.barRect, reg.tabsRect, reg.tabSizePx)

	tb := ctx.titleBarRect
	if reg.barRect.H > tb.H {
		t.Fatalf("высота TabBar %d больше высоты заголовка %d", reg.barRect.H, tb.H)
	}
	if reg.barRect.X+reg.barRect.W > tb.X+tb.W-3*44 {
		t.Fatalf("TabBar заходит на системные кнопки: %+v", reg.barRect)
	}
	// Центр первой вкладки - HitClient
	cx, cy := reg.tabsRect.X+20, tb.Y+tb.H/2
	if res := ctx.TitleBarHitTest(Px(cx), Px(cy)); res != platform.HitClient {
		t.Fatalf("ожидалось HitClient над вкладкой, получено %v", res)
	}
	// Свободная область справа от вкладок - HitCaption
	if res := ctx.TitleBarHitTest(Px(600), Px(cy)); res != platform.HitCaption {
		t.Fatalf("ожидалось HitCaption в свободной части, получено %v", res)
	}
}
