package ui

import (
	"strings"

	"anyui/platform"
	"anyui/render"
)

func (c *Context) ensureTextMaps() {
	if c.caretPos == nil {
		c.caretPos = make(map[string]int)
	}
	if c.textSelStart == nil {
		c.textSelStart = make(map[string]int)
	}
	if c.textScroll == nil {
		c.textScroll = make(map[string]int)
	}
}

func clampRuneIdx(idx, n int) int {
	if idx < 0 {
		return 0
	}
	if idx > n {
		return n
	}
	return idx
}

func selectionBounds(caret, selStart, n int) (int, int, bool) {
	c1 := clampRuneIdx(caret, n)
	c2 := clampRuneIdx(selStart, n)
	if c1 == c2 {
		return c1, c1, false
	}
	if c1 > c2 {
		return c2, c1, true
	}
	return c1, c2, true
}

func deleteRuneRange(runes []rune, start, end int) ([]rune, int) {
	next := make([]rune, 0, len(runes)-(end-start))
	next = append(next, runes[:start]...)
	next = append(next, runes[end:]...)
	return next, start
}

// CaretIndexAtX вычисляет индекс руны в строке по локальной X-координате.
func (c *Context) CaretIndexAtX(runes []rune, relX, pxSize int, style render.FontStyle) int {
	if c.Font == nil || len(runes) == 0 || relX <= 0 {
		return 0
	}
	prevW := 0
	for i := 1; i <= len(runes); i++ {
		w, _ := c.Font.MeasureTextStyled(string(runes[:i]), pxSize, style)
		mid := prevW + (w-prevW)/2
		if relX < mid {
			return i - 1
		}
		prevW = w
	}
	return len(runes)
}

func maskRunes(runes []rune) []rune {
	out := make([]rune, len(runes))
	for i := range out {
		out[i] = '•'
	}
	return out
}

// TextInput рисует однострочное поле ввода текста и возвращает true при изменении значения.
func (c *Context) TextInput(id string, value *string, placeholder string, widthDp int) bool {
	if widthDp > 0 {
		return c.TextInputOpts(id, value, placeholder, Width(widthDp))
	}
	return c.TextInputOpts(id, value, placeholder)
}

// TextInputOpts рисует однострочное поле ввода текста с поддержкой выделения, курсора мышью,
// буфера обмена, режима пароля, иконок и кнопки очистки.
func (c *Context) TextInputOpts(id string, value *string, placeholder string, opts ...Option) bool {
	if value == nil {
		return false
	}
	c.ensureTextMaps()
	st := ApplyOptions(opts)

	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(38)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	r := c.allocRect(w, h)
	if !st.Disabled {
		c.registerFocusable(id)
	}

	padX := c.S(12)
	if st.PadDp != nil && *st.PadDp >= 0 {
		padX = c.S(*st.PadDp)
	}
	iconSize := c.S(16)
	iconGap := c.S(8)

	leftOffset := padX
	if st.IconSVG != "" {
		leftOffset += iconSize + iconGap
	}
	rightOffset := padX
	showClear := st.Clearable && !st.Disabled && !st.ReadOnly && len(*value) > 0
	if showClear {
		rightOffset += iconSize + iconGap
	} else if st.IconRightSVG != "" {
		rightOffset += iconSize + iconGap
	}

	textArea := render.Rect{
		X: r.X + leftOffset,
		Y: r.Y + c.S(2),
		W: r.W - leftOffset - rightOffset,
		H: r.H - c.S(4),
	}

	fontSizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		fontSizeDp = *st.FontSizeDp
	}
	pxSize := c.S(fontSizeDp)

	runes := []rune(*value)
	displayRunes := runes
	if st.Password {
		displayRunes = maskRunes(runes)
	}

	changed := false
	clearClicked := false
	if showClear {
		clearRect := render.Rect{
			X: r.X + r.W - padX - iconSize,
			Y: r.Y + (r.H-iconSize)/2,
			W: iconSize,
			H: iconSize,
		}
		if !c.IsInputBlocked() && clearRect.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorHand
			if c.MousePressed {
				*value = ""
				c.caretPos[id] = 0
				c.textSelStart[id] = 0
				c.textScroll[id] = 0
				c.FocusID = id
				changed = true
				clearClicked = true
				c.RequestRedraw()
			}
		}
	}

	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered && !clearClicked {
		c.HotID = id
		if textArea.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorIBeam
		}
		if c.MousePressed {
			c.ActiveID = id
			c.FocusID = id
			idx := c.CaretIndexAtX(displayRunes, c.MouseX-textArea.X+c.textScroll[id], pxSize, st.FontStyle)
			c.caretPos[id] = idx
			c.textSelStart[id] = idx
		}
	}

	// Выделение текста протяжкой мыши
	if !st.Disabled && c.ActiveID == id && c.MouseDown && !clearClicked {
		idx := c.CaretIndexAtX(displayRunes, c.MouseX-textArea.X+c.textScroll[id], pxSize, st.FontStyle)
		c.caretPos[id] = idx
	}

	focused := !st.Disabled && c.FocusID == id
	if focused && !clearClicked {
		runes = []rune(*value)
		caret := clampRuneIdx(c.caretPos[id], len(runes))
		selStart := clampRuneIdx(c.textSelStart[id], len(runes))

		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl {
				switch kp.Code {
				case KeyA:
					selStart = 0
					caret = len(runes)
				case KeyC:
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						_ = c.Win.SetClipboardText(string(runes[a:b]))
					} else if !st.Password {
						_ = c.Win.SetClipboardText(string(runes))
					}
				case KeyX:
					if !st.ReadOnly {
						if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
							if !st.Password {
								_ = c.Win.SetClipboardText(string(runes[a:b]))
							}
							runes, caret = deleteRuneRange(runes, a, b)
							selStart = caret
							changed = true
						}
					}
				case KeyV:
					if !st.ReadOnly {
						if clip, err := c.Win.ClipboardText(); err == nil && clip != "" {
							clip = strings.ReplaceAll(clip, "\r", "")
							clip = strings.ReplaceAll(clip, "\n", " ")
							var ins []rune
							for _, ch := range clip {
								if st.InputFilter == nil || st.InputFilter(ch) {
									ins = append(ins, ch)
								}
							}
							if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
								runes, caret = deleteRuneRange(runes, a, b)
							}
							next := make([]rune, 0, len(runes)+len(ins))
							next = append(next, runes[:caret]...)
							next = append(next, ins...)
							next = append(next, runes[caret:]...)
							runes = next
							caret += len(ins)
							selStart = caret
							changed = true
						}
					}
				}
				continue
			}

			switch kp.Code {
			case KeyBackspace:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						runes, caret = deleteRuneRange(runes, a, b)
						selStart = caret
						changed = true
					} else if caret > 0 && len(runes) > 0 {
						runes = append(runes[:caret-1], runes[caret:]...)
						caret--
						selStart = caret
						changed = true
					}
				}
			case KeyDelete:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						runes, caret = deleteRuneRange(runes, a, b)
						selStart = caret
						changed = true
					} else if caret < len(runes) {
						runes = append(runes[:caret], runes[caret+1:]...)
						selStart = caret
						changed = true
					}
				}
			case KeyLeft:
				if caret > 0 {
					caret--
				}
				if !kp.Mods.Shift {
					selStart = caret
				}
			case KeyRight:
				if caret < len(runes) {
					caret++
				}
				if !kp.Mods.Shift {
					selStart = caret
				}
			case KeyHome:
				caret = 0
				if !kp.Mods.Shift {
					selStart = caret
				}
			case KeyEnd:
				caret = len(runes)
				if !kp.Mods.Shift {
					selStart = caret
				}
			case KeyEscape, KeyEnter:
				c.FocusID = ""
				focused = false
			}
		}

		if !st.ReadOnly {
			for _, ch := range c.TypedRunes {
				if ch < 32 {
					continue
				}
				if st.InputFilter != nil && !st.InputFilter(ch) {
					continue
				}
				if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
					runes, caret = deleteRuneRange(runes, a, b)
				}
				next := make([]rune, 0, len(runes)+1)
				next = append(next, runes[:caret]...)
				next = append(next, ch)
				next = append(next, runes[caret:]...)
				runes = next
				caret++
				selStart = caret
				changed = true
			}
		}

		c.caretPos[id] = caret
		c.textSelStart[id] = selStart
		if changed {
			*value = string(runes)
			displayRunes = runes
			if st.Password {
				displayRunes = maskRunes(runes)
			}
			c.RequestRedraw()
		}
	}

	focusTarget := float32(0)
	if focused {
		focusTarget = 1
	}
	focusAnim := c.animFloat("inp_f:"+id, focusTarget, 0.14)

	baseBorder := c.Theme.Border
	if st.BorderColor != nil {
		baseBorder = *st.BorderColor
	}
	borderCol := render.LerpColor(baseBorder, c.Theme.BorderFocus, focusAnim)

	bg := c.Theme.SurfaceDark
	if st.Bg != nil {
		bg = *st.Bg
	}
	rad := c.Theme.RadiusSm
	if st.Radius != nil {
		rad = *st.Radius
	}

	c.Canvas.DrawRoundedBox(r, c.Sf(rad), bg, 1.0, borderCol)

	if st.IconSVG != "" {
		c.Canvas.DrawSVG(st.IconSVG, r.X+padX, r.Y+(r.H-iconSize)/2, iconSize, c.Theme.TextMuted)
	}
	if showClear {
		cx := float32(r.X + r.W - padX - iconSize)
		cy := float32(r.Y + (r.H-iconSize)/2)
		pad := float32(iconSize) * 0.28
		c.Canvas.DrawLine(cx+pad, cy+pad, cx+float32(iconSize)-pad, cy+float32(iconSize)-pad, c.Sf(1.5), c.Theme.TextMuted)
		c.Canvas.DrawLine(cx+float32(iconSize)-pad, cy+pad, cx+pad, cy+float32(iconSize)-pad, c.Sf(1.5), c.Theme.TextMuted)
	} else if st.IconRightSVG != "" {
		c.Canvas.DrawSVG(st.IconRightSVG, r.X+r.W-padX-iconSize, r.Y+(r.H-iconSize)/2, iconSize, c.Theme.TextMuted)
	}

	if textArea.W <= 0 || c.Font == nil {
		return changed
	}

	c.Canvas.PushClip(textArea)

	caret := clampRuneIdx(c.caretPos[id], len(displayRunes))
	selStart := clampRuneIdx(c.textSelStart[id], len(displayRunes))

	prefixW, lineH := c.Font.MeasureTextStyled(string(displayRunes[:caret]), pxSize, st.FontStyle)
	scroll := c.textScroll[id]
	if prefixW-scroll > textArea.W-c.S(4) {
		scroll = prefixW - (textArea.W - c.S(4))
	}
	if prefixW-scroll < 0 {
		scroll = prefixW
	}
	if scroll < 0 {
		scroll = 0
	}
	c.textScroll[id] = scroll

	if focused {
		if a, b, ok := selectionBounds(caret, selStart, len(displayRunes)); ok {
			x1, _ := c.Font.MeasureTextStyled(string(displayRunes[:a]), pxSize, st.FontStyle)
			x2, _ := c.Font.MeasureTextStyled(string(displayRunes[:b]), pxSize, st.FontStyle)
			selRect := render.Rect{
				X: textArea.X + x1 - scroll,
				Y: textArea.Y + c.S(5),
				W: x2 - x1,
				H: textArea.H - c.S(10),
			}
			c.Canvas.FillRect(selRect, c.Theme.Accent.WithAlpha(90))
		}
	}

	textCol := c.Theme.TextPrimary
	if st.Color != nil {
		textCol = *st.Color
	}
	if st.Disabled {
		textCol = c.Theme.TextMuted
	}

	textY := textArea.Y + (textArea.H-lineH)/2
	if len(displayRunes) == 0 && !focused && placeholder != "" {
		c.Font.DrawTextStyled(c.Canvas, textArea.X, textY, placeholder, pxSize, c.Theme.TextMuted, st.FontStyle)
	} else {
		c.Font.DrawTextStyled(c.Canvas, textArea.X-scroll, textY, string(displayRunes), pxSize, textCol, st.FontStyle)
	}

	if focused {
		caretX := textArea.X + prefixW - scroll
		caretTop := textArea.Y + c.S(6)
		caretH := textArea.H - c.S(12)
		c.Canvas.FillRect(render.Rect{X: caretX, Y: caretTop, W: c.S(2), H: caretH}, c.Theme.Accent)
	}

	c.Canvas.PopClip()
	return changed
}

// WrapTextLines разбивает текст на визуальные строки с учётом '\n' и максимальной ширины.
func (c *Context) WrapTextLines(runes []rune, maxW, pxSize int, style render.FontStyle) []textLineSpan {
	type rawLine struct {
		start, end int
	}
	var raw []rawLine
	s := 0
	for i, ch := range runes {
		if ch == '\n' {
			raw = append(raw, rawLine{start: s, end: i})
			s = i + 1
		}
	}
	raw = append(raw, rawLine{start: s, end: len(runes)})

	var lines []textLineSpan
	for _, rl := range raw {
		if rl.start == rl.end || c.Font == nil || maxW <= 0 {
			lines = append(lines, textLineSpan{start: rl.start, end: rl.end})
			continue
		}
		cur := rl.start
		for cur < rl.end {
			end := cur
			for end < rl.end {
				w, _ := c.Font.MeasureTextStyled(string(runes[cur:end+1]), pxSize, style)
				if w > maxW && end > cur {
					break
				}
				end++
			}
			lines = append(lines, textLineSpan{start: cur, end: end})
			cur = end
		}
	}
	if len(lines) == 0 {
		lines = append(lines, textLineSpan{start: 0, end: 0})
	}
	return lines
}

type textLineSpan struct {
	start int
	end   int
}

// TextArea рисует многострочное поле ввода текста с переносами и вертикальным скроллом.
func (c *Context) TextArea(id string, value *string, placeholder string, opts ...Option) bool {
	if value == nil {
		return false
	}
	c.ensureTextMaps()
	st := ApplyOptions(opts)

	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(120)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	r := c.allocRect(w, h)
	if !st.Disabled {
		c.registerFocusable(id)
	}

	pad := c.S(10)
	if st.PadDp != nil && *st.PadDp >= 0 {
		pad = c.S(*st.PadDp)
	}
	textArea := render.Rect{
		X: r.X + pad,
		Y: r.Y + pad,
		W: r.W - pad*2,
		H: r.H - pad*2,
	}

	fontSizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		fontSizeDp = *st.FontSizeDp
	}
	pxSize := c.S(fontSizeDp)
	_, lineH := 0, c.S(18)
	if c.Font != nil {
		_, lineH = c.Font.MeasureTextStyled("Аy", pxSize, st.FontStyle)
		if lineH <= 0 {
			lineH = c.S(18)
		}
	}
	lineStep := lineH + c.S(4)

	runes := []rune(*value)
	lines := c.WrapTextLines(runes, textArea.W-c.S(6), pxSize, st.FontStyle)

	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered {
		c.HotID = id
		c.nextCursor = platform.CursorIBeam
		if c.WheelY != 0 {
			sc := c.textScroll[id] - int(c.WheelY*float32(lineStep)*2)
			maxSc := len(lines)*lineStep - textArea.H
			if maxSc < 0 {
				maxSc = 0
			}
			if sc < 0 {
				sc = 0
			}
			if sc > maxSc {
				sc = maxSc
			}
			c.textScroll[id] = sc
		}
		if c.MousePressed {
			c.ActiveID = id
			c.FocusID = id
			lineIdx := (c.MouseY - textArea.Y + c.textScroll[id]) / lineStep
			if lineIdx < 0 {
				lineIdx = 0
			}
			if lineIdx >= len(lines) {
				lineIdx = len(lines) - 1
			}
			sp := lines[lineIdx]
			col := c.CaretIndexAtX(runes[sp.start:sp.end], c.MouseX-textArea.X, pxSize, st.FontStyle)
			c.caretPos[id] = sp.start + col
			c.textSelStart[id] = c.caretPos[id]
		}
	}

	changed := false
	focused := !st.Disabled && c.FocusID == id
	if focused {
		caret := clampRuneIdx(c.caretPos[id], len(runes))
		selStart := clampRuneIdx(c.textSelStart[id], len(runes))

		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl {
				switch kp.Code {
				case KeyA:
					selStart = 0
					caret = len(runes)
				case KeyC:
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						_ = c.Win.SetClipboardText(string(runes[a:b]))
					} else {
						_ = c.Win.SetClipboardText(string(runes))
					}
				case KeyV:
					if !st.ReadOnly {
						if clip, err := c.Win.ClipboardText(); err == nil && clip != "" {
							clip = strings.ReplaceAll(clip, "\r\n", "\n")
							clip = strings.ReplaceAll(clip, "\r", "\n")
							ins := []rune(clip)
							if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
								runes, caret = deleteRuneRange(runes, a, b)
							}
							next := make([]rune, 0, len(runes)+len(ins))
							next = append(next, runes[:caret]...)
							next = append(next, ins...)
							next = append(next, runes[caret:]...)
							runes = next
							caret += len(ins)
							selStart = caret
							changed = true
						}
					}
				}
				continue
			}

			switch kp.Code {
			case KeyEnter:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						runes, caret = deleteRuneRange(runes, a, b)
					}
					next := make([]rune, 0, len(runes)+1)
					next = append(next, runes[:caret]...)
					next = append(next, '\n')
					next = append(next, runes[caret:]...)
					runes = next
					caret++
					selStart = caret
					changed = true
				}
			case KeyBackspace:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						runes, caret = deleteRuneRange(runes, a, b)
						selStart = caret
						changed = true
					} else if caret > 0 {
						runes = append(runes[:caret-1], runes[caret:]...)
						caret--
						selStart = caret
						changed = true
					}
				}
			case KeyDelete:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						runes, caret = deleteRuneRange(runes, a, b)
						selStart = caret
						changed = true
					} else if caret < len(runes) {
						runes = append(runes[:caret], runes[caret+1:]...)
						selStart = caret
						changed = true
					}
				}
			case KeyLeft:
				if caret > 0 {
					caret--
				}
				if !kp.Mods.Shift {
					selStart = caret
				}
			case KeyRight:
				if caret < len(runes) {
					caret++
				}
				if !kp.Mods.Shift {
					selStart = caret
				}
			case KeyEscape:
				c.FocusID = ""
				focused = false
			}
		}

		if !st.ReadOnly {
			for _, ch := range c.TypedRunes {
				if ch < 32 {
					continue
				}
				if st.InputFilter != nil && !st.InputFilter(ch) {
					continue
				}
				if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
					runes, caret = deleteRuneRange(runes, a, b)
				}
				next := make([]rune, 0, len(runes)+1)
				next = append(next, runes[:caret]...)
				next = append(next, ch)
				next = append(next, runes[caret:]...)
				runes = next
				caret++
				selStart = caret
				changed = true
			}
		}

		c.caretPos[id] = caret
		c.textSelStart[id] = selStart
		if changed {
			*value = string(runes)
			lines = c.WrapTextLines(runes, textArea.W-c.S(6), pxSize, st.FontStyle)
			c.RequestRedraw()
		}
	}

	bg := c.Theme.SurfaceDark
	if st.Bg != nil {
		bg = *st.Bg
	}
	borderCol := c.Theme.Border
	if focused {
		borderCol = c.Theme.BorderFocus
	}
	rad := c.Theme.RadiusSm
	if st.Radius != nil {
		rad = *st.Radius
	}
	c.Canvas.DrawRoundedBox(r, c.Sf(rad), bg, 1.0, borderCol)

	if textArea.W <= 0 || textArea.H <= 0 || c.Font == nil {
		return changed
	}

	c.Canvas.PushClip(textArea)
	scroll := c.textScroll[id]
	if len(runes) == 0 && !focused && placeholder != "" {
		c.Font.DrawTextStyled(c.Canvas, textArea.X, textArea.Y, placeholder, pxSize, c.Theme.TextMuted, st.FontStyle)
	} else {
		textCol := c.Theme.TextPrimary
		if st.Color != nil {
			textCol = *st.Color
		}
		if st.Disabled {
			textCol = c.Theme.TextMuted
		}

		caret := clampRuneIdx(c.caretPos[id], len(runes))
		for li, sp := range lines {
			y := textArea.Y + li*lineStep - scroll
			if y+lineStep < textArea.Y || y > textArea.Y+textArea.H {
				continue
			}
			lineStr := string(runes[sp.start:sp.end])
			c.Font.DrawTextStyled(c.Canvas, textArea.X, y, lineStr, pxSize, textCol, st.FontStyle)

			if focused && caret >= sp.start && caret <= sp.end {
				cxOff, _ := c.Font.MeasureTextStyled(string(runes[sp.start:caret]), pxSize, st.FontStyle)
				c.Canvas.FillRect(render.Rect{
					X: textArea.X + cxOff,
					Y: y,
					W: c.S(2),
					H: lineH,
				}, c.Theme.Accent)
			}
		}
	}
	c.Canvas.PopClip()

	return changed
}

