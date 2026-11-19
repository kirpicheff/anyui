package ui

import (
	"math"
	"time"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

// ToastKind задаёт тип всплывающего уведомления.
type ToastKind uint8

const (
	ToastInfo ToastKind = iota
	ToastSuccess
	ToastWarning
	ToastError
)

// ToastPosition задаёт позицию всплывающих уведомлений на экране.
type ToastPosition uint8

const (
	ToastBottomRight ToastPosition = iota
	ToastBottomLeft
	ToastTopRight
	ToastTopLeft
	ToastBottomCenter
	ToastTopCenter
)

// ToastMessage описывает активное всплывающее уведомление.
type ToastMessage struct {
	ID        int
	Title     string
	Text      string
	Kind      ToastKind
	Duration  time.Duration
	CreatedAt time.Time
}

// ShowToast добавляет всплывающее уведомление в очередь отображения.
// Если duration не задано, используется длительность по умолчанию (3.5 секунды).
func (c *Context) ShowToast(title, text string, kind ToastKind, duration ...time.Duration) {
	dur := 3500 * time.Millisecond
	if len(duration) > 0 && duration[0] > 0 {
		dur = duration[0]
	}
	c.nextToastID++
	c.toasts = append(c.toasts, &ToastMessage{
		ID:        c.nextToastID,
		Title:     title,
		Text:      text,
		Kind:      kind,
		Duration:  dur,
		CreatedAt: time.Now(),
	})
	// Ограничим максимум 5 одновременных тостов
	if len(c.toasts) > 5 {
		c.toasts = c.toasts[len(c.toasts)-5:]
	}
	if c.Animator != nil {
		c.Animator.MarkActive()
	}
	c.RequestRedraw()
}

// ToastsCount возвращает количество активных уведомлений (удобно для тестов).
func (c *Context) ToastsCount() int {
	return len(c.toasts)
}

// SetToastPosition задаёт угол или сторону окна для отображения всплывающих уведомлений.
func (c *Context) SetToastPosition(pos ToastPosition) {
	c.toastPos = pos
}

// SetToastOffset задаёт отступы всплывающих уведомлений от краёв окна в логических пикселях (по умолчанию 16, 16).
func (c *Context) SetToastOffset(xDp, yDp int) {
	if xDp < 0 {
		xDp = 0
	}
	if yDp < 0 {
		yDp = 0
	}
	c.toastOffsetXDp = xDp
	c.toastOffsetYDp = yDp
	c.toastHasOffset = true
}

// drawToasts удаляет просроченные уведомления и рисует активные поверх UI.
func (c *Context) drawToasts() {
	if len(c.toasts) == 0 {
		return
	}

	now := time.Now()
	alive := c.toasts[:0]
	for _, t := range c.toasts {
		if now.Sub(t.CreatedAt) < t.Duration {
			alive = append(alive, t)
		}
	}
	c.toasts = alive
	if len(c.toasts) == 0 {
		return
	}

	if c.Animator != nil {
		c.Animator.MarkActive()
	}

	toastW := c.S(300)
	toastH := c.S(64)
	gap := c.S(10)
	offX := c.S(16)
	offY := c.S(16)
	if c.toastHasOffset {
		offX = c.S(c.toastOffsetXDp)
		offY = c.S(c.toastOffsetYDp)
	}
	minPad := c.S(8)

	for i := len(c.toasts) - 1; i >= 0; i-- {
		t := c.toasts[i]
		idxFromEdge := len(c.toasts) - 1 - i
		rx, ry, ok := c.calcToastRectPos(toastW, toastH, gap, offX, offY, minPad, idxFromEdge)
		if !ok {
			continue
		}
		tr := render.Rect{X: rx, Y: ry, W: toastW, H: toastH}

		accent := c.Theme.Accent
		switch t.Kind {
		case ToastSuccess:
			accent = c.Theme.Success
		case ToastWarning:
			accent = c.Theme.Warning
		case ToastError:
			accent = c.Theme.Danger
		}

		elapsed := now.Sub(t.CreatedAt)
		remainRatio := 1.0 - float32(elapsed)/float32(t.Duration)
		if remainRatio < 0 {
			remainRatio = 0
		}

		toastRef := t
		c.PushOverlay(tr, func() {
			c.Canvas.DrawBoxShadow(tr, render.BoxShadow{
				OffsetY: c.Sf(4),
				Blur:    c.Sf(12),
				Color:   render.RGBA(0, 0, 0, 120),
				Radii:   render.UniformRadii(c.Sf(10)),
			})
			c.Canvas.DrawRoundedBox(tr, c.Sf(10), c.Theme.Card, 1.0, c.Theme.Border)

			// Левая цветная полоса типа уведомления
			barR := render.Rect{X: tr.X + c.S(4), Y: tr.Y + c.S(8), W: c.S(4), H: tr.H - c.S(16)}
			c.Canvas.FillRoundedRect(barR, c.Sf(2), accent)

			// Кнопка закрытия '×'
			closeR := render.Rect{X: tr.X + tr.W - c.S(26), Y: tr.Y + c.S(8), W: c.S(18), H: c.S(18)}
			c.RegisterHit("toast:close:"+toastRef.Title, closeR, platform.CursorHand)
			if closeR.Contains(c.MouseX, c.MouseY) {
				c.nextCursor = platform.CursorHand
				c.Canvas.FillRoundedRect(closeR, c.Sf(4), c.Theme.Surface)
				if c.MousePressed {
					toastRef.Duration = 0
					c.RequestRedraw()
				}
			}
			p := c.Sf(4)
			c.Canvas.DrawLine(float32(closeR.X)+p, float32(closeR.Y)+p, float32(closeR.X+closeR.W)-p, float32(closeR.Y+closeR.H)-p, c.Sf(1.4), c.Theme.TextMuted)
			c.Canvas.DrawLine(float32(closeR.X+closeR.W)-p, float32(closeR.Y)+p, float32(closeR.X)+p, float32(closeR.Y+closeR.H)-p, c.Sf(1.4), c.Theme.TextMuted)

			if c.Font != nil {
				textX := tr.X + c.S(16)
				textW := tr.W - c.S(46)
				c.Canvas.PushClip(render.Rect{X: textX, Y: tr.Y, W: textW, H: tr.H})
				if toastRef.Title != "" {
					c.Font.DrawTextStyled(c.Canvas, textX, tr.Y+c.S(10), toastRef.Title, c.S(13), c.Theme.TextPrimary, render.FontBold)
				}
				if toastRef.Text != "" {
					c.Font.DrawText(c.Canvas, textX, tr.Y+c.S(32), toastRef.Text, c.S(12), c.Theme.TextMuted)
				}
				c.Canvas.PopClip()
			}

			// Индикатор оставшегося времени внизу
			progW := int(float32(tr.W-c.S(16)) * remainRatio)
			if progW > 0 {
				c.Canvas.FillRoundedRect(render.Rect{X: tr.X + c.S(8), Y: tr.Y + tr.H - c.S(4), W: progW, H: c.S(2)}, c.Sf(1), accent.WithAlpha(180))
			}
		})
	}
}

// Spinner рисует анимированный индикатор загрузки.
func (c *Context) Spinner(id string, sizeDp int, opts ...Option) {
	id = c.ResolveID("spin", id)
	if sizeDp <= 0 {
		sizeDp = 24
	}
	if c.spinnerAngle == nil {
		c.spinnerAngle = make(map[string]float32)
	}
	st := ApplyOptions(opts)
	sz := c.S(sizeDp)
	rect := c.allocRect(sz, sz)

	dt := float32(1.0 / 60.0)
	if c.Animator != nil {
		if d := c.Animator.Delta(); d > 0 {
			dt = d
		}
		c.Animator.MarkActive()
	}

	angle := c.spinnerAngle[id] + dt*4.5 // ~0.72 оборота в секунду
	if angle > float32(2*math.Pi) {
		angle -= float32(2 * math.Pi)
	}
	c.spinnerAngle[id] = angle

	cx := float32(rect.X) + float32(rect.W)*0.5
	cy := float32(rect.Y) + float32(rect.H)*0.5
	rad := float32(sz)*0.5 - c.Sf(3)
	if rad < 2 {
		rad = 2
	}
	thick := c.Sf(2.8)
	if sz >= c.S(36) {
		thick = c.Sf(3.5)
	}

	col := c.Theme.Accent
	if st.Color != nil {
		col = *st.Color
	}

	// Фоновое кольцо + вращающаяся дуга
	c.Canvas.DrawArc(cx, cy, rad, 0, float32(2*math.Pi), thick, c.Theme.Border.WithAlpha(90))
	c.Canvas.DrawArc(cx, cy, rad, angle, angle+float32(math.Pi*1.15), thick, col)
}

func (c *Context) calcToastRectPos(toastW, toastH, gap, offX, offY, minPad, idx int) (int, int, bool) {
	var rx int
	switch c.toastPos {
	case ToastBottomLeft, ToastTopLeft:
		rx = offX
	case ToastBottomCenter, ToastTopCenter:
		rx = (c.Canvas.Width - toastW) / 2
	default:
		rx = c.Canvas.Width - toastW - offX
	}
	if rx < minPad {
		rx = minPad
	}

	var ry int
	switch c.toastPos {
	case ToastTopRight, ToastTopLeft, ToastTopCenter:
		ry = offY + idx*(toastH+gap)
		if ry+toastH > c.Canvas.Height-minPad {
			return 0, 0, false
		}
	default:
		ry = c.Canvas.Height - offY - (idx+1)*toastH - idx*gap
		if ry < minPad {
			return 0, 0, false
		}
	}
	return rx, ry, true
}
