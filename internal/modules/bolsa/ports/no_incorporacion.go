package ports

import (
	"context"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// ConsecuenciaNoIncorporacion es lo que el catálogo de Bolsa fija para la
// clave de consecuencia del evento; viaja con él a la bandeja, que la aplica
// una sola vez. Nil si el catálogo no reconoce la clave: el evento se
// conserva sin efecto para revisión.
type ConsecuenciaNoIncorporacion struct {
	Clave                    string  `json:"clave"`
	Etiqueta                 string  `json:"etiqueta"`
	Efecto                   string  `json:"efecto"`
	ReglaRef                 string  `json:"regla_ref"`
	ReglaHuellaSHA256        string  `json:"regla_huella_sha256"`
	SuspensionHasta          *string `json:"suspension_hasta"`
	RecursoVence             string  `json:"recurso_vence"`
	RecursoReglaRef          string  `json:"recurso_regla_ref"`
	RecursoReglaHuellaSHA256 string  `json:"recurso_regla_huella_sha256"`
	OrdenFinal               bool    `json:"orden_final"`
	FinAutomatico            bool    `json:"fin_automatico"`
}

// EventoNoIncorporacionRecibido conserva el contenido exacto y su huella.
type EventoNoIncorporacionRecibido struct {
	Evento         dominiobolsa.EventoNoIncorporacion
	Contenido      []byte
	HuellaSHA256   string
	OrigenCreadaEn time.Time
	OrigenPosicion int64
	Consecuencia   *ConsecuenciaNoIncorporacion
}

// ResultadoRegistroNoIncorporacion: Estado dice qué efecto tuvo («aplicada»,
// «participacion_no_constituida», «sin_aceptacion»...).
type ResultadoRegistroNoIncorporacion struct {
	Reutilizado      bool
	Estado           string
	ParticipacionRef string
}

// BuzonNoIncorporaciones es la bandeja idempotente de Bolsa 000042.
type BuzonNoIncorporaciones interface {
	CursorNoIncorporaciones(context.Context) (CursorContratosParticipacion, bool, error)
	RegistrarNoIncorporacion(context.Context, EventoNoIncorporacionRecibido) (ResultadoRegistroNoIncorporacion, error)
}

// ResolvedorSancionNoIncorporacion resuelve la consecuencia con el catálogo
// de sanciones de Bolsa (b24.sancion.*).
type ResolvedorSancionNoIncorporacion interface {
	ResolverSancion(ctx context.Context, clave string, notificadaEn time.Time) (ResolucionCatalogoSancion, error)
}
