package httpapi

import (
	"context"
	"net/http"

	"vec-diputacion-granada/internal/vec/ports"
)

// RegistrarDenegacionFronteraPreparacion anota un rechazo previo al contexto
// personal usando el registrador común. El binder de la raíz comprueba antes
// la capacidad sellada y la misma instancia del catálogo para método/ruta.
// Esta función no autentica ni concede acceso, y no recibe identidad,
// superficie, cuerpo ni cabeceras. El llamador responde 503 si devuelve error.
func RegistrarDenegacionFronteraPreparacion(
	ctx context.Context,
	registrador ports.RegistradorAuditoriaFronteraRutaExacta,
	metodo, ruta string,
	motivo ports.MotivoAuditoriaFronteraRutaExacta,
) error {
	if ctx == nil || ctx.Err() != nil || metodo != http.MethodPost {
		return ErrAutoridadRutaExactaNoDisponible
	}
	superficie := superficieAuditoriaFronteraRutaExacta(ruta)
	if superficie != ports.SuperficieAuditoriaFronteraRutaExactaSeleccionPreparacionBases &&
		superficie != ports.SuperficieAuditoriaFronteraRutaExactaBolsaReglasBaremo {
		return ErrAutoridadRutaExactaNoDisponible
	}
	orden := ports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: "corr_no_disponible",
		Motivo:         motivo, Superficie: superficie, Ruta: ruta,
	}
	if orden.Validar() != nil {
		return ErrAutoridadRutaExactaNoDisponible
	}
	var rechazo error
	switch motivo {
	case ports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado:
		rechazo = ErrAccesoRutaExactaDenegado
	case ports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida:
		rechazo = ErrAutenticacionRutaExactaRequerida
	default:
		return ErrAutoridadRutaExactaNoDisponible
	}
	h := Handler{registradorAuditoriaFronteraRutasExactas: registrador}
	return h.registrarDenegacionRutaExacta(ctx, ruta, rechazo, nuevaCorrelacionRutaExacta())
}
