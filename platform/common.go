package platform

import (
	"errors"
	"strings"
)

// validateOpenURL проверяет, что URL использует безопасную схему (http, https или mailto).
func validateOpenURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return errors.New("пустой URL")
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] < 0x20 || trimmed[i] == 0x7f {
			return errors.New("URL содержит недопустимые управляющие символы")
		}
	}
	colon := strings.IndexByte(trimmed, ':')
	if colon <= 0 {
		return errors.New("некорректный формат URL")
	}
	scheme := strings.ToLower(trimmed[:colon])
	rest := trimmed[colon+1:]
	switch scheme {
	case "http", "https":
		if !strings.HasPrefix(rest, "//") || len(rest) <= 2 {
			return errors.New("некорректный HTTP/HTTPS URL")
		}
		return nil
	case "mailto":
		if rest == "" {
			return errors.New("пустой адрес mailto")
		}
		return nil
	default:
		return errors.New("неподдерживаемая схема URL: разрешены только http, https и mailto")
	}
}

// isValidWindowState проверяет корректность сохранённой геометрии окна.
func isValidWindowState(s WindowState) bool {
	return s.Width > 0 && s.Height > 0
}

// ComputeFramelessHitTest вычисляет результат hit-test для безрамочного окна.
func ComputeFramelessHitTest(cx, cy, width, height, borderPx, titleBarPx int, maximized bool, customHit func(x, y int) HitTestResult) HitTestResult {
	if !maximized && borderPx > 0 {
		onLeft := cx >= 0 && cx < borderPx
		onRight := cx >= width-borderPx && cx < width
		onTop := cy >= 0 && cy < borderPx
		onBottom := cy >= height-borderPx && cy < height

		switch {
		case onTop && onLeft:
			return HitTopLeft
		case onTop && onRight:
			return HitTopRight
		case onBottom && onLeft:
			return HitBottomLeft
		case onBottom && onRight:
			return HitBottomRight
		case onLeft:
			return HitLeft
		case onRight:
			return HitRight
		case onTop:
			return HitTop
		case onBottom:
			return HitBottom
		}
	}

	if customHit != nil {
		return customHit(cx, cy)
	}
	if titleBarPx > 0 && cy >= 0 && cy < titleBarPx {
		return HitCaption
	}
	return HitClient
}
