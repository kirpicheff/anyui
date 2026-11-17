package anyui

import "git.gtw.su/kirpich/anyui/ui"

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

// MinWidth задаёт минимальную ширину виджета в логических пикселях.
func MinWidth(dp int) Option { return ui.MinWidth(dp) }

// MaxWidth задаёт максимальную ширину виджета в логических пикселях.
func MaxWidth(dp int) Option { return ui.MaxWidth(dp) }

// MinHeight задаёт минимальную высоту виджета в логических пикселях.
func MinHeight(dp int) Option { return ui.MinHeight(dp) }

// MaxHeight задаёт максимальную высоту виджета в логических пикселях.
func MaxHeight(dp int) Option { return ui.MaxHeight(dp) }

// FillWidth растягивает виджет на всю доступную ширину контейнера.
func FillWidth() Option { return ui.FillWidth() }

// FillHeight растягивает виджет на всю доступную высоту контейнера.
func FillHeight() Option { return ui.FillHeight() }

// AutoHeight включает автоматический подбор высоты контейнера по содержимому.
func AutoHeight() Option { return ui.AutoHeight() }

// Flex задаёт пропорциональный вес виджета при делении пространства.
func Flex(weight float32) Option { return ui.Flex(weight) }

// WordWrap включает перенос текста по ширине (для Label и SelectableLabel).
func WordWrap(v bool) Option { return ui.WordWrap(v) }

// Ellipsis включает обрезку текста с многоточием при нехватке ширины (для Label).
func Ellipsis(v bool) Option { return ui.Ellipsis(v) }

// LineNumbers включает отображение номеров строк в TextArea.
func LineNumbers(v bool) Option { return ui.LineNumbers(v) }

// Error задаёт текст ошибки валидации и подсвечивает рамку поля ввода.
func Error(text string) Option { return ui.Error(text) }

