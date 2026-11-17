package anyui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/ui"
)

// ErrAlreadyRunning возвращается из App.Run, если включён режим SingleInstance
// и другой экземпляр приложения уже запущен.
var ErrAlreadyRunning = errors.New("приложение уже запущено")

// NotificationIcon задаёт иконку нативного системного уведомления.
type NotificationIcon = platform.NotificationIcon

const (
	NotifyNone    = platform.NotifyNone
	NotifyInfo    = platform.NotifyInfo
	NotifyWarning = platform.NotifyWarning
	NotifyError   = platform.NotifyError
)

// Notification описывает нативное системное уведомление (Balloon / Toast).
type Notification = platform.TrayNotification

// WindowState хранит позицию, размер и состояние разворачивания окна.
type WindowState = platform.WindowState

// SingleInstanceGuard удерживает блокировку единственного экземпляра приложения.
type SingleInstanceGuard = platform.SingleInstanceGuard

// SingleInstanceConfig задаёт параметры защиты от запуска второй копии приложения.
type SingleInstanceConfig struct {
	ID               string
	Guard            *SingleInstanceGuard
	Args             []string
	SkipActivate     bool
	OnSecondInstance func(app *App, args []string)
}

// Strings содержит локализуемые встроенные строки стандартных виджетов (i18n).
type Strings = ui.Strings

// RussianStrings возвращает набор встроенных строк на русском языке (по умолчанию).
func RussianStrings() Strings {
	return ui.RussianStrings()
}

// EnglishStrings возвращает набор встроенных строк на английском языке.
func EnglishStrings() Strings {
	return ui.EnglishStrings()
}

// IsSystemDarkTheme возвращает true, если в операционной системе активна тёмная тема.
func IsSystemDarkTheme() bool {
	return platform.IsSystemDarkTheme()
}

// SystemTheme возвращает подходящую предустановленную тему (DarkTheme или LightTheme)
// в соответствии с текущей темой операционной системы.
func SystemTheme() Theme {
	if platform.IsSystemDarkTheme() {
		return DarkTheme()
	}
	return LightTheme()
}

// OpenURL безопасно открывает веб-ссылку (http, https, mailto) в приложении по умолчанию.
func OpenURL(rawURL string) error {
	return platform.OpenURL(rawURL)
}

// RevealInFileManager открывает файловый менеджер ОС и выделяет указанный файл или папку.
func RevealInFileManager(path string) error {
	return platform.RevealInFileManager(path)
}

// AcquireSingleInstance проверяет, не запущена ли уже копия приложения с данным appID.
// Если копия уже работает, передаёт ей args и возвращает (nil, true, nil).
func AcquireSingleInstance(appID string, args []string) (*SingleInstanceGuard, bool, error) {
	return platform.AcquireSingleInstance(appID, args, nil)
}

// LoadWindowState загружает геометрию окна из JSON-файла.
func LoadWindowState(path string) (WindowState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return WindowState{}, err
	}
	var st WindowState
	if err := json.Unmarshal(data, &st); err != nil {
		return WindowState{}, err
	}
	return st, nil
}

// SaveWindowState сохраняет геометрию окна в JSON-файл.
func SaveWindowState(path string, state WindowState) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func toPlatformWindowColor(c *Color) *platform.WindowColor {
	if c == nil {
		return nil
	}
	return &platform.WindowColor{R: c.R, G: c.G, B: c.B}
}

func isThemeDark(t Theme) bool {
	return t.Bg.R < 128 && t.Bg.G < 128 && t.Bg.B < 128
}

