package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// ServicioRespuestasPortalRRHH conserva la propiedad Bolsa de la respuesta.
// El adaptador debe cotejar lectura y auditoría común en una misma transacción.
type ServicioRespuestasPortalRRHH struct {
	lector ports.LectorRespuestasPortalRRHH
}

func NuevoServicioRespuestasPortalRRHH(lector ports.LectorRespuestasPortalRRHH) (*ServicioRespuestasPortalRRHH, error) {
	if lector == nil {
		return nil, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	return &ServicioRespuestasPortalRRHH{lector: lector}, nil
}

func (s *ServicioRespuestasPortalRRHH) ConsultarRespuestasPortalLlamamiento(
	ctx context.Context, q ports.ConsultaRespuestasPortalRRHH,
) (ports.RespuestasPortalLlamamientoRRHH, error) {
	vacia := ports.RespuestasPortalLlamamientoRRHH{}
	if s == nil || s.lector == nil || ctx == nil || ctx.Err() != nil ||
		q.BolsaRef == "" || q.LlamamientoRef == "" || q.ResultadoContexto.Validar() != nil ||
		q.Vinculo.ValidarPara(q.ResultadoContexto) != nil ||
		q.Captura.BolsaRef != q.BolsaRef || q.Captura.LlamamientoRef != q.LlamamientoRef {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	r, err := s.lector.ConsultarRespuestasPortalLlamamiento(ctx, q)
	if err != nil || r.LlamamientoRef != q.LlamamientoRef || r.Respuestas == nil || len(r.Respuestas) > 100 {
		return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
	}
	vistas := make(map[string]bool, len(r.Respuestas))
	for _, respuesta := range r.Respuestas {
		if respuesta.ParticipacionRef == "" || vistas[respuesta.ParticipacionRef] ||
			(respuesta.Respuesta != "acepta" && respuesta.Respuesta != "renuncia" && respuesta.Respuesta != "renuncia_justificada") {
			return vacia, ports.ErrRespuestasPortalRRHHNoDisponibles
		}
		vistas[respuesta.ParticipacionRef] = true
	}
	return r, nil
}
