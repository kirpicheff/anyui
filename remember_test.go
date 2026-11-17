package anyui

import (
	"sync/atomic"
	"testing"
	"time"

	"git.gtw.su/kirpich/anyui/platform"
	"git.gtw.su/kirpich/anyui/ui"
)

type stubWindow struct {
	platform.Window
}

func (stubWindow) WakeUp()         {}
func (stubWindow) Size() (int, int) { return 320, 240 }
func (stubWindow) Scale() float32  { return 1.0 }

func TestRememberAndInvalidate(t *testing.T) {
	ctx := ui.NewContext(stubWindow{}, nil)
	u := &UI{ctx: ctx}

	var calls atomic.Int32
	done := make(chan struct{}, 2)

	val, loading, err := Remember(u, "k1", func() (string, error) {
		n := calls.Add(1)
		done <- struct{}{}
		if n == 1 {
			return "first", nil
		}
		return "second", nil
	})
	if !loading || err != nil || val != "" {
		t.Fatalf("ожидалось состояние загрузки на первом вызове, получено val=%q loading=%v err=%v", val, loading, err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("таймаут ожидания фоновой загрузки Remember")
	}

	// Небольшое ожидание завершения RememberFinish после сигнала из loader
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		val, loading, err = Remember(u, "k1", func() (string, error) {
			calls.Add(1)
			return "unexpected", nil
		})
		if !loading {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if loading || err != nil || val != "first" {
		t.Fatalf("ожидался кэшированный результат 'first', получено val=%q loading=%v err=%v", val, loading, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("ожидался 1 вызов loader, получено %d", calls.Load())
	}

	// Сброс кэша и повторная загрузка
	u.InvalidateRemember("k1")
	_, loading, _ = Remember(u, "k1", func() (string, error) {
		calls.Add(1)
		done <- struct{}{}
		return "second", nil
	})
	if !loading {
		t.Fatal("после InvalidateRemember ожидалась повторная загрузка")
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("таймаут ожидания повторной загрузки Remember")
	}
}
