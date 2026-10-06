package auditoria

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

// MarcadorSinAnteriorV5 ocupa anterior_sha256 en los asientos escritos tras
// el corte de AD207. El escritor ya no lee la cabeza: su huella liga contenido
// y número de orden, y el enlace con el asiento anterior lo da el eslabón.
var MarcadorSinAnteriorV5 = strings.Repeat("f", 64)

// EslabonCadenaV5 es la fila de eslabon_auditoria_v5 de un asiento sellado.
// Posicion es su lugar en la cadena; Secuencia, su número de orden.
type EslabonCadenaV5 struct {
	Posicion       uint64 `json:"posicion"`
	Secuencia      uint64 `json:"secuencia"`
	AnteriorSHA256 string `json:"anterior_sha256"`
	EslabonSHA256  string `json:"eslabon_sha256"`
}

// HuellaEslabonV5 reproduce vec_autorizacion_atestada_v3.eslabon_auditoria_v5.
func HuellaEslabonV5(cadena string, posicion uint64, anterior string, secuencia uint64, auditoriaRef, tipo, huella string) string {
	h := sha256.Sum256(huellaEncuadradaIntento("vec.auditoria.eslabon.v5", cadena, strconv.FormatUint(posicion, 10), anterior,
		strconv.FormatUint(secuencia, 10), auditoriaRef, tipo, huella))
	return hex.EncodeToString(h[:])
}
