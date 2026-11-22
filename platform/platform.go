package platform

import "image"

// EventType описывает тип события окна или ввода.
type EventType uint8

const (
	EventNone EventType = iota
	EventClose
	EventResize
	EventDPIChange
	EventPaint
	EventWakeUp
	EventMouseMove
	EventMouseDown
	EventMouseUp
	EventMouseWheel
	EventKeyDown
	EventKeyUp
	EventChar
	EventDropFiles
	EventCaptureLost
	EventFocusLost
	EventThemeChange
)

// MouseButton определяет кнопку мыши.
type MouseButton uint8

const (
	MouseLeft MouseButton = iota + 1
	MouseRight
	MouseMiddle
)

// CursorType задаёт системный курсор.
type CursorType uint8

const (
	CursorArrow CursorType = iota
	CursorHand
	CursorIBeam
	CursorResizeH
	CursorResizeV
	CursorResizeNWSE
	CursorResizeNESW
	CursorNotAllowed
	CursorCrosshair
	CursorMove
)

// HitTestResult определяет роль точки клиентской области в режиме Frameless.
type HitTestResult uint8

const (
	HitClient HitTestResult = iota
	HitCaption
	HitLeft
	HitRight
	HitTop
	HitBottom
	HitTopLeft
	HitTopRight
	HitBottomLeft
	HitBottomRight
)

// Modifiers хранит состояние модификаторов клавиатуры.
type Modifiers struct {
	Shift bool
	Ctrl  bool
	Alt   bool
	Win   bool
}

// Event представляет единое событие от платформенного слоя.
type Event struct {
	Type     EventType
	Width    int         // Для EventResize (в физических пикселях)
	Height   int         // Для EventResize (в физических пикселях)
	Scale    float32     // Для EventDPIChange (1.0 = 96 DPI)
	X        int         // Координата мыши X (в физических пикселях)
	Y        int         // Координата мыши Y (в физических пикселях)
	Button   MouseButton // Для EventMouseDown / EventMouseUp
	WheelX   float32     // Для EventMouseWheel (положительное — вправо)
	WheelY   float32     // Для EventMouseWheel (положительное — вверх)
	KeyCode  uint32      // Виртуальный код клавиши (VK_* на Windows)
	ScanCode uint16      // Аппаратный скан-код клавиши
	Extended bool        // Признак расширенной клавиши (bit 24 lParam на Windows)
	Char      rune       // Для EventChar (введённый символ Unicode)
	Mods      Modifiers
	Files     []string    // Для EventDropFiles (список путей перетащенных файлов)
	DarkTheme bool        // Для EventThemeChange (true, если системная тема тёмная)
}

// WindowColor задаёт RGB-цвет для оформления системной шапки и рамки окна.
type WindowColor struct {
	R, G, B uint8
}

// WindowState хранит геометрию окна для сохранения и восстановления между запусками.
type WindowState struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

// WindowConfig задаёт параметры создания окна.
type WindowConfig struct {
	Title           string
	Width           int  // Логическая ширина при Scale = 1.0
	Height          int  // Логическая высота при Scale = 1.0
	MinWidth        int
	MinHeight       int
	DarkTitle       bool         // Включить тёмный системный заголовок (если поддерживается ОС)
	TitleBarColor   *WindowColor // Цвет системной шапки окна (Windows 11+)
	TitleTextColor  *WindowColor // Цвет текста системной шапки (Windows 11+)
	BorderColor     *WindowColor // Цвет рамки окна (Windows 11+)
	InitialState    *WindowState // Восстанавливаемая позиция и размер окна
	Parent          Window       // Родительское окно (owner), если есть
	Frameless       bool         // Окно без системной рамки с поддержкой перетаскивания и Aero Snap
	AlwaysOnTop     bool         // Закрепить окно поверх всех окон ОС (TopMost)
	TitleBarHeight  int          // Логическая высота области перетаскивания (по умолчанию 36, если HitTest не задан)
	HitTest         func(x, y int) HitTestResult
	AcceptDropFiles bool   // Разрешить перетаскивание файлов из проводника (WM_DROPFILES)
	IconRGBA        []byte // Пиксели иконки окна в формате RGBA (width * height * 4)
	IconWidth       int    // Ширина иконки в пикселях
	IconHeight      int    // Высота иконки в пикселях
}

// DirtyPresenter — опциональный интерфейс окна для частичного обновления экрана (Dirty Rectangles).
type DirtyPresenter interface {
	PresentRect(pixels []byte, width, height, rx, ry, rw, rh int)
}

// BackBufferProvider — опциональный интерфейс окна для предоставления прямого пиксельного буфера
// (например, Windows DIBSection), позволяющего рендерить кадр без промежуточного копирования.
type BackBufferProvider interface {
	BackBuffer(width, height int) []byte
}

// WindowStater — опциональный интерфейс окна для сохранения и восстановления геометрии.
type WindowStater interface {
	State() WindowState
	RestoreState(state WindowState)
}

// TitleBarStyler — опциональный интерфейс окна для динамической настройки системной шапки и рамки.
type TitleBarStyler interface {
	SetDarkTitle(dark bool)
	SetTitleBarColors(caption, text, border *WindowColor)
}

// ClipboardImager — опциональный интерфейс окна для обмена изображениями через буфер обмена.
type ClipboardImager interface {
	ClipboardImage() (image.Image, error)
	SetClipboardImage(img image.Image) error
}

// WindowActivator — опциональный интерфейс окна для вывода на передний план.
type WindowActivator interface {
	BringToFront()
}

// WindowIconSetter — опциональный интерфейс окна для программной установки иконки.
type WindowIconSetter interface {
	SetIcon(rgba []byte, width, height int) error
}

// WindowTopMoster — опциональный интерфейс окна для закрепления поверх всех окон ОС (AlwaysOnTop).
type WindowTopMoster interface {
	SetAlwaysOnTop(onTop bool)
	IsAlwaysOnTop() bool
}

// Window — кроссплатформенный интерфейс нативного окна.
type Window interface {
	// Run запускает главный цикл событий и вызывает handler на каждое событие.
	// Блокирует вызывающую горутину до закрытия окна.
	Run(handler func(Event)) error

	// Open создаёт и отображает окно, привязывая его к уже запущенному циклу событий.
	Open(handler func(Event)) error

	// Present передаёт BGRA-буфер (размером Width * Height * 4) для отображения в окне.
	Present(pixels []byte, width, height int)

	// Invalidate помечает окно как требующее перерисовки.
	Invalidate()

	// WakeUp безопасно пробуждает цикл событий из любой горутины.
	WakeUp()

	// Show отображает окно, если оно было скрыто.
	Show()

	// Hide скрывает окно без его уничтожения.
	Hide()

	// Close запрашивает закрытие окна.
	Close()

	// SetTitle меняет заголовок окна.
	SetTitle(title string)

	// SetCursor меняет курсор мыши над окном.
	SetCursor(cursor CursorType)

	// Size возвращает текущий размер клиентской области в физических пикселях.
	Size() (width, height int)

	// Scale возвращает текущий коэффициент масштабирования DPI (1.0 = 96 DPI).
	Scale() float32

	// ClipboardText возвращает текст из буфера обмена.
	ClipboardText() (string, error)

	// SetClipboardText записывает текст в буфер обмена.
	SetClipboardText(text string) error

	// Minimize сворачивает окно.
	Minimize()

	// Maximize разворачивает окно на весь экран рабочей области.
	Maximize()

	// Restore восстанавливает окно из свёрнутого или развёрнутого состояния.
	Restore()

	// IsMaximized возвращает true, если окно развёрнуто.
	IsMaximized() bool

	// IsMinimized возвращает true, если окно свёрнуто.
	IsMinimized() bool

	// SetFullscreen включает или выключает полноэкранный режим.
	SetFullscreen(fullscreen bool)

	// IsFullscreen возвращает true, если окно находится в полноэкранном режиме.
	IsFullscreen() bool

	// Position возвращает позицию верхнего левого угла окна на экране (в физических пикселях).
	Position() (x, y int)

	// SetPosition перемещает окно в указанные экранные координаты (в физических пикселях).
	SetPosition(x, y int)

	// SetSize изменяет размер клиентской области окна (в логических пикселях).
	SetSize(width, height int)

	// SetOnClose задаёт обработчик попытки закрытия окна. Возврат false отменяет закрытие.
	SetOnClose(fn func() bool)

	// NativeHandle возвращает нативный дескриптор окна (HWND на Windows).
	NativeHandle() uintptr

	// SetCapture захватывает ввод мыши окном, чтобы события приходили даже за его пределами.
	SetCapture()

	// ReleaseCapture освобождает захват ввода мыши.
	ReleaseCapture()

	// DoubleClickTimeMs возвращает системное время двойного клика в миллисекундах.
	DoubleClickTimeMs() int
}

// TrayMenuItem описывает пункт контекстного меню системного трея.
type TrayMenuItem struct {
	Label     string
	Separator bool
	Checked   bool
	Disabled  bool
	OnClick   func()
}

// NotificationIcon задаёт иконку нативного системного уведомления.
type NotificationIcon uint8

const (
	NotifyNone NotificationIcon = iota
	NotifyInfo
	NotifyWarning
	NotifyError
)

// TrayNotification описывает системное уведомление (Balloon / Toast) из области уведомлений.
type TrayNotification struct {
	Title   string
	Message string
	Icon    NotificationIcon
	Silent  bool
}

// TrayConfig задаёт параметры иконки в системном трее.
type TrayConfig struct {
	Tooltip             string
	IconRGBA            []byte
	IconWidth           int
	IconHeight          int
	Menu                []TrayMenuItem
	OnClick             func()
	OnDoubleClick       func()
	OnRightClick        func()
	OnNotificationClick func()
}

// Tray — кроссплатформенный интерфейс иконки в системном трее.
type Tray interface {
	SetTooltip(tooltip string) error
	SetIconRGBA(rgba []byte, width, height int) error
	SetMenu(items []TrayMenuItem)
	ShowNotification(n TrayNotification) error
	Close() error
}

// FileFilter задаёт фильтр расширений для системных файловых диалогов.
type FileFilter struct {
	Name    string // Например: "Изображения (*.png;*.jpg)"
	Pattern string // Например: "*.png;*.jpg"
}

// FileDialogOptions задаёт параметры диалога выбора или сохранения файла.
type FileDialogOptions struct {
	Title         string
	InitialDir    string
	DefaultName   string
	Filters       []FileFilter
	AllowMultiple bool
	Owner         Window
}

// FolderDialogOptions задаёт параметры диалога выбора папки.
type FolderDialogOptions struct {
	Title      string
	InitialDir string
	Owner      Window
}

// MessageBoxStyle задаёт иконку системного сообщения.
type MessageBoxStyle uint8

const (
	MsgInfo MessageBoxStyle = iota
	MsgWarning
	MsgError
	MsgQuestion
)

// MessageBoxButtons задаёт набор кнопок системного сообщения.
type MessageBoxButtons uint8

const (
	MsgOK MessageBoxButtons = iota
	MsgOKCancel
	MsgYesNo
	MsgYesNoCancel
)

// MessageBoxResult представляет нажатую пользователем кнопку в MessageBox.
type MessageBoxResult uint8

const (
	ResNone MessageBoxResult = iota
	ResOK
	ResCancel
	ResYes
	ResNo
)

