//go:build darwin

package platform

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// OpenFileDialog открывает системный диалог выбора одного файла на macOS через osascript.
func OpenFileDialog(opts FileDialogOptions) (string, bool, error) {
	opts.AllowMultiple = false
	files, ok, err := runDarwinFileDialog(opts, false)
	if err != nil || !ok || len(files) == 0 {
		return "", false, err
	}
	return files[0], true, nil
}

// OpenFilesDialog открывает системный диалог выбора нескольких файлов на macOS.
func OpenFilesDialog(opts FileDialogOptions) ([]string, bool, error) {
	opts.AllowMultiple = true
	return runDarwinFileDialog(opts, false)
}

// SaveFileDialog открывает системный диалог сохранения файла на macOS через osascript.
func SaveFileDialog(opts FileDialogOptions) (string, bool, error) {
	files, ok, err := runDarwinFileDialog(opts, true)
	if err != nil || !ok || len(files) == 0 {
		return "", false, err
	}
	return files[0], true, nil
}

// SelectFolderDialog открывает системный диалог выбора папки на macOS через osascript.
func SelectFolderDialog(opts FolderDialogOptions) (string, bool, error) {
	var parts []string
	parts = append(parts, "choose folder")
	if opts.Title != "" {
		parts = append(parts, fmt.Sprintf(`with prompt "%s"`, escapeAppleScriptString(opts.Title)))
	}
	if opts.InitialDir != "" {
		parts = append(parts, fmt.Sprintf(`default location POSIX file "%s"`, escapeAppleScriptString(opts.InitialDir)))
	}
	script := "POSIX path of (" + strings.Join(parts, " ") + ")"

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		if isAppleScriptCancelled(err) {
			return "", false, nil
		}
		return "", false, err
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", false, nil
	}
	return path, true, nil
}

// MessageBox отображает системное диалоговое окно с кнопками на macOS через osascript.
func MessageBox(owner Window, title, text string, style MessageBoxStyle, buttons MessageBoxButtons) MessageBoxResult {
	_ = owner
	btnLabels, defaultBtn, cancelBtn := buildDarwinMessageButtons(buttons)
	iconName := darwinDialogIconName(style)

	var args []string
	args = append(args, fmt.Sprintf(`display dialog "%s"`, escapeAppleScriptString(text)))
	if title != "" {
		args = append(args, fmt.Sprintf(`with title "%s"`, escapeAppleScriptString(title)))
	}
	args = append(args, fmt.Sprintf(`buttons {%s}`, strings.Join(btnLabels, ", ")))
	if defaultBtn != "" {
		args = append(args, fmt.Sprintf(`default button "%s"`, defaultBtn))
	}
	if cancelBtn != "" {
		args = append(args, fmt.Sprintf(`cancel button "%s"`, cancelBtn))
	}
	if iconName != "" {
		args = append(args, "with icon "+iconName)
	}

	script := "button returned of (" + strings.Join(args, " ") + ")"
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		if cancelBtn != "" && isAppleScriptCancelled(err) {
			return ResCancel
		}
		return ResNone
	}
	return parseDarwinMessageResult(strings.TrimSpace(string(out)), buttons)
}

func runDarwinFileDialog(opts FileDialogOptions, save bool) ([]string, bool, error) {
	script := buildDarwinFileDialogScript(opts, save)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		if isAppleScriptCancelled(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, false, nil
	}
	if opts.AllowMultiple && !save {
		lines := strings.Split(raw, "\n")
		var res []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				res = append(res, line)
			}
		}
		return res, len(res) > 0, nil
	}
	return []string{raw}, true, nil
}

func buildDarwinFileDialogScript(opts FileDialogOptions, save bool) string {
	var parts []string
	if save {
		parts = append(parts, "choose file name")
		if opts.DefaultName != "" {
			parts = append(parts, fmt.Sprintf(`default name "%s"`, escapeAppleScriptString(opts.DefaultName)))
		}
	} else {
		parts = append(parts, "choose file")
		if exts := extractDarwinFileExtensions(opts.Filters); len(exts) > 0 {
			quoted := make([]string, len(exts))
			for i, ext := range exts {
				quoted[i] = `"` + escapeAppleScriptString(ext) + `"`
			}
			parts = append(parts, fmt.Sprintf(`of type {%s}`, strings.Join(quoted, ", ")))
		}
		if opts.AllowMultiple {
			parts = append(parts, "with multiple selections allowed")
		}
	}
	if opts.Title != "" {
		parts = append(parts, fmt.Sprintf(`with prompt "%s"`, escapeAppleScriptString(opts.Title)))
	}
	if opts.InitialDir != "" {
		parts = append(parts, fmt.Sprintf(`default location POSIX file "%s"`, escapeAppleScriptString(opts.InitialDir)))
	}

	cmd := strings.Join(parts, " ")
	if opts.AllowMultiple && !save {
		return fmt.Sprintf(
			"set chosenItems to (%s)\nset outPaths to {}\nrepeat with anItem in chosenItems\nset end of outPaths to POSIX path of anItem\nend repeat\nset AppleScript's text item delimiters to linefeed\nreturn outPaths as text",
			cmd,
		)
	}
	return "POSIX path of (" + cmd + ")"
}

// extractDarwinFileExtensions извлекает расширения без точки и звёздочки из FileFilter.
func extractDarwinFileExtensions(filters []FileFilter) []string {
	seen := make(map[string]struct{})
	var res []string
	for _, f := range filters {
		for _, token := range strings.Split(f.Pattern, ";") {
			token = strings.TrimSpace(token)
			token = strings.TrimPrefix(token, "*.")
			token = strings.TrimPrefix(token, ".")
			if token == "" || token == "*" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			res = append(res, token)
		}
	}
	return res
}

func buildDarwinMessageButtons(buttons MessageBoxButtons) (labels []string, defaultBtn, cancelBtn string) {
	switch buttons {
	case MsgOKCancel:
		return []string{`"Cancel"`, `"OK"`}, "OK", "Cancel"
	case MsgYesNo:
		return []string{`"No"`, `"Yes"`}, "Yes", ""
	case MsgYesNoCancel:
		return []string{`"Cancel"`, `"No"`, `"Yes"`}, "Yes", "Cancel"
	default:
		return []string{`"OK"`}, "OK", ""
	}
}

func parseDarwinMessageResult(btn string, buttons MessageBoxButtons) MessageBoxResult {
	switch btn {
	case "OK":
		return ResOK
	case "Cancel":
		return ResCancel
	case "Yes":
		return ResYes
	case "No":
		return ResNo
	default:
		if buttons == MsgOK {
			return ResOK
		}
		return ResNone
	}
}

func darwinDialogIconName(style MessageBoxStyle) string {
	switch style {
	case MsgWarning:
		return "caution"
	case MsgError:
		return "stop"
	case MsgInfo, MsgQuestion:
		return "note"
	default:
		return ""
	}
}

// isAppleScriptCancelled проверяет отмену диалога пользователем (код ошибки -128 в osascript).
func isAppleScriptCancelled(err error) bool {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode() == 1
	}
	return false
}

