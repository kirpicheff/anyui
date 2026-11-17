package ui

// RememberEntry хранит состояние кешированного асинхронного значения для Remember.
type RememberEntry struct {
	Value   any
	Err     error
	Loading bool
	Loaded  bool
	Gen     uint64
}

// RememberKey преобразует произвольный ключ в строковый идентификатор без паники на несравнимых типах.
func RememberKey(key any) string {
	return formatIDKey(key)
}

// RememberGet возвращает текущее состояние кеша по строковому ключу.
func (c *Context) RememberGet(key string) (val any, err error, loading bool, ok bool) {
	c.rememberMu.Lock()
	defer c.rememberMu.Unlock()
	if c.rememberStore == nil {
		return nil, nil, false, false
	}
	e, exists := c.rememberStore[key]
	if !exists {
		return nil, nil, false, false
	}
	return e.Value, e.Err, e.Loading, true
}

// RememberStart проверяет необходимость запуска загрузки и переводит запись в состояние Loading.
func (c *Context) RememberStart(key string) (gen uint64, shouldStart bool) {
	c.rememberMu.Lock()
	defer c.rememberMu.Unlock()
	if c.rememberStore == nil {
		c.rememberStore = make(map[string]*RememberEntry)
	}
	e, exists := c.rememberStore[key]
	if exists && (e.Loaded || e.Loading) {
		return e.Gen, false
	}
	if !exists {
		e = &RememberEntry{}
		c.rememberStore[key] = e
	}
	e.Gen++
	e.Loading = true
	e.Loaded = false
	e.Err = nil
	return e.Gen, true
}

// RememberFinish сохраняет результат загрузки, если запись не была сброшена за время выполнения.
func (c *Context) RememberFinish(key string, gen uint64, val any, err error) {
	c.rememberMu.Lock()
	if c.rememberStore != nil {
		if e, exists := c.rememberStore[key]; exists && e.Gen == gen {
			e.Value = val
			e.Err = err
			e.Loading = false
			e.Loaded = true
		}
	}
	c.rememberMu.Unlock()
	c.RequestRedraw()
}

// InvalidateRemember сбрасывает кешированное значение по ключу и запрашивает перерисовку.
func (c *Context) InvalidateRemember(key any) {
	k := formatIDKey(key)
	c.rememberMu.Lock()
	if c.rememberStore != nil {
		delete(c.rememberStore, k)
	}
	c.rememberMu.Unlock()
	c.RequestRedraw()
}

// ClearRemember полностью очищает кеш Remember текущего контекста и запрашивает перерисовку.
func (c *Context) ClearRemember() {
	c.rememberMu.Lock()
	c.rememberStore = nil
	c.rememberMu.Unlock()
	c.RequestRedraw()
}
