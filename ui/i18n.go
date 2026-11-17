package ui

// Strings содержит локализуемые встроенные строки стандартных виджетов (i18n).
type Strings struct {
	// Контекстное меню редактирования текста
	Cut       string
	Copy      string
	Paste     string
	SelectAll string

	// Поиск, выбор и списки
	SearchPlaceholder  string
	SelectPlaceholder  string
	SelectedSummaryFmt string // Формат вида "Выбрано: %d (%s)"
	SelectAllBtn       string
	ResetBtn           string

	// Календарь и время (DatePicker / TimePicker)
	Months      [12]string
	ShortMonths [12]string
	Weekdays    [7]string // Понедельник .. Воскресенье
	Today       string
	Now         string

	// Просмотрщик логов (LogView)
	LogFilterPlaceholder string
	LogCopy              string
	LogTail              string

	// Кнопки заголовка окна (TitleBar)
	MinimizeTooltip string
	MaximizeTooltip string
	RestoreTooltip  string
	CloseTooltip    string
}

// RussianStrings возвращает набор встроенных строк на русском языке (по умолчанию).
func RussianStrings() Strings {
	return Strings{
		Cut:                  "Вырезать",
		Copy:                 "Копировать",
		Paste:                "Вставить",
		SelectAll:            "Выделить всё",
		SearchPlaceholder:    "Поиск...",
		SelectPlaceholder:    "Выберите...",
		SelectedSummaryFmt:   "Выбрано: %d (%s)",
		SelectAllBtn:         "Выбрать все",
		ResetBtn:             "Сбросить",
		Months:               [12]string{"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь", "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"},
		ShortMonths:          [12]string{"Янв", "Фев", "Мар", "Апр", "Май", "Июн", "Июл", "Авг", "Сен", "Окт", "Ноя", "Дек"},
		Weekdays:             [7]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"},
		Today:                "Сегодня",
		Now:                  "Сейчас",
		LogFilterPlaceholder: "Фильтр логов...",
		LogCopy:              "Копировать",
		LogTail:              "Tail",
		MinimizeTooltip:      "Свернуть",
		MaximizeTooltip:      "Развернуть",
		RestoreTooltip:       "Восстановить",
		CloseTooltip:         "Закрыть",
	}
}

// EnglishStrings возвращает набор встроенных строк на английском языке.
func EnglishStrings() Strings {
	return Strings{
		Cut:                  "Cut",
		Copy:                 "Copy",
		Paste:                "Paste",
		SelectAll:            "Select All",
		SearchPlaceholder:    "Search...",
		SelectPlaceholder:    "Select...",
		SelectedSummaryFmt:   "Selected: %d (%s)",
		SelectAllBtn:         "Select All",
		ResetBtn:             "Reset",
		Months:               [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
		ShortMonths:          [12]string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
		Weekdays:             [7]string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"},
		Today:                "Today",
		Now:                  "Now",
		LogFilterPlaceholder: "Filter logs...",
		LogCopy:              "Copy",
		LogTail:              "Tail",
		MinimizeTooltip:      "Minimize",
		MaximizeTooltip:      "Maximize",
		RestoreTooltip:       "Restore",
		CloseTooltip:         "Close",
	}
}

func (c *Context) str() Strings {
	def := RussianStrings()
	s := c.Strings
	if s.Cut == "" {
		s.Cut = def.Cut
	}
	if s.Copy == "" {
		s.Copy = def.Copy
	}
	if s.Paste == "" {
		s.Paste = def.Paste
	}
	if s.SelectAll == "" {
		s.SelectAll = def.SelectAll
	}
	if s.SearchPlaceholder == "" {
		s.SearchPlaceholder = def.SearchPlaceholder
	}
	if s.SelectPlaceholder == "" {
		s.SelectPlaceholder = def.SelectPlaceholder
	}
	if s.SelectedSummaryFmt == "" {
		s.SelectedSummaryFmt = def.SelectedSummaryFmt
	}
	if s.SelectAllBtn == "" {
		s.SelectAllBtn = def.SelectAllBtn
	}
	if s.ResetBtn == "" {
		s.ResetBtn = def.ResetBtn
	}
	if s.Months[0] == "" {
		s.Months = def.Months
	}
	if s.ShortMonths[0] == "" {
		s.ShortMonths = def.ShortMonths
	}
	if s.Weekdays[0] == "" {
		s.Weekdays = def.Weekdays
	}
	if s.Today == "" {
		s.Today = def.Today
	}
	if s.Now == "" {
		s.Now = def.Now
	}
	if s.LogFilterPlaceholder == "" {
		s.LogFilterPlaceholder = def.LogFilterPlaceholder
	}
	if s.LogCopy == "" {
		s.LogCopy = def.LogCopy
	}
	if s.LogTail == "" {
		s.LogTail = def.LogTail
	}
	if s.MinimizeTooltip == "" {
		s.MinimizeTooltip = def.MinimizeTooltip
	}
	if s.MaximizeTooltip == "" {
		s.MaximizeTooltip = def.MaximizeTooltip
	}
	if s.RestoreTooltip == "" {
		s.RestoreTooltip = def.RestoreTooltip
	}
	if s.CloseTooltip == "" {
		s.CloseTooltip = def.CloseTooltip
	}
	return s
}

