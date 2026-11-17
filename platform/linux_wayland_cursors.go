//go:build linux

package platform

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type wlCursorBitmap struct {
	buf  *wlShmBuffer
	hotX int32
	hotY int32
}

type wlCursorCache struct {
	disp      *wlDisplay
	surfaceID uint32

	mu      sync.Mutex
	cursors map[CursorType]*wlCursorBitmap
}

func newWLCursorCache(d *wlDisplay) *wlCursorCache {
	surfID := d.conn.allocID(nil)
	var cw wlWriter
	cw.uint32(surfID)
	if err := d.conn.send(d.compositorID, wlCompositorCreateSurface, cw.bytes()); err != nil {
		d.conn.removeHandler(surfID)
		return nil
	}
	return &wlCursorCache{
		disp:      d,
		surfaceID: surfID,
		cursors:   make(map[CursorType]*wlCursorBitmap),
	}
}

func (cc *wlCursorCache) setCursor(pointerID, serial uint32, cursor CursorType) {
	if cc == nil || pointerID == 0 || serial == 0 {
		return
	}
	bmp := cc.getOrCreate(cursor)
	if bmp == nil || bmp.buf == nil {
		return
	}

	var aw wlWriter
	aw.uint32(bmp.buf.bufferID)
	aw.int32(0)
	aw.int32(0)
	_ = cc.disp.conn.send(cc.surfaceID, wlSurfaceAttach, aw.bytes())

	if cc.disp.compositorVer >= 4 {
		var dw wlWriter
		dw.int32(0)
		dw.int32(0)
		dw.int32(int32(bmp.buf.width))
		dw.int32(int32(bmp.buf.height))
		_ = cc.disp.conn.send(cc.surfaceID, wlSurfaceDamageBuffer, dw.bytes())
	} else {
		var dw wlWriter
		dw.int32(0)
		dw.int32(0)
		dw.int32(int32(bmp.buf.width))
		dw.int32(int32(bmp.buf.height))
		_ = cc.disp.conn.send(cc.surfaceID, wlSurfaceDamage, dw.bytes())
	}
	_ = cc.disp.conn.send(cc.surfaceID, wlSurfaceCommit, nil)

	var pw wlWriter
	pw.uint32(serial)
	pw.uint32(cc.surfaceID)
	pw.int32(bmp.hotX)
	pw.int32(bmp.hotY)
	_ = cc.disp.conn.send(pointerID, wlPointerSetCursor, pw.bytes())
}

func (cc *wlCursorCache) getOrCreate(cursor CursorType) *wlCursorBitmap {
	cc.mu.Lock()
	if bmp, ok := cc.cursors[cursor]; ok {
		cc.mu.Unlock()
		return bmp
	}
	cc.mu.Unlock()

	targetSize := resolveXCursorSize()
	var w, h int
	var hotX, hotY int32
	var pixels []byte

	if xPix, xw, xh, xhx, xhy, ok := loadSystemXCursor(cursor, targetSize); ok {
		pixels, w, h, hotX, hotY = xPix, xw, xh, xhx, xhy
	} else {
		w, h = 24, 24
		pixels, hotX, hotY = renderFallbackCursorBitmap(cursor, 24)
	}

	buf, err := createWLShmBuffer(cc.disp, w, h)
	if err != nil {
		return nil
	}
	copy(buf.data, pixels)

	bmp := &wlCursorBitmap{
		buf:  buf,
		hotX: hotX,
		hotY: hotY,
	}
	cc.mu.Lock()
	cc.cursors[cursor] = bmp
	cc.mu.Unlock()
	return bmp
}

func resolveXCursorSize() int {
	if raw := strings.TrimSpace(os.Getenv("XCURSOR_SIZE")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 16 && n <= 128 {
			return n
		}
	}
	return 24
}

func xcursorNames(cursor CursorType) []string {
	switch cursor {
	case CursorHand:
		return []string{"hand2", "pointer", "hand1"}
	case CursorIBeam:
		return []string{"xterm", "text", "ibeam"}
	case CursorResizeH:
		return []string{"ew-resize", "sb_h_double_arrow", "h_double_arrow", "col-resize"}
	case CursorResizeV:
		return []string{"ns-resize", "sb_v_double_arrow", "v_double_arrow", "row-resize"}
	case CursorResizeNWSE:
		return []string{"nwse-resize", "bd_double_arrow", "size_fdiag"}
	case CursorResizeNESW:
		return []string{"nesw-resize", "fd_double_arrow", "size_bdiag"}
	case CursorNotAllowed:
		return []string{"not-allowed", "crossed_circle"}
	case CursorCrosshair:
		return []string{"crosshair", "cross"}
	case CursorMove:
		return []string{"fleur", "move", "all-scroll"}
	default:
		return []string{"left_ptr", "default", "arrow"}
	}
}

func loadSystemXCursor(cursor CursorType, targetSize int) (pixels []byte, w, h int, hotX, hotY int32, ok bool) {
	dirs := xcursorSearchDirs()
	names := xcursorNames(cursor)

	for _, dir := range dirs {
		for _, name := range names {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			if pix, cw, ch, hx, hy, parsed := parseXCursorFile(data, targetSize); parsed {
				return pix, cw, ch, hx, hy, true
			}
		}
	}
	return nil, 0, 0, 0, 0, false
}

func xcursorSearchDirs() []string {
	var bases []string
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		bases = append(bases, filepath.Join(home, ".icons"), filepath.Join(home, ".local/share/icons"))
	}
	bases = append(bases, "/usr/share/icons", "/usr/share/pixmaps")

	var themes []string
	seen := make(map[string]bool)
	addTheme := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		themes = append(themes, name)
	}

	if env := os.Getenv("XCURSOR_THEME"); env != "" {
		addTheme(env)
	}
	// Читаем Inherits= из default/index.theme, если тема задана через него
	for _, base := range bases {
		for _, inherited := range readXCursorInherits(filepath.Join(base, "default", "index.theme")) {
			addTheme(inherited)
		}
	}
	for _, fallback := range []string{"default", "Adwaita", "breeze_cursors", "DMZ-White", "DMZ-Black"} {
		addTheme(fallback)
	}

	var dirs []string
	for _, base := range bases {
		for _, theme := range themes {
			p := filepath.Join(base, theme, "cursors")
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				dirs = append(dirs, p)
			}
		}
	}
	return dirs
}

func readXCursorInherits(indexPath string) []string {
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Inherits=") {
			val := strings.TrimPrefix(line, "Inherits=")
			for _, part := range strings.Split(val, ",") {
				if p := strings.TrimSpace(part); p != "" {
					out = append(out, p)
				}
			}
		}
	}
	return out
}

// parseXCursorFile декодирует бинарный файл Xcursor, выбирает кадр ближайшего размера
// и гарантирует premultiplied ARGB8888 для Wayland wl_shm.
func parseXCursorFile(data []byte, targetSize int) (pixels []byte, w, h int, hotX, hotY int32, ok bool) {
	if len(data) < 16 || string(data[0:4]) != "Xcur" {
		return nil, 0, 0, 0, 0, false
	}
	headerLen := int(binary.LittleEndian.Uint32(data[4:8]))
	ntoc := int(binary.LittleEndian.Uint32(data[12:16]))
	if headerLen < 16 || ntoc <= 0 || headerLen+ntoc*12 > len(data) {
		return nil, 0, 0, 0, 0, false
	}

	const xcurImageType = 0xFFFD0002
	bestPos := -1
	bestDiff := 1 << 30
	for i := 0; i < ntoc; i++ {
		off := headerLen + i*12
		typ := binary.LittleEndian.Uint32(data[off : off+4])
		subtype := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		pos := int(binary.LittleEndian.Uint32(data[off+8 : off+12]))
		if typ != xcurImageType {
			continue
		}
		diff := subtype - targetSize
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			bestPos = pos
		}
	}
	if bestPos < 0 || bestPos+36 > len(data) {
		return nil, 0, 0, 0, 0, false
	}

	cw := int(binary.LittleEndian.Uint32(data[bestPos+16 : bestPos+20]))
	ch := int(binary.LittleEndian.Uint32(data[bestPos+20 : bestPos+24]))
	hx := int32(binary.LittleEndian.Uint32(data[bestPos+24 : bestPos+28]))
	hy := int32(binary.LittleEndian.Uint32(data[bestPos+28 : bestPos+32]))
	pixOff := bestPos + 36
	pixLen := cw * ch * 4
	if cw <= 0 || ch <= 0 || cw > 256 || ch > 256 || pixOff+pixLen > len(data) {
		return nil, 0, 0, 0, 0, false
	}

	out := make([]byte, pixLen)
	copy(out, data[pixOff:pixOff+pixLen])
	// Нормализуем premultiplied alpha: каналы B, G, R не должны превышать A.
	for i := 0; i+3 < len(out); i += 4 {
		a := out[i+3]
		if out[i] > a {
			out[i] = a
		}
		if out[i+1] > a {
			out[i+1] = a
		}
		if out[i+2] > a {
			out[i+2] = a
		}
	}
	return out, cw, ch, hx, hy, true
}

func renderFallbackCursorBitmap(cursor CursorType, size int) ([]byte, int32, int32) {
	pix := make([]byte, size*size*4)
	setPx := func(x, y int, r, g, b, a uint8) {
		if x < 0 || y < 0 || x >= size || y >= size {
			return
		}
		off := (y*size + x) * 4
		pix[off] = uint8(int(b) * int(a) / 255)
		pix[off+1] = uint8(int(g) * int(a) / 255)
		pix[off+2] = uint8(int(r) * int(a) / 255)
		pix[off+3] = a
	}

	drawLine := func(x0, y0, x1, y1 int, r, g, b uint8) {
		dx := x1 - x0
		if dx < 0 {
			dx = -dx
		}
		dy := y1 - y0
		if dy < 0 {
			dy = -dy
		}
		sx, sy := 1, 1
		if x0 > x1 {
			sx = -1
		}
		if y0 > y1 {
			sy = -1
		}
		err := dx - dy
		for {
			setPx(x0, y0, r, g, b, 255)
			if x0 == x1 && y0 == y1 {
				break
			}
			e2 := 2 * err
			if e2 > -dy {
				err -= dy
				x0 += sx
			}
			if e2 < dx {
				err += dx
				y0 += sy
			}
		}
	}

	stroke := func(x0, y0, x1, y1 int) {
		drawLine(x0-1, y0, x1-1, y1, 20, 20, 20)
		drawLine(x0+1, y0, x1+1, y1, 20, 20, 20)
		drawLine(x0, y0-1, x1, y1-1, 20, 20, 20)
		drawLine(x0, y0+1, x1, y1+1, 20, 20, 20)
		drawLine(x0, y0, x1, y1, 255, 255, 255)
	}

	c := size / 2
	switch cursor {
	case CursorIBeam:
		stroke(c, 4, c, size-5)
		stroke(c-3, 4, c+3, 4)
		stroke(c-3, size-5, c+3, size-5)
		return pix, int32(c), int32(c)

	case CursorHand:
		stroke(10, 3, 10, 13)
		stroke(12, 3, 12, 12)
		stroke(10, 3, 12, 3)
		stroke(12, 8, 17, 10)
		stroke(17, 10, 17, 17)
		stroke(10, 13, 7, 11)
		stroke(7, 11, 7, 15)
		stroke(7, 15, 10, 19)
		stroke(10, 19, 17, 17)
		return pix, 11, 3

	case CursorResizeH:
		stroke(4, c, size-5, c)
		stroke(4, c, 8, c-4)
		stroke(4, c, 8, c+4)
		stroke(size-5, c, size-9, c-4)
		stroke(size-5, c, size-9, c+4)
		return pix, int32(c), int32(c)

	case CursorResizeV:
		stroke(c, 4, c, size-5)
		stroke(c, 4, c-4, 8)
		stroke(c, 4, c+4, 8)
		stroke(c, size-5, c-4, size-9)
		stroke(c, size-5, c+4, size-9)
		return pix, int32(c), int32(c)

	case CursorResizeNWSE:
		stroke(5, 5, size-6, size-6)
		stroke(5, 5, 10, 5)
		stroke(5, 5, 5, 10)
		stroke(size-6, size-6, size-11, size-6)
		stroke(size-6, size-6, size-6, size-11)
		return pix, int32(c), int32(c)

	case CursorResizeNESW:
		stroke(size-6, 5, 5, size-6)
		stroke(size-6, 5, size-11, 5)
		stroke(size-6, 5, size-6, 10)
		stroke(5, size-6, 10, size-6)
		stroke(5, size-6, 5, size-11)
		return pix, int32(c), int32(c)

	case CursorCrosshair:
		stroke(c, 4, c, size-5)
		stroke(4, c, size-5, c)
		return pix, int32(c), int32(c)

	case CursorMove:
		stroke(c, 3, c, size-4)
		stroke(3, c, size-4, c)
		stroke(c, 3, c-3, 6)
		stroke(c, 3, c+3, 6)
		stroke(c, size-4, c-3, size-7)
		stroke(c, size-4, c+3, size-7)
		stroke(3, c, 6, c-3)
		stroke(3, c, 6, c+3)
		stroke(size-4, c, size-7, c-3)
		stroke(size-4, c, size-7, c+3)
		return pix, int32(c), int32(c)

	case CursorNotAllowed:
		r := 7
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				d := x*x + y*y
				if d >= (r-1)*(r-1) && d <= r*r {
					setPx(c+x, c+y, 230, 70, 70, 255)
				}
			}
		}
		drawLine(c-4, c-4, c+4, c+4, 230, 70, 70)
		return pix, int32(c), int32(c)

	default:
		for y := 0; y < 14; y++ {
			for x := 0; x <= y && x < 10; x++ {
				if x == 0 || y == 13 || x == y {
					setPx(2+x, 2+y, 20, 20, 20, 255)
				} else {
					setPx(2+x, 2+y, 255, 255, 255, 255)
				}
			}
		}
		return pix, 2, 2
	}
}
