package ui

import (
	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

// TitleBarConfig задаёт настройки кастомной шапки окна (для режима Frameless и тем на любых ОС).
type TitleBarConfig struct {
	IconSVG       string
	HeightDp      Dp
	HideMinimize  bool
	HideMaximize  bool
	HideClose     bool
	LeftContent   func()
	CenterContent func()
	RightContent  func()
	OnDoubleClick func()
}

// TitleBar отрисовывает кастомную шапку окна с иконкой, заголовком, пользовательским контентом
// и системными кнопками управления окном (Свернуть / Развернуть / Закрыть).
func (c *Context) TitleBar(id, title string, cfg TitleBarConfig, opts ...Option) {
	st := ApplyOptions(opts)
	hDp := Dp(36)
	if cfg.HeightDp > 0 {
		hDp = cfg.HeightDp
	}
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}

	r := c.allocWidth(hDp, st)
	c.titleBarRect = r
	c.titleBarInteractive = c.titleBarInteractive[:0]

	bg := c.Theme.Surface
	if st.Bg != nil {
		bg = *st.Bg
	}
	c.Canvas.FillRect(r, bg)
	c.Canvas.FillRect(render.Rect{X: r.X, Y: r.Y + r.H - 1, W: r.W, H: 1}, c.Theme.Border)

	btnW := c.Scale(44).Int()
	btnCount := 0
	if !cfg.HideMinimize {
		btnCount++
	}
	if !cfg.HideMaximize {
		btnCount++
	}
	if !cfg.HideClose {
		btnCount++
	}

	// Левая часть: иконка и заголовок
	curX := r.X + c.Scale(12).Int()
	if cfg.IconSVG != "" {
		iconSize := c.Scale(16).Int()
		iy := r.Y + (r.H-iconSize)/2
		c.Canvas.DrawSVG(cfg.IconSVG, curX, iy, iconSize, c.Theme.Accent)
		curX += iconSize + c.Scale(8).Int()
	}

	if title != "" && c.Font != nil {
		pxSize := c.Scale(13).Int()
		if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
			pxSize = c.Scale(*st.FontSizeDp).Int()
		}
		fg := c.Theme.TextPrimary
		if st.Color != nil {
			fg = *st.Color
		}
		tw, th := c.Font.MeasureText(title, pxSize)
		c.Font.DrawText(c.Canvas, curX, r.Y+(r.H-th)/2, title, pxSize, fg)
		curX += tw + c.Scale(12).Int()
	}

	// Правая граница доступная для контента (до системных кнопок)
	rightLimit := r.X + r.W - btnCount*btnW

	if cfg.LeftContent != nil && curX < rightLimit {
		leftBox := render.Rect{X: curX, Y: r.Y, W: rightLimit - curX, H: r.H}
		c.renderTitleBarSlot(leftBox, cfg.LeftContent)
	}

	if cfg.CenterContent != nil && rightLimit > r.X {
		cw := rightLimit - r.X
		centerBox := render.Rect{X: r.X + cw/4, Y: r.Y, W: cw / 2, H: r.H}
		if centerBox.W > 0 {
			c.renderTitleBarSlot(centerBox, cfg.CenterContent)
		}
	}

	// Кнопки управления окном (справа налево: Close, Maximize/Restore, Minimize)
	bx := r.X + r.W
	if !cfg.HideClose {
		bx -= btnW
		cr := render.Rect{X: bx, Y: r.Y, W: btnW, H: r.H - 1}
		c.drawTitleBarCloseBtn(id+":close", cr)
	}
	if !cfg.HideMaximize {
		bx -= btnW
		mr := render.Rect{X: bx, Y: r.Y, W: btnW, H: r.H - 1}
		c.drawTitleBarMaxBtn(id+":max", mr)
	}
	if !cfg.HideMinimize {
		bx -= btnW
		mnr := render.Rect{X: bx, Y: r.Y, W: btnW, H: r.H - 1}
		c.drawTitleBarMinBtn(id+":min", mnr)
	}

	if cfg.RightContent != nil && bx > curX {
		rw := c.Scale(140).Int()
		if bx-curX < rw {
			rw = bx - curX
		}
		if rw > 0 {
			rightBox := render.Rect{X: bx - rw - c.Scale(4).Int(), Y: r.Y, W: rw, H: r.H}
			c.renderTitleBarSlot(rightBox, cfg.RightContent)
		}
	}

	// Двойной клик по свободной области шапки разворачивает/восстанавливает окно
	if r.Contains(c.MouseX, c.MouseY) && c.DoubleClicked() && !c.isPointInTitleBarInteractive(c.MouseX, c.MouseY) {
		if cfg.OnDoubleClick != nil {
			cfg.OnDoubleClick()
		} else if c.Win != nil {
			if c.Win.IsMaximized() {
				c.Win.Restore()
			} else {
				c.Win.Maximize()
			}
		}
	}
}

func (c *Context) renderTitleBarSlot(box render.Rect, fn func()) {
	startHits := len(c.nextHitRegions)
	c.pushBox(box, dirRow, AlignCenter, c.Scale(6).Int())
	fn()
	c.popBox()
	for i := startHits; i < len(c.nextHitRegions); i++ {
		c.titleBarInteractive = append(c.titleBarInteractive, c.nextHitRegions[i].rect)
	}
}

func (c *Context) drawTitleBarMinBtn(id string, r render.Rect) {
	c.titleBarInteractive = append(c.titleBarInteractive, r)
	c.RegisterHit(id, r, platform.CursorArrow)
	hov := !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hov {
		c.Canvas.FillRect(r, c.Theme.Card)
		if c.MousePressed && c.Win != nil {
			c.Win.Minimize()
		}
	}
	cx := float32(r.X) + float32(r.W)/2
	cy := float32(r.Y) + float32(r.H)/2
	half := c.ScaleF(5)
	c.Canvas.DrawLine(cx-half, cy, cx+half, cy, c.ScaleF(1.3), c.Theme.TextPrimary)
	c.Tooltip(c.str().MinimizeTooltip, r)
}

func (c *Context) drawTitleBarMaxBtn(id string, r render.Rect) {
	c.titleBarInteractive = append(c.titleBarInteractive, r)
	c.RegisterHit(id, r, platform.CursorArrow)
	isMax := c.Win != nil && c.Win.IsMaximized()
	hov := !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hov {
		c.Canvas.FillRect(r, c.Theme.Card)
		if c.MousePressed && c.Win != nil {
			if isMax {
				c.Win.Restore()
			} else {
				c.Win.Maximize()
			}
		}
	}

	cx := r.X + r.W/2
	cy := r.Y + r.H/2
	sz := c.Scale(9).Int()
	if isMax {
		// Два смещённых квадрата для состояния Restore
		back := render.Rect{X: cx - sz/2 + c.Scale(2).Int(), Y: cy - sz/2 - c.Scale(1).Int(), W: sz - c.Scale(2).Int(), H: sz - c.Scale(2).Int()}
		front := render.Rect{X: cx - sz/2 - c.Scale(1).Int(), Y: cy - sz/2 + c.Scale(2).Int(), W: sz - c.Scale(2).Int(), H: sz - c.Scale(2).Int()}
		c.Canvas.DrawRoundedBox(back, 0, render.RGBA(0, 0, 0, 0), 1.2, c.Theme.TextPrimary)
		c.Canvas.DrawRoundedBox(front, 0, c.Theme.Surface, 1.2, c.Theme.TextPrimary)
		c.Tooltip(c.str().RestoreTooltip, r)
	} else {
		box := render.Rect{X: cx - sz/2, Y: cy - sz/2, W: sz, H: sz}
		c.Canvas.DrawRoundedBox(box, 0, render.RGBA(0, 0, 0, 0), 1.2, c.Theme.TextPrimary)
		c.Tooltip(c.str().MaximizeTooltip, r)
	}
}

func (c *Context) drawTitleBarCloseBtn(id string, r render.Rect) {
	c.titleBarInteractive = append(c.titleBarInteractive, r)
	c.RegisterHit(id, r, platform.CursorArrow)
	hov := !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	iconCol := c.Theme.TextPrimary
	if hov {
		c.Canvas.FillRect(r, c.Theme.Danger)
		iconCol = render.RGBA(255, 255, 255, 255)
		if c.MousePressed && c.Win != nil {
			c.Win.Close()
		}
	}
	cx := float32(r.X) + float32(r.W)/2
	cy := float32(r.Y) + float32(r.H)/2
	half := c.ScaleF(4.5)
	stroke := c.ScaleF(1.4)
	c.Canvas.DrawLine(cx-half, cy-half, cx+half, cy+half, stroke, iconCol)
	c.Canvas.DrawLine(cx+half, cy-half, cx-half, cy+half, stroke, iconCol)
	c.Tooltip(c.str().CloseTooltip, r)
}

func (c *Context) isPointInTitleBarInteractive(x, y int) bool {
	for _, ir := range c.titleBarInteractive {
		if ir.Contains(x, y) {
			return true
		}
	}
	return false
}

// TitleBarHitTest возвращает HitCaption для свободной области последней отрисованной TitleBar
// и HitClient для кнопок управления окном, вложенных контролов и остальной части окна.
func (c *Context) TitleBarHitTest(xPx, yPx Px) platform.HitTestResult {
	x, y := xPx.Int(), yPx.Int()
	if c.titleBarRect.Empty() || !c.titleBarRect.Contains(x, y) {
		return platform.HitClient
	}
	if c.isPointInTitleBarInteractive(x, y) {
		return platform.HitClient
	}
	return platform.HitCaption
}
