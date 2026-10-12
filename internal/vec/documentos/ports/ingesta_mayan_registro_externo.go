package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// FuentePlanConfirmadoRegistroExterno debe obtener el plan D16 con una
// autorización V3 fresca, ligada al actor V2 y al recurso, antes de leerlo.
// La referencia opaca del plan por sí sola nunca concede acceso.
type FuentePlanConfirmadoRegistroExterno interface {
	ConsultarPlanConfirmadoParaRegistro(context.Context, string,
		vecdomain.ResultadoContextoActorRegistradoV2,
		vecdomain.VinculoAutenticacionActorV2) (PlanIngestaMayan, error)
}

// PoliticaRegistroPlanMayan obtiene la solicitud publicada por el servidor.
// No acepta política, tipo documental ni fecha provenientes del navegador.
type PoliticaRegistroPlanMayan interface {
	SolicitudParaPlan(context.Context, PlanIngestaMayan) (vecports.SolicitudPoliticaConservacionDocumental, error)
}

// RegistradorExternoMayan reutiliza el efecto D3, su V3 y su recibo durable.
type RegistradorExternoMayan interface {
	RegistrarExternoAutorizado(context.Context, AltaExterna, AutorizadorRegistroExterno) (domain.Documento, error)
}
