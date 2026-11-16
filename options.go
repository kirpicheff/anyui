package anyui

import "anyui/ui"

// Option настраивает стиль, размеры или поведение отдельного виджета.
type Option = ui.Option

// Bg переопределяет цвет фона виджета.
func Bg(c Color) Option { return ui.Bg(c) }

// TextColor переопределяет цвет текста виджета.
func TextColor(c Color) Option { return ui.TextColor(c) }

// BorderColor переопределяет цвет рамки виджета.
func BorderColor(c Color) Option { return ui.BorderColor(c) }

// Radius переопределяет радиус скругления углов в логических пикселях.
func Radius(dp float32) Option { return ui.Radius(dp) }

// Width задаёт фиксированную ширину виджета в логических пикселях.
func Width(dp int) Option { return ui.Width(dp) }

// Height задаёт фиксированную высоту виджета в логических пикселях.
func Height(dp int) Option { return ui.Height(dp) }

// FontSize переопределяет размер шрифта виджета в логических пикселях.
func FontSize(dp int) Option { return ui.FontSize(dp) }

// Padding задаёт внутренний отступ виджета в логических пикселях.
func Padding(dp int) Option { return ui.Padding(dp) }

// Icon задаёт SVG-иконку виджета.
func Icon(svg string) Option { return ui.Icon(svg) }

// Disabled переводит виджет в неактивное состояние.
func Disabled(v bool) Option { return ui.Disabled(v) }

// Primary задаёт акцентный стиль виджета.
func Primary() Option { return ui.Primary() }

// Secondary задаёт второстепенный стиль виджета.
func Secondary() Option { return ui.Secondary() }

// Danger задаёт стиль предупреждения об опасном действии.
func Danger() Option { return ui.Danger() }

// Ghost задаёт прозрачный стиль без заливки фона в покое.
func Ghost() Option { return ui.Ghost() }

// Bold выбирает полужирное начертание шрифта для текста виджета.
func Bold() Option { return ui.Bold() }

// Mono выбирает моноширинное начертание шрифта для текста виджета.
func Mono() Option { return ui.Mono() }

// IconRight задаёт SVG-иконку справа.
func IconRight(svg string) Option { return ui.IconRight(svg) }

// Password включает режим пароля для поля ввода.
func Password() Option { return ui.Password() }

// ReadOnly запрещает редактирование текстового поля.
func ReadOnly(v bool) Option { return ui.ReadOnly(v) }

// Clearable добавляет кнопку очистки в поле ввода.
func Clearable(v bool) Option { return ui.Clearable(v) }

// Indeterminate переводит Checkbox в промежуточное состояние.
func Indeterminate(v bool) Option { return ui.Indeterminate(v) }

// Searchable включает поиск в выпадающем списке.
func Searchable(v bool) Option { return ui.Searchable(v) }

// InputFilter задаёт фильтр допустимых символов для ввода.
func InputFilter(fn func(rune) bool) Option { return ui.InputFilter(fn) }

