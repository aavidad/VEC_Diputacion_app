package internactproveedores

import (
	"context"

	"vec-diputacion-granada/internal/app/composicion/gobiernov3lector"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Cada emisión toma la configuración publicada justo antes de autorizar.
// El servicio resultante queda fijo durante esa emisión.
type emisorMaterialRenovable struct {
	lector    *gobiernov3lector.Lector
	pdp       vp.AutorizadorSolicitudLigadaV3
	atestador *app.ServicioAtestacionesAutorizacionV3
	capacidad *confianza.EmisorCapacidadesAtestacionAutorizacionV3
}

func (e *emisorMaterialRenovable) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	if e == nil || e.lector == nil || ctx == nil {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrProveedoresCTNoDisponibles
	}
	// El lector protege su publicación compartida y devuelve una instantánea
	// fija para esta emisión, sin serializar otras operaciones.
	snapshot, err := e.lector.Instantanea(ctx)
	if err != nil {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrProveedoresCTNoDisponibles
	}
	emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(e.pdp, e.atestador, snapshot, e.capacidad)
	if err != nil {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrProveedoresCTNoDisponibles
	}
	return emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
}
