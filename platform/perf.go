package platform

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Внутренние счётчики производительности. Включаются переменной окружения ANYUI_PERF=1,
// раз в секунду печатают сводку в stderr. При выключенном флаге стоимость - одна проверка bool.

var perfOn = os.Getenv("ANYUI_PERF") == "1"

type perfStat struct {
	count   atomic.Int64
	totalNs atomic.Int64
	maxNs   atomic.Int64
}

var (
	perfStats     sync.Map // имя -> *perfStat
	perfStartOnce sync.Once
)

// PerfEnabled сообщает, включены ли счётчики производительности.
func PerfEnabled() bool { return perfOn }

// PerfNow возвращает текущее время, если счётчики включены, иначе нулевое время.
func PerfNow() time.Time {
	if !perfOn {
		return time.Time{}
	}
	return time.Now()
}

// PerfSince записывает длительность с момента start; при выключенных счётчиках или нулевом start ничего не делает.
func PerfSince(name string, start time.Time) {
	if !perfOn || start.IsZero() {
		return
	}
	PerfRecord(name, time.Since(start))
}

// PerfCount увеличивает счётчик событий без длительности.
func PerfCount(name string) {
	if !perfOn {
		return
	}
	perfGet(name).count.Add(1)
}

// PerfRecord записывает одно измерение длительности.
func PerfRecord(name string, d time.Duration) {
	if !perfOn {
		return
	}
	s := perfGet(name)
	ns := d.Nanoseconds()
	s.count.Add(1)
	s.totalNs.Add(ns)
	for {
		cur := s.maxNs.Load()
		if ns <= cur || s.maxNs.CompareAndSwap(cur, ns) {
			break
		}
	}
}

func perfGet(name string) *perfStat {
	perfStartOnce.Do(func() { go perfReportLoop() })
	if v, ok := perfStats.Load(name); ok {
		return v.(*perfStat)
	}
	v, _ := perfStats.LoadOrStore(name, &perfStat{})
	return v.(*perfStat)
}

func perfReportLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		var lines []string
		perfStats.Range(func(k, v any) bool {
			s := v.(*perfStat)
			n := s.count.Swap(0)
			total := s.totalNs.Swap(0)
			mx := s.maxNs.Swap(0)
			if n == 0 {
				return true
			}
			if total == 0 {
				lines = append(lines, fmt.Sprintf("  %-28s n=%d", k.(string), n))
			} else {
				lines = append(lines, fmt.Sprintf("  %-28s n=%d avg=%.2fms max=%.2fms",
					k.(string), n, float64(total)/float64(n)/1e6, float64(mx)/1e6))
			}
			return true
		})
		if len(lines) == 0 {
			continue
		}
		sort.Strings(lines)
		if perfFile != nil {
			// Файловый режим: префикс pid, чтобы различать процессы (лаунчер и вьювер).
			ts := time.Now().Format("15:04:05")
			for _, l := range lines {
				fmt.Fprintf(perfFile, "pid=%d %s %s\n", os.Getpid(), ts, l)
			}
			continue
		}
		fmt.Fprintf(os.Stderr, "[anyui-perf]\n%s\n", strings.Join(lines, "\n"))
	}
}

// perfFile - необязательный файл для сводки (ANYUI_PERF_FILE), открывается в режиме дозаписи.
// Нужен, когда у процесса нет консоли или stderr не наследуется (например, вьювер, запущенный лаунчером).
var perfFile = func() *os.File {
	path := os.Getenv("ANYUI_PERF_FILE")
	if !perfOn || path == "" {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil
	}
	return f
}()

