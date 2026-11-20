package anyui

import "github.com/kirpicheff/anyui/ui"

// Option настраивает стиль, размеры или поведение отдельного виджета.
type Option = ui.Option

// AutoSize — константа sentinel-значения для автоматического подбора размера по остатку доступной области.
const AutoSize = ui.AutoSize

// Auto возвращает sentinel-значение (-1) для автоматического занятия доступного остатка области контейнера.
func Auto() Dp { return ui.Auto() }

// Bg переопределяет цвет фона виджета.
func Bg(c Color) Option { return ui.Bg(c) }

// TextColor переопределяет цвет текста виджета.
func TextColor(c Color) Option { return ui.TextColor(c) }

// BorderColor переопределяет цвет рамки виджета.
func BorderColor(c Color) Option { return ui.BorderColor(c) }

// Radius переопределяет радиус скругления углов в логических пикселях.
func Radius(dp float32) Option { return ui.Radius(dp) }

// Width задаёт фиксированную ширину виджета в логических пикселях.
func Width(dp Dp) Option { return ui.Width(dp) }

// Height задаёт фиксированную высоту виджета в логических пикселях.
func Height(dp Dp) Option { return ui.Height(dp) }

// FontSize переопределяет размер шрифта виджета в логических пикселях.
func FontSize(dp Dp) Option { return ui.FontSize(dp) }

// Padding задаёт внутренний отступ виджета в логических пикселях.
func Padding(dp Dp) Option { return ui.Padding(dp) }

// Icon задаёт SVG-иконку виджета.
func Icon(svg string) Option { return ui.Icon(svg) }

// Disabled переводит виджет в неактивное состояние.
func Disabled(enabled ...bool) Option { return ui.Disabled(enabled...) }

// Primary задаёт акцентный стиль виджета.
func Primary(enabled ...bool) Option { return ui.Primary(enabled...) }

// Secondary задаёт второстепенный стиль виджета.
func Secondary() Option { return ui.Secondary() }

// Danger задаёт стиль предупреждения об опасном действии.
func Danger() Option { return ui.Danger() }

// Ghost задаёт прозрачный стиль без заливки фона в покое.
func Ghost(enabled ...bool) Option { return ui.Ghost(enabled...) }

// Bold выбирает полужирное начертание шрифта для текста виджета.
func Bold(enabled ...bool) Option { return ui.Bold(enabled...) }

// Mono выбирает моноширинное начертание шрифта для текста виджета.
func Mono(enabled ...bool) Option { return ui.Mono(enabled...) }

// IconRight задаёт SVG-иконку справа.
func IconRight(svg string) Option { return ui.IconRight(svg) }

// Password включает режим пароля для поля ввода.
func Password() Option { return ui.Password() }

// PasswordToggle включает режим пароля со встроенной кнопкой-глазиком справа.
func PasswordToggle(visible ...*bool) Option { return ui.PasswordToggle(visible...) }

// ReadOnly запрещает редактирование текстового поля.
func ReadOnly(enabled ...bool) Option { return ui.ReadOnly(enabled...) }

// Clearable добавляет кнопку очистки в поле ввода.
func Clearable(enabled ...bool) Option { return ui.Clearable(enabled...) }

// Indeterminate переводит Checkbox в промежуточное состояние.
func Indeterminate(enabled ...bool) Option { return ui.Indeterminate(enabled...) }

// Searchable включает поиск в выпадающем списке.
func Searchable(v bool) Option { return ui.Searchable(v) }

// InputFilter задаёт фильтр допустимых символов для ввода.
func InputFilter(fn func(rune) bool) Option { return ui.InputFilter(fn) }

// MinWidth задаёт минимальную ширину виджета в логических пикселях.
func MinWidth(dp Dp) Option { return ui.MinWidth(dp) }

// MaxWidth задаёт максимальную ширину виджета в логических пикселях.
func MaxWidth(dp Dp) Option { return ui.MaxWidth(dp) }

// MinHeight задаёт минимальную высоту виджета в логических пикселях.
func MinHeight(dp Dp) Option { return ui.MinHeight(dp) }

// MaxHeight задаёт максимальную высоту виджета в логических пикселях.
func MaxHeight(dp Dp) Option { return ui.MaxHeight(dp) }

// FillWidth растягивает виджет на всю доступную ширину контейнера.
func FillWidth() Option { return ui.FillWidth() }

// FillHeight растягивает виджет на всю доступную высоту контейнера.
func FillHeight() Option { return ui.FillHeight() }

// AutoHeight включает автоматический подбор высоты контейнера по содержимому.
func AutoHeight() Option { return ui.AutoHeight() }

// Flex задаёт пропорциональный вес виджета при делении пространства.
func Flex(weight float32) Option { return ui.Flex(weight) }

// WordWrap включает перенос текста по ширине (для Label и SelectableLabel).
func WordWrap(enabled ...bool) Option { return ui.WordWrap(enabled...) }

// Ellipsis включает обрезку текста с многоточием при нехватке ширины (для Label).
func Ellipsis(v bool) Option { return ui.Ellipsis(v) }

// LineNumbers включает отображение номеров строк в TextArea.
func LineNumbers(v bool) Option { return ui.LineNumbers(v) }

// Error задаёт текст ошибки валидации и подсвечивает рамку поля ввода.
func Error(text string) Option { return ui.Error(text) }

// TextAlign задаёт горизонтальное выравнивание текста внутри виджета.
type TextAlign = ui.TextAlign

const (
	AlignTextLeft   = ui.AlignTextLeft
	AlignTextCenter = ui.AlignTextCenter
	AlignTextRight  = ui.AlignTextRight
)

// HoverTextColor переопределяет цвет текста и иконки при наведении (для Button).
func HoverTextColor(c Color) Option { return ui.HoverTextColor(c) }

// HoverBg переопределяет цвет фона при наведении (для Button).
func HoverBg(c Color) Option { return ui.HoverBg(c) }

// HoverBorderColor переопределяет цвет рамки при наведении (для Button).
func HoverBorderColor(c Color) Option { return ui.HoverBorderColor(c) }

// FocusBorderColor переопределяет цвет рамки при фокусе (для TextInput).
func FocusBorderColor(c Color) Option { return ui.FocusBorderColor(c) }

// TextAlignCenter выравнивает текст и placeholder по центру (для TextInput и Label).
func TextAlignCenter() Option { return ui.TextAlignCenter() }

// TextAlignLeft выравнивает текст и placeholder по левому краю (для TextInput и Label).
func TextAlignLeft() Option { return ui.TextAlignLeft() }

// TextAlignRight выравнивает текст и placeholder по правому краю (для TextInput и Label).
func TextAlignRight() Option { return ui.TextAlignRight() }

// LetterSpacing задаёт межбуквенный интервал в логических пикселях (для Label и Button).
func LetterSpacing(dp float32) Option { return ui.LetterSpacing(dp) }

// TextShadow задаёт тень или мягкое свечение текста (для Label и Button).
func TextShadow(offsetX, offsetY, blur float32, c Color) Option {
	return ui.TextShadow(offsetX, offsetY, blur, c)
}

