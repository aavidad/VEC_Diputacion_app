package xlsconvoca

import "bytes"

// FormatoContenido identifica el contenedor físico sin interpretar sus hojas.
// El lector completo valida después la estructura y el contenido del libro.
func FormatoContenido(contenido []byte) (string, error) {
	if len(contenido) > maximoBytesXLS {
		return "", ErrLimiteXLSExcedido
	}
	if len(contenido) >= len(firmaContenedorOLE2) &&
		bytes.Equal(contenido[:len(firmaContenedorOLE2)], firmaContenedorOLE2[:]) {
		return "xls", nil
	}
	if len(contenido) >= 4 && bytes.Equal(contenido[:4], []byte{'P', 'K', 3, 4}) {
		return "xlsx", nil
	}
	return "", ErrXLSInvalido
}
