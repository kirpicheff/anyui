package anyui

import (
	"testing"

	"github.com/kirpicheff/anyui/render"
	"github.com/kirpicheff/anyui/ui"
)

func TestMousePosDpAndPx(t *testing.T) {
	mw := &stylerMockWindow{w: 800, h: 600, scale: 1.5}
	ctx := ui.NewContext(mw, nil)
	u := &UI{ctx: ctx}

	ctx.MouseX = 150
	ctx.MouseY = 300

	pxX, pxY := u.MousePosPx()
	if pxX != 150 || pxY != 300 {
		t.Fatalf("MousePosPx: got (%d, %d), expected (150, 300)", pxX, pxY)
	}

	dpX, dpY := u.MousePosDp()
	if dpX != 100 || dpY != 200 {
		t.Fatalf("MousePosDp at scale 1.5: got (%d, %d), expected (100, 200)", dpX, dpY)
	}
}

func TestOrientationAndSplitPane(t *testing.T) {
	mw := &stylerMockWindow{w: 600, h: 400, scale: 1.0}
	ctx := ui.NewContext(mw, nil)
	u := &UI{ctx: ctx}

	ratioH := float32(0.5)
	ratioV := float32(0.3)

	ctx.Render(func(_ *ui.Context) {
		u.SplitPane("sp_h", Horizontal, &ratioH, func() {}, func() {})
		u.SplitPane("sp_v", Vertical, &ratioV, func() {}, func() {})
	})

	if ratioH != 0.5 || ratioV != 0.3 {
		t.Errorf("unexpected split ratios: h=%v, v=%v", ratioH, ratioV)
	}
}

func TestNavyTheme(t *testing.T) {
	th := NavyTheme()
	if th.Bg != render.Hex(0x0F172A) {
		t.Fatalf("unexpected NavyTheme Bg: %v", th.Bg)
	}
}
