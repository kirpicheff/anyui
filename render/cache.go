package render

import (
	"container/list"
	"sync"
)

type lruEntry[K comparable, V any] struct {
	key       K
	val       V
	sizeBytes int64
}

// lruCache реализует потокобезопасный LRU-кеш с ограничением по количеству записей и объёму памяти.
type lruCache[K comparable, V any] struct {
	mu         sync.Mutex
	maxEntries int
	maxBytes   int64
	curBytes   int64
	items      map[K]*list.Element
	order      *list.List
}

func newLRUCache[K comparable, V any](maxEntries int, maxBytes int64) *lruCache[K, V] {
	return &lruCache[K, V]{
		maxEntries: maxEntries,
		maxBytes:   maxBytes,
		items:      make(map[K]*list.Element),
		order:      list.New(),
	}
}

// Get возвращает значение по ключу и перемещает запись в начало списка недавних обращений.
func (c *lruCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*lruEntry[K, V]).val, true
	}
	var zero V
	return zero, false
}

// Put добавляет или обновляет запись в кеше с учётом лимитов по количеству и памяти.
func (c *lruCache[K, V]) Put(key K, val V, sizeBytes int64) {
	if sizeBytes < 0 {
		sizeBytes = 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxEntries == 0 {
		return
	}
	if c.maxBytes > 0 && sizeBytes > c.maxBytes {
		return
	}

	if el, ok := c.items[key]; ok {
		entry := el.Value.(*lruEntry[K, V])
		c.curBytes -= entry.sizeBytes
		entry.val = val
		entry.sizeBytes = sizeBytes
		c.curBytes += sizeBytes
		c.order.MoveToFront(el)
		c.evictLocked()
		return
	}

	entry := &lruEntry[K, V]{
		key:       key,
		val:       val,
		sizeBytes: sizeBytes,
	}
	el := c.order.PushFront(entry)
	c.items[key] = el
	c.curBytes += sizeBytes
	c.evictLocked()
}

// Clear полностью очищает кеш.
func (c *lruCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[K]*list.Element)
	c.order.Init()
	c.curBytes = 0
}

// SetLimits изменяет лимиты кеша и при необходимости вытесняет старые записи.
func (c *lruCache[K, V]) SetLimits(maxEntries int, maxBytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.maxEntries = maxEntries
	c.maxBytes = maxBytes
	c.evictLocked()
}

// Stats возвращает текущее число записей и суммарный размер в байтах.
func (c *lruCache[K, V]) Stats() (entries int, bytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items), c.curBytes
}

func (c *lruCache[K, V]) evictLocked() {
	for c.order.Len() > 0 {
		overCount := c.maxEntries > 0 && len(c.items) > c.maxEntries
		overBytes := c.maxBytes > 0 && c.curBytes > c.maxBytes
		if !overCount && !overBytes {
			break
		}
		tail := c.order.Back()
		if tail == nil {
			break
		}
		entry := c.order.Remove(tail).(*lruEntry[K, V])
		delete(c.items, entry.key)
		c.curBytes -= entry.sizeBytes
	}
}

