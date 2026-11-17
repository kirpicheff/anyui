package ui

import (
	"fmt"
	"strconv"
	"strings"

	"anyui/platform"
	"anyui/render"
)

// LogLevel задаёт уровень важности записи лога для цветовой подсветки.
type LogLevel int

const (
	LogInfo LogLevel = iota
	LogWarn
	LogError
	LogDebug
)

// LogEntry описывает одну строку в LogView.
type LogEntry struct {
	Level     LogLevel
	Timestamp string
	Text      string
}

type logViewState struct {
	followTail bool
	lastCount  int
	selAnchor  int
	selActive  int
	selected   map[int]bool
	searchText string
	showSearch bool
}

var logViewStore = make(map[string]*logViewState)

func getLogViewState(id string) *logViewState {
	st, ok := logViewStore[id]
	if !ok {
		st = &logViewState{
			followTail: true,
			selAnchor:  -1,
			selActive:  -1,
			selected:   make(map[int]bool),
		}
		logViewStore[id] = st
	}
	return st
}

// LogView отображает виртуальный список записей лога с автопрокруткой вниз (Follow tail),
// раскраской уровней, поиском и копированием выделенных строк.
func (c *Context) LogView(id string, entries []LogEntry, followTail *bool, opts ...Option) {
	st := ApplyOptions(opts)
	hDp := 260
	if st.HeightDp != nil && *st.HeightDp > 0 {
		hDp = *st.HeightDp
	}
	r := c.allocWidth(hDp, st)

	lvs := getLogViewState(id)
	if followTail != nil {
		lvs.followTail = *followTail
	}

	c.registerFocusable(id)

	bg := c.Theme.Bg
	if st.Bg != nil {
		bg = *st.Bg
	}
	c.Canvas.DrawRoundedBox(r, c.Sf(c.Theme.RadiusSm), bg, 1.0, c.Theme.Border)

	// Верхняя мини-панель инструментов LogView (поиск + кнопка Follow tail + Копировать)
	barH := c.S(30)
	barRect := render.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: barH}
	c.Canvas.FillRect(barRect, c.Theme.Surface)
	c.Canvas.FillRect(render.Rect{X: barRect.X, Y: barRect.Y + barRect.H - 1, W: barRect.W, H: 1}, c.Theme.Border)

	pxSmall := c.S(11)
	curX := barRect.X + c.S(8)

	// Поле быстрого поиска прямо в шапке LogView
	searchW := c.S(180)
	if searchW > barRect.W/2 {
		searchW = barRect.W / 2
	}
	searchRect := render.Rect{X: curX, Y: barRect.Y + c.S(4), W: searchW, H: barRect.H - c.S(8)}
	curX += searchW + c.S(8)

	loc := c.str()
	c.drawInlineSearchBox(id+":search", searchRect, &lvs.searchText, loc.LogFilterPlaceholder)

	// Фильтрация записей
	var visibleIndices []int
	query := strings.ToLower(strings.TrimSpace(lvs.searchText))
	for i, e := range entries {
		if query == "" ||
			strings.Contains(strings.ToLower(e.Text), query) ||
			strings.Contains(strings.ToLower(e.Timestamp), query) {
			visibleIndices = append(visibleIndices, i)
		}
	}

	// Кнопка "Follow tail" справа
	tailLabel := loc.LogTail
	tailW := c.S(48)
	copyLabel := loc.LogCopy
	copyW := c.S(76)
	rightX := barRect.X + barRect.W - c.S(6)

	tailRect := render.Rect{X: rightX - tailW, Y: barRect.Y + c.S(4), W: tailW, H: barRect.H - c.S(8)}
	rightX -= tailW + c.S(6)
	copyRect := render.Rect{X: rightX - copyW, Y: barRect.Y + c.S(4), W: copyW, H: barRect.H - c.S(8)}

	// Отрисовка кнопки Tail
	c.RegisterHit(id+":tail", tailRect, platform.CursorHand)
	hovTail := !c.IsInputBlocked() && tailRect.Contains(c.MouseX, c.MouseY)
	tailBg := c.Theme.Card
	tailFg := c.Theme.TextMuted
	tailBorder := c.Theme.Border
	if lvs.followTail {
		tailBg = c.Theme.Accent.WithAlpha(40)
		tailFg = c.Theme.Accent
		tailBorder = c.Theme.Accent
	} else if hovTail {
		c.nextCursor = platform.CursorHand
	}
	if hovTail {
		c.nextCursor = platform.CursorHand
	}
	c.Canvas.DrawRoundedBox(tailRect, c.Sf(4), tailBg, 1.0, tailBorder)
	if c.Font != nil {
		tw, th := c.Font.MeasureText(tailLabel, pxSmall)
		c.Font.DrawText(c.Canvas, tailRect.X+(tailRect.W-tw)/2, tailRect.Y+(tailRect.H-th)/2, tailLabel, pxSmall, tailFg)
	}
	if hovTail && c.MousePressed {
		lvs.followTail = !lvs.followTail
		if followTail != nil {
			*followTail = lvs.followTail
		}
		c.RequestRedraw()
	}

	// Кнопка Копировать
	hasSel := len(lvs.selected) > 0
	c.RegisterHit(id+":copy", copyRect, platform.CursorHand)
	hovCopy := !c.IsInputBlocked() && copyRect.Contains(c.MouseX, c.MouseY)
	if hovCopy {
		c.nextCursor = platform.CursorHand
	}
	copyFg := c.Theme.TextMuted
	if !hasSel {
		copyFg = c.Theme.TextMuted
	}
	c.Canvas.DrawRoundedBox(copyRect, c.Sf(4), c.Theme.Card, 1.0, c.Theme.Border)
	if c.Font != nil {
		cw, ch := c.Font.MeasureText(copyLabel, pxSmall)
		c.Font.DrawText(c.Canvas, copyRect.X+(copyRect.W-cw)/2, copyRect.Y+(copyRect.H-ch)/2, copyLabel, pxSmall, copyFg)
	}
	if hovCopy && c.MousePressed && len(visibleIndices) > 0 {
		c.copyLogSelection(entries, visibleIndices, lvs)
	}

	// Счётчик строк
	if c.Font != nil && curX+c.S(60) < copyRect.X {
		cntText := fmt.Sprintf("%d/%d", len(visibleIndices), len(entries))
		_, ch := c.Font.MeasureText(cntText, pxSmall)
		c.Font.DrawText(c.Canvas, curX, barRect.Y+(barRect.H-ch)/2, cntText, pxSmall, c.Theme.TextMuted)
	}

	// Основная область списка строк
	viewRect := render.Rect{X: r.X + 1, Y: r.Y + barH + 1, W: r.W - 2, H: r.H - barH - 2}
	if viewRect.H <= 0 || viewRect.W <= 0 {
		return
	}

	rowH := c.S(20)
	sbW := c.S(8)
	totalH := len(visibleIndices) * rowH
	si := c.getScrollInfo(id)
	maxScroll := totalH - viewRect.H
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Автопрокрутка вниз при включённом FollowTail
	if lvs.followTail && (len(entries) != lvs.lastCount || si.y > maxScroll) {
		si.y = maxScroll
	}
	lvs.lastCount = len(entries)

	// Прокрутка колесом мыши
	if !c.IsInputBlocked() && viewRect.Contains(c.MouseX, c.MouseY) && c.WheelY != 0 {
		delta := -int(c.WheelY * float32(rowH*3))
		si.y += delta
		if si.y < 0 {
			si.y = 0
		}
		if si.y >= maxScroll {
			si.y = maxScroll
		} else if delta < 0 {
			// Пользователь прокрутил вверх — отключаем FollowTail
			lvs.followTail = false
			if followTail != nil {
				*followTail = false
			}
		}
		c.RequestRedraw()
	}
	if si.y > maxScroll {
		si.y = maxScroll
	}
	if si.y < 0 {
		si.y = 0
	}

	// Обработка клавиатуры (Ctrl+A, Ctrl+C) при фокусе
	if c.FocusID == id && !c.IsInputBlocked() {
		for _, kp := range c.KeyEvents {
			if kp.Mods.Ctrl && kp.Code == KeyA {
				lvs.selected = make(map[int]bool, len(visibleIndices))
				for vi := range visibleIndices {
					lvs.selected[vi] = true
				}
				c.RequestRedraw()
			} else if kp.Mods.Ctrl && kp.Code == KeyC {
				c.copyLogSelection(entries, visibleIndices, lvs)
			}
		}
	}

	contentRect := render.Rect{X: viewRect.X, Y: viewRect.Y, W: viewRect.W - sbW, H: viewRect.H}
	c.Canvas.PushClip(contentRect)

	startIdx, endIdx := ComputeVisibleRange(si.y, viewRect.H, rowH, len(visibleIndices))
	pxLog := c.S(12)
	mods := c.currentMods()

	for vi := startIdx; vi < endIdx; vi++ {
		entry := entries[visibleIndices[vi]]
		rowY := viewRect.Y + vi*rowH - si.y
		rowRect := render.Rect{X: contentRect.X, Y: rowY, W: contentRect.W, H: rowH}
		c.RegisterHit(id+":row:"+strconv.Itoa(vi), viewRect.Intersect(rowRect), platform.CursorArrow)

		isSel := lvs.selected[vi]
		hovRow := !c.IsInputBlocked() && rowRect.Contains(c.MouseX, c.MouseY)

		if isSel {
			c.Canvas.FillRect(rowRect, c.Theme.Accent.WithAlpha(55))
		} else if hovRow {
			c.Canvas.FillRect(rowRect, c.Theme.Surface.WithAlpha(160))
		}

		if hovRow && c.MousePressed {
			c.FocusID = id
			if mods.Shift && lvs.selAnchor >= 0 && lvs.selAnchor < len(visibleIndices) {
				lo, hi := lvs.selAnchor, vi
				if lo > hi {
					lo, hi = hi, lo
				}
				if !mods.Ctrl {
					lvs.selected = make(map[int]bool)
				}
				for k := lo; k <= hi; k++ {
					lvs.selected[k] = true
				}
				lvs.selActive = vi
			} else if mods.Ctrl {
				if lvs.selected[vi] {
					delete(lvs.selected, vi)
				} else {
					lvs.selected[vi] = true
				}
				lvs.selAnchor = vi
				lvs.selActive = vi
			} else {
				lvs.selected = map[int]bool{vi: true}
				lvs.selAnchor = vi
				lvs.selActive = vi
			}
			c.RequestRedraw()
		}

		if c.Font != nil {
			tx := rowRect.X + c.S(8)
			if entry.Timestamp != "" {
				tw, th := c.Font.MeasureTextStyled(entry.Timestamp, pxLog, render.FontMono)
				c.Font.DrawTextStyled(c.Canvas, tx, rowRect.Y+(rowRect.H-th)/2, entry.Timestamp, pxLog, c.Theme.TextMuted, render.FontMono)
				tx += tw + c.S(8)
			}

			lvlTag, lvlCol := c.logLevelBadge(entry.Level)
			lw, lh := c.Font.MeasureTextStyled(lvlTag, pxLog, render.FontMono)
			c.Font.DrawTextStyled(c.Canvas, tx, rowRect.Y+(rowRect.H-lh)/2, lvlTag, pxLog, lvlCol, render.FontMono)
			tx += lw + c.S(8)

			textCol := c.Theme.TextPrimary
			if entry.Level == LogError {
				textCol = c.Theme.Danger
			} else if entry.Level == LogDebug {
				textCol = c.Theme.TextMuted
			}
			_, th := c.Font.MeasureTextStyled(entry.Text, pxLog, render.FontMono)
			c.Font.DrawTextStyled(c.Canvas, tx, rowRect.Y+(rowRect.H-th)/2, entry.Text, pxLog, textCol, render.FontMono)
		}
	}

	c.Canvas.PopClip()

	if totalH > viewRect.H {
		sbTrack := render.Rect{X: viewRect.X + viewRect.W - sbW, Y: viewRect.Y, W: sbW, H: viewRect.H}
		c.drawVerticalScrollbar(id+":vscroll", sbTrack, si, totalH, viewRect.H)
	}
}

// VirtualTextView отображает большой массив текстовых строк с виртуализацией и возможностью выделения/копирования.
func (c *Context) VirtualTextView(id string, lines []string, opts ...Option) {
	entries := make([]LogEntry, len(lines))
	for i, line := range lines {
		entries[i] = LogEntry{Level: LogInfo, Text: line}
	}
	follow := false
	c.LogView(id, entries, &follow, opts...)
}

func (c *Context) logLevelBadge(lvl LogLevel) (string, render.Color) {
	switch lvl {
	case LogWarn:
		return "[WARN]", c.Theme.Warning
	case LogError:
		return "[ERR ]", c.Theme.Danger
	case LogDebug:
		return "[DBG ]", c.Theme.TextMuted
	default:
		return "[INFO]", c.Theme.Accent
	}
}

func (c *Context) copyLogSelection(entries []LogEntry, visibleIndices []int, lvs *logViewState) {
	if c.Win == nil || len(visibleIndices) == 0 {
		return
	}
	var sb strings.Builder
	for vi, origIdx := range visibleIndices {
		if len(lvs.selected) > 0 && !lvs.selected[vi] {
			continue
		}
		e := entries[origIdx]
		if sb.Len() > 0 {
			sb.WriteByte('\n')
		}
		if e.Timestamp != "" {
			sb.WriteString(e.Timestamp)
			sb.WriteByte(' ')
		}
		tag, _ := c.logLevelBadge(e.Level)
		sb.WriteString(tag)
		sb.WriteByte(' ')
		sb.WriteString(e.Text)
	}
	if sb.Len() > 0 {
		c.Win.SetClipboardText(sb.String())
	}
}

// drawInlineSearchBox рисует компактное поле поиска без аллокации в основном layoutStack.
func (c *Context) drawInlineSearchBox(id string, r render.Rect, value *string, placeholder string) {
	if value == nil {
		return
	}
	focused := c.FocusID == id
	border := c.Theme.Border
	if focused {
		border = c.Theme.Accent
	}
	c.Canvas.DrawRoundedBox(r, c.Sf(4), c.Theme.Bg, 1.0, border)
	c.RegisterHit(id, r, platform.CursorIBeam)
	if !c.IsInputBlocked() && r.Contains(c.MouseX, c.MouseY) {
		c.nextCursor = platform.CursorIBeam
		if c.MousePressed {
			c.FocusID = id
			c.RequestRedraw()
		}
	} else if c.MousePressed && focused {
		// Снимаем фокус при клике мимо
		c.FocusID = ""
		c.RequestRedraw()
	}

	if focused && !c.IsInputBlocked() {
		runes := []rune(*value)
		changed := false
		for _, kp := range c.KeyEvents {
			if kp.Code == KeyBackspace && len(runes) > 0 {
				runes = runes[:len(runes)-1]
				changed = true
			} else if kp.Code == KeyEscape {
				c.FocusID = ""
				c.RequestRedraw()
			}
		}
		for _, ch := range c.TypedRunes {
			if ch >= 32 {
				runes = append(runes, ch)
				changed = true
			}
		}
		if changed {
			*value = string(runes)
			c.RequestRedraw()
		}
	}

	if c.Font != nil {
		c.Canvas.PushClip(r)
		px := c.S(11)
		txt := *value
		col := c.Theme.TextPrimary
		if txt == "" {
			txt = placeholder
			col = c.Theme.TextMuted
		}
		_, th := c.Font.MeasureText(txt, px)
		c.Font.DrawText(c.Canvas, r.X+c.S(6), r.Y+(r.H-th)/2, txt, px, col)
		c.Canvas.PopClip()
	}
}
