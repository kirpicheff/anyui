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
		CreatedAt: c.Now(),
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
func (c *Context) SetToastOffset(xDp, yDp Dp) {
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

	now := c.Now()
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

	toastW := c.Scale(300).Int()
	toastH := c.Scale(64).Int()
	gap := c.Scale(10).Int()
	offX := c.Scale(16).Int()
	offY := c.Scale(16).Int()
	if c.toastHasOffset {
		offX = c.Scale(c.toastOffsetXDp).Int()
		offY = c.Scale(c.toastOffsetYDp).Int()
	}
	minPad := c.Scale(8).Int()

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

		c.Canvas.DrawBoxShadow(tr, render.BoxShadow{
			OffsetY: c.ScaleF(4),
			Blur:    c.ScaleF(12),
			Color:   render.RGBA(0, 0, 0, 120),
			Radii:   render.UniformRadii(c.ScaleF(10)),
		})
		c.Canvas.DrawRoundedBox(tr, c.ScaleF(10), c.Theme.Card, 1.0, c.Theme.Border)

		// Левая цветная полоса типа уведомления
		barR := render.Rect{X: tr.X + c.Scale(4).Int(), Y: tr.Y + c.Scale(8).Int(), W: c.Scale(4).Int(), H: tr.H - c.Scale(16).Int()}
		c.Canvas.FillRoundedRect(barR, c.ScaleF(2), accent)

		// Кнопка закрытия '×'
		closeR := render.Rect{X: tr.X + tr.W - c.Scale(26).Int(), Y: tr.Y + c.Scale(8).Int(), W: c.Scale(18).Int(), H: c.Scale(18).Int()}
		c.RegisterHit("toast:close:"+t.Title, closeR, platform.CursorHand)
		if closeR.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorHand
			c.Canvas.FillRoundedRect(closeR, c.ScaleF(4), c.Theme.Surface)
			if c.MousePressed {
				t.Duration = 0
				c.RequestRedraw()
			}
		}
		p := c.ScaleF(4)
		c.Canvas.DrawLine(float32(closeR.X)+p, float32(closeR.Y)+p, float32(closeR.X+closeR.W)-p, float32(closeR.Y+closeR.H)-p, c.ScaleF(1.4), c.Theme.TextMuted)
		c.Canvas.DrawLine(float32(closeR.X+closeR.W)-p, float32(closeR.Y)+p, float32(closeR.X)+p, float32(closeR.Y+closeR.H)-p, c.ScaleF(1.4), c.Theme.TextMuted)

		if c.Font != nil {
			textX := tr.X + c.Scale(16).Int()
			textW := tr.W - c.Scale(46).Int()
			c.Canvas.PushClip(render.Rect{X: textX, Y: tr.Y, W: textW, H: tr.H})
			if t.Title != "" {
				c.Font.DrawTextStyled(c.Canvas, textX, tr.Y+c.Scale(10).Int(), t.Title, c.Scale(13).Int(), c.Theme.TextPrimary, render.FontBold)
			}
			if t.Text != "" {
				c.Font.DrawText(c.Canvas, textX, tr.Y+c.Scale(32).Int(), t.Text, c.Scale(12).Int(), c.Theme.TextMuted)
			}
			c.Canvas.PopClip()
		}

		// Индикатор оставшегося времени внизу
		progW := int(float32(tr.W-c.Scale(16).Int()) * remainRatio)
		if progW > 0 {
			c.Canvas.FillRoundedRect(render.Rect{X: tr.X + c.Scale(8).Int(), Y: tr.Y + tr.H - c.Scale(4).Int(), W: progW, H: c.Scale(2).Int()}, c.ScaleF(1), accent.WithAlpha(180))
		}
	}
}

// Spinner рисует анимированный индикатор загрузки.
func (c *Context) Spinner(id string, sizeDp Dp, opts ...Option) {
	id = c.ResolveID("spin", id)
	if sizeDp <= 0 {
		sizeDp = 24
	}
	if c.spinnerAngle == nil {
		c.spinnerAngle = make(map[string]float32)
	}
	st := ApplyOptions(opts)
	sz := c.Scale(sizeDp).Int()
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
	rad := float32(sz)*0.5 - c.ScaleF(3)
	if rad < 2 {
		rad = 2
	}
	thick := c.ScaleF(2.8)
	if sz >= c.Scale(36).Int() {
		thick = c.ScaleF(3.5)
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
