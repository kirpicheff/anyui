package ui

import "github.com/kirpicheff/anyui/render"

// Variant задаёт стилевой вариант контрола (например, кнопки).
type Variant uint8

const (
	VariantDefault Variant = iota
	VariantPrimary
	VariantSecondary
	VariantDanger
	VariantGhost
)

// TextAlign задаёт горизонтальное выравнивание текста внутри виджета.
type TextAlign uint8

const (
	AlignTextLeft TextAlign = iota
	AlignTextCenter
	AlignTextRight
)

// DropZone определяет позицию сброса перетаскиваемого элемента относительно целевого.
type DropZone uint8

const (
	DropAbove DropZone = iota // Сброс выше/перед целевым элементом
	DropBelow                 // Сброс ниже/после целевого элемента
	DropInside                // Сброс внутрь целевого элемента (для папок/контейнеров)
)

// DragEvent описывает событие завершения перетаскивания элемента (Reorder).
type DragEvent struct {
	FromWidget string
	ToWidget   string
	FromKey    string
	ToKey      string
	Drop       DropZone
}

// WidgetStyle хранит переопределения оформления и размеров конкретного виджета.
type WidgetStyle struct {
	Bg               *render.Color
	HoverBg          *render.Color
	Color            *render.Color
	HoverTextColor   *render.Color
	BorderColor      *render.Color
	HoverBorderColor *render.Color
	FocusBorderColor *render.Color
	Radius           *float32
	WidthDp          *Dp
	HeightDp         *Dp
	MinWidthDp       *Dp
	MaxWidthDp       *Dp
	MinHeightDp      *Dp
	MaxHeightDp      *Dp
	FillWidth        bool
	FillHeight       bool
	AutoHeight       bool
	FlexWeight       float32
	FontSizeDp       *Dp
	PadDp            *Dp
	IconSVG          string
	IconRightSVG     string
	FontStyle        render.FontStyle
	TextAlign        TextAlign
	LetterSpacing    float32
	TextShadow       *render.TextShadowStyle
	Disabled         bool
	Variant          Variant
	Password         bool
	PasswordToggle   bool
	PasswordVisible  *bool
	ReadOnly         bool
	Clearable        bool
	Indeterminate    bool
	Searchable       bool
	InputFilter      func(rune) bool
	WordWrap         bool
	Ellipsis         bool
	LineNumbers      bool
	ErrorText        string
	ScrollToIndex    *int
	ScrollToID       string
	Reorderable      bool
	OnReorder        func(ev DragEvent)
}

// AutoSize — константа sentinel-значения для автоматического подбора размера по остатку доступной области.
const AutoSize Dp = -1

// Auto возвращает sentinel-значение (-1) для автоматического занятия доступного остатка области контейнера.
func Auto() Dp {
	return AutoSize
}

// Option настраивает стиль или поведение отдельного виджета.
type Option func(*WidgetStyle)

// ApplyOptions применяет список опций и возвращает итоговый WidgetStyle.
func ApplyOptions(opts []Option) WidgetStyle {
	var s WidgetStyle
	for _, opt := range opts {
		if opt != nil {
			opt(&s)
		}
	}
	return s
}

// Bg переопределяет цвет фона виджета.
func Bg(c render.Color) Option {
	return func(s *WidgetStyle) {
		s.Bg = &c
	}
}

// TextColor переопределяет цвет текста или основного содержимого виджета.
func TextColor(c render.Color) Option {
	return func(s *WidgetStyle) {
		s.Color = &c
	}
}

// BorderColor переопределяет цвет рамки виджета.
func BorderColor(c render.Color) Option {
	return func(s *WidgetStyle) {
		s.BorderColor = &c
	}
}

// Radius переопределяет радиус скругления углов в логических пикселях.
func Radius(dp float32) Option {
	return func(s *WidgetStyle) {
		s.Radius = &dp
	}
}

// Width задаёт фиксированную ширину виджета в логических пикселях.
func Width(dp Dp) Option {
	return func(s *WidgetStyle) {
		s.WidthDp = &dp
	}
}

// Height задаёт фиксированную высоту виджета в логических пикселях.
func Height(dp Dp) Option {
	return func(s *WidgetStyle) {
		s.HeightDp = &dp
	}
}

// FontSize переопределяет размер шрифта виджета в логических пикселях.
func FontSize(dp Dp) Option {
	return func(s *WidgetStyle) {
		s.FontSizeDp = &dp
	}
}

// Padding задаёт внутренний отступ виджета в логических пикселях.
func Padding(dp Dp) Option {
	return func(s *WidgetStyle) {
		s.PadDp = &dp
	}
}

// Icon задаёт SVG-иконку для виджета (растеризация будет добавлена в Фазе 2).
func Icon(svg string) Option {
	return func(s *WidgetStyle) {
		s.IconSVG = svg
	}
}

func optBool(enabled []bool) bool {
	if len(enabled) == 0 {
		return true
	}
	return enabled[0]
}

// Disabled переводит виджет в неактивное состояние.
func Disabled(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.Disabled = val
	}
}

// Primary задаёт акцентный стиль виджета.
func Primary(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.Variant = VariantPrimary
		} else {
			s.Variant = VariantDefault
		}
	}
}

// Secondary задаёт второстепенный стиль виджета.
func Secondary() Option {
	return func(s *WidgetStyle) {
		s.Variant = VariantSecondary
	}
}

// Danger задаёт стиль предупреждения об опасном действии.
func Danger() Option {
	return func(s *WidgetStyle) {
		s.Variant = VariantDanger
	}
}

// Ghost задаёт прозрачный стиль без заливки фона в покое.
func Ghost(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.Variant = VariantGhost
		} else {
			s.Variant = VariantDefault
		}
	}
}

// Bold выбирает полужирное начертание шрифта для текста виджета.
func Bold(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.FontStyle = render.FontBold
		} else {
			s.FontStyle = render.FontRegular
		}
	}
}

// Mono выбирает моноширинное начертание шрифта для текста виджета.
func Mono(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.FontStyle = render.FontMono
		} else {
			s.FontStyle = render.FontRegular
		}
	}
}

// IconRight задаёт SVG-иконку справа (например, для поля ввода).
func IconRight(svg string) Option {
	return func(s *WidgetStyle) {
		s.IconRightSVG = svg
	}
}

// Password включает режим маскировки вводимых символов (звёздочки/точки).
func Password() Option {
	return func(s *WidgetStyle) {
		s.Password = true
	}
}

// PasswordToggle включает режим поля пароля со встроенной кнопкой-глазиком справа.
// Опционально принимает указатель на bool для внешнего управления видимостью пароля.
// Если указатель не передан, AnyUI сохраняет состояние видимости автоматически.
func PasswordToggle(visible ...*bool) Option {
	return func(s *WidgetStyle) {
		s.Password = true
		s.PasswordToggle = true
		if len(visible) > 0 && visible[0] != nil {
			s.PasswordVisible = visible[0]
		}
	}
}

// ReadOnly запрещает редактирование текста, оставляя возможность выделения и копирования.
func ReadOnly(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.ReadOnly = val
	}
}

// Clearable добавляет кнопку быстрой очистки в правую часть поля ввода.
func Clearable(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.Clearable = val
	}
}

// Indeterminate переводит Checkbox в промежуточное (неопределённое) состояние.
func Indeterminate(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.Indeterminate = val
	}
}

// Searchable включает строку поиска/фильтрации внутри выпадающего списка Select/ComboBox.
func Searchable(v bool) Option {
	return func(s *WidgetStyle) {
		s.Searchable = v
	}
}

// InputFilter задаёт предикат допустимых символов при вводе в текстовое поле.
func InputFilter(fn func(rune) bool) Option {
	return func(s *WidgetStyle) {
		s.InputFilter = fn
	}
}

// MinWidth задаёт минимальную ширину виджета в логических пикселях.
func MinWidth(dp Dp) Option {
	return func(s *WidgetStyle) {
		s.MinWidthDp = &dp
	}
}

// MaxWidth задаёт максимальную ширину виджета в логических пикселях.
func MaxWidth(dp Dp) Option {
	return func(s *WidgetStyle) {
		s.MaxWidthDp = &dp
	}
}

// MinHeight задаёт минимальную высоту виджета в логических пикселях.
func MinHeight(dp Dp) Option {
	return func(s *WidgetStyle) {
		s.MinHeightDp = &dp
	}
}

// MaxHeight задаёт максимальную высоту виджета в логических пикселях.
func MaxHeight(dp Dp) Option {
	return func(s *WidgetStyle) {
		s.MaxHeightDp = &dp
	}
}

// FillWidth растягивает виджет на всю доступную ширину контейнера.
func FillWidth() Option {
	return func(s *WidgetStyle) {
		s.FillWidth = true
	}
}

// FillHeight растягивает виджет на всю доступную высоту контейнера.
func FillHeight() Option {
	return func(s *WidgetStyle) {
		s.FillHeight = true
	}
}

// AutoHeight включает автоматический подбор высоты контейнера (например, Card) по содержимому.
func AutoHeight() Option {
	return func(s *WidgetStyle) {
		s.AutoHeight = true
	}
}

// Flex задаёт пропорциональный вес виджета при делении пространства.
func Flex(weight float32) Option {
	return func(s *WidgetStyle) {
		s.FlexWeight = weight
	}
}

// WordWrap включает перенос текста по ширине (для Label и SelectableLabel).
func WordWrap(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.WordWrap = val
	}
}

// Ellipsis включает обрезку текста с многоточием при нехватке ширины (для Label).
func Ellipsis(v bool) Option {
	return func(s *WidgetStyle) {
		s.Ellipsis = v
	}
}

// LineNumbers включает отображение номеров строк в TextArea.
func LineNumbers(v bool) Option {
	return func(s *WidgetStyle) {
		s.LineNumbers = v
	}
}

// Error задаёт текст ошибки валидации и подсвечивает рамку поля ввода.
func Error(text string) Option {
	return func(s *WidgetStyle) {
		s.ErrorText = text
	}
}

// HoverTextColor переопределяет цвет текста и иконки при наведении (для Button).
func HoverTextColor(c render.Color) Option {
	return func(s *WidgetStyle) {
		s.HoverTextColor = &c
	}
}

// HoverBg переопределяет цвет фона при наведении (для Button).
func HoverBg(c render.Color) Option {
	return func(s *WidgetStyle) {
		s.HoverBg = &c
	}
}

// HoverBorderColor переопределяет цвет рамки при наведении (для Button).
func HoverBorderColor(c render.Color) Option {
	return func(s *WidgetStyle) {
		s.HoverBorderColor = &c
	}
}

// FocusBorderColor переопределяет цвет рамки при фокусе (для TextInput).
func FocusBorderColor(c render.Color) Option {
	return func(s *WidgetStyle) {
		s.FocusBorderColor = &c
	}
}

// TextAlignCenter выравнивает текст и placeholder по центру (для TextInput и Label).
func TextAlignCenter() Option {
	return func(s *WidgetStyle) {
		s.TextAlign = AlignTextCenter
	}
}

// TextAlignLeft выравнивает текст и placeholder по левому краю (для TextInput и Label).
func TextAlignLeft() Option {
	return func(s *WidgetStyle) {
		s.TextAlign = AlignTextLeft
	}
}

// TextAlignRight выравнивает текст и placeholder по правому краю (для TextInput и Label).
func TextAlignRight() Option {
	return func(s *WidgetStyle) {
		s.TextAlign = AlignTextRight
	}
}

// LetterSpacing задаёт межбуквенный интервал в логических пикселях (для Label и Button).
func LetterSpacing(dp float32) Option {
	return func(s *WidgetStyle) {
		s.LetterSpacing = dp
	}
}

// TextShadow задаёт тень или мягкое свечение текста (для Label и Button).
func TextShadow(offsetX, offsetY, blur float32, c render.Color) Option {
	return func(s *WidgetStyle) {
		s.TextShadow = &render.TextShadowStyle{
			OffsetX: offsetX,
			OffsetY: offsetY,
			Blur:    blur,
			Color:   c,
		}
	}
}

// ScrollToIndex задаёт индекс элемента для программной прокрутки в ListView и Table.
func ScrollToIndex(idx int) Option {
	return func(s *WidgetStyle) {
		s.ScrollToIndex = &idx
	}
}

// ScrollToID задаёт строковый идентификатор дочернего элемента или якоря для программной прокрутки в ScrollView.
func ScrollToID(id string) Option {
	return func(s *WidgetStyle) {
		s.ScrollToID = id
	}
}

// Reorderable разрешает перетаскивание элементов (строк или узлов) внутри компонента для изменения порядка.
func Reorderable(enabled ...bool) Option {
	return func(s *WidgetStyle) {
		if len(enabled) == 0 || enabled[0] {
			s.Reorderable = true
		} else {
			s.Reorderable = false
		}
	}
}

// OnReorder регистрирует обработчик события завершения перетаскивания элемента.
func OnReorder(fn func(ev DragEvent)) Option {
	return func(s *WidgetStyle) {
		s.Reorderable = true
		s.OnReorder = fn
	}
}



