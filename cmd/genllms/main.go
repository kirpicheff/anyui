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

const modulePath = "github.com/kirpicheff/anyui"

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
		{"PageConfig", modulePath + ".PageConfig"},
		{"TrayConfig", modulePath + ".TrayConfig"},
		{"FlexItem", modulePath + ".FlexItem"},
		{"TableColumn", modulePath + ".TableColumn"},
		{"TabItem", modulePath + ".TabItem"},
		{"StatusBarSection", modulePath + ".StatusBarSection"},
		{"CanvasCallbacks", modulePath + ".CanvasCallbacks"},
		{"KeyEvent", modulePath + ".KeyEvent"},
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
	out.WriteString("- Импорт: `import \"github.com/kirpicheff/anyui\"` (`go get github.com/kirpicheff/anyui`).\n")
	out.WriteString("- Подпакеты `ui/`, `platform/`, `render/` — внутренние; прикладной код использует только корневой пакет `anyui`.\n")
	out.WriteString("- Для уточнения сигнатур любого типа или метода используй `go doc github.com/kirpicheff/anyui.<Symbol>` вместо чтения исходников.\n")
	out.WriteString("- Модель Immediate Mode: состояние хранится в переменных/структурах приложения, `drawFn(ui *anyui.UI)` перерисовывает UI на каждом кадре.\n")
	out.WriteString("- Идентификаторы виджетов: передавайте уникальный `id`, либо `\"\"` для автогенерации по счётчику в скоупе; в циклах и компонентах используйте `ui.IDScope(key, fn)` (или `PushID`/`PopID`).\n")
	out.WriteString("- Все размеры, координаты и отступы задаются в `dp` и масштабируются под DPI автоматически.\n")
	out.WriteString("- Обновление UI из фоновых горутин выполняется через `app.Dispatch(func() { ... })`, либо через `anyui.Remember(ui, key, loader)` прямо внутри `drawFn`.\n")
	out.WriteString("- Иконка окна подхватывается автоматически из `.syso`-ресурсов или задаётся через `Config.IconRGBA` / `win.SetIcon(...)`.\n")
	out.WriteString("- Раскладка без обрезания: для экранов с шапкой/телом/футером используйте `ui.Page(cfg, headerFn, bodyFn, footerFn)` или `ui.DockBottom` / `ui.DockTop` / `ui.ColumnFlex` и `anyui.FillHeight()`.\n")
	out.WriteString("- Модальные окна и диалоги: всегда используйте `ui.Modal(id, open, title, fn, opts...)`, который сам затемняет фон, центрирует окно, изолирует ввод и обрабатывает Escape/крестик. Запрещено создавать самодельные модалки через `PushOverlay(fullRect)` — передача полноэкранного прямоугольника в `PushOverlay` заблокирует весь экран на последующих кадрах.\n")
	out.WriteString("- `PushOverlay(bounds, fn)` предназначен только для всплывающих контролов (кастомных выпадающих списков, меню, тултипов), которым нужно отрисоваться поверх родительских контейнеров без обрезания клипом.\n")
	out.WriteString("- Горячие клавиши: `ui.Shortcut` регистрирует глобальный шорткат окна; не вызывайте его повторно на каждом кадре внутри циклических веток UI.\n")
	out.WriteString("- `Card(fn, ...)` по умолчанию подбирает высоту по содержимому (`AutoHeight`); чтобы растянуть карточку на весь остаток высоты, передайте `anyui.FillHeight()`.\n")
	out.WriteString("- В `Table` / `TableEx` колонки с `WidthDp <= 0` автоматически делят оставшуюся ширину таблицы.\n\n")
}

func writeQuickStart(out *bytes.Buffer) {
	out.WriteString("## Минимальный пример\n\n```go\n")
	out.WriteString(`package main

import (
	"log"

	"github.com/kirpicheff/anyui"
)

func main() {
	app := anyui.NewApp(anyui.Config{
		Title:  "My App",
		Width:  800,
		Height: 500,
		Theme:  anyui.NavyTheme(),
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

