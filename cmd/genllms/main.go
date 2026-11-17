// Утилита genllms генерирует компактный файл llms.txt в корне модуля
// на основе актуального публичного API корневого пакета (через go doc).
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const modulePath = "git.gtw.su/kirpich/anyui"

func main() {
	version := detectVersion()

	var out bytes.Buffer
	writeHeader(&out, version)
	writeQuickStart(&out)

	sections := []struct {
		title  string
		target string
	}{
		{"Корневой пакет (константы, функции, типы, опции)", modulePath},
		{"Config", modulePath + ".Config"},
		{"TrayConfig", modulePath + ".TrayConfig"},
		{"FlexItem", modulePath + ".FlexItem"},
		{"TableColumn", modulePath + ".TableColumn"},
		{"TabItem", modulePath + ".TabItem"},
		{"StatusBarSection", modulePath + ".StatusBarSection"},
		{"App", modulePath + ".App"},
		{"Window", modulePath + ".Window"},
		{"UI (Immediate Mode API)", modulePath + ".UI"},
	}

	for _, sec := range sections {
		doc, err := runGoDoc(sec.target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка go doc %s: %v\n", sec.target, err)
			os.Exit(1)
		}
		fmt.Fprintf(&out, "## %s\n\n```go\n%s\n```\n\n", sec.title, doc)
	}

	if err := os.WriteFile("llms.txt", out.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка записи llms.txt: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("llms.txt обновлён (версия: %s)\n", version)
}

// detectVersion возвращает последний git-тег или "dev", если тегов нет.
func detectVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	raw, err := cmd.Output()
	if err != nil {
		return "dev"
	}
	ver := strings.TrimSpace(string(raw))
	if ver == "" {
		return "dev"
	}
	return ver
}

// runGoDoc выполняет go doc для указанного символа или пакета.
func runGoDoc(target string) (string, error) {
	cmd := exec.Command("go", "doc", target)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(raw)))
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	return strings.TrimSpace(text), nil
}

func writeHeader(out *bytes.Buffer, version string) {
	fmt.Fprintf(out, "# AnyUI (%s)\n\n", version)
	out.WriteString("Лёгкий кроссплатформенный UI-фреймворк на Go с программным 2D-рендерингом (без WebView2 и OpenGL/GPU).\n\n")
	out.WriteString("## Инструкция для ИИ-агента\n\n")
	out.WriteString("- Импорт: `import \"git.gtw.su/kirpich/anyui\"` (`go get git.gtw.su/kirpich/anyui`).\n")
	out.WriteString("- Подпакеты `ui/`, `platform/`, `render/` — внутренние; прикладной код использует только корневой пакет `anyui`.\n")
	out.WriteString("- Для уточнения сигнатур любого типа или метода используй `go doc git.gtw.su/kirpich/anyui.<Symbol>` вместо чтения исходников.\n")
	out.WriteString("- Модель Immediate Mode: состояние хранится в переменных/структурах приложения, `drawFn(ui *anyui.UI)` перерисовывает UI на каждом кадре.\n")
	out.WriteString("- Все интерактивные виджеты требуют уникальный строковый `id`.\n")
	out.WriteString("- Все размеры, координаты и отступы задаются в `dp` и масштабируются под DPI автоматически.\n")
	out.WriteString("- Обновление UI из фоновых горутин выполняется только через `app.Dispatch(func() { ... })`.\n")
	out.WriteString("- Иконка окна подхватывается автоматически из `.syso`-ресурсов или задаётся через `Config.IconRGBA` / `win.SetIcon(...)`.\n")
	out.WriteString("- Раскладка без обрезания: не угадывай `Height(...)` у списков/таблиц/карточек перед нижними кнопками — используй `ui.DockBottom`, `ui.DockTop`, `ui.ColumnFlex` / `ui.RowFlex` и `anyui.FillHeight()`.\n")
	out.WriteString("- `Card(fn, anyui.AutoHeight())` подбирает высоту по содержимому; без `AutoHeight()` карточка занимает весь остаток высоты.\n")
	out.WriteString("- В `Table` / `TableEx` колонки с `WidthDp <= 0` автоматически делят оставшуюся ширину таблицы.\n\n")
}

func writeQuickStart(out *bytes.Buffer) {
	out.WriteString("## Минимальный пример\n\n```go\n")
	out.WriteString(`package main

import (
	"log"

	"git.gtw.su/kirpich/anyui"
)

func main() {
	app := anyui.NewApp(anyui.Config{
		Title:  "My App",
		Width:  800,
		Height: 500,
		Theme:  anyui.DeepNavyTheme(),
	})

	name := ""
	err := app.Run(func(ui *anyui.UI) {
		ui.Padding(16, func() {
			ui.Column(10, func() {
				ui.Label("Привет, AnyUI", anyui.FontSize(18), anyui.Bold())
				ui.TextInput("inp_name", &name, "Введите имя", anyui.Width(260))
				if ui.Button("btn_ok", "Готово", anyui.Primary()) {
					log.Println("Имя:", name)
				}
			})
		})
	})
	if err != nil {
		log.Fatal(err)
	}
}
`)
	out.WriteString("```\n\n")
}

