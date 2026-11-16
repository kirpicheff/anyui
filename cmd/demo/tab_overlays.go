package main

import (
	"fmt"

	"anyui"
	"anyui/render"
)

func (st *demoState) drawTabOverlays(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_overlays", 0, 0, func() {
		ui.Label("Модальные окна, Popover, FloatingPanel и Drawer", anyui.FontSize(16), anyui.Bold())
		ui.RowCentered(10, 40, func() {
			if ui.Button("ov_modal", "Открыть Modal", anyui.Primary()) {
				st.modalOpen = true
			}
			if ui.Button("ov_pop", "Открыть Popover") {
				st.popoverOpen = !st.popoverOpen
			}
			popAnchor := ui.CurrentArea()
			popAnchor.W = ui.S(160)
			popAnchor.H = ui.S(38)
			ui.Popover("pop_demo", &st.popoverOpen, popAnchor, func() {
				ui.Label("Это Popover!", anyui.Bold())
				ui.Toggle("pop_tgl", "Опция внутри", &st.toggleVal)
				if ui.Button("pop_ok", "Закрыть", anyui.Primary(), anyui.Height(30)) {
					st.popoverOpen = false
				}
			}, anyui.Width(230), anyui.Height(140))

			if ui.Button("ov_float", "FloatingPanel") {
				st.floatOpen = !st.floatOpen
			}
			if ui.Button("ov_drw_l", "Drawer Слева") {
				st.drawerLeft = true
			}
			if ui.Button("ov_drw_r", "Drawer Справа") {
				st.drawerRight = true
			}
		})

		ui.Separator()
		ui.Label("ContextMenu (ПКМ) и Tooltip (наведение)", anyui.FontSize(16), anyui.Bold())

		area := ui.CurrentArea()
		box := anyui.Rect{X: area.X, Y: area.Y + ui.S(120), W: ui.S(460), H: ui.S(74)}
		ui.CanvasView(460, 74, func(canvas *render.Canvas, r anyui.Rect) {
			box = r
			canvas.DrawRoundedBox(r, ui.Sf(8), th.Surface, 1.0, th.Border)
		})
		ui.Label(st.contextLog, anyui.FontSize(13), anyui.TextColor(th.TextMuted))

		ui.Tooltip("Подсказка Tooltip: нажмите правую кнопку мыши!", box)
		ui.ContextMenu("cm_demo", box, []anyui.ContextMenuItem{
			{Label: "Копировать", Shortcut: "Ctrl+C", OnClick: func() {
				st.contextLog = "Выбрано: Копировать"
				st.statusMsg = st.contextLog
			}},
			{Label: "Вставить", Shortcut: "Ctrl+V", OnClick: func() {
				st.contextLog = "Выбрано: Вставить"
				st.statusMsg = st.contextLog
			}},
			{Separator: true},
			{Label: "Заблокированный пункт", Disabled: true},
			{Label: "Сбросить счётчик", OnClick: func() {
				st.clickCount = 0
				st.contextLog = fmt.Sprintf("Счётчик сброшен (%d)", st.clickCount)
			}},
		})

		ui.Separator()
		ui.Label("ScrollView (прокрутка колесом и перетаскиванием ползунка)", anyui.FontSize(16), anyui.Bold())
		ui.ScrollView("sv_inner", 460, 140, func() {
			for i := 1; i <= 15; i++ {
				ui.Card(func() {
					ui.Label(fmt.Sprintf("Карточка внутри ScrollView #%d", i))
				}, anyui.Padding(10), anyui.Height(44))
			}
		})
	})
}

func (st *demoState) drawGlobalOverlays(ui *anyui.UI) {
	ui.Modal("mdl_demo", &st.modalOpen, "Модальное окно AnyUI", func() {
		ui.Label("Фон затемнён, клики под модалкой заблокированы.")
		ui.Label("Нажмите Escape или кнопку ниже для закрытия.", anyui.TextColor(ui.Theme().TextMuted))
		ui.Spacer(16)
		ui.Row(10, 38, func() {
			if ui.Button("mdl_ok", "Понятно", anyui.Primary()) {
				st.modalOpen = false
			}
			if ui.Button("mdl_cancel", "Отмена", anyui.Secondary()) {
				st.modalOpen = false
			}
		})
	}, anyui.Width(460), anyui.Height(220))

	ui.FloatingPanel("fp_demo", "Плавающая панель", &st.floatOpen, func() {
		ui.Label("Перетаскивается за заголовок и сворачивается кнопкой '-'.")
		ui.ProgressBar(st.progressVal, anyui.Width(220))
	}, anyui.Width(280), anyui.Height(160))

	ui.Drawer("drw_left", &st.drawerLeft, anyui.DrawerLeft, func() {
		ui.Label("Левый Drawer", anyui.FontSize(16), anyui.Bold())
		ui.Separator()
		ui.Label("Выезжающая панель навигации.")
		if ui.Button("drw_l_close", "Закрыть", anyui.Primary()) {
			st.drawerLeft = false
		}
	})

	ui.Drawer("drw_right", &st.drawerRight, anyui.DrawerRight, func() {
		ui.Label("Правый Drawer", anyui.FontSize(16), anyui.Bold())
		ui.Separator()
		ui.Label("Панель свойств или инспектора.")
		if ui.Button("drw_r_close", "Закрыть", anyui.Primary()) {
			st.drawerRight = false
		}
	})
}
