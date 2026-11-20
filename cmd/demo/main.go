package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"runtime/debug"
	"time"

	"github.com/kirpicheff/anyui"
)

// buildVersion задаётся при сборке через -ldflags "-X main.buildVersion=...".
var buildVersion = "dev"

// resolveBuildVersion возвращает версию билда: переданную через ldflags, тег модуля или короткий хэш коммита.
func resolveBuildVersion() string {
	if buildVersion != "" && buildVersion != "dev" {
		return buildVersion
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			return bi.Main.Version
		}
		var rev string
		var dirty bool
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				dirty = (s.Value == "true")
			}
		}
		if len(rev) > 7 {
			rev = rev[:7]
		}
		if rev != "" {
			if dirty {
				return rev + "-dirty"
			}
			return rev
		}
	}
	return "dev"
}

const (
	svgStar = `<svg viewBox="0 0 24 24"><path d="M12 2L15 8.5L22 9.3L17 14.2L18.2 21L12 17.6L5.8 21L7 14.2L2 9.3L9 8.5Z"/></svg>`
	svgGear = `<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="4"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3"/></svg>`
	svgPlus = `<svg viewBox="0 0 24 24"><path d="M12 5v14M5 12h14"/></svg>`
)

type demoState struct {
	app         *anyui.App
	activeNav   int
	activeTab   int
	tabs        []anyui.TabItem
	nextTabNum  int
	statusMsg   string
	droppedInfo string

	// Раздел контролов
	chk1        bool
	chk2        bool
	chkIndet    bool
	radioIdx    int
	toggleVal   bool
	progressVal float32
	clickCount  int

	// Раздел ввода и списков
	textVal      string
	passVal      string
	digitsVal    string
	readOnlyVal  string
	notesVal     string
	selectKey    string
	comboKey     string
	listSelected int
	listItems    []string

	// Раздел оверлеев
	modalOpen    bool
	popoverOpen  bool
	floatOpen    bool
	drawerLeft   bool
	drawerRight  bool
	drawerTop    bool
	drawerBottom bool
	contextLog   string

	// Раздел графики и системы
	sampleImg  image.Image
	dialogLog  string
	childCount int

	// Раздел Фазы 5 (данные, структура, расширенный ввод)
	tableRows       []demoTableRow
	tableSelected   int
	tableSort       anyui.TableSort
	treeRoots       []*anyui.TreeNode
	treeSelected    string
	treeFavRoots    []*anyui.TreeNode
	treeFavSelected string
	splitRatio      float32
	collapsibleOpen bool
	accordionOpen   map[string]bool
	numVal          float64
	sliderVal       float32
	rangeLow        float32
	rangeHigh       float32
	pickedColor     anyui.Color
	pickedDate      time.Time
	shortcutsInit   bool

	// Раздел Фазы 6.1 (продвинутая раскладка)
	formHost  string
	formPort  string
	formToken string

	// Раздел Фазы 6.2 (работа с текстом)
	emailVal     string
	phoneMaskVal string
	dateMaskVal  string
	ipMaskVal    string
	linkClicks   int
}

func newDemoState() *demoState {
	listItems := make([]string, 500)
	for i := range listItems {
		listItems[i] = fmt.Sprintf("Элемент списка #%03d — виртуализированная строка", i+1)
	}

	okCol := anyui.NavyTheme().Success
	warnCol := anyui.NavyTheme().Warning

	return &demoState{
		tabs: []anyui.TabItem{
			{ID: "t1", Title: "Витрина", StatusColor: &okCol, Closable: false},
			{ID: "t2", Title: "Логи системы", StatusColor: &warnCol, Closable: true},
			{ID: "t3", Title: "Настройки", IconSVG: svgGear, Closable: true},
		},
		nextTabNum:      4,
		statusMsg:       "Готово к работе",
		droppedInfo:     "Перетащите файлы из проводника в окно",
		chk1:            true,
		chkIndet:        true,
		toggleVal:       true,
		progressVal:     0.65,
		textVal:         "Привет, AnyUI!",
		digitsVal:       "12345",
		readOnlyVal:     "Только чтение (можно выделить и скопировать)",
		notesVal:        "Многострочный редактор TextArea с нумерацией строк.\nПоддерживает перенос строк, скролл колесом мыши,\nнавигацию Up/Down, Undo/Redo (Ctrl+Z/Ctrl+Y),\nвыделение по словам (Ctrl+Left/Right) и ПКМ-меню.",
		emailVal:        "bad-email",
		phoneMaskVal:    "+7 (999) 123-45-67",
		dateMaskVal:     "17.11.2026",
		ipMaskVal:       "192.168.001.010",
		selectKey:       "win11",
		comboKey:        "win10",
		listItems:       listItems,
		sampleImg:       makeSampleImage(96, 96),
		contextLog:      "Нажмите ПКМ по области ниже для ContextMenu",
		dialogLog:       "Нажмите любую кнопку диалога",
		tableRows:       makeDemoTableRows(),
		tableSelected:   -1,
		tableSort:       anyui.TableSort{Column: 0, Desc: false},
		treeRoots:       makeDemoTree(),
		treeSelected:    "f_data",
		treeFavRoots: []*anyui.TreeNode{
			{ID: "fav_root", Label: "Избранное (drag сюда)", Expanded: true, Children: []*anyui.TreeNode{
				{ID: "fav_item", Label: "Быстрый доступ.go"},
			}},
		},
		splitRatio:      0.45,
		collapsibleOpen: true,
		accordionOpen:   map[string]bool{"sec1": true},
		numVal:          15.0,
		sliderVal:       62,
		formHost:        "api.example.com",
		formPort:        "8443",
		formToken:       "secret-token",
		rangeLow:        20,
		rangeHigh:       75,
		pickedColor:     anyui.NavyTheme().Accent,
		pickedDate:      time.Now(),
	}
}

func makeSampleImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 255) / w),
				G: uint8((y * 255) / h),
				B: 210,
				A: 255,
			})
		}
	}
	return img
}

func main() {
	st := newDemoState()
	buildVer := resolveBuildVersion()

	app := anyui.NewApp(anyui.Config{
		Title:           "AnyUI — Витрина возможностей (Фазы 1–7)",
		Width:           1160,
		Height:          780,
		MinWidth:        820,
		MinHeight:       560,
		Theme:           anyui.NavyTheme(),
		AcceptDropFiles: true,
		HideOnClose:     false,
		OnDropFiles: func(files []string, x, y int) {
			st.droppedInfo = fmt.Sprintf("Drop (%d файл.): %s", len(files), files[0])
			st.statusMsg = st.droppedInfo
		},
		Tray: &anyui.TrayConfig{
			Tooltip: "AnyUI Demo — кликните дважды для показа",
			OnDoubleClick: func(a *anyui.App) {
				a.Show()
			},
			OnNotificationClick: func(a *anyui.App) {
				a.Show()
			},
			Menu: []anyui.MenuItem{
				{Label: "Показать окно", OnClick: func(a *anyui.App) { a.Show() }},
				{
					Label: "Показать уведомление",
					OnClick: func(a *anyui.App) {
						_ = a.ShowNotification(anyui.Notification{
							Title:   "AnyUI Demo",
							Message: "Уведомление из системного трея!",
							Icon:    anyui.NotifyInfo,
						})
					},
				},
				{Separator: true},
				{Label: "Выход", OnClick: func(a *anyui.App) { a.Quit() }},
			},
		},
	})
	st.app = app

	navItems := []string{
		"1. Кнопки и переключатели",
		"2. Ввод и списки",
		"3. Оверлеи и контейнеры",
		"4. Графика и система",
		"5. Данные и структура",
		"6. Расширенный ввод и Toast",
		"7. Продвинутая раскладка",
		"8. Десктоп и графики (7.1)",
		"9. Слои, кеш и курсоры (7.2)",
	}

	err := app.Run(func(ui *anyui.UI) {
		th := ui.Theme()
		if !st.shortcutsInit {
			st.shortcutsInit = true
			ui.Shortcut(0x1B, anyui.Modifiers{}, func() {
				if ui.Window().IsFullscreen() {
					ui.Window().SetFullscreen(false)
					st.statusMsg = "Выход из Fullscreen (Esc)"
				}
			})
			ui.Shortcut(0x7A, anyui.Modifiers{}, func() {
				next := !ui.Window().IsFullscreen()
				ui.Window().SetFullscreen(next)
				if next {
					st.statusMsg = "Fullscreen включён (Esc или F11 для выхода)"
				} else {
					st.statusMsg = "Выход из Fullscreen (F11)"
				}
			})
		}

		// Верхняя панель с темами и TabBar
		ui.PaddingXY(16, 10, func() {
			ui.Column(8, func() {
				ui.RowCentered(10, 36, func() {
					ui.StaticLayer("demo_header_brand", func() {
						ui.Label("AnyUI Showcase", anyui.FontSize(18), anyui.Bold())
						ui.Badge("v0.7", anyui.Primary())
						ui.Badge(buildVer, anyui.Secondary())
					})
					ui.FlexSpacer(240)
					if ui.Button("th_navy", "Navy", anyui.Ghost(), anyui.Padding(10)) {
						ui.SetTheme(anyui.NavyTheme())
					}
					if ui.Button("th_dark", "Dark", anyui.Ghost(), anyui.Padding(10)) {
						ui.SetTheme(anyui.DarkTheme())
					}
					if ui.Button("th_light", "Light", anyui.Ghost(), anyui.Padding(10)) {
						ui.SetTheme(anyui.LightTheme())
					}
				})

				ev := ui.TabBar("main_tabbar", st.tabs, &st.activeTab)
				if ev.Added {
					st.tabs = append(st.tabs, anyui.TabItem{
						ID:       fmt.Sprintf("t%d", st.nextTabNum),
						Title:    fmt.Sprintf("Вкладка %d", st.nextTabNum),
						Closable: true,
					})
					st.activeTab = len(st.tabs) - 1
					st.nextTabNum++
				}
				if ev.Closed >= 0 && ev.Closed < len(st.tabs) {
					st.tabs = append(st.tabs[:ev.Closed], st.tabs[ev.Closed+1:]...)
					if st.activeTab >= len(st.tabs) {
						st.activeTab = len(st.tabs) - 1
					}
				}
				if ev.MovedFrom >= 0 && ev.MovedTo >= 0 && ev.MovedFrom < len(st.tabs) && ev.MovedTo < len(st.tabs) {
					item := st.tabs[ev.MovedFrom]
					st.tabs = append(st.tabs[:ev.MovedFrom], st.tabs[ev.MovedFrom+1:]...)
					st.tabs = append(st.tabs[:ev.MovedTo], append([]anyui.TabItem{item}, st.tabs[ev.MovedTo:]...)...)
					st.statusMsg = fmt.Sprintf("Вкладка '%s' перемещена", item.Title)
				}
				if ev.Detached >= 0 && ev.Detached < len(st.tabs) && len(st.tabs) > 1 {
					detachedTab := st.tabs[ev.Detached]
					st.tabs = append(st.tabs[:ev.Detached], st.tabs[ev.Detached+1:]...)
					if st.activeTab >= len(st.tabs) {
						st.activeTab = len(st.tabs) - 1
					}
					st.statusMsg = fmt.Sprintf("Откреплена вкладка: %s", detachedTab.Title)
					_, _ = st.app.OpenWindow(anyui.Config{
						Title:  "Откреплённая вкладка — " + detachedTab.Title,
						Width:  460,
						Height: 300,
						Theme:  ui.Theme(),
					}, func(wui *anyui.UI) {
						wui.Padding(20, func() {
							wui.Column(10, func() {
								wui.Label(detachedTab.Title, anyui.FontSize(18), anyui.Bold())
								wui.Label("Это отдельное окно, созданное при вытягивании вкладки из TabBar (app.OpenWindow).", anyui.FontSize(13), anyui.TextColor(wui.Theme().TextMuted))
								wui.Spinner("det_spin", 28)
							})
						})
					})
				}
			})
		})

		ui.Separator()

		// Основная рабочая область: слева Tabs, справа контент выбранного раздела
		ui.PaddingXY(16, 12, func() {
			area := ui.CurrentArea()
			contentH := area.H - 36
			if contentH < 200 {
				contentH = 200
			}
			ui.Row(16, anyui.Dp(contentH), func() {
				ui.Card(func() {
					ui.Column(6, func() {
						ui.StaticLayer("demo_nav_title", func() {
							ui.Label("Разделы", anyui.FontSize(12), anyui.TextColor(th.TextMuted), anyui.Bold())
						})
						ui.Tabs("nav_tabs", navItems, &st.activeNav, anyui.Vertical, anyui.Width(210))
						ui.Spacer(12)
						ui.Separator()
						ui.Spacer(6)
						ui.Label(st.droppedInfo, anyui.FontSize(11), anyui.TextColor(th.TextMuted))
					})
				}, anyui.Width(236), anyui.Padding(12), anyui.FillHeight())

				ui.Card(func() {
					switch st.activeNav {
					case 0:
						st.drawTabControls(ui)
					case 1:
						st.drawTabInputs(ui)
					case 2:
						st.drawTabOverlays(ui)
					case 3:
						st.drawTabGraphicsSystem(ui)
					case 4:
						st.drawTabData(ui)
					case 5:
						st.drawTabAdvanced(ui)
					case 6:
						st.drawTabLayout(ui)
					case 7:
						st.drawTabDesktop(ui)
					case 8:
						st.drawTabPerf(ui)
					}
				}, anyui.Padding(16), anyui.FillHeight())
			})
		})

		// Нижний статус-бар
		ui.FlexSpacer(32)
		barArea := ui.CurrentAreaPx()
		barRect := anyui.Rect{
			X: barArea.X,
			Y: barArea.Y + barArea.H - ui.Scale(32).Int(),
			W: barArea.W,
			H: ui.Scale(32).Int(),
		}
		ui.Canvas().FillRect(barRect, th.SurfaceDark)
		ui.Canvas().FillRect(anyui.Rect{X: barRect.X, Y: barRect.Y, W: barRect.W, H: 1}, th.Border)
		ui.PaddingXY(16, 0, func() {
			ui.RowCentered(20, 32, func() {
				ui.StatusDot(th.Success, st.statusMsg)
				ui.FlexSpacer(140)
				ui.Label("AnyUI Фаза 1–7 • Pure Go", anyui.FontSize(12), anyui.TextColor(th.TextMuted))
			})
		})

		// Глобальные оверлеи (Modal, FloatingPanel, Drawer)
		st.drawGlobalOverlays(ui)
	})
	if err != nil {
		log.Fatal(err)
	}
}
