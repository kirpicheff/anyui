//go:build darwin

package platform

// Константы NSEventModifierFlags из AppKit (64-битные маски).
const (
	nsEventModifierFlagShift   uint64 = 1 << 17
	nsEventModifierFlagControl uint64 = 1 << 18
	nsEventModifierFlagOption  uint64 = 1 << 19
	nsEventModifierFlagCommand uint64 = 1 << 20
)

// darwinModifiersFromFlags преобразует NSEventModifierFlags в Modifiers.
// Command (Cmd) и Control маппятся в Ctrl, Option — в Alt, Shift — в Shift.
func darwinModifiersFromFlags(flags uint64) Modifiers {
	return Modifiers{
		Shift: (flags & nsEventModifierFlagShift) != 0,
		Ctrl:  (flags&(nsEventModifierFlagControl|nsEventModifierFlagCommand)) != 0,
		Alt:   (flags & nsEventModifierFlagOption) != 0,
	}
}

// Таблица соответствия аппаратных кодов клавиш macOS (HIToolbox/Events.h kVK_*)
// и виртуальных кодов клавиш VK_*, используемых в GoUI.
var darwinKeycodeTable = map[uint16]uint32{
	// Буквы (ANSI раскладка)
	0x00: 'A',
	0x0B: 'B',
	0x08: 'C',
	0x02: 'D',
	0x0E: 'E',
	0x03: 'F',
	0x05: 'G',
	0x04: 'H',
	0x22: 'I',
	0x26: 'J',
	0x28: 'K',
	0x25: 'L',
	0x2E: 'M',
	0x2D: 'N',
	0x1F: 'O',
	0x23: 'P',
	0x0C: 'Q',
	0x0F: 'R',
	0x01: 'S',
	0x11: 'T',
	0x20: 'U',
	0x09: 'V',
	0x0D: 'W',
	0x07: 'X',
	0x10: 'Y',
	0x06: 'Z',

	// Цифры верхнего ряда
	0x1D: '0',
	0x12: '1',
	0x13: '2',
	0x14: '3',
	0x15: '4',
	0x17: '5',
	0x16: '6',
	0x1A: '7',
	0x1C: '8',
	0x19: '9',

	// Управляющие и навигационные клавиши
	0x33: 0x08, // Delete (Backspace)
	0x30: 0x09, // Tab
	0x24: 0x0D, // Return
	0x4C: 0x0D, // Keypad Enter
	0x35: 0x1B, // Escape
	0x31: 0x20, // Space
	0x74: 0x21, // Page Up
	0x79: 0x22, // Page Down
	0x77: 0x23, // End
	0x73: 0x24, // Home
	0x7B: 0x25, // Left Arrow
	0x7E: 0x26, // Up Arrow
	0x7C: 0x27, // Right Arrow
	0x7D: 0x28, // Down Arrow
	0x72: 0x2D, // Help / Insert
	0x75: 0x2E, // Forward Delete

	// Функциональные клавиши F1..F12
	0x7A: 0x70, // F1
	0x78: 0x71, // F2
	0x63: 0x72, // F3
	0x76: 0x73, // F4
	0x60: 0x74, // F5
	0x61: 0x75, // F6
	0x62: 0x76, // F7
	0x64: 0x77, // F8
	0x65: 0x78, // F9
	0x6D: 0x79, // F10
	0x67: 0x7A, // F11
	0x6F: 0x7B, // F12
}

// darwinKeycodeToVK преобразует аппаратный keycode macOS в виртуальный код клавиши GoUI.
func darwinKeycodeToVK(keyCode uint16) uint32 {
	return darwinKeycodeTable[keyCode]
}

