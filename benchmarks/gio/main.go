package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func recordStartup(start time.Time) {
	if p := os.Getenv("BENCH_STARTUP_FILE"); p != "" {
		_ = os.WriteFile(p, []byte(fmt.Sprintf("%.2f", float64(time.Since(start).Microseconds())/1000.0)), 0644)
	}
}

func main() {
	start := time.Now()
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Gio Benchmark"), app.Size(unit.Dp(480), unit.Dp(320)))
		if err := run(w, start); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func run(w *app.Window, start time.Time) error {
	th := material.NewTheme()
	var ops op.Ops
	var editor widget.Editor
	editor.SingleLine = true
	var btn widget.Clickable
	var once sync.Once
	clicks := 0

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			if btn.Clicked(gtx) {
				clicks++
			}
			layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(material.H6(th, "Hello, Gio!").Layout),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					layout.Rigid(material.Editor(th, &editor, "Enter text").Layout),
					layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout),
					layout.Rigid(material.Button(th, &btn, fmt.Sprintf("Clicks: %d", clicks)).Layout),
				)
			})
			e.Frame(gtx.Ops)
			once.Do(func() { recordStartup(start) })
		}
	}
}
