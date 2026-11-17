package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var assets embed.FS

func recordStartup(start time.Time) {
	if p := os.Getenv("BENCH_STARTUP_FILE"); p != "" {
		_ = os.WriteFile(p, []byte(fmt.Sprintf("%.2f", float64(time.Since(start).Microseconds())/1000.0)), 0644)
	}
}

func main() {
	start := time.Now()
	var once sync.Once

	err := wails.Run(&options.App{
		Title:  "Wails Benchmark",
		Width:  480,
		Height: 320,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnDomReady: func(ctx context.Context) {
			once.Do(func() { recordStartup(start) })
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
