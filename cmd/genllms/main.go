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
	"regexp"
	"sort"
	"strings"
)

type aliasTarget struct {
	pkg  string
	name string
}

type constEntry struct {
	name      string
	typeName  string
	valueExpr string
}

type apiIndex struct {
	fset         *token.FileSet
	rootFiles    []*ast.File
	subPkgTypes  map[string]map[string]*ast.TypeSpec // pkg -> typeName -> TypeSpec
	subPkgConsts map[string]map[string]string        // pkg -> constName -> literal value
	aliases      map[string]aliasTarget              // publicName -> subpkg target
	localTypes   map[string]*ast.TypeSpec
	constGroups  [][]constEntry
	vars         []string
	optionFuncs  []string
	optionNames  map[string]bool
	otherFuncs   []string
	methods      map[string][]string // receiverType -> []fullSignature
	methodNames  map[string]map[string]bool
	allSymbols   map[string]bool // все экспортированные символы пакета anyui (типы, функции, константы, переменные)
}

func main() {
	versionFlag := flag.String("version", "", "переопределить версию в шапке (по умолчанию берётся из git describe)")
	checkFlag := flag.Bool("check", false, "проверить актуальность llms.txt и llms-full.txt без перезаписи (exit 1 при расхождении)")
	flag.Parse()

	version := strings.TrimSpace(*versionFlag)
	if version == "" {
		version = detectVersion()
	}

	shortBytes, fullBytes, err := generateAll(".", version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ошибка генерации: %v\n", err)
		os.Exit(1)
	}

	if *checkFlag {
		if err := checkFile("llms.txt", shortBytes); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		if err := checkFile("llms-full.txt", fullBytes); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		fmt.Printf("llms.txt и llms-full.txt актуальны (версия: %s)\n", version)
		return
	}

	if err := os.WriteFile("llms.txt", shortBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка записи llms.txt: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile("llms-full.txt", fullBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка записи llms-full.txt: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("llms.txt и llms-full.txt обновлены (версия: %s)\n", version)
}

func checkFile(path string, expected []byte) error {
	actual, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("файл %s отсутствует или не читается: %w (запустите go run ./cmd/genllms)", path, err)
	}
	normActual := bytes.ReplaceAll(actual, []byte("\r\n"), []byte("\n"))
	normExpected := bytes.ReplaceAll(expected, []byte("\r\n"), []byte("\n"))
	if !bytes.Equal(normActual, normExpected) {
		return fmt.Errorf("файл %s устарел (запустите go run ./cmd/genllms)", path)
	}
	return nil
}

func generateAll(rootDir, version string) ([]byte, []byte, error) {
	idx, err := parseAPI(rootDir)
	if err != nil {
		return nil, nil, err
	}

	var shortOut bytes.Buffer
	writeShortLLMs(&shortOut, version, idx)

	var fullOut bytes.Buffer
	writeFullLLMs(&fullOut, version, idx)

	if err := lintOutputs(idx, shortOut.String(), fullOut.String()); err != nil {
		return nil, nil, err
	}

	return shortOut.Bytes(), fullOut.Bytes(), nil
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
		fset:         token.NewFileSet(),
		subPkgTypes:  make(map[string]map[string]*ast.TypeSpec),
		subPkgConsts: make(map[string]map[string]string),
		aliases:      make(map[string]aliasTarget),
		localTypes:   make(map[string]*ast.TypeSpec),
		optionNames:  make(map[string]bool),
		methods:      make(map[string][]string),
		methodNames:  make(map[string]map[string]bool),
		allSymbols:   make(map[string]bool),
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
		idx.subPkgConsts[sub] = make(map[string]string)
		for _, f := range subFiles {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				switch gd.Tok {
				case token.TYPE:
					for _, sp := range gd.Specs {
						ts, ok := sp.(*ast.TypeSpec)
						if ok && ast.IsExported(ts.Name.Name) {
							idx.subPkgTypes[sub][ts.Name.Name] = ts
						}
					}
				case token.CONST:
					iotaOffset := 0
					hasIota := false
					for i, sp := range gd.Specs {
						vs, ok := sp.(*ast.ValueSpec)
						if !ok {
							continue
						}
						if len(vs.Values) > 0 {
							valStr := exprToString(idx.fset, vs.Values[0])
							if valStr == "iota" {
								hasIota = true
								iotaOffset = 0
							} else if valStr == "iota + 1" || valStr == "iota+1" {
								hasIota = true
								iotaOffset = 1
							} else {
								hasIota = false
							}
						}
						for j, name := range vs.Names {
							if !ast.IsExported(name.Name) {
								continue
							}
							if hasIota {
								idx.subPkgConsts[sub][name.Name] = fmt.Sprintf("%d", i+iotaOffset)
							} else if j < len(vs.Values) {
								idx.subPkgConsts[sub][name.Name] = exprToString(idx.fset, vs.Values[j])
							}
						}
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
					var grp []constEntry
					for _, sp := range d.Specs {
						vs, ok := sp.(*ast.ValueSpec)
						if !ok {
							continue
						}
						tName := ""
						if vs.Type != nil {
							tName = sanitizeInternalTypes(exprToString(idx.fset, vs.Type))
						}
						for j, name := range vs.Names {
							if !ast.IsExported(name.Name) {
								continue
							}
							idx.allSymbols[name.Name] = true
							val := ""
							if j < len(vs.Values) {
								if sel, ok := vs.Values[j].(*ast.SelectorExpr); ok {
									if pkgID, ok := sel.X.(*ast.Ident); ok {
										if v, ok := idx.subPkgConsts[pkgID.Name][sel.Sel.Name]; ok {
											val = v
										}
									}
								} else {
									val = exprToString(idx.fset, vs.Values[j])
								}
							}
							grp = append(grp, constEntry{
								name:      name.Name,
								typeName:  tName,
								valueExpr: val,
							})
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
								idx.allSymbols[name.Name] = true
							}
						}
					}
				case token.TYPE:
					for _, sp := range d.Specs {
						ts, ok := sp.(*ast.TypeSpec)
						if !ok || !ast.IsExported(ts.Name.Name) {
							continue
						}
						idx.allSymbols[ts.Name.Name] = true
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
					idx.allSymbols[d.Name.Name] = true
					if returnsOption(d) {
						idx.optionFuncs = append(idx.optionFuncs, sig)
						idx.optionNames[d.Name.Name] = true
					} else {
						idx.otherFuncs = append(idx.otherFuncs, sig)
					}
				} else {
					recvName := receiverTypeName(d.Recv.List[0].Type)
					if ast.IsExported(recvName) {
						idx.methods[recvName] = append(idx.methods[recvName], sig)
						if idx.methodNames[recvName] == nil {
							idx.methodNames[recvName] = make(map[string]bool)
						}
						idx.methodNames[recvName][d.Name.Name] = true
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

func exprToString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	_ = format.Node(&buf, fset, expr)
	return strings.TrimSpace(buf.String())
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
		out.WriteString("> Полный офлайн-справочник со всеми сигнатурами, опциями, константами и структурами: `llms-full.txt`.\n\n")
	}
	out.WriteString("## Версия и источник истины\n\n")
	out.WriteString("- Импорт: `import \"github.com/kirpicheff/anyui\"` (`go get github.com/kirpicheff/anyui`).\n")
	out.WriteString("- Проверить установленную версию модуля: `go list -m github.com/kirpicheff/anyui`.\n")
	out.WriteString("- При расхождении этого документа с установленным пакетом источник истины — `go doc github.com/kirpicheff/anyui.UI.<Method>` (для методов `UI`/`App`/`Window`/`Headless`) и `go doc github.com/kirpicheff/anyui.<Symbol>`.\n")
	out.WriteString("- Подпакеты являются внутренними; прикладной код импортирует **только** корневой пакет `github.com/kirpicheff/anyui`.\n\n")

	out.WriteString("## Правила работы с API\n\n")
	out.WriteString("- **Immediate Mode и многопоточность:** состояние хранится вне `drawFn`; вызовы методов `UI` и чтение/изменение состояния интерфейса выполняются **только в главном потоке** внутри `drawFn`. Для обновления состояния из фоновых горутин всегда используйте `App.Dispatch` (`app.Dispatch(func() { ... })`) или `Remember`.\n")
	out.WriteString("- **Единицы измерения (`Dp` vs `Px`):** весь API раскладки, опций, оверлеев и ввода Headless (`LayoutRect`, `Tooltip`, `PushOverlay`, `Popover`, `ContextMenu`, `RegisterHoverArea`, `InjectMouseMove`/`InjectMouseClick`, `CurrentArea`, `MousePosDp`) работает строго в логических `Dp`. Физические `Px` используются только в колбэках рисования (`CanvasView`, `InteractiveCanvas`, `Table`, `ListView`), прямых методах (`DrawText`, `DrawSVG`, `CurrentAreaPx`, `MousePosPx`, `LastDirtyRectsPx`, `TitleBarHitTest`, `InjectMouseMovePx`/`InjectMouseClickPx`) и на границе ОС (`Config`, `Window`).\n")
	out.WriteString("- **`Rect` и `PointF` не типизированы по единицам:** `Rect` содержит `int`-поля, `PointF` — `float32`. В `Dp`-API (`CurrentArea`, `LayoutRect`, `PushOverlay`, `Tooltip`, `Popover`, `ContextMenu`, `RegisterHoverArea`) они трактуются как `Dp`, а в колбэках рисования и `CurrentAreaPx`/`LastDirtyRectsPx` — как `Px`.\n")
	out.WriteString("- **Единое задание размеров через опции:** `ScrollView`, `CanvasView`, `InteractiveCanvas`, `Image`, `ListView`, `Table`, `TextArea`, `LogView` по умолчанию занимают доступное пространство (`Auto`); явные размеры задаются опциями `Width`, `Height`, `FillWidth`, `FillHeight`. `Card` по умолчанию подбирает высоту по содержимому (`AutoHeight`); для растягивания передайте `FillHeight`.\n")
	out.WriteString("- **Колонки таблиц:** в `Table` и `TableEx` колонки с `WidthDp <= 0` автоматически делят оставшуюся ширину таблицы.\n")
	out.WriteString("- **Декларативная навигация:** для программной прокрутки используйте `ScrollToIndex` в `ListView` и `Table`, а для `ScrollView` — `ScrollToID` в паре с маркером `Anchor` (или `id` дочернего виджета).\n")
	out.WriteString("- **Иконка окна:** подхватывается автоматически из `.syso`-ресурсов, либо задаётся через `Config.IconRGBA` или `Window.SetIcon`.\n")
	out.WriteString("- **Идентификаторы, ключи и `IDScope`:** `IDScope` (или `PushID`/`PopID`) изолирует `id` виджетов внутри циклов и переиспользуемых компонентов. В `Select`, `MultiSelect`, `TabBar`, `TreeView`, `TreeViewEx` и `Breadcrumbs` ключи элементов (`Key` / `ID`) локальны для конкретного виджета и должны быть уникальны в его пределах — при дублировании ключей выделение и клик совпадут для всех дубликатов. В `Breadcrumbs`, если у сегмента `ID == \"\"`, возвращается его `Label`.\n")
	out.WriteString("- **Выпадающие списки и вкладки:** `Select` с опцией `Searchable` заменяет комбобокс; `MultiSelect` работает с `map[string]bool`; `TabBar` и `TabItems` работают на строковых ключах.\n")
	out.WriteString("- **Раскладка и оверлеи:** для типового экрана используйте `Page` или `DockTop` / `DockBottom` / `ColumnFlex`. Для диалогов всегда используйте `Modal` — `PushOverlay` предназначен только для небольших всплывающих контролов.\n")
	out.WriteString("- **Drag & Drop (Reorder):** `Reorderable` + `OnReorder` работают в `Table`, `TableEx`, `ListView`, `ListViewEx` (индексы `FromIndex`/`ToIndex`) и `TreeView`, `TreeViewEx` (ключи `FromKey`/`ToKey`; с `AllowReparent` доступны смена родителя и `DropInside`).\n\n")

	out.WriteString("## Семантика возвращаемых значений\n\n")
	out.WriteString("- `Button`, `Hyperlink` → `bool`: `true` **только в кадре нажатия**.\n")
	out.WriteString("- `Toggle`, `Checkbox`, `RadioButton`, `RadioGroup`, `TextInput`, `TextArea`, `MaskedInput`, `Select`, `MultiSelect`, `TagInput`, `Slider`, `RangeSlider`, `NumberInput`, `ColorPicker`, `DatePicker`, `TimePicker`, `DateTimePicker`, `ListView`, `ListViewEx`, `TreeView`, `TreeViewEx`, `Pagination`, `Collapsible`, `Accordion` → `bool`: `true` **только в кадре изменения значения** (не означает «виджет сейчас активен/включён»; текущее состояние хранится в переданном указателе/карте).\n")
	out.WriteString("- `Breadcrumbs` → `string`: возвращает `ID` кликнутого сегмента (или `Label`, если `ID == \"\"`), либо `\"\"` если клика в этом кадре не было.\n")
	out.WriteString("- `TabBar` → `TabBarEvent`: `SelectedID` (смена вкладки), `ClosedID` (клик по крестику/СКМ), `Added` (клик по `+`), `MovedFrom`/`MovedTo` (перетаскивание, `-1` если не было), `DetachedID` (отрыв вкладки).\n")
	out.WriteString("- `Table`, `TableEx` → `TableEvent`: `RowClicked`, `RowDoubleClicked`, `RowMiddleClicked` (`-1` если события не было), `SortChanged`, `ColumnResized`, `Drag`.\n\n")

	out.WriteString("## Асинхронная загрузка данных (`Remember`)\n\n")
	out.WriteString("`Remember` запускает `loader` **асинхронно в отдельной горутине** при первом вызове для `key` (пока идёт загрузка, `loading == true`, повторных запусков не происходит). По завершении результат и ошибка кешируются по `key`, и окно автоматически перерисовывается.\n")
	out.WriteString("- **Внутри `loader` нельзя вызывать методы `UI`** (он выполняется в фоновой горутине; при необходимости побочных обновлений используйте `App.Dispatch`).\n")
	out.WriteString("- **Область кеша — конкретное окно (`UI`):** одинаковый `key` в двух разных окнах хранит два независимых кеша. Сброс: `InvalidateRemember` (по ключу) или `ClearRemember` (всё окно).\n\n")
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

	writeBaseTypes(out)

	out.WriteString("## Матрица применимости опций (`Option`)\n\n")
	out.WriteString("Неприменимые для конкретного виджета опции игнорируются без ошибки. Проверенные по реализации комбинации:\n\n")
	out.WriteString("| Виджет | Поддерживаемые опции |\n")
	out.WriteString("|---|---|\n")
	for _, row := range optionMatrixRows {
		out.WriteString("| ")
		out.WriteString(row.widgets)
		out.WriteString(" | ")
		out.WriteString(row.options)
		out.WriteString(" |\n")
	}
	out.WriteString("\n")

	out.WriteString("## Частые ошибки\n\n")
	out.WriteString("1. **Состояние внутри `drawFn`:** объявление `name := \"\"` внутри `drawFn` сбрасывает значение на каждом кадре. Выносите состояние в структуру приложения выше `App.Run`.\n")
	out.WriteString("2. **Обновление UI из фоновой горутины напрямую:** вызывает гонки данных. Всегда оборачивайте мутации состояния в `App.Dispatch`.\n")
	out.WriteString("3. **Значение вместо указателя:** виджеты ввода и выбора (`TextInput`, `Toggle`, `Checkbox`, `Select`, `TabBar`, `Slider`, `ListView`) мутируют состояние по указателю, а возвращают лишь факт изменения в кадре.\n")
	out.WriteString("4. **`Shortcut` внутри условных веток:** метод регистрирует глобальную горячую клавишу окна — не перерегистрируйте его динамически в циклах.\n")
	out.WriteString("5. **Самодельные модалки через `PushOverlay` на весь экран:** блокирует ввод всего окна на следующих кадрах. Всегда используйте `Modal`.\n")
	out.WriteString("6. **Смешивание `Px` и `Dp`:** передача `CurrentAreaPx` в `PushOverlay`, `LayoutRect`, `Tooltip`, `Popover`, `ContextMenu` или `RegisterHoverArea` ломает координаты при DPI > 100%. Для них всегда берите `CurrentArea` (`Dp`).\n\n")
}

type optionMatrixRow struct {
	widgets string
	options string
}

// optionMatrixRows проверена по реальным обращениям к WidgetStyle в ui/*.go.
var optionMatrixRows = []optionMatrixRow{
	{
		"`Label`, `SelectableLabel`, `StatusDot`",
		"`FontSize`, `Bold`, `Mono`, `TextColor`, `TextAlignLeft`, `TextAlignCenter`, `TextAlignRight`, `LetterSpacing`, `TextShadow`, `WordWrap`, `Ellipsis`, `Width`, `MaxWidth`, `Disabled`",
	},
	{
		"`Button`, `Hyperlink`",
		"`Primary`, `Secondary`, `Danger`, `Ghost`, `Icon`, `IconRight`, `Width`, `Height`, `FillWidth`, `Flex`, `Padding`, `Radius`, `FontSize`, `Bold`, `Bg`, `HoverBg`, `TextColor`, `HoverTextColor`, `BorderColor`, `HoverBorderColor`, `Disabled`",
	},
	{
		"`Checkbox`, `Toggle`, `RadioButton`",
		"`Width`, `FillWidth`, `Flex`, `Disabled`, `Indeterminate` (`Checkbox`)",
	},
	{
		"`TextInput`, `MaskedInput`",
		"`Width`, `Height`, `FillWidth`, `Flex`, `FontSize`, `Mono`, `Password`, `PasswordToggle`, `Clearable`, `ReadOnly`, `Disabled`, `InputFilter`, `Error`, `Icon`, `IconRight`, `FocusBorderColor`",
	},
	{
		"`TextArea`",
		"`Width`, `Height`, `FillWidth`, `FillHeight`, `Flex`, `Padding`, `FontSize`, `Mono`, `LineNumbers`, `ReadOnly`, `Disabled`, `InputFilter`, `Error`, `Bg`, `BorderColor`, `Radius`, `TextColor`",
	},
	{
		"`Card`",
		"`Width`, `Height`, `FillWidth`, `FillHeight`, `AutoHeight`, `Flex`, `Padding`, `Radius`, `Bg`, `BorderColor`",
	},
	{
		"`ScrollView`, `CanvasView`, `InteractiveCanvas`, `Image`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Flex`, `Padding` (`ScrollView`), `ScrollToID` (`ScrollView`)",
	},
	{
		"`Select`, `MultiSelect`",
		"`Width`, `Height`, `FillWidth`, `Flex`, `Searchable` (`Select`), `Bg` (`Select`), `Radius` (`Select`), `Disabled`",
	},
	{
		"`ListView`, `ListViewEx`",
		"`Width`, `Height`, `FillWidth`, `FillHeight`, `Flex`, `Bg` (`ListView`), `Radius` (`ListView`), `Disabled` (`ListView`), `ScrollToIndex` (`ListView`), `Reorderable`, `OnReorder`",
	},
	{
		"`Table`, `TableEx`",
		"`Width`, `Height`, `FillWidth`, `FillHeight`, `Flex`, `Radius` (`Table`), `Disabled` (`Table`), `ScrollToIndex` (`Table`), `Reorderable`, `OnReorder`",
	},
	{
		"`TreeView`, `TreeViewEx`",
		"`Width`, `Height`, `FillWidth`, `FillHeight`, `Flex`, `Radius` (`TreeView`), `Disabled` (`TreeView`), `Reorderable`, `AllowReparent`, `OnReorder`",
	},
	{
		"`Modal`, `Popover`, `FloatingPanel`, `Drawer`",
		"`Width`, `Height`",
	},
	{
		"`ContextMenu`, `Tooltip`",
		"`Width` (`ContextMenu`), `Bg`, `BorderColor`, `Radius`, `FontSize`, `TextColor`, `Padding` (`Tooltip`)",
	},
}

func writeBaseTypes(out *bytes.Buffer) {
	out.WriteString("## Базовые типы\n\n```go\n")
	out.WriteString("type Dp int               // Логические DPI-независимые пиксели (.Int() int)\n")
	out.WriteString("type Px int               // Физические пиксели буфера отрисовки (.Int() int)\n")
	out.WriteString("const Auto Dp = -1        // Sentinel-значение: занять всё доступное пространство контейнера\n")
	out.WriteString("type Key = uint32         // Виртуальный код клавиши (VK-коды, передаётся в Shortcut / InjectKey / KeyEvent.KeyCode)\n")
	out.WriteString("type Modifiers struct { Ctrl, Shift, Alt bool }\n")
	out.WriteString("type MouseButton uint8    // MouseLeft = 1, MouseRight = 2, MouseMiddle = 3\n")
	out.WriteString("type CursorType uint8     // Системный курсор (CursorArrow, CursorHand, CursorIBeam, CursorResizeH/V/NWSE/NESW, CursorNotAllowed, CursorCrosshair, CursorMove)\n")
	out.WriteString("type Orientation uint8    // Horizontal, Vertical\n")
	out.WriteString("type DropZone uint8       // DropAbove, DropBelow, DropInside (для DragEvent)\n")
	out.WriteString("type Align uint8          // AlignStart, AlignCenter, AlignEnd, AlignStretch\n")
	out.WriteString("type Justify uint8        // JustifyStart, JustifyCenter, JustifyEnd, JustifySpaceBetween, JustifySpaceAround\n")
	out.WriteString("type DrawerSide uint8     // DrawerLeft, DrawerRight, DrawerTop, DrawerBottom\n")
	out.WriteString("type ToastKind uint8      // ToastInfo, ToastSuccess, ToastWarning, ToastError\n")
	out.WriteString("type ToastPosition uint8  // ToastBottomRight/Left/Center, ToastTopRight/Left/Center\n")
	out.WriteString("type LogLevel uint8       // LogInfo, LogWarn, LogError, LogDebug\n")
	out.WriteString("type ImageScaleMode uint8 // ScaleFit, ScaleFill, ScaleOneToOne\n")
	out.WriteString("type FontStyle uint8      // FontRegular, FontBold, FontMono\n")
	out.WriteString("type TextAlign uint8      // AlignTextLeft, AlignTextCenter, AlignTextRight\n")
	out.WriteString("type HitTestResult int    // HitClient, HitCaption, HitLeft/Right/Top/Bottom/TopLeft/TopRight/BottomLeft/BottomRight\n")
	out.WriteString("```\n\n")
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

	out.WriteString("## Карта API (полные сигнатуры и структуры — в `llms-full.txt`)\n\n")
	out.WriteString("- **Приложение и окна:** `NewApp`, `App.Run`, `App.Dispatch`, `App.OpenWindow`, `App.MainWindow`, `App.Quit`, `App.SetTrayMenu`, `App.SetTrayIcon`, `App.SetTrayTooltip`, `App.ShowNotification`, `Window.SetTitle`, `Window.SetSize`, `Window.SetPosition`, `Window.SetIcon`, `Window.SetTheme`, `Window.Invalidate`, `Window.WakeUp`, `Window.Close`.\n")
	out.WriteString("- **Раскладка и контейнеры (`UI`):** `Column`, `ColumnCentered`, `ColumnWith`, `ColumnFlex`, `Row`, `RowCentered`, `RowWith`, `RowFlex`, `Grid`, `Flow`, `FormLayout`, `FormRow`, `Padding`, `PaddingXY`, `Spacer`, `FlexSpacer`, `Expand`, `DockTop`, `DockBottom`, `Page`, `Card`, `ScrollView`, `Anchor`, `SplitPane`, `Collapsible`, `Accordion`, `LayoutRect`, `IDScope`, `PushID`, `PopID`.\n")
	out.WriteString("- **Ввод и формы (`UI`):** `Button`, `Hyperlink`, `Toggle`, `Checkbox`, `RadioButton`, `RadioGroup`, `TextInput`, `TextArea`, `MaskedInput`, `NumberInput`, `Slider`, `RangeSlider`, `Select`, `MultiSelect`, `TagInput`, `ColorPicker`, `DatePicker`, `TimePicker`, `DateTimePicker`.\n")
	out.WriteString("- **Списки, деревья, таблицы и графики (`UI`):** `ListView`, `ListViewEx`, `TreeView`, `TreeViewEx`, `Table`, `TableEx`, `LogView`, `VirtualTextView`, `Sparkline`, `LineChart`, `BarChart`.\n")
	out.WriteString("- **Текст, индикаторы и навигация (`UI`):** `Label`, `SelectableLabel`, `StatusDot`, `Badge`, `Separator`, `ProgressBar`, `Spinner`, `TabBar`, `MenuBar`, `Toolbar`, `StatusBar`, `Breadcrumbs`, `Pagination`, `TitleBar`, `TitleBarHitTest`.\n")
	out.WriteString("- **Оверлеи и уведомления (`UI`):** `Modal`, `Popover`, `FloatingPanel`, `Drawer`, `ContextMenu`, `Tooltip`, `PushOverlay`, `ShowToast`, `SetToastPosition`, `SetToastOffset`.\n")
	out.WriteString("- **Графика и холст (`UI`):** `Image`, `CanvasView`, `InteractiveCanvas`, `Canvas`, `DrawText`, `DrawSVG`, `StaticLayer`, `InvalidateStatic`, `RequestRedraw`, `CurrentArea`, `CurrentAreaPx`, `Scale`, `ScaleF`, `PxToDp`, `PxToDpf`.\n")
	out.WriteString("- **Ввод, буфер обмена, темы и система:** `Shortcut`, `FocusID`, `SetFocus`, `SetCursor`, `RegisterHoverArea`, `MousePosDp`, `MousePosPx`, `MousePressed`, `MouseDown`, `DoubleClicked`, `MiddleClicked`, `ClipboardText`, `SetClipboardText`, `ClipboardImage`, `SetClipboardImage`, `Remember`, `InvalidateRemember`, `ClearRemember`, `Theme`, `SetTheme`, `Strings`, `SetStrings`, `OpenFileDialog`, `OpenFilesDialog`, `SaveFileDialog`, `SelectFolderDialog`, `MessageBox`, `OpenURL`, `RevealInFileManager`, `AcquireSingleInstance`, `LoadWindowState`, `SaveWindowState`.\n")
	out.WriteString("- **Тестирование (`Headless`):** `NewHeadless`, `Headless.SetDraw`, `Headless.Step`, `Headless.AdvanceTime`, `Headless.SetScale`, `Headless.InjectMouseMove`, `Headless.InjectMouseClick`, `Headless.InjectMouseWheel`, `Headless.InjectKey`, `Headless.InjectText`, `Headless.InjectFocus`, `Headless.CaptureImage`, `AssertGolden`, `DefaultTestFonts`.\n")
}

func writeFullLLMs(out *bytes.Buffer, version string, idx *apiIndex) {
	writeCommonHeader(out, version, true)
	writeQuickStart(out)

	out.WriteString("## Константы и переменные пакета `anyui`\n\n```go\n")
	out.WriteString("const Auto Dp = -1\n\n")
	for _, grp := range idx.constGroups {
		if len(grp) == 1 && grp[0].name == "Auto" {
			continue
		}
		first := grp[0].name
		// Для кодов клавиш и кнопок мыши печатаем по одной константе с типом и реальным значением.
		if strings.HasPrefix(first, "Key") || strings.HasPrefix(first, "Mouse") {
			out.WriteString("const (\n")
			for _, c := range grp {
				tName := c.typeName
				if tName == "" && strings.HasPrefix(c.name, "Mouse") {
					tName = "MouseButton"
				}
				if c.valueExpr != "" && tName != "" {
					fmt.Fprintf(out, "\t%-14s %s = %s\n", c.name, tName, c.valueExpr)
				} else if c.valueExpr != "" {
					fmt.Fprintf(out, "\t%-14s = %s\n", c.name, c.valueExpr)
				} else {
					fmt.Fprintf(out, "\t%s\n", c.name)
				}
			}
			out.WriteString(")\n")
			continue
		}
		names := make([]string, 0, len(grp))
		for _, c := range grp {
			names = append(names, c.name)
		}
		out.WriteString("const ( ")
		out.WriteString(strings.Join(names, ", "))
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

var (
	reBacktick = regexp.MustCompile("`([^`]+)`")
	reIdent    = regexp.MustCompile(`^[A-Z][A-Za-z0-9_]*$`)
	reRecvSym  = regexp.MustCompile(`^(UI|App|Window|Headless)\.([A-Z][A-Za-z0-9_]*)$`)
)

// lintOutputs проверяет сгенерированные llms.txt и llms-full.txt на:
// 1. отсутствие утечек внутренних подпакетов (ui., platform., render.);
// 2. корректность всех упомянутых символов в ручных секциях (symbol-lint);
// 3. валидность всех опций в матрице применимости;
// 4. наличие ссылки на llms-full.txt в llms.txt.
func lintOutputs(idx *apiIndex, shortText, fullText string) error {
	if !strings.Contains(shortText, "`llms-full.txt`") {
		return fmt.Errorf("llms.txt не содержит ссылку на llms-full.txt")
	}

	// 1. Проверка отсутствия утечек подпакетов
	forbidden := []string{
		"= ui.", "= platform.", "= render.",
		"platform.", "render.",
		"github.com/kirpicheff/anyui/ui",
		"github.com/kirpicheff/anyui/platform",
		"github.com/kirpicheff/anyui/render",
	}
	for name, txt := range map[string]string{"llms.txt": shortText, "llms-full.txt": fullText} {
		for _, bad := range forbidden {
			if strings.Contains(txt, bad) {
				return fmt.Errorf("%s содержит утечку внутреннего подпакета %q", name, bad)
			}
		}
	}

	// 2. Проверка матрицы опций: каждое имя в колонке опций должно быть реальной Option-функцией (или именем виджета в скобках)
	for _, row := range optionMatrixRows {
		for _, m := range reBacktick.FindAllStringSubmatch(row.options, -1) {
			tok := m[1]
			if idx.optionNames[tok] {
				continue
			}
			// Допускаются пояснения вида (`Checkbox`), (`ScrollView`), (`Select`), (`ListView`), (`Table`), (`TreeView`), (`ContextMenu`), (`Tooltip`)
			if idx.methodNames["UI"][tok] {
				continue
			}
			return fmt.Errorf("матрица опций содержит неизвестную опцию %q", tok)
		}
	}

	// 3. Symbol-lint по ручным секциям llms.txt (вне fenced-блоков кода)
	ignoreSymbols := map[string]bool{
		// Поля структур, упоминаемые в прозе
		"WidthDp": true, "IconRGBA": true, "Key": true, "ID": true, "Label": true,
		"FromIndex": true, "ToIndex": true, "FromKey": true, "ToKey": true,
		"SelectedID": true, "ClosedID": true, "Added": true, "MovedFrom": true, "MovedTo": true, "DetachedID": true,
		"RowClicked": true, "RowDoubleClicked": true, "RowMiddleClicked": true, "SortChanged": true, "ColumnResized": true, "Drag": true,
	}

	inCode := false
	for lineNo, line := range strings.Split(shortText, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		for _, m := range reBacktick.FindAllStringSubmatch(line, -1) {
			tok := m[1]
			if rm := reRecvSym.FindStringSubmatch(tok); rm != nil {
				recv, meth := rm[1], rm[2]
				if !idx.methodNames[recv][meth] {
					return fmt.Errorf("llms.txt:%d: метод %s.%s не найден в пакете anyui", lineNo+1, recv, meth)
				}
				continue
			}
			if !reIdent.MatchString(tok) {
				continue
			}
			if ignoreSymbols[tok] || idx.allSymbols[tok] || idx.methodNames["UI"][tok] || idx.methodNames["Headless"][tok] {
				continue
			}
			return fmt.Errorf("llms.txt:%d: неизвестный символ `%s`", lineNo+1, tok)
		}
	}
	return nil
}
