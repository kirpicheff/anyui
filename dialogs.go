package anyui

import "anyui/platform"

// FileFilter задаёт фильтр расширений для системных файловых диалогов.
type FileFilter = platform.FileFilter

// MessageBoxStyle задаёт иконку системного сообщения.
type MessageBoxStyle = platform.MessageBoxStyle

const (
	MsgInfo     = platform.MsgInfo
	MsgWarning  = platform.MsgWarning
	MsgError    = platform.MsgError
	MsgQuestion = platform.MsgQuestion
)

// MessageBoxButtons задаёт набор кнопок системного сообщения.
type MessageBoxButtons = platform.MessageBoxButtons

const (
	MsgOK          = platform.MsgOK
	MsgOKCancel    = platform.MsgOKCancel
	MsgYesNo       = platform.MsgYesNo
	MsgYesNoCancel = platform.MsgYesNoCancel
)

// MessageBoxResult представляет нажатую пользователем кнопку в MessageBox.
type MessageBoxResult = platform.MessageBoxResult

const (
	ResNone   = platform.ResNone
	ResOK     = platform.ResOK
	ResCancel = platform.ResCancel
	ResYes    = platform.ResYes
	ResNo     = platform.ResNo
)

// FileDialogOptions задаёт параметры диалога выбора или сохранения файла.
type FileDialogOptions struct {
	Title       string
	InitialDir  string
	DefaultName string
	Filters     []FileFilter
	Owner       *Window
}

// FolderDialogOptions задаёт параметры диалога выбора каталога.
type FolderDialogOptions struct {
	Title      string
	InitialDir string
	Owner      *Window
}

func toPlatformOwner(w *Window) platform.Window {
	if w == nil {
		return nil
	}
	return w.platWin
}

// OpenFileDialog открывает системный диалог выбора одного файла.
func OpenFileDialog(opts FileDialogOptions) (string, bool, error) {
	return platform.OpenFileDialog(platform.FileDialogOptions{
		Title:       opts.Title,
		InitialDir:  opts.InitialDir,
		DefaultName: opts.DefaultName,
		Filters:     opts.Filters,
		Owner:       toPlatformOwner(opts.Owner),
	})
}

// OpenFilesDialog открывает системный диалог выбора одного или нескольких файлов.
func OpenFilesDialog(opts FileDialogOptions) ([]string, bool, error) {
	return platform.OpenFilesDialog(platform.FileDialogOptions{
		Title:         opts.Title,
		InitialDir:    opts.InitialDir,
		DefaultName:   opts.DefaultName,
		Filters:       opts.Filters,
		AllowMultiple: true,
		Owner:         toPlatformOwner(opts.Owner),
	})
}

// SaveFileDialog открывает системный диалог сохранения файла.
func SaveFileDialog(opts FileDialogOptions) (string, bool, error) {
	return platform.SaveFileDialog(platform.FileDialogOptions{
		Title:       opts.Title,
		InitialDir:  opts.InitialDir,
		DefaultName: opts.DefaultName,
		Filters:     opts.Filters,
		Owner:       toPlatformOwner(opts.Owner),
	})
}

// SelectFolderDialog открывает системный диалог выбора папки.
func SelectFolderDialog(opts FolderDialogOptions) (string, bool, error) {
	return platform.SelectFolderDialog(platform.FolderDialogOptions{
		Title:      opts.Title,
		InitialDir: opts.InitialDir,
		Owner:      toPlatformOwner(opts.Owner),
	})
}

// MessageBox отображает системное диалоговое окно с сообщением.
func MessageBox(owner *Window, title, text string, style MessageBoxStyle, buttons MessageBoxButtons) MessageBoxResult {
	return platform.MessageBox(toPlatformOwner(owner), title, text, style, buttons)
}

