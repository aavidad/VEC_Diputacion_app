package ports

import (
	"context"
	"errors"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrRespuestasPortalRRHHNoDisponibles = errors.New("bolsa: respuestas del llamamiento no disponibles")

// ConsultaRespuestasPortalRRHH liga la lectura al contexto nominal ya resuelto
// para el GET de candidatos. La bolsa y el llamamiento proceden de la ruta
// validada; la identidad y el perfil no proceden de cabeceras del cliente.
type ConsultaRespuestasPortalRRHH struct {
	BolsaRef, LlamamientoRef string
	Vinculo                  core.VinculoAutenticacionActorV2
	ResultadoContexto        core.ResultadoContextoActorRegistradoV2
	Captura                  vecports.CapturaLecturaNominalRespuestasLlamamiento
}

type RespuestaPortalRRHH struct {
	ParticipacionRef string `json:"participacion_ref"`
	Respuesta        string `json:"respuesta"`
}

// RespuestasPortalLlamamientoRRHH contiene solo la decisión firme necesaria
// para decidir si el seguimiento puede ofrecer otro llamamiento.
type RespuestasPortalLlamamientoRRHH struct {
	LlamamientoRef string
	Respuestas     []RespuestaPortalRRHH
}

type LectorRespuestasPortalRRHH interface {
	ConsultarRespuestasPortalLlamamiento(context.Context, ConsultaRespuestasPortalRRHH) (RespuestasPortalLlamamientoRRHH, error)
}
