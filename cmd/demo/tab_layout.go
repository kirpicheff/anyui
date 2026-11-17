package main

import (
	"fmt"

	"github.com/kirpicheff/anyui"
)

func (st *demoState) drawTabLayout(ui *anyui.UI) {
	th := ui.Theme()
	ui.ScrollView("sv_layout", 0, 0, func() {
		ui.Label("Продвинутая раскладка (Фаза 6.1)", anyui.FontSize(16), anyui.Bold())

		// 1. RowFlex: фиксированный слот + пропорциональные веса
		ui.Label("RowFlex (Fixed 120dp + Weight 1 + Weight 2)", anyui.FontSize(13), anyui.TextColor(th.TextMuted))
		ui.RowFlex(8, 38,
			anyui.FlexItem{FixedDp: 120, Fn: func() {
				ui.Button("lf_fixed", "Fixed 120", anyui.Secondary(), anyui.FillWidth())
			}},
			anyui.FlexItem{Weight: 1, Fn: func() {
				ui.Button("lf_w1", "Weight 1 (1/3)", anyui.Primary(), anyui.FillWidth())
			}},
			anyui.FlexItem{Weight: 2, Fn: func() {
				ui.Button("lf_w2", "Weight 2 (2/3)", anyui.Ghost(), anyui.FillWidth())
			}},
		)

		ui.Separator()

		// 2. Grid: 3 колонки с фиксированной высотой строки
		ui.Label("Grid (3 колонки, авторазмещение ячеек)", anyui.FontSize(13), anyui.TextColor(th.TextMuted))
		ui.Grid(3, 10, 10, 68, func() {
			cards := []struct {
				title string
				val   string
			}{
				{"CPU", "12%"},
				{"RAM", "1.4 GB"},
				{"Сеть", "48 Mbps"},
				{"Аптайм", "14д 06ч"},
				{"Сессии", "128"},
				{"Ошибки", "0"},
			}
			for _, item := range cards {
				it := item
				ui.Card(func() {
					ui.Label(it.title, anyui.FontSize(11), anyui.TextColor(th.TextMuted))
					ui.Label(it.val, anyui.FontSize(16), anyui.Bold())
				}, anyui.Padding(10))
			}
		})

		ui.Separator()

		// 3. Flow: автоперенос тегов/кнопок при нехватке ширины
		ui.Label("Flow (автоперенос при изменении ширины окна)", anyui.FontSize(13), anyui.TextColor(th.TextMuted))
		ui.Flow(8, 8, func() {
			tags := []string{"Go", "Pure Software", "No WebView2", "Windows", "DPI Aware", "Immediate Mode", "Zero-CPU Idle", "SVG Icons", "Glassmorphism", "Flex & Grid"}
			for i, tag := range tags {
				if ui.Button(fmt.Sprintf("flow_tag_%d", i), tag, anyui.Secondary(), anyui.Height(28), anyui.Padding(10)) {
					st.statusMsg = "Выбран тег: " + tag
				}
			}
		})

		ui.Separator()

		// 4. FormLayout + FormRow + AutoHeight Card
		ui.Label("FormLayout + Card (AutoHeight)", anyui.FontSize(13), anyui.TextColor(th.TextMuted))
		ui.Card(func() {
			ui.FormLayout(130, 12, 8, func() {
				ui.FormRow("Сервер:", func() {
					ui.TextInput("frm_host", &st.formHost, "host", anyui.Width(260))
				})
				ui.FormRow("Порт:", func() {
					ui.TextInput("frm_port", &st.formPort, "443", anyui.Width(120))
				})
				ui.FormRow("Токен доступа:", func() {
					ui.TextInput("frm_tok", &st.formToken, "Введите токен", anyui.Password(), anyui.Width(260))
				})
			})
		}, anyui.Padding(14), anyui.AutoHeight())

		ui.Separator()

		// 5. RowWith: JustifySpaceBetween и выравнивание
		ui.Label("RowWith (JustifySpaceBetween + AlignCenter)", anyui.FontSize(13), anyui.TextColor(th.TextMuted))
		ui.RowWith(anyui.AlignCenter, anyui.JustifySpaceBetween, 8, 42,
			func() {
				ui.Button("rw_left", "Слева", anyui.Secondary(), anyui.MinWidth(90))
			},
			func() {
				ui.Badge("По центру", anyui.Primary())
			},
			func() {
				ui.Button("rw_right", "Справа", anyui.Primary(), anyui.MinWidth(90))
			},
		)
	})
}

