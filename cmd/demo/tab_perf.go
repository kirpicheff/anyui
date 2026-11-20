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
	imgSizeDp       int
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

	dirtyList := ui.LastDirtyRects()
	if len(dirtyList) > 0 {
		u := dirtyList[0]
		for _, r := range dirtyList[1:] {
			u = u.Union(r)
		}
		if !u.Empty() {
			perfTab.lastDirty = u
		}
	}

	ui.ScrollView("scroll_perf", 0, 0, func() {
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
							ui.CanvasView(320, 74, func(c *render.Canvas, r anyui.Rect) {
								c.FillLinearGradient(r, render.UniformRadii(ui.Sf(8)), render.LinearGradient{
									StartX: 0, StartY: 0, EndX: 1, EndY: 1,
									Stops: []render.GradientStop{
										{Pos: 0, Color: th.SurfaceDark},
										{Pos: 1, Color: th.Accent.WithAlpha(90)},
									},
								})
								c.DrawSVG(svgStar, r.X+ui.S(12), r.Y+ui.S(18), ui.S(36), th.Warning)
								c.DrawSVG(svgGear, r.X+ui.S(58), r.Y+ui.S(18), ui.S(36), th.Success)
								pts := make([]render.PointF, 0, 12)
								for k := 0; k < 12; k++ {
									px := float32(r.X+ui.S(115)) + float32(k)*ui.Sf(16)
									py := float32(r.Y+ui.S(38)) + float32(math.Sin(float64(k)*0.8))*ui.Sf(18)
									pts = append(pts, render.PointF{X: px, Y: py})
								}
								c.DrawPolyline(pts, ui.Sf(2), th.Accent, false)
							})
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
				area := ui.CurrentAreaPx()
				ovRect := anyui.Rect{
					X: area.X + ui.S(24),
					Y: area.Y + ui.S(90),
					W: ui.S(260),
					H: ui.S(64),
				}
				ui.PushOverlay(ovRect, func() {
					ui.Canvas().DrawBoxShadow(ovRect, render.BoxShadow{
						OffsetY: ui.Sf(4), Blur: ui.Sf(12), Color: render.RGBA(0, 0, 0, 130), Radii: render.UniformRadii(ui.Sf(8)),
					})
					ui.Canvas().DrawRoundedBox(ovRect, ui.Sf(8), th.Card, 1.5, th.Accent)
					ui.DrawText(ovRect.X+ui.S(12), ovRect.Y+ui.S(12), "Overlay Layer поверх Static", 12, th.TextPrimary)
					ui.DrawText(ovRect.X+ui.S(12), ovRect.Y+ui.S(34), "Static Layer под ним не перерисовывался", 11, th.TextMuted)
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
				ui.CanvasView(340, 50, func(c *render.Canvas, r anyui.Rect) {
					ui.RegisterHoverArea("perf_hover_static", r, anyui.CursorHand, false)
					mx, my := ui.MousePosPx()
					hov := r.Contains(mx, my)
					bg := th.Surface
					if hov {
						bg = th.BtnSecondaryHover
					}
					c.DrawRoundedBox(r, ui.Sf(8), bg, 1.0, th.Border)
					ui.DrawText(r.X+ui.S(12), r.Y+ui.S(10), "Зона без перерисовки внутри (trackMove=false)", 12, th.TextPrimary)
					ui.DrawText(r.X+ui.S(12), r.Y+ui.S(28), "Водите мышью здесь — Render не вызывается", 11, th.TextMuted)
				})
				ui.CanvasView(340, 50, func(c *render.Canvas, r anyui.Rect) {
					ui.RegisterHoverArea("perf_hover_track", r, anyui.CursorCrosshair, true)
					mx, my := ui.MousePosPx()
					hov := r.Contains(mx, my)
					bg := th.Surface
					if hov {
						bg = th.Accent.WithAlpha(45)
					}
					c.DrawRoundedBox(r, ui.Sf(8), bg, 1.0, th.BorderFocus)
					ui.DrawText(r.X+ui.S(12), r.Y+ui.S(10), "Зона с трекингом (trackMove=true)", 12, th.TextPrimary)
					if hov {
						ui.DrawText(r.X+ui.S(12), r.Y+ui.S(28), fmt.Sprintf("Курсор внутри: (%d, %d)", mx-r.X, my-r.Y), 11, th.Accent)
					} else {
						ui.DrawText(r.X+ui.S(12), r.Y+ui.S(28), "Наведите для трекинга координат и Crosshair", 11, th.TextMuted)
					}
				})
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
					ui.CanvasView(134, 34, func(c *render.Canvas, r anyui.Rect) {
						ui.RegisterHoverArea("perf_"+item.id, r, item.cursor, false)
						mx, my := ui.MousePosPx()
						hov := r.Contains(mx, my)
						bg := th.Surface
						border := th.Border
						if hov {
							ui.SetCursor(item.cursor)
							bg = th.Accent.WithAlpha(40)
							border = th.Accent
						}
						c.DrawRoundedBox(r, ui.Sf(6), bg, 1.0, border)
						ui.DrawText(r.X+ui.S(10), r.Y+ui.S(10), item.label, 12, th.TextPrimary)
					})
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
							ui.Image(st.sampleImg, perfTab.imgSizeDp, perfTab.imgSizeDp, perfTab.imgScaleMode)
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

