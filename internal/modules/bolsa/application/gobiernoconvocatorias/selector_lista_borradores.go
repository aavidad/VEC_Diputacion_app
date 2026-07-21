package gobiernoconvocatorias

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

type SelectorListaBorradores struct {
	Limite    int
	Cursor    string
	Texto     string
	Categoria string
}

const versionUnicodeNormalizadorBorradoresV1 = "15.0.0"

// Validar fija el contrato de búsqueda antes de autorización o persistencia.
// Cursor es opaco: solo se acepta la forma emitida por PostgreSQL, nunca una
// referencia libre reconstruida por un cliente.
func (s SelectorListaBorradores) Validar() error {
	if s.Limite < 1 || s.Limite > 50 || !cursorListaBorradoresValido(s.Cursor) ||
		!textoFiltroListaBorradoresValido(s.Texto) ||
		!categoriaFiltroListaBorradoresValida(s.Categoria) {
		return ErrSolicitudBorradorInvalida
	}
	return nil
}

func cursorListaBorradoresValido(cursor string) bool {
	const prefijo = "cursor-borrador-"
	return cursor == "" || (strings.HasPrefix(cursor, prefijo) &&
		huellaHexValida(strings.TrimPrefix(cursor, prefijo)))
}

func textoFiltroListaBorradoresValido(texto string) bool {
	// Guard deliberado: una subida de toolchain debe revisar y versionar el
	// perfil, aunque sus exclusiones ya anticipen las tablas Unicode 17.
	if norm.Version != versionUnicodeNormalizadorBorradoresV1 {
		return false
	}
	if texto == "" {
		return true
	}
	if !utf8.ValidString(texto) || texto != strings.TrimSpace(texto) ||
		!norm.NFC.IsNormalString(texto) || utf8.RuneCountInString(texto) > 180 ||
		len(texto) > 720 {
		return false
	}
	for _, caracter := range texto {
		if unicode.IsControl(caracter) || unicode.Is(unicode.Cf, caracter) ||
			caracter == unicode.ReplacementChar ||
			runaFueraPerfilNormalizacionComunBorradoresV1(caracter) {
			return false
		}
	}
	return true
}

// runaFueraPerfilNormalizacionComunBorradoresV1 mantiene un dominio estable
// entre las tablas Unicode 15 que x/text usa hasta Go 1.26, Unicode 16 de
// PostgreSQL 18 y Unicode 17 que x/text selecciona desde Go 1.27. Se excluyen
// exclusivamente las runas cuyas propiedades NFC cambiaron en ese intervalo:
// CCC, descomposicion, Quick_Check o participacion en composicion.
//
// La lista se deriva de UnicodeData/DerivedNormalizationProps 16.0 y de
// unicode/norm/data{15,17}.0.0_test.go de x/text v0.40.0. Rechazarla en ambas
// fronteras evita que una actualizacion del toolchain cambie si una misma
// secuencia es canonica.
func runaFueraPerfilNormalizacionComunBorradoresV1(caracter rune) bool {
	switch {
	case caracter == 0x0897,
		caracter >= 0x1ACF && caracter <= 0x1ADD,
		caracter >= 0x1AE0 && caracter <= 0x1AEB,
		caracter == 0x105C9,
		caracter == 0x105D2,
		caracter == 0x105DA,
		caracter == 0x105E4,
		caracter >= 0x10D69 && caracter <= 0x10D6D,
		caracter >= 0x10EFA && caracter <= 0x10EFB,
		caracter >= 0x11382 && caracter <= 0x11385,
		caracter == 0x1138B,
		caracter == 0x1138E,
		caracter >= 0x11390 && caracter <= 0x11391,
		caracter == 0x113B8,
		caracter == 0x113BB,
		caracter == 0x113C2,
		caracter == 0x113C5,
		caracter >= 0x113C7 && caracter <= 0x113C9,
		caracter >= 0x113CE && caracter <= 0x113D0,
		caracter >= 0x1611E && caracter <= 0x16129,
		caracter == 0x1612F,
		caracter == 0x16D63,
		caracter >= 0x16D67 && caracter <= 0x16D6A,
		caracter >= 0x1E5EE && caracter <= 0x1E5EF,
		caracter == 0x1E6E3,
		caracter == 0x1E6E6,
		caracter >= 0x1E6EE && caracter <= 0x1E6EF,
		caracter == 0x1E6F5:
		return true
	default:
		return false
	}
}

func categoriaFiltroListaBorradoresValida(categoria string) bool {
	if categoria == "" {
		return true
	}
	if len(categoria) > 80 || !caracterClaveListaBorradoresValido(categoria[0], false) {
		return false
	}
	for indice := 1; indice < len(categoria); indice++ {
		if !caracterClaveListaBorradoresValido(categoria[indice], true) {
			return false
		}
	}
	return true
}

func caracterClaveListaBorradoresValido(caracter byte, admiteSeparador bool) bool {
	return caracter >= 'a' && caracter <= 'z' || caracter >= '0' && caracter <= '9' ||
		admiteSeparador && (caracter == '.' || caracter == '_' || caracter == '-')
}
