package ui

import (
	"math"
	"time"
)

type animEntry struct {
	value  float32
	target float32
}

// Animator управляет плавной интерполяцией float32-значений по строковому идентификатору.
type Animator struct {
	items     map[string]*animEntry
	lastTime  time.Time
	dt        float32
	hasActive bool
}

// NewAnimator создаёт новый экземпляр движка анимаций.
func NewAnimator() *Animator {
	return &Animator{
		items: make(map[string]*animEntry),
	}
}

// BeginFrame вычисляет дельту времени dt для текущего кадра.
func (a *Animator) BeginFrame(now time.Time) float32 {
	if a.lastTime.IsZero() {
		a.dt = 1.0 / 60.0
	} else {
		dt := float32(now.Sub(a.lastTime).Seconds())
		if dt < 0 {
			dt = 0
		}
		if dt > 0.05 {
			dt = 0.05
		}
		a.dt = dt
	}
	a.lastTime = now
	a.hasActive = false
	return a.dt
}

// ResetClock сбрасывает отсчёт времени перед выходом из режима простоя.
func (a *Animator) ResetClock() {
	a.lastTime = time.Time{}
}

// Float возвращает текущее плавно интерполированное значение для ключа id.
// При первом обращении значение инициализируется сразу в target.
func (a *Animator) Float(id string, target float32, durationSec float32) float32 {
	entry, ok := a.items[id]
	if !ok {
		a.items[id] = &animEntry{
			value:  target,
			target: target,
		}
		return target
	}

	entry.target = target
	diff := target - entry.value
	if math.Abs(float64(diff)) < 0.001 {
		entry.value = target
		return target
	}

	if durationSec <= 0 {
		entry.value = target
		return target
	}

	// Экспоненциальное сглаживание: плавный старт и мягкое замедление у цели
	k := float32(1.0 - math.Exp(float64(-a.dt*6.0/durationSec)))
	if k > 1 {
		k = 1
	}
	if k < 0 {
		k = 0
	}
	entry.value += diff * k

	if math.Abs(float64(target-entry.value)) < 0.001 {
		entry.value = target
	} else {
		a.hasActive = true
	}

	return entry.value
}

// MarkActive помечает, что в текущем кадре требуется продолжать цикл анимации.
func (a *Animator) MarkActive() {
	a.hasActive = true
}

// Delta возвращает длительность последнего кадра в секундах.
func (a *Animator) Delta() float32 {
	return a.dt
}

// HasActive возвращает true, если в текущем кадре есть незавершённые анимации.
func (a *Animator) HasActive() bool {
	return a.hasActive
}

