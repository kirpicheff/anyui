package main

import (
	"fmt"
	"strings"

	"github.com/kirpicheff/anyui"
)

func (st *demoState) drawTabInputs(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_inputs", 0, 0, func() {
		ui.Label("Однострочные поля ввода (TextInput) — Undo/Redo, слова, ПКМ-меню, валидация", anyui.FontSize(16), anyui.Bold())
		ui.Row(14, 38, func() {
			ui.TextInput("ti_main", &st.textVal, "Обычное поле...", anyui.Width(240), anyui.Clearable(true), anyui.Icon(svgStar))
			ui.TextInput("ti_pass", &st.passVal, "Пароль...", anyui.Width(200), anyui.PasswordToggle(), anyui.Clearable(true))
		})
		emailErr := ""
		if !strings.Contains(st.emailVal, "@") || !strings.Contains(st.emailVal, ".") {
			emailErr = "Введите корректный email (например, user@domain.ru)"
		}
		ui.Row(14, 58, func() {
			onlyDigits := func(r rune) bool { return r >= '0' && r <= '9' }
			ui.TextInput("ti_digits", &st.digitsVal, "Только цифры", anyui.Width(160), anyui.InputFilter(onlyDigits), anyui.Mono())
			ui.TextInput("ti_email", &st.emailVal, "Email...", anyui.Width(240), anyui.Error(emailErr), anyui.Clearable(true))
			ui.TextInput("ti_ro", &st.readOnlyVal, "", anyui.Width(240), anyui.ReadOnly(true))
		})

		ui.Separator()
		ui.Label("Ввод по маске (MaskedInput)", anyui.FontSize(16), anyui.Bold())
		ui.Row(14, 38, func() {
			ui.MaskedInput("mi_phone", &st.phoneMaskVal, "+7 (000) 000-00-00", anyui.Width(200), anyui.Mono())
			ui.MaskedInput("mi_date", &st.dateMaskVal, "00.00.0000", anyui.Width(140), anyui.Mono())
			ui.MaskedInput("mi_ip", &st.ipMaskVal, "000.000.000.000", anyui.Width(170), anyui.Mono())
		})

		ui.Separator()
		ui.Label("Многострочный ввод (TextArea с нумерацией строк)", anyui.FontSize(16), anyui.Bold())
		ui.TextArea("ta_notes", &st.notesVal, "Введите заметку...", anyui.Height(120), anyui.LineNumbers(true))

		ui.Separator()
		ui.Label("SelectableLabel, WordWrap / Ellipsis и Hyperlink", anyui.FontSize(16), anyui.Bold())
		ui.Card(func() {
			ui.SelectableLabel("sel_id", "ID сессии: 8f4c-2a19-b70e | IP: 10.24.18.5 | Копируется мышью, Ctrl+C и ПКМ", anyui.Mono())
			ui.Label("Многострочный Label с WordWrap: длинный текст автоматически переносится по ширине карточки без обрезки слов за границей контейнера.", anyui.WordWrap(true), anyui.TextColor(th.TextMuted))
			ui.Label("Длинный заголовок с Ellipsis: этот текст не помещается в 260dp и аккуратно обрезается с многоточием", anyui.Width(260), anyui.Ellipsis(true))
			ui.Row(14, 24, func() {
				if ui.Hyperlink("hl_docs", "Открыть документацию AnyUI") {
					st.linkClicks++
					st.statusMsg = fmt.Sprintf("Переход по ссылке (%d)", st.linkClicks)
					ui.ShowToast("Hyperlink", "Клик по ссылке зарегистрирован", anyui.ToastInfo)
				}
				if ui.Hyperlink("hl_reset", "Сбросить ошибку Email") {
					st.emailVal = "user@anyui.dev"
					st.statusMsg = "Email сброшен на корректный"
				}
			})
		}, anyui.AutoHeight(), anyui.Padding(12))

		ui.Separator()
		ui.Label("Выпадающие списки (Select и Select с anyui.Searchable())", anyui.FontSize(16), anyui.Bold())
		opts := []anyui.SelectItem{
			{Key: "win11", Label: "Windows 11"},
			{Key: "win2022", Label: "Windows Server 2022"},
			{Key: "win10", Label: "Windows 10"},
			{Key: "win7", Label: "Windows 7"},
			{Key: "linux", Label: "Linux X11"},
			{Key: "macos", Label: "macOS"},
		}
		ui.Row(14, 38, func() {
			ui.Select("sel_os", opts, &st.selectKey, anyui.Width(220))
			ui.Select("cmb_os", opts, &st.comboKey, anyui.Searchable(), anyui.Width(240))
		})

		ui.Separator()
		ui.Label("Виртуализированный список (ListView — 500 строк, стрелки/PageUp/PageDown/Home/End, DoubleClick/MiddleClick)", anyui.FontSize(15), anyui.Bold())
		if ui.ListView("lv_demo", len(st.listItems), 32, &st.listSelected, func(idx int, area anyui.Rect, selected bool) {
			col := th.TextPrimary
			dotCol := th.TextMuted
			if selected {
				col = th.Accent
				dotCol = th.Accent
			}
			ui.Canvas().FillCircle(float32(area.X+ui.Scale(14).Int()), float32(area.Y+area.H/2), ui.ScaleF(4), dotCol)
			ui.DrawText(area.X+ui.Scale(26).Int(), area.Y+ui.Scale(8).Int(), st.listItems[idx], 13, col)
		}, anyui.Height(140)) {
			if st.listSelected >= 0 && st.listSelected < len(st.listItems) {
				st.statusMsg = fmt.Sprintf("Выбран %s", st.listItems[st.listSelected])
				if ui.DoubleClicked() {
					ui.ShowToast("ListView DoubleClick", st.listItems[st.listSelected], anyui.ToastInfo)
				}
			}
		}

		ui.Separator()
		ui.Label("Горизонтальный ScrollView (Shift+Колесо или WheelX)", anyui.FontSize(15), anyui.Bold())
		ui.ScrollView("sv_horiz_demo", 0, 74, func() {
			ui.Row(8, 36, func() {
				for i := 1; i <= 10; i++ {
					ui.Button(fmt.Sprintf("hbtn_%d", i), fmt.Sprintf("Элемент #%d", i), anyui.Width(120))
				}
			})
		})
	})
}
