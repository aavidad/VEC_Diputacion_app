package documental

// ReferenciaASCIIBasicaValida equivale exactamente a
// ^[a-z][a-z0-9._:-]{0,255}$ sin construir un automata regular. Todos los
// caracteres admitidos son ASCII de un byte, por lo que longitud de bytes y
// numero de caracteres coinciden para cualquier entrada valida.
func ReferenciaASCIIBasicaValida(valor string) bool {
	return identificadorASCIIValido(valor, 256, true)
}

// IDClaveHMACASCIIBasicoValido equivale exactamente a
// ^[a-z][a-z0-9._-]{0,127}$.
func IDClaveHMACASCIIBasicoValido(valor string) bool {
	return identificadorASCIIValido(valor, 128, false)
}

func identificadorASCIIValido(valor string, maximo int, admiteDosPuntos bool) bool {
	if len(valor) == 0 || len(valor) > maximo || valor[0] < 'a' || valor[0] > 'z' {
		return false
	}
	for indice := 1; indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9') ||
			caracter == '.' || caracter == '_' || caracter == '-' || (admiteDosPuntos && caracter == ':') {
			continue
		}
		return false
	}
	return true
}

// DNINIEASCIIMinusculoEvidente equivale exactamente a
// ^(?:[0-9]{8}[a-z]|[xyz][0-9]{7}[a-z])$.
func DNINIEASCIIMinusculoEvidente(valor string) bool {
	if len(valor) != 9 {
		return false
	}
	ultimo := valor[8]
	if ultimo < 'a' || ultimo > 'z' {
		return false
	}
	inicioDigitos := 0
	if valor[0] == 'x' || valor[0] == 'y' || valor[0] == 'z' {
		inicioDigitos = 1
	}
	for indice := inicioDigitos; indice < 8; indice++ {
		if valor[indice] < '0' || valor[indice] > '9' {
			return false
		}
	}
	return inicioDigitos == 1 || (valor[0] >= '0' && valor[0] <= '9')
}
