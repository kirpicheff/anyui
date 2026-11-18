//go:build darwin

package platform

import "math"

// normalizeScale возвращает корректный коэффициент масштабирования Retina (>= 1.0).
func normalizeDarwinScale(scale float64) float32 {
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 1.0
	}
	return float32(scale)
}

// appKitPointToGoUIPixels переводит логические координаты AppKit (начало координат снизу-слева)
// в физические пиксели GoUI (начало координат сверху-слева).
func appKitPointToGoUIPixels(x, y, viewHeight, scale float64) (int, int) {
	s := float64(normalizeDarwinScale(scale))
	px := int(math.Round(x * s))
	py := int(math.Round((viewHeight - y) * s))
	return px, py
}

// appKitWindowRectToGoUI переводит координаты окна AppKit (origin снизу-слева экрана)
// в верхний левый угол окна в физических пикселях GoUI.
func appKitWindowRectToGoUI(x, y, winW, winH, screenH, scale float64) (gx, gy, gw, gh int) {
	s := float64(normalizeDarwinScale(scale))
	gx = int(math.Round(x * s))
	gy = int(math.Round((screenH - (y + winH)) * s))
	gw = int(math.Round(winW * s))
	gh = int(math.Round(winH * s))
	return gx, gy, gw, gh
}

// goUITopLeftYToAppKit переводит координату Y верхнего левого угла окна (в физических пикселях GoUI)
// в координату Y нижнего левого угла окна в системе AppKit (в логических точках).
func goUITopLeftYToAppKit(topY, winHeightPx int, screenHeightPts, scale float64) float64 {
	s := float64(normalizeDarwinScale(scale))
	topPts := float64(topY) / s
	winHPts := float64(winHeightPx) / s
	return screenHeightPts - topPts - winHPts
}

// darwinNormalizeWheel преобразует дельты прокрутки AppKit (scrollingDeltaX/Y)
// в значения WheelX (положительное — вправо) и WheelY (положительное — вверх) для GoUI.
func darwinNormalizeWheel(deltaX, deltaY float64, precise bool) (float32, float32) {
	wx := -deltaX
	wy := deltaY
	if precise {
		wx /= 10.0
		wy /= 10.0
	}
	return float32(wx), float32(wy)
}

