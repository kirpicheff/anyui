package ui

import (
	"strings"
	"unicode"

	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/render"
)

// SelectableLabel отображает статический текст, который можно выделить мышью и скопировать
// (Ctrl+C, Ctrl+A, двойной/тройной клик, встроенное контекстное меню).
func (c *Context) SelectableLabel(id, text string, opts ...Option) {
	id = c.ResolveID("slbl", id)
	c.ensureTextEditingMaps()
	st := ApplyOptions(opts)

	sizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		sizeDp = *st.FontSizeDp
	}
	pxSize := c.S(sizeDp)
	lineH := c.S(18)
	if c.Font != nil {
		if _, mh := c.Font.MeasureTextStyled("Аy", pxSize, st.FontStyle); mh > 0 {
			lineH = mh
		}
	}
	gap := c.S(3)
	lineStep := lineH + gap

	runes := []rune(text)
	maxW := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		maxW = c.S(*st.WidthDp)
	}
	if st.MaxWidthDp != nil && *st.MaxWidthDp > 0 && c.S(*st.MaxWidthDp) < maxW {
		maxW = c.S(*st.MaxWidthDp)
	}

	var lines []textLineSpan
	if st.WordWrap && maxW > 0 {
		lines = c.WrapTextLines(runes, maxW, pxSize, st.FontStyle)
	} else {
		lines = c.WrapTextLines(runes, 0, pxSize, st.FontStyle)
	}

	totalW := 0
	for _, sp := range lines {
		lw := (sp.end - sp.start) * c.S(8)
		if c.Font != nil {
			lw, _ = c.Font.MeasureTextStyled(string(runes[sp.start:sp.end]), pxSize, st.FontStyle)
		}
		if lw > totalW {
			totalW = lw
		}
	}
	if st.WidthDp != nil && *st.WidthDp > 0 {
		totalW = c.S(*st.WidthDp)
	}
	totalH := lineH
	if len(lines) > 1 {
		totalH = len(lines)*lineStep - gap
	}

	r := c.allocRectStyled(totalW, totalH, st)
	if !st.Disabled {
		c.registerFocusable(id)
	}

	caretFromPoint := func(mx, my int) int {
		lineIdx := (my - r.Y) / lineStep
		if lineIdx < 0 {
			lineIdx = 0
		}
		if lineIdx >= len(lines) {
			lineIdx = len(lines) - 1
		}
		sp := lines[lineIdx]
		col := c.CaretIndexAtX(runes[sp.start:sp.end], mx-r.X, pxSize, st.FontStyle)
		return sp.start + col
	}

	if !st.Disabled {
		c.RegisterHit(id, r, platform.CursorIBeam)
	}
	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered {
		c.HotID = id
		c.nextCursor = platform.CursorIBeam
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
		}
	}

	if !st.Disabled && c.ActiveID == id && c.MouseDown && !c.MousePressed {
		c.caretPos[id] = caretFromPoint(c.MouseX, c.MouseY)
	}

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
						_ = c.Win.SetClipboardText(text)
					}
				case KeyLeft:
					caret = prevWordStart(runes, caret)
					if !kp.Mods.Shift {
						selStart = caret
					}
				case KeyRight:
					caret = nextWordEnd(runes, caret)
					if !kp.Mods.Shift {
						selStart = caret
					}
				}
				continue
			}
			switch kp.Code {
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
			}
		}
		c.caretPos[id] = caret
		c.textSelStart[id] = selStart
	}

	col := c.Theme.TextPrimary
	if st.Color != nil {
		col = *st.Color
	}
	if st.Disabled {
		col = c.Theme.TextMuted
	}

	caret := clampRuneIdx(c.caretPos[id], len(runes))
	selStart := clampRuneIdx(c.textSelStart[id], len(runes))
	selA, selB, hasSel := selectionBounds(caret, selStart, len(runes))

	if c.Font != nil {
		for i, sp := range lines {
			y := r.Y + i*lineStep
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
					c.Canvas.FillRect(render.Rect{X: r.X + x1, Y: y, W: x2 - x1, H: lineH}, c.Theme.Accent.WithAlpha(90))
				}
			}
			c.Font.DrawTextStyled(c.Canvas, r.X, y, string(runes[sp.start:sp.end]), pxSize, col, st.FontStyle)
		}
	}

	if !st.Disabled {
		c.selectableContextMenu(id, r, text)
	}
}

// Hyperlink отображает кликабельную текстовую ссылку и возвращает true при нажатии.
func (c *Context) Hyperlink(id, text string, opts ...Option) bool {
	id = c.ResolveID("link", id)
	st := ApplyOptions(opts)
	sizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		sizeDp = *st.FontSizeDp
	}
	pxSize := c.S(sizeDp)
	w, h := len([]rune(text))*c.S(8), c.S(18)
	if c.Font != nil {
		w, h = c.Font.MeasureTextStyled(text, pxSize, st.FontStyle)
	}
	r := c.allocRectStyled(w, h+c.S(2), st)
	if !st.Disabled {
		c.registerFocusable(id)
		c.RegisterHit(id, r, platform.CursorHand)
	}

	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	clicked := false
	if hovered {
		c.HotID = id
		c.nextCursor = platform.CursorHand
		if c.MousePressed {
			c.ActiveID = id
			c.FocusID = id
		}
		if c.MouseReleased && c.ActiveID == id {
			clicked = true
			c.RequestRedraw()
		}
	}
	focused := !st.Disabled && c.FocusID == id
	if focused && c.consumeEnter() {
		clicked = true
		c.RequestRedraw()
	}

	col := c.Theme.Accent
	if st.Color != nil {
		col = *st.Color
	}
	if hovered {
		col = render.LerpColor(col, render.RGB(255, 255, 255), 0.18)
	}
	if st.Disabled {
		col = c.Theme.TextMuted
	}

	if c.Font != nil {
		c.Font.DrawTextStyled(c.Canvas, r.X, r.Y, text, pxSize, col, st.FontStyle)
	}
	if !st.Disabled && (hovered || focused) {
		lineY := float32(r.Y + h)
		c.Canvas.DrawLine(float32(r.X), lineY, float32(r.X+w), lineY, c.Sf(1), col)
	}
	return clicked
}

func isMaskSlot(m rune) bool {
	return m == '0' || m == '9' || m == 'A' || m == '*'
}

func matchMaskSlot(m, ch rune) bool {
	switch m {
	case '0', '9':
		return unicode.IsDigit(ch)
	case 'A':
		return unicode.IsLetter(ch)
	case '*':
		return unicode.IsLetter(ch) || unicode.IsDigit(ch)
	default:
		return false
	}
}

func parseMaskedSlots(value string, maskRunes []rune) []rune {
	slots := make([]rune, len(maskRunes))
	for i, m := range maskRunes {
		if !isMaskSlot(m) {
			slots[i] = m
		}
	}
	valRunes := []rune(value)
	vi := 0
	for i, m := range maskRunes {
		if vi >= len(valRunes) {
			break
		}
		if !isMaskSlot(m) {
			if valRunes[vi] == m {
				vi++
			}
			continue
		}
		for vi < len(valRunes) && !matchMaskSlot(m, valRunes[vi]) {
			vi++
		}
		if vi < len(valRunes) {
			slots[i] = valRunes[vi]
			vi++
		}
	}
	return slots
}

func formatMaskedValue(slots, maskRunes []rune) string {
	lastFilled := -1
	for i, m := range maskRunes {
		if isMaskSlot(m) && slots[i] != 0 {
			lastFilled = i
		}
	}
	if lastFilled < 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i <= lastFilled; i++ {
		if isMaskSlot(maskRunes[i]) {
			if slots[i] != 0 {
				b.WriteRune(slots[i])
			} else {
				b.WriteRune('_')
			}
		} else {
			b.WriteRune(maskRunes[i])
		}
	}
	return b.String()
}

func nextMaskSlot(maskRunes []rune, pos int) int {
	for i := pos; i < len(maskRunes); i++ {
		if isMaskSlot(maskRunes[i]) {
			return i
		}
	}
	return len(maskRunes)
}

func prevMaskSlot(maskRunes []rune, pos int) int {
	for i := pos - 1; i >= 0; i-- {
		if isMaskSlot(maskRunes[i]) {
			return i
		}
	}
	return -1
}

// MaskedInput рисует однострочное поле ввода с шаблоном маски
// ('0'/'9' — цифра, 'A' — буква, '*' — буква/цифра, прочие символы — фиксированные разделители).
func (c *Context) MaskedInput(id string, value *string, mask string, opts ...Option) bool {
	if value == nil {
		return false
	}
	maskRunes := []rune(mask)
	if len(maskRunes) == 0 {
		return c.TextInputOpts(id, value, "", opts...)
	}
	id = c.ResolveID("mask", id)

	c.ensureTextEditingMaps()
	st := ApplyOptions(opts)

	w := c.RemainingWidth()
	if st.WidthDp != nil && *st.WidthDp > 0 {
		w = c.S(*st.WidthDp)
	}
	h := c.S(38)
	if st.HeightDp != nil && *st.HeightDp > 0 {
		h = c.S(*st.HeightDp)
	}
	errH := c.inputErrorExtraH(st.ErrorText)
	r := c.allocRect(w, h+errH)
	r.H = h
	if !st.Disabled {
		c.registerFocusable(id)
	}

	padX := c.S(12)
	if st.PadDp != nil && *st.PadDp >= 0 {
		padX = c.S(*st.PadDp)
	}
	textArea := render.Rect{
		X: r.X + padX,
		Y: r.Y + c.S(2),
		W: r.W - padX*2,
		H: r.H - c.S(4),
	}

	fontSizeDp := 14
	if st.FontSizeDp != nil && *st.FontSizeDp > 0 {
		fontSizeDp = *st.FontSizeDp
	}
	pxSize := c.S(fontSizeDp)

	slots := parseMaskedSlots(*value, maskRunes)
	displayRunes := make([]rune, len(maskRunes))
	for i, m := range maskRunes {
		if isMaskSlot(m) {
			if slots[i] != 0 {
				displayRunes[i] = slots[i]
			} else {
				displayRunes[i] = '_'
			}
		} else {
			displayRunes[i] = m
		}
	}

	changed := false
	if c.textMenuChanged[id] {
		changed = true
		delete(c.textMenuChanged, id)
	}

	if !st.Disabled {
		c.RegisterHit(id, r, platform.CursorIBeam)
	}
	hovered := !st.Disabled && !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY)
	if hovered {
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

	if !st.Disabled && c.ActiveID == id && c.MouseDown && !c.MousePressed {
		idx := c.CaretIndexAtX(displayRunes, c.MouseX-textArea.X+c.textScroll[id], pxSize, st.FontStyle)
		c.caretPos[id] = idx
	}

	focused := !st.Disabled && c.FocusID == id
	if focused {
		caret := clampRuneIdx(c.caretPos[id], len(maskRunes))
		selStart := clampRuneIdx(c.textSelStart[id], len(maskRunes))

		clearSlotRange := func(a, b int) {
			for i := a; i < b && i < len(maskRunes); i++ {
				if isMaskSlot(maskRunes[i]) {
					slots[i] = 0
				}
			}
		}

		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl {
				switch kp.Code {
				case KeyA:
					selStart = 0
					caret = len(maskRunes)
				case KeyC:
					if a, b, ok := selectionBounds(caret, selStart, len(displayRunes)); ok {
						_ = c.Win.SetClipboardText(string(displayRunes[a:b]))
					} else {
						_ = c.Win.SetClipboardText(*value)
					}
				case KeyZ:
					if !st.ReadOnly {
						var nextText string
						var ok bool
						if kp.Mods.Shift {
							nextText, caret, selStart, ok = c.redoText(id, *value, caret, selStart)
						} else {
							nextText, caret, selStart, ok = c.undoText(id, *value, caret, selStart)
						}
						if ok {
							slots = parseMaskedSlots(nextText, maskRunes)
							changed = true
						}
					}
				case KeyY:
					if !st.ReadOnly {
						if nextText, nc, ns, ok := c.redoText(id, *value, caret, selStart); ok {
							slots = parseMaskedSlots(nextText, maskRunes)
							caret, selStart = nc, ns
							changed = true
						}
					}
				case KeyV:
					if !st.ReadOnly {
						if clip, err := c.Win.ClipboardText(); err == nil && clip != "" {
							c.pushTextUndo(id, *value, caret, selStart)
							if a, b, ok := selectionBounds(caret, selStart, len(maskRunes)); ok {
								clearSlotRange(a, b)
								caret = a
							}
							pos := nextMaskSlot(maskRunes, caret)
							for _, ch := range clip {
								if pos >= len(maskRunes) {
									break
								}
								if matchMaskSlot(maskRunes[pos], ch) {
									slots[pos] = ch
									pos = nextMaskSlot(maskRunes, pos+1)
								}
							}
							caret = pos
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
					if a, b, ok := selectionBounds(caret, selStart, len(maskRunes)); ok {
						c.pushTextUndo(id, *value, caret, selStart)
						clearSlotRange(a, b)
						caret = nextMaskSlot(maskRunes, a)
						selStart = caret
						changed = true
					} else if prev := prevMaskSlot(maskRunes, caret); prev >= 0 {
						c.pushTextUndo(id, *value, caret, selStart)
						slots[prev] = 0
						caret = prev
						selStart = caret
						changed = true
					}
				}
			case KeyDelete:
				if !st.ReadOnly {
					if a, b, ok := selectionBounds(caret, selStart, len(maskRunes)); ok {
						c.pushTextUndo(id, *value, caret, selStart)
						clearSlotRange(a, b)
						caret = nextMaskSlot(maskRunes, a)
						selStart = caret
						changed = true
					} else if next := nextMaskSlot(maskRunes, caret); next < len(maskRunes) {
						c.pushTextUndo(id, *value, caret, selStart)
						slots[next] = 0
						caret = next
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
				if caret < len(maskRunes) {
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
				caret = len(maskRunes)
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
				if a, b, ok := selectionBounds(caret, selStart, len(maskRunes)); ok {
					pos := nextMaskSlot(maskRunes, a)
					if pos < b && matchMaskSlot(maskRunes[pos], ch) {
						c.pushTextUndo(id, *value, caret, selStart)
						clearSlotRange(a, b)
						slots[pos] = ch
						caret = nextMaskSlot(maskRunes, pos+1)
						selStart = caret
						changed = true
					}
					continue
				}
				pos := nextMaskSlot(maskRunes, caret)
				if pos < len(maskRunes) && matchMaskSlot(maskRunes[pos], ch) {
					c.pushTextUndo(id, *value, caret, selStart)
					slots[pos] = ch
					caret = nextMaskSlot(maskRunes, pos+1)
					selStart = caret
					changed = true
				}
			}
		}

		c.caretPos[id] = caret
		c.textSelStart[id] = selStart
		if changed {
			*value = formatMaskedValue(slots, maskRunes)
			for i, m := range maskRunes {
				if isMaskSlot(m) {
					if slots[i] != 0 {
						displayRunes[i] = slots[i]
					} else {
						displayRunes[i] = '_'
					}
				} else {
					displayRunes[i] = m
				}
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
	c.Canvas.DrawRoundedBox(r, c.Sf(rad), bg, 1.0, borderCol)

	if textArea.W > 0 && c.Font != nil {
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
				c.Canvas.FillRect(render.Rect{
					X: textArea.X + x1 - scroll,
					Y: textArea.Y + c.S(5),
					W: x2 - x1,
					H: textArea.H - c.S(10),
				}, c.Theme.Accent.WithAlpha(90))
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
		if *value == "" && !focused {
			c.Font.DrawTextStyled(c.Canvas, textArea.X, textY, string(displayRunes), pxSize, c.Theme.TextMuted, st.FontStyle)
		} else {
			c.Font.DrawTextStyled(c.Canvas, textArea.X-scroll, textY, string(displayRunes), pxSize, textCol, st.FontStyle)
		}

		if focused {
			caretX := textArea.X + prefixW - scroll
			c.Canvas.FillRect(render.Rect{X: caretX, Y: textArea.Y + c.S(6), W: c.S(2), H: textArea.H - c.S(12)}, c.Theme.Accent)
		}
		c.Canvas.PopClip()
	}

	c.drawInputError(r, st.ErrorText)
	c.textFieldContextMenu(id, r, value, st, false)
	return changed
}
