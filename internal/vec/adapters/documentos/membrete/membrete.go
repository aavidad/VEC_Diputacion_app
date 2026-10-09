// Package membrete guarda el logotipo institucional que encabeza los
// documentos generados. Se embebe en el binario para no depender de rutas de
// disco en tiempo de ejecución. Origen: web/static/assets/logo-diputacion-granada.svg,
// rasterizado a 750x252 px sobre fondo blanco y sin metadatos.
package membrete

import (
	"bytes"
	_ "embed"
)

//go:embed logo.png
var logoPNG []byte

const (
	// AnchoPx y AltoPx son las dimensiones del PNG embebido.
	AnchoPx = 750
	AltoPx  = 252
)

// LogoPNG devuelve una copia del logotipo para que ningún consumidor pueda
// alterar el activo compartido.
func LogoPNG() []byte {
	return bytes.Clone(logoPNG)
}

// EsLogo indica si unos bytes son exactamente el logotipo embebido. Los
// validadores lo usan para admitir la imagen y ninguna otra.
func EsLogo(datos []byte) bool {
	return bytes.Equal(datos, logoPNG)
}
