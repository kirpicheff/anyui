package anyui

import "github.com/kirpicheff/anyui/ui"

// Option настраивает стиль, размеры или поведение отдельного виджета.
type Option = ui.Option

// Auto — константа sentinel-значения для автоматического занятия доступного остатка области контейнера.
const Auto Dp = ui.Auto

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
func Secondary(enabled ...bool) Option { return ui.Secondary(enabled...) }

// Danger задаёт стиль предупреждения об опасном действии.
func Danger(enabled ...bool) Option { return ui.Danger(enabled...) }

// Ghost задаёт прозрачный стиль без заливки фона в покое.
func Ghost(enabled ...bool) Option { return ui.Ghost(enabled...) }

// Bold выбирает полужирное начертание шрифта для текста виджета.
func Bold(enabled ...bool) Option { return ui.Bold(enabled...) }

// Mono выбирает моноширинное начертание шрифта для текста виджета.
func Mono(enabled ...bool) Option { return ui.Mono(enabled...) }

// IconRight задаёт SVG-иконку справа.
func IconRight(svg string) Option { return ui.IconRight(svg) }

// Password включает режим пароля для поля ввода.
func Password(enabled ...bool) Option { return ui.Password(enabled...) }

// PasswordToggle включает режим пароля со встроенной кнопкой-глазиком справа.
func PasswordToggle(visible ...*bool) Option { return ui.PasswordToggle(visible...) }

// ReadOnly запрещает редактирование текстового поля.
func ReadOnly(enabled ...bool) Option { return ui.ReadOnly(enabled...) }

// Clearable добавляет кнопку очистки в поле ввода.
func Clearable(enabled ...bool) Option { return ui.Clearable(enabled...) }

// Indeterminate переводит Checkbox в промежуточное состояние.
func Indeterminate(enabled ...bool) Option { return ui.Indeterminate(enabled...) }

// Searchable включает поиск в выпадающем списке.
func Searchable(enabled ...bool) Option { return ui.Searchable(enabled...) }

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
func FillWidth(enabled ...bool) Option { return ui.FillWidth(enabled...) }

// FillHeight растягивает виджет на всю доступную высоту контейнера.
func FillHeight(enabled ...bool) Option { return ui.FillHeight(enabled...) }

// AutoHeight включает автоматический подбор высоты контейнера по содержимому.
func AutoHeight(enabled ...bool) Option { return ui.AutoHeight(enabled...) }

// WordWrap включает перенос текста по ширине (для Label и SelectableLabel).
func WordWrap(enabled ...bool) Option { return ui.WordWrap(enabled...) }

// Ellipsis включает обрезку текста с многоточием при нехватке ширины (для Label).
func Ellipsis(enabled ...bool) Option { return ui.Ellipsis(enabled...) }

// LineNumbers включает отображение номеров строк в TextArea.
func LineNumbers(enabled ...bool) Option { return ui.LineNumbers(enabled...) }

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

// FocusBorderColor переопределяет цвет рамки при фокусе (для Button, TextInput, MaskedInput, TextArea, ListView, ListViewEx, Table, TableEx, TreeView и TreeViewEx).
func FocusBorderColor(c Color) Option { return ui.FocusBorderColor(c) }

// TextAlignCenter выравнивает текст и placeholder по центру (для TextInput и Label).
func TextAlignCenter(enabled ...bool) Option { return ui.TextAlignCenter(enabled...) }

// TextAlignLeft выравнивает текст и placeholder по левому краю (для TextInput и Label).
func TextAlignLeft(enabled ...bool) Option { return ui.TextAlignLeft(enabled...) }

// TextAlignRight выравнивает текст и placeholder по правому краю (для TextInput и Label).
func TextAlignRight(enabled ...bool) Option { return ui.TextAlignRight(enabled...) }

// LetterSpacing задаёт межбуквенный интервал в логических пикселях (для Label, StatusDot, Button, ContextMenu и Tooltip).
func LetterSpacing(dp float32) Option { return ui.LetterSpacing(dp) }

// TextShadow задаёт тень или мягкое свечение текста (для Label, StatusDot, Button, ContextMenu и Tooltip).
func TextShadow(offsetX, offsetY, blur float32, c Color) Option {
	return ui.TextShadow(offsetX, offsetY, blur, c)
}

// ScrollToIndex задаёт индекс элемента для программной прокрутки в ListView, ListViewEx, Table и TableEx.
func ScrollToIndex(idx int) Option { return ui.ScrollToIndex(idx) }

// ScrollToID задаёт строковый идентификатор дочернего элемента или якоря для программной прокрутки в ScrollView.
func ScrollToID(id string) Option { return ui.ScrollToID(id) }

// Reorderable разрешает перетаскивание элементов внутри компонента для изменения порядка.
// Действует для Table, TableEx, ListView, ListViewEx, TreeView, TreeViewEx.
// Без опции OnReorder результат перетаскивания отбрасывается.
func Reorderable(enabled ...bool) Option { return ui.Reorderable(enabled...) }

// AllowReparent разрешает в TreeView и TreeViewEx смену родителя при перетаскивании узлов
// (DropAbove/DropBelow между разными родителями) и вложение внутрь целевого узла (DropInside)
// с автоматической защитой от циклов. Работает совместно с Reorderable() и OnReorder(...).
func AllowReparent(enabled ...bool) Option { return ui.AllowReparent(enabled...) }

// OnReorder регистрирует обработчик события завершения перетаскивания элемента.
// Действует для Table, TableEx, ListView, ListViewEx, TreeView, TreeViewEx.
func OnReorder(fn func(ev DragEvent)) Option { return ui.OnReorder(fn) }

// DragSource разрешает виджету выступать источником кросс-виджетного перетаскивания с типизированным DragPayload.
// Если fn == nil, автоматически формируется DragPayload{Type: typeName}.
// Действует для Table, TableEx, ListView, ListViewEx, TreeView, TreeViewEx.
func DragSource(typeName string, fn func(index int, key string) DragPayload) Option {
	return ui.DragSource(typeName, fn)
}

// AcceptDrop разрешает виджету принимать внешний перенос элементов с указанными типами Payload.Type.
// Работает совместно с OnDrop(...). Действует для Table, TableEx, ListView, ListViewEx, TreeView, TreeViewEx.
func AcceptDrop(types ...string) Option { return ui.AcceptDrop(types...) }

// OnDrop регистрирует обработчик события внешнего кросс-виджетного сброса (Drop).
// Работает совместно с AcceptDrop(...). Действует для Table, TableEx, ListView, ListViewEx, TreeView, TreeViewEx.
func OnDrop(fn func(ev DragEvent)) Option { return ui.OnDrop(fn) }

// TabOrientation задаёт ориентацию панели вкладок TabBar (Horizontal или Vertical).
// По умолчанию TabBar располагается горизонтально.
func TabOrientation(orient Orientation) Option { return ui.TabOrientation(orient) }

// HideAddButton скрывает кнопку '+' добавления новой вкладки в TabBar.
func HideAddButton(enabled ...bool) Option { return ui.HideAddButton(enabled...) }

// OnSubmit регистрирует обработчик подтверждения ввода по клавише Enter (TextInput, MaskedInput).
func OnSubmit(fn func()) Option { return ui.OnSubmit(fn) }
