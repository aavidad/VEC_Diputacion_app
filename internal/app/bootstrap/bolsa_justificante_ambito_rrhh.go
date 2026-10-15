package bootstrap

import (
	"context"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Reutiliza la lectura certificada de las respuestas de Bolsa. Solo el ámbito
// cotejado por su propietario puede llegar después al emisor documental.
type capturadorAmbitoJustificanteBolsaRRHH struct {
	fuente vecports.FuenteAutorizacion
}

func (c capturadorAmbitoJustificanteBolsaRRHH) CapturarAmbitoJustificanteRenunciaRRHH(ctx context.Context,
	q bolsaports.SolicitudConsultaJustificanteRenunciaRRHH, a bolsaports.AmbitoJustificanteRenunciaRRHH) (vecports.CapturaLecturaNominalBolsa, error) {
	if a.BolsaRef != q.BolsaRef || a.ParticipacionRef != q.ParticipacionRef {
		return vecports.CapturaLecturaNominalBolsa{}, vec.ErrAutorizacionDenegada
	}
	return CapturarLecturaRespuestasLlamamiento(ctx, c.fuente, q.VinculoV2, q.ActorRegistradoV2,
		a.UnidadRef, a.AmbitoRef, a.BolsaRef, a.LlamamientoRef)
}
