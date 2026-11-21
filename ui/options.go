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
	DropAbove  DropZone = iota // Сброс выше/перед целевым элементом
	DropBelow                  // Сброс ниже/после целевого элемента
	DropInside                 // Сброс внутрь целевого элемента (для папок/контейнеров)
)

// ВАЖНО: данный docstring должен быть строго синхронизирован с ui_controls_extra.go (anyui.DragEvent).

// DragEvent описывает событие завершения перетаскивания элемента (Reorder).
//
// Для списков и таблиц (Table, ListView, ListViewEx):
//   - FromIndex: исходный индекс перемещаемого элемента (>= 0).
//   - ToIndex: уже вычисленная позиция вставки в списке после удаления перемещаемого элемента (0 <= ToIndex < rowCount).
//   - FromKey и ToKey строго пусты ("").
//   - Допустимые зоны Drop: только DropAbove и DropBelow.
//   - Если ToIndex == FromIndex, операция является no-op и событие не генерируется.
//   - ListViewEx: при Reorder приложение обязано синхронно переставлять элементы в checked []bool и selected map[int]bool, иначе отметки сохранятся за исходными числовыми индексами.
//
// Для иерархических деревьев (TreeView, TreeViewEx):
//   - FromIndex и ToIndex строго равны -1.
//   - FromKey: ID перемещаемого узла (sourceNode.ID).
//   - ToKey: ID якорного узла (targetNode.ID), относительно которого применяется Drop; позицию вставки в список детей родителя вычисляет приложение.
//   - По умолчанию перетаскивание узлов разрешено только внутри одного родителя (source.Parent == target.Parent), зоны DropAbove и DropBelow. При попытке сброса на узел другого родителя событие отбрасывается.
//   - По умолчанию DropBelow на узле (в том числе раскрытом) вставляет элемент в того же родителя сразу после всего поддерева ToKey; для раскрытого узла с детьми при включённой AllowReparent() — первым ребёнком в ToKey.Children (см. ниже).
//   - Сброс узла в самого себя (ToKey == FromKey) является no-op и событие не генерирует.
//   - TreeViewEx: при перетаскивании одного узла из группы выделенных перемещается только он, при этом выделение группы не сбрасывается.
//
// Режим AllowReparent() в TreeView и TreeViewEx (смена родителя и вложение):
//   - DropInside: новым родителем становится сам ToKey (вставка в ToKey.Children).
//   - DropAbove: новым родителем становится target.ParentID (для корневого узла — "" / верхний уровень), вставка перед ToKey.
//   - DropBelow: для раскрытого узла с детьми (ToKey.Expanded && len(ToKey.Children) > 0) новым родителем становится сам ToKey (позиция первого ребёнка); для свёрнутого узла или листа — target.ParentID (сразу после поддерева ToKey).
//   - Защита от циклов и no-op: сброс узла в любого своего потомка и DropInside на собственного текущего родителя являются no-op и событие не генерируют. При этом DropAbove/DropBelow на собственного текущего родителя не являются no-op (перемещают узел на уровень родителя).
type DragEvent struct {
	FromWidget string   // ID виджета-источника перетаскивания
	ToWidget   string   // ID целевого виджета
	FromKey    string   // Ключ перетаскиваемого узла (для TreeView — node.ID; "" для Table/ListView)
	ToKey      string   // Ключ якорного целевого узла (для TreeView — node.ID; "" для Table/ListView)
	FromIndex  int      // Индекс перетаскиваемой строки (для Table/ListView; -1 для TreeView)
	ToIndex    int      // Позиция вставки в списке после удаления перемещаемого элемента (0 <= ToIndex < rowCount; -1 для TreeView)
	Drop       DropZone // Зона сброса (DropAbove, DropBelow; DropInside при AllowReparent в TreeView)
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
	AllowReparent    bool
	OnReorder        func(ev DragEvent)
}

// Auto — константа sentinel-значения для автоматического занятия доступного остатка области контейнера.
const Auto Dp = -1

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
func Secondary(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.Variant = VariantSecondary
		} else {
			s.Variant = VariantDefault
		}
	}
}

// Danger задаёт стиль предупреждения об опасном действии.
func Danger(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.Variant = VariantDanger
		} else {
			s.Variant = VariantDefault
		}
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
func Password(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.Password = val
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

// Searchable включает строку поиска/фильтрации внутри выпадающего списка Select.
func Searchable(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.Searchable = val
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
func FillWidth(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.FillWidth = val
	}
}

// FillHeight растягивает виджет на всю доступную высоту контейнера.
func FillHeight(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.FillHeight = val
	}
}

// AutoHeight включает автоматический подбор высоты контейнера (например, Card) по содержимому.
func AutoHeight(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.AutoHeight = val
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
func Ellipsis(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.Ellipsis = val
	}
}

// LineNumbers включает отображение номеров строк в TextArea.
func LineNumbers(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		s.LineNumbers = val
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
func TextAlignCenter(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.TextAlign = AlignTextCenter
		} else {
			s.TextAlign = AlignTextLeft
		}
	}
}

// TextAlignLeft выравнивает текст и placeholder по левому краю (для TextInput и Label).
func TextAlignLeft(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.TextAlign = AlignTextLeft
		}
	}
}

// TextAlignRight выравнивает текст и placeholder по правому краю (для TextInput и Label).
func TextAlignRight(enabled ...bool) Option {
	val := optBool(enabled)
	return func(s *WidgetStyle) {
		if val {
			s.TextAlign = AlignTextRight
		} else {
			s.TextAlign = AlignTextLeft
		}
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

// Reorderable разрешает перетаскивание элементов внутри компонента для изменения порядка.
// Действует для Table, ListView, ListViewEx, TreeView, TreeViewEx; TableEx пока не поддерживается.
// Без опции OnReorder результат перетаскивания отбрасывается.
func Reorderable(enabled ...bool) Option {
	return func(s *WidgetStyle) {
		if len(enabled) == 0 || enabled[0] {
			s.Reorderable = true
		} else {
			s.Reorderable = false
		}
	}
}

// AllowReparent разрешает в TreeView и TreeViewEx смену родителя при перетаскивании узлов
// (DropAbove/DropBelow между разными родителями) и вложение внутрь целевого узла (DropInside)
// с автоматической защитой от циклов. Работает совместно с Reorderable() и OnReorder(...).
func AllowReparent(enabled ...bool) Option {
	return func(s *WidgetStyle) {
		if len(enabled) == 0 || enabled[0] {
			s.AllowReparent = true
		} else {
			s.AllowReparent = false
		}
	}
}

// OnReorder регистрирует обработчик события завершения перетаскивания элемента.
// Действует для Table, ListView, ListViewEx, TreeView, TreeViewEx; TableEx пока не поддерживается.
func OnReorder(fn func(ev DragEvent)) Option {
	return func(s *WidgetStyle) {
		s.OnReorder = fn
	}
}
