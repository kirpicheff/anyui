//go:build linux

package platform

import (
	"image"
	"image/color"
	"math"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const csdLogicalHeight = 32

const (
	csdBtnNone = 0
	csdBtnMin  = 1
	csdBtnMax  = 2
	csdBtnClose = 3
)

func (w *waylandWindow) csdHeightPx() int {
	w.mu.Lock()
	enabled := w.csdEnabled
	fs := w.fullscreen
	scale := w.scale
	w.mu.Unlock()
	if !enabled || fs {
		return 0
	}
	if scale <= 0 {
		scale = 1.0
	}
	h := int(math.Round(float64(csdLogicalHeight) * float64(scale)))
	if h < 24 {
		h = 24
	}
	return h
}

func (w *waylandWindow) csdButtonWidthPx() int {
	w.mu.Lock()
	scale := w.scale
	w.mu.Unlock()
	if scale <= 0 {
		scale = 1.0
	}
	return int(math.Round(42.0 * float64(scale)))
}

func (w *waylandWindow) csdHitButton(px, py, totalW, barH int) int {
	if barH <= 0 || py < 0 || py >= barH {
		return csdBtnNone
	}
	bw := w.csdButtonWidthPx()
	if px >= totalW-bw && px < totalW {
		return csdBtnClose
	}
	if px >= totalW-2*bw && px < totalW-bw {
		return csdBtnMax
	}
	if px >= totalW-3*bw && px < totalW-2*bw {
		return csdBtnMin
	}
	return csdBtnNone
}

func (w *waylandWindow) csdEdgeHit(px, py, totalW, totalH int) HitTestResult {
	w.mu.Lock()
	scale := w.scale
	maxed := w.maximized
	w.mu.Unlock()
	if maxed {
		return HitClient
	}
	border := int(math.Round(5.0 * float64(scale)))
	if border < 4 {
		border = 4
	}
	return ComputeFramelessHitTest(px, py, totalW, totalH, border, 0, maxed, nil)
}

func (w *waylandWindow) handleCSDMouseMove(px, py int) bool {
	barH := w.csdHeightPx()
	if barH <= 0 {
		return false
	}
	w.mu.Lock()
	totalW, totalH := w.width, w.height
	oldHover := w.csdHoverBtn
	w.mu.Unlock()

	edge := w.csdEdgeHit(px, py, totalW, totalH)
	if edge != HitClient {
		if edgeCursor := csdEdgeToCursor(edge); edgeCursor != CursorArrow {
			if w.disp.seat != nil {
				w.disp.seat.applyCursor(edgeCursor)
			}
			return true
		}
	}

	if py >= 0 && py < barH {
		btn := w.csdHitButton(px, py, totalW, barH)
		if btn != oldHover {
			w.mu.Lock()
			w.csdHoverBtn = btn
			w.mu.Unlock()
			w.needsPaint.Store(true)
		}
		if w.disp.seat != nil {
			w.disp.seat.applyCursor(CursorArrow)
		}
		return true
	}

	if oldHover != csdBtnNone {
		w.mu.Lock()
		w.csdHoverBtn = csdBtnNone
		w.mu.Unlock()
		w.needsPaint.Store(true)
	}
	return false
}

func (w *waylandWindow) handleCSDMouseDown(px, py int, btn MouseButton, seatID, serial uint32) bool {
	barH := w.csdHeightPx()
	if barH <= 0 {
		return false
	}

	w.mu.Lock()
	totalW, totalH := w.width, w.height
	lastClick := w.csdLastClick
	w.mu.Unlock()

	if btn == MouseLeft {
		edge := w.csdEdgeHit(px, py, totalW, totalH)
		if edgeCode := hitTestToXDGRegionEdge(edge); edgeCode != 0 && seatID != 0 {
			var wr wlWriter
			wr.uint32(seatID)
			wr.uint32(serial)
			wr.uint32(edgeCode)
			_ = w.disp.conn.send(w.toplevelID, xdgToplevelResize, wr.bytes())
			return true
		}
	}

	if py < 0 || py >= barH {
		return false
	}
	if btn != MouseLeft {
		return true
	}

	hitBtn := w.csdHitButton(px, py, totalW, barH)
	switch hitBtn {
	case csdBtnMin:
		w.Minimize()
		return true
	case csdBtnMax:
		if w.IsMaximized() {
			w.Restore()
		} else {
			w.Maximize()
		}
		return true
	case csdBtnClose:
		w.Close()
		return true
	}

	now := time.Now()
	w.mu.Lock()
	w.csdLastClick = now
	w.mu.Unlock()
	if !lastClick.IsZero() && now.Sub(lastClick) < 400*time.Millisecond {
		if w.IsMaximized() {
			w.Restore()
		} else {
			w.Maximize()
		}
		return true
	}

	if seatID != 0 && w.toplevelID != 0 {
		var wr wlWriter
		wr.uint32(seatID)
		wr.uint32(serial)
		_ = w.disp.conn.send(w.toplevelID, xdgToplevelMove, wr.bytes())
	}
	return true
}

func csdEdgeToCursor(hit HitTestResult) CursorType {
	switch hit {
	case HitLeft, HitRight:
		return CursorResizeH
	case HitTop, HitBottom:
		return CursorResizeV
	case HitTopLeft, HitBottomRight:
		return CursorResizeNWSE
	case HitTopRight, HitBottomLeft:
		return CursorResizeNESW
	default:
		return CursorArrow
	}
}

func (w *waylandWindow) renderCSDTitleBar(dst []byte, totalW, barH int) {
	if barH <= 0 || totalW <= 0 || len(dst) < totalW*barH*4 {
		return
	}

	w.mu.Lock()
	title := w.csdTitle
	hover := w.csdHoverBtn
	scale := w.scale
	w.mu.Unlock()
	if title == "" {
		title = w.cfg.Title
	}
	if scale <= 0 {
		scale = 1.0
	}

	dark := w.cfg.DarkTitle || IsSystemDarkTheme()
	bgR, bgG, bgB := uint8(243), uint8(243), uint8(243)
	fgR, fgG, fgB := uint8(32), uint8(32), uint8(32)
	sepR, sepG, sepB := uint8(215), uint8(215), uint8(215)
	hovR, hovG, hovB := uint8(224), uint8(224), uint8(224)
	if dark {
		bgR, bgG, bgB = 32, 32, 36
		fgR, fgG, fgB = 232, 232, 236
		sepR, sepG, sepB = 56, 56, 62
		hovR, hovG, hovB = 55, 55, 62
	}
	if w.cfg.TitleBarColor != nil {
		bgR, bgG, bgB = w.cfg.TitleBarColor.R, w.cfg.TitleBarColor.G, w.cfg.TitleBarColor.B
	}
	if w.cfg.TitleTextColor != nil {
		fgR, fgG, fgB = w.cfg.TitleTextColor.R, w.cfg.TitleTextColor.G, w.cfg.TitleTextColor.B
	}

	fillRect := func(x0, y0, x1, y1 int, r, g, b uint8) {
		if x0 < 0 {
			x0 = 0
		}
		if y0 < 0 {
			y0 = 0
		}
		if x1 > totalW {
			x1 = totalW
		}
		if y1 > barH {
			y1 = barH
		}
		for y := y0; y < y1; y++ {
			row := y * totalW * 4
			for x := x0; x < x1; x++ {
				off := row + x*4
				dst[off] = b
				dst[off+1] = g
				dst[off+2] = r
				dst[off+3] = 255
			}
		}
	}

	// Фон шапки и нижняя граница
	fillRect(0, 0, totalW, barH-1, bgR, bgG, bgB)
	fillRect(0, barH-1, totalW, barH, sepR, sepG, sepB)

	bw := w.csdButtonWidthPx()
	// Подсветка кнопки под курсором
	switch hover {
	case csdBtnMin:
		fillRect(totalW-3*bw, 0, totalW-2*bw, barH-1, hovR, hovG, hovB)
	case csdBtnMax:
		fillRect(totalW-2*bw, 0, totalW-bw, barH-1, hovR, hovG, hovB)
	case csdBtnClose:
		fillRect(totalW-bw, 0, totalW, barH-1, 196, 43, 28)
	}

	// Рисуем иконки кнопок
	setPx := func(x, y int, r, g, b uint8) {
		if x < 0 || y < 0 || x >= totalW || y >= barH-1 {
			return
		}
		off := (y*totalW + x) * 4
		dst[off] = b
		dst[off+1] = g
		dst[off+2] = r
		dst[off+3] = 255
	}

	iconHalf := int(math.Round(4.5 * float64(scale)))
	if iconHalf < 4 {
		iconHalf = 4
	}
	cy := barH / 2

	// Кнопка Minimize (_)
	minCx := totalW - 2*bw - bw/2
	for x := minCx - iconHalf; x <= minCx+iconHalf; x++ {
		setPx(x, cy+2, fgR, fgG, fgB)
	}

	// Кнопка Maximize (□)
	maxCx := totalW - bw - bw/2
	for x := maxCx - iconHalf; x <= maxCx+iconHalf; x++ {
		setPx(x, cy-iconHalf, fgR, fgG, fgB)
		setPx(x, cy+iconHalf, fgR, fgG, fgB)
	}
	for y := cy - iconHalf; y <= cy+iconHalf; y++ {
		setPx(maxCx-iconHalf, y, fgR, fgG, fgB)
		setPx(maxCx+iconHalf, y, fgR, fgG, fgB)
	}

	// Кнопка Close (×)
	clsCx := totalW - bw/2
	cr, cg, cb := fgR, fgG, fgB
	if hover == csdBtnClose {
		cr, cg, cb = 255, 255, 255
	}
	for d := -iconHalf; d <= iconHalf; d++ {
		setPx(clsCx+d, cy+d, cr, cg, cb)
		setPx(clsCx+d, cy-d, cr, cg, cb)
	}

	// Заголовок окна (системный TrueType-шрифт с поддержкой кириллицы и HiDPI)
	if title != "" && totalW > 3*bw+24 {
		face := getCSDTitleFont(scale)
		dotY := cy + int(math.Round(4.5*float64(scale)))
		rgba := &bgraImage{pix: dst, stride: totalW * 4, rect: image.Rect(0, 0, totalW-3*bw-8, barH-1)}
		drawer := &font.Drawer{
			Dst:  rgba,
			Src:  image.NewUniform(color.RGBA{R: fgR, G: fgG, B: fgB, A: 255}),
			Face: face,
			Dot:  fixed.P(12, dotY),
		}
		drawer.DrawString(title)
	}
}

var (
	csdFontOnce  sync.Once
	csdFontOT    *opentype.Font
	csdFacesMu   sync.Mutex
	csdFacesByPx = make(map[int]font.Face)
)

func getCSDTitleFont(scale float32) font.Face {
	csdFontOnce.Do(func() {
		paths := []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
			"/usr/share/fonts/TTF/DejaVuSans.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
			"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
		}
		if out, err := exec.Command("fc-match", "-f", "%{file}", "sans").Output(); err == nil {
			if p := strings.TrimSpace(string(out)); p != "" {
				paths = append([]string{p}, paths...)
			}
		}
		for _, p := range paths {
			data, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			if f, err := opentype.Parse(data); err == nil {
				csdFontOT = f
				return
			}
		}
	})

	if csdFontOT == nil {
		return basicfont.Face7x13
	}

	px := int(math.Round(13.0 * float64(scale)))
	if px < 11 {
		px = 11
	}

	csdFacesMu.Lock()
	defer csdFacesMu.Unlock()
	if face, ok := csdFacesByPx[px]; ok {
		return face
	}
	face, err := opentype.NewFace(csdFontOT, &opentype.FaceOptions{
		Size:    float64(px),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return basicfont.Face7x13
	}
	csdFacesByPx[px] = face
	return face
}

type bgraImage struct {
	pix    []byte
	stride int
	rect   image.Rectangle
}

func (m *bgraImage) ColorModel() color.Model { return color.RGBAModel }
func (m *bgraImage) Bounds() image.Rectangle { return m.rect }
func (m *bgraImage) At(x, y int) color.Color {
	if !image.Pt(x, y).In(m.rect) {
		return color.RGBA{}
	}
	off := y*m.stride + x*4
	return color.RGBA{R: m.pix[off+2], G: m.pix[off+1], B: m.pix[off], A: m.pix[off+3]}
}
func (m *bgraImage) Set(x, y int, c color.Color) {
	if !image.Pt(x, y).In(m.rect) {
		return
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return
	}
	off := y*m.stride + x*4
	alpha := uint32(a >> 8)
	if alpha == 255 {
		m.pix[off] = uint8(b >> 8)
		m.pix[off+1] = uint8(g >> 8)
		m.pix[off+2] = uint8(r >> 8)
		m.pix[off+3] = 255
		return
	}
	inv := 255 - alpha
	m.pix[off] = uint8((uint32(m.pix[off])*inv + uint32(b>>8)*alpha) / 255)
	m.pix[off+1] = uint8((uint32(m.pix[off+1])*inv + uint32(g>>8)*alpha) / 255)
	m.pix[off+2] = uint8((uint32(m.pix[off+2])*inv + uint32(r>>8)*alpha) / 255)
	m.pix[off+3] = 255
}

