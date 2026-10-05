package bootstrap

import (
	"context"
	"net/http"
	"net/url"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// correlacionConsultaContactos liga la V3 y el intento AD169 de las dos
// lecturas HTTP de contactos a la correlación de la petición, como la consulta
// documental. Son lecturas sin idempotencia; las escrituras de contactos no
// pasan por aquí y conservan su correlación propia. Fuera de esas rutas (la
// página RRHH de la bolsa pagina varias lecturas en una sola petición) se
// genera una correlación nueva por lectura, como hasta ahora.
func (p *preparadorBorradorLlamamientoDesarrollo) correlacionConsultaContactos(ctx context.Context) (dominiovec.ReferenciaCorrelacionAutorizacionV2, error) {
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if ok && rutaConsultaContactosBolsaDesarrollo(capacidad.ruta) {
		return puertosvec.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	}
	return dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.generar)
}

func rutaConsultaContactosBolsaDesarrollo(ruta string) bool {
	if ruta == bolsahttp.RutaContactosOferta {
		return true
	}
	_, _, ok := bolsahttp.ReferenciasRutaContactosParticipacion(&http.Request{URL: &url.URL{Path: ruta}})
	return ok
}
