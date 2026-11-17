package ui

import (
	"testing"

	"git.gtw.su/kirpich/anyui/render"
)

func TestResolveIDAndScope(t *testing.T) {
	ctx := newTestContext(400, 300)
	var ids []string

	ctx.Render(func(c *Context) {
		ids = append(ids,
			c.ResolveID("btn", ""),
			c.ResolveID("btn", ""),
			c.ResolveID("btn", "custom"),
		)
		c.IDScope("row_1", func() {
			ids = append(ids,
				c.ResolveID("btn", ""),
				c.ResolveID("btn", "save"),
			)
			c.IDScope(42, func() {
				ids = append(ids, c.ResolveID("inp", ""))
			})
		})
		ids = append(ids, c.ResolveID("btn", ""))
	})

	want := []string{
		"btn#0",
		"btn#1",
		"custom",
		"row_1/btn#0",
		"row_1/save",
		"row_1/42/inp#0",
		"btn#2",
	}
	if len(ids) != len(want) {
		t.Fatalf("expected %d ids, got %d", len(want), len(ids))
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("id[%d]: expected %q, got %q", i, want[i], ids[i])
		}
	}
}

func TestCardAutoHeightByDefaultAndFillHeight(t *testing.T) {
	ctx := newTestContext(400, 400)
	var afterCardY int

	// 2 кадра для стабилизации AutoHeight по умолчанию
	for frame := 0; frame < 2; frame++ {
		ctx.Render(func(c *Context) {
			c.CardOpts(func() {
				c.Label("Короткий текст", 14, c.Theme.TextPrimary)
			}, Padding(10))
			c.ButtonOpts("", "Ниже карточки")
			afterCardY = c.topLayout().cursorY
		})
	}

	if afterCardY <= 0 || afterCardY >= 300 {
		t.Fatalf("Card должен подстраиваться по содержимому, ожидался cursorY < 300, получено %d", afterCardY)
	}

	// С FillHeight карточка занимает оставшуюся высоту контейнера
	var fillRect render.Rect
	ctx.Render(func(c *Context) {
		c.CardOpts(func() {
			fillRect = c.CurrentArea()
		}, FillHeight(), Padding(0))
	})
	if fillRect.H != 400 {
		t.Fatalf("Card с FillHeight должен занимать всю высоту 400, получено %d", fillRect.H)
	}
}

func TestPageLayout(t *testing.T) {
	ctx := newTestContext(500, 400)
	var hdr, body, ftr render.Rect

	ctx.Render(func(c *Context) {
		c.Page(PageConfig{GapDp: 10, HeaderDp: 50, FooterDp: 40},
			func() { hdr = c.CurrentArea() },
			func() { body = c.CurrentArea() },
			func() { ftr = c.CurrentArea() },
		)
	})

	if hdr.H != 50 {
		t.Fatalf("ожидалась высота шапки 50, получено %d", hdr.H)
	}
	if ftr.H != 40 || ftr.Y != 360 {
		t.Fatalf("ожидался футер Y=360 H=40, получено %+v", ftr)
	}
	if body.Y <= hdr.Y+hdr.H || body.Y >= ftr.Y {
		t.Fatalf("некорректное расположение тела Page: hdr=%+v body=%+v ftr=%+v", hdr, body, ftr)
	}
}
