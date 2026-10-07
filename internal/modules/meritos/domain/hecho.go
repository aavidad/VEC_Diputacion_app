package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"

	documentos "vec-diputacion-granada/internal/vec/domain"
)

var ErrHecho = errors.New("meritos.error.hecho_invalido")

type Estado string

const (
	Declarado  Estado = "declarado"
	Pendiente  Estado = "pendiente"
	Acreditado Estado = "acreditado"
	Rechazado  Estado = "rechazado"
)

// Procedencia identifica el hecho original, con independencia de su versión.
// Su correspondencia con Persona y su autenticidad requieren consulta autorizada.
type Procedencia struct {
	FuenteRef      string `json:"fuente_ref"`
	Version        string `json:"version"`
	HechoOrigenRef string `json:"hecho_origen_ref"`
	CapturadaEn    string `json:"capturada_en"`
}

// Vigencia usa fechas civiles inclusivas; una fecha final ausente es abierta.
// No decide en qué hito una convocatoria exige cumplir un requisito.
type Vigencia struct {
	Desde string `json:"desde"`
	Hasta string `json:"hasta,omitempty"`
}

// Revision conserva una afirmación de revisión, sin otorgar autoridad al actor.
type Revision struct {
	Referencia string `json:"referencia"`
	ActorRef   string `json:"actor_ref"`
	MotivoRef  string `json:"motivo_ref"`
	Fecha      string `json:"fecha"`
}

// Hecho es una instantánea versionada. No contiene puntos, relación de empleo,
// identidad civil, documento completo ni mecanismos para acreditar o rectificar.
type Hecho struct {
	Referencia   string                           `json:"referencia"`
	PersonaRef   string                           `json:"persona_ref"`
	Version      int                              `json:"version"`
	Tipo         string                           `json:"tipo"`
	ConceptoRef  string                           `json:"concepto_ref"`
	Denominacion string                           `json:"denominacion"`
	Horas        *int                             `json:"horas,omitempty"`
	Procedencia  Procedencia                      `json:"procedencia"`
	Vigencia     Vigencia                         `json:"vigencia"`
	Estado       Estado                           `json:"estado"`
	Evidencias   []documentos.ReferenciaDocumento `json:"evidencias"`
	Revision     *Revision                        `json:"revision,omitempty"`
}

// Validar comprueba integridad estructural, nunca acreditación, firma o custodia.
func (h Hecho) Validar() error {
	if !ReferenciaValida(h.Referencia) || !ReferenciaValida(h.PersonaRef) || h.Version < 1 ||
		!ReferenciaValida(h.ConceptoRef) || !textoValido(h.Denominacion, 512) ||
		!ReferenciaValida(h.Procedencia.FuenteRef) || !ReferenciaValida(h.Procedencia.Version) ||
		!ReferenciaValida(h.Procedencia.HechoOrigenRef) || !instanteValido(h.Procedencia.CapturadaEn) ||
		!fechaValida(h.Vigencia.Desde) || (h.Vigencia.Hasta != "" && (!fechaValida(h.Vigencia.Hasta) || h.Vigencia.Hasta < h.Vigencia.Desde)) ||
		len(h.Evidencias) > 32 {
		return ErrHecho
	}
	switch h.Tipo {
	case "titulacion", "curso_asistencia", "curso_superacion", "experiencia", "idioma", "otro":
	default:
		return ErrHecho
	}
	if h.Horas != nil && (*h.Horas < 0 || (h.Tipo != "curso_asistencia" && h.Tipo != "curso_superacion")) {
		return ErrHecho
	}
	switch h.Estado {
	case Declarado, Pendiente:
	case Acreditado, Rechazado:
		if h.Revision == nil || (h.Estado == Acreditado && len(h.Evidencias) == 0) {
			return ErrHecho
		}
	default:
		return ErrHecho
	}
	if h.Revision != nil && (!ReferenciaValida(h.Revision.Referencia) || !ReferenciaValida(h.Revision.ActorRef) ||
		!ReferenciaValida(h.Revision.MotivoRef) || !instanteValido(h.Revision.Fecha)) {
		return ErrHecho
	}
	seen := make(map[documentos.ReferenciaDocumento]bool)
	for _, ev := range h.Evidencias {
		if ev.Validar() != nil || !ReferenciaValida(ev.ID) || seen[ev] {
			return ErrHecho
		}
		seen[ev] = true
	}
	return nil
}

// ReferenciaValida acepta identificadores opacos, sin exigir un prefijo que
// conceda titularidad. Su emisor conserva la autoridad y correspondencia real.
func ReferenciaValida(s string) bool {
	if s == "" || len(s) > 256 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(":._-", r)) {
			return false
		}
	}
	return true
}

func textoValido(s string, limite int) bool {
	return s != "" && len(s) <= limite && strings.TrimSpace(s) == s && !strings.ContainsFunc(s, unicode.IsControl)
}

func fechaValida(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil && len(s) == 10
}

func instanteValido(s string) bool {
	_, err := time.Parse(time.RFC3339Nano, s)
	return err == nil && len(s) <= 40
}
