package main

import (
	"fmt"

	"github.com/kirpicheff/anyui"
)

func (st *demoState) drawTabAdvanced(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_advanced", func() {
		ui.Label("Расширенный ввод: NumberInput, Slider, RangeSlider", anyui.FontSize(16), anyui.Bold())
		ui.RowCentered(14, 40, func() {
			ui.Label("Количество (Stepper):", anyui.FontSize(13))
			if ui.NumberInput("num_qty", &st.numVal, 0, 100, 2.5, 1, anyui.Width(150)) {
				st.statusMsg = fmt.Sprintf("NumberInput: %.1f", st.numVal)
			}
		})

		ui.Spacer(6)
		ui.RowCentered(12, 30, func() {
			ui.Label(fmt.Sprintf("Slider (%.0f%%):", st.sliderVal), anyui.FontSize(13), anyui.Width(130))
			if ui.Slider("sld_vol", &st.sliderVal, 0, 100, anyui.Width(280)) {
				st.statusMsg = fmt.Sprintf("Slider: %.0f", st.sliderVal)
			}
		})

		ui.RowCentered(12, 30, func() {
			ui.Label(fmt.Sprintf("Range (%.0f .. %.0f):", st.rangeLow, st.rangeHigh), anyui.FontSize(13), anyui.Width(130))
			if ui.RangeSlider("rsld_rng", &st.rangeLow, &st.rangeHigh, 0, 100, anyui.Width(280)) {
				st.statusMsg = fmt.Sprintf("Range: %.0f .. %.0f", st.rangeLow, st.rangeHigh)
			}
		})

		ui.Spacer(14)
		ui.Label("ColorPicker и DatePicker", anyui.FontSize(16), anyui.Bold())
		ui.RowCentered(16, 42, func() {
			ui.Label("Цвет:", anyui.FontSize(13))
			if ui.ColorPicker("cp_accent", &st.pickedColor, anyui.Width(150)) {
				st.statusMsg = fmt.Sprintf("Выбран цвет: #%02X%02X%02X", st.pickedColor.R, st.pickedColor.G, st.pickedColor.B)
			}
			ui.Label("Дата:", anyui.FontSize(13))
			if ui.DatePicker("dp_date", &st.pickedDate, anyui.Width(170)) {
				st.statusMsg = fmt.Sprintf("Выбрана дата: %s", st.pickedDate.Format("2006-01-02"))
			}
		})

		ui.Spacer(14)
		ui.Label("Обратная связь: Spinner и Toast-уведомления", anyui.FontSize(16), anyui.Bold())
		ui.RowCentered(14, 42, func() {
			ui.Spinner("spin_demo", 28)
			ui.Label("Загрузка...", anyui.FontSize(13), anyui.TextColor(th.TextMuted))
			ui.Spacer(10)
			if ui.Button("btn_t_info", "Info Toast", anyui.Primary()) {
				ui.ShowToast("Информация", "Система работает в штатном режиме", anyui.ToastInfo)
			}
			if ui.Button("btn_t_ok", "Success", anyui.Secondary()) {
				ui.ShowToast("Успешно", "Конфигурация сохранена", anyui.ToastSuccess)
			}
			if ui.Button("btn_t_warn", "Warning", anyui.Secondary()) {
				ui.ShowToast("Внимание", "Высокая загрузка канала", anyui.ToastWarning)
			}
			if ui.Button("btn_t_err", "Error", anyui.Danger()) {
				ui.ShowToast("Ошибка", "Таймаут подключения к узлу", anyui.ToastError)
			}
		})

		ui.Spacer(14)
		ui.Label("TabBar: Drag & Detach и вертикальный режим", anyui.FontSize(15), anyui.Bold())
		ui.Label("Попробуйте перетащить вкладку в верхнем TabBar влево/вправо (drag-reorder) или потянуть её вниз за пределы панели, чтобы открепить в отдельное окно!", anyui.FontSize(12), anyui.TextColor(th.TextMuted))
		ui.Spacer(8)
		if st.vertTabID == "" {
			st.vertTabID = "Общие"
		}
		if ui.Button("btn_tbt_demo", "Окно: вкладки в заголовке (при скрытом тулбаре)", anyui.Secondary()) {
			st.openTitleBarTabsDemo(ui)
		}
		ui.Spacer(8)
		ui.Row(14, 114, func() {
			ui.TabBar("tb_vert_demo", anyui.TabItems("Общие", "Сеть", "Вид"), &st.vertTabID,
				anyui.TabOrientation(anyui.Vertical),
				anyui.HideAddButton(),
				anyui.Width(160),
			)
			ui.Card(func() {
				ui.Label("Раздел настроек: "+st.vertTabID, anyui.FontSize(14), anyui.Bold())
				ui.Spacer(4)
				ui.Label("Пример вертикального TabBar с опциями TabOrientation(Vertical) и HideAddButton().", anyui.FontSize(12), anyui.TextColor(th.TextMuted), anyui.WordWrap())
			}, anyui.FillWidth(), anyui.Height(110))
		})
	})
}

