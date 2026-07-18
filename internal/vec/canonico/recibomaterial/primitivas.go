package recibomaterial

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
)

const tamanoMaximoAtestacion = 16 * 1024

// AliasLogicoValido acepta solo referencias logicas ASCII y rechaza indicios
// de ubicacion fisica o identificadores personales.
func AliasLogicoValido(valor string, maximo int) bool {
	if valor == "" || len(valor) > maximo || valor != strings.TrimSpace(valor) ||
		!utf8.ValidString(valor) || !TextoASCIICanonico(valor) ||
		strings.Contains(valor, "/") || strings.Contains(valor, "\\") ||
		strings.Contains(valor, "..") || strings.Contains(valor, "://") ||
		strings.ContainsAny(valor, "?#@*") {
		return false
	}
	minusculas := strings.ToLower(valor)
	for _, marca := range []string{
		"arn:", "etag:", "kms:", "bucket:", "bucket_", "endpoint:",
		"ruta:", "path:", "file:", "s3:", "http:", "https:",
		"dni:", "nif:", "nie:", "nombre:", "apellido:", "correo:",
		"email:", "telefono:", "direccion:",
	} {
		if strings.Contains(minusculas, marca) {
			return false
		}
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return !PareceIdentificadorPersonal(valor)
}

// TextoASCIICanonico limita la entrada al repertorio imprimible no ambiguo.
func TextoASCIICanonico(valor string) bool {
	for indice := 0; indice < len(valor); indice++ {
		if valor[indice] < 0x21 || valor[indice] > 0x7e {
			return false
		}
	}
	return true
}

// PareceIdentificadorPersonal reconoce las formas basicas de DNI y NIE.
func PareceIdentificadorPersonal(valor string) bool {
	mayusculas := strings.ToUpper(valor)
	if len(mayusculas) != 9 {
		return false
	}
	digitosDNI := true
	for _, caracter := range mayusculas[:8] {
		if caracter < '0' || caracter > '9' {
			digitosDNI = false
			break
		}
	}
	ultima := mayusculas[8]
	if digitosDNI && ultima >= 'A' && ultima <= 'Z' {
		return true
	}
	primera := mayusculas[0]
	digitosNIE := primera == 'X' || primera == 'Y' || primera == 'Z'
	for _, caracter := range mayusculas[1:8] {
		if caracter < '0' || caracter > '9' {
			digitosNIE = false
			break
		}
	}
	return digitosNIE && ultima >= 'A' && ultima <= 'Z'
}

// MIMEValido acepta un tipo canonico sin parametros ni rutas ambiguas.
func MIMEValido(valor string) bool {
	if !almacencanonico.TextoSeguro(valor, 255) || strings.Count(valor, "/") != 1 ||
		!TextoASCIICanonico(valor) || strings.ContainsAny(valor, ";?#\\") ||
		valor != strings.ToLower(valor) {
		return false
	}
	partes := strings.Split(valor, "/")
	return len(partes) == 2 && partes[0] != "" && partes[1] != ""
}

// InstanteValido exige UTC exacto a microsegundos.
func InstanteValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

// CodigoAtestacionValido aplica limites cerrados por algoritmo.
func CodigoAtestacionValido(algoritmo string, codigo []byte) bool {
	if algoritmo == AlgoritmoHMACSHA256 {
		return len(codigo) == sha256.Size
	}
	return algoritmo == AlgoritmoCOSESign1 && len(codigo) >= 16 && len(codigo) <= tamanoMaximoAtestacion
}

// DominioAtestacionValido mantiene separadas las dos finalidades admitidas.
func DominioAtestacionValido(dominio string) bool {
	return dominio == DominioPerfil || dominio == DominioRecibo
}

// DecodificarSHA256 exige hexadecimal minusculo y rechaza el valor cero.
func DecodificarSHA256(valor string) ([sha256.Size]byte, error) {
	var resultado [sha256.Size]byte
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return resultado, ErrReciboNoValido
	}
	contenido, err := hex.DecodeString(valor)
	if err != nil || len(contenido) != sha256.Size {
		return resultado, ErrReciboNoValido
	}
	copy(resultado[:], contenido)
	if resultado == ([sha256.Size]byte{}) {
		return [sha256.Size]byte{}, ErrReciboNoValido
	}
	return resultado, nil
}

func SumaSHA256(contenido []byte) [sha256.Size]byte { return sha256.Sum256(contenido) }

// HuellasIguales compara sin filtrar informacion temporal sobre el prefijo.
func HuellasIguales(primera, segunda [sha256.Size]byte) bool {
	return subtle.ConstantTimeCompare(primera[:], segunda[:]) == 1
}

// AnexarTLV compone etiqueta y longitud en orden de red.
func AnexarTLV(destino []byte, etiqueta uint16, valor []byte) []byte {
	var cabecera [10]byte
	binary.BigEndian.PutUint16(cabecera[0:2], etiqueta)
	binary.BigEndian.PutUint64(cabecera[2:10], uint64(len(valor)))
	destino = append(destino, cabecera[:]...)
	return append(destino, valor...)
}

func Uint16(valor uint16) []byte {
	resultado := make([]byte, 2)
	binary.BigEndian.PutUint16(resultado, valor)
	return resultado
}
func Uint32(valor uint32) []byte {
	resultado := make([]byte, 4)
	binary.BigEndian.PutUint32(resultado, valor)
	return resultado
}
func Int64(valor int64) []byte {
	resultado := make([]byte, 8)
	binary.BigEndian.PutUint64(resultado, uint64(valor))
	return resultado
}
func Bool(valor bool) []byte {
	if valor {
		return []byte{1}
	}
	return []byte{0}
}

// DependenciaNula detecta interfaces que esconden un puntero tipado nulo.
func DependenciaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func FormatoRedactado(estado fmt.State)       { _, _ = io.WriteString(estado, TextoRedactado) }
func SerializacionProhibida() ([]byte, error) { return nil, ErrSerializacionProhibida }
func DeserializacionProhibida() error         { return ErrSerializacionProhibida }
