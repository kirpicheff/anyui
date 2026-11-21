// Утилита genllms генерирует компактный llms.txt и полный офлайн-справочник llms-full.txt
// в корне модуля на основе AST публичного API корневого пакета anyui.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type aliasTarget struct {
	pkg  string
	name string
}

type apiIndex struct {
	fset        *token.FileSet
	rootFiles   []*ast.File
	subPkgTypes map[string]map[string]*ast.TypeSpec // pkg -> typeName -> TypeSpec
	aliases     map[string]aliasTarget              // publicName -> subpkg target
	localTypes  map[string]*ast.TypeSpec
	constGroups [][]string
	vars        []string
	optionFuncs []string
	otherFuncs  []string
	methods     map[string][]string // receiverType -> []fullSignature
}

func main() {
	versionFlag := flag.String("version", "", "переопределить версию в шапке (по умолчанию берётся из git describe)")
	flag.Parse()

	version := strings.TrimSpace(*versionFlag)
	if version == "" {
		version = detectVersion()
	}

	idx, err := parseAPI(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка разбора API: %v\n", err)
		os.Exit(1)
	}

	var shortOut bytes.Buffer
	writeShortLLMs(&shortOut, version, idx)
	if err := os.WriteFile("llms.txt", shortOut.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка записи llms.txt: %v\n", err)
		os.Exit(1)
	}

	var fullOut bytes.Buffer
	writeFullLLMs(&fullOut, version, idx)
	if err := os.WriteFile("llms-full.txt", fullOut.Bytes(), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка записи llms-full.txt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("llms.txt и llms-full.txt обновлены (версия: %s)\n", version)
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

func parseDirNonTest(fset *token.FileSet, dir string) ([]*ast.File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			names = append(names, filepath.Join(dir, n))
		}
	}
	sort.Strings(names)
	files := make([]*ast.File, 0, len(names))
	for _, p := range names {
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		files = append(files, f)
	}
	return files, nil
}

func parseAPI(rootDir string) (*apiIndex, error) {
	idx := &apiIndex{
		fset:        token.NewFileSet(),
		subPkgTypes: make(map[string]map[string]*ast.TypeSpec),
		aliases:     make(map[string]aliasTarget),
		localTypes:  make(map[string]*ast.TypeSpec),
		methods:     make(map[string][]string),
	}

	rootFiles, err := parseDirNonTest(idx.fset, rootDir)
	if err != nil {
		return nil, err
	}
	idx.rootFiles = rootFiles

	for _, sub := range []string{"ui", "platform", "render"} {
		subFiles, err := parseDirNonTest(idx.fset, filepath.Join(rootDir, sub))
		if err != nil {
			continue
		}
		idx.subPkgTypes[sub] = make(map[string]*ast.TypeSpec)
		for _, f := range subFiles {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, sp := range gd.Specs {
					ts, ok := sp.(*ast.TypeSpec)
					if ok && ast.IsExported(ts.Name.Name) {
						idx.subPkgTypes[sub][ts.Name.Name] = ts
					}
				}
			}
		}
	}

	for _, f := range rootFiles {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				switch d.Tok {
				case token.CONST:
					var grp []string
					for _, sp := range d.Specs {
						vs, ok := sp.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, name := range vs.Names {
							if ast.IsExported(name.Name) {
								grp = append(grp, name.Name)
							}
						}
					}
					if len(grp) > 0 {
						idx.constGroups = append(idx.constGroups, grp)
					}
				case token.VAR:
					for _, sp := range d.Specs {
						vs, ok := sp.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, name := range vs.Names {
							if ast.IsExported(name.Name) {
								idx.vars = append(idx.vars, name.Name)
							}
						}
					}
				case token.TYPE:
					for _, sp := range d.Specs {
						ts, ok := sp.(*ast.TypeSpec)
						if !ok || !ast.IsExported(ts.Name.Name) {
							continue
						}
						if ts.Assign.IsValid() {
							if sel, ok := ts.Type.(*ast.SelectorExpr); ok {
								if id, ok := sel.X.(*ast.Ident); ok {
									idx.aliases[ts.Name.Name] = aliasTarget{pkg: id.Name, name: sel.Sel.Name}
									continue
								}
							}
						}
						idx.localTypes[ts.Name.Name] = ts
					}
				}
			case *ast.FuncDecl:
				if !ast.IsExported(d.Name.Name) {
					continue
				}
				sig := idx.formatFuncSig(d)
				if d.Recv == nil || len(d.Recv.List) == 0 {
					if returnsOption(d) {
						idx.optionFuncs = append(idx.optionFuncs, sig)
					} else {
						idx.otherFuncs = append(idx.otherFuncs, sig)
					}
				} else {
					recvName := receiverTypeName(d.Recv.List[0].Type)
					if ast.IsExported(recvName) {
						idx.methods[recvName] = append(idx.methods[recvName], sig)
					}
				}
			}
		}
	}

	sort.Strings(idx.vars)
	sort.Strings(idx.optionFuncs)
	sort.Strings(idx.otherFuncs)
	for k := range idx.methods {
		sort.Strings(idx.methods[k])
	}
	return idx, nil
}

func returnsOption(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	id, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	return ok && id.Name == "Option"
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}

func (idx *apiIndex) formatFuncSig(fn *ast.FuncDecl) string {
	clone := *fn
	clone.Body = nil
	clone.Doc = nil
	var buf bytes.Buffer
	_ = format.Node(&buf, idx.fset, &clone)
	return sanitizeInternalTypes(strings.TrimSpace(buf.String()))
}

// formatTypeDef возвращает полное определение типа без префиксов внутренних подпакетов.
func (idx *apiIndex) formatTypeDef(name string) (string, bool) {
	if ts, ok := idx.localTypes[name]; ok {
		return idx.formatTypeSpec(name, ts), true
	}
	if target, ok := idx.aliases[name]; ok {
		if m, ok := idx.subPkgTypes[target.pkg]; ok {
			if ts, ok := m[target.name]; ok {
				return idx.formatTypeSpec(name, ts), true
			}
		}
	}
	return "", false
}

func (idx *apiIndex) formatTypeSpec(publicName string, ts *ast.TypeSpec) string {
	clone := *ts
	clone.Name = ast.NewIdent(publicName)
	clone.Doc = nil
	clone.Comment = nil
	decl := &ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{&clone},
	}
	var buf bytes.Buffer
	_ = format.Node(&buf, idx.fset, decl)
	s := strings.TrimSpace(buf.String())
	return sanitizeInternalTypes(s)
}

// sanitizeInternalTypes удаляет утечки имён внутренних подпакетов из отформатированных сигнатур и структур.
func sanitizeInternalTypes(s string) string {
	replacer := strings.NewReplacer(
		"platform.TrayNotification", "Notification",
		"platform.Modifiers", "Modifiers",
		"platform.MouseButton", "MouseButton",
		"platform.CursorType", "CursorType",
		"platform.HitTestResult", "HitTestResult",
		"platform.FileFilter", "FileFilter",
		"platform.WindowState", "WindowState",
		"platform.MessageBoxStyle", "MessageBoxStyle",
		"platform.MessageBoxButtons", "MessageBoxButtons",
		"platform.MessageBoxResult", "MessageBoxResult",
		"platform.NotificationIcon", "NotificationIcon",
		"render.Canvas", "Canvas",
		"render.Color", "Color",
		"render.Rect", "Rect",
		"render.CornerRadii", "CornerRadii",
		"render.GradientStop", "GradientStop",
		"render.LinearGradient", "LinearGradient",
		"render.RadialGradient", "RadialGradient",
		"render.BoxShadow", "BoxShadow",
		"render.PointF", "PointF",
		"render.ImageScaleMode", "ImageScaleMode",
		"render.FontStyle", "FontStyle",
		"ui.RawKeyEvent", "KeyEvent",
		"RawKeyEvent", "KeyEvent",
		"ui.", "",
		"render.", "",
		"platform.", "",
	)
	return replacer.Replace(s)
}

func writeCommonHeader(out *bytes.Buffer, version string, isFull bool) {
	if isFull {
		fmt.Fprintf(out, "# AnyUI (%s) — Полный справочник API\n\n", version)
	} else {
		fmt.Fprintf(out, "# AnyUI (%s)\n\n", version)
	}
	out.WriteString("Лёгкий кроссплатформенный UI-фреймворк на Go с программным 2D-рендерингом (без WebView2 и OpenGL/GPU).\n\n")
	if !isFull {
		out.WriteString("> Полный офлайн-справочник со всеми сигнатурами, опциями и структурами: `llms-full.txt`.\n\n")
	}
	out.WriteString("## Версия и источник истины\n\n")
	out.WriteString("- Импорт: `import \"github.com/kirpicheff/anyui\"` (`go get github.com/kirpicheff/anyui`).\n")
	out.WriteString("- Проверить установленную версию модуля: `go list -m github.com/kirpicheff/anyui`.\n")
	out.WriteString("- При расхождении этого документа с установленным пакетом источник истины — `go doc github.com/kirpicheff/anyui.<Symbol>`.\n")
	out.WriteString("- Подпакеты являются внутренними; прикладной код импортирует **только** корневой пакет `github.com/kirpicheff/anyui`.\n\n")

	out.WriteString("## Правила работы с API\n\n")
	out.WriteString("- **Immediate Mode:** состояние хранится вне `drawFn` (в структурах/переменных приложения), `drawFn(ui *anyui.UI)` перерисовывает весь интерфейс на каждом кадре.\n")
	out.WriteString("- **Единицы измерения (`Dp` vs `Px`):** весь API раскладки, опций, оверлеев и ввода Headless (`LayoutRect`, `Tooltip`, `PushOverlay`, `Popover`, `ContextMenu`, `RegisterHoverArea`, `InjectMouse*`, `CurrentArea()`, `MousePosDp()`) работает строго в логических `Dp`. Физические `Px` используются только в колбэках рисования (`CanvasView`, `InteractiveCanvas`, `Table.drawCell`, `ListView.drawItem`), прямых методах (`DrawText`, `DrawSVG`, `CurrentAreaPx()`, `MousePosPx()`, `LastDirtyRectsPx()`, `TitleBarHitTest`, `InjectMouse*Px`) и на границе ОС (`Config`, `Window`).\n")
	out.WriteString("- **Единое задание размеров через опции:** `ScrollView`, `CanvasView`, `InteractiveCanvas`, `Image`, `ListView`, `Table`, `TextArea`, `LogView` по умолчанию занимают доступное пространство (`Auto`); явные размеры задаются опциями `Width(...)`, `Height(...)`, `FillWidth()`, `FillHeight()`. `Card` по умолчанию подбирает высоту по содержимому (`AutoHeight`); для растягивания передайте `FillHeight()`.\n")
	out.WriteString("- **Идентификаторы, ключи и `IDScope`:** `ui.IDScope(key, fn)` (или `PushID`/`PopID`) изолирует `id` виджетов внутри циклов и переиспользуемых компонентов. В `Select`, `MultiSelect`, `TabBar`, `TreeView`, `TreeViewEx` и `Breadcrumbs` ключи элементов (`Key` / `ID`) локальны для конкретного виджета и должны быть уникальны в его пределах — при дублировании ключей в одном списке/дереве выделение и клик совпадут для всех дубликатов.\n")
	out.WriteString("- **Выпадающие списки и вкладки:** `ui.Select` с `anyui.Searchable()` заменяет комбобокс; `ui.MultiSelect` работает с `map[string]bool`; `ui.TabBar` и `anyui.TabItems(...)` работают на строковых ключах.\n")
	out.WriteString("- **Раскладка и оверлеи:** для типового экрана используйте `ui.Page` или `DockTop` / `DockBottom` / `ColumnFlex`. Для диалогов всегда используйте `ui.Modal` — `PushOverlay(bounds, fn)` предназначен только для небольших всплывающих контролов.\n")
	out.WriteString("- **Drag & Drop (Reorder):** `anyui.Reorderable()` + `anyui.OnReorder(func(ev anyui.DragEvent))` работают в `Table`, `TableEx`, `ListView`, `ListViewEx` (индексы `FromIndex`/`ToIndex`) и `TreeView`, `TreeViewEx` (ключи `FromKey`/`ToKey`; с `anyui.AllowReparent()` доступны смена родителя и `DropInside`).\n\n")

	out.WriteString("## Семантика возвращаемых значений\n\n")
	out.WriteString("- `Button`, `Hyperlink` → `bool`: `true` **только в кадре нажатия**.\n")
	out.WriteString("- `Toggle`, `Checkbox`, `Radio`, `TextInput`, `TextArea`, `MaskedInput`, `Select`, `MultiSelect`, `TagInput`, `Slider`, `RangeSlider`, `NumberInput`, `ColorPicker`, `DatePicker`, `TimePicker`, `DateTimePicker`, `ListView`, `ListViewEx`, `TreeView`, `TreeViewEx`, `Pagination`, `Collapsible`, `Accordion` → `bool`: `true` **только в кадре изменения значения** (не означает «виджет сейчас активен/включён»; текущее состояние хранится в переданном указателе/карте).\n")
	out.WriteString("- `Breadcrumbs(...) string`: возвращает `ID` кликнутого сегмента (или `Label`, если `ID == \"\"`), либо `\"\"` если клика в этом кадре не было.\n")
	out.WriteString("- `TabBar(...) TabBarEvent`: `SelectedID` (смена вкладки), `ClosedID` (клик по крестику/СКМ), `Added` (клик по `+`), `MovedFrom`/`MovedTo` (перетаскивание, `-1` если не было), `DetachedID` (отрыв вкладки).\n")
	out.WriteString("- `Table` / `TableEx` → `TableEvent`: `RowClicked`, `RowDoubleClicked`, `RowMiddleClicked` (`-1` если события не было), `SortChanged`, `ColumnResized`, `Drag *DragEvent`.\n\n")

	out.WriteString("## Асинхронная загрузка данных (`Remember`)\n\n")
	out.WriteString("`anyui.Remember[T](ui, key, loader)` запускает `loader` **асинхронно в отдельной горутине** при первом вызове для `key` (пока идёт загрузка, `loading == true`, повторных запусков не происходит). По завершении результат и ошибка кешируются в окне по `key`, и окно автоматически перерисовывается. Для повторного запроса сбросьте кеш через `ui.InvalidateRemember(key)` или `ui.ClearRemember()`.\n\n")
	out.WriteString("```go\n")
	out.WriteString("users, loading, err := anyui.Remember(ui, \"users_list\", func() ([]string, error) {\n")
	out.WriteString("\treturn api.FetchUsers()\n")
	out.WriteString("})\n")
	out.WriteString("if loading {\n")
	out.WriteString("\tui.Spinner(\"load_users\", 20)\n")
	out.WriteString("} else if err != nil {\n")
	out.WriteString("\tui.Label(\"Ошибка: \"+err.Error(), anyui.TextColor(ui.Theme().Danger))\n")
	out.WriteString("\tif ui.Button(\"retry\", \"Повторить\") { ui.InvalidateRemember(\"users_list\") }\n")
	out.WriteString("}\n")
	out.WriteString("```\n\n")

	out.WriteString("## Матрица применимости опций (`Option`)\n\n")
	out.WriteString("Неприменимые для виджета опции молча игнорируются. Основные комбинации:\n\n")
	out.WriteString("| Виджет | Поддерживаемые опции |\n")
	out.WriteString("|---|---|\n")
	out.WriteString("| `Label`, `SelectableLabel`, `StatusDot` | `FontSize`, `Bold`, `Mono`, `TextColor`, `TextAlignLeft/Center/Right`, `LetterSpacing`, `TextShadow`, `WordWrap`, `Ellipsis`, `Width`, `MaxWidth`, `Disabled` |\n")
	out.WriteString("| `Button`, `Hyperlink` | `Primary`, `Secondary`, `Danger`, `Ghost`, `Icon`, `IconRight`, `Width`, `Height`, `FillWidth`, `Flex`, `Padding`, `Radius`, `FontSize`, `Bold`, `Bg`, `HoverBg`, `TextColor`, `HoverTextColor`, `BorderColor`, `HoverBorderColor`, `Disabled` |\n")
	out.WriteString("| `TextInput`, `MaskedInput` | `Width`, `Height`, `FillWidth`, `Flex`, `FontSize`, `Mono`, `Password`, `PasswordToggle`, `Clearable`, `ReadOnly`, `Disabled`, `InputFilter`, `Error`, `Icon`, `IconRight`, `FocusBorderColor` |\n")
	out.WriteString("| `TextArea` | `Width`, `Height`, `FillWidth`, `FillHeight`, `Flex`, `FontSize`, `Mono`, `LineNumbers`, `ReadOnly`, `Disabled`, `Error`, `FocusBorderColor` |\n")
	out.WriteString("| `Card` | `Width`, `Height`, `FillWidth`, `FillHeight`, `AutoHeight`, `Flex`, `Padding`, `Radius`, `Bg`, `BorderColor` |\n")
	out.WriteString("| `ScrollView`, `CanvasView`, `InteractiveCanvas`, `Image` | `Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Flex` (+ `ScrollToID` для `ScrollView`) |\n")
	out.WriteString("| `Select`, `MultiSelect` | `Width`, `FillWidth`, `Flex`, `Searchable`, `Disabled`, `Error` |\n")
	out.WriteString("| `ListView`, `ListViewEx` | `Width`, `Height`, `FillWidth`, `FillHeight`, `Flex`, `ScrollToIndex`, `Reorderable`, `OnReorder` |\n")
	out.WriteString("| `Table`, `TableEx` | `Width`, `Height`, `FillWidth`, `FillHeight`, `Flex`, `ScrollToIndex`, `Reorderable`, `OnReorder` |\n")
	out.WriteString("| `TreeView`, `TreeViewEx` | `Width`, `Height`, `FillWidth`, `FillHeight`, `Flex`, `Reorderable`, `AllowReparent`, `OnReorder` |\n")
	out.WriteString("| `Modal`, `Popover`, `FloatingPanel`, `Drawer` | `Width`, `Height`, `Bg`, `BorderColor`, `Radius`, `Padding` |\n")
	out.WriteString("| `ContextMenu`, `Tooltip` | `Width` (`ContextMenu`), `Bg`, `BorderColor`, `Radius`, `FontSize`, `TextColor`, `Padding` (`Tooltip`) |\n\n")

	out.WriteString("## Частые ошибки\n\n")
	out.WriteString("1. **Состояние внутри `drawFn`:** объявление `name := \"\"` внутри `drawFn` сбрасывает значение на каждом кадре. Выносите состояние в структуру приложения выше `app.Run`.\n")
	out.WriteString("2. **Значение вместо указателя:** виджеты ввода и выбора (`TextInput`, `Toggle`, `Checkbox`, `Select`, `TabBar`, `Slider`, `ListView`) мутируют состояние по указателю (`&val`), а возвращают лишь факт изменения в кадре.\n")
	out.WriteString("3. **`ui.Shortcut` внутри условных веток:** метод регистрирует глобальную горячую клавишу окна — не перерегистрируйте его динамически в циклах.\n")
	out.WriteString("4. **Самодельные модалки через `PushOverlay(fullRect)`:** блокирует ввод всего окна на следующих кадрах. Всегда используйте `ui.Modal`.\n")
	out.WriteString("5. **Смешивание `Px` и `Dp`:** передача `CurrentAreaPx()` в `PushOverlay`, `LayoutRect`, `Tooltip`, `Popover`, `ContextMenu` или `RegisterHoverArea` ломает координаты при DPI > 100%. Для них всегда берите `ui.CurrentArea()` (`Dp`).\n\n")
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

func writeShortLLMs(out *bytes.Buffer, version string, idx *apiIndex) {
	writeCommonHeader(out, version, false)
	writeQuickStart(out)

	// Ключевые структуры для офлайн-работы
	keyStructs := []string{
		"Config", "PageConfig", "FlexItem", "TableColumn", "TableEvent",
		"ListViewExConfig", "TreeViewExConfig", "TableExConfig",
		"TabItem", "TabBarEvent", "SelectItem", "TreeNode",
		"ToolbarItem", "ContextMenuItem", "MenuBarItem", "MenuBarEntry",
		"BreadcrumbItem", "DragEvent",
	}
	out.WriteString("## Ключевые структуры (офлайн-минимум)\n\n```go\n")
	for _, name := range keyStructs {
		if def, ok := idx.formatTypeDef(name); ok {
			out.WriteString(def)
			out.WriteString("\n\n")
		}
	}
	out.WriteString("```\n\n")

	// Полные сигнатуры основных методов UI (без многоточий)
	out.WriteString("## Основные методы `UI` (полные сигнатуры)\n\n```go\n")
	for _, sig := range idx.methods["UI"] {
		out.WriteString(sig)
		out.WriteString("\n")
	}
	out.WriteString("```\n")
}

func writeFullLLMs(out *bytes.Buffer, version string, idx *apiIndex) {
	writeCommonHeader(out, version, true)
	writeQuickStart(out)

	out.WriteString("## Константы и переменные пакета `anyui`\n\n```go\n")
	out.WriteString("const Auto Dp = -1\n")
	for _, grp := range idx.constGroups {
		if len(grp) == 1 && grp[0] == "Auto" {
			continue
		}
		out.WriteString("const ( ")
		out.WriteString(strings.Join(grp, ", "))
		out.WriteString(" )\n")
	}
	for _, v := range idx.vars {
		out.WriteString("var " + v + "\n")
	}
	out.WriteString("```\n\n")

	out.WriteString("## Функции пакета `anyui`\n\n```go\n")
	for _, sig := range idx.otherFuncs {
		out.WriteString(sig + "\n")
	}
	out.WriteString("```\n\n")

	out.WriteString("## Опции стилизации (`Option`)\n\n```go\n")
	for _, sig := range idx.optionFuncs {
		out.WriteString(sig + "\n")
	}
	out.WriteString("```\n\n")

	// Все структуры и интерфейсы
	structOrder := []string{
		"Config", "PageConfig", "FlexItem", "TitleBarConfig",
		"TableColumn", "TableSort", "TableEvent",
		"ListViewExConfig", "TreeViewExConfig", "TableExConfig",
		"TabItem", "TabBarEvent", "SelectItem", "TreeNode",
		"ToolbarItem", "ContextMenuItem", "MenuBarItem", "MenuBarEntry",
		"StatusBarSection", "BreadcrumbItem", "AccordionItem",
		"LogEntry", "ChartSeries", "CanvasCallbacks", "KeyEvent", "DragEvent",
		"TrayConfig", "MenuItem", "Notification",
		"FileDialogOptions", "FolderDialogOptions", "FileFilter",
		"SingleInstanceConfig", "WindowState",
		"Color", "Rect", "CornerRadii", "GradientStop", "LinearGradient", "RadialGradient", "BoxShadow", "PointF",
		"Theme", "Strings", "TestHelper",
	}
	out.WriteString("## Структуры и типы данных\n\n```go\n")
	for _, name := range structOrder {
		if def, ok := idx.formatTypeDef(name); ok {
			out.WriteString(def)
			out.WriteString("\n\n")
		}
	}
	out.WriteString("```\n\n")

	for _, recv := range []string{"App", "Window", "Headless", "UI"} {
		out.WriteString(fmt.Sprintf("## Методы `%s`\n\n```go\n", recv))
		for _, sig := range idx.methods[recv] {
			out.WriteString(sig + "\n")
		}
		out.WriteString("```\n\n")
	}
}
