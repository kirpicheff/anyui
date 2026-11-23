package main

import (
	"fmt"

	"github.com/kirpicheff/anyui"
)

func (st *demoState) drawTabOverlays(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_overlays", func() {
		ui.Label("Модальные окна, Popover, FloatingPanel и Drawer", anyui.FontSize(16), anyui.Bold())
		ui.RowCentered(10, 40, func() {
			if ui.Button("ov_modal", "Открыть Modal", anyui.Primary(), anyui.Width(140)) {
				st.modalOpen = true
			}
			if ui.Button("ov_pop", "Открыть Popover", anyui.Width(160)) {
				st.popoverOpen = !st.popoverOpen
			}
			popAnchor := ui.CurrentArea()
			popAnchor.X += 150
			popAnchor.W = 160
			popAnchor.H = 38
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
			if ui.Button("ov_drw_t", "Drawer Сверху") {
				st.drawerTop = true
			}
			if ui.Button("ov_drw_b", "Drawer Снизу") {
				st.drawerBottom = true
			}
		})

		ui.Separator()
		ui.Label("ContextMenu (ПКМ) и Tooltip (наведение)", anyui.FontSize(16), anyui.Bold())

		var box anyui.Rect
		ui.CanvasView(func(canvas *anyui.Canvas, r anyui.Rect) {
			box = pxRectToDp(ui, r)
			canvas.DrawRoundedBox(r, ui.ScaleF(8), th.Surface, 1.0, th.Border)
		}, anyui.Width(460), anyui.Height(74))
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
		ui.ScrollView("sv_inner", func() {
			for i := 1; i <= 15; i++ {
				ui.Card(func() {
					ui.Label(fmt.Sprintf("Карточка внутри ScrollView #%d", i))
				}, anyui.Padding(10), anyui.Height(44))
			}
		}, anyui.Width(460), anyui.Height(140))
	})
}

func (st *demoState) drawGlobalOverlays(ui *anyui.UI) {
	ui.Modal("mdl_demo", &st.modalOpen, "Модальное окно AnyUI — Кросс-виджетный Drag & Drop в оверлее", func() {
		ui.Label("Фон затемнён. Попробуйте перетащить строку из левой таблицы в правую:")
		ui.Label("Ghost-плашка следует за курсором, а drop гарантированно срабатывает внутри оверлея.", anyui.TextColor(ui.Theme().TextMuted))
		ui.Spacer(8)

		cols := []anyui.TableColumn{{Title: "Элемент", WidthDp: 240}}
		ui.Row(12, 170, func() {
			ui.Column(4, func() {
				ui.Label(fmt.Sprintf("Доступные файлы (%d)", len(st.modalSrcItems)), anyui.Bold())
				ui.TableEx("mdl_tbl_src", cols, len(st.modalSrcItems), nil, nil, func(row, col int, area anyui.Rect, isSel bool) {
					ui.DrawText(anyui.Px(area.X+8), anyui.Px(area.Y+7), st.modalSrcItems[row], 12, ui.Theme().TextPrimary)
				}, anyui.Width(260), anyui.Height(140), anyui.DragSource("modal_item", func(idx int, _ string) anyui.DragPayload {
					return anyui.DragPayload{Type: "modal_item", Data: idx, Label: st.modalSrcItems[idx]}
				}))
			})

			ui.Column(4, func() {
				ui.Label(fmt.Sprintf("Очередь передачи (%d)", len(st.modalDstItems)), anyui.Bold())
				ui.TableEx("mdl_tbl_dst", cols, len(st.modalDstItems), nil, nil, func(row, col int, area anyui.Rect, isSel bool) {
					ui.DrawText(anyui.Px(area.X+8), anyui.Px(area.Y+7), st.modalDstItems[row], 12, ui.Theme().TextPrimary)
				}, anyui.Width(260), anyui.Height(140), anyui.AcceptDrop("modal_item"), anyui.OnDrop(func(ev anyui.DragEvent) {
					if ev.Payload != nil && ev.FromIndex >= 0 && ev.FromIndex < len(st.modalSrcItems) {
						item := st.modalSrcItems[ev.FromIndex]
						st.modalDstItems = append(st.modalDstItems, item)
						ui.ShowToast("Перенос в Modal", fmt.Sprintf("Добавлен: %s", item), anyui.ToastSuccess)
					}
				}))
			})
		})

		ui.Spacer(12)
		ui.Row(10, 34, func() {
			if ui.Button("mdl_ok", "Закрыть", anyui.Primary()) {
				st.modalOpen = false
			}
			if ui.Button("mdl_reset", "Сбросить списки", anyui.Secondary()) {
				st.modalSrcItems = []string{"Модуль core.sys", "Конфиг config.yaml", "База state.db", "Журнал audit.log"}
				st.modalDstItems = []string{"Архив backup.tar"}
			}
		})
	}, anyui.Width(580), anyui.Height(380))

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

	ui.Drawer("drw_top", &st.drawerTop, anyui.DrawerTop, func() {
		ui.Label("Верхний Drawer (баннер)", anyui.FontSize(16), anyui.Bold())
		ui.Separator()
		ui.Label("Выезжающая сверху панель оповещения или быстрого действия.")
		if ui.Button("drw_t_close", "Закрыть", anyui.Primary()) {
			st.drawerTop = false
		}
	}, anyui.Height(180))

	ui.Drawer("drw_bottom", &st.drawerBottom, anyui.DrawerBottom, func() {
		ui.Label("Нижний Drawer (Bottom Sheet)", anyui.FontSize(16), anyui.Bold())
		ui.Separator()
		ui.Label("Выезжающая снизу панель действий или выбора.")
		if ui.Button("drw_b_close", "Закрыть", anyui.Primary()) {
			st.drawerBottom = false
		}
	}, anyui.Height(200))
}
