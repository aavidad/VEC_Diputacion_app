package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarPoliticaCese    = "bolsa.politica_cese.consultar"
	FinalidadConsultarPoliticaCese = "consulta_politica_cese_rrhh"
	AudienciaConsultarPoliticaCese = "vec_bolsa_llamamientos.politica_cese.consultar.v1"
	RecursoPoliticaCeseRef         = "politica:bolsa:cese:vigente"
	TipoRecursoPoliticaCese        = "politica_cese_bolsa"
	CampoPoliticaCese              = "politica_cese"
)

var ErrConsultaPoliticaCeseNoDisponible = errors.New("bolsa: consulta de politica de cese no disponible")

type OrdenConsultaPoliticaCese struct {
	ResultadoContexto vd.ResultadoContextoActorRegistradoV2
	Vinculo           vd.VinculoAutenticacionActorV2
	Motivo            vd.ReferenciaEntradaCatalogo
	Correlacion       vd.ReferenciaCorrelacionAutorizacionV2
}

type ConsultaPoliticaCese interface {
	ConsultarPoliticaCese(context.Context, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.PoliticaCese, error)
}

type AutorizadorPoliticaCeseV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vd.SolicitudAutorizacionLigadaV3, vd.ResultadoContextoActorRegistradoV2) (vd.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}
