package main

import (
	"fmt"

	"github.com/kirpicheff/anyui"
)

// openTitleBarTabsDemo открывает отдельное безрамочное окно: при скрытом тулбаре вкладки
// отображаются в заголовке (LeftContent), при показанном тулбаре — скрыты.
func (st *demoState) openTitleBarTabsDemo(parent *anyui.UI) {
	var winUI *anyui.UI
	tabs := []anyui.TabItem{
		{ID: "s1", Title: "Сервер 1", Closable: true},
		{ID: "s2", Title: "Сервер 2", Closable: true},
	}
	active := "s1"
	nextNum := 3
	toolbarVisible := true
	showAdd := true

	_, err := st.app.OpenWindow(anyui.Config{
		Title:     "Вкладки в заголовке",
		Width:     900,
		Height:    520,
		Theme:     parent.Theme(),
		Frameless: true,
		HitTest: func(x, y int) anyui.HitTestResult {
			if winUI != nil {
				return winUI.TitleBarHitTest(anyui.Px(x), anyui.Px(y))
			}
			return anyui.HitClient
		},
	}, func(ui *anyui.UI) {
		winUI = ui
		th := ui.Theme()

		cfg := anyui.TitleBarConfig{HeightDp: 36}
		if !toolbarVisible {
			cfg.LeftContent = func() {
				opts := []anyui.Option{}
				if !showAdd {
					opts = append(opts, anyui.HideAddButton())
				}
				ev := ui.TabBar("tbt_tabs", tabs, &active, opts...)
				if ev.Added {
					id := fmt.Sprintf("s%d", nextNum)
					tabs = append(tabs, anyui.TabItem{ID: id, Title: fmt.Sprintf("Сервер %d", nextNum), Closable: true})
					active = id
					nextNum++
				}
				if ev.ClosedID != "" && len(tabs) > 1 {
					for i, t := range tabs {
						if t.ID == ev.ClosedID {
							tabs = append(tabs[:i], tabs[i+1:]...)
							if active == ev.ClosedID {
								idx := i
								if idx >= len(tabs) {
									idx = len(tabs) - 1
								}
								active = tabs[idx].ID
							}
							break
						}
					}
				}
			}
		}
		ui.TitleBar("tbt_title", "Удалённый рабочий стол", cfg)

		if toolbarVisible {
			ui.PaddingXY(12, 8, func() {
				ui.RowCentered(8, 32, func() {
					ui.Button("tbt_b1", "Подключить", anyui.Primary())
					ui.Button("tbt_b2", "Отключить", anyui.Secondary())
					ui.Checkbox("tbt_add", "Кнопка «+»", &showAdd)
					ui.FlexSpacer(140)
					if ui.Button("tbt_hide", "Скрыть тулбар", anyui.Ghost()) {
						toolbarVisible = false
					}
				})
			})
		}

		ui.Padding(16, func() {
			ui.Column(10, func() {
				ui.Label("Активная вкладка: "+active, anyui.FontSize(16), anyui.Bold())
				if toolbarVisible {
					ui.Label("Тулбар показан: вкладок в заголовке нет. Скройте тулбар, чтобы они появились в заголовке.", anyui.FontSize(12), anyui.TextColor(th.TextMuted), anyui.WordWrap())
				} else {
					ui.Label("Тулбар скрыт: вкладки доступны в заголовке.", anyui.FontSize(12), anyui.TextColor(th.TextMuted))
					if ui.Button("tbt_show", "Показать тулбар", anyui.Primary()) {
						toolbarVisible = true
					}
				}
			})
		})
	})
	if err != nil {
		st.statusMsg = "Не удалось открыть окно: " + err.Error()
	}
}

