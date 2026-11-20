package anyui

import (
	"testing"

	"github.com/kirpicheff/anyui/platform"
	"github.com/kirpicheff/anyui/ui"
)

// mockClipWindow реализует platform.Window для проверки буфера обмена.
type mockClipWindow struct {
	w, h     int
	scale    float32
	clipText string
}

func (m *mockClipWindow) Run(func(platform.Event)) error          { return nil }
func (m *mockClipWindow) Open(func(platform.Event)) error         { return nil }
func (m *mockClipWindow) Present([]byte, int, int)                {}
func (m *mockClipWindow) PresentRect([]byte, int, int, int, int, int, int) {}
func (m *mockClipWindow) Invalidate()                             {}
func (m *mockClipWindow) WakeUp()                                 {}
func (m *mockClipWindow) Show()                                   {}
func (m *mockClipWindow) Hide()                                   {}
func (m *mockClipWindow) Close()                                  {}
func (m *mockClipWindow) SetTitle(string)                         {}
func (m *mockClipWindow) SetCursor(platform.CursorType)           {}
func (m *mockClipWindow) Size() (int, int)                        { return m.w, m.h }
func (m *mockClipWindow) Scale() float32                          { return m.scale }
func (m *mockClipWindow) ClipboardText() (string, error)          { return m.clipText, nil }
func (m *mockClipWindow) SetClipboardText(s string) error         { m.clipText = s; return nil }
func (m *mockClipWindow) Minimize()                               {}
func (m *mockClipWindow) Maximize()                               {}
func (m *mockClipWindow) Restore()                                {}
func (m *mockClipWindow) IsMaximized() bool                       { return false }
func (m *mockClipWindow) IsMinimized() bool                       { return false }
func (m *mockClipWindow) SetFullscreen(bool)                      {}
func (m *mockClipWindow) IsFullscreen() bool                      { return false }
func (m *mockClipWindow) Position() (int, int)                    { return 0, 0 }
func (m *mockClipWindow) SetPosition(int, int)                    {}
func (m *mockClipWindow) SetSize(int, int)                        {}
func (m *mockClipWindow) SetOnClose(func() bool)                  {}
func (m *mockClipWindow) NativeHandle() uintptr                   { return 0 }
func (m *mockClipWindow) SetCapture()                             {}
func (m *mockClipWindow) ReleaseCapture()                         {}
func (m *mockClipWindow) DoubleClickTimeMs() int                  { return 500 }

func TestClipboardWindowAndUI(t *testing.T) {
	mw := &mockClipWindow{w: 800, h: 600, scale: 1.0}
	ctx := ui.NewContext(mw, nil)
	win := &Window{platWin: mw, ctx: ctx}
	u := &UI{ctx: ctx, win: win}

	// Проверка через Window
	if err := win.SetClipboardText("test-window"); err != nil {
		t.Fatalf("SetClipboardText failed on Window: %v", err)
	}
	got, err := win.ClipboardText()
	if err != nil || got != "test-window" {
		t.Fatalf("ClipboardText on Window: got %q, err %v", got, err)
	}

	// Проверка через UI
	if err := u.SetClipboardText("test-ui"); err != nil {
		t.Fatalf("SetClipboardText failed on UI: %v", err)
	}
	gotUI, err := u.ClipboardText()
	if err != nil || gotUI != "test-ui" {
		t.Fatalf("ClipboardText on UI: got %q, err %v", gotUI, err)
	}

	// Проверка на неинициализированном окне
	var nilWin *Window
	if _, err := nilWin.ClipboardText(); err == nil {
		t.Fatal("expected error on nil Window.ClipboardText")
	}
	if err := nilWin.SetClipboardText("fail"); err == nil {
		t.Fatal("expected error on nil Window.SetClipboardText")
	}

	var nilUI UI
	if _, err := nilUI.ClipboardText(); err == nil {
		t.Fatal("expected error on uninitialized UI.ClipboardText")
	}
	if err := nilUI.SetClipboardText("fail"); err == nil {
		t.Fatal("expected error on uninitialized UI.SetClipboardText")
	}
}

