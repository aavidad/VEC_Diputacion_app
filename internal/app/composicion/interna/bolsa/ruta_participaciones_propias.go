package bolsa

import (
	"net/http"

	httpinternobolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const rutaParticipacionesPropiasB11 = httpinternobolsa.RutaParticipacionesPropias

// DependenciasRutaParticipacionesPropiasB11 declara la unica composicion
// admisible para B11. El extractor solo puede venir de la frontera de proxy
// interna ya confiable; la ruta no conoce nombres de cabecera ni credenciales.
type DependenciasRutaParticipacionesPropiasB11 struct {
	Autenticador  autenticadorCanalB11
	Sesiones      resolvedorPeticionSesionB11
	Extractor     ExtractorSobrePeticionInterno
	Preparador    httpinternobolsa.PreparadorOrdenConsultaParticipacionesPropias
	Consultor     httpinternobolsa.ConsultorParticipacionesPropias
	Denegaciones  puertosvec.RegistradorDenegacionFronteraIdentidadV1
	Correlaciones puertosvec.GeneradorReferenciasAutorizacionV2
}

func NuevaRutaParticipacionesPropiasB11(d DependenciasRutaParticipacionesPropiasB11) (http.Handler, error) {
	handler, err := httpinternobolsa.NuevoHandlerParticipacionesPropias(d.Preparador, d.Consultor)
	if err != nil {
		return nil, ErrRutaParticipacionesPropiasB11Invalida
	}
	return NuevoMiddlewarePeticionSesionB11(
		d.Autenticador, d.Sesiones, d.Extractor, d.Denegaciones, d.Correlaciones, handler,
	)
}
