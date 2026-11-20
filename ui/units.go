package ui

// Dp задаёт логические аппаратно-независимые пиксели (Density-independent Pixels).
type Dp int

// Int возвращает целочисленное значение Dp.
func (d Dp) Int() int {
	return int(d)
}

// Px задаёт физические пиксели буфера отрисовки (Device Pixels).
type Px int

// Int возвращает целочисленное значение Px.
func (p Px) Int() int {
	return int(p)
}

