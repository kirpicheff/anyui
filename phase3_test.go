package anyui

import "testing"

func TestWindowCloseBehavior(t *testing.T) {
	app := NewApp(Config{Title: "Test"})

	// 1. HideOnClose отменяет закрытие, пока приложение не выходит
	w := &Window{
		app:         app,
		hideOnClose: true,
	}
	if w.handleCloseRequest() {
		t.Fatal("при hideOnClose=true закрытие должно быть отменено")
	}

	// 2. Пользовательский OnClose может отменить закрытие
	called := false
	w2 := &Window{
		app: app,
		onClose: func(_ *Window) bool {
			called = true
			return false
		},
	}
	if w2.handleCloseRequest() {
		t.Fatal("onClose вернул false, закрытие должно быть отменено")
	}
	if !called {
		t.Fatal("обработчик onClose не был вызван")
	}

	// 3. При App.Quit() окно закрывается даже с hideOnClose=true
	app.quitting = true
	if !w.handleCloseRequest() {
		t.Fatal("при выходе из приложения окно должно закрываться безусловно")
	}
}

func TestConvertTrayMenu(t *testing.T) {
	app := NewApp(Config{Title: "TrayTest"})
	clicked := false
	items := app.convertTrayMenu([]MenuItem{
		{Label: "Показать", Checked: true, OnClick: func(a *App) {
			if a == app {
				clicked = true
			}
		}},
		{Separator: true},
		{Label: "Недоступно", Disabled: true},
	})

	if len(items) != 3 {
		t.Fatalf("ожидалось 3 пункта меню, получено %d", len(items))
	}
	if !items[0].Checked || !items[1].Separator || !items[2].Disabled {
		t.Fatal("флаги пунктов меню перенесены некорректно")
	}
	items[0].OnClick()
	if !clicked {
		t.Fatal("коллбэк пункта меню не сработал")
	}
}

