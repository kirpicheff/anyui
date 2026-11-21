package main

import (
	"fmt"
	"math"

	"github.com/kirpicheff/anyui"
	"github.com/kirpicheff/anyui/render"
)

type perfTabState struct {
	renderCount     uint64
	staticDraws     int
	dynamicClicks   int
	smartMove       bool
	showOverlayTest bool
	highlightDirty  bool
	lastDirty       anyui.Rect
	imgSizeDp       anyui.Dp
	imgScaleMode    anyui.ImageScaleMode
	vtvAltSet       bool
	vtvLinesA       []string
	vtvLinesB       []string
}

var perfTab = perfTabState{
	smartMove:    true,
	imgSizeDp:    96,
	imgScaleMode: anyui.ScaleFit,
}

func ensurePerfLines() []string {
	if perfTab.vtvLinesA == nil {
		perfTab.vtvLinesA = make([]string, 260)
		perfTab.vtvLinesB = make([]string, 260)
		for i := 0; i < 260; i++ {
			perfTab.vtvLinesA[i] = fmt.Sprintf("[%03d] Глифы кириллицы/ASCII: ABCxyz 0123456789 — проверка LRU-кеша шрифтов", i+1)
			perfTab.vtvLinesB[i] = fmt.Sprintf("[%03d] Набор B: Пропускная способность буфера, SDF, DirtyRect #%d", i+1, i*17)
		}
	}
	if perfTab.vtvAltSet {
		return perfTab.vtvLinesB
	}
	return perfTab.vtvLinesA
}

func (st *demoState) drawTabPerf(ui *anyui.UI) {
	th := ui.Theme()
	perfTab.renderCount++

	dirtyList := ui.LastDirtyRectsPx()
	if len(dirtyList) > 0 {
		u := dirtyList[0]
		for _, r := range dirtyList[1:] {
			u = u.Union(r)
		}
		if !u.Empty() {
			perfTab.lastDirty = u
		}
	}

	ui.ScrollView("scroll_perf", func() {
		ui.Column(14, func() {
			ui.Label("Фаза 7.2: Слои, Dirty Rectangles, фильтры и LRU-кеши", anyui.FontSize(16), anyui.Bold())

			// 1. Стенд слоёв (Static / Dynamic / Overlay) и Dirty Rectangles
			ui.Label("1. Слои (Static / Dynamic / Overlay) и Dirty Rectangles", anyui.FontSize(13), anyui.Bold())
			ui.Row(12, 182, func() {
				// Статический блок: не перерисовывается при кликах по Dynamic Layer
				ui.StaticLayer("perf_static_card", func() {
					perfTab.staticDraws++
					ui.Card(func() {
						ui.Column(6, func() {
							ui.Label(fmt.Sprintf("Static Layer (перерисовок: #%d)", perfTab.staticDraws), anyui.FontSize(13), anyui.Bold(), anyui.TextColor(th.Accent))
							ui.Label("Этот блок с градиентом и SVG кешируется в отдельный буфер.", anyui.FontSize(11), anyui.TextColor(th.TextMuted))
							ui.CanvasView(func(c *anyui.Canvas, r anyui.Rect) {
								c.FillLinearGradient(r, render.UniformRadii(ui.ScaleF(8)), render.LinearGradient{
									StartX: 0, StartY: 0, EndX: 1, EndY: 1,
									Stops: []render.GradientStop{
										{Pos: 0, Color: th.SurfaceDark},
										{Pos: 1, Color: th.Accent.WithAlpha(90)},
									},
								})
								c.DrawSVG(svgStar, r.X+ui.Scale(12).Int(), r.Y+ui.Scale(18).Int(), ui.Scale(36).Int(), th.Warning)
								c.DrawSVG(svgGear, r.X+ui.Scale(58).Int(), r.Y+ui.Scale(18).Int(), ui.Scale(36).Int(), th.Success)
								pts := make([]render.PointF, 0, 12)
								for k := 0; k < 12; k++ {
									px := float32(r.X+ui.Scale(115).Int()) + float32(k)*ui.ScaleF(16)
									py := float32(r.Y+ui.Scale(38).Int()) + float32(math.Sin(float64(k)*0.8))*ui.ScaleF(18)
									pts = append(pts, render.PointF{X: px, Y: py})
								}
								c.DrawPolyline(pts, ui.ScaleF(2), th.Accent, false)
							}, anyui.Width(320), anyui.Height(74))
						})
					}, anyui.Width(370), anyui.Height(176), anyui.Padding(12))
				})

				// Динамический блок справа
				ui.Card(func() {
					ui.Column(8, func() {
						ui.Label("Dynamic Layer (обновляется отдельно)", anyui.FontSize(13), anyui.Bold())
						ui.RowCentered(8, 34, func() {
							if ui.Button("perf_dyn_btn", fmt.Sprintf("Обновить Dynamic (%d)", perfTab.dynamicClicks), anyui.Primary()) {
								perfTab.dynamicClicks++
								st.statusMsg = fmt.Sprintf("Обновлён только Dynamic Layer (#%d)", perfTab.dynamicClicks)
							}
							if ui.Button("perf_inv_btn", "Сбросить Static") {
								ui.InvalidateStatic()
								st.statusMsg = "Static Layer принудительно сброшен (InvalidateStatic)"
							}
						})
						ui.RowCentered(10, 28, func() {
							ui.Toggle("perf_ov_tog", "Overlay-подсказка", &perfTab.showOverlayTest)
							if ui.Checkbox("perf_dirty_hl", "Рамка DirtyRect", &perfTab.highlightDirty) {
								ui.SetDebugDirtyRect(perfTab.highlightDirty)
							}
						})

						blittedText := "Нет (полная отрисовка статики)"
						if ui.LastFrameStaticBlitted() {
							blittedText = "Да (BlitFrom из Static Layer)"
						}
						winArea := ui.Canvas().Width * ui.Canvas().Height
						dirtyPct := 0.0
						if winArea > 0 {
							dirtyPct = float64(perfTab.lastDirty.W*perfTab.lastDirty.H) * 100.0 / float64(winArea)
						}
						ui.Label(fmt.Sprintf("Static Blit: %s | Перерисовок статики: %d", blittedText, perfTab.staticDraws), anyui.FontSize(11), anyui.TextColor(th.TextMuted))
						ui.Label(fmt.Sprintf("Последний DirtyRect: (%d, %d) %dx%d (%.1f%% экрана)",
							perfTab.lastDirty.X, perfTab.lastDirty.Y, perfTab.lastDirty.W, perfTab.lastDirty.H, dirtyPct),
							anyui.FontSize(11), anyui.TextColor(th.Success))
					})
				}, anyui.Flex(1), anyui.Height(176), anyui.Padding(12))
			})

			if perfTab.showOverlayTest {
				area := ui.CurrentArea()
				ovRect := anyui.Rect{
					X: area.X + 24,
					Y: area.Y + 90,
					W: 260,
					H: 64,
				}
				ui.PushOverlay(ovRect, func() {
					ovRectPx := anyui.Rect{
						X: ui.Scale(anyui.Dp(ovRect.X)).Int(),
						Y: ui.Scale(anyui.Dp(ovRect.Y)).Int(),
						W: ui.Scale(anyui.Dp(ovRect.W)).Int(),
						H: ui.Scale(anyui.Dp(ovRect.H)).Int(),
					}
					ui.Canvas().DrawBoxShadow(ovRectPx, render.BoxShadow{
						OffsetY: ui.ScaleF(4), Blur: ui.ScaleF(12), Color: render.RGBA(0, 0, 0, 130), Radii: render.UniformRadii(ui.ScaleF(8)),
					})
					ui.Canvas().DrawRoundedBox(ovRectPx, ui.ScaleF(8), th.Card, 1.5, th.Accent)
					ui.DrawText(ovRectPx.X+ui.Scale(12).Int(), ovRectPx.Y+ui.Scale(12).Int(), "Overlay Layer поверх Static", 12, th.TextPrimary)
					ui.DrawText(ovRectPx.X+ui.Scale(12).Int(), ovRectPx.Y+ui.Scale(34).Int(), "Static Layer под ним не перерисовывался", 11, th.TextMuted)
				})
			}

			ui.Separator()

			// 2. Умная фильтрация EventMouseMove
			ui.Label("2. Умная фильтрация EventMouseMove (0% CPU в простое)", anyui.FontSize(13), anyui.Bold())
			ui.RowCentered(12, 30, func() {
				if ui.Toggle("perf_smart_mm", "Умная фильтрация EventMouseMove", &perfTab.smartMove) {
					ui.SetSmartMouseMove(perfTab.smartMove)
				}
				ui.Label(fmt.Sprintf("Кадров Render: %d | Отфильтровано MouseMove: %d", perfTab.renderCount, ui.SkippedMouseMoves()),
					anyui.FontSize(12), anyui.TextColor(th.Accent))
			})
			ui.Row(12, 54, func() {
				rStatic := ui.CurrentArea()
				rStatic.W = 340
				rStatic.H = 50
				ui.RegisterHoverArea("perf_hover_static", rStatic, anyui.CursorHand, false)
				ui.CanvasView(func(c *anyui.Canvas, r anyui.Rect) {
					mx, my := ui.MousePosPx()
					hov := r.Contains(mx.Int(), my.Int())
					bg := th.Surface
					if hov {
						bg = th.BtnSecondaryHover
					}
					c.DrawRoundedBox(r, ui.ScaleF(8), bg, 1.0, th.Border)
					ui.DrawText(r.X+ui.Scale(12).Int(), r.Y+ui.Scale(10).Int(), "Зона без перерисовки внутри (trackMove=false)", 12, th.TextPrimary)
					ui.DrawText(r.X+ui.Scale(12).Int(), r.Y+ui.Scale(28).Int(), "Водите мышью здесь — Render не вызывается", 11, th.TextMuted)
				}, anyui.Width(340), anyui.Height(50))
				rTrack := ui.CurrentArea()
				rTrack.W = 340
				rTrack.H = 50
				ui.RegisterHoverArea("perf_hover_track", rTrack, anyui.CursorCrosshair, true)
				ui.CanvasView(func(c *anyui.Canvas, r anyui.Rect) {
					mx, my := ui.MousePosPx()
					hov := r.Contains(mx.Int(), my.Int())
					bg := th.Surface
					if hov {
						bg = th.Accent.WithAlpha(45)
					}
					c.DrawRoundedBox(r, ui.ScaleF(8), bg, 1.0, th.BorderFocus)
					ui.DrawText(r.X+ui.Scale(12).Int(), r.Y+ui.Scale(10).Int(), "Зона с трекингом (trackMove=true)", 12, th.TextPrimary)
					if hov {
						ui.DrawText(r.X+ui.Scale(12).Int(), r.Y+ui.Scale(28).Int(), fmt.Sprintf("Курсор внутри: (%d, %d)", mx.Int()-r.X, my.Int()-r.Y), 11, th.Accent)
					} else {
						ui.DrawText(r.X+ui.Scale(12).Int(), r.Y+ui.Scale(28).Int(), "Наведите для трекинга координат и Crosshair", 11, th.TextMuted)
					}
				}, anyui.Width(340), anyui.Height(50))
			})

			ui.Separator()

			// 3. Расширенный набор системных курсоров
			ui.Label("3. Системные курсоры (наведите на карточку для проверки)", anyui.FontSize(13), anyui.Bold())
			cursorItems := []struct {
				id     string
				label  string
				cursor anyui.CursorType
			}{
				{"c_arrow", "Arrow", anyui.CursorArrow},
				{"c_hand", "Hand", anyui.CursorHand},
				{"c_ibeam", "IBeam", anyui.CursorIBeam},
				{"c_resh", "ResizeH", anyui.CursorResizeH},
				{"c_resv", "ResizeV", anyui.CursorResizeV},
				{"c_nwse", "ResizeNWSE", anyui.CursorResizeNWSE},
				{"c_nesw", "ResizeNESW", anyui.CursorResizeNESW},
				{"c_no", "NotAllowed", anyui.CursorNotAllowed},
				{"c_cross", "Crosshair", anyui.CursorCrosshair},
				{"c_move", "Move", anyui.CursorMove},
			}
			ui.Flow(8, 8, func() {
				for _, ci := range cursorItems {
					item := ci
					itemRect := ui.CurrentArea()
					itemRect.W = 134
					itemRect.H = 34
					ui.RegisterHoverArea("perf_"+item.id, itemRect, item.cursor, false)
					ui.CanvasView(func(c *anyui.Canvas, r anyui.Rect) {
						mx, my := ui.MousePosPx()
						hov := r.Contains(mx.Int(), my.Int())
						bg := th.Surface
						border := th.Border
						if hov {
							ui.SetCursor(item.cursor)
							bg = th.Accent.WithAlpha(40)
							border = th.Accent
						}
						c.DrawRoundedBox(r, ui.ScaleF(6), bg, 1.0, border)
						ui.DrawText(r.X+ui.Scale(10).Int(), r.Y+ui.Scale(10).Int(), item.label, 12, th.TextPrimary)
					}, anyui.Width(134), anyui.Height(34))
				}
			})

			ui.Separator()

			// 4. LRU-кеш глифов и масштабированных изображений
			ui.Label("4. LRU-кеш глифов (FontManager) и масштабированных изображений", anyui.FontSize(13), anyui.Bold())
			ui.Row(12, 175, func() {
				// Кеш глифов + VirtualTextView
				ui.Card(func() {
					ui.Column(6, func() {
						gEntries, gBytes := ui.GlyphCacheStats()
						ui.RowCentered(8, 28, func() {
							ui.Label(fmt.Sprintf("Кеш глифов: %d шт. (%.1f КБ)", gEntries, float64(gBytes)/1024.0), anyui.FontSize(12), anyui.Bold())
							if ui.Button("perf_vtv_swap", "Сменить текст", anyui.Height(26), anyui.Padding(8)) {
								perfTab.vtvAltSet = !perfTab.vtvAltSet
							}
							if ui.Button("perf_glyph_clr", "Очистить кеш", anyui.Height(26), anyui.Padding(8)) {
								ui.ClearGlyphCache()
							}
						})
						ui.VirtualTextView("perf_vtv", ensurePerfLines(), anyui.Height(110))
					})
				}, anyui.Flex(1), anyui.Height(170), anyui.Padding(10))

				// Кеш масштабированных изображений
				ui.Card(func() {
					ui.Column(6, func() {
						iEntries, iBytes := ui.ScaledImageCacheStats()
						ui.Label(fmt.Sprintf("Кеш изображений: %d шт. (%.1f КБ)", iEntries, float64(iBytes)/1024.0), anyui.FontSize(12), anyui.Bold())
						ui.Row(10, 120, func() {
							ui.Image(st.sampleImg, perfTab.imgScaleMode, anyui.Width(perfTab.imgSizeDp), anyui.Height(perfTab.imgSizeDp))
							ui.Column(4, func() {
								ui.RowCentered(4, 26, func() {
									if ui.Button("perf_img_64", "64dp", anyui.Height(24), anyui.Padding(6)) {
										perfTab.imgSizeDp = 64
									}
									if ui.Button("perf_img_96", "96dp", anyui.Height(24), anyui.Padding(6)) {
										perfTab.imgSizeDp = 96
									}
									if ui.Button("perf_img_116", "116dp", anyui.Height(24), anyui.Padding(6)) {
										perfTab.imgSizeDp = 116
									}
								})
								ui.RowCentered(4, 26, func() {
									if ui.Button("perf_mode_fit", "Fit", anyui.Height(24), anyui.Padding(6)) {
										perfTab.imgScaleMode = anyui.ScaleFit
									}
									if ui.Button("perf_mode_fill", "Fill", anyui.Height(24), anyui.Padding(6)) {
										perfTab.imgScaleMode = anyui.ScaleFill
									}
									if ui.Button("perf_mode_1x", "1:1", anyui.Height(24), anyui.Padding(6)) {
										perfTab.imgScaleMode = anyui.ScaleOneToOne
									}
								})
								if ui.Button("perf_img_clr", "Очистить кеш изображений", anyui.Height(26), anyui.Padding(8)) {
									ui.ClearScaledImageCache()
								}
							})
						})
					})
				}, anyui.Width(355), anyui.Height(170), anyui.Padding(10))
			})
		})
	})
}
