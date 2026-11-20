package ui

import (
	"fmt"
	"strconv"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

type datePickerMode int

const (
	dpModeDays datePickerMode = iota
	dpModeMonths
	dpModeYears
)

var (
	dpModeStore    = make(map[string]datePickerMode)
	dpEditText     = make(map[string]string)
	dpEditActive   = make(map[string]bool)
	tpActiveField  = make(map[string]int) // 0=час, 1=мин, 2=сек
)

// TimePicker отрисовывает выбор времени (ЧЧ:ММ:СС) с кнопками +/- и вводом с клавиатуры.
func (c *Context) TimePicker(id string, t *time.Time, opts ...Option) bool {
	if t == nil {
		return false
	}
	id = c.ResolveID("tp", id)
	if t.IsZero() {
		*t = c.Now()
	}
	if c.openState == nil {
		c.openState = make(map[string]bool)
	}

	st := ApplyOptions(opts)
	w := c.S(140)
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(34)
	rect := c.allocRect(w, h)
	radius := c.Sf(c.Theme.RadiusSm)

	isOpen := c.openState[id]
	if !st.Disabled {
		c.RegisterHit(id, rect, platform.CursorHand)
	}
	hover := !c.IsInputBlocked() && !st.Disabled && rect.Contains(c.MouseX, c.MouseY)
	if hover {
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			isOpen = !isOpen
			c.openState[id] = isOpen
			c.RequestRedraw()
		}
	}

	borderCol := c.Theme.Border
	if isOpen {
		borderCol = c.Theme.BorderFocus
	}
	c.Canvas.DrawRoundedBox(rect, radius, c.Theme.SurfaceDark, 1.0, borderCol)

	if c.Font != nil {
		pxSize := c.S(13)
		label := t.Format("15:04:05")
		_, th := c.Font.MeasureText(label, pxSize)
		c.Font.DrawText(c.Canvas, rect.X+c.S(10), rect.Y+(rect.H-th)/2, label, pxSize, c.Theme.TextPrimary)
	}

	// Иконка часов справа
	cx := float32(rect.X + rect.W - c.S(16))
	cy := float32(rect.Y + rect.H/2)
	cr := render.Rect{X: rect.X + rect.W - c.S(23), Y: rect.Y + (rect.H-c.S(14))/2, W: c.S(14), H: c.S(14)}
	c.Canvas.DrawRoundedBox(cr, c.Sf(7), render.RGBA(0, 0, 0, 0), 1.2, c.Theme.TextMuted)
	c.Canvas.DrawLine(cx, cy, cx, cy-c.Sf(3.5), 1.2, c.Theme.TextMuted)
	c.Canvas.DrawLine(cx, cy, cx+c.Sf(3), cy+c.Sf(1), 1.2, c.Theme.TextMuted)

	changed := false
	if isOpen {
		popW := c.S(210)
		popH := c.S(146)
		popX := rect.X
		popY := rect.Y + rect.H + c.S(4)
		if popX+popW > c.Canvas.Width-c.S(8) {
			popX = c.Canvas.Width - popW - c.S(8)
		}
		if popY+popH > c.Canvas.Height-c.S(8) {
			popY = rect.Y - popH - c.S(4)
		}
		popR := render.Rect{X: popX, Y: popY, W: popW, H: popH}

		if c.MousePressed && !popR.Contains(c.MouseX, c.MouseY) && !rect.Contains(c.MouseX, c.MouseY) {
			c.openState[id] = false
			c.RequestRedraw()
			return false
		}

		c.PushOverlay(popR, func() {
			c.drawDropShadow(popR, 10, 14, 120)
			c.Canvas.DrawRoundedBox(popR, c.Sf(10), c.Theme.Card, 1.0, c.Theme.Border)

			hr, mn, sc := t.Hour(), t.Minute(), t.Second()
			colW := (popR.W - c.S(24)) / 3
			baseX := popR.X + c.S(12)
			topY := popR.Y + c.S(12)

			stepCol := func(idx int, label string, val int, maxVal int, setFn func(int)) {
				cxR := baseX + idx*colW
				upR := render.Rect{X: cxR + c.S(6), Y: topY, W: colW - c.S(12), H: c.S(26)}
				valR := render.Rect{X: cxR + c.S(6), Y: topY + c.S(30), W: colW - c.S(12), H: c.S(32)}
				dnR := render.Rect{X: cxR + c.S(6), Y: topY + c.S(66), W: colW - c.S(12), H: c.S(26)}

				drawStepBtn := func(br render.Rect, txt string, delta int) {
					c.RegisterHit(id+":tp:"+label+":"+txt, br, platform.CursorHand)
					hov := br.Contains(c.MouseX, c.MouseY)
					bg := c.Theme.Surface
					if hov {
						c.nextCursor = platform.CursorHand
						bg = c.Theme.Border
						if c.MousePressed {
							nv := (val + delta + maxVal) % maxVal
							setFn(nv)
							changed = true
							c.RequestRedraw()
						}
					}
					c.Canvas.FillRoundedRect(br, c.Sf(4), bg)
					if c.Font != nil {
						tw, th := c.Font.MeasureText(txt, c.S(12))
						c.Font.DrawText(c.Canvas, br.X+(br.W-tw)/2, br.Y+(br.H-th)/2, txt, c.S(12), c.Theme.TextPrimary)
					}
				}

				drawStepBtn(upR, "+", 1)
				drawStepBtn(dnR, "-", -1)

				c.Canvas.DrawRoundedBox(valR, c.Sf(4), c.Theme.Bg, 1.0, c.Theme.Border)
				if valR.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
					d := 1
					if c.WheelY < 0 {
						d = -1
					}
					nv := (val + d + maxVal) % maxVal
					setFn(nv)
					changed = true
					c.RequestRedraw()
				}
				if c.Font != nil {
					vs := fmt.Sprintf("%02d", val)
					tw, th := c.Font.MeasureText(vs, c.S(15))
					c.Font.DrawText(c.Canvas, valR.X+(valR.W-tw)/2, valR.Y+(valR.H-th)/2, vs, c.S(15), c.Theme.TextPrimary)
				}
			}

			stepCol(0, "Ч", hr, 24, func(nv int) {
				*t = time.Date(t.Year(), t.Month(), t.Day(), nv, t.Minute(), t.Second(), 0, t.Location())
			})
			stepCol(1, "М", mn, 60, func(nv int) {
				*t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), nv, t.Second(), 0, t.Location())
			})
			stepCol(2, "С", sc, 60, func(nv int) {
				*t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), nv, 0, t.Location())
			})

			// Кнопка "Сейчас" внизу
			nowR := render.Rect{X: popR.X + c.S(12), Y: popR.Y + popR.H - c.S(34), W: popR.W - c.S(24), H: c.S(24)}
			c.RegisterHit(id+":now", nowR, platform.CursorHand)
			hovNow := nowR.Contains(c.MouseX, c.MouseY)
			nowBg := c.Theme.Surface
			if hovNow {
				c.nextCursor = platform.CursorHand
				nowBg = c.Theme.Accent.WithAlpha(40)
				if c.MousePressed {
					now := c.Now()
					*t = time.Date(t.Year(), t.Month(), t.Day(), now.Hour(), now.Minute(), now.Second(), 0, t.Location())
					changed = true
					c.RequestRedraw()
				}
			}
			c.Canvas.DrawRoundedBox(nowR, c.Sf(4), nowBg, 1.0, c.Theme.Border)
			if c.Font != nil {
				nowLbl := c.str().Now
				tw, th := c.Font.MeasureText(nowLbl, c.S(11))
				c.Font.DrawText(c.Canvas, nowR.X+(nowR.W-tw)/2, nowR.Y+(nowR.H-th)/2, nowLbl, c.S(11), c.Theme.Accent)
			}
		})
	}
	return changed
}

// DateTimePicker объединяет выбор даты (DatePicker) и времени (TimePicker) в одной строке.
func (c *Context) DateTimePicker(id string, dt *time.Time, opts ...Option) bool {
	if dt == nil {
		return false
	}
	id = c.nextLocalID("dtp", id)
	if dt.IsZero() {
		*dt = c.Now()
	}
	changed := false
	dCopy := *dt
	tCopy := *dt
	c.Row(8, 34, func() {
		if c.DatePicker(id+":date", &dCopy, Width(150)) {
			*dt = time.Date(dCopy.Year(), dCopy.Month(), dCopy.Day(), dt.Hour(), dt.Minute(), dt.Second(), 0, dt.Location())
			changed = true
		}
		if c.TimePicker(id+":time", &tCopy, Width(120)) {
			*dt = time.Date(dt.Year(), dt.Month(), dt.Day(), tCopy.Hour(), tCopy.Minute(), tCopy.Second(), 0, dt.Location())
			changed = true
		}
	})
	return changed
}

func parseDateInput(s string, loc *time.Location) (time.Time, bool) {
	if loc == nil {
		loc = time.Local
	}
	parsed, err := time.ParseInLocation("2006-01-02", s, loc)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func formatYearRangeTitle(baseYear int) string {
	start := (baseYear / 12) * 12
	return strconv.Itoa(start) + " - " + strconv.Itoa(start+11)
}
