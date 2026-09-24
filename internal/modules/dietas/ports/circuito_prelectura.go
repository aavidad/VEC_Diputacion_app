package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaEfectoPrelecturaCircuitoV1 = "vec.dietas.circuito-prelectura.v1"

type SolicitudPrelecturaCircuito struct {
	Referencia string               `json:"referencia"`
	Etapa      domain.EtapaCircuito `json:"etapa"`
	UnidadRef  string               `json:"unidad_ref"`
}

// La respuesta minimizada no contiene motivo, rutas, gastos ni documento.
// SQL solo la entrega después de consumir la concesión nominal y comprobar
// la competencia del actor para la etapa solicitada.
type ContextoComisionCircuito struct {
	Referencia               string `json:"referencia"`
	RelacionRef              string `json:"relacion_ref"`
	UnidadRef                string `json:"unidad_ref"`
	Version                  uint64 `json:"version"`
	Estado                   string `json:"estado"`
	AsignacionRef            string `json:"asignacion_ref"`
	AsignacionVersion        uint64 `json:"asignacion_version"`
	GrupoDieta               string `json:"grupo_dieta"`
	CentroRef                string `json:"centro_ref"`
	AdministrativoPersonaRef string `json:"administrativo_persona_ref"`
	ResponsablePersonaRef    string `json:"responsable_persona_ref"`
}

type AutorizacionPrelecturaCircuitoDurable struct {
	Material   vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Accion     string
	RecursoRef string
	Finalidad  string
}

type IdentidadEfectivaPrelecturaCircuito struct {
	Vinculo              vecdomain.VinculoAutenticacionActorV2
	ContextoRegistrado   vecdomain.ResultadoContextoActorRegistradoV2
	Autorizacion         AutorizacionPrelecturaCircuitoDurable
	UnidadCompetenciaRef string
}

type ResolutorIdentidadPrelecturaCircuito interface {
	ResolverIdentidadPrelecturaCircuito(context.Context, SolicitudPrelecturaCircuito) (IdentidadEfectivaPrelecturaCircuito, error)
}

type RepositorioPrelecturaCircuito interface {
	Preleer(context.Context, IdentidadEfectivaPrelecturaCircuito, SolicitudPrelecturaCircuito) (ContextoComisionCircuito, error)
}
