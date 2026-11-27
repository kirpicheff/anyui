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

func TestDrawTextMono(t *testing.T) {
	reg, bold, mono := DefaultTestFonts()
	fm, err := render.NewFontFromBytes(reg)
	if err != nil {
		t.Fatalf("NewFontFromBytes(reg): %v", err)
	}
	if err := fm.RegisterFontBytes(render.FontBold, bold); err != nil {
		t.Fatalf("RegisterFontBytes(bold): %v", err)
	}
	if err := fm.RegisterFontBytes(render.FontMono, mono); err != nil {
		t.Fatalf("RegisterFontBytes(mono): %v", err)
	}

	mw1 := &stylerMockWindow{w: 320, h: 64, scale: 1.0}
	ctx1 := ui.NewContext(mw1, fm)
	u1 := &UI{ctx: ctx1}
	ctx1.Canvas.Clear(render.RGB(0, 0, 0))
	u1.DrawText(10, 10, "iiiiWWWW", 16, render.RGB(255, 255, 255))

	mw2 := &stylerMockWindow{w: 320, h: 64, scale: 1.0}
	ctx2 := ui.NewContext(mw2, fm)
	u2 := &UI{ctx: ctx2}
	ctx2.Canvas.Clear(render.RGB(0, 0, 0))
	u2.DrawTextMono(10, 10, "iiiiWWWW", 16, render.RGB(255, 255, 255))

	nonZero := 0
	diff := 0
	for i := 0; i < len(ctx2.Canvas.Pixels); i += 4 {
		if ctx2.Canvas.Pixels[i] != 0 || ctx2.Canvas.Pixels[i+1] != 0 || ctx2.Canvas.Pixels[i+2] != 0 {
			nonZero++
		}
		if ctx1.Canvas.Pixels[i] != ctx2.Canvas.Pixels[i] ||
			ctx1.Canvas.Pixels[i+1] != ctx2.Canvas.Pixels[i+1] ||
			ctx1.Canvas.Pixels[i+2] != ctx2.Canvas.Pixels[i+2] {
			diff++
		}
	}
	if nonZero == 0 {
		t.Fatal("DrawTextMono не отрисовал ни одного пикселя текста")
	}
	if diff == 0 {
		t.Fatal("DrawTextMono должен отличаться от обычного DrawText (пропорционального шрифта)")
	}
}
