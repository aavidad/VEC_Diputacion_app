// Package documental concentra reglas puras y deterministas de la ejecucion
// documental. No conoce puertos, adaptadores, persistencia ni transporte.
package documental

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const prefijoHMACSHA256 = "hmac-sha256"

// SerializarCamposV3 fija la preimagen usada por los compromisos y
// atestaciones documentales V3. La longitud se expresa en bytes, no en runas.
// El formato es deliberadamente simple y no admite representaciones alternas:
// <longitud-decimal>:<valor>\n para cada campo, incluido el campo vacio.
func SerializarCamposV3(valores []string) []byte {
	var salida []byte
	for _, valor := range valores {
		salida = strconv.AppendInt(salida, int64(len(valor)), 10)
		salida = append(salida, ':')
		salida = append(salida, valor...)
		salida = append(salida, '\n')
	}
	return salida
}

// HuellaCamposSHA256V3 deriva la huella hexadecimal canonica de una lista de
// campos. Comparte exactamente la misma preimagen que SerializarCamposV3.
func HuellaCamposSHA256V3(valores []string) string {
	suma := sha256.Sum256(SerializarCamposV3(valores))
	return hex.EncodeToString(suma[:])
}

// HuellaBytesSHA256 devuelve la codificacion hexadecimal minuscula de SHA-256.
func HuellaBytesSHA256(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

// BytesIguales conserva la comparacion exacta de las preimagenes canonicas.
func BytesIguales(primero, segundo []byte) bool {
	return bytes.Equal(primero, segundo)
}

// ReferenciaEjecucionV3Valida aplica la lista positiva de caracteres y excluye
// esquemas que podrian convertir una referencia opaca en dato personal o URL.
func ReferenciaEjecucionV3Valida(valor string) bool {
	if len(valor) == 0 || len(valor) > 256 || valor[0] < 'a' || valor[0] > 'z' ||
		strings.Contains(valor, "://") || strings.ContainsAny(valor, "*\\/@?#%=") {
		return false
	}
	for indice := 1; indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9') ||
			caracter == '.' || caracter == '_' || caracter == ':' || caracter == '-' {
			continue
		}
		return false
	}
	for _, prefijo := range []string{"dni:", "nif:", "nie:", "email:", "mailto:"} {
		if strings.HasPrefix(valor, prefijo) {
			return false
		}
	}
	return true
}

// ReferenciasEjecucionV3Distintas exige forma opaca, cardinalidad exacta y
// ausencia de reutilizacion entre referencias con finalidades distintas.
func ReferenciasEjecucionV3Distintas(valores ...string) bool {
	vistas := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		if !ReferenciaEjecucionV3Valida(valor) {
			return false
		}
		if _, existe := vistas[valor]; existe {
			return false
		}
		vistas[valor] = struct{}{}
	}
	return true
}

// SHA256HexadecimalValido acepta exclusivamente 32 bytes expresados como 64
// caracteres hexadecimales minusculos.
func SHA256HexadecimalValido(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.TrimSpace(valor) ||
		valor != strings.ToLower(valor) {
		return false
	}
	decodificado, err := hex.DecodeString(valor)
	return err == nil && len(decodificado) == sha256.Size
}

// HuellasSHA256Distintas exige forma hexadecimal canonica y ausencia de
// reutilizacion entre compromisos que representan finalidades distintas.
// La coleccion vacia satisface la condicion de forma vacua, igual que el
// contrato historico de materializacion documental.
func HuellasSHA256Distintas(huellas ...string) bool {
	vistas := make(map[string]struct{}, len(huellas))
	for _, huella := range huellas {
		if !SHA256HexadecimalValido(huella) {
			return false
		}
		if _, repetida := vistas[huella]; repetida {
			return false
		}
		vistas[huella] = struct{}{}
	}
	return true
}

func referenciaClaveHMACValida(valor string) bool {
	if valor == "" || len(valor) > 64 || valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return true
}

// HMACSHA256V3Valido comprueba algoritmo, referencia de clave y digest.
func HMACSHA256V3Valido(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == prefijoHMACSHA256 &&
		referenciaClaveHMACValida(partes[1]) && SHA256HexadecimalValido(partes[2])
}

// BytesNoNulos exige al menos un octeto distinto de cero.
func BytesNoNulos(valor []byte) bool {
	for _, octeto := range valor {
		if octeto != 0 {
			return true
		}
	}
	return false
}

// InstanteV3Valido exige UTC, rango representable y precision de microsegundo,
// que es la precision contractual de persistencia de la ejecucion documental.
func InstanteV3Valido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

// ClaveHMACSHA256V3 proyecta el identificador de clave de una huella con forma
// hmac-sha256:<clave>:<digest>. La validacion del digest corresponde al contrato
// que consume esta proyeccion.
func ClaveHMACSHA256V3(valor string) string {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != prefijoHMACSHA256 {
		return ""
	}
	return partes[1]
}

// ClavesHMACSHA256V3Distintas impide reutilizar una clave entre dominios
// criptograficos que deben permanecer independientes.
func ClavesHMACSHA256V3Distintas(valores ...string) bool {
	claves := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		clave := ClaveHMACSHA256V3(valor)
		if clave == "" {
			return false
		}
		if _, existe := claves[clave]; existe {
			return false
		}
		claves[clave] = struct{}{}
	}
	return true
}

// Uint64Decimal fija la representacion decimal sin signo usada en preimagenes.
func Uint64Decimal(valor uint64) string {
	return strconv.FormatUint(valor, 10)
}
