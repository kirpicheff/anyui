package render

import (
	"image"
	"os"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const (
	defaultAlphaGlyphCacheCap    = 2048
	defaultColoredGlyphCacheCap  = 4096
	defaultColoredGlyphCacheByte = 8 << 20
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

type coloredGlyphKey struct {
	style FontStyle
	r     rune
	size  int
	color Color
}

type cachedGlyph struct {
	w, h     int
	offX     int
	offY     int
	advance  int
	alpha    []byte
	notEmpty bool
}

type coloredGlyph struct {
	w, h     int
	offX     int
	offY     int
	advance  int
	bgra     []byte
	notEmpty bool
}

type faceEntry struct {
	face       font.Face
	ascent     int
	lineHeight int
}

type fontSet struct {
	mu       sync.Mutex
	sfntFont *sfnt.Font
	faces    map[int]*faceEntry
	glyphs   *lruCache[glyphKey, *cachedGlyph]
}

func newFontSet(f *sfnt.Font) *fontSet {
	return &fontSet{
		sfntFont: f,
		faces:    make(map[int]*faceEntry),
		glyphs:   newLRUCache[glyphKey, *cachedGlyph](defaultAlphaGlyphCacheCap, 0),
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
	fs.mu.Lock()
	defer fs.mu.Unlock()

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
	if g, ok := fs.glyphs.Get(key); ok {
		return g
	}

	fe := fs.getFace(sizePx)
	if fe == nil {
		return nil
	}

	fs.mu.Lock()
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
	fs.mu.Unlock()

	fs.glyphs.Put(key, g, int64(len(g.alpha)))
	return g
}

// FontManager управляет начертаниями шрифтов (Regular, Bold, Mono), fallback-цепочкой и LRU-кешем глифов.
type FontManager struct {
	styles        map[FontStyle]*fontSet
	fallbacks     []*fontSet
	coloredGlyphs *lruCache[coloredGlyphKey, *coloredGlyph]
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
		coloredGlyphs: newLRUCache[coloredGlyphKey, *coloredGlyph](defaultColoredGlyphCacheCap, defaultColoredGlyphCacheByte),
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

func (fm *FontManager) ensureColoredCache() *lruCache[coloredGlyphKey, *coloredGlyph] {
	if fm.coloredGlyphs == nil {
		fm.coloredGlyphs = newLRUCache[coloredGlyphKey, *coloredGlyph](defaultColoredGlyphCacheCap, defaultColoredGlyphCacheByte)
	}
	return fm.coloredGlyphs
}

// SetGlyphCacheCapacity задаёт максимальное число записей в LRU-кеше цветных и базовых глифов.
func (fm *FontManager) SetGlyphCacheCapacity(maxEntries int) {
	if fm == nil {
		return
	}
	if maxEntries < 1 {
		maxEntries = 1
	}
	fm.ensureColoredCache().SetLimits(maxEntries, defaultColoredGlyphCacheByte)
	for _, fs := range fm.styles {
		if fs != nil && fs.glyphs != nil {
			fs.glyphs.SetLimits(maxEntries, 0)
		}
	}
	for _, fb := range fm.fallbacks {
		if fb != nil && fb.glyphs != nil {
			fb.glyphs.SetLimits(maxEntries, 0)
		}
	}
}

// GlyphCacheStats возвращает число закешированных цветных глифов и занимаемую ими память в байтах.
func (fm *FontManager) GlyphCacheStats() (entries int, bytes int64) {
	if fm == nil || fm.coloredGlyphs == nil {
		return 0, 0
	}
	return fm.coloredGlyphs.Stats()
}

// ClearGlyphCache сбрасывает кеш растеризованных глифов.
func (fm *FontManager) ClearGlyphCache() {
	if fm == nil {
		return
	}
	if fm.coloredGlyphs != nil {
		fm.coloredGlyphs.Clear()
	}
	for _, fs := range fm.styles {
		if fs != nil && fs.glyphs != nil {
			fs.glyphs.Clear()
		}
	}
	for _, fb := range fm.fallbacks {
		if fb != nil && fb.glyphs != nil {
			fb.glyphs.Clear()
		}
	}
}

func (fm *FontManager) styleSet(style FontStyle) *fontSet {
	if fs, ok := fm.styles[style]; ok && fs != nil {
		return fs
	}
	return fm.styles[FontRegular]
}

func (fm *FontManager) resolveGlyph(style FontStyle, r rune, sizePx int) *cachedGlyph {
	if sizePx < 6 {
		sizePx = 6
	}
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

func (fm *FontManager) resolveColoredGlyph(style FontStyle, r rune, sizePx int, col Color) *coloredGlyph {
	if sizePx < 6 {
		sizePx = 6
	}
	cache := fm.ensureColoredCache()
	key := coloredGlyphKey{
		style: style,
		r:     r,
		size:  sizePx,
		color: col,
	}
	if cg, ok := cache.Get(key); ok {
		return cg
	}

	g := fm.resolveGlyph(style, r, sizePx)
	if g == nil {
		return nil
	}

	cg := &coloredGlyph{
		w:        g.w,
		h:        g.h,
		offX:     g.offX,
		offY:     g.offY,
		advance:  g.advance,
		notEmpty: g.notEmpty,
	}
	if g.notEmpty && len(g.alpha) == g.w*g.h {
		bgra := make([]byte, g.w*g.h*4)
		for i, cov := range g.alpha {
			if cov == 0 {
				continue
			}
			a := (uint32(col.A)*uint32(cov) + 127) / 255
			if a == 0 {
				continue
			}
			off := i * 4
			if a >= 255 {
				bgra[off+0] = col.B
				bgra[off+1] = col.G
				bgra[off+2] = col.R
				bgra[off+3] = 255
			} else {
				bgra[off+0] = uint8((uint32(col.B)*a + 127) / 255)
				bgra[off+1] = uint8((uint32(col.G)*a + 127) / 255)
				bgra[off+2] = uint8((uint32(col.R)*a + 127) / 255)
				bgra[off+3] = uint8(a)
			}
		}
		cg.bgra = bgra
	}

	cache.Put(key, cg, int64(len(cg.bgra)))
	return cg
}

// MeasureText возвращает ширину и высоту строки в пикселях для начертания Regular.
func (fm *FontManager) MeasureText(text string, sizePx int) (int, int) {
	return fm.MeasureTextStyled(text, sizePx, FontRegular)
}

// MeasureTextStyled возвращает ширину и высоту строки в пикселях для указанного начертания.
func (fm *FontManager) MeasureTextStyled(text string, sizePx int, style FontStyle) (int, int) {
	if fm == nil {
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
	if text == "" {
		return 0, fe.lineHeight
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
	if clip.Empty() || y >= clip.Y+clip.H || y+fe.lineHeight <= clip.Y {
		return
	}

	penX := x
	baselineY := y + fe.ascent
	var dirty Rect

	for _, r := range text {
		cg := fm.resolveColoredGlyph(style, r, sizePx, col)
		if cg == nil {
			continue
		}
		if cg.notEmpty && len(cg.bgra) == cg.w*cg.h*4 {
			gx := penX + cg.offX
			gy := baselineY + cg.offY

			// Быстрая проверка видимости глифа относительно клипа
			if gx+cg.w > clip.X && gx < clip.X+clip.W && gy+cg.h > clip.Y && gy < clip.Y+clip.H {
				startY := maxInt(0, clip.Y-gy)
				endY := minInt(cg.h, clip.Y+clip.H-gy)
				startX := maxInt(0, clip.X-gx)
				endX := minInt(cg.w, clip.X+clip.W-gx)
				dirty = dirty.Union(Rect{
					X: gx + startX,
					Y: gy + startY,
					W: endX - startX,
					H: endY - startY,
				})

				for py := startY; py < endY; py++ {
					srcRow := (py*cg.w + startX) * 4
					dstRow := ((gy+py)*c.Width + (gx + startX)) * 4
					spanW := endX - startX
					for px := 0; px < spanW; px++ {
						so := srcRow + px*4
						a := cg.bgra[so+3]
						if a == 0 {
							continue
						}
						do := dstRow + px*4
						if a == 255 {
							c.Pixels[do+0] = cg.bgra[so+0]
							c.Pixels[do+1] = cg.bgra[so+1]
							c.Pixels[do+2] = cg.bgra[so+2]
							c.Pixels[do+3] = 255
							continue
						}
						invA := uint32(255 - a)
						b := uint32(cg.bgra[so+0]) + (uint32(c.Pixels[do+0])*invA+127)/255
						g := uint32(cg.bgra[so+1]) + (uint32(c.Pixels[do+1])*invA+127)/255
						r := uint32(cg.bgra[so+2]) + (uint32(c.Pixels[do+2])*invA+127)/255
						if b > 255 {
							b = 255
						}
						if g > 255 {
							g = 255
						}
						if r > 255 {
							r = 255
						}
						c.Pixels[do+0] = uint8(b)
						c.Pixels[do+1] = uint8(g)
						c.Pixels[do+2] = uint8(r)
						c.Pixels[do+3] = 255
					}
				}
			}
		}
		penX += cg.advance
	}

	if !dirty.Empty() {
		c.AddDirtyRect(dirty)
	}
}
