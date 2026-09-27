package ports

import (
	"context"
	"errors"
	"time"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrReincorporacionTitularNoDisponible = errors.New("bolsa: reincorporacion del titular no disponible")

// PublicacionReincorporacionTitularCT contiene solo la triple necesaria para
// que la bandeja compruebe el hecho original en CT. La fecha y el cese los
// obtiene Bolsa exclusivamente del verificador CT130.
type PublicacionReincorporacionTitularCT struct {
	EventoRef, OrigenRef, HuellaSHA256 string
	OrigenPosicion                     int64
}

type ResultadoReincorporacionTitularCT struct {
	Reutilizada, CeseAplicado bool
	Estado                    string
	CeseEventoRef             string
	DisponibleDesde           *time.Time
}

type ReincorporacionTitularFicha struct {
	EventoRef, ExpedienteRef, RelacionRef, ReciboCTRef, CeseEventoRef string
	FechaEfectiva                                                     time.Time
	Estado                                                            string
	DisponibleDesde                                                   *time.Time
	ReglaVersion                                                      *int64
	ReglaHuellaSHA256                                                 string
}

type FuenteReincorporacionesTitularCT interface {
	LeerReincorporacionesTitular(context.Context, *CursorContratosParticipacion, int) ([]PublicacionReincorporacionTitularCT, error)
}

type BuzonReincorporacionesTitularCT interface {
	CursorReincorporacionesTitular(context.Context) (CursorContratosParticipacion, bool, error)
	RegistrarReincorporacionTitular(context.Context, PublicacionReincorporacionTitularCT) (ResultadoReincorporacionTitularCT, error)
}

type RepositorioReincorporacionesTitularCT interface {
	ListarReincorporacionesTitular(context.Context, string, string, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ReincorporacionTitularFicha, error)
}
