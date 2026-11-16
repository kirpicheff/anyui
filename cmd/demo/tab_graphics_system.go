package main

import (
	"fmt"
	"math"

	"anyui"
	"anyui/render"
)

func (st *demoState) drawTabGraphicsSystem(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_gfx", 0, 0, func() {
		ui.Label("Графика Фазы 2 (Градиенты, Тени, Blur, Вектор, SVG, Image)", anyui.FontSize(16), anyui.Bold())

		ui.Row(14, 150, func() {
			ui.CanvasView(460, 145, func(c *render.Canvas, r anyui.Rect) {
				c.DrawRoundedBox(r, ui.Sf(10), th.SurfaceDark, 1.0, th.Border)

				// 1. Линейный градиент + тень
				g1 := anyui.Rect{X: r.X + ui.S(14), Y: r.Y + ui.S(16), W: ui.S(120), H: ui.S(70)}
				c.DrawBoxShadow(g1, render.BoxShadow{
					OffsetY: ui.Sf(4),
					Blur:    ui.Sf(10),
					Color:   render.RGBA(0, 0, 0, 120),
					Radii:   render.UniformRadii(ui.Sf(10)),
				})
				c.FillLinearGradient(g1, render.CornerRadii{TL: ui.Sf(14), TR: ui.Sf(4), BR: ui.Sf(14), BL: ui.Sf(4)}, render.LinearGradient{
					StartX: 0, StartY: 0, EndX: 1, EndY: 1,
					Stops: []render.GradientStop{
						{Pos: 0, Color: th.Accent},
						{Pos: 1, Color: render.RGB(168, 85, 247)},
					},
				})

				// 2. BackdropBlur поверх градиента
				blurR := anyui.Rect{X: g1.X + ui.S(40), Y: g1.Y + ui.S(34), W: ui.S(100), H: ui.S(54)}
				c.BackdropBlur(blurR, ui.S(6), render.UniformRadii(ui.Sf(8)))
				c.DrawRoundedBox(blurR, ui.Sf(8), render.RGBA(255, 255, 255, 28), 1.0, render.RGBA(255, 255, 255, 70))

				// 3. Радиальный градиент
				g2 := anyui.Rect{X: r.X + ui.S(175), Y: r.Y + ui.S(16), W: ui.S(90), H: ui.S(90)}
				c.FillRadialGradient(g2, render.UniformRadii(ui.Sf(45)), render.RadialGradient{
					CenterX: 0.4, CenterY: 0.4, Radius: 0.6,
					Stops: []render.GradientStop{
						{Pos: 0, Color: render.RGB(56, 189, 248)},
						{Pos: 1, Color: render.RGB(30, 58, 138)},
					},
				})

				// 4. Векторные примитивы (Polyline + Arc) и SVG с tint
				pts := []render.PointF{
					{X: float32(r.X + ui.S(285)), Y: float32(r.Y + ui.S(85))},
					{X: float32(r.X + ui.S(315)), Y: float32(r.Y + ui.S(45))},
					{X: float32(r.X + ui.S(345)), Y: float32(r.Y + ui.S(68))},
					{X: float32(r.X + ui.S(375)), Y: float32(r.Y + ui.S(25))},
				}
				c.DrawPolyline(pts, ui.Sf(2.5), th.Success, false)
				c.DrawArc(float32(r.X+ui.S(415)), float32(r.Y+ui.S(55)), ui.Sf(22), 0, math.Pi*1.45, ui.Sf(3.5), th.Warning)
				c.DrawSVG(svgStar, r.X+ui.S(285), r.Y+ui.S(95), ui.S(28), th.Warning)
				c.DrawSVG(svgGear, r.X+ui.S(325), r.Y+ui.S(95), ui.S(28), th.Accent)
			})

			ui.Column(6, func() {
				ui.Label("Image:", anyui.FontSize(12), anyui.TextColor(th.TextMuted))
				ui.Image(st.sampleImg, 96, 96, render.ScaleFit)
			})
		})

		ui.Separator()
		ui.Label("Системные диалоги (Фаза 3)", anyui.FontSize(16), anyui.Bold())

		ui.RowCentered(8, 38, func() {
			if ui.Button("dlg_open", "OpenFileDialog") {
				if path, ok, err := anyui.OpenFileDialog(anyui.FileDialogOptions{
					Title: "Выберите файл",
					Owner: ui.Window(),
				}); err == nil && ok {
					st.dialogLog = "Файл: " + path
				} else {
					st.dialogLog = "Выбор файла отменён"
				}
			}
			if ui.Button("dlg_save", "SaveFileDialog") {
				if path, ok, err := anyui.SaveFileDialog(anyui.FileDialogOptions{
					Title:       "Сохранить как",
					DefaultName: "demo.txt",
					Owner:       ui.Window(),
				}); err == nil && ok {
					st.dialogLog = "Сохранить в: " + path
				} else {
					st.dialogLog = "Сохранение отменено"
				}
			}
			if ui.Button("dlg_dir", "SelectFolderDialog") {
				if dir, ok, err := anyui.SelectFolderDialog(anyui.FolderDialogOptions{
					Title: "Выберите папку",
					Owner: ui.Window(),
				}); err == nil && ok {
					st.dialogLog = "Папка: " + dir
				} else {
					st.dialogLog = "Выбор папки отменён"
				}
			}
			if ui.Button("dlg_msg", "MessageBox", anyui.Primary()) {
				res := anyui.MessageBox(ui.Window(), "AnyUI MessageBox", "Сохранить изменения?", anyui.MsgQuestion, anyui.MsgYesNoCancel)
				st.dialogLog = fmt.Sprintf("MessageBox результат: %d", res)
			}
		})
		ui.Label(st.dialogLog, anyui.FontSize(12), anyui.TextColor(th.TextMuted))

		ui.Separator()
		ui.Label("Управление окном и многооконность", anyui.FontSize(16), anyui.Bold())
		ui.RowCentered(8, 38, func() {
			if ui.Button("win_min", "Minimize") {
				ui.Window().Minimize()
			}
			if ui.Button("win_max", "Maximize / Restore") {
				if ui.Window().IsMaximized() {
					ui.Window().Restore()
				} else {
					ui.Window().Maximize()
				}
			}
			if ui.Button("win_fs", "Fullscreen (Esc / F11)") {
				ui.Window().SetFullscreen(!ui.Window().IsFullscreen())
			}
			if ui.Button("win_tray", "Скрыть в трей") {
				ui.Window().Hide()
			}
			if ui.Button("win_child", "Открыть дочернее окно", anyui.Primary()) {
				st.childCount++
				num := st.childCount
				_, _ = st.app.OpenWindow(anyui.Config{
					Title:  fmt.Sprintf("Дочернее окно #%d", num),
					Width:  420,
					Height: 260,
					Parent: ui.Window(),
					Theme:  ui.Theme(),
				}, func(cui *anyui.UI) {
					cui.Padding(20, func() {
						cui.Column(12, func() {
							cui.Label(fmt.Sprintf("Независимое окно #%d", num), anyui.FontSize(16), anyui.Bold())
							cui.Label("Закрытие этого окна не завершает главное приложение.")
							if cui.Button("ch_close", "Закрыть окно", anyui.Primary()) {
								cui.Window().Close()
							}
						})
					})
				})
			}
		})
	})
}
