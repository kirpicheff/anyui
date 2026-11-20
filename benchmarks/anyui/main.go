package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/kirpicheff/anyui"
)

func recordStartup(start time.Time) {
	if p := os.Getenv("BENCH_STARTUP_FILE"); p != "" {
		_ = os.WriteFile(p, []byte(fmt.Sprintf("%.2f", float64(time.Since(start).Microseconds())/1000.0)), 0644)
	}
}

func main() {
	start := time.Now()
	var once sync.Once

	app := anyui.NewApp(anyui.Config{
		Title:  "AnyUI Benchmark",
		Width:  480,
		Height: 320,
		Theme:  anyui.NavyTheme(),
	})

	text := ""
	clicks := 0

	err := app.Run(func(ui *anyui.UI) {
		ui.Padding(20, func() {
			ui.Column(12, func() {
				ui.Label("Hello, AnyUI!", anyui.FontSize(18), anyui.Bold())
				ui.TextInput("inp", &text, "Enter text", anyui.Width(260))
				if ui.Button("btn", fmt.Sprintf("Clicks: %d", clicks), anyui.Primary()) {
					clicks++
				}
			})
		})
		once.Do(func() { recordStartup(start) })
	})
	if err != nil {
		log.Fatal(err)
	}
}
