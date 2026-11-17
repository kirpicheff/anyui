package anyui

import (
	"errors"
	"os"
	"sync"

	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/render"
	"git.gtw.su/kirpich/anyui/ui"
)

// HitTestResult определяет роль точки клиентской области в режиме Frameless.
type HitTestResult = platform.HitTestResult

const (
	HitClient      = platform.HitClient
	HitCaption     = platform.HitCaption
	HitLeft        = platform.HitLeft
	HitRight       = platform.HitRight
	HitTop         = platform.HitTop
	HitBottom      = platform.HitBottom
	HitTopLeft     = platform.HitTopLeft
	HitTopRight    = platform.HitTopRight
	HitBottomLeft  = platform.HitBottomLeft
	HitBottomRight = platform.HitBottomRight
)

// MenuItem описывает пункт меню системного трея.
type MenuItem struct {
	Label     string
	Separator bool
	Checked   bool
	Disabled  bool
	OnClick   func(app *App)
}

// TrayConfig задаёт настройки иконки в системном трее.
type TrayConfig struct {
	Tooltip             string
	IconRGBA            []byte
	IconWidth           int
	IconHeight          int
	Menu                []MenuItem
	OnClick             func(app *App)
	OnDoubleClick       func(app *App)
	OnRightClick        func(app *App)
	OnNotificationClick func(app *App)
}

// Config задаёт параметры создания приложения или окна.
type Config struct {
	Title               string
	Width               int
	Height              int
	MinWidth            int
	MinHeight           int
	DarkTitle           bool
	TitleBarColor       *Color
	TitleTextColor      *Color
	BorderColor         *Color
	InitialState        *WindowState
	FollowSystemTheme   bool
	OnSystemThemeChange func(app *App, dark bool)
	Theme               Theme
	Strings             *Strings
	SingleInstance      *SingleInstanceConfig
	Tray                *TrayConfig
	Parent              *Window
	Frameless           bool
	TitleBarHeight      int
	HitTest             func(x, y int) HitTestResult
	AcceptDropFiles     bool
	HideOnClose         bool
	OnClose             func(w *Window) bool
	OnDropFiles         func(files []string, x, y int)
}

// App управляет единым циклом событий ОС, реестром окон и потокобезопасной очередью Dispatch.
type App struct {
	cfg         Config
	font        *render.FontManager
	mu          sync.Mutex
	windows     []*Window
	mainWin     *Window
	tray        platform.Tray
	singleGuard *SingleInstanceGuard
	quitting    bool

	dispatchMu    sync.Mutex
	dispatchQueue []func()
}

// NewApp создаёт новый экземпляр приложения anyui с указанной конфигурацией.
func NewApp(cfg Config) *App {
	if cfg.Width <= 0 {
		cfg.Width = 960
	}
	if cfg.Height <= 0 {
		cfg.Height = 600
	}
	if cfg.Theme.Bg.A == 0 && cfg.Theme.TextPrimary.A == 0 {
		if cfg.FollowSystemTheme {
			cfg.Theme = SystemTheme()
		} else {
			cfg.Theme = DeepNavyTheme()
		}
	}
	if !cfg.DarkTitle {
		// По умолчанию включаем тёмный заголовок для тёмных тем
		cfg.DarkTitle = isThemeDark(cfg.Theme)
	}
	return &App{
		cfg: cfg,
	}
}

// Run создаёт главное окно, запускает цикл событий ОС и блокирует вызывающую горутину до выхода.
func (a *App) Run(drawFn func(ui *UI)) error {
	if drawFn == nil {
		return errors.New("функция отрисовки не задана")
	}

	if err := a.setupSingleInstance(); err != nil {
		return err
	}
	if a.singleGuard != nil {
		defer func() {
			_ = a.singleGuard.Close()
			a.singleGuard = nil
		}()
	}

	fm, err := render.LoadDefaultSystemFont()
	if err != nil {
		return err
	}
	a.font = fm

	return platform.RunEventLoop(func() error {
		if a.singleGuard != nil {
			if err := a.singleGuard.StartIPC(); err != nil {
				return err
			}
		}
		if a.cfg.Tray != nil {
			if err := a.initTray(a.cfg.Tray); err != nil {
				return err
			}
		}
		win, err := a.createWindow(a.cfg, drawFn)
		if err != nil {
			if a.tray != nil {
				_ = a.tray.Close()
				a.tray = nil
			}
			return err
		}
		a.mu.Lock()
		a.mainWin = win
		a.mu.Unlock()
		return win.platWin.Open(win.handleEvent)
	})
}

// OpenWindow открывает дополнительное независимое окно в рамках уже запущенного App.
func (a *App) OpenWindow(cfg Config, drawFn func(ui *UI)) (*Window, error) {
	if drawFn == nil {
		return nil, errors.New("функция отрисовки не задана")
	}
	if cfg.Theme.Bg.A == 0 && cfg.Theme.TextPrimary.A == 0 {
		cfg.Theme = a.cfg.Theme
	}
	win, err := a.createWindow(cfg, drawFn)
	if err != nil {
		return nil, err
	}
	if err := win.platWin.Open(win.handleEvent); err != nil {
		a.removeWindow(win)
		return nil, err
	}
	return win, nil
}

func (a *App) createWindow(cfg Config, drawFn func(*UI)) (*Window, error) {
	var parentPlat platform.Window
	if cfg.Parent != nil {
		parentPlat = cfg.Parent.platWin
	}

	platWin, err := platform.NewWindow(platform.WindowConfig{
		Title:           cfg.Title,
		Width:           cfg.Width,
		Height:          cfg.Height,
		MinWidth:        cfg.MinWidth,
		MinHeight:       cfg.MinHeight,
		DarkTitle:       cfg.DarkTitle,
		TitleBarColor:   toPlatformWindowColor(cfg.TitleBarColor),
		TitleTextColor:  toPlatformWindowColor(cfg.TitleTextColor),
		BorderColor:     toPlatformWindowColor(cfg.BorderColor),
		InitialState:    cfg.InitialState,
		Parent:          parentPlat,
		Frameless:       cfg.Frameless,
		TitleBarHeight:  cfg.TitleBarHeight,
		HitTest:         cfg.HitTest,
		AcceptDropFiles: cfg.AcceptDropFiles,
	})
	if err != nil {
		return nil, err
	}

	ctx := ui.NewContext(platWin, a.font)
	if cfg.Theme.Bg.A != 0 || cfg.Theme.TextPrimary.A != 0 {
		ctx.Theme = cfg.Theme
	} else {
		ctx.Theme = a.cfg.Theme
	}
	if cfg.Strings != nil {
		ctx.Strings = *cfg.Strings
	} else if a.cfg.Strings != nil {
		ctx.Strings = *a.cfg.Strings
	}

	w := &Window{
		app:         a,
		platWin:     platWin,
		ctx:         ctx,
		drawFn:      drawFn,
		hideOnClose: cfg.HideOnClose,
		onClose:     cfg.OnClose,
		onDropFiles: cfg.OnDropFiles,
	}
	w.ui = &UI{
		ctx: ctx,
		win: w,
		app: a,
	}

	platWin.SetOnClose(w.handleCloseRequest)

	a.mu.Lock()
	a.windows = append(a.windows, w)
	a.mu.Unlock()

	return w, nil
}

// Dispatch ставит функцию fn в очередь на выполнение в главном потоке UI
// и безопасно пробуждает цикл событий.
func (a *App) Dispatch(fn func()) {
	if fn == nil {
		return
	}
	a.dispatchMu.Lock()
	a.dispatchQueue = append(a.dispatchQueue, fn)
	a.dispatchMu.Unlock()

	a.mu.Lock()
	var target *Window
	if len(a.windows) > 0 {
		target = a.windows[0]
	}
	a.mu.Unlock()

	if target != nil && target.platWin != nil {
		target.platWin.WakeUp()
	}
}

func (a *App) flushDispatch() bool {
	a.dispatchMu.Lock()
	if len(a.dispatchQueue) == 0 {
		a.dispatchMu.Unlock()
		return false
	}
	queue := a.dispatchQueue
	a.dispatchQueue = nil
	a.dispatchMu.Unlock()

	for _, fn := range queue {
		fn()
	}
	return true
}

func (a *App) invalidateAllExcept(skip *Window) {
	a.mu.Lock()
	wins := make([]*Window, len(a.windows))
	copy(wins, a.windows)
	a.mu.Unlock()

	for _, w := range wins {
		if w != skip && w.platWin != nil {
			w.platWin.WakeUp()
		}
	}
}

func (a *App) removeWindow(target *Window) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, w := range a.windows {
		if w == target {
			a.windows = append(a.windows[:i], a.windows[i+1:]...)
			break
		}
	}
	if a.mainWin == target {
		a.mainWin = nil
	}
}

// MainWindow возвращает главное окно приложения.
func (a *App) MainWindow() *Window {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.mainWin
}

// Show отображает главное окно приложения.
func (a *App) Show() {
	if w := a.MainWindow(); w != nil {
		w.Show()
	}
}

// Quit закрывает все окна, удаляет иконку трея и завершает приложение.
func (a *App) Quit() {
	a.mu.Lock()
	a.quitting = true
	tray := a.tray
	a.tray = nil
	wins := make([]*Window, len(a.windows))
	copy(wins, a.windows)
	a.mu.Unlock()

	if tray != nil {
		_ = tray.Close()
	}
	for _, w := range wins {
		w.Close()
	}
}

func (a *App) isQuitting() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.quitting
}

func (a *App) isPrimaryEventWindow(w *Window) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mainWin != nil {
		return a.mainWin == w
	}
	return len(a.windows) > 0 && a.windows[0] == w
}

func (a *App) initTray(cfg *TrayConfig) error {
	pCfg := platform.TrayConfig{
		Tooltip:    cfg.Tooltip,
		IconRGBA:   cfg.IconRGBA,
		IconWidth:  cfg.IconWidth,
		IconHeight: cfg.IconHeight,
		Menu:       a.convertTrayMenu(cfg.Menu),
	}
	if cfg.OnClick != nil {
		fn := cfg.OnClick
		pCfg.OnClick = func() { fn(a) }
	}
	if cfg.OnDoubleClick != nil {
		fn := cfg.OnDoubleClick
		pCfg.OnDoubleClick = func() { fn(a) }
	}
	if cfg.OnRightClick != nil {
		fn := cfg.OnRightClick
		pCfg.OnRightClick = func() { fn(a) }
	}
	if cfg.OnNotificationClick != nil {
		fn := cfg.OnNotificationClick
		pCfg.OnNotificationClick = func() { fn(a) }
	}
	tray, err := platform.NewTray(pCfg)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.tray = tray
	a.mu.Unlock()
	return nil
}

func (a *App) convertTrayMenu(items []MenuItem) []platform.TrayMenuItem {
	if len(items) == 0 {
		return nil
	}
	res := make([]platform.TrayMenuItem, len(items))
	for i, item := range items {
		var cb func()
		if item.OnClick != nil {
			fn := item.OnClick
			cb = func() { fn(a) }
		}
		res[i] = platform.TrayMenuItem{
			Label:     item.Label,
			Separator: item.Separator,
			Checked:   item.Checked,
			Disabled:  item.Disabled,
			OnClick:   cb,
		}
	}
	return res
}

// SetTrayTooltip обновляет всплывающую подсказку иконки в трее.
func (a *App) SetTrayTooltip(tooltip string) error {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray == nil {
		return nil
	}
	return tray.SetTooltip(tooltip)
}

// SetTrayIcon обновляет изображение иконки в трее из буфера RGBA.
func (a *App) SetTrayIcon(rgba []byte, width, height int) error {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray == nil {
		return nil
	}
	return tray.SetIconRGBA(rgba, width, height)
}

// SetTrayMenu обновляет контекстное меню иконки в трее.
func (a *App) SetTrayMenu(items []MenuItem) {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray == nil {
		return
	}
	tray.SetMenu(a.convertTrayMenu(items))
}

// ShowNotification отображает нативное системное уведомление ОС от иконки трея.
func (a *App) ShowNotification(n Notification) error {
	a.mu.Lock()
	tray := a.tray
	a.mu.Unlock()
	if tray == nil {
		return errors.New("системный трей не настроен")
	}
	return tray.ShowNotification(n)
}

// SetStrings устанавливает локализацию встроенных строк для всех окон приложения.
func (a *App) SetStrings(s Strings) {
	a.mu.Lock()
	sCopy := s
	a.cfg.Strings = &sCopy
	wins := make([]*Window, len(a.windows))
	copy(wins, a.windows)
	a.mu.Unlock()

	for _, w := range wins {
		w.SetStrings(s)
	}
}

func (a *App) setupSingleInstance() error {
	si := a.cfg.SingleInstance
	if si == nil {
		return nil
	}
	guard := si.Guard
	if guard == nil {
		if si.ID == "" {
			return nil
		}
		args := si.Args
		if args == nil && len(os.Args) > 1 {
			args = os.Args[1:]
		}
		g, already, err := platform.AcquireSingleInstance(si.ID, args, nil)
		if err != nil {
			return err
		}
		if already {
			return ErrAlreadyRunning
		}
		guard = g
	}
	guard.SetCallback(func(args []string) {
		if !si.SkipActivate {
			if mw := a.MainWindow(); mw != nil {
				mw.BringToFront()
			}
		}
		if si.OnSecondInstance != nil {
			si.OnSecondInstance(a, args)
		}
	})
	a.singleGuard = guard
	return nil
}

func (a *App) handleSystemThemeChange(dark bool) {
	if a.cfg.FollowSystemTheme {
		nextTheme := LightTheme()
		if dark {
			nextTheme = DarkTheme()
		}
		a.mu.Lock()
		a.cfg.Theme = nextTheme
		a.cfg.DarkTitle = dark
		wins := make([]*Window, len(a.windows))
		copy(wins, a.windows)
		a.mu.Unlock()

		for _, w := range wins {
			w.SetTheme(nextTheme)
		}
	}
	if a.cfg.OnSystemThemeChange != nil {
		a.cfg.OnSystemThemeChange(a, dark)
	}
}

