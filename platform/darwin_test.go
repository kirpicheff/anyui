//go:build darwin

package platform

import (
	"reflect"
	"testing"
)

func TestParseAppleInterfaceStyle(t *testing.T) {
	if !parseAppleInterfaceStyle("Dark\n") {
		t.Fatal("ожидалась тёмная тема для Dark")
	}
	if !parseAppleInterfaceStyle("dark") {
		t.Fatal("ожидалась тёмная тема без учёта регистра")
	}
	if parseAppleInterfaceStyle("") {
		t.Fatal("пустая строка должна означать светлую тему")
	}
	if parseAppleInterfaceStyle("Light") {
		t.Fatal("Light должен означать светлую тему")
	}
}

func TestEscapeAppleScriptString(t *testing.T) {
	in := "Путь \"C:\\test\"\nстрока"
	want := `Путь \"C:\\test\"\nстрока`
	if got := escapeAppleScriptString(in); got != want {
		t.Fatalf("получено %q, ожидалось %q", got, want)
	}
}

func TestExtractDarwinFileExtensions(t *testing.T) {
	filters := []FileFilter{
		{Name: "Картинки", Pattern: "*.png;*.jpg;*.png"},
		{Name: "Все", Pattern: "*.*"},
		{Name: "Текст", Pattern: ".txt"},
	}
	got := extractDarwinFileExtensions(filters)
	want := []string{"png", "jpg", "txt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("получено %v, ожидалось %v", got, want)
	}
}

func TestDarwinCoordsConversion(t *testing.T) {
	px, py := appKitPointToGoUIPixels(100, 150, 600, 2.0)
	if px != 200 || py != 900 {
		t.Fatalf("ожидалось (200, 900), получено (%d, %d)", px, py)
	}

	gx, gy, gw, gh := appKitWindowRectToGoUI(50, 100, 400, 300, 900, 2.0)
	if gx != 100 || gy != 1000 || gw != 800 || gh != 600 {
		t.Fatalf("ожидалось (100, 1000, 800, 600), получено (%d, %d, %d, %d)", gx, gy, gw, gh)
	}

	backY := goUITopLeftYToAppKit(1000, 600, 900, 2.0)
	if backY != 100 {
		t.Fatalf("ожидалось backY=100, получено %v", backY)
	}
}

func TestDarwinKeycodeAndModifiers(t *testing.T) {
	if got := darwinKeycodeToVK(0x00); got != 'A' {
		t.Fatalf("kVK_ANSI_A: ожидалось 'A', получено %#x", got)
	}
	if got := darwinKeycodeToVK(0x24); got != 0x0D {
		t.Fatalf("kVK_Return: ожидалось 0x0D, получено %#x", got)
	}
	if got := darwinKeycodeToVK(0x7B); got != 0x25 {
		t.Fatalf("kVK_LeftArrow: ожидалось 0x25, получено %#x", got)
	}
	if got := darwinKeycodeToVK(0xFFFF); got != 0 {
		t.Fatalf("неизвестный код должен возвращать 0, получено %#x", got)
	}

	mods := darwinModifiersFromFlags(nsEventModifierFlagShift | nsEventModifierFlagCommand | nsEventModifierFlagOption)
	if !mods.Shift || !mods.Ctrl || !mods.Alt {
		t.Fatalf("ожидались Shift, Ctrl (Command) и Alt (Option), получено %+v", mods)
	}
}

