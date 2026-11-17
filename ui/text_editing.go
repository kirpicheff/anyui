package ui

import (
	"strings"
	"unicode"

	"git.gtw.su/kirpich/anyui/render"
)

const maxTextUndoEntries = 100

// textSnapshot хранит состояние текста и выделения для истории правок.
type textSnapshot struct {
	text     string
	caret    int
	selStart int
}

// textUndoState хранит стеки отмены и повтора для конкретного поля ввода.
type textUndoState struct {
	undo []textSnapshot
	redo []textSnapshot
}

func (c *Context) ensureTextEditingMaps() {
	c.ensureTextMaps()
	if c.textPrefX == nil {
		c.textPrefX = make(map[string]int)
	}
	if c.textHistory == nil {
		c.textHistory = make(map[string]*textUndoState)
	}
	if c.textMenuChanged == nil {
		c.textMenuChanged = make(map[string]bool)
	}
}

func (c *Context) getTextHistory(id string) *textUndoState {
	c.ensureTextEditingMaps()
	h, ok := c.textHistory[id]
	if !ok {
		h = &textUndoState{}
		c.textHistory[id] = h
	}
	return h
}

// pushTextUndo сохраняет снимок состояния перед правкой и очищает стек повтора.
func (c *Context) pushTextUndo(id, text string, caret, selStart int) {
	h := c.getTextHistory(id)
	if n := len(h.undo); n > 0 && h.undo[n-1].text == text {
		return
	}
	h.undo = append(h.undo, textSnapshot{text: text, caret: caret, selStart: selStart})
	if len(h.undo) > maxTextUndoEntries {
		copy(h.undo, h.undo[len(h.undo)-maxTextUndoEntries:])
		h.undo = h.undo[:maxTextUndoEntries]
	}
	h.redo = nil
}

// undoText откатывает текст к предыдущему снимку и возвращает новое состояние.
func (c *Context) undoText(id, curText string, curCaret, curSelStart int) (string, int, int, bool) {
	h := c.getTextHistory(id)
	if len(h.undo) == 0 {
		return curText, curCaret, curSelStart, false
	}
	prev := h.undo[len(h.undo)-1]
	h.undo = h.undo[:len(h.undo)-1]
	h.redo = append(h.redo, textSnapshot{text: curText, caret: curCaret, selStart: curSelStart})
	return prev.text, prev.caret, prev.selStart, true
}

// redoText повторяет отменённую правку и возвращает новое состояние.
func (c *Context) redoText(id, curText string, curCaret, curSelStart int) (string, int, int, bool) {
	h := c.getTextHistory(id)
	if len(h.redo) == 0 {
		return curText, curCaret, curSelStart, false
	}
	next := h.redo[len(h.redo)-1]
	h.redo = h.redo[:len(h.redo)-1]
	h.undo = append(h.undo, textSnapshot{text: curText, caret: curCaret, selStart: curSelStart})
	return next.text, next.caret, next.selStart, true
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// prevWordStart возвращает индекс начала слова слева от позиции pos.
func prevWordStart(runes []rune, pos int) int {
	i := clampRuneIdx(pos, len(runes))
	if i == 0 {
		return 0
	}
	for i > 0 && unicode.IsSpace(runes[i-1]) {
		i--
	}
	if i == 0 {
		return 0
	}
	if isWordRune(runes[i-1]) {
		for i > 0 && isWordRune(runes[i-1]) {
			i--
		}
	} else {
		for i > 0 && !isWordRune(runes[i-1]) && !unicode.IsSpace(runes[i-1]) {
			i--
		}
	}
	return i
}

// nextWordEnd возвращает индекс конца слова (или начала следующего слова) справа от pos.
func nextWordEnd(runes []rune, pos int) int {
	n := len(runes)
	i := clampRuneIdx(pos, n)
	if i >= n {
		return n
	}
	if unicode.IsSpace(runes[i]) {
		for i < n && unicode.IsSpace(runes[i]) {
			i++
		}
		return i
	}
	if isWordRune(runes[i]) {
		for i < n && isWordRune(runes[i]) {
			i++
		}
	} else {
		for i < n && !isWordRune(runes[i]) && !unicode.IsSpace(runes[i]) {
			i++
		}
	}
	for i < n && unicode.IsSpace(runes[i]) {
		i++
	}
	return i
}

// wordBoundsAt возвращает границы слова (или группы символов) под индексом pos для двойного клика.
func wordBoundsAt(runes []rune, pos int) (int, int) {
	n := len(runes)
	if n == 0 {
		return 0, 0
	}
	i := clampRuneIdx(pos, n-1)
	if unicode.IsSpace(runes[i]) {
		start, end := i, i+1
		for start > 0 && unicode.IsSpace(runes[start-1]) {
			start--
		}
		for end < n && unicode.IsSpace(runes[end]) {
			end++
		}
		return start, end
	}
	if isWordRune(runes[i]) {
		start, end := i, i+1
		for start > 0 && isWordRune(runes[start-1]) {
			start--
		}
		for end < n && isWordRune(runes[end]) {
			end++
		}
		return start, end
	}
	start, end := i, i+1
	for start > 0 && !isWordRune(runes[start-1]) && !unicode.IsSpace(runes[start-1]) {
		start--
	}
	for end < n && !isWordRune(runes[end]) && !unicode.IsSpace(runes[end]) {
		end++
	}
	return start, end
}

// lineBoundsAt возвращает границы логической строки (между '\n') вокруг pos для тройного клика.
func lineBoundsAt(runes []rune, pos int) (int, int) {
	n := len(runes)
	i := clampRuneIdx(pos, n)
	start := i
	for start > 0 && runes[start-1] != '\n' {
		start--
	}
	end := i
	for end < n && runes[end] != '\n' {
		end++
	}
	return start, end
}

// inputErrorExtraH возвращает дополнительную высоту под сообщение об ошибке валидации.
func (c *Context) inputErrorExtraH(errText string) int {
	if errText == "" {
		return 0
	}
	return c.S(20)
}

// drawInputError рисует текст ошибки валидации непосредственно под полем ввода.
func (c *Context) drawInputError(fieldRect render.Rect, errText string) {
	if errText == "" || c.Font == nil {
		return
	}
	c.Font.DrawText(c.Canvas, fieldRect.X+c.S(2), fieldRect.Y+fieldRect.H+c.S(4), errText, c.S(12), c.Theme.Danger)
}

// textFieldContextMenu открывает встроенное контекстное меню по ПКМ для текстовых полей.
func (c *Context) textFieldContextMenu(id string, anchor render.Rect, value *string, st WidgetStyle, multiline bool) {
	if st.Disabled || value == nil {
		return
	}
	if c.RightMousePressed && !c.IsInputBlocked() && anchor.Contains(c.RightMouseX, c.RightMouseY) {
		c.FocusID = id
	}
	runes := []rune(*value)
	caret := clampRuneIdx(c.caretPos[id], len(runes))
	selStart := clampRuneIdx(c.textSelStart[id], len(runes))
	_, _, hasSel := selectionBounds(caret, selStart, len(runes))

	loc := c.str()
	items := []ContextMenuItem{
		{
			Label:    loc.Cut,
			Shortcut: "Ctrl+X",
			Disabled: st.ReadOnly || !hasSel || st.Password,
			OnClick: func() {
				cur := []rune(*value)
				ca := clampRuneIdx(c.caretPos[id], len(cur))
				ss := clampRuneIdx(c.textSelStart[id], len(cur))
				if a, b, ok := selectionBounds(ca, ss, len(cur)); ok {
					c.pushTextUndo(id, *value, ca, ss)
					_ = c.Win.SetClipboardText(string(cur[a:b]))
					next, nextCaret := deleteRuneRange(cur, a, b)
					*value = string(next)
					c.caretPos[id] = nextCaret
					c.textSelStart[id] = nextCaret
					c.FocusID = id
					c.textMenuChanged[id] = true
				}
			},
		},
		{
			Label:    loc.Copy,
			Shortcut: "Ctrl+C",
			Disabled: !hasSel || st.Password,
			OnClick: func() {
				cur := []rune(*value)
				ca := clampRuneIdx(c.caretPos[id], len(cur))
				ss := clampRuneIdx(c.textSelStart[id], len(cur))
				if a, b, ok := selectionBounds(ca, ss, len(cur)); ok {
					_ = c.Win.SetClipboardText(string(cur[a:b]))
				}
				c.FocusID = id
			},
		},
		{
			Label:    loc.Paste,
			Shortcut: "Ctrl+V",
			Disabled: st.ReadOnly,
			OnClick: func() {
				clip, err := c.Win.ClipboardText()
				if err != nil || clip == "" {
					c.FocusID = id
					return
				}
				if multiline {
					clip = strings.ReplaceAll(clip, "\r\n", "\n")
					clip = strings.ReplaceAll(clip, "\r", "\n")
				} else {
					clip = strings.ReplaceAll(clip, "\r", "")
					clip = strings.ReplaceAll(clip, "\n", " ")
				}
				var ins []rune
				for _, ch := range clip {
					if st.InputFilter == nil || st.InputFilter(ch) {
						ins = append(ins, ch)
					}
				}
				if len(ins) == 0 {
					c.FocusID = id
					return
				}
				cur := []rune(*value)
				ca := clampRuneIdx(c.caretPos[id], len(cur))
				ss := clampRuneIdx(c.textSelStart[id], len(cur))
				c.pushTextUndo(id, *value, ca, ss)
				if a, b, ok := selectionBounds(ca, ss, len(cur)); ok {
					cur, ca = deleteRuneRange(cur, a, b)
				}
				next := make([]rune, 0, len(cur)+len(ins))
				next = append(next, cur[:ca]...)
				next = append(next, ins...)
				next = append(next, cur[ca:]...)
				*value = string(next)
				c.caretPos[id] = ca + len(ins)
				c.textSelStart[id] = ca + len(ins)
				c.FocusID = id
				c.textMenuChanged[id] = true
			},
		},
		{Separator: true},
		{
			Label:    loc.SelectAll,
			Shortcut: "Ctrl+A",
			Disabled: len(runes) == 0,
			OnClick: func() {
				cur := []rune(*value)
				c.textSelStart[id] = 0
				c.caretPos[id] = len(cur)
				c.FocusID = id
			},
		},
	}
	c.ContextMenu("txt_cm:"+id, anchor, items)
}

// selectableContextMenu открывает контекстное меню для SelectableLabel («Копировать», «Выделить всё»).
func (c *Context) selectableContextMenu(id string, anchor render.Rect, text string) {
	if c.RightMousePressed && !c.IsInputBlocked() && anchor.Contains(c.RightMouseX, c.RightMouseY) {
		c.FocusID = id
	}
	runes := []rune(text)
	caret := clampRuneIdx(c.caretPos[id], len(runes))
	selStart := clampRuneIdx(c.textSelStart[id], len(runes))
	_, _, hasSel := selectionBounds(caret, selStart, len(runes))

	loc := c.str()
	items := []ContextMenuItem{
		{
			Label:    loc.Copy,
			Shortcut: "Ctrl+C",
			Disabled: !hasSel,
			OnClick: func() {
				if a, b, ok := selectionBounds(caret, selStart, len(runes)); ok {
					_ = c.Win.SetClipboardText(string(runes[a:b]))
				}
				c.FocusID = id
			},
		},
		{
			Label:    loc.SelectAll,
			Shortcut: "Ctrl+A",
			Disabled: len(runes) == 0,
			OnClick: func() {
				c.textSelStart[id] = 0
				c.caretPos[id] = len(runes)
				c.FocusID = id
			},
		},
	}
	c.ContextMenu("sel_cm:"+id, anchor, items)
}

