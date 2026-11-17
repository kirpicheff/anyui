//go:build windows

package platform

import (
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

func TestColorToCOLORREF(t *testing.T) {
	got := colorToCOLORREF(WindowColor{R: 0x12, G: 0x34, B: 0x56})
	want := uint32(0x00563412)
	if got != want {
		t.Fatalf("ожидалось 0x%08X, получено 0x%08X", want, got)
	}
}

func TestEncodeDecodeDIB32And24(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 10, B: 20, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{R: 30, G: 200, B: 40, A: 128})
	src.SetNRGBA(2, 1, color.NRGBA{R: 50, G: 60, B: 220, A: 255})

	dib, err := encodeImageToDIB(src)
	if err != nil {
		t.Fatalf("encodeImageToDIB вернул ошибку: %v", err)
	}
	decoded, err := decodeDIBToImage(dib)
	if err != nil {
		t.Fatalf("decodeDIBToImage вернул ошибку: %v", err)
	}

	if decoded.Bounds().Dx() != 3 || decoded.Bounds().Dy() != 2 {
		t.Fatalf("неверные размеры декодированного изображения: %v", decoded.Bounds())
	}
	c0 := color.NRGBAModel.Convert(decoded.At(0, 0)).(color.NRGBA)
	if c0 != (color.NRGBA{R: 255, G: 10, B: 20, A: 255}) {
		t.Fatalf("пиксель (0,0) не совпал: %+v", c0)
	}
	c1 := color.NRGBAModel.Convert(decoded.At(1, 0)).(color.NRGBA)
	if c1 != (color.NRGBA{R: 30, G: 200, B: 40, A: 128}) {
		t.Fatalf("пиксель (1,0) не совпал: %+v", c1)
	}
	c2 := color.NRGBAModel.Convert(decoded.At(2, 1)).(color.NRGBA)
	if c2 != (color.NRGBA{R: 50, G: 60, B: 220, A: 255}) {
		t.Fatalf("пиксель (2,1) не совпал: %+v", c2)
	}

	// Проверка 24-битного top-down DIB шириной 1 (выравнивание строки до 4 байт)
	dib24 := make([]byte, dibHeaderSize+4)
	binary.LittleEndian.PutUint32(dib24[0:4], dibHeaderSize)
	binary.LittleEndian.PutUint32(dib24[4:8], 1)
	binary.LittleEndian.PutUint32(dib24[8:12], ^uint32(0)) // -1 (top-down)
	binary.LittleEndian.PutUint16(dib24[12:14], 1)
	binary.LittleEndian.PutUint16(dib24[14:16], 24)
	dib24[dibHeaderSize+0] = 11 // B
	dib24[dibHeaderSize+1] = 22 // G
	dib24[dibHeaderSize+2] = 33 // R

	img24, err := decodeDIBToImage(dib24)
	if err != nil {
		t.Fatalf("ошибка декодирования 24bpp DIB: %v", err)
	}
	px := color.NRGBAModel.Convert(img24.At(0, 0)).(color.NRGBA)
	if px != (color.NRGBA{R: 33, G: 22, B: 11, A: 255}) {
		t.Fatalf("неверный пиксель 24bpp DIB: %+v", px)
	}
}

func TestValidateOpenURL(t *testing.T) {
	valid := []string{
		"https://example.com/docs",
		"http://localhost:8080/health",
		"mailto:support@example.com",
	}
	for _, u := range valid {
		if err := validateOpenURL(u); err != nil {
			t.Fatalf("ожидался валидный URL %q, получена ошибка: %v", u, err)
		}
	}

	invalid := []string{
		"",
		"   ",
		"file:///C:/Windows/System32/cmd.exe",
		"javascript:alert(1)",
		"https://example.com/\r\nX-Inject: 1",
		"https://",
		"mailto:",
		"cmd.exe",
	}
	for _, u := range invalid {
		if err := validateOpenURL(u); err == nil {
			t.Fatalf("ожидалась ошибка для небезопасного URL %q", u)
		}
	}
}

func TestEncodeDecodeInstanceArgs(t *testing.T) {
	// Пустой список
	emptyRaw, err := encodeInstanceArgs(nil)
	if err != nil {
		t.Fatalf("encodeInstanceArgs(nil): %v", err)
	}
	emptyGot, err := decodeInstanceArgs(emptyRaw)
	if err != nil || len(emptyGot) != 0 {
		t.Fatalf("ожидался пустой срез, получено %v (err=%v)", emptyGot, err)
	}
	// Защита от усечённых данных
	if _, err := decodeInstanceArgs([]byte{1, 0, 0, 0, 10, 0, 0, 0, 'a'}); err == nil {
		t.Fatal("ожидалась ошибка при усечённом буфере IPC")
	}

	orig := []string{"--open", `C:\Users\Тест\файл с пробелами.txt`, "--flag=1"}
	raw, err := encodeInstanceArgs(orig)
	if err != nil {
		t.Fatalf("encodeInstanceArgs: %v", err)
	}
	got, err := decodeInstanceArgs(raw)
	if err != nil {
		t.Fatalf("decodeInstanceArgs: %v", err)
	}
	if len(got) != len(orig) {
		t.Fatalf("ожидалось %d аргументов, получено %d", len(orig), len(got))
	}
	for i := range orig {
		if got[i] != orig[i] {
			t.Fatalf("аргумент %d: ожидалось %q, получено %q", i, orig[i], got[i])
		}
	}
}

func TestIsValidWindowState(t *testing.T) {
	if isValidWindowState(WindowState{Width: 0, Height: 600}) {
		t.Fatal("нулевая ширина не должна считаться валидной")
	}
	if !isValidWindowState(WindowState{X: 100, Y: 50, Width: 800, Height: 600, Maximized: true}) {
		t.Fatal("корректная геометрия должна быть валидной")
	}
}

