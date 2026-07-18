package documental

import (
	"bytes"
	"strconv"
	"unicode/utf8"
)

const (
	// EsquemaCanonizacionEntradaNeutralV1 identifica el codec cerrado de la
	// entrada neutral documental. Cambiarlo exige una nueva version del codec.
	EsquemaCanonizacionEntradaNeutralV1 = "vec.documentos.entrada-neutral.contenido-longitud-prefijada.v1"
	maximoBytesEntradaNeutralV1         = 16 * 1024 * 1024
	maximosParrafosEntradaNeutralV1     = 100_000
)

// CanonizarEntradaNeutralV1 fija titulo, cardinalidad y parrafos mediante
// campos de longitud prefijada en bytes. Conserva el orden, no normaliza
// Unicode y rechaza controles salvo tabulador y saltos de linea. El booleano
// es falso cuando la entrada no pertenece al dominio cerrado del codec.
func CanonizarEntradaNeutralV1(titulo string, parrafos []string) ([]byte, bool) {
	if len(parrafos) > maximosParrafosEntradaNeutralV1 || !textoEntradaNeutralV1Valido(titulo) ||
		(titulo == "" && len(parrafos) == 0) {
		return nil, false
	}

	contador := strconv.Itoa(len(parrafos))
	tamano := 0
	reservar := func(valor string) bool {
		for _, incremento := range []int{len(strconv.Itoa(len(valor))), 1, len(valor), 1} {
			if incremento < 0 || incremento > maximoBytesEntradaNeutralV1 ||
				tamano > maximoBytesEntradaNeutralV1-incremento {
				return false
			}
			tamano += incremento
		}
		return true
	}
	if !reservar(EsquemaCanonizacionEntradaNeutralV1) || !reservar(titulo) || !reservar(contador) {
		return nil, false
	}
	for _, parrafo := range parrafos {
		if !textoEntradaNeutralV1Valido(parrafo) || !reservar(parrafo) {
			return nil, false
		}
	}

	canonico := make([]byte, 0, tamano)
	anadir := func(valor string) {
		canonico = strconv.AppendInt(canonico, int64(len(valor)), 10)
		canonico = append(canonico, ':')
		canonico = append(canonico, valor...)
		canonico = append(canonico, '\n')
	}
	anadir(EsquemaCanonizacionEntradaNeutralV1)
	anadir(titulo)
	anadir(contador)
	for _, parrafo := range parrafos {
		anadir(parrafo)
	}
	return canonico, len(canonico) == tamano
}

// PreimagenEntradaNeutralV1Valida comprueba de nuevo el dominio y la
// correspondencia byte a byte de una preimagen ya fijada. Mantiene privados
// los limites del codec y permite que el puerto solo traduzca el resultado a
// su error nominal, sin duplicar reglas de canonizacion.
func PreimagenEntradaNeutralV1Valida(
	titulo string,
	parrafos []string,
	preimagen []byte,
	tamano uint64,
) bool {
	if len(preimagen) == 0 || uint64(len(preimagen)) != tamano || tamano > maximoBytesEntradaNeutralV1 {
		return false
	}
	esperada, valida := CanonizarEntradaNeutralV1(titulo, parrafos)
	return valida && bytes.Equal(esperada, preimagen)
}

func textoEntradaNeutralV1Valido(valor string) bool {
	if len(valor) > maximoBytesEntradaNeutralV1 || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if caracter == '\t' || caracter == '\n' || caracter == '\r' {
			continue
		}
		if caracter < 0x20 || caracter == 0x7f {
			return false
		}
	}
	return true
}
