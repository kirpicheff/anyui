//go:build linux

package platform

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type linuxDialogBackend int

const (
	dialogBackendNone linuxDialogBackend = iota
	dialogBackendZenity
	dialogBackendYad
	dialogBackendKdialog
)

func detectLinuxDialogBackend() linuxDialogBackend {
	if _, err := exec.LookPath("zenity"); err == nil {
		return dialogBackendZenity
	}
	if _, err := exec.LookPath("yad"); err == nil {
		return dialogBackendYad
	}
	if _, err := exec.LookPath("kdialog"); err == nil {
		return dialogBackendKdialog
	}
	return dialogBackendNone
}

// OpenFileDialog открывает системный диалог выбора одного файла.
func OpenFileDialog(opts FileDialogOptions) (string, bool, error) {
	opts.AllowMultiple = false
	files, ok, err := runLinuxFileDialog(opts, false)
	if err != nil || !ok || len(files) == 0 {
		return "", ok, err
	}
	return files[0], true, nil
}

// OpenFilesDialog открывает системный диалог выбора одного или нескольких файлов.
func OpenFilesDialog(opts FileDialogOptions) ([]string, bool, error) {
	opts.AllowMultiple = true
	return runLinuxFileDialog(opts, false)
}

// SaveFileDialog открывает системный диалог сохранения файла.
func SaveFileDialog(opts FileDialogOptions) (string, bool, error) {
	files, ok, err := runLinuxFileDialog(opts, true)
	if err != nil || !ok || len(files) == 0 {
		return "", ok, err
	}
	return files[0], true, nil
}

// SelectFolderDialog открывает системный диалог выбора каталога.
func SelectFolderDialog(opts FolderDialogOptions) (string, bool, error) {
	backend := detectLinuxDialogBackend()
	if backend == dialogBackendNone {
		return "", false, errors.New("не найдена утилита системных диалогов (zenity, yad или kdialog)")
	}

	var args []string
	var bin string
	switch backend {
	case dialogBackendZenity, dialogBackendYad:
		if backend == dialogBackendZenity {
			bin = "zenity"
		} else {
			bin = "yad"
		}
		args = append(args, "--file-selection", "--directory")
		if opts.Title != "" {
			args = append(args, "--title="+opts.Title)
		}
		if opts.InitialDir != "" {
			args = append(args, "--filename="+ensureTrailingSlash(opts.InitialDir))
		}
	case dialogBackendKdialog:
		bin = "kdialog"
		args = append(args, "--getexistingdirectory")
		if opts.InitialDir != "" {
			args = append(args, opts.InitialDir)
		} else {
			args = append(args, ".")
		}
		if opts.Title != "" {
			args = append(args, "--title", opts.Title)
		}
		if opts.Owner != nil && opts.Owner.NativeHandle() != 0 {
			args = append(args, "--attach", strconv.FormatUint(uint64(opts.Owner.NativeHandle()), 10))
		}
	}

	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		if isDialogCancelled(err) {
			return "", false, nil
		}
		return "", false, err
	}
	res := strings.TrimSpace(string(out))
	if res == "" {
		return "", false, nil
	}
	return res, true, nil
}

// MessageBox отображает системное диалоговое окно с сообщением и кнопками.
func MessageBox(owner Window, title, text string, style MessageBoxStyle, buttons MessageBoxButtons) MessageBoxResult {
	backend := detectLinuxDialogBackend()
	if backend == dialogBackendNone {
		return ResNone
	}

	if backend == dialogBackendKdialog {
		return runKdialogMessageBox(owner, title, text, style, buttons)
	}

	bin := "zenity"
	if backend == dialogBackendYad {
		bin = "yad"
	}

	var modeFlag string
	switch buttons {
	case MsgOK:
		switch style {
		case MsgWarning:
			modeFlag = "--warning"
		case MsgError:
			modeFlag = "--error"
		default:
			modeFlag = "--info"
		}
	case MsgOKCancel, MsgYesNo, MsgYesNoCancel:
		modeFlag = "--question"
	}

	args := []string{modeFlag, "--text=" + text}
	if title != "" {
		args = append(args, "--title="+title)
	}
	if buttons == MsgOKCancel {
		args = append(args, "--ok-label=OK", "--cancel-label=Отмена")
	}

	err := exec.Command(bin, args...).Run()
	if err == nil {
		if buttons == MsgYesNo || buttons == MsgYesNoCancel {
			return ResYes
		}
		return ResOK
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		if buttons == MsgYesNo {
			return ResNo
		}
		return ResCancel
	}
	return ResNone
}

func runKdialogMessageBox(owner Window, title, text string, style MessageBoxStyle, buttons MessageBoxButtons) MessageBoxResult {
	var modeFlag string
	switch buttons {
	case MsgOK:
		switch style {
		case MsgWarning:
			modeFlag = "--sorry"
		case MsgError:
			modeFlag = "--error"
		default:
			modeFlag = "--msgbox"
		}
	case MsgOKCancel:
		modeFlag = "--warningcontinuecancel"
	case MsgYesNo:
		modeFlag = "--yesno"
	case MsgYesNoCancel:
		modeFlag = "--yesnocancel"
	}

	args := []string{modeFlag, text}
	if title != "" {
		args = append(args, "--title", title)
	}
	if owner != nil && owner.NativeHandle() != 0 {
		args = append(args, "--attach", strconv.FormatUint(uint64(owner.NativeHandle()), 10))
	}

	err := exec.Command("kdialog", args...).Run()
	if err == nil {
		if buttons == MsgYesNo || buttons == MsgYesNoCancel {
			return ResYes
		}
		return ResOK
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		if buttons == MsgYesNoCancel {
			if code == 1 {
				return ResNo
			}
			return ResCancel
		}
		if buttons == MsgYesNo && code == 1 {
			return ResNo
		}
		if code == 1 || code == 2 {
			return ResCancel
		}
	}
	return ResNone
}

func runLinuxFileDialog(opts FileDialogOptions, save bool) ([]string, bool, error) {
	backend := detectLinuxDialogBackend()
	if backend == dialogBackendNone {
		return nil, false, errors.New("не найдена утилита системных диалогов (zenity, yad или kdialog)")
	}

	var bin string
	var args []string
	const sep = "\n"

	switch backend {
	case dialogBackendZenity, dialogBackendYad:
		if backend == dialogBackendZenity {
			bin = "zenity"
		} else {
			bin = "yad"
		}
		args = append(args, "--file-selection")
		if save {
			args = append(args, "--save", "--confirm-overwrite")
		} else if opts.AllowMultiple {
			args = append(args, "--multiple", "--separator="+sep)
		}
		if opts.Title != "" {
			args = append(args, "--title="+opts.Title)
		}
		initial := buildInitialPath(opts.InitialDir, opts.DefaultName)
		if initial != "" {
			args = append(args, "--filename="+initial)
		}
		for _, f := range opts.Filters {
			pat := strings.ReplaceAll(f.Pattern, ";", " ")
			if f.Name != "" {
				args = append(args, "--file-filter="+f.Name+" | "+pat)
			} else {
				args = append(args, "--file-filter="+pat)
			}
		}

	case dialogBackendKdialog:
		bin = "kdialog"
		if save {
			args = append(args, "--getsavefilename")
		} else {
			args = append(args, "--getopenfilename")
		}
		initial := buildInitialPath(opts.InitialDir, opts.DefaultName)
		if initial != "" {
			args = append(args, initial)
		} else {
			args = append(args, ".")
		}
		if filterStr := buildKdialogFilter(opts.Filters); filterStr != "" {
			args = append(args, filterStr)
		}
		if !save && opts.AllowMultiple {
			args = append(args, "--multiple", "--separate-output")
		}
		if opts.Title != "" {
			args = append(args, "--title", opts.Title)
		}
		if opts.Owner != nil && opts.Owner.NativeHandle() != 0 {
			args = append(args, "--attach", strconv.FormatUint(uint64(opts.Owner.NativeHandle()), 10))
		}
	}

	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		if isDialogCancelled(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, false, nil
	}
	lines := strings.Split(raw, "\n")
	res := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			res = append(res, line)
		}
	}
	if len(res) == 0 {
		return nil, false, nil
	}
	return res, true, nil
}

func buildInitialPath(dir, name string) string {
	if dir != "" && name != "" {
		return filepath.Join(dir, name)
	}
	if dir != "" {
		return ensureTrailingSlash(dir)
	}
	return name
}

func ensureTrailingSlash(dir string) string {
	if dir == "" || strings.HasSuffix(dir, "/") {
		return dir
	}
	return dir + "/"
}

func buildKdialogFilter(filters []FileFilter) string {
	if len(filters) == 0 {
		return ""
	}
	parts := make([]string, 0, len(filters))
	for _, f := range filters {
		pat := strings.ReplaceAll(f.Pattern, ";", " ")
		if f.Name != "" {
			parts = append(parts, f.Name+" ("+pat+")")
		} else {
			parts = append(parts, pat)
		}
	}
	return strings.Join(parts, "\n")
}

func isDialogCancelled(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 1
	}
	return false
}

