package bootstrap

import (
	"context"

	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Mapeo de configuración del ejercicio: ficha de trazabilidad, no formato
// homologado por GINPIX ni orden de transmisión a un sistema externo.
type mapeoFichaGINPIXDesarrollo struct{ m dom.MapeoVersionadoGINPIX }

func nuevoMapeoFichaGINPIXDesarrollo() (mapeoFichaGINPIXDesarrollo, error) {
	claves := []dom.ClaveCatalogo{"actuacion_ref", "expediente_ref", "recibo_ref", "relacion_ref", "seguimiento_ref", "solicitud_personal_ref"}
	reglas := make([]dom.ReglaMapeoGINPIX, 0, len(claves))
	for _, k := range claves {
		reglas = append(reglas, dom.ReglaMapeoGINPIX{CampoCanonico: k, CampoDestino: k, Obligatorio: true})
	}
	m, err := dom.PublicarMapeoVersionadoGINPIX(dom.BorradorMapeoVersionadoGINPIX{
		Esquema: dom.EsquemaMapeoGINPIXV1, Referencia: "mapeo:ct:ginpix:ficha-incorporacion-ejercicio", Version: 1,
		ProcedenciaRef: "configuracion:ct:ginpix:ficha-trazabilidad-ejercicio:v1", Reglas: reglas,
	})
	return mapeoFichaGINPIXDesarrollo{m}, err
}
func (m mapeoFichaGINPIXDesarrollo) ResolverMapeoFichaGINPIXV2(ctx context.Context, _ ct.ReciboIncorporacionAplicacionV2) (dom.MapeoVersionadoGINPIX, error) {
	if ctx == nil {
		return dom.MapeoVersionadoGINPIX{}, ct.ErrComposicionIncorporacionAplicacion
	}
	if err := ctx.Err(); err != nil {
		return dom.MapeoVersionadoGINPIX{}, err
	}
	return dom.RestaurarMapeoVersionadoGINPIX(m.m.Publicacion())
}
