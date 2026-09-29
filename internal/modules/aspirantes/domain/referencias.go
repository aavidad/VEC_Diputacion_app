// Package domain contiene las reglas del módulo Aspirantes: la ficha de la
// persona aspirante, su documento de identidad tipado y los datos de contacto
// que puede pedir el catálogo. No conoce HTTP, SQL, X.509 ni criptografía.
package domain

import (
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

var ErrReferenciaInvalida = errors.New("aspirantes: referencia invalida")

const (
	prefijoAspirante = "asp_"
	prefijoDocumento = "aspdoc_"
	// 16 bytes aleatorios en base64url sin relleno ocupan 22 caracteres.
	bytesReferencia = 16
	largoAleatorio  = 22
)

// GenerarReferenciaAspirante crea la referencia opaca de una ficha. Solo usa
// azar: nunca se deriva del documento, del nombre ni de otra referencia.
func GenerarReferenciaAspirante(azar io.Reader) (string, error) {
	return generar(prefijoAspirante, azar)
}

// GenerarReferenciaDocumento crea la referencia opaca de una entrada del
// historial de documentos de identidad.
func GenerarReferenciaDocumento(azar io.Reader) (string, error) {
	return generar(prefijoDocumento, azar)
}

func generar(prefijo string, azar io.Reader) (string, error) {
	if azar == nil {
		return "", ErrReferenciaInvalida
	}
	var b [bytesReferencia]byte
	if _, err := io.ReadFull(azar, b[:]); err != nil {
		return "", ErrReferenciaInvalida
	}
	ref := prefijo + base64.RawURLEncoding.EncodeToString(b[:])
	for i := range b {
		b[i] = 0
	}
	return ref, nil
}

// ReferenciaAspiranteValida acepta solo el formato que emite este módulo.
func ReferenciaAspiranteValida(ref string) bool { return referenciaValida(prefijoAspirante, ref) }

// ReferenciaDocumentoValida acepta solo el formato que emite este módulo.
func ReferenciaDocumentoValida(ref string) bool { return referenciaValida(prefijoDocumento, ref) }

func referenciaValida(prefijo, ref string) bool {
	resto, ok := strings.CutPrefix(ref, prefijo)
	if !ok || len(resto) != largoAleatorio {
		return false
	}
	b, err := base64.RawURLEncoding.DecodeString(resto)
	return err == nil && len(b) == bytesReferencia && base64.RawURLEncoding.EncodeToString(b) == resto
}
