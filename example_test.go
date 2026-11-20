package anyui_test

import (
	"log"

	"github.com/kirpicheff/anyui"
)

func ExampleNewApp() {
	app := anyui.NewApp(anyui.Config{
		Title:  "My App",
		Width:  960,
		Height: 600,
		Theme:  anyui.NavyTheme(),
	})

	name := ""
	enabled := true

	err := app.Run(func(ui *anyui.UI) {
		ui.Padding(20, func() {
			ui.Column(12, func() {
				ui.Label("Настройки", anyui.FontSize(18))
				ui.TextInput("inp_name", &name, "Введите имя", anyui.Width(260))
				ui.Toggle("tgl_on", "Включить уведомления", &enabled)
				if ui.Button("btn_save", "Сохранить", anyui.Primary()) {
					log.Println("Сохранено:", name, enabled)
				}
			})
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleApp_Dispatch() {
	app := anyui.NewApp(anyui.Config{
		Title:  "Async Example",
		Width:  640,
		Height: 400,
	})

	status := "Ожидание..."

	go func() {
		// Обновление состояния из фоновой горутины безопасно через Dispatch.
		app.Dispatch(func() {
			status = "Готово"
		})
	}()

	_ = app.Run(func(ui *anyui.UI) {
		ui.Padding(16, func() {
			ui.Label(status)
		})
	})
}

func ExampleUI_Button() {
	app := anyui.NewApp(anyui.Config{
		Title:  "Button Example",
		Width:  400,
		Height: 200,
	})

	_ = app.Run(func(ui *anyui.UI) {
		ui.Padding(16, func() {
			if ui.Button("btn_ok", "OK", anyui.Primary(), anyui.Width(120)) {
				log.Println("Кнопка нажата")
			}
		})
	})
}

