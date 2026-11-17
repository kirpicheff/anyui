package render

import (
	"encoding/xml"
	"hash/fnv"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type svgCacheKey struct {
	hash   uint64
	sizePx int
}

var (
	svgCacheMu sync.RWMutex
	svgCache   = make(map[svgCacheKey][]byte)
)

type svgShape struct {
	contours    [][]PointF
	closed      bool
	hasFill     bool
	hasStroke   bool
	strokeWidth float32
}

// DrawSVG парсит SVG-строку, кеширует её альфа-маску для размера sizePx и рисует с покраской tint.
func (c *Canvas) DrawSVG(svgData string, x, y, sizePx int, tint Color) {
	if c == nil || svgData == "" || sizePx <= 0 || tint.A == 0 {
		return
	}

	mask := getOrRasterizeSVG(svgData, sizePx)
	if len(mask) != sizePx*sizePx {
		return
	}

	clip := c.CurrentClip()
	rect := Rect{X: x, Y: y, W: sizePx, H: sizePx}.Intersect(clip)
	if rect.Empty() {
		return
	}
	c.AddDirtyRect(rect)

	for py := rect.Y; py < rect.Y+rect.H; py++ {
		sy := py - y
		for px := rect.X; px < rect.X+rect.W; px++ {
			sx := px - x
			a := mask[sy*sizePx+sx]
			if a > 0 {
				c.blendPixelUnchecked(px, py, tint, a)
			}
		}
	}
}

func getOrRasterizeSVG(svgData string, sizePx int) []byte {
	h := fnv.New64a()
	_, _ = h.Write([]byte(svgData))
	key := svgCacheKey{hash: h.Sum64(), sizePx: sizePx}

	svgCacheMu.RLock()
	if m, ok := svgCache[key]; ok {
		svgCacheMu.RUnlock()
		return m
	}
	svgCacheMu.RUnlock()

	mask := rasterizeSVG(svgData, sizePx)

	svgCacheMu.Lock()
	svgCache[key] = mask
	svgCacheMu.Unlock()

	return mask
}

func rasterizeSVG(svgData string, sizePx int) []byte {
	vbX, vbY, vbW, vbH, shapes := parseSVG(svgData)
	mask := make([]byte, sizePx*sizePx)
	if len(shapes) == 0 || vbW <= 0 || vbH <= 0 {
		return mask
	}

	scaleX := float32(sizePx) / vbW
	scaleY := float32(sizePx) / vbH
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	offX := (float32(sizePx) - vbW*scale) * 0.5
	offY := (float32(sizePx) - vbH*scale) * 0.5

	// Переводим все координаты фигур в пространство пикселей [0..sizePx]
	for i := range shapes {
		for j := range shapes[i].contours {
			for k := range shapes[i].contours[j] {
				p := shapes[i].contours[j][k]
				shapes[i].contours[j][k] = PointF{
					X: (p.X-vbX)*scale + offX,
					Y: (p.Y-vbY)*scale + offY,
				}
			}
		}
		if shapes[i].hasStroke {
			shapes[i].strokeWidth *= scale
			if shapes[i].strokeWidth < 1 {
				shapes[i].strokeWidth = 1
			}
		}
	}

	// Растеризуем каждую фигуру с 4x4 SSAA для заливки и SDF для обводки
	const ss = 4
	const invSS = 1.0 / float32(ss)
	for _, sh := range shapes {
		if sh.hasFill && len(sh.contours) > 0 {
			for y := 0; y < sizePx; y++ {
				for x := 0; x < sizePx; x++ {
					hits := 0
					for sy := 0; sy < ss; sy++ {
						py := float32(y) + (float32(sy)+0.5)*invSS
						for sx := 0; sx < ss; sx++ {
							px := float32(x) + (float32(sx)+0.5)*invSS
							if pointInContours(px, py, sh.contours) {
								hits++
							}
						}
					}
					if hits > 0 {
						a := uint8(hits * 255 / (ss * ss))
						idx := y*sizePx + x
						if a > mask[idx] {
							mask[idx] = a
						}
					}
				}
			}
		}

		if sh.hasStroke && len(sh.contours) > 0 {
			halfW := sh.strokeWidth * 0.5
			for y := 0; y < sizePx; y++ {
				py := float32(y) + 0.5
				for x := 0; x < sizePx; x++ {
					px := float32(x) + 0.5
					minD := float32(math.MaxFloat32)
					for _, pts := range sh.contours {
						if len(pts) < 2 {
							continue
						}
						segCount := len(pts) - 1
						if sh.closed && len(pts) > 2 {
							segCount = len(pts)
						}
						for i := 0; i < segCount; i++ {
							d := distToSegment(px, py, pts[i].X, pts[i].Y, pts[(i+1)%len(pts)].X, pts[(i+1)%len(pts)].Y)
							if d < minD {
								minD = d
							}
						}
					}
					cov := clamp01(0.5 - (minD - halfW))
					if cov > 0 {
						a := uint8(cov*255 + 0.5)
						idx := y*sizePx + x
						if a > mask[idx] {
							mask[idx] = a
						}
					}
				}
			}
		}
	}

	return mask
}

func pointInContours(px, py float32, contours [][]PointF) bool {
	inside := false
	for _, pts := range contours {
		n := len(pts)
		if n < 3 {
			continue
		}
		j := n - 1
		for i := 0; i < n; i++ {
			yi := pts[i].Y
			yj := pts[j].Y
			if (yi > py) != (yj > py) {
				xIntersect := (pts[j].X-pts[i].X)*(py-yi)/(yj-yi) + pts[i].X
				if px < xIntersect {
					inside = !inside
				}
			}
			j = i
		}
	}
	return inside
}

func parseSVG(svgData string) (vbX, vbY, vbW, vbH float32, shapes []svgShape) {
	vbW, vbH = 24, 24
	dec := xml.NewDecoder(strings.NewReader(svgData))

	type styleState struct {
		fill        string
		stroke      string
		strokeWidth float32
	}
	stack := []styleState{{fill: "currentColor", stroke: "none", strokeWidth: 2}}

	for {
		tok, err := dec.Token()
		if err == io.EOF || err != nil {
			break
		}
		switch el := tok.(type) {
		case xml.StartElement:
			cur := stack[len(stack)-1]
			attrs := make(map[string]string, len(el.Attr))
			for _, a := range el.Attr {
				attrs[a.Name.Local] = strings.TrimSpace(a.Value)
			}
			if v, ok := attrs["fill"]; ok {
				cur.fill = v
			}
			if v, ok := attrs["stroke"]; ok {
				cur.stroke = v
			}
			if v, ok := attrs["stroke-width"]; ok {
				if f, err := strconv.ParseFloat(strings.TrimSuffix(v, "px"), 32); err == nil && f > 0 {
					cur.strokeWidth = float32(f)
				}
			}
			stack = append(stack, cur)

			hasFill := cur.fill != "none" && cur.fill != "transparent" && cur.fill != ""
			hasStroke := cur.stroke != "none" && cur.stroke != "transparent" && cur.stroke != ""

			switch el.Name.Local {
			case "svg":
				if vb, ok := attrs["viewBox"]; ok {
					nums := parseFloatList(vb)
					if len(nums) >= 4 {
						vbX, vbY, vbW, vbH = nums[0], nums[1], nums[2], nums[3]
					}
				} else {
					if w, ok := attrs["width"]; ok {
						if f, err := strconv.ParseFloat(strings.TrimSuffix(w, "px"), 32); err == nil && f > 0 {
							vbW = float32(f)
						}
					}
					if h, ok := attrs["height"]; ok {
						if f, err := strconv.ParseFloat(strings.TrimSuffix(h, "px"), 32); err == nil && f > 0 {
							vbH = float32(f)
						}
					}
				}
			case "path":
				if d, ok := attrs["d"]; ok {
					contours, closed := parseSVGPath(d)
					if len(contours) > 0 {
						shapes = append(shapes, svgShape{
							contours:    contours,
							closed:      closed,
							hasFill:     hasFill,
							hasStroke:   hasStroke,
							strokeWidth: cur.strokeWidth,
						})
					}
				}
			case "rect":
				rx := parseAttrFloat(attrs["x"], 0)
				ry := parseAttrFloat(attrs["y"], 0)
				rw := parseAttrFloat(attrs["width"], 0)
				rh := parseAttrFloat(attrs["height"], 0)
				if rw > 0 && rh > 0 {
					pts := []PointF{
						{rx, ry}, {rx + rw, ry}, {rx + rw, ry + rh}, {rx, ry + rh},
					}
					shapes = append(shapes, svgShape{
						contours:    [][]PointF{pts},
						closed:      true,
						hasFill:     hasFill,
						hasStroke:   hasStroke,
						strokeWidth: cur.strokeWidth,
					})
				}
			case "circle":
				cx := parseAttrFloat(attrs["cx"], 0)
				cy := parseAttrFloat(attrs["cy"], 0)
				r := parseAttrFloat(attrs["r"], 0)
				if r > 0 {
					const steps = 24
					pts := make([]PointF, 0, steps)
					for i := 0; i < steps; i++ {
						ang := 2 * math.Pi * float64(i) / float64(steps)
						pts = append(pts, PointF{
							X: cx + float32(math.Cos(ang))*r,
							Y: cy + float32(math.Sin(ang))*r,
						})
					}
					shapes = append(shapes, svgShape{
						contours:    [][]PointF{pts},
						closed:      true,
						hasFill:     hasFill,
						hasStroke:   hasStroke,
						strokeWidth: cur.strokeWidth,
					})
				}
			case "line":
				x1 := parseAttrFloat(attrs["x1"], 0)
				y1 := parseAttrFloat(attrs["y1"], 0)
				x2 := parseAttrFloat(attrs["x2"], 0)
				y2 := parseAttrFloat(attrs["y2"], 0)
				shapes = append(shapes, svgShape{
					contours:    [][]PointF{{{x1, y1}, {x2, y2}}},
					closed:      false,
					hasFill:     false,
					hasStroke:   hasStroke,
					strokeWidth: cur.strokeWidth,
				})
			case "polyline", "polygon":
				nums := parseFloatList(attrs["points"])
				if len(nums) >= 4 {
					pts := make([]PointF, 0, len(nums)/2)
					for i := 0; i+1 < len(nums); i += 2 {
						pts = append(pts, PointF{X: nums[i], Y: nums[i+1]})
					}
					closed := el.Name.Local == "polygon"
					shapes = append(shapes, svgShape{
						contours:    [][]PointF{pts},
						closed:      closed,
						hasFill:     closed && hasFill,
						hasStroke:   hasStroke,
						strokeWidth: cur.strokeWidth,
					})
				}
			}
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return
}

func parseAttrFloat(s string, def float32) float32 {
	if s == "" {
		return def
	}
	s = strings.TrimSuffix(s, "px")
	f, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return def
	}
	return float32(f)
}

func parseFloatList(s string) []float32 {
	var res []float32
	i := 0
	n := len(s)
	for i < n {
		for i < n && (s[i] == ' ' || s[i] == ',' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
		if i >= n {
			break
		}
		start := i
		if s[i] == '+' || s[i] == '-' {
			i++
		}
		hasDot := false
		for i < n && (unicode.IsDigit(rune(s[i])) || (s[i] == '.' && !hasDot)) {
			if s[i] == '.' {
				hasDot = true
			}
			i++
		}
		if i > start {
			if f, err := strconv.ParseFloat(s[start:i], 32); err == nil {
				res = append(res, float32(f))
			}
		} else {
			i++
		}
	}
	return res
}

func parseSVGPath(d string) ([][]PointF, bool) {
	var contours [][]PointF
	var cur []PointF
	anyClosed := false

	var cx, cy float32
	var startX, startY float32
	var lastCtrlX, lastCtrlY float32
	var lastCmd byte

	i := 0
	n := len(d)

	skipSep := func() {
		for i < n && (d[i] == ' ' || d[i] == ',' || d[i] == '\t' || d[i] == '\n' || d[i] == '\r') {
			i++
		}
	}
	nextNum := func() (float32, bool) {
		skipSep()
		if i >= n {
			return 0, false
		}
		start := i
		if d[i] == '+' || d[i] == '-' {
			i++
		}
		hasDot := false
		for i < n && ((d[i] >= '0' && d[i] <= '9') || (d[i] == '.' && !hasDot)) {
			if d[i] == '.' {
				hasDot = true
			}
			i++
		}
		if i == start {
			return 0, false
		}
		f, err := strconv.ParseFloat(d[start:i], 32)
		if err != nil {
			return 0, false
		}
		return float32(f), true
	}

	flushContour := func() {
		if len(cur) > 0 {
			contours = append(contours, cur)
			cur = nil
		}
	}

	var cmd byte
	for {
		skipSep()
		if i >= n {
			break
		}
		ch := d[i]
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			cmd = ch
			i++
		} else if cmd == 0 {
			break
		}

		switch cmd {
		case 'M', 'm':
			x, ok1 := nextNum()
			y, ok2 := nextNum()
			if !ok1 || !ok2 {
				break
			}
			flushContour()
			if cmd == 'm' {
				x += cx
				y += cy
				cmd = 'l'
			} else {
				cmd = 'L'
			}
			cx, cy = x, y
			startX, startY = x, y
			cur = append(cur, PointF{cx, cy})
			lastCmd = 'M'
		case 'L', 'l':
			x, ok1 := nextNum()
			y, ok2 := nextNum()
			if !ok1 || !ok2 {
				break
			}
			if cmd == 'l' {
				x += cx
				y += cy
			}
			cx, cy = x, y
			cur = append(cur, PointF{cx, cy})
			lastCmd = 'L'
		case 'H', 'h':
			x, ok := nextNum()
			if !ok {
				break
			}
			if cmd == 'h' {
				x += cx
			}
			cx = x
			cur = append(cur, PointF{cx, cy})
			lastCmd = 'H'
		case 'V', 'v':
			y, ok := nextNum()
			if !ok {
				break
			}
			if cmd == 'v' {
				y += cy
			}
			cy = y
			cur = append(cur, PointF{cx, cy})
			lastCmd = 'V'
		case 'C', 'c':
			x1, ok1 := nextNum()
			y1, ok2 := nextNum()
			x2, ok3 := nextNum()
			y2, ok4 := nextNum()
			x, ok5 := nextNum()
			y, ok6 := nextNum()
			if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 {
				break
			}
			if cmd == 'c' {
				x1 += cx
				y1 += cy
				x2 += cx
				y2 += cy
				x += cx
				y += cy
			}
			appendCubic(&cur, cx, cy, x1, y1, x2, y2, x, y)
			lastCtrlX, lastCtrlY = x2, y2
			cx, cy = x, y
			lastCmd = 'C'
		case 'S', 's':
			x2, ok1 := nextNum()
			y2, ok2 := nextNum()
			x, ok3 := nextNum()
			y, ok4 := nextNum()
			if !ok1 || !ok2 || !ok3 || !ok4 {
				break
			}
			if cmd == 's' {
				x2 += cx
				y2 += cy
				x += cx
				y += cy
			}
			x1, y1 := cx, cy
			if lastCmd == 'C' || lastCmd == 'S' {
				x1 = 2*cx - lastCtrlX
				y1 = 2*cy - lastCtrlY
			}
			appendCubic(&cur, cx, cy, x1, y1, x2, y2, x, y)
			lastCtrlX, lastCtrlY = x2, y2
			cx, cy = x, y
			lastCmd = 'S'
		case 'Q', 'q':
			x1, ok1 := nextNum()
			y1, ok2 := nextNum()
			x, ok3 := nextNum()
			y, ok4 := nextNum()
			if !ok1 || !ok2 || !ok3 || !ok4 {
				break
			}
			if cmd == 'q' {
				x1 += cx
				y1 += cy
				x += cx
				y += cy
			}
			appendQuad(&cur, cx, cy, x1, y1, x, y)
			lastCtrlX, lastCtrlY = x1, y1
			cx, cy = x, y
			lastCmd = 'Q'
		case 'Z', 'z':
			anyClosed = true
			cx, cy = startX, startY
			flushContour()
			lastCmd = 'Z'
			cmd = 0
		default:
			i++
		}
	}
	flushContour()
	return contours, anyClosed
}

func appendCubic(pts *[]PointF, x0, y0, x1, y1, x2, y2, x3, y3 float32) {
	const steps = 12
	for s := 1; s <= steps; s++ {
		t := float32(s) / float32(steps)
		u := 1 - t
		uu := u * u
		tt := t * t
		x := uu*u*x0 + 3*uu*t*x1 + 3*u*tt*x2 + tt*t*x3
		y := uu*u*y0 + 3*uu*t*y1 + 3*u*tt*y2 + tt*t*y3
		*pts = append(*pts, PointF{X: x, Y: y})
	}
}

func appendQuad(pts *[]PointF, x0, y0, x1, y1, x2, y2 float32) {
	const steps = 10
	for s := 1; s <= steps; s++ {
		t := float32(s) / float32(steps)
		u := 1 - t
		x := u*u*x0 + 2*u*t*x1 + t*t*x2
		y := u*u*y0 + 2*u*t*y1 + t*t*y2
		*pts = append(*pts, PointF{X: x, Y: y})
	}
}

