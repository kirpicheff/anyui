package main

import (
	"fmt"

	"anyui"
)

func (st *demoState) drawTabInputs(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_inputs", 0, 0, func() {
		ui.Label("Однострочные поля ввода (TextInput)", anyui.FontSize(16), anyui.Bold())
		ui.Row(14, 38, func() {
			ui.TextInput("ti_main", &st.textVal, "Обычное поле...", anyui.Width(240), anyui.Clearable(true), anyui.Icon(svgStar))
			ui.TextInput("ti_pass", &st.passVal, "Пароль...", anyui.Width(200), anyui.Password(), anyui.Clearable(true))
		})
		ui.Row(14, 38, func() {
			onlyDigits := func(r rune) bool { return r >= '0' && r <= '9' }
			ui.TextInput("ti_digits", &st.digitsVal, "Только цифры", anyui.Width(180), anyui.InputFilter(onlyDigits), anyui.Mono())
			ui.TextInput("ti_ro", &st.readOnlyVal, "", anyui.Width(260), anyui.ReadOnly(true))
		})

		ui.Separator()
		ui.Label("Многострочный ввод (TextArea)", anyui.FontSize(16), anyui.Bold())
		ui.TextArea("ta_notes", &st.notesVal, "Введите заметку...", anyui.Height(105))

		ui.Separator()
		ui.Label("Выпадающие списки (Select и ComboBox с поиском)", anyui.FontSize(16), anyui.Bold())
		opts := []string{"Windows 11", "Windows Server 2022", "Windows 10", "Windows 7", "Linux X11", "macOS"}
		ui.Row(14, 38, func() {
			ui.Select("sel_os", opts, &st.selectIdx, anyui.Width(220))
			ui.ComboBox("cmb_os", opts, &st.comboIdx, anyui.Width(240))
		})

		ui.Separator()
		ui.Label("Виртуализированный список (ListView — 500 строк)", anyui.FontSize(16), anyui.Bold())
		ui.ListView("lv_demo", len(st.listItems), 32, &st.listSelected, func(idx int, area anyui.Rect, selected bool) {
			col := th.TextPrimary
			dotCol := th.TextMuted
			if selected {
				col = th.Accent
				dotCol = th.Accent
				st.statusMsg = fmt.Sprintf("Выбран %s", st.listItems[idx])
			}
			ui.Canvas().FillCircle(float32(area.X+ui.S(14)), float32(area.Y+area.H/2), ui.Sf(4), dotCol)
			ui.DrawText(area.X+ui.S(26), area.Y+ui.S(8), st.listItems[idx], 13, col)
		}, anyui.Height(150))
	})
}
