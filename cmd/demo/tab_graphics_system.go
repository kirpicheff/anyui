package main

import (
	"fmt"
	"math"

	"github.com/kirpicheff/anyui"
)

func (st *demoState) drawTabGraphicsSystem(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_gfx", func() {
		ui.Label("Графика Фазы 2 (Градиенты, Тени, Blur, Вектор, SVG, Image)", anyui.FontSize(16), anyui.Bold())

		ui.Row(14, 150, func() {
			ui.CanvasView(func(c *anyui.Canvas, r anyui.Rect) {
				c.DrawRoundedBox(r, ui.ScaleF(10), th.SurfaceDark, 1.0, th.Border)

				// 1. Линейный градиент + тень
				g1 := anyui.Rect{X: r.X + ui.Scale(14).Int(), Y: r.Y + ui.Scale(16).Int(), W: ui.Scale(120).Int(), H: ui.Scale(70).Int()}
				c.DrawBoxShadow(g1, anyui.BoxShadow{
					OffsetY: ui.ScaleF(4),
					Blur:    ui.ScaleF(10),
					Color:   anyui.RGBA(0, 0, 0, 120),
					Radii:   anyui.UniformRadii(ui.ScaleF(10)),
				})
				c.FillLinearGradient(g1, anyui.CornerRadii{TL: ui.ScaleF(14), TR: ui.ScaleF(4), BR: ui.ScaleF(14), BL: ui.ScaleF(4)}, anyui.LinearGradient{
					StartX: 0, StartY: 0, EndX: 1, EndY: 1,
					Stops: []anyui.GradientStop{
						{Pos: 0, Color: th.Accent},
						{Pos: 1, Color: anyui.RGB(168, 85, 247)},
					},
				})

				// 2. BackdropBlur поверх градиента
				blurR := anyui.Rect{X: g1.X + ui.Scale(40).Int(), Y: g1.Y + ui.Scale(34).Int(), W: ui.Scale(100).Int(), H: ui.Scale(54).Int()}
				c.BackdropBlur(blurR, ui.Scale(6).Int(), anyui.UniformRadii(ui.ScaleF(8)))
				c.DrawRoundedBox(blurR, ui.ScaleF(8), anyui.RGBA(255, 255, 255, 28), 1.0, anyui.RGBA(255, 255, 255, 70))

				// 3. Радиальный градиент
				g2 := anyui.Rect{X: r.X + ui.Scale(175).Int(), Y: r.Y + ui.Scale(16).Int(), W: ui.Scale(90).Int(), H: ui.Scale(90).Int()}
				c.FillRadialGradient(g2, anyui.UniformRadii(ui.ScaleF(45)), anyui.RadialGradient{
					CenterX: 0.4, CenterY: 0.4, Radius: 0.6,
					Stops: []anyui.GradientStop{
						{Pos: 0, Color: anyui.RGB(56, 189, 248)},
						{Pos: 1, Color: anyui.RGB(30, 58, 138)},
					},
				})

				// 4. Векторные примитивы (Polyline + Arc) и SVG с tint
				pts := []anyui.PointF{
					{X: float32(r.X + ui.Scale(285).Int()), Y: float32(r.Y + ui.Scale(85).Int())},
					{X: float32(r.X + ui.Scale(315).Int()), Y: float32(r.Y + ui.Scale(45).Int())},
					{X: float32(r.X + ui.Scale(345).Int()), Y: float32(r.Y + ui.Scale(68).Int())},
					{X: float32(r.X + ui.Scale(375).Int()), Y: float32(r.Y + ui.Scale(25).Int())},
				}
				c.DrawPolyline(pts, ui.ScaleF(2.5), th.Success, false)
				c.DrawArc(float32(r.X+ui.Scale(415).Int()), float32(r.Y+ui.Scale(55).Int()), ui.ScaleF(22), 0, math.Pi*1.45, ui.ScaleF(3.5), th.Warning)
				c.DrawSVG(svgStar, r.X+ui.Scale(285).Int(), r.Y+ui.Scale(95).Int(), ui.Scale(28).Int(), th.Warning)
				c.DrawSVG(svgGear, r.X+ui.Scale(325).Int(), r.Y+ui.Scale(95).Int(), ui.Scale(28).Int(), th.Accent)
			}, anyui.Width(460), anyui.Height(145))

			ui.Column(6, func() {
				ui.Label("Image:", anyui.FontSize(12), anyui.TextColor(th.TextMuted))
				ui.Image(st.sampleImg, anyui.ScaleFit, anyui.Width(96), anyui.Height(96))
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
			if ui.Button("win_notify", "Нативное уведомление ОС") {
				if st.app != nil {
					_ = st.app.ShowNotification(anyui.Notification{
						Title:   "AnyUI Demo",
						Message: "Нативное системное уведомление ОС работает!",
						Icon:    anyui.NotifyInfo,
					})
				}
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
