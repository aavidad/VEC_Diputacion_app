package constitucion

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

// Referencia opaca de la persona candidata (`can_*`), derivada con clave de
// su identidad enmascarada tal como figura en las listas Convoca (documento
// enmascarado y nombre). La frontera de identidad deriva la misma referencia
// al acreditar a la persona (enmascara el NIF del certificado y normaliza el
// nombre), de modo que el vínculo `can_* → participación` no exige guardar
// el documento en ningún sitio. La clave HMAC vive en el material KMS.
//
// Límites conocidos (modelo en estudio de protección de datos):
//   - la entrada procede de listas públicas; quien obtenga la clave puede
//     recalcular la referencia de cualquier persona publicada, así que la
//     clave es tan sensible como el propio vínculo;
//   - la clave no lleva versión: cambiarla cambia todas las referencias y
//     deja huérfanos los vínculos ya registrados;
//   - dos personas con los mismos cuatro dígitos y el mismo nombre derivan
//     la misma referencia. Dentro de un acta se detecta y esas filas quedan
//     pendientes de revisión; entre actas distintas no se puede distinguir de
//     la misma persona inscrita en dos bolsas.

var (
	ErrClaveCandidatoRequerida    = errors.New("bolsa constitucion: clave de derivacion de candidato requerida")
	ErrIdentidadCandidatoInvalida = errors.New("bolsa constitucion: identidad de candidato invalida")

	patronDocumentoEnmascarado = regexp.MustCompile(`^\*{3}[0-9]{4}\*{2}$`)
	patronNIF                  = regexp.MustCompile(`^[0-9]{8}[A-Z]$|^[XYZ][0-9]{7}[A-Z]$`)
	patronReferenciaCandidato  = regexp.MustCompile(`^can_[A-Za-z0-9_-]{43}$`)
)

const prefijoReferenciaCandidato = "can_"

// DerivadorCandidato deriva la referencia opaca de una persona.
type DerivadorCandidato interface {
	CandidatoRef(importacion.IdentidadEnmascarada) (string, error)
}

// DerivadorCandidatoHMAC deriva `can_` + base64url(HMAC-SHA256(clave, clave_identidad)).
type DerivadorCandidatoHMAC struct{ clave [32]byte }

func NuevoDerivadorCandidatoHMAC(clave [32]byte) (*DerivadorCandidatoHMAC, error) {
	if clave == ([32]byte{}) {
		return nil, ErrClaveCandidatoRequerida
	}
	return &DerivadorCandidatoHMAC{clave: clave}, nil
}

func (d *DerivadorCandidatoHMAC) CandidatoRef(identidad importacion.IdentidadEnmascarada) (string, error) {
	if d == nil {
		return "", ErrClaveCandidatoRequerida
	}
	clave, err := ClaveIdentidadCandidato(identidad)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, d.clave[:])
	mac.Write([]byte(clave))
	return prefijoReferenciaCandidato + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

// ReferenciaCandidatoValida comprueba la forma `can_` + 43 caracteres base64url.
func ReferenciaCandidatoValida(ref string) bool {
	return patronReferenciaCandidato.MatchString(ref)
}

// ClaveIdentidadCandidato construye la cadena canónica de identidad:
// documento enmascarado y nombre normalizados (mayúsculas sin diacríticos,
// espacios colapsados), separados por «|». El documento admite ya
// enmascarado o un NIF/NIE completo, que se enmascara aquí.
func ClaveIdentidadCandidato(identidad importacion.IdentidadEnmascarada) (string, error) {
	documento, err := EnmascararDocumento(identidad.Documento)
	if err != nil {
		return "", err
	}
	apellido1 := normalizarTextoIdentidad(identidad.PrimerApellido)
	apellido2 := normalizarTextoIdentidad(identidad.SegundoApellido)
	nombre := normalizarTextoIdentidad(identidad.Nombre)
	if apellido1 == "" || nombre == "" {
		return "", ErrIdentidadCandidatoInvalida
	}
	return strings.Join([]string{documento, apellido1, apellido2, nombre}, "|"), nil
}

// EnmascararDocumento devuelve la forma canónica `***NNNN**`, que es la que
// usan las exportaciones de Convoca tanto para DNI como para NIE. Conserva
// los mismos cuatro dígitos que publica la orientación de la AEPD: en el DNI,
// las posiciones 4–7 (12345678X → ***4567**, igual que la AEPD); en el NIE,
// las posiciones 5–8 del documento completo, que la AEPD publica como
// `****4567*` (X1234567L) y aquí se normalizan a `***4567**`. La forma
// `****NNNN*` de la AEPD no se acepta como entrada: la importación de Convoca
// solo admite `***NNNN**`.
//
// Cuatro dígitos más el nombre no identifican de forma única a una persona:
// la constitución detecta las filas que coinciden y no las vincula (ver
// referenciasCandidato).
func EnmascararDocumento(documento string) (string, error) {
	limpio := strings.ToUpper(strings.Join(strings.Fields(strings.ReplaceAll(documento, "-", "")), ""))
	if patronDocumentoEnmascarado.MatchString(limpio) {
		return limpio, nil
	}
	if !patronNIF.MatchString(limpio) {
		return "", ErrIdentidadCandidatoInvalida
	}
	if limpio[0] >= 'A' {
		return "***" + limpio[4:8] + "**", nil
	}
	return "***" + limpio[3:7] + "**", nil
}

func normalizarTextoIdentidad(valor string) string {
	sinDiacriticos, _, err := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), valor)
	if err != nil {
		sinDiacriticos = valor
	}
	return strings.Join(strings.Fields(strings.ToUpper(sinDiacriticos)), " ")
}
