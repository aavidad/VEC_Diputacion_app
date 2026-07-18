// Package pagos concentra las reglas puras y deterministas del contrato de
// cobros. No contiene puertos ni conoce adaptadores, redes o persistencia.
package pagos

import (
	"net/url"
	"strings"
	"unicode"

	"vec-diputacion-granada/internal/vec/domain"
)

// ClaveValida comprueba una clave tecnica de lista cerrada.
func ClaveValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 128 {
		return false
	}
	for indice, caracter := range valor {
		if (caracter >= 'a' && caracter <= 'z') || (indice > 0 && caracter >= '0' && caracter <= '9') ||
			(indice > 0 && (caracter == '.' || caracter == '_' || caracter == '-')) {
			continue
		}
		return false
	}
	return true
}

// ReferenciaOpacaValida comprueba una referencia opaca ligada a un prefijo.
func ReferenciaOpacaValida(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) {
		return false
	}
	parte := strings.TrimPrefix(valor, prefijo)
	if len(parte) < 22 || len(parte) > 128 {
		return false
	}
	for _, caracter := range parte {
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || caracter == '_' || caracter == '-' {
			continue
		}
		return false
	}
	return true
}

// HuellaHMACDeDominioValida exige el dominio criptografico explicito.
func HuellaHMACDeDominioValida(valor, dominio string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && partes[1] == dominio &&
		HuellaSHA256Valida(partes[2])
}

// HuellaSHA256Valida acepta solamente la representacion hexadecimal canonica.
func HuellaSHA256Valida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

// MetodoAutenticacionPermitido mantiene cerrada la matriz de autenticacion.
func MetodoAutenticacionPermitido(metodo domain.AuthMethod) bool {
	switch metodo {
	case domain.AuthMethodCertificate, domain.AuthMethodDNIe, domain.AuthMethodSSO,
		domain.AuthMethodClave, domain.AuthMethodKerberos:
		return true
	default:
		return false
	}
}

// GarantiaAutenticacionPermitida mantiene cerrados los niveles aceptados.
func GarantiaAutenticacionPermitida(garantia domain.AuthAssurance) bool {
	switch garantia {
	case domain.AuthAssuranceLow, domain.AuthAssuranceSubstantial, domain.AuthAssuranceHigh:
		return true
	default:
		return false
	}
}

// AccionAuditoriaPermitida mantiene cerradas las acciones auditables de cobro.
func AccionAuditoriaPermitida(accion domain.AccionCobro) bool {
	switch accion {
	case domain.AccionCobroCrearOrden, domain.AccionCobroIniciarOperacion, domain.AccionCobroProcesarResultado,
		domain.AccionCobroSolicitarDevolucion, domain.AccionCobroProcesarDevolucion, domain.AccionCobroConciliar,
		domain.AccionCobroCancelar, domain.AccionCobroCaducar:
		return true
	default:
		return false
	}
}

// TextoValido rechaza controles, espacios exteriores y posibles datos de tarjeta.
func TextoValido(valor string, maximo int) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo {
		return false
	}
	for _, caracter := range valor {
		if caracter < 0x20 || caracter == 0x7f {
			return false
		}
	}
	return !ContieneDatoTarjeta(valor)
}

// ContieneDatoTarjeta detecta etiquetas sensibles y numeros que superan Luhn,
// incluidos digitos arabigos y de ancho completo y separadores invisibles.
func ContieneDatoTarjeta(valor string) bool {
	minusculas := strings.Map(func(caracter rune) rune {
		if unicode.Is(unicode.Cf, caracter) {
			return -1
		}
		return unicode.ToLower(caracter)
	}, valor)
	reemplazador := strings.NewReplacer("_", " ", "-", " ", ":", " ", "=", " ", ".", " ", "/", " ")
	for _, palabra := range strings.Fields(reemplazador.Replace(minusculas)) {
		switch palabra {
		case "pan", "cvv", "cvc", "cvn", "pin", "criptograma", "cryptogram", "tarjeta", "card", "cardnumber":
			return true
		}
	}
	digitos := make([]byte, 0, 32)
	comprobar := func() bool {
		for longitud := 13; longitud <= 19 && longitud <= len(digitos); longitud++ {
			for inicio := 0; inicio+longitud <= len(digitos); inicio++ {
				if numeroTarjetaValido(digitos[inicio : inicio+longitud]) {
					return true
				}
			}
		}
		return false
	}
	for _, caracter := range valor {
		if numero, esDigito := valorDigitoDecimal(caracter); esDigito {
			digitos = append(digitos, byte('0'+numero))
			continue
		}
		if (unicode.IsSpace(caracter) || unicode.Is(unicode.Dash, caracter) ||
			unicode.Is(unicode.Cf, caracter) || caracter == '.') && len(digitos) > 0 {
			continue
		}
		if comprobar() {
			return true
		}
		digitos = digitos[:0]
	}
	return comprobar()
}

func valorDigitoDecimal(caracter rune) (byte, bool) {
	switch {
	case caracter >= '0' && caracter <= '9':
		return byte(caracter - '0'), true
	case caracter >= '\u0660' && caracter <= '\u0669':
		return byte(caracter - '\u0660'), true
	case caracter >= '\u06f0' && caracter <= '\u06f9':
		return byte(caracter - '\u06f0'), true
	case caracter >= '\uff10' && caracter <= '\uff19':
		return byte(caracter - '\uff10'), true
	default:
		return 0, false
	}
}

func numeroTarjetaValido(digitos []byte) bool {
	suma := 0
	par := len(digitos)%2 == 0
	for indice, caracter := range digitos {
		numero := int(caracter - '0')
		if (indice%2 == 0) == par {
			numero *= 2
			if numero > 9 {
				numero -= 9
			}
		}
		suma += numero
	}
	return suma > 0 && suma%10 == 0
}

// ListaCerradaValida comprueba unicidad y sintaxis sin modificar la entrada.
func ListaCerradaValida(valores []string, rutas bool) bool {
	if len(valores) == 0 || len(valores) > 64 {
		return false
	}
	vistos := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		valido := ClaveValida(valor)
		if rutas {
			valido = RutaHandoffValida(valor)
		}
		if !valido {
			return false
		}
		if _, repetido := vistos[valor]; repetido {
			return false
		}
		vistos[valor] = struct{}{}
	}
	return true
}

// RutaHandoffValida solo admite rutas relativas canonicas sin escapes ni datos.
func RutaHandoffValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > 1024 ||
		!strings.HasPrefix(valor, "/") || strings.HasPrefix(valor, "//") || strings.Contains(valor, "\\") ||
		ContieneDatoTarjeta(valor) {
		return false
	}
	ruta, err := url.Parse(valor)
	if err != nil || ruta.IsAbs() || ruta.Opaque != "" || ruta.Host != "" || ruta.User != nil ||
		ruta.RawQuery != "" || ruta.ForceQuery || ruta.Fragment != "" || ruta.RawPath != "" || ruta.Path != valor {
		return false
	}
	segmentos := strings.Split(strings.TrimPrefix(valor, "/"), "/")
	for _, segmento := range segmentos {
		if segmento == "" || segmento == "." || segmento == ".." {
			return false
		}
		for _, caracter := range segmento {
			if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
				(caracter >= '0' && caracter <= '9') || caracter == '-' || caracter == '_' || caracter == '.' || caracter == '~' {
				continue
			}
			return false
		}
	}
	return true
}

// ContieneCadenaExacta busca sin normalizaciones ambiguas.
func ContieneCadenaExacta(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}
	return false
}

// TipoContenidoNotificacionPermitido mantiene cerrados los formatos remotos.
func TipoContenidoNotificacionPermitido(valor string) bool {
	switch valor {
	case "application/json", "application/jose", "application/jose+json":
		return true
	default:
		return false
	}
}
