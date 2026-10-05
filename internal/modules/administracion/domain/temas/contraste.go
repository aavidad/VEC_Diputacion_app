package temas

import (
	"math"
	"strconv"
)

func luminancia(hex string) float64 {
	if !color.MatchString(hex) {
		return math.NaN()
	}
	var resultado float64
	for i, peso := range []float64{0.2126, 0.7152, 0.0722} {
		canal, err := strconv.ParseUint(hex[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return math.NaN()
		}
		v := float64(canal) / 255
		if v <= 0.04045 {
			v /= 12.92
		} else {
			v = math.Pow((v+0.055)/1.055, 2.4)
		}
		resultado += v * peso
	}
	return resultado
}

// Contraste calcula la razón sRGB de dos colores opacos; no certifica una UI.
func Contraste(a, b string) float64 {
	l1, l2 := luminancia(a), luminancia(b)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}
