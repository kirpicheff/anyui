package render

import (
	"image"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// FontStyle задаёт начертание шрифта.
type FontStyle uint8

const (
	FontRegular FontStyle = iota
	FontBold
	FontMono
)

type glyphKey struct {
	r    rune
	size int
}

type cachedGlyph struct {
	w, h     int
	offX     int
	offY     int
	advance  int
	alpha    []byte
	notEmpty bool
}

type faceEntry struct {
	face       font.Face
	ascent     int
	lineHeight int
}

type fontSet struct {
	sfntFont *sfnt.Font
	faces    map[int]*faceEntry
	glyphs   map[glyphKey]*cachedGlyph
}

func newFontSet(f *sfnt.Font) *fontSet {
	return &fontSet{
		sfntFont: f,
		faces:    make(map[int]*faceEntry),
		glyphs:   make(map[glyphKey]*cachedGlyph),
	}
}

func (fs *fontSet) hasRune(r rune) bool {
	if fs == nil || fs.sfntFont == nil {
		return false
	}
	var buf sfnt.Buffer
	idx, err := fs.sfntFont.GlyphIndex(&buf, r)
	return err == nil && idx != 0
}

func (fs *fontSet) getFace(sizePx int) *faceEntry {
	if sizePx < 6 {
		sizePx = 6
	}
	if fe, ok := fs.faces[sizePx]; ok {
		return fe
	}
	face, err := opentype.NewFace(fs.sfntFont, &opentype.FaceOptions{
		Size:    float64(sizePx),
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil
	}
	m := face.Metrics()
	fe := &faceEntry{
		face:       face,
		ascent:     m.Ascent.Round(),
		lineHeight: m.Height.Round(),
	}
	fs.faces[sizePx] = fe
	return fe
}

func (fs *fontSet) getGlyph(r rune, sizePx int) *cachedGlyph {
	key := glyphKey{r: r, size: sizePx}
	if g, ok := fs.glyphs[key]; ok {
		return g
	}

	fe := fs.getFace(sizePx)
	if fe == nil {
		return nil
	}

	dr, mask, maskp, adv, ok := fe.face.Glyph(fixed.P(0, 0), r)
	g := &cachedGlyph{
		advance: adv.Round(),
	}
	if ok && !dr.Empty() && mask != nil {
		w := dr.Dx()
		h := dr.Dy()
		alpha := make([]byte, w*h)

		if alphaImg, isAlpha := mask.(*image.Alpha); isAlpha {
			for y := 0; y < h; y++ {
				srcOff := alphaImg.PixOffset(maskp.X, maskp.Y+y)
				copy(alpha[y*w:(y+1)*w], alphaImg.Pix[srcOff:srcOff+w])
			}
		} else {
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					_, _, _, a := mask.At(maskp.X+x, maskp.Y+y).RGBA()
					alpha[y*w+x] = uint8(a >> 8)
				}
			}
		}

		g.w = w
		g.h = h
		g.offX = dr.Min.X
		g.offY = dr.Min.Y
		g.alpha = alpha
		g.notEmpty = true
	}

	fs.glyphs[key] = g
	return g
}

// FontManager управляет начертаниями шрифтов (Regular, Bold, Mono), fallback-цепочкой и кешем глифов.
type FontManager struct {
	styles    map[FontStyle]*fontSet
	fallbacks []*fontSet
}

// NewFontFromBytes создаёт FontManager с основным шрифтом Regular из байтов TTF/OTF (удобно для //go:embed).
func NewFontFromBytes(data []byte) (*FontManager, error) {
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, err
	}
	return &FontManager{
		styles: map[FontStyle]*fontSet{
			FontRegular: newFontSet(f),
		},
	}, nil
}

// NewFontFromFile загружает основной шрифт Regular из файла на диске.
func NewFontFromFile(path string) (*FontManager, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return NewFontFromBytes(data)
}

// RegisterFontBytes регистрирует дополнительное начертание (например, FontBold или FontMono) из байтов.
func (fm *FontManager) RegisterFontBytes(style FontStyle, data []byte) error {
	f, err := sfnt.Parse(data)
	if err != nil {
		return err
	}
	if fm.styles == nil {
		fm.styles = make(map[FontStyle]*fontSet)
	}
	fm.styles[style] = newFontSet(f)
	return nil
}

// RegisterFontFile регистрирует дополнительное начертание из файла на диске.
func (fm *FontManager) RegisterFontFile(style FontStyle, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return fm.RegisterFontBytes(style, data)
}

// AddFallbackBytes добавляет запасной шрифт в конец fallback-цепочки для отсутствующих символов.
func (fm *FontManager) AddFallbackBytes(data []byte) error {
	f, err := sfnt.Parse(data)
	if err != nil {
		return err
	}
	fm.fallbacks = append(fm.fallbacks, newFontSet(f))
	return nil
}

// AddFallbackFile добавляет запасной шрифт из файла в конец fallback-цепочки.
func (fm *FontManager) AddFallbackFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return fm.AddFallbackBytes(data)
}

// LoadDefaultSystemFont загружает стандартный системный шрифт ОС, а также Bold, Mono и fallback-шрифты.
func LoadDefaultSystemFont() (*FontManager, error) {
	regularCandidates := []string{
		`C:\Windows\Fonts\segoeui.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
		`C:\Windows\Fonts\tahoma.ttf`,
		`/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf`,
		`/usr/share/fonts/TTF/DejaVuSans.ttf`,
	}
	var fm *FontManager
	var lastErr error
	for _, p := range regularCandidates {
		loaded, err := NewFontFromFile(p)
		if err == nil {
			fm = loaded
			break
		}
		lastErr = err
	}
	if fm == nil {
		return nil, lastErr
	}

	boldCandidates := []string{
		`C:\Windows\Fonts\segoeuib.ttf`,
		`C:\Windows\Fonts\arialbd.ttf`,
		`/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf`,
		`/usr/share/fonts/TTF/DejaVuSans-Bold.ttf`,
	}
	for _, p := range boldCandidates {
		if fm.RegisterFontFile(FontBold, p) == nil {
			break
		}
	}

	monoCandidates := []string{
		`C:\Windows\Fonts\consola.ttf`,
		`C:\Windows\Fonts\cour.ttf`,
		`/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf`,
		`/usr/share/fonts/TTF/DejaVuSansMono.ttf`,
	}
	for _, p := range monoCandidates {
		if fm.RegisterFontFile(FontMono, p) == nil {
			break
		}
	}

	fallbackCandidates := []string{
		`C:\Windows\Fonts\seguisym.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
	}
	for _, p := range fallbackCandidates {
		_ = fm.AddFallbackFile(p)
	}

	return fm, nil
}

func (fm *FontManager) styleSet(style FontStyle) *fontSet {
	if fs, ok := fm.styles[style]; ok && fs != nil {
		return fs
	}
	return fm.styles[FontRegular]
}

func (fm *FontManager) resolveGlyph(style FontStyle, r rune, sizePx int) *cachedGlyph {
	primary := fm.styleSet(style)
	if primary == nil {
		return nil
	}
	if r <= ' ' || primary.hasRune(r) {
		return primary.getGlyph(r, sizePx)
	}
	for _, fb := range fm.fallbacks {
		if fb.hasRune(r) {
			return fb.getGlyph(r, sizePx)
		}
	}
	return primary.getGlyph(r, sizePx)
}

// MeasureText возвращает ширину и высоту строки в пикселях для начертания Regular.
func (fm *FontManager) MeasureText(text string, sizePx int) (int, int) {
	return fm.MeasureTextStyled(text, sizePx, FontRegular)
}

// MeasureTextStyled возвращает ширину и высоту строки в пикселях для указанного начертания.
func (fm *FontManager) MeasureTextStyled(text string, sizePx int, style FontStyle) (int, int) {
	if fm == nil || text == "" {
		return 0, sizePx
	}
	fs := fm.styleSet(style)
	if fs == nil {
		return 0, sizePx
	}
	fe := fs.getFace(sizePx)
	if fe == nil {
		return 0, sizePx
	}
	width := 0
	for _, r := range text {
		g := fm.resolveGlyph(style, r, sizePx)
		if g != nil {
			width += g.advance
		}
	}
	return width, fe.lineHeight
}

// DrawText рисует строку текста начертанием Regular, где (x, y) — левый верхний угол строки.
func (fm *FontManager) DrawText(c *Canvas, x, y int, text string, sizePx int, col Color) {
	fm.DrawTextStyled(c, x, y, text, sizePx, col, FontRegular)
}

// DrawTextStyled рисует строку текста указанным начертанием.
func (fm *FontManager) DrawTextStyled(c *Canvas, x, y int, text string, sizePx int, col Color, style FontStyle) {
	if fm == nil || c == nil || text == "" || col.A == 0 {
		return
	}
	fs := fm.styleSet(style)
	if fs == nil {
		return
	}
	fe := fs.getFace(sizePx)
	if fe == nil {
		return
	}

	clip := c.CurrentClip()
	penX := x
	baselineY := y + fe.ascent

	for _, r := range text {
		g := fm.resolveGlyph(style, r, sizePx)
		if g == nil {
			continue
		}
		if g.notEmpty {
			gx := penX + g.offX
			gy := baselineY + g.offY

			// Быстрая проверка видимости глифа относительно клипа
			if gx+g.w > clip.X && gx < clip.X+clip.W && gy+g.h > clip.Y && gy < clip.Y+clip.H {
				startY := maxInt(0, clip.Y-gy)
				endY := minInt(g.h, clip.Y+clip.H-gy)
				startX := maxInt(0, clip.X-gx)
				endX := minInt(g.w, clip.X+clip.W-gx)

				for py := startY; py < endY; py++ {
					rowOff := py * g.w
					dstY := gy + py
					for px := startX; px < endX; px++ {
						cov := g.alpha[rowOff+px]
						if cov > 0 {
							c.blendPixelUnchecked(gx+px, dstY, col, cov)
						}
					}
				}
			}
		}
		penX += g.advance
	}
}
