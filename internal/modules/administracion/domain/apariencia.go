package domain

import (
	"errors"
	"math"
	"strings"
	"time"
)

var (
	ErrTemaInvalido             = errors.New("administracion: tema fuera del catálogo")
	ErrEstadoAparienciaInvalido = errors.New("administracion: estado global de apariencia inválido")
	ErrOrdenPublicacionInvalida = errors.New("administracion: orden de publicación inválida")
)

type TemaID string

const (
	TemaInstitucional  TemaID = "institucional"
	TemaGranate        TemaID = "granate"
	RevisionTemaActual uint64 = 1
)

// Tema identifica un paquete incluido en el artefacto. No contiene CSS, URL ni código.
type Tema struct {
	ID       TemaID
	Revision uint64
}

func (t Tema) Validar() error {
	if (t.ID != TemaInstitucional && t.ID != TemaGranate) || t.Revision != RevisionTemaActual {
		return ErrTemaInvalido
	}
	return nil
}

// EstadoApariencia es la revisión global confirmada; cero representa ausencia de publicación.
type EstadoApariencia struct {
	Configurado    bool
	Tema           Tema
	RevisionGlobal uint64
	ReciboRef      string
	HistoriaRef    string
	PublicadaEn    time.Time
}

func (e EstadoApariencia) Validar() error {
	if !e.Configurado {
		if e.Tema != (Tema{}) || e.RevisionGlobal != 0 || e.ReciboRef != "" ||
			e.HistoriaRef != "" || !e.PublicadaEn.IsZero() {
			return ErrEstadoAparienciaInvalido
		}
		return nil
	}
	if e.Tema.Validar() != nil || e.RevisionGlobal == 0 ||
		!ReferenciaAparienciaValida(e.ReciboRef) || !ReferenciaAparienciaValida(e.HistoriaRef) ||
		e.PublicadaEn.IsZero() || e.PublicadaEn.Location() != time.UTC {
		return ErrEstadoAparienciaInvalido
	}
	return nil
}

// OrdenPublicacion no transporta identidad ni permisos. La revisión global
// esperada es obligatoria incluso para la primera publicación (valor cero).
type OrdenPublicacion struct {
	Tema                   Tema
	RevisionGlobalEsperada uint64
	ClaveIdempotencia      string
}

func (o OrdenPublicacion) Validar() error {
	if o.Tema.Validar() != nil || o.RevisionGlobalEsperada == math.MaxUint64 ||
		!ClaveIdempotenciaValida(o.ClaveIdempotencia) {
		return ErrOrdenPublicacionInvalida
	}
	return nil
}

func (o OrdenPublicacion) RevisionSiguiente() (uint64, error) {
	if err := o.Validar(); err != nil {
		return 0, err
	}
	return o.RevisionGlobalEsperada + 1, nil
}

func ReferenciaAparienciaValida(ref string) bool {
	return len(ref) >= 8 && len(ref) <= 128 && ref == strings.TrimSpace(ref) && sinControles(ref)
}

func ClaveIdempotenciaValida(clave string) bool {
	return len(clave) >= 16 && len(clave) <= 128 && clave == strings.TrimSpace(clave) && sinControles(clave)
}

func sinControles(s string) bool {
	for _, r := range s {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	return true
}
