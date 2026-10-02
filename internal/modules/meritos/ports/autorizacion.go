package ports

import (
	"context"

	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Autorizador usa el emisor común. Estas firmas no publican acciones, perfiles
// ni concesiones; el gobierno nominal de Méritos sigue pendiente de D/RUM03.
type Autorizador interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vec.SolicitudAutorizacionLigadaV3, vec.ResultadoContextoActorRegistradoV2) (vec.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type AutorizacionOperacion struct {
	Contexto     vec.ResultadoContextoActorRegistradoV2
	Solicitud    vec.SolicitudAutorizacionLigadaV3
	Decision     vec.DecisionAutorizacionLigadaV3
	Confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	Material     vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}
