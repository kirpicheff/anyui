package anyui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/ui"
)

func TestWindowDragDropEvents(t *testing.T) {
	app := NewApp(Config{})
	mw := &stylerMockWindow{w: 400, h: 300, scale: 1.0}
	ctx := ui.NewContext(mw, nil)

	w := &Window{
		app:     app,
		platWin: mw,
		ctx:     ctx,
	}

	var (
		enterCalled bool
		enterX, enterY int
		enterHasFiles  bool

		overCalled bool
		overX, overY int
		overHasFiles  bool

		leaveCalled bool

		dropCalled bool
		dropFiles  []string
		dropX, dropY int
	)

	w.SetOnDragEnter(func(xPx, yPx int, hasFiles bool) {
		enterCalled = true
		enterX = xPx
		enterY = yPx
		enterHasFiles = hasFiles
	})

	w.SetOnDragOver(func(xPx, yPx int, hasFiles bool) {
		overCalled = true
		overX = xPx
		overY = yPx
		overHasFiles = hasFiles
	})

	w.SetOnDragLeave(func() {
		leaveCalled = true
	})

	w.SetOnDropFiles(func(files []string, xPx, yPx int) {
		dropCalled = true
		dropFiles = files
		dropX = xPx
		dropY = yPx
	})

	// 1. DragEnter
	w.handleEvent(platform.Event{
		Type:     platform.EventDragEnter,
		X:        50,
		Y:        60,
		HasFiles: true,
	})
	if !enterCalled || enterX != 50 || enterY != 60 || !enterHasFiles {
		t.Fatalf("ошибка в SetOnDragEnter: called=%v, x=%d, y=%d, hasFiles=%v", enterCalled, enterX, enterY, enterHasFiles)
	}

	// 2. DragOver
	w.handleEvent(platform.Event{
		Type:     platform.EventDragOver,
		X:        70,
		Y:        80,
		HasFiles: true,
	})
	if !overCalled || overX != 70 || overY != 80 || !overHasFiles {
		t.Fatalf("ошибка в SetOnDragOver: called=%v, x=%d, y=%d, hasFiles=%v", overCalled, overX, overY, overHasFiles)
	}

	// 3. DragLeave
	w.handleEvent(platform.Event{
		Type: platform.EventDragLeave,
	})
	if !leaveCalled {
		t.Fatalf("ошибка в SetOnDragLeave: не был вызван")
	}

	// 4. DropFiles
	w.handleEvent(platform.Event{
		Type:  platform.EventDropFiles,
		X:     90,
		Y:     100,
		Files: []string{"C:\\file1.txt", "C:\\file2.txt"},
	})
	if !dropCalled || dropX != 90 || dropY != 100 || len(dropFiles) != 2 || dropFiles[0] != "C:\\file1.txt" {
		t.Fatalf("ошибка в SetOnDropFiles: called=%v, files=%v, x=%d, y=%d", dropCalled, dropFiles, dropX, dropY)
	}
}
