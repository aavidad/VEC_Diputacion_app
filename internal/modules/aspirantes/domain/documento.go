package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

var ErrDocumentoInvalido = errors.New("aspirantes: documento de identidad invalido")

// TipoDocumento es el tipo de documento de identidad. El vocabulario es
// cerrado: añadir un tipo exige cambiar también la migración SQL.
type TipoDocumento string

const (
	DocumentoDNI       TipoDocumento = "dni"
	DocumentoNIE       TipoDocumento = "nie"
	DocumentoPasaporte TipoDocumento = "pasaporte"
	DocumentoOtro      TipoDocumento = "otro"
)

// String y GoString nunca muestran el número: un %v en un registro o en un
// error solo enseña el tipo, el país y la forma enmascarada.
func (d DocumentoIdentidad) String() string {
	return "aspirantes.DocumentoIdentidad{" + string(d.Tipo) + " " + d.Pais + " " + d.Enmascarado() + "}"
}
func (d DocumentoIdentidad) GoString() string { return d.String() }

// MarshalJSON y LogValue tampoco sacan el número completo.
func (d DocumentoIdentidad) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }
func (d DocumentoIdentidad) LogValue() slog.Value         { return slog.StringValue(d.String()) }

// Format redacta también con %d, %x y demás verbos.
func (d DocumentoIdentidad) Format(f fmt.State, _ rune) { _, _ = f.Write([]byte(d.String())) }

func (t TipoDocumento) Valido() bool {
	switch t {
	case DocumentoDNI, DocumentoNIE, DocumentoPasaporte, DocumentoOtro:
		return true
	}
	return false
}

// DocumentoIdentidad guarda tipo, país emisor (ISO 3166-1 alfa-2) y número
// normalizado. Solo se construye con NuevoDocumentoIdentidad.
type DocumentoIdentidad struct {
	Tipo   TipoDocumento
	Pais   string
	Numero string
}

const letrasControlDNI = "TRWAGMYFPDXBNJZSQVHLCKE"

// NuevoDocumentoIdentidad normaliza (mayúsculas; sin espacios, guiones ni
// puntos) y valida. DNI y NIE exigen país ES y letra de control correcta.
func NuevoDocumentoIdentidad(tipo TipoDocumento, pais, numero string) (DocumentoIdentidad, error) {
	d := DocumentoIdentidad{Tipo: tipo, Pais: strings.ToUpper(strings.TrimSpace(pais)), Numero: normalizarNumero(numero)}
	if err := d.Validar(); err != nil {
		return DocumentoIdentidad{}, err
	}
	return d, nil
}

func normalizarNumero(numero string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(numero) {
		if r == ' ' || r == '-' || r == '.' || r == '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Validar comprueba un documento ya normalizado.
func (d DocumentoIdentidad) Validar() error {
	if !d.Tipo.Valido() || !paisValido(d.Pais) || d.Numero != normalizarNumero(d.Numero) {
		return ErrDocumentoInvalido
	}
	switch d.Tipo {
	case DocumentoDNI:
		if d.Pais != "ES" || !dniValido(d.Numero) {
			return ErrDocumentoInvalido
		}
	case DocumentoNIE:
		if d.Pais != "ES" || !nieValido(d.Numero) {
			return ErrDocumentoInvalido
		}
	case DocumentoPasaporte:
		if !alfanumerico(d.Numero, 5, 20) {
			return ErrDocumentoInvalido
		}
	case DocumentoOtro:
		// Un documento de otro Estado nunca se hace pasar por DNI o NIE.
		if d.Pais == "ES" || !alfanumerico(d.Numero, 3, 30) {
			return ErrDocumentoInvalido
		}
	}
	return nil
}

// paisValido comprueba la forma ISO 3166-1 alfa-2. La lista cerrada de
// países llegará con su catálogo; hoy no se usa para decidir nada más que
// DNI/NIE (ES) frente a documentos de otros Estados.
func paisValido(p string) bool {
	return len(p) == 2 && p[0] >= 'A' && p[0] <= 'Z' && p[1] >= 'A' && p[1] <= 'Z'
}

func alfanumerico(s string, minimo, maximo int) bool {
	if len(s) < minimo || len(s) > maximo {
		return false
	}
	for _, r := range s {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func digitos(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

func dniValido(s string) bool {
	if len(s) != 9 {
		return false
	}
	n, ok := digitos(s[:8])
	return ok && s[8] == letrasControlDNI[n%23]
}

func nieValido(s string) bool {
	if len(s) != 9 {
		return false
	}
	prefijo := strings.IndexByte("XYZ", s[0])
	if prefijo < 0 {
		return false
	}
	n, ok := digitos(s[1:8])
	return ok && s[8] == letrasControlDNI[(prefijo*10_000_000+n)%23]
}

// Enmascarado devuelve la forma que puede verse en pantalla, según la
// orientación de la AEPD de 2019 (a confirmar por el DPD): DNI «***4567**»,
// NIE «****4567*»; en pasaportes y otros, los caracteres 4 a 7 si el número
// tiene al menos 8 y todo oculto si es más corto.
func (d DocumentoIdentidad) Enmascarado() string {
	if d.Validar() != nil {
		return ""
	}
	n := d.Numero
	switch d.Tipo {
	case DocumentoDNI:
		return "***" + n[3:7] + "**"
	case DocumentoNIE:
		return "****" + n[4:8] + "*"
	}
	if len(n) < 8 {
		return strings.Repeat("*", len(n))
	}
	return "***" + n[3:7] + strings.Repeat("*", len(n)-7)
}
