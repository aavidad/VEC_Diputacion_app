// Package domain contiene el núcleo neutral del expediente de contratación
// temporal. No depende de HTTP, PostgreSQL, documentos ni conectores.
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	ErrDatoInvalido       = errors.New("contratacion temporal: dato invalido")
	ErrExpedienteInvalido = errors.New("contratacion temporal: expediente invalido")
	ErrVersionEnConflicto = errors.New("contratacion temporal: version en conflicto")
	ErrTransicionInvalida = errors.New("contratacion temporal: transicion invalida")
)

var (
	patronReferencia = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$`)
	patronClave      = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)
	patronHuella     = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronNumero     = regexp.MustCompile(`^[0-9]{4}/[A-Za-z0-9._-]{1,40}$`)
	patronGrupo      = regexp.MustCompile(`^[A-Z][A-Z0-9/+.-]{0,19}$`)
)

// EstadoOperativo es una invariante técnica compartida por todos los flujos.
// La etiqueta y el color de cada estado pertenecen a la presentación.
type EstadoOperativo string

const (
	EstadoPendiente     EstadoOperativo = "pendiente"
	EstadoEnCurso       EstadoOperativo = "en_curso"
	EstadoEsperaExterna EstadoOperativo = "espera_externa"
	EstadoCompletado    EstadoOperativo = "completado"
	EstadoIncidencia    EstadoOperativo = "incidencia"
	EstadoCancelado     EstadoOperativo = "cancelado"
)

func (e EstadoOperativo) Valido() bool {
	switch e {
	case EstadoPendiente, EstadoEnCurso, EstadoEsperaExterna,
		EstadoCompletado, EstadoIncidencia, EstadoCancelado:
		return true
	default:
		return false
	}
}

// ClaveCatalogo representa una opción funcional gobernada. El dominio solo
// comprueba su forma; la definición publicada decide si existe y está vigente.
type ClaveCatalogo string

func (c ClaveCatalogo) Valida() bool {
	return patronClave.MatchString(string(c))
}

type ClaveFase string

func (c ClaveFase) Valida() bool {
	return patronClave.MatchString(string(c))
}

type ReferenciaFlujo struct {
	DefinicionRef string `json:"definicion_ref"`
	Version       uint64 `json:"version"`
	HuellaSHA256  string `json:"huella_sha256"`
}

func (r ReferenciaFlujo) Validar() error {
	if !referenciaValida(r.DefinicionRef) || r.Version == 0 ||
		!huellaValida(r.HuellaSHA256) {
		return ErrDatoInvalido
	}
	return nil
}

func huellaValida(valor string) bool {
	return patronHuella.MatchString(valor) && valor != strings.Repeat("0", 64)
}

type PeriodoPrevisto struct {
	Inicio time.Time `json:"inicio"`
	Fin    time.Time `json:"fin"`
}

func (p PeriodoPrevisto) Validar() error {
	if !fechaCivilCanonica(p.Inicio) || !fechaCivilCanonica(p.Fin) ||
		p.Fin.Before(p.Inicio) {
		return ErrDatoInvalido
	}
	return nil
}

type Importe struct {
	Centimos int64  `json:"centimos"`
	Moneda   string `json:"moneda"`
}

func (i Importe) Validar(permiteCero bool) error {
	if i.Moneda != "EUR" || i.Centimos < 0 || (!permiteCero && i.Centimos == 0) {
		return ErrDatoInvalido
	}
	return nil
}

func referenciaValida(valor string) bool {
	return patronReferencia.MatchString(valor)
}

func grupoValido(valor string) bool {
	return patronGrupo.MatchString(valor)
}

func instanteCanonico(valor time.Time) bool {
	return !valor.IsZero() && valor.Location() == time.UTC &&
		valor.Equal(valor.Truncate(time.Microsecond))
}

func fechaCivilCanonica(valor time.Time) bool {
	return instanteCanonico(valor) && valor.Hour() == 0 && valor.Minute() == 0 &&
		valor.Second() == 0 && valor.Nanosecond() == 0
}

func textoValido(valor string, maximo int, permiteVacio bool) bool {
	if valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) ||
		!norm.NFC.IsNormalString(valor) || utf8.RuneCountInString(valor) > maximo ||
		(!permiteVacio && valor == "") {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) && caracter != '\n' && caracter != '\t' {
			return false
		}
	}
	return true
}

func referenciasUnicasValidas(valores []string, maximo int) bool {
	if len(valores) > maximo {
		return false
	}
	vistas := make(map[string]struct{}, len(valores))
	for _, valor := range valores {
		if !referenciaValida(valor) {
			return false
		}
		if _, repetida := vistas[valor]; repetida {
			return false
		}
		vistas[valor] = struct{}{}
	}
	return true
}

// ReferenciaOpacaValida permite que los puertos validen sus contratos sin
// duplicar la gramática técnica del dominio.
func ReferenciaOpacaValida(valor string) bool {
	return referenciaValida(valor)
}

// NumeroExpedienteValido expone la gramática técnica del identificador
// visible para que persistencia y recibos no acepten formas que el agregado
// rechazaría.
func NumeroExpedienteValido(valor string) bool {
	return patronNumero.MatchString(valor)
}

// InstanteUTCCanonico expone exclusivamente la regla de representación
// temporal compartida; no obtiene la hora ni actúa como reloj.
func InstanteUTCCanonico(valor time.Time) bool {
	return instanteCanonico(valor)
}
