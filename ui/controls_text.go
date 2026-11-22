package ui

import (
	"fmt"
	"strings"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/render"
)

const (
	svgEye    = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>`
	svgEyeOff = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>`
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
	if c.passwordVisible == nil {
		c.passwordVisible = make(map[string]bool)
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
func (c *Context) TextInput(id string, value *string, placeholder string, widthDp Dp) bool {
	if widthDp > 0 {
		return c.TextInputOpts(id, value, placeholder, Width(widthDp))
	}
	return c.TextInputOpts(id, value, placeholder)
}

// TextInputOpts рисует однострочное поле ввода текста с поддержкой выделения, курсора мышью,
// буфера обмена, режима пароля, иконок, истории правок, контекстного меню и ошибки валидации.
func (c *Context) TextInputOpts(id string, value *string, placeholder string, opts ...Option) bool {
	if value == nil {
		return false
	}
	id = c.ResolveID("inp", id)
	c.ensureTextEditingMaps()
	st := ApplyOptions(opts)

	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}
	h := c.Scale(38).Int()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	}
	w, h = c.applySizeConstraints(w, h, st)
	errH := c.inputErrorExtraH(st.ErrorText)
	r := c.allocRect(w, h+errH)
	r.H = h
	if !st.Disabled {
		c.registerFocusable(id)
	}

	padX := c.Scale(12).Int()
	if st.PadDp != nil && *st.PadDp >= 0 {
		padX = c.Scale(*st.PadDp).Int()
	}
	iconSize := c.Scale(16).Int()
	iconGap := c.Scale(8).Int()

	leftOffset := padX
	if st.IconSVG != "" {
		leftOffset += iconSize + iconGap
	}
	passVisible := false
	if st.PasswordToggle {
		if st.PasswordVisible != nil {
			passVisible = *st.PasswordVisible
		} else if c.passwordVisible != nil {
			passVisible = c.passwordVisible[id]
		}
	}
	isMasked := (st.Password || st.PasswordToggle) && !passVisible

	rightOffset := padX
	showClear := st.Clearable && !st.Disabled && !st.ReadOnly && len(*value) > 0
	if st.PasswordToggle {
		rightOffset += iconSize + iconGap
	}
	if showClear {
		rightOffset += iconSize + iconGap
	} else if !st.PasswordToggle && st.IconRightSVG != "" {
		rightOffset += iconSize + iconGap
	}

	textArea := render.Rect{
		X: r.X + leftOffset,
		Y: r.Y + c.Scale(2).Int(),
		W: r.W - leftOffset - rightOffset,
		H: r.H - c.Scale(4).Int(),
	}

	fontSizeDp := Dp(14)
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		fontSizeDp = *st.FontSizeDp
	}
	pxSize := c.Scale(fontSizeDp).Int()

	runes := []rune(*value)
	displayRunes := runes
	if isMasked {
		displayRunes = maskRunes(runes)
	}

	changed := false
	if c.textMenuChanged[id] {
		changed = true
		delete(c.textMenuChanged, id)
	}
	if !st.Disabled {
		c.RegisterHit(id, r, platform.CursorIBeam)
	}
	eyeClicked := false
	if st.PasswordToggle && !st.Disabled {
		eyeRect := render.Rect{
			X: r.X + r.W - padX - iconSize,
			Y: r.Y + (r.H-iconSize)/2,
			W: iconSize,
			H: iconSize,
		}
		c.RegisterHit(id+":eye", eyeRect, platform.CursorHand)
		if !c.IsInputBlocked() && eyeRect.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorHand
			if c.MousePressed {
				if st.PasswordVisible != nil {
					*st.PasswordVisible = !*st.PasswordVisible
				} else {
					c.passwordVisible[id] = !c.passwordVisible[id]
				}
				eyeClicked = true
				c.FocusID = id
				c.RequestRedraw()
			}
		}
	}
	clearClicked := false
	if showClear {
		clearX := r.X + r.W - padX - iconSize
		if st.PasswordToggle {
			clearX -= iconSize + iconGap
		}
		clearRect := render.Rect{
			X: clearX,
			Y: r.Y + (r.H-iconSize)/2,
			W: iconSize,
			H: iconSize,
		}
		if !st.Disabled {
			c.RegisterHit(id+":clear", clearRect, platform.CursorHand)
		}
		if !c.IsInputBlocked() && clearRect.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorHand
			if c.MousePressed {
				c.pushTextUndo(id, *value, c.caretPos[id], c.textSelStart[id])
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
	if hovered && !clearClicked && !eyeClicked {
		c.HotID = id
		if textArea.Contains(c.MouseX, c.MouseY) {
			c.nextCursor = platform.CursorIBeam
		}
		if c.MousePressed {
			c.ActiveID = id
			c.FocusID = id
			alignOff := c.textAlignOffset(displayRunes, textArea.W, pxSize, c.textScroll[id], st)
			idx := c.CaretIndexAtX(displayRunes, c.MouseX-(textArea.X+alignOff)+c.textScroll[id], pxSize, st.FontStyle)
			cc := c.ClickCount()
			if cc == 2 {
				if isMasked || len(runes) == 0 {
					c.textSelStart[id] = 0
					c.caretPos[id] = len(runes)
				} else {
					a, b := wordBoundsAt(runes, idx)
					c.textSelStart[id] = a
					c.caretPos[id] = b
				}
			} else if cc == 3 {
				c.textSelStart[id] = 0
				c.caretPos[id] = len(runes)
			} else {
				c.caretPos[id] = idx
				c.textSelStart[id] = idx
			}
		}
	}

	// Выделение текста протяжкой мыши
	if !st.Disabled && c.ActiveID == id && c.MouseDown && !c.MousePressed && !clearClicked && !eyeClicked {
		alignOff := c.textAlignOffset(displayRunes, textArea.W, pxSize, c.textScroll[id], st)
		idx := c.CaretIndexAtX(displayRunes, c.MouseX-(textArea.X+alignOff)+c.textScroll[id], pxSize, st.FontStyle)
		c.caretPos[id] = idx
	}

	focused := !st.Disabled && c.FocusID == id
	if focused && !clearClicked && !eyeClicked {
		runes = []rune(*value)
		caret := clampRuneIdx(c.caretPos[id], len(runes))
		selStart := clampRuneIdx(c.textSelStart[id], len(runes))
		submitted := false

		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl {
				switch kp.Code {
				case KeyA:
					selStart = 0
					caret = len(runes)
				case KeyC:
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						_ = c.Win.SetClipboardText(string(runes[a:b]))
					} else if !isMasked {
						_ = c.Win.SetClipboardText(string(runes))
					}
				case KeyX:
					if !st.ReadOnly {
						if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
							c.pushTextUndo(id, string(runes), caret, selStart)
							if !isMasked {
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
							if len(ins) > 0 {
								c.pushTextUndo(id, string(runes), caret, selStart)
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
				case KeyZ:
					if !st.ReadOnly {
						var nextText string
						var ok bool
						if kp.Mods.Shift {
							nextText, caret, selStart, ok = c.redoText(id, string(runes), caret, selStart)
						} else {
							nextText, caret, selStart, ok = c.undoText(id, string(runes), caret, selStart)
						}
						if ok {
							runes = []rune(nextText)
							changed = true
						}
					}
				case KeyY:
					if !st.ReadOnly {
						if nextText, nc, ns, ok := c.redoText(id, string(runes), caret, selStart); ok {
							runes = []rune(nextText)
							caret, selStart = nc, ns
							changed = true
						}
					}
				case KeyLeft:
					if isMasked {
						caret = 0
					} else {
						caret = prevWordStart(runes, caret)
					}
					if !kp.Mods.Shift {
						selStart = caret
					}
				case KeyRight:
					if isMasked {
						caret = len(runes)
					} else {
						caret = nextWordEnd(runes, caret)
					}
					if !kp.Mods.Shift {
						selStart = caret
					}
				case KeyBackspace:
					if !st.ReadOnly {
						if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
							c.pushTextUndo(id, string(runes), caret, selStart)
							runes, caret = deleteRuneRange(runes, a, b)
							selStart = caret
							changed = true
						} else if caret > 0 {
							c.pushTextUndo(id, string(runes), caret, selStart)
							start := 0
							if !isMasked {
								start = prevWordStart(runes, caret)
							}
							runes, caret = deleteRuneRange(runes, start, caret)
							selStart = caret
							changed = true
						}
					}
				case KeyDelete:
					if !st.ReadOnly {
						if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
							c.pushTextUndo(id, string(runes), caret, selStart)
							runes, caret = deleteRuneRange(runes, a, b)
							selStart = caret
							changed = true
						} else if caret < len(runes) {
							c.pushTextUndo(id, string(runes), caret, selStart)
							end := len(runes)
							if !isMasked {
								end = nextWordEnd(runes, caret)
							}
							runes, caret = deleteRuneRange(runes, caret, end)
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
						c.pushTextUndo(id, string(runes), caret, selStart)
						runes, caret = deleteRuneRange(runes, a, b)
						selStart = caret
						changed = true
					} else if caret > 0 && len(runes) > 0 {
						c.pushTextUndo(id, string(runes), caret, selStart)
						runes = append(runes[:caret-1], runes[caret:]...)
						caret--
						selStart = caret
						changed = true
					}
				}
			case KeyDelete:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						c.pushTextUndo(id, string(runes), caret, selStart)
						runes, caret = deleteRuneRange(runes, a, b)
						selStart = caret
						changed = true
					} else if caret < len(runes) {
						c.pushTextUndo(id, string(runes), caret, selStart)
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
			case KeyEscape:
				c.FocusID = ""
				focused = false
			case KeyEnter:
				c.FocusID = ""
				focused = false
				submitted = true
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
				c.pushTextUndo(id, string(runes), caret, selStart)
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
			if isMasked {
				displayRunes = maskRunes(runes)
			}
			c.RequestRedraw()
		}
		if submitted {
			if st.OnSubmit != nil {
				st.OnSubmit()
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
	focusBorder := c.Theme.BorderFocus
	if st.FocusBorderColor != nil {
		focusBorder = *st.FocusBorderColor
	}
	borderCol := render.LerpColor(baseBorder, focusBorder, focusAnim)
	if st.ErrorText != "" {
		borderCol = c.Theme.Danger
	}

	bg := c.Theme.SurfaceDark
	if st.Bg != nil {
		bg = *st.Bg
	}
	rad := c.Theme.RadiusSm
	if st.Radius != nil {
		rad = *st.Radius
	}

	c.Canvas.DrawRoundedBox(r, c.ScaleF(rad), bg, 1.0, borderCol)

	if st.IconSVG != "" {
		c.Canvas.DrawSVG(st.IconSVG, r.X+padX, r.Y+(r.H-iconSize)/2, iconSize, c.Theme.TextMuted)
	}
	if st.PasswordToggle {
		eyeX := r.X + r.W - padX - iconSize
		eyeY := r.Y + (r.H-iconSize)/2
		eyeSVG := svgEye
		if passVisible {
			eyeSVG = svgEyeOff
		}
		eyeCol := c.Theme.TextMuted
		eyeRect := render.Rect{X: eyeX, Y: eyeY, W: iconSize, H: iconSize}
		if !c.IsInputBlocked() && eyeRect.Contains(c.MouseX, c.MouseY) {
			eyeCol = c.Theme.TextPrimary
		}
		c.Canvas.DrawSVG(eyeSVG, eyeX, eyeY, iconSize, eyeCol)
	}
	if showClear {
		clearX := r.X + r.W - padX - iconSize
		if st.PasswordToggle {
			clearX -= iconSize + iconGap
		}
		cx := float32(clearX)
		cy := float32(r.Y + (r.H-iconSize)/2)
		pad := float32(iconSize) * 0.28
		c.Canvas.DrawLine(cx+pad, cy+pad, cx+float32(iconSize)-pad, cy+float32(iconSize)-pad, c.ScaleF(1.5), c.Theme.TextMuted)
		c.Canvas.DrawLine(cx+float32(iconSize)-pad, cy+pad, cx+pad, cy+float32(iconSize)-pad, c.ScaleF(1.5), c.Theme.TextMuted)
	} else if !st.PasswordToggle && st.IconRightSVG != "" {
		c.Canvas.DrawSVG(st.IconRightSVG, r.X+r.W-padX-iconSize, r.Y+(r.H-iconSize)/2, iconSize, c.Theme.TextMuted)
	}

	menuSt := st
	menuSt.Password = isMasked
	if textArea.W <= 0 || c.Font == nil {
		c.drawInputError(r, st.ErrorText)
		c.textFieldContextMenu(id, r, value, menuSt, false)
		return changed
	}

	c.Canvas.PushClip(textArea)

	caret := clampRuneIdx(c.caretPos[id], len(displayRunes))
	selStart := clampRuneIdx(c.textSelStart[id], len(displayRunes))

	prefixW, _ := c.Font.MeasureTextStyled(string(displayRunes[:caret]), pxSize, st.FontStyle)
	scroll := c.textScroll[id]
	if prefixW-scroll > textArea.W-c.Scale(4).Int() {
		scroll = prefixW - (textArea.W - c.Scale(4).Int())
	}
	if prefixW-scroll < 0 {
		scroll = prefixW
	}
	if scroll < 0 {
		scroll = 0
	}
	c.textScroll[id] = scroll
	alignOff := c.textAlignOffset(displayRunes, textArea.W, pxSize, scroll, st)
	baseX := textArea.X + alignOff - scroll

	if focused {
		if a, b, ok := selectionBounds(caret, selStart, len(displayRunes)); ok {
			x1, _ := c.Font.MeasureTextStyled(string(displayRunes[:a]), pxSize, st.FontStyle)
			x2, _ := c.Font.MeasureTextStyled(string(displayRunes[:b]), pxSize, st.FontStyle)
			selRect := render.Rect{
				X: baseX + x1,
				Y: textArea.Y + c.Scale(5).Int(),
				W: x2 - x1,
				H: textArea.H - c.Scale(10).Int(),
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

	textY := c.Font.CenterTextY(textArea.Y, textArea.H, pxSize, st.FontStyle)
	if len(displayRunes) == 0 && !focused && placeholder != "" {
		phX := textArea.X
		if st.TextAlign != AlignTextLeft {
			pw, _ := c.Font.MeasureTextStyled(placeholder, pxSize, st.FontStyle)
			if st.TextAlign == AlignTextCenter && textArea.W > pw {
				phX = textArea.X + (textArea.W-pw)/2
			} else if st.TextAlign == AlignTextRight && textArea.W > pw {
				phX = textArea.X + textArea.W - pw
			}
		}
		c.Font.DrawTextStyled(c.Canvas, phX, textY, placeholder, pxSize, c.Theme.TextMuted, st.FontStyle)
	} else {
		c.Font.DrawTextStyled(c.Canvas, baseX, textY, string(displayRunes), pxSize, textCol, st.FontStyle)
	}

	if focused {
		caretX := baseX + prefixW
		caretTop := textArea.Y + c.Scale(6).Int()
		caretH := textArea.H - c.Scale(12).Int()
		c.Canvas.FillRect(render.Rect{X: caretX, Y: caretTop, W: c.Scale(2).Int(), H: caretH}, c.Theme.Accent)
	}

	c.Canvas.PopClip()
	c.drawInputError(r, st.ErrorText)
	c.textFieldContextMenu(id, r, value, menuSt, false)
	return changed
}

func (c *Context) textAlignOffset(displayRunes []rune, areaW, pxSize, scroll int, st WidgetStyle) int {
	if st.TextAlign == AlignTextLeft || scroll > 0 || c.Font == nil || areaW <= 0 {
		return 0
	}
	fullW, _ := c.Font.MeasureTextStyled(string(displayRunes), pxSize, st.FontStyle)
	if fullW >= areaW {
		return 0
	}
	if st.TextAlign == AlignTextCenter {
		return (areaW - fullW) / 2
	}
	return areaW - fullW
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
			if end < rl.end {
				breakPos := -1
				for k := end - 1; k >= cur; k-- {
					if runes[k] == ' ' || runes[k] == '\t' {
						breakPos = k
						break
					}
				}
				if breakPos >= cur {
					end = breakPos + 1
				}
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

// findVisualLine возвращает индекс визуальной строки, содержащей позицию caret.
func findVisualLine(lines []textLineSpan, runes []rune, caret int) int {
	for i, sp := range lines {
		if caret >= sp.start && caret <= sp.end {
			if caret == sp.end && i+1 < len(lines) && lines[i+1].start == caret && (caret == 0 || runes[caret-1] != '\n') {
				return i + 1
			}
			return i
		}
	}
	if len(lines) > 0 {
		return len(lines) - 1
	}
	return 0
}

// TextArea рисует многострочное поле ввода текста с переносами, вертикальным скроллом,
// выделением, навигацией по строкам и словам, историей правок, нумерацией строк и контекстным меню.
func (c *Context) TextArea(id string, value *string, placeholder string, opts ...Option) bool {
	if value == nil {
		return false
	}
	id = c.ResolveID("ta", id)
	c.ensureTextEditingMaps()
	st := ApplyOptions(opts)

	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.Scale(*st.WidthDp).Int()
	}
	h := c.Scale(120).Int()
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.Scale(*st.HeightDp).Int()
	}
	w, h = c.applySizeConstraints(w, h, st)
	errH := c.inputErrorExtraH(st.ErrorText)
	r := c.allocRect(w, h+errH)
	r.H = h
	if !st.Disabled {
		c.registerFocusable(id)
	}

	pad := c.Scale(10).Int()
	if st.PadDp != nil && *st.PadDp >= 0 {
		pad = c.Scale(*st.PadDp).Int()
	}

	fontSizeDp := Dp(14)
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		fontSizeDp = *st.FontSizeDp
	}
	pxSize := c.Scale(fontSizeDp).Int()
	_, lineH := 0, c.Scale(18).Int()
	if c.Font != nil {
		_, lineH = c.Font.MeasureTextStyled("Аy", pxSize, st.FontStyle)
		if lineH <= 0 {
			lineH = c.Scale(18).Int()
		}
	}
	lineStep := lineH + c.Scale(4).Int()

	runes := []rune(*value)
	gutterW := 0
	if st.LineNumbers {
		logicalCount := 1
		for _, ch := range runes {
			if ch == '\n' {
				logicalCount++
			}
		}
		sample := "99"
		if logicalCount >= 100 {
			sample = fmt.Sprintf("%d", logicalCount)
		}
		numW := c.Scale(24).Int()
		if c.Font != nil {
			mw, _ := c.Font.MeasureTextStyled(sample, c.Scale(fontSizeDp-1).Int(), st.FontStyle)
			numW = mw
		}
		gutterW = numW + c.Scale(16).Int()
	}

	textArea := render.Rect{
		X: r.X + pad + gutterW,
		Y: r.Y + pad,
		W: r.W - pad*2 - gutterW,
		H: r.H - pad*2,
	}

	lines := c.WrapTextLines(runes, textArea.W-c.Scale(6).Int(), pxSize, st.FontStyle)

	caretFromPoint := func(mx, my int) int {
		lineIdx := (my - textArea.Y + c.textScroll[id]) / lineStep
		if lineIdx < 0 {
			lineIdx = 0
		}
		if lineIdx >= len(lines) {
			lineIdx = len(lines) - 1
		}
		sp := lines[lineIdx]
		col := c.CaretIndexAtX(runes[sp.start:sp.end], mx-textArea.X, pxSize, st.FontStyle)
		return sp.start + col
	}

	if !st.Disabled {
		c.RegisterHit(id, r, platform.CursorIBeam)
	}
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
			idx := caretFromPoint(c.MouseX, c.MouseY)
			cc := c.ClickCount()
			if cc == 2 {
				a, b := wordBoundsAt(runes, idx)
				c.textSelStart[id] = a
				c.caretPos[id] = b
			} else if cc == 3 {
				a, b := lineBoundsAt(runes, idx)
				c.textSelStart[id] = a
				c.caretPos[id] = b
			} else {
				c.caretPos[id] = idx
				c.textSelStart[id] = idx
			}
			c.textPrefX[id] = -1
		}
	}

	// Выделение текста протяжкой мыши по нескольким строкам
	if !st.Disabled && c.ActiveID == id && c.MouseDown && !c.MousePressed {
		c.caretPos[id] = caretFromPoint(c.MouseX, c.MouseY)
		c.textPrefX[id] = -1
	}

	changed := false
	if c.textMenuChanged[id] {
		changed = true
		delete(c.textMenuChanged, id)
	}
	focused := !st.Disabled && c.FocusID == id
	if focused {
		caret := clampRuneIdx(c.caretPos[id], len(runes))
		selStart := clampRuneIdx(c.textSelStart[id], len(runes))
		hadKeyOrRune := len(c.KeyEvents) > 0 || len(c.TypedRunes) > 0

		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl {
				switch kp.Code {
				case KeyA:
					selStart = 0
					caret = len(runes)
					c.textPrefX[id] = -1
				case KeyC:
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						_ = c.Win.SetClipboardText(string(runes[a:b]))
					} else {
						_ = c.Win.SetClipboardText(string(runes))
					}
				case KeyX:
					if !st.ReadOnly {
						if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
							c.pushTextUndo(id, string(runes), caret, selStart)
							_ = c.Win.SetClipboardText(string(runes[a:b]))
							runes, caret = deleteRuneRange(runes, a, b)
							selStart = caret
							c.textPrefX[id] = -1
							changed = true
						}
					}
				case KeyV:
					if !st.ReadOnly {
						if clip, err := c.Win.ClipboardText(); err == nil && clip != "" {
							clip = strings.ReplaceAll(clip, "\r\n", "\n")
							clip = strings.ReplaceAll(clip, "\r", "\n")
							var ins []rune
							for _, ch := range clip {
								if st.InputFilter == nil || st.InputFilter(ch) {
									ins = append(ins, ch)
								}
							}
							if len(ins) > 0 {
								c.pushTextUndo(id, string(runes), caret, selStart)
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
								c.textPrefX[id] = -1
								changed = true
							}
						}
					}
				case KeyZ:
					if !st.ReadOnly {
						var nextText string
						var ok bool
						if kp.Mods.Shift {
							nextText, caret, selStart, ok = c.redoText(id, string(runes), caret, selStart)
						} else {
							nextText, caret, selStart, ok = c.undoText(id, string(runes), caret, selStart)
						}
						if ok {
							runes = []rune(nextText)
							c.textPrefX[id] = -1
							changed = true
						}
					}
				case KeyY:
					if !st.ReadOnly {
						if nextText, nc, ns, ok := c.redoText(id, string(runes), caret, selStart); ok {
							runes = []rune(nextText)
							caret, selStart = nc, ns
							c.textPrefX[id] = -1
							changed = true
						}
					}
				case KeyLeft:
					caret = prevWordStart(runes, caret)
					if !kp.Mods.Shift {
						selStart = caret
					}
					c.textPrefX[id] = -1
				case KeyRight:
					caret = nextWordEnd(runes, caret)
					if !kp.Mods.Shift {
						selStart = caret
					}
					c.textPrefX[id] = -1
				case KeyBackspace:
					if !st.ReadOnly {
						if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
							c.pushTextUndo(id, string(runes), caret, selStart)
							runes, caret = deleteRuneRange(runes, a, b)
							selStart = caret
							changed = true
						} else if caret > 0 {
							c.pushTextUndo(id, string(runes), caret, selStart)
							start := prevWordStart(runes, caret)
							runes, caret = deleteRuneRange(runes, start, caret)
							selStart = caret
							changed = true
						}
						c.textPrefX[id] = -1
					}
				case KeyDelete:
					if !st.ReadOnly {
						if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
							c.pushTextUndo(id, string(runes), caret, selStart)
							runes, caret = deleteRuneRange(runes, a, b)
							selStart = caret
							changed = true
						} else if caret < len(runes) {
							c.pushTextUndo(id, string(runes), caret, selStart)
							end := nextWordEnd(runes, caret)
							runes, caret = deleteRuneRange(runes, caret, end)
							selStart = caret
							changed = true
						}
						c.textPrefX[id] = -1
					}
				case KeyHome:
					caret = 0
					if !kp.Mods.Shift {
						selStart = caret
					}
					c.textPrefX[id] = -1
				case KeyEnd:
					caret = len(runes)
					if !kp.Mods.Shift {
						selStart = caret
					}
					c.textPrefX[id] = -1
				}
				continue
			}

			switch kp.Code {
			case KeyEnter:
				if !st.ReadOnly {
					c.pushTextUndo(id, string(runes), caret, selStart)
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
					c.textPrefX[id] = -1
					changed = true
				}
			case KeyBackspace:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						c.pushTextUndo(id, string(runes), caret, selStart)
						runes, caret = deleteRuneRange(runes, a, b)
						selStart = caret
						changed = true
					} else if caret > 0 {
						c.pushTextUndo(id, string(runes), caret, selStart)
						runes = append(runes[:caret-1], runes[caret:]...)
						caret--
						selStart = caret
						changed = true
					}
					c.textPrefX[id] = -1
				}
			case KeyDelete:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
						c.pushTextUndo(id, string(runes), caret, selStart)
						runes, caret = deleteRuneRange(runes, a, b)
						selStart = caret
						changed = true
					} else if caret < len(runes) {
						c.pushTextUndo(id, string(runes), caret, selStart)
						runes = append(runes[:caret], runes[caret+1:]...)
						selStart = caret
						changed = true
					}
					c.textPrefX[id] = -1
				}
			case KeyLeft:
				if caret > 0 {
					caret--
				}
				if !kp.Mods.Shift {
					selStart = caret
				}
				c.textPrefX[id] = -1
			case KeyRight:
				if caret < len(runes) {
					caret++
				}
				if !kp.Mods.Shift {
					selStart = caret
				}
				c.textPrefX[id] = -1
			case KeyUp, KeyDown:
				curLines := lines
				if changed {
					curLines = c.WrapTextLines(runes, textArea.W-c.Scale(6).Int(), pxSize, st.FontStyle)
				}
				li := findVisualLine(curLines, runes, caret)
				prefX := c.textPrefX[id]
				if prefX < 0 && c.Font != nil {
					sp := curLines[li]
					prefX, _ = c.Font.MeasureTextStyled(string(runes[sp.start:caret]), pxSize, st.FontStyle)
					c.textPrefX[id] = prefX
				}
				if kp.Code == KeyUp && li > 0 {
					target := curLines[li-1]
					col := c.CaretIndexAtX(runes[target.start:target.end], prefX, pxSize, st.FontStyle)
					caret = target.start + col
				} else if kp.Code == KeyDown && li+1 < len(curLines) {
					target := curLines[li+1]
					col := c.CaretIndexAtX(runes[target.start:target.end], prefX, pxSize, st.FontStyle)
					caret = target.start + col
				}
				if !kp.Mods.Shift {
					selStart = caret
				}
			case KeyHome:
				curLines := lines
				if changed {
					curLines = c.WrapTextLines(runes, textArea.W-c.Scale(6).Int(), pxSize, st.FontStyle)
				}
				li := findVisualLine(curLines, runes, caret)
				caret = curLines[li].start
				if !kp.Mods.Shift {
					selStart = caret
				}
				c.textPrefX[id] = -1
			case KeyEnd:
				curLines := lines
				if changed {
					curLines = c.WrapTextLines(runes, textArea.W-c.Scale(6).Int(), pxSize, st.FontStyle)
				}
				li := findVisualLine(curLines, runes, caret)
				caret = curLines[li].end
				if !kp.Mods.Shift {
					selStart = caret
				}
				c.textPrefX[id] = -1
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
				c.pushTextUndo(id, string(runes), caret, selStart)
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
				c.textPrefX[id] = -1
				changed = true
			}
		}

		c.caretPos[id] = caret
		c.textSelStart[id] = selStart
		if changed {
			*value = string(runes)
			lines = c.WrapTextLines(runes, textArea.W-c.Scale(6).Int(), pxSize, st.FontStyle)
			c.RequestRedraw()
		}

		if hadKeyOrRune || changed {
			li := findVisualLine(lines, runes, caret)
			caretY := li * lineStep
			sc := c.textScroll[id]
			if caretY < sc {
				sc = caretY
			}
			if textArea.H > 0 && caretY+lineStep > sc+textArea.H {
				sc = caretY + lineStep - textArea.H
			}
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
	}

	bg := c.Theme.SurfaceDark
	if st.Bg != nil {
		bg = *st.Bg
	}
	borderCol := c.Theme.Border
	if st.BorderColor != nil {
		borderCol = *st.BorderColor
	}
	if focused {
		borderCol = c.Theme.BorderFocus
		if st.FocusBorderColor != nil {
			borderCol = *st.FocusBorderColor
		}
	}
	if st.ErrorText != "" {
		borderCol = c.Theme.Danger
	}
	rad := c.Theme.RadiusSm
	if st.Radius != nil {
		rad = *st.Radius
	}
	c.Canvas.DrawRoundedBox(r, c.ScaleF(rad), bg, 1.0, borderCol)

	if textArea.W <= 0 || textArea.H <= 0 || c.Font == nil {
		c.drawInputError(r, st.ErrorText)
		c.textFieldContextMenu(id, r, value, st, true)
		return changed
	}

	scroll := c.textScroll[id]
	if st.LineNumbers && gutterW > 0 {
		gutterRect := render.Rect{X: r.X + 1, Y: r.Y + 1, W: gutterW + pad - c.Scale(4).Int(), H: r.H - 2}
		c.Canvas.PushClip(gutterRect)
		c.Canvas.FillRect(gutterRect, c.Theme.Surface.WithAlpha(120))
		c.Canvas.FillRect(render.Rect{X: gutterRect.X + gutterRect.W - 1, Y: gutterRect.Y, W: 1, H: gutterRect.H}, c.Theme.Border)
		numPx := c.Scale(fontSizeDp - 1).Int()
		if numPx < c.Scale(10).Int() {
			numPx = c.Scale(10).Int()
		}
		logicalNum := 0
		for li, sp := range lines {
			isLogicalStart := li == 0 || (sp.start > 0 && runes[sp.start-1] == '\n')
			if isLogicalStart {
				logicalNum++
			}
			y := textArea.Y + li*lineStep - scroll
			if !isLogicalStart || y+lineStep < textArea.Y || y > textArea.Y+textArea.H {
				continue
			}
			numStr := fmt.Sprintf("%d", logicalNum)
			nw, _ := c.Font.MeasureTextStyled(numStr, numPx, st.FontStyle)
			c.Font.DrawTextStyled(c.Canvas, gutterRect.X+gutterRect.W-nw-c.Scale(6).Int(), y+c.Scale(1).Int(), numStr, numPx, c.Theme.TextMuted, st.FontStyle)
		}
		c.Canvas.PopClip()
	}

	c.Canvas.PushClip(textArea)
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
		selStart := clampRuneIdx(c.textSelStart[id], len(runes))
		selA, selB, hasSel := selectionBounds(caret, selStart, len(runes))
		caretLine := findVisualLine(lines, runes, caret)

		for li, sp := range lines {
			y := textArea.Y + li*lineStep - scroll
			if y+lineStep < textArea.Y || y > textArea.Y+textArea.H {
				continue
			}

			if focused && hasSel && selB > sp.start && selA < sp.end {
				a := selA
				if a < sp.start {
					a = sp.start
				}
				b := selB
				if b > sp.end {
					b = sp.end
				}
				if b > a {
					x1, _ := c.Font.MeasureTextStyled(string(runes[sp.start:a]), pxSize, st.FontStyle)
					x2, _ := c.Font.MeasureTextStyled(string(runes[sp.start:b]), pxSize, st.FontStyle)
					c.Canvas.FillRect(render.Rect{
						X: textArea.X + x1,
						Y: y,
						W: x2 - x1,
						H: lineH,
					}, c.Theme.Accent.WithAlpha(90))
				}
			} else if focused && hasSel && sp.start == sp.end && selA <= sp.start && selB > sp.start {
				c.Canvas.FillRect(render.Rect{
					X: textArea.X,
					Y: y,
					W: c.Scale(6).Int(),
					H: lineH,
				}, c.Theme.Accent.WithAlpha(90))
			}

			lineStr := string(runes[sp.start:sp.end])
			c.Font.DrawTextStyled(c.Canvas, textArea.X, y, lineStr, pxSize, textCol, st.FontStyle)

			if focused && li == caretLine {
				cxOff, _ := c.Font.MeasureTextStyled(string(runes[sp.start:caret]), pxSize, st.FontStyle)
				c.Canvas.FillRect(render.Rect{
					X: textArea.X + cxOff,
					Y: y,
					W: c.Scale(2).Int(),
					H: lineH,
				}, c.Theme.Accent)
			}
		}
	}
	c.Canvas.PopClip()

	c.drawInputError(r, st.ErrorText)
	c.textFieldContextMenu(id, r, value, st, true)
	return changed
}
