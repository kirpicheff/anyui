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
	"runtime/debug"
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
	subPkgTypes   map[string]map[string]*ast.TypeSpec        // pkg -> typeName -> TypeSpec
	subPkgConsts  map[string]map[string]string               // pkg -> constName -> literal value
	subPkgMethods map[string]map[string]map[string]string    // pkg -> recvType -> methodName -> fullSig
	aliases      map[string]aliasTarget              // publicName -> subpkg target
	localTypes   map[string]*ast.TypeSpec
	constGroups  [][]constEntry
	vars         []string
	optionFuncs    []string
	optionNames    map[string]bool
	otherFuncs     []string
	otherFuncNames map[string]bool
	methods        map[string][]string // receiverType -> []fullSignature
	methodNames    map[string]map[string]bool
	uiBoolMethods  map[string]bool // методы UI, возвращающие bool
	allTypeNames   map[string]bool // все экспортированные типы пакета anyui
	allSymbols     map[string]bool // все экспортированные символы пакета anyui (типы, функции, константы, переменные)
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

// detectVersion возвращает последний git-тег, версию из BuildInfo или "dev".
func detectVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	if raw, err := cmd.Output(); err == nil {
		if ver := strings.TrimSpace(string(raw)); ver != "" {
			return ver
		}
	}
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "dev"
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
		subPkgTypes:    make(map[string]map[string]*ast.TypeSpec),
		subPkgConsts:   make(map[string]map[string]string),
		subPkgMethods:  make(map[string]map[string]map[string]string),
		aliases:      make(map[string]aliasTarget),
		localTypes:   make(map[string]*ast.TypeSpec),
		optionNames:    make(map[string]bool),
		otherFuncNames: make(map[string]bool),
		methods:        make(map[string][]string),
		methodNames:    make(map[string]map[string]bool),
		uiBoolMethods:  make(map[string]bool),
		allTypeNames:   make(map[string]bool),
		allSymbols:     make(map[string]bool),
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
		idx.subPkgMethods[sub] = make(map[string]map[string]string)
		for _, f := range subFiles {
			for _, decl := range f.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok {
					if fd.Recv != nil && len(fd.Recv.List) > 0 && ast.IsExported(fd.Name.Name) {
						rn := receiverTypeName(fd.Recv.List[0].Type)
						if ast.IsExported(rn) {
							if idx.subPkgMethods[sub][rn] == nil {
								idx.subPkgMethods[sub][rn] = make(map[string]string)
							}
							if _, exists := idx.subPkgMethods[sub][rn][fd.Name.Name]; !exists {
								idx.subPkgMethods[sub][rn][fd.Name.Name] = idx.formatFuncSig(fd)
							}
						}
					}
					continue
				}
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
						idx.allTypeNames[ts.Name.Name] = true
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
						idx.otherFuncNames[d.Name.Name] = true
					}
				} else {
					recvName := receiverTypeName(d.Recv.List[0].Type)
					if ast.IsExported(recvName) {
						idx.methods[recvName] = append(idx.methods[recvName], sig)
						if idx.methodNames[recvName] == nil {
							idx.methodNames[recvName] = make(map[string]bool)
						}
						idx.methodNames[recvName][d.Name.Name] = true
						if recvName == "UI" && returnsSingleType(d, "bool") {
							idx.uiBoolMethods[d.Name.Name] = true
						}
					}
				}
			}
		}
	}

	// Переносим методы алиасных типов из подпакетов (Canvas, SingleInstanceGuard, Dp, Px).
	for _, pubName := range []string{"Canvas", "SingleInstanceGuard", "Dp", "Px"} {
		target, ok := idx.aliases[pubName]
		if !ok {
			continue
		}
		mMap := idx.subPkgMethods[target.pkg][target.name]
		if len(mMap) == 0 {
			continue
		}
		if idx.methodNames[pubName] == nil {
			idx.methodNames[pubName] = make(map[string]bool)
		}
		for mName, sig := range mMap {
			idx.methods[pubName] = append(idx.methods[pubName], sig)
			idx.methodNames[pubName][mName] = true
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

func returnsSingleType(fn *ast.FuncDecl, typeName string) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	id, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	return ok && id.Name == typeName
}

func returnsOption(fn *ast.FuncDecl) bool {
	return returnsSingleType(fn, "Option")
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
		"platform.CustomCursor", "CustomCursor",
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

func writeCommonHeader(out *bytes.Buffer, version string, isFull bool, idx *apiIndex) {
	if isFull {
		fmt.Fprintf(out, "# AnyUI (%s) — Полный справочник API\n\n", version)
	} else {
		fmt.Fprintf(out, "# AnyUI (%s)\n\n", version)
	}
	out.WriteString("Лёгкий кроссплатформенный UI-фреймворк на Go с программным 2D-рендерингом (без WebView2 и OpenGL/GPU).\n\n")
	if !isFull {
		out.WriteString("> Полный офлайн-справочник со всеми сигнатурами, опциями, константами и структурами: `llms-full.txt`.\n\n")
	}
	out.WriteString("## Документация и примеры\n\n")
	out.WriteString("- Примеры godoc: `example_test.go`; интерактивная витрина всех виджетов: `go run ./cmd/demo`.\n")
	out.WriteString("- Оффлайн-документация: `docs/en/*.md`, `docs/ru/*.md` (обзор, виджеты, графика, темы, системная интеграция, тестирование, единицы Dp/Px).\n\n")
	out.WriteString("## Версия и источник истины\n\n")
	out.WriteString("- Импорт: `import \"github.com/kirpicheff/anyui\"` (`go get github.com/kirpicheff/anyui`).\n")
	out.WriteString("- Проверить установленную версию модуля: `go list -m github.com/kirpicheff/anyui`.\n")
	out.WriteString("- При расхождении этого документа с установленным пакетом источник истины — `go doc` (например, `go doc github.com/kirpicheff/anyui.UI.Table` или `go doc github.com/kirpicheff/anyui.Config`).\n")
	out.WriteString("- Подпакеты являются внутренними; прикладной код импортирует **только** корневой пакет `github.com/kirpicheff/anyui`.\n\n")

	out.WriteString("## Правила работы с API\n\n")
	out.WriteString("- **Immediate Mode и многопоточность:** состояние хранится вне `drawFn`; вызовы методов `UI` и чтение/изменение состояния интерфейса выполняются **только в главном потоке** внутри `drawFn`. Для обновления состояния из фоновых горутин всегда используйте `App.Dispatch` (`app.Dispatch(func() { ... })`) или `Remember`. Если второй процесс приложения стартует с уже занятым `Config.SingleInstance.ID`, `App.Run` возвращает `ErrAlreadyRunning` (см. правило «Единственный экземпляр» ниже).\n")
	out.WriteString("- **Единицы измерения (`Dp` vs `Px`):** весь API раскладки, опций, оверлеев и ввода Headless (`LayoutRect`, `Tooltip`, `PushOverlay`, `PushNonBlockingOverlay`, `PushBackdrop`, `Popover`, `ContextMenu`, `RegisterHoverArea`, `InjectMouse*`, `CurrentArea`, `MousePosDp`) работает строго в логических `Dp`. Физические `Px` используются только в колбэках рисования (`CanvasView`, `InteractiveCanvas`, `Table`, `ListView`), прямых методах (`DrawText`, `DrawTextMono`, `DrawSVG`, `CurrentAreaPx`, `MousePosPx`, `LastDirtyRectsPx`, `ActiveOverlayRects`, `TitleBarHitTest`, `InjectMouse*Px`) и на границе ОС (`Config`, `Window`).\n")
	out.WriteString("- **`Rect` и `PointF` не типизированы по единицам:** `Rect` содержит `int`-поля, `PointF` — `float32`. В `Dp`-API (`CurrentArea`, `LayoutRect`, `PushOverlay`, `PushNonBlockingOverlay`, `PushBackdrop`, `Tooltip`, `Popover`, `ContextMenu`, `RegisterHoverArea`) они трактуются как `Dp`, а в колбэках рисования и `CurrentAreaPx`/`LastDirtyRectsPx`/`ActiveOverlayRects` — как `Px`.\n")
	out.WriteString("- **Холст (`Canvas`) и текст на нём:** все методы `Canvas` принимают и возвращают физические `Px` (конвертация — через `Scale`/`ScaleF`/`PxToDp`/`PxToDpf`). У `UI.DrawText`, `UI.DrawTextMono` и `UI.DrawSVG` единицы смешанные: координаты `(xPx, yPx)` — в `Px`, а размер (`sizeDp`) — в `Dp`; у `Canvas.DrawSVG` и координаты, и размер — в `Px`. У `Canvas` нет метода рисования текста: внутри `CanvasView` / `InteractiveCanvas` текст выводится через `UI.DrawText` / `UI.DrawTextMono` в абсолютных координатах `Px` (`areaPx.X + offset`, `areaPx.Y + offset`). `UI.Canvas` возвращает холст текущего кадра для прямой отрисовки вне `CanvasView` (например, в `ListView`/`Table`/`PushOverlay`) — координаты абсолютные `Px` окна с активным клипом.\n")
	out.WriteString("- **Статический слой (`StaticLayer`):** `StaticLayer` кеширует результат `fn` между кадрами в отдельный буфер; после изменения данных внутри слоя вызывайте `InvalidateStatic`, иначе слой не перерисуется.\n")
	out.WriteString("- **Единственный экземпляр (`SingleInstanceGuard`):** два варианта. (а) Ручной: `anyui.AcquireSingleInstance` → `SingleInstanceGuard.SetCallback` (получает `args []string` второго экземпляра) → `SingleInstanceGuard.StartIPC` после запуска цикла событий → `SingleInstanceGuard.Close` при выходе. (б) Декларативный: заполните `Config.SingleInstance` (`ID` или заранее полученный `Guard`, опционально `OnSecondInstance`, `SkipActivate`, `Args`) — `App` сам захватит экземпляр, сам вызовет `SingleInstanceGuard.StartIPC`, сам закроет guard при выходе и сам доставит `OnSecondInstance`; вручную `SingleInstanceGuard.SetCallback`/`SingleInstanceGuard.StartIPC`/`SingleInstanceGuard.Close` в этом варианте вызывать **не нужно** (свой callback будет перезаписан).\n")
	out.WriteString("- **Единое задание размеров через опции:** `ScrollView`, `CanvasView`, `InteractiveCanvas`, `Image`, `ListView`, `ListViewEx`, `SplitPane` по умолчанию занимают всё доступное пространство по ширине и высоте (`Auto`). `TextArea` (120 dp), `Table`, `TableEx`, `LogView`, `VirtualTextView` (260 dp), `TreeView`, `TreeViewEx` (240 dp), `LineChart`, `BarChart` (220 dp) и `Sparkline` (44 dp) по умолчанию занимают доступную ширину, но имеют фиксированную дефолтную высоту — для растягивания на остаток высоты контейнера передайте `FillHeight` (или `Height(Auto)`). Явные размеры и ограничения задаются опциями `Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`. `Card` по умолчанию подбирает высоту по содержимому (`AutoHeight`); для растягивания передайте `FillHeight`.\n")
	out.WriteString("- **Потоковый холст (`InteractiveCanvas`):** в `CanvasCallbacks` флаг `Streaming: true` включает быстрый путь обновления по `Window.WakeUp` (перерисовывается только `OnDraw` холста без прохода всего дерева `UI` и без полного diff буфера; при активных оверлеях поверх реально перезаписанной области `Canvas.DirtyBounds` перерисовываются только пересекающиеся оверлеи с клиппингом по этой области и частичной отправкой в окно), а `DirectInput: true` доставляет события мыши и клавиатуры напрямую в колбэки холста. Если в обработчике изменилось внешнее состояние интерфейса за пределами холста, вызовите `UI.RequestRedraw`.\n")
	out.WriteString("- **Колонки таблиц:** в `Table` и `TableEx` колонки с `WidthDp <= 0` автоматически делят оставшуюся ширину таблицы.\n")
	out.WriteString("- **Декларативная навигация:** для программной прокрутки используйте `ScrollToIndex` в `ListView`, `ListViewEx`, `Table` и `TableEx`, а для `ScrollView` — `ScrollToID` в паре с маркером `Anchor` (или `id` дочернего виджета).\n")
	out.WriteString("- **Иконка окна:** подхватывается автоматически из `.syso`-ресурсов, либо задаётся через `Config.IconRGBA` или `Window.SetIcon`.\n")
	out.WriteString("- **Идентификаторы, ключи и `IDScope`:** `IDScope` (или `PushID`/`PopID`) изолирует `id` виджетов внутри циклов и переиспользуемых компонентов. Передача пустого `id == \"\"` автоматически генерирует детерминированный идентификатор по порядковому номеру виджета в текущем `IDScope`. В `Select`, `MultiSelect`, `TabBar`, `TreeView`, `TreeViewEx` и `Breadcrumbs` ключи элементов (`Key` / `ID`) локальны для конкретного виджета и должны быть уникальны в его пределах — при дублировании ключей выделение и клик совпадут для всех дубликатов. В `Breadcrumbs`, если у сегмента `ID == \"\"`, возвращается его `Label`.\n")
	out.WriteString("- **Выпадающие списки и вкладки:** `Select` с опцией `Searchable` заменяет комбобокс; `MultiSelect` работает с `map[string]bool`; `TabBar` и `TabItems` работают на строковых ключах.\n")
	out.WriteString("- **Раскладка и оверлеи:** для типового экрана используйте `Page` или `DockTop` / `DockBottom` / `ColumnFlex`. Для диалогов всегда используйте `Modal` — `PushOverlay` предназначен только для небольших всплывающих контролов. В `Modal`, `Popover`, `FloatingPanel` и `Drawer` передавайте в `open *bool` напрямую указатель на поле состояния (`&state.ModalOpen`), а не на локальную переменную — отрисовка оверлея и закрытие по `×` выполняются в конце кадра внутри `PushOverlay`.\n")
	out.WriteString("- **Точки расширения для нативных интеграций:** `Window.NativeHandle` возвращает нативный дескриптор окна ОС (HWND на Windows, XID на X11, ID wl_surface на Wayland, NSWindow* на macOS; `0` в `Headless`), `HasActiveOverlays` возвращает признак наличия активных оверлеев, а `ActiveOverlayRects` — их прямоугольники в `Px` (включая `Tooltip`, `PushNonBlockingOverlay` и уведомления `ShowToast`), позволяя внешним нативным поверхностям уступать z-order и переключаться на софтверный фолбэк (`Canvas.DrawBGRAFast`) только при реальном пересечении.\n")
	out.WriteString("- **Drag & Drop (Reorder и кросс-виджетный перенос):** `Reorderable` + `OnReorder` работают внутри `Table`, `TableEx`, `ListView`, `ListViewEx` (индексы `FromIndex`/`ToIndex`) и `TreeView`, `TreeViewEx` (ключи `FromKey`/`ToKey`; с `AllowReparent` доступны смена родителя и `DropInside`). Для кросс-виджетного переноса (например, `TreeView` → `Table`) используйте `DragSource` с типизированным `DragPayload` на источнике и `AcceptDrop` + `OnDrop` на приёмнике.\n")
	out.WriteString("- **Горячие клавиши (`Key`) и Headless-тесты:** из букв экспортированы базовые константы (`KeyA`, `KeyC`, `KeyV`, `KeyX`, `KeyY`, `KeyZ`); для остальных клавиш в `Shortcut` / `InjectKey` передавайте виртуальный код `anyui.Key('S')` или VK-код (например, `0x70` для F1). Для `anyui.NewHeadless` обязательно указывать `Config.FontRegular` (для детерминированных тестов используйте `anyui.DefaultTestFonts`).\n\n")

	out.WriteString("## Семантика возвращаемых значений\n\n")
	out.WriteString("- `Button`, `IconButton`, `Hyperlink` → `bool`: `true` **только в кадре нажатия**.\n")
	out.WriteString("- `Toggle`, `Checkbox`, `RadioButton`, `RadioGroup`, `TextInput`, `TextArea`, `MaskedInput`, `Select`, `MultiSelect`, `TagInput`, `Slider`, `RangeSlider`, `NumberInput`, `ColorPicker`, `DatePicker`, `TimePicker`, `DateTimePicker`, `ListView`, `ListViewEx`, `TreeView`, `TreeViewEx`, `Pagination`, `Collapsible`, `Accordion` → `bool`: `true` **только в кадре изменения значения** (не означает «виджет сейчас активен/включён»; текущее состояние хранится в переданном указателе/карте).\n")
	out.WriteString("- `Breadcrumbs` → `string`: возвращает `ID` кликнутого сегмента (или `Label`, если `ID == \"\"`), либо `\"\"` если клика в этом кадре не было.\n")
	out.WriteString("- `TabBar` → `TabBarEvent`: `SelectedID` (смена вкладки), `ClosedID` (клик по крестику/СКМ), `Added` (клик по `+`), `MovedFrom`/`MovedTo` (перетаскивание, `-1` если не было), `DetachedID` (отрыв вкладки), `AttachedID`/`AttachedIndex` (обратное прикрепление откреплённого окна, `-1` если не было).\n")
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

	writeBaseTypes(out, idx)

	out.WriteString("## Матрица применимости опций (`Option`)\n\n")
	out.WriteString("Основные поддерживаемые опции (неприменимые игнорируются без ошибки):\n\n")
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
	out.WriteString("6. **Смешивание `Px` и `Dp`:** передача `CurrentAreaPx` в `PushOverlay`, `PushNonBlockingOverlay`, `LayoutRect`, `Tooltip`, `Popover`, `ContextMenu` или `RegisterHoverArea` ломает координаты при DPI > 100%. Для них всегда берите `CurrentArea` (`Dp`).\n")
	out.WriteString("7. **Локальная переменная `&open` в `Modal` / `Popover` / `FloatingPanel` / `Drawer`:** эти виджеты откладывают отрисовку и обработку клика по `×` в `PushOverlay` до конца кадра (после завершения `drawFn` и вспомогательных функций). Если передать `&open` от локальной копии (`open := st.Open; ui.Modal(..., &open, ...); st.Open = open`), закрытие по `×` и кнопкам внутри `fn` потеряется. Всегда передавайте указатель напрямую на долгоживущее поле состояния (`&st.Open`).\n\n")
}

type optionMatrixRow struct {
	widgets string
	options string
}

// optionMatrixRows проверена по реальным обращениям к WidgetStyle в ui/*.go.
var optionMatrixRows = []optionMatrixRow{
	{
		"`Label`, `SelectableLabel`, `StatusDot`, `Hyperlink`",
		"`FontSize`, `Bold`, `Mono`, `TextColor`, `Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Disabled` (кроме `StatusDot`), `WordWrap` (`Label`, `SelectableLabel`), `Ellipsis` (`Label`), `TextAlignLeft`/`TextAlignCenter`/`TextAlignRight` (`Label`), `LetterSpacing`/`TextShadow` (`Label`, `StatusDot`)",
	},
	{
		"`Button`, `IconButton`",
		"`Primary`, `Secondary`, `Danger`, `Ghost`, `Icon`, `Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Padding`, `Radius`, `FontSize`, `Bold`, `Mono`, `LetterSpacing`, `TextShadow`, `Bg`, `HoverBg`, `TextColor`, `HoverTextColor`, `BorderColor`, `HoverBorderColor`, `FocusBorderColor` (`Button`), `Disabled`",
	},
	{
		"`Checkbox`, `Toggle`, `RadioButton`, `RadioGroup`",
		"`Disabled`, `FontSize`, `Bold`, `Mono`, `TextColor` (в `Checkbox` красит заливку флажка), `Bg` (`Toggle`), `Height`/`Radius`/`Indeterminate` (`Checkbox`)",
	},
	{
		"`TextInput`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Padding`, `FontSize`, `Bold`, `Mono`, `TextColor`, `Bg`, `BorderColor`, `FocusBorderColor`, `Radius`, `TextAlignLeft`, `TextAlignCenter`, `TextAlignRight`, `Password`, `PasswordToggle`, `Clearable`, `ReadOnly`, `Disabled`, `InputFilter`, `OnSubmit`, `Error`, `Icon`, `IconRight`",
	},
	{
		"`MaskedInput`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Padding`, `FontSize`, `Bold`, `Mono`, `TextColor`, `Bg`, `BorderColor`, `FocusBorderColor`, `Radius`, `ReadOnly`, `Disabled`, `OnSubmit`, `Error`",
	},
	{
		"`TextArea`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Padding`, `FontSize`, `Bold`, `Mono`, `TextColor`, `Bg`, `BorderColor`, `FocusBorderColor`, `Radius`, `LineNumbers`, `ReadOnly`, `Disabled`, `InputFilter`, `Error`",
	},
	{
		"`NumberInput`, `Slider`, `RangeSlider`, `ColorPicker`, `DatePicker`, `TimePicker`, `DateTimePicker`",
		"`Disabled`, `Width`, `Height`/`Radius` (`NumberInput`)",
	},
	{
		"`Select`, `MultiSelect`",
		"`Width`, `Height`, `Disabled`, `Bg`, `Radius`, `Bold`, `Mono`, `Searchable` (`Select`)",
	},
	{
		"`TagInput`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Disabled`",
	},
	{
		"`Card`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `AutoHeight`, `Padding`, `Radius`, `Bg`, `BorderColor`",
	},
	{
		"`ScrollView`, `CanvasView`, `InteractiveCanvas`, `Image`, `SplitPane`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Padding` (`ScrollView`), `ScrollToID` (`ScrollView`), `Radius` (`SplitPane`)",
	},
	{
		"`ListView`, `ListViewEx`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Bg`, `BorderColor`, `FocusBorderColor`, `Radius`, `Disabled`, `ScrollToIndex`, `Reorderable`, `OnReorder`, `DragSource`, `AcceptDrop`, `OnDrop`",
	},
	{
		"`Table`, `TableEx`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Bg`, `BorderColor`, `FocusBorderColor`, `Radius`, `Disabled`, `ScrollToIndex`, `Reorderable`, `OnReorder`, `DragSource`, `AcceptDrop`, `OnDrop`",
	},
	{
		"`TreeView`, `TreeViewEx`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Bg`, `BorderColor`, `FocusBorderColor`, `Radius`, `Disabled`, `Reorderable`, `AllowReparent`, `OnReorder`, `DragSource`, `AcceptDrop`, `OnDrop`",
	},
	{
		"`LogView`, `VirtualTextView`, `LineChart`, `BarChart`, `Sparkline`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Bg`, `TextColor` (`Sparkline`)",
	},
	{
		"`TabBar`, `Collapsible`, `Accordion`",
		"`Width`, `Height`, `FillWidth`, `FillHeight`, `TabOrientation`, `HideAddButton` (`TabBar`)",
	},
	{
		"`MenuBar`, `Toolbar`, `StatusBar`, `Breadcrumbs`, `Pagination`, `TitleBar`",
		"`Width`, `Height`, `MinWidth`, `MaxWidth`, `MinHeight`, `MaxHeight`, `FillWidth`, `FillHeight`, `Bg` (`MenuBar`, `Toolbar`, `StatusBar`, `TitleBar`), `FontSize` (`MenuBar`, `Breadcrumbs`, `TitleBar`), `TextColor` (`TitleBar`)",
	},
	{
		"`Badge`, `ProgressBar`, `Separator`, `Spinner`",
		"`TextColor`, `Width`/`Height`/`Bg` (кроме `Spinner`), `Radius`/`Danger` (`Badge`, `ProgressBar`), `Padding`/`FontSize`/`Bold`/`Mono`/`Primary`/`Secondary` (`Badge`)",
	},
	{
		"`Modal`, `Popover`, `FloatingPanel`, `Drawer`",
		"`Width`, `Height`, `AutoHeight`/`MinHeight`/`MaxHeight` (`Modal`), `Padding`/`Bg` (`Drawer`)",
	},
	{
		"`ContextMenu`, `Tooltip`",
		"`Width` (`ContextMenu`), `Padding` (`Tooltip`), `Bg`, `BorderColor`, `Radius`, `FontSize`, `Bold`, `Mono`, `TextColor`, `LetterSpacing`, `TextShadow`",
	},
}

func writeBaseTypes(out *bytes.Buffer, idx *apiIndex) {
	out.WriteString("## Базовые типы\n\n```go\n")
	out.WriteString("type Dp int               // Логические DPI-независимые пиксели\n")
	for _, sig := range idx.methods["Dp"] {
		out.WriteString(sig + "\n")
	}
	out.WriteString("type Px int               // Физические пиксели буфера отрисовки\n")
	for _, sig := range idx.methods["Px"] {
		out.WriteString(sig + "\n")
	}
	out.WriteString("const Auto Dp = -1        // Sentinel-значение: занять всё доступное пространство контейнера\n")
	out.WriteString("type Key = uint32         // Виртуальный код клавиши (VK-коды, передаётся в Shortcut / InjectKey / KeyEvent.KeyCode)\n")
	out.WriteString("type Modifiers struct { Shift, Ctrl, Alt, Win bool }\n")
	out.WriteString("type MouseButton uint8    // MouseLeft = 1, MouseRight = 2, MouseMiddle = 3\n")
	out.WriteString("type CursorType uint8     // Системный курсор (CursorArrow, CursorHand, CursorIBeam, CursorResizeH/V/NWSE/NESW, CursorNotAllowed, CursorCrosshair, CursorMove, CursorNone)\n")
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
	out.WriteString("type HitTestResult uint8     // HitClient, HitCaption, HitLeft/Right/Top/Bottom/TopLeft/TopRight/BottomLeft/BottomRight\n")
	out.WriteString("type NotificationIcon uint8  // NotifyNone, NotifyInfo, NotifyWarning, NotifyError\n")
	out.WriteString("type MessageBoxStyle uint8   // MsgInfo, MsgWarning, MsgError, MsgQuestion\n")
	out.WriteString("type MessageBoxButtons uint8 // MsgOK, MsgOKCancel, MsgYesNo, MsgYesNoCancel\n")
	out.WriteString("type MessageBoxResult uint8  // ResNone, ResOK, ResCancel, ResYes, ResNo\n")
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
	writeCommonHeader(out, version, false, idx)
	writeQuickStart(out)

	out.WriteString("## Ключевые структуры старта (офлайн-минимум; все структуры — в `llms-full.txt`)\n\n```go\n")
	for _, name := range []string{"Config", "PageConfig"} {
		if def, ok := idx.formatTypeDef(name); ok {
			out.WriteString(def)
			out.WriteString("\n\n")
		}
	}
	out.WriteString("```\n\n")

	out.WriteString("## Карта API (полные сигнатуры и структуры — в `llms-full.txt`)\n\n")
	out.WriteString("- **Приложение (`App`):** `anyui.NewApp`, `App.Run`, `App.Dispatch`, `App.OpenWindow`, `App.MainWindow`, `App.Show`, `App.Quit`, `App.SetStrings`, `App.SetTrayMenu`, `App.SetTrayIcon`, `App.SetTrayTooltip`, `App.ShowNotification`.\n")
	out.WriteString("- **Окно (`Window`):** `Window.SetTitle`, `Window.SetCustomCursor`, `Window.SetSize`, `Window.SetPosition`, `Window.Position`, `Window.SetIcon`, `Window.SetTheme`, `Window.SetStrings`, `Window.SetDarkTitle`, `Window.SetTitleBarColors`, `Window.SetHideOnClose`, `Window.SetOnClose`, `Window.SetOnMove`, `Window.SetOnMoveEnd`, `Window.SetReattachTabID`, `Window.ReattachTabID`, `Window.SetOnDropFiles`, `Window.SetOnDragEnter`, `Window.SetOnDragOver`, `Window.SetOnDragLeave`, `Window.Show`, `Window.Hide`, `Window.BringToFront`, `Window.Minimize`, `Window.Maximize`, `Window.Restore`, `Window.IsMinimized`, `Window.IsMaximized`, `Window.IsFullscreen`, `Window.SetFullscreen`, `Window.SetAlwaysOnTop`, `Window.IsAlwaysOnTop`, `Window.State`, `Window.RestoreState`, `Window.ClipboardText`, `Window.SetClipboardText`, `Window.ClipboardImage`, `Window.SetClipboardImage`, `Window.NativeHandle`, `Window.Invalidate`, `Window.WakeUp`, `Window.Close`.\n")
	out.WriteString("- **Раскладка и контейнеры (`UI`):** `Column`, `ColumnCentered`, `ColumnWith`, `ColumnFlex`, `Row`, `RowCentered`, `RowWith`, `RowFlex`, `Grid`, `Flow`, `FormLayout`, `FormRow`, `Padding`, `PaddingXY`, `Spacer`, `FlexSpacer`, `Expand`, `DockTop`, `DockBottom`, `Page`, `Card`, `ScrollView`, `Anchor`, `SplitPane`, `Collapsible`, `Accordion`, `LayoutRect`, `IDScope`, `PushID`, `PopID`.\n")
	out.WriteString("- **Ввод и формы (`UI`):** `Button`, `IconButton`, `Hyperlink`, `Toggle`, `Checkbox`, `RadioButton`, `RadioGroup`, `TextInput`, `TextArea`, `MaskedInput`, `NumberInput`, `Slider`, `RangeSlider`, `Select`, `MultiSelect`, `TagInput`, `ColorPicker`, `DatePicker`, `TimePicker`, `DateTimePicker`.\n")
	out.WriteString("- **Списки, деревья, таблицы и графики (`UI`):** `ListView`, `ListViewEx`, `TreeView`, `TreeViewEx`, `Table`, `TableEx`, `LogView`, `VirtualTextView`, `Sparkline`, `LineChart`, `BarChart`.\n")
	out.WriteString("- **Текст, индикаторы и навигация (`UI`):** `Label`, `SelectableLabel`, `StatusDot`, `Badge`, `Separator`, `ProgressBar`, `Spinner`, `TabBar`, `MenuBar`, `Toolbar`, `StatusBar`, `Breadcrumbs`, `Pagination`, `TitleBar`, `TitleBarHitTest`.\n")
	out.WriteString("- **Оверлеи и уведомления (`UI`):** `Modal`, `Popover`, `FloatingPanel`, `Drawer`, `ContextMenu`, `Tooltip`, `PushOverlay`, `PushNonBlockingOverlay`, `PushBackdrop`, `HasActiveOverlays`, `ActiveOverlayRects`, `ShowToast`, `SetToastPosition`, `SetToastOffset`.\n")
	out.WriteString("- **Графика и слои (`UI`):** `Image`, `CanvasView`, `InteractiveCanvas`, `Canvas`, `DrawText`, `DrawTextMono`, `DrawSVG`, `StaticLayer`, `InvalidateStatic`, `RequestRedraw`, `CurrentArea`, `CurrentAreaPx`, `Scale`, `ScaleF`, `PxToDp`, `PxToDpf`.\n")
	out.WriteString("- **Отладка и кеши (`UI`, в прикладном коде обычно не нужны):** `LastFrameStaticBlitted`, `LastDirtyRectsPx`, `SetDebugDirtyRect`, `SetSmartMouseMove`, `SkippedMouseMoves`, `GlyphCacheStats`, `ClearGlyphCache`, `ScaledImageCacheStats`, `ClearScaledImageCache`.\n")
	out.WriteString("- **Ввод, буфер обмена, контекст и темы (`UI`):** `UI.App`, `UI.Window`, `Shortcut`, `FocusID`, `SetFocus`, `CaretPos`, `SetCaretPos`, `SetCursor`, `RegisterHoverArea`, `MousePosDp`, `MousePosPx`, `MousePressed`, `MouseDown`, `DoubleClicked`, `MiddleClicked`, `ClipboardText`, `SetClipboardText`, `ClipboardImage`, `SetClipboardImage`, `InvalidateRemember`, `ClearRemember`, `Theme`, `SetTheme`, `Strings`, `SetStrings`.\n")
	out.WriteString("- **Холст (`Canvas`):** ")
	canvasNames := sortedKeys(idx.methodNames["Canvas"])
	for i, m := range canvasNames {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString("`Canvas." + m + "`")
	}
	out.WriteString(".\n")
	out.WriteString("- **Single-instance (`SingleInstanceGuard`):** ")
	sigNames := sortedKeys(idx.methodNames["SingleInstanceGuard"])
	for i, m := range sigNames {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString("`SingleInstanceGuard." + m + "`")
	}
	out.WriteString(".\n")
	out.WriteString("- **Функции пакета (`anyui.*`):** `anyui.Remember` (generic), `anyui.RGB`, `anyui.RGBA`, `anyui.Hex`, `anyui.UniformRadii`, `anyui.NavyTheme`, `anyui.DarkTheme`, `anyui.LightTheme`, `anyui.SystemTheme`, `anyui.IsSystemDarkTheme`, `anyui.RussianStrings`, `anyui.EnglishStrings`, `anyui.TabItems`, `anyui.SelectItems`, `anyui.SetTreeNodeChecked`, `anyui.SyncTreeCheckState`, `anyui.OpenFileDialog`, `anyui.OpenFilesDialog`, `anyui.SaveFileDialog`, `anyui.SelectFolderDialog`, `anyui.MessageBox`, `anyui.OpenURL`, `anyui.RevealInFileManager`, `anyui.AcquireSingleInstance`, `anyui.LoadWindowState`, `anyui.SaveWindowState`.\n")
	out.WriteString("- **Тестирование (`Headless`):** `anyui.NewHeadless`, `Headless.SetDraw`, `Headless.Step`, `Headless.AdvanceTime`, `Headless.SetTime`, `Headless.Time`, `Headless.SetScale`, `Headless.InjectMouseMove`, `Headless.InjectMouseMovePx`, `Headless.InjectMouseDown`, `Headless.InjectMouseDownPx`, `Headless.InjectMouseUp`, `Headless.InjectMouseUpPx`, `Headless.InjectMouseClick`, `Headless.InjectMouseClickPx`, `Headless.InjectMouseWheel`, `Headless.InjectMouseWheelPx`, `Headless.InjectKey`, `Headless.InjectKeyDown`, `Headless.InjectKeyUp`, `Headless.InjectText`, `Headless.InjectFocus`, `Headless.FocusID`, `Headless.FocusOrder`, `Headless.CaptureImage`, `Headless.CapturePNG`, `Headless.App`, `Headless.Window`, `Headless.UI`, `Headless.Close`, `anyui.AssertGolden`, `anyui.DefaultTestFonts`.\n")
}

func writeFullLLMs(out *bytes.Buffer, version string, idx *apiIndex) {
	writeCommonHeader(out, version, true, idx)
	writeQuickStart(out)

	out.WriteString("## Константы и переменные пакета `anyui`\n\n```go\n")
	out.WriteString("// Значения важны для Key и MouseButton; порядковые enum-ы используются по имени.\n")
	out.WriteString("// Константа Auto Dp = -1 описана выше в разделе «Базовые типы».\n\n")
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
		if strings.HasPrefix(v, "Err") {
			out.WriteString("var " + v + " error\n")
		} else {
			out.WriteString("var " + v + "\n")
		}
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
		"LogEntry", "ChartSeries", "CanvasCallbacks", "CustomCursor", "KeyEvent", "DragPayload", "DragEvent",
		"TrayConfig", "MenuItem", "Notification",
		"FileDialogOptions", "FolderDialogOptions", "FileFilter",
		"SingleInstanceConfig", "WindowState",
		"Color", "Rect", "CornerRadii", "GradientStop", "LinearGradient", "RadialGradient", "BoxShadow", "PointF",
		"Theme", "Strings", "TestHelper",
	}
	out.WriteString("## Структуры и типы данных\n\n```go\n")
	out.WriteString("// Непрозрачные типы (конструируются только через API пакета):\n")
	out.WriteString("type Option interface{ /* непрозрачный тип; создаётся конструкторами Option */ }\n")
	out.WriteString("type Canvas struct{}  // Программный 2D-холст (возвращается UI.Canvas() или передаётся параметром в колбэк CanvasView/InteractiveCanvas; все методы в Px)\n")
	out.WriteString("type SingleInstanceGuard struct{} // Непрозрачный тип (возвращается AcquireSingleInstance; методы см. в секции «Методы SingleInstanceGuard»)\n\n")
	for _, name := range structOrder {
		if def, ok := idx.formatTypeDef(name); ok {
			out.WriteString(def)
			out.WriteString("\n\n")
		}
	}
	out.WriteString("```\n\n")

	for _, recv := range []string{"App", "Window", "Headless", "UI", "Canvas", "SingleInstanceGuard"} {
		out.WriteString(fmt.Sprintf("## Методы `%s`\n\n", recv))
		if recv == "Canvas" {
			out.WriteString("Все методы `Canvas` работают строго в физических `Px`; клиппинг в `CanvasView`/`InteractiveCanvas` устанавливается фреймворком автоматически, а текст выводится через `UI.DrawText`. Методы `Resize`, `ResizeWithBuffer`, `AddDirtyRect`, `MarkFullDirty`, `ResetDirtyRects`, `DirtyBounds`, `DirtyRects` управляются фреймворком и внутри `OnDraw` обычно не вызываются:\n\n```go\n")
			out.WriteString("ui.CanvasView(func(c *anyui.Canvas, areaPx anyui.Rect) {\n")
			out.WriteString("\tc.FillRect(areaPx, ui.Theme().Card)\n")
			out.WriteString("\tc.FillCircle(float32(areaPx.X+areaPx.W/2), float32(areaPx.Y+areaPx.H/2), 20, ui.Theme().Accent)\n")
			out.WriteString("\tui.DrawText(anyui.Px(areaPx.X+8), anyui.Px(areaPx.Y+8), \"подпись\", 12, ui.Theme().TextMuted)\n")
			out.WriteString("}, anyui.Height(120))\n```\n\n")
			out.WriteString("```go\n")
			canvasHints := map[string]string{
				"BackdropBlur(":  "radius в Px, cr задаёт скругление области",
				"BlendPixel(":    "cov: покрытие 0..255 для сглаживания",
				"BlitFrom(":      "копирование области без альфа-смешивания (замена пикселей)",
				"CompositeFrom(": "наложение области с учётом альфа-канала",
				"DrawBGRA(":      "BGRA-буфер с альфой и кешем масштабирования",
				"DrawBGRAFast(":  "непрозрачный BGRA-видеопоток без альфы, кеша и аллокаций (mode задаёт целевой прямоугольник: 1:1 -> построчный copy, иначе nearest-neighbor)",
				"DrawRGBA(":      "RGBA-буфер с альфой и кешем масштабирования",
			}
			for _, sig := range idx.methods[recv] {
				hint := ""
				for needle, h := range canvasHints {
					if strings.Contains(sig, needle) {
						hint = h
						break
					}
				}
				if hint != "" {
					out.WriteString(sig + " // " + hint + "\n")
				} else {
					out.WriteString(sig + "\n")
				}
			}
			out.WriteString("```\n\n")
			continue
		}
		if recv == "SingleInstanceGuard" {
			out.WriteString("Порядок вызовов (ручной вариант): `anyui.AcquireSingleInstance` → `SetCallback` (получает `args []string` второго экземпляра) → `StartIPC` → `Close` при выходе. Альтернатива — заполнить `Config.SingleInstance` (`ID` или заранее полученный `Guard`, `OnSecondInstance`, `SkipActivate`, `Args`): тогда `App` выполняет весь цикл самостоятельно, включая `Close`.\n\n")
		}
		out.WriteString("```go\n")
		for _, sig := range idx.methods[recv] {
			out.WriteString(sig + "\n")
		}
		out.WriteString("```\n\n")
	}
}

func sortedKeys(m map[string]bool) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	sort.Strings(res)
	return res
}

var (
	reBacktick = regexp.MustCompile("`([^`]+)`")
	reIdent    = regexp.MustCompile(`^[A-Z][A-Za-z0-9_]*$`)
	reRecvSym  = regexp.MustCompile(`^(UI|App|Window|Headless|Canvas|SingleInstanceGuard)\.([A-Z][A-Za-z0-9_]*)$`)
	rePkgSym   = regexp.MustCompile(`^anyui\.([A-Z][A-Za-z0-9_]*)$`)
)

// lintOutputs проверяет сгенерированные llms.txt и llms-full.txt на:
// 1. отсутствие утечек внутренних подпакетов (ui., platform., render.);
// 2. двустороннюю консистентность матрицы опций (матрица <-> AST);
// 3. покрытие всех bool-виджетов UI в разделе «Семантика возвращаемых значений»;
// 4. корректность всех упомянутых символов в ручных секциях (symbol-lint);
// 5. наличие ссылки на llms-full.txt в llms.txt.
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

	// 2. Двусторонняя проверка матрицы опций
	seenOptions := make(map[string]bool)
	for _, row := range optionMatrixRows {
		for _, m := range reBacktick.FindAllStringSubmatch(row.options, -1) {
			tok := m[1]
			if idx.optionNames[tok] {
				seenOptions[tok] = true
				continue
			}
			// Допускаются уточнения виджета в скобках (`Checkbox`, `ScrollView` и т. д.)
			if idx.methodNames["UI"][tok] {
				continue
			}
			return fmt.Errorf("матрица опций содержит неизвестную опцию %q", tok)
		}
	}
	for optName := range idx.optionNames {
		if !seenOptions[optName] {
			return fmt.Errorf("экспортированная опция %q не упомянута в матрице опций", optName)
		}
	}

	// 3. Проверка покрытия всех bool-виджетов UI в разделе «Семантика возвращаемых значений»
	stateGetters := map[string]bool{
		"MousePressed":           true,
		"MouseDown":              true,
		"DoubleClicked":          true,
		"MiddleClicked":          true,
		"LastFrameStaticBlitted": true,
		"HasActiveOverlays":      true,
	}
	semStart := strings.Index(shortText, "## Семантика возвращаемых значений")
	if semStart < 0 {
		return fmt.Errorf("в llms.txt отсутствует раздел «Семантика возвращаемых значений»")
	}
	semBlock := shortText[semStart:]
	if nextSec := strings.Index(semBlock[3:], "\n## "); nextSec >= 0 {
		semBlock = semBlock[:nextSec+3]
	}
	for meth := range idx.uiBoolMethods {
		if stateGetters[meth] {
			continue
		}
		if !strings.Contains(semBlock, "`"+meth+"`") {
			return fmt.Errorf("метод UI.%s (возвращает bool) не упомянут в разделе «Семантика возвращаемых значений»", meth)
		}
	}

	// 4. Проверка 100% покрытия карты API в llms.txt (ни одна функция пакета и ни один метод UI/App/Window/Headless не потеряны)
	mapStart := strings.Index(shortText, "## Карта API")
	if mapStart < 0 {
		return fmt.Errorf("в llms.txt отсутствует раздел «Карта API»")
	}
	mapBlock := shortText[mapStart:]
	var missing []string
	for fnName := range idx.otherFuncNames {
		if !strings.Contains(mapBlock, "`anyui."+fnName+"`") {
			missing = append(missing, "anyui."+fnName)
		}
	}
	for meth := range idx.methodNames["UI"] {
		if !strings.Contains(mapBlock, "`"+meth+"`") && !strings.Contains(mapBlock, "`UI."+meth+"`") {
			missing = append(missing, "UI."+meth)
		}
	}
	for _, recv := range []string{"App", "Window", "Headless", "Canvas", "SingleInstanceGuard"} {
		for meth := range idx.methodNames[recv] {
			if !strings.Contains(mapBlock, "`"+recv+"."+meth+"`") {
				missing = append(missing, recv+"."+meth)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("отсутствуют в карте API llms.txt: %s", strings.Join(missing, ", "))
	}
	// Проверка отсутствия дублей символов в карте API (каждый символ — ровно в одной категории)
	seenMapTok := make(map[string]bool)
	for _, line := range strings.Split(mapBlock, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "- ") {
			continue
		}
		// Пропускаем заголовок категории до первого двоеточия (`**Категория (`UI`):**`)
		if colon := strings.Index(line, ":"); colon >= 0 {
			line = line[colon+1:]
		}
		for _, m := range reBacktick.FindAllStringSubmatch(line, -1) {
			tok := m[1]
			if seenMapTok[tok] {
				return fmt.Errorf("символ `%s` дублируется в карте API llms.txt", tok)
			}
			seenMapTok[tok] = true
		}
	}

	// 5. Проверка 100% покрытия экспортированных типов в llms-full.txt
	var missingTypes []string
	for typeName := range idx.allTypeNames {
		hasTypeDef := strings.Contains(fullText, "type "+typeName+" ")
		hasMethodsSec := strings.Contains(fullText, "## Методы `"+typeName+"`")
		if !hasTypeDef && !hasMethodsSec {
			missingTypes = append(missingTypes, typeName)
		}
	}
	if len(missingTypes) > 0 {
		sort.Strings(missingTypes)
		return fmt.Errorf("экспортированные типы не описаны в llms-full.txt: %s", strings.Join(missingTypes, ", "))
	}

	// 6. Symbol-lint по ручным секциям llms.txt (вне fenced-блоков кода)
	ignoreSymbols := map[string]bool{
		// Поля структур, упоминаемые в прозе
		"WidthDp": true, "IconRGBA": true, "Key": true, "ID": true, "Label": true,
		"FromIndex": true, "ToIndex": true, "FromKey": true, "ToKey": true,
		"SelectedID": true, "ClosedID": true, "Added": true, "MovedFrom": true, "MovedTo": true, "DetachedID": true, "AttachedID": true, "AttachedIndex": true,
		"RowClicked": true, "RowDoubleClicked": true, "RowMiddleClicked": true, "SortChanged": true, "ColumnResized": true, "Drag": true,
		"Streaming": true, "DirectInput": true, "OnDraw": true,
		"Guard": true, "Args": true, "SkipActivate": true, "OnSecondInstance": true,
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
			if pm := rePkgSym.FindStringSubmatch(tok); pm != nil {
				sym := pm[1]
				if !idx.allSymbols[sym] {
					return fmt.Errorf("llms.txt:%d: символ пакета anyui.%s не найден", lineNo+1, sym)
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
