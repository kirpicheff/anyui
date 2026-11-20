package ui

import (
	"fmt"
	"math"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

// ChartSeries описывает одну серию данных для LineChart и BarChart.
type ChartSeries struct {
	Name   string
	Values []float64
	Color  *render.Color
}

func (c *Context) chartPalette(idx int) render.Color {
	palette := []render.Color{
		c.Theme.Accent,
		c.Theme.Success,
		c.Theme.Warning,
		c.Theme.Danger,
		render.RGB(168, 85, 247),
		render.RGB(6, 182, 212),
	}
	return palette[idx%len(palette)]
}

func calcMinMax(series []ChartSeries) (float64, float64) {
	hasVal := false
	minV, maxV := 0.0, 0.0
	for _, s := range series {
		for _, v := range s.Values {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			if !hasVal {
				minV, maxV = v, v
				hasVal = true
			} else {
				if v < minV {
					minV = v
				}
				if v > maxV {
					maxV = v
				}
			}
		}
	}
	if !hasVal {
		return 0, 1
	}
	if maxV == minV {
		return minV - 1, maxV + 1
	}
	return minV, maxV
}

func formatChartVal(v float64) string {
	if math.Abs(v) >= 1000 || (math.Abs(v) > 0 && math.Abs(v) < 0.01) {
		return fmt.Sprintf("%.1f", v)
	}
	if math.Abs(v-math.Round(v)) < 1e-6 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

// Sparkline отрисовывает компактный мини-график тренда без осей, с тултипом при наведении.
func (c *Context) Sparkline(id string, values []float64, opts ...Option) {
	id = c.ResolveID("spark", id)
	st := ApplyOptions(opts)
	hDp := Dp(44)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	bg := c.Theme.Surface
	if st.Bg != nil {
		bg = *st.Bg
	}
	c.Canvas.DrawRoundedBox(r, c.ScaleF(c.Theme.RadiusSm), bg, 1.0, c.Theme.Border)

	if len(values) < 2 {
		return
	}
	c.RegisterHitTrack(id, r, platform.CursorArrow)

	minV, maxV := calcMinMax([]ChartSeries{{Values: values}})
	padX := c.Scale(8).Int()
	padY := c.Scale(6).Int()
	plot := render.Rect{X: r.X + padX, Y: r.Y + padY, W: r.W - padX*2, H: r.H - padY*2}
	if plot.W <= 0 || plot.H <= 0 {
		return
	}

	col := c.Theme.Accent
	if st.Color != nil {
		col = *st.Color
	}

	pts := make([]render.PointF, len(values))
	for i, v := range values {
		t := float32(i) / float32(len(values)-1)
		normY := float32((v - minV) / (maxV - minV))
		pts[i] = render.PointF{
			X: float32(plot.X) + t*float32(plot.W),
			Y: float32(plot.Y+plot.H) - normY*float32(plot.H),
		}
	}

	for i := 0; i < len(pts)-1; i++ {
		c.Canvas.DrawLine(pts[i].X, pts[i].Y, pts[i+1].X, pts[i+1].Y, c.ScaleF(2.0), col)
	}

	// Подсветка ближайшей точки и тултип при наведении
	if !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY) {
		bestIdx := 0
		bestDist := float32(math.MaxFloat32)
		for i, p := range pts {
			d := float32(math.Abs(float64(float32(c.MouseX) - p.X)))
			if d < bestDist {
				bestDist = d
				bestIdx = i
			}
		}
		bp := pts[bestIdx]
		c.Canvas.FillCircle(bp.X, bp.Y, c.ScaleF(3.5), col)
		c.Tooltip(formatChartVal(values[bestIdx]), r)
	}
}

// LineChart отрисовывает линейный график с сеткой, осями, легендой и всплывающей подсказкой при наведении.
func (c *Context) LineChart(id string, labels []string, series []ChartSeries, opts ...Option) {
	id = c.ResolveID("lchart", id)
	st := ApplyOptions(opts)
	hDp := Dp(220)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	bg := c.Theme.Card
	if st.Bg != nil {
		bg = *st.Bg
	}
	c.Canvas.DrawRoundedBox(r, c.ScaleF(c.Theme.RadiusMd), bg, 1.0, c.Theme.Border)

	if len(series) == 0 {
		return
	}
	c.RegisterHitTrack(id, r, platform.CursorArrow)

	legendH := c.drawChartLegend(render.Rect{X: r.X + c.Scale(12).Int(), Y: r.Y + c.Scale(8).Int(), W: r.W - c.Scale(24).Int(), H: c.Scale(20).Int()}, series)
	padLeft := c.Scale(48).Int()
	padRight := c.Scale(16).Int()
	padTop := c.Scale(12).Int() + legendH
	padBottom := c.Scale(28).Int()

	plot := render.Rect{
		X: r.X + padLeft,
		Y: r.Y + padTop,
		W: r.W - padLeft - padRight,
		H: r.H - padTop - padBottom,
	}
	if plot.W <= 10 || plot.H <= 10 {
		return
	}

	minV, maxV := calcMinMax(series)
	// Небольшой запас сверху и снизу
	span := maxV - minV
	minV -= span * 0.05
	maxV += span * 0.05

	c.drawChartGridAndYAxis(plot, minV, maxV, 4)

	maxLen := len(labels)
	for _, s := range series {
		if len(s.Values) > maxLen {
			maxLen = len(s.Values)
		}
	}
	if maxLen == 0 {
		return
	}

	c.drawChartXAxisLabels(plot, labels, maxLen, false)

	// Рисуем серии
	type hoverCand struct {
		pt    render.PointF
		label string
		name  string
		val   float64
		col   render.Color
	}
	var best *hoverCand
	bestDistX := float32(c.Scale(18).Int())

	for si, s := range series {
		if len(s.Values) == 0 {
			continue
		}
		col := c.chartPalette(si)
		if s.Color != nil {
			col = *s.Color
		}
		pts := make([]render.PointF, len(s.Values))
		denom := float32(maxLen - 1)
		if denom <= 0 {
			denom = 1
		}
		for i, v := range s.Values {
			tx := float32(i) / denom
			if maxLen == 1 {
				tx = 0.5
			}
			ty := float32((v - minV) / (maxV - minV))
			pts[i] = render.PointF{
				X: float32(plot.X) + tx*float32(plot.W),
				Y: float32(plot.Y+plot.H) - ty*float32(plot.H),
			}
		}

		for i := 0; i < len(pts)-1; i++ {
			c.Canvas.DrawLine(pts[i].X, pts[i].Y, pts[i+1].X, pts[i+1].Y, c.ScaleF(2.0), col)
		}
		for i, p := range pts {
			c.Canvas.FillCircle(p.X, p.Y, c.ScaleF(3.0), col)
			if !c.IsInputBlocked() && plot.Contains(c.MouseX, c.MouseY) {
				dx := float32(math.Abs(float64(float32(c.MouseX) - p.X)))
				dy := float32(math.Abs(float64(float32(c.MouseY) - p.Y)))
				d := dx + dy*0.35
				if d < bestDistX {
					bestDistX = d
					lbl := fmt.Sprintf("#%d", i+1)
					if i < len(labels) && labels[i] != "" {
						lbl = labels[i]
					}
					best = &hoverCand{pt: p, label: lbl, name: s.Name, val: s.Values[i], col: col}
				}
			}
		}
	}

	if best != nil {
		c.Canvas.DrawLine(best.pt.X, float32(plot.Y), best.pt.X, float32(plot.Y+plot.H), 1.0, c.Theme.Border)
		c.Canvas.FillCircle(best.pt.X, best.pt.Y, c.ScaleF(5.0), best.col)
		tip := fmt.Sprintf("%s: %s", best.label, formatChartVal(best.val))
		if best.name != "" {
			tip = fmt.Sprintf("%s (%s): %s", best.name, best.label, formatChartVal(best.val))
		}
		c.Tooltip(tip, plot)
	}
}

// BarChart отрисовывает столбчатую диаграмма с сеткой, осями, легендой и тултипами.
func (c *Context) BarChart(id string, labels []string, series []ChartSeries, opts ...Option) {
	id = c.ResolveID("bchart", id)
	st := ApplyOptions(opts)
	hDp := Dp(220)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	bg := c.Theme.Card
	if st.Bg != nil {
		bg = *st.Bg
	}
	c.Canvas.DrawRoundedBox(r, c.ScaleF(c.Theme.RadiusMd), bg, 1.0, c.Theme.Border)

	if len(series) == 0 {
		return
	}
	c.RegisterHitTrack(id, r, platform.CursorArrow)

	legendH := c.drawChartLegend(render.Rect{X: r.X + c.Scale(12).Int(), Y: r.Y + c.Scale(8).Int(), W: r.W - c.Scale(24).Int(), H: c.Scale(20).Int()}, series)
	padLeft := c.Scale(48).Int()
	padRight := c.Scale(16).Int()
	padTop := c.Scale(12).Int() + legendH
	padBottom := c.Scale(28).Int()

	plot := render.Rect{
		X: r.X + padLeft,
		Y: r.Y + padTop,
		W: r.W - padLeft - padRight,
		H: r.H - padTop - padBottom,
	}
	if plot.W <= 10 || plot.H <= 10 {
		return
	}

	minV, maxV := calcMinMax(series)
	if minV > 0 {
		minV = 0
	}
	if maxV <= minV {
		maxV = minV + 1
	}

	c.drawChartGridAndYAxis(plot, minV, maxV, 4)

	maxLen := len(labels)
	for _, s := range series {
		if len(s.Values) > maxLen {
			maxLen = len(s.Values)
		}
	}
	if maxLen == 0 {
		return
	}

	c.drawChartXAxisLabels(plot, labels, maxLen, true)

	groupW := float32(plot.W) / float32(maxLen)
	innerPad := groupW * 0.16
	if innerPad < 2 {
		innerPad = 2
	}
	availGroupW := groupW - innerPad*2
	if availGroupW < 2 {
		availGroupW = 2
	}
	barGap := float32(c.Scale(2).Int())
	barW := (availGroupW - barGap*float32(len(series)-1)) / float32(len(series))
	if barW < 2 {
		barW = 2
	}

	var hovTip string
	var hovRect render.Rect

	for gi := 0; gi < maxLen; gi++ {
		groupX := float32(plot.X) + float32(gi)*groupW + innerPad
		for si, s := range series {
			if gi >= len(s.Values) {
				continue
			}
			v := s.Values[gi]
			normH := float32((v - minV) / (maxV - minV))
			bh := int(normH * float32(plot.H))
			if bh < 1 && v > minV {
				bh = 1
			}
			bx := int(groupX + float32(si)*(barW+barGap))
			by := plot.Y + plot.H - bh
			br := render.Rect{X: bx, Y: by, W: int(barW), H: bh}

			col := c.chartPalette(si)
			if s.Color != nil {
				col = *s.Color
			}

			hov := !c.IsInputBlocked() && br.Contains(c.MouseX, c.MouseY)
			if hov {
				col = col.WithAlpha(210)
				lbl := fmt.Sprintf("#%d", gi+1)
				if gi < len(labels) && labels[gi] != "" {
					lbl = labels[gi]
				}
				if s.Name != "" {
					hovTip = fmt.Sprintf("%s (%s): %s", s.Name, lbl, formatChartVal(v))
				} else {
					hovTip = fmt.Sprintf("%s: %s", lbl, formatChartVal(v))
				}
				hovRect = br
			}
			c.Canvas.FillRoundedRect(br, c.ScaleF(3), col)
		}
	}

	if hovTip != "" {
		c.Tooltip(hovTip, hovRect)
	}
}

func (c *Context) drawChartLegend(area render.Rect, series []ChartSeries) int {
	if len(series) == 0 {
		return 0
	}
	hasNames := false
	for _, s := range series {
		if s.Name != "" {
			hasNames = true
			break
		}
	}
	if !hasNames {
		return 0
	}

	curX := area.X
	px := c.Scale(11).Int()
	boxSize := c.Scale(10).Int()
	for i, s := range series {
		if s.Name == "" {
			continue
		}
		col := c.chartPalette(i)
		if s.Color != nil {
			col = *s.Color
		}
		c.Canvas.FillRoundedRect(render.Rect{X: curX, Y: area.Y + (area.H-boxSize)/2, W: boxSize, H: boxSize}, 2, col)
		curX += boxSize + c.Scale(5).Int()
		if c.Font != nil {
			tw, th := c.Font.MeasureText(s.Name, px)
			c.Font.DrawText(c.Canvas, curX, area.Y+(area.H-th)/2, s.Name, px, c.Theme.TextMuted)
			curX += tw + c.Scale(14).Int()
		}
	}
	return area.H + c.Scale(4).Int()
}

func (c *Context) drawChartGridAndYAxis(plot render.Rect, minV, maxV float64, steps int) {
	px := c.Scale(10).Int()
	for i := 0; i <= steps; i++ {
		t := float32(i) / float32(steps)
		y := plot.Y + plot.H - int(t*float32(plot.H))
		col := c.Theme.Border.WithAlpha(120)
		if i == 0 {
			col = c.Theme.Border
		}
		c.Canvas.FillRect(render.Rect{X: plot.X, Y: y, W: plot.W, H: 1}, col)

		if c.Font != nil {
			val := minV + float64(t)*(maxV-minV)
			lbl := formatChartVal(val)
			tw, th := c.Font.MeasureText(lbl, px)
			c.Font.DrawText(c.Canvas, plot.X-tw-c.Scale(6).Int(), y-th/2, lbl, px, c.Theme.TextMuted)
		}
	}
	c.Canvas.FillRect(render.Rect{X: plot.X, Y: plot.Y, W: 1, H: plot.H}, c.Theme.Border)
}

func (c *Context) drawChartXAxisLabels(plot render.Rect, labels []string, maxLen int, centered bool) {
	if c.Font == nil || len(labels) == 0 {
		return
	}
	px := c.Scale(10).Int()
	step := 1
	if maxLen > 8 {
		step = (maxLen + 7) / 8
	}
	for i := 0; i < len(labels) && i < maxLen; i += step {
		lbl := labels[i]
		if lbl == "" {
			continue
		}
		var cx int
		if centered {
			groupW := float32(plot.W) / float32(maxLen)
			cx = plot.X + int((float32(i)+0.5)*groupW)
		} else {
			denom := float32(maxLen - 1)
			if denom <= 0 {
				denom = 1
			}
			t := float32(i) / denom
			if maxLen == 1 {
				t = 0.5
			}
			cx = plot.X + int(t*float32(plot.W))
		}
		tw, _ := c.Font.MeasureText(lbl, px)
		c.Font.DrawText(c.Canvas, cx-tw/2, plot.Y+plot.H+c.Scale(6).Int(), lbl, px, c.Theme.TextMuted)
	}
}
