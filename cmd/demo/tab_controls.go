package main

import (
	"fmt"

	"github.com/kirpicheff/anyui"
)

func (st *demoState) drawTabControls(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_controls", 0, 0, func() {
		ui.Label("Кнопки и варианты стилей", anyui.FontSize(16), anyui.Bold())
		ui.RowCentered(10, 40, func() {
			if ui.Button("btn_prim", "Primary", anyui.Primary(), anyui.Icon(svgStar)) {
				st.clickCount++
				st.statusMsg = fmt.Sprintf("Нажата Primary (%d)", st.clickCount)
			}
			if ui.Button("btn_sec", "Secondary", anyui.Secondary()) {
				st.statusMsg = "Нажата Secondary"
			}
			if ui.Button("btn_dng", "Danger", anyui.Danger()) {
				st.statusMsg = "Нажата Danger"
			}
			if ui.Button("btn_gh", "Ghost", anyui.Ghost()) {
				st.statusMsg = "Нажата Ghost"
			}
			ui.Button("btn_dis", "Disabled", anyui.Disabled(true))
			if ui.IconButton("btn_ic", svgPlus, anyui.Primary()) {
				st.clickCount++
				st.statusMsg = fmt.Sprintf("Нажата IconButton (%d)", st.clickCount)
			}
		})

		ui.Separator()
		ui.Label("Чекбоксы, радиокнопки и переключатель", anyui.FontSize(16), anyui.Bold())

		ui.Row(24, 120, func() {
			ui.Column(10, func() {
				ui.Checkbox("chk_1", "Включить уведомления", &st.chk1)
				ui.Checkbox("chk_2", "Неопределённое состояние", &st.chk2, anyui.Indeterminate(st.chkIndet))
				ui.Checkbox("chk_3", "Недоступный чекбокс", &st.chk1, anyui.Disabled(true))
				ui.Toggle("tgl_1", "Фоновая синхронизация", &st.toggleVal)
			})
			ui.Column(8, func() {
				ui.Label("Режим работы:", anyui.FontSize(13), anyui.TextColor(th.TextMuted))
				ui.RadioGroup("rg_mode", []string{"Производительность", "Баланс", "Энергосбережение"}, &st.radioIdx)
			})
		})

		ui.Separator()
		ui.Label("Индикаторы и прогресс-бар", anyui.FontSize(16), anyui.Bold())

		ui.RowCentered(10, 32, func() {
			ui.Badge("ONLINE", anyui.Primary())
			ui.Badge("WARNING", anyui.Danger())
			ui.Badge("MUTED", anyui.Secondary())
			ui.StatusDot(th.Success, "Сервер активен")
			ui.StatusDot(th.Warning, "Ожидание")
		})

		ui.RowCentered(10, 36, func() {
			if ui.Button("prg_dec", "- 10%", anyui.Width(72), anyui.Height(30)) {
				st.progressVal -= 0.1
				if st.progressVal < 0 {
					st.progressVal = 0
				}
			}
			ui.ProgressBar(st.progressVal, anyui.Width(320), anyui.Height(10))
			if ui.Button("prg_inc", "+ 10%", anyui.Width(72), anyui.Height(30), anyui.Primary()) {
				st.progressVal += 0.1
				if st.progressVal > 1 {
					st.progressVal = 1
				}
			}
			ui.Label(fmt.Sprintf("%.0f%%", st.progressVal*100), anyui.Mono())
		})
	})
}

