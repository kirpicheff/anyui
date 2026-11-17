package ui

import "git.gtw.su/kirpich/anyui/render"

// Theme хранит цвета и базовые параметры оформления интерфейса.
type Theme struct {
	Bg          render.Color
	Surface     render.Color
	SurfaceDark render.Color
	Card        render.Color
	Border      render.Color
	BorderFocus render.Color

	TextPrimary render.Color
	TextMuted   render.Color

	Accent      render.Color
	AccentHover render.Color
	AccentDown  render.Color

	BtnSecondary      render.Color
	BtnSecondaryHover render.Color

	Success render.Color
	Warning render.Color
	Danger  render.Color

	RadiusSm float32
	RadiusMd float32
	RadiusLg float32
}

// DeepNavyTheme возвращает стандартную тёмную тему в стиле AnyPort Deep Navy.
func DeepNavyTheme() Theme {
	return Theme{
		Bg:          render.Hex(0x0F172A),
		Surface:     render.Hex(0x162033),
		SurfaceDark: render.Hex(0x0B1120),
		Card:        render.Hex(0x1E293B),
		Border:      render.Hex(0x334155),
		BorderFocus: render.Hex(0x3B82F6),

		TextPrimary: render.Hex(0xF8FAFC),
		TextMuted:   render.Hex(0x94A3B8),

		Accent:      render.Hex(0x3B82F6),
		AccentHover: render.Hex(0x2563EB),
		AccentDown:  render.Hex(0x1D4ED8),

		BtnSecondary:      render.Hex(0x1E293B),
		BtnSecondaryHover: render.Hex(0x283548),

		Success: render.Hex(0x22C55E),
		Warning: render.Hex(0xF59E0B),
		Danger:  render.Hex(0xEF4444),

		RadiusSm: 8,
		RadiusMd: 12,
		RadiusLg: 16,
	}
}

// DarkTheme возвращает нейтральную тёмную тему.
func DarkTheme() Theme {
	return Theme{
		Bg:          render.Hex(0x18181B),
		Surface:     render.Hex(0x27272A),
		SurfaceDark: render.Hex(0x111113),
		Card:        render.Hex(0x202024),
		Border:      render.Hex(0x3F3F46),
		BorderFocus: render.Hex(0x60A5FA),

		TextPrimary: render.Hex(0xFAFAFA),
		TextMuted:   render.Hex(0xA1A1AA),

		Accent:      render.Hex(0x3B82F6),
		AccentHover: render.Hex(0x2563EB),
		AccentDown:  render.Hex(0x1D4ED8),

		BtnSecondary:      render.Hex(0x27272A),
		BtnSecondaryHover: render.Hex(0x323238),

		Success: render.Hex(0x22C55E),
		Warning: render.Hex(0xF59E0B),
		Danger:  render.Hex(0xEF4444),

		RadiusSm: 8,
		RadiusMd: 12,
		RadiusLg: 16,
	}
}

// LightTheme возвращает чистую светлую тему.
func LightTheme() Theme {
	return Theme{
		Bg:          render.Hex(0xF8FAFC),
		Surface:     render.Hex(0xFFFFFF),
		SurfaceDark: render.Hex(0xF1F5F9),
		Card:        render.Hex(0xFFFFFF),
		Border:      render.Hex(0xCBD5E1),
		BorderFocus: render.Hex(0x2563EB),

		TextPrimary: render.Hex(0x0F172A),
		TextMuted:   render.Hex(0x64748B),

		Accent:      render.Hex(0x2563EB),
		AccentHover: render.Hex(0x1D4ED8),
		AccentDown:  render.Hex(0x1E40AF),

		BtnSecondary:      render.Hex(0xE2E8F0),
		BtnSecondaryHover: render.Hex(0xCBD5E1),

		Success: render.Hex(0x16A34A),
		Warning: render.Hex(0xD97706),
		Danger:  render.Hex(0xDC2626),

		RadiusSm: 8,
		RadiusMd: 12,
		RadiusLg: 16,
	}
}

