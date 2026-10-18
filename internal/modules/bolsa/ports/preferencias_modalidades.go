package ports

import (
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

// ModalidadPreferencia procede del catálogo gobernado común; su clave es la
// misma que conserva la oferta publicada, sin consultar el análisis actual.
type ModalidadPreferencia struct {
	Clave  string
	Fuente core.ReferenciaEntradaCatalogo
}

// EstadoPreferenciasModalidades tiene versión y recibo independientes del
// orden de bolsas. La versión cero expresa que todavía no hay una elección.
type EstadoPreferenciasModalidades struct {
	Version                uint64
	ModalidadesGuardadas   []string
	ModalidadesDisponibles []ModalidadPreferencia
	ReciboRef              string
	RegistradoEn           *time.Time
}
