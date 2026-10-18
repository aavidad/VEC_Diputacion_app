package ports

import (
	"context"
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
// Una selección nil muestra todas; una lista vacía no nula expresa ninguna.
type EstadoPreferenciasModalidades struct {
	Version                uint64
	ModalidadesGuardadas   []string
	ModalidadesDisponibles []ModalidadPreferencia
	ReciboRef              string
	RegistradoEn           *time.Time
}

// CatalogoPreferenciasModalidades aporta las opciones y sus referencias desde
// el catálogo común vigente. Nunca recibe claves ni fuentes del navegador.
type CatalogoPreferenciasModalidades interface {
	ConsultarModalidadesDisponibles(context.Context) ([]ModalidadPreferencia, error)
}
