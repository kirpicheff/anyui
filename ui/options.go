package ui

import "anyui/render"

// Variant задаёт стилевой вариант контрола (например, кнопки).
type Variant uint8

const (
	VariantDefault Variant = iota
	VariantPrimary
	VariantSecondary
	VariantDanger
	VariantGhost
)

// WidgetStyle хранит переопределения оформления и размеров конкретного виджета.
type WidgetStyle struct {
	Bg            *render.Color
	Color         *render.Color
	BorderColor   *render.Color
	Radius        *float32
	WidthDp       *int
	HeightDp      *int
	MinWidthDp    *int
	MaxWidthDp    *int
	MinHeightDp   *int
	MaxHeightDp   *int
	FillWidth     bool
	FillHeight    bool
	AutoHeight    bool
	FlexWeight    float32
	FontSizeDp    *int
	PadDp         *int
	IconSVG       string
	IconRightSVG  string
	FontStyle     render.FontStyle
	Disabled      bool
	Variant       Variant
	Password      bool
	ReadOnly      bool
	Clearable     bool
	Indeterminate bool
	Searchable    bool
	InputFilter   func(rune) bool
	WordWrap      bool
	Ellipsis      bool
	LineNumbers   bool
	ErrorText     string
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
func Width(dp int) Option {
	return func(s *WidgetStyle) {
		s.WidthDp = &dp
	}
}

// Height задаёт фиксированную высоту виджета в логических пикселях.
func Height(dp int) Option {
	return func(s *WidgetStyle) {
		s.HeightDp = &dp
	}
}

// FontSize переопределяет размер шрифта виджета в логических пикселях.
func FontSize(dp int) Option {
	return func(s *WidgetStyle) {
		s.FontSizeDp = &dp
	}
}

// Padding задаёт внутренний отступ виджета в логических пикселях.
func Padding(dp int) Option {
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

// Disabled переводит виджет в неактивное состояние.
func Disabled(v bool) Option {
	return func(s *WidgetStyle) {
		s.Disabled = v
	}
}

// Primary задаёт акцентный стиль виджета.
func Primary() Option {
	return func(s *WidgetStyle) {
		s.Variant = VariantPrimary
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
func Ghost() Option {
	return func(s *WidgetStyle) {
		s.Variant = VariantGhost
	}
}

// Bold выбирает полужирное начертание шрифта для текста виджета.
func Bold() Option {
	return func(s *WidgetStyle) {
		s.FontStyle = render.FontBold
	}
}

// Mono выбирает моноширинное начертание шрифта для текста виджета.
func Mono() Option {
	return func(s *WidgetStyle) {
		s.FontStyle = render.FontMono
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

// ReadOnly запрещает редактирование текста, оставляя возможность выделения и копирования.
func ReadOnly(v bool) Option {
	return func(s *WidgetStyle) {
		s.ReadOnly = v
	}
}

// Clearable добавляет кнопку быстрой очистки в правую часть поля ввода.
func Clearable(v bool) Option {
	return func(s *WidgetStyle) {
		s.Clearable = v
	}
}

// Indeterminate переводит Checkbox в промежуточное (неопределённое) состояние.
func Indeterminate(v bool) Option {
	return func(s *WidgetStyle) {
		s.Indeterminate = v
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
func MinWidth(dp int) Option {
	return func(s *WidgetStyle) {
		s.MinWidthDp = &dp
	}
}

// MaxWidth задаёт максимальную ширину виджета в логических пикселях.
func MaxWidth(dp int) Option {
	return func(s *WidgetStyle) {
		s.MaxWidthDp = &dp
	}
}

// MinHeight задаёт минимальную высоту виджета в логических пикселях.
func MinHeight(dp int) Option {
	return func(s *WidgetStyle) {
		s.MinHeightDp = &dp
	}
}

// MaxHeight задаёт максимальную высоту виджета в логических пикселях.
func MaxHeight(dp int) Option {
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
func WordWrap(v bool) Option {
	return func(s *WidgetStyle) {
		s.WordWrap = v
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

