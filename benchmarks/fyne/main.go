package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func recordStartup(start time.Time) {
	if p := os.Getenv("BENCH_STARTUP_FILE"); p != "" {
		_ = os.WriteFile(p, []byte(fmt.Sprintf("%.2f", float64(time.Since(start).Microseconds())/1000.0)), 0644)
	}
}

func main() {
	start := time.Now()
	var once sync.Once

	a := app.New()
	a.Lifecycle().SetOnEnteredForeground(func() {
		once.Do(func() { recordStartup(start) })
	})

	w := a.NewWindow("Fyne Benchmark")
	w.Resize(fyne.NewSize(480, 320))

	clicks := 0
	lbl := widget.NewLabel("Hello, Fyne!")
	inp := widget.NewEntry()
	inp.SetPlaceHolder("Enter text")

	var btn *widget.Button
	btn = widget.NewButton("Clicks: 0", func() {
		clicks++
		btn.SetText(fmt.Sprintf("Clicks: %d", clicks))
	})

	w.SetContent(container.NewPadded(
		container.NewVBox(lbl, inp, btn),
	))
	w.ShowAndRun()
}
