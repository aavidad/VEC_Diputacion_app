package bootstrap

import (
	"net/http"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

// esRutaContratacionTemporalDesarrollo enumera las rutas del perfil de
// desarrollo que el revalidador protege con la identidad mTLS.
func esRutaContratacionTemporalDesarrollo(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if _, noCompuesta := rutasCapacidadNoCompuestaContratacionTemporal[r.URL.Path]; noCompuesta {
		return true
	}
	return rutaContinuidadNominal(r.URL.Path) || r.URL.Path == httpinterno.RutaConsultaSeguimientoV2 || r.URL.Path == httpinterno.RutaFichaGINPIXV2 || r.URL.Path == httpinterno.RutaIncorporacionEjercicioV2 || r.URL.Path == httpinterno.RutaResolucionFormalizacion || r.URL.Path == rutaEntregaPeticionCentro || rutaPeticionCentroDesarrollo(r.URL.Path) || rutaAnalisisContratacionTemporalDesarrollo(r.URL.Path) ||
		r.URL.Path == httpinterno.RutaResolucionComunicacionLlamamiento ||
		r.URL.Path == httpinterno.RutaContinuacionLlamamiento ||
		r.URL.Path == httpinterno.RutaRegistroRespuestaRecibida ||
		r.URL.Path == httpinterno.RutaEventoPlazoLlamamiento ||
		r.URL.Path == httpinterno.RutaRegistroComunicacionLlamamiento ||
		r.URL.Path == httpinterno.RutaResultadosFiscalizacion ||
		r.URL.Path == httpinterno.RutaSubsanacionReparos ||
		r.URL.Path == httpinterno.RutaEstadisticasRRHH ||
		bolsapersonal.EsRutaPortal(r.URL.Path) ||
		r.URL.Path == httpinterno.RutaAltaSolicitudes ||
		r.URL.Path == httpinterno.RutaPropuestaCobertura ||
		r.URL.Path == httpinterno.RutaDecisionCobertura ||
		r.URL.Path == httpinterno.RutaRectificacionCobertura ||
		r.URL.Path == httpinterno.RutaResultadoCobertura ||
		r.URL.Path == httpinterno.RutaAsignaciones ||
		r.URL.Path == httpinterno.RutaReasignaciones ||
		r.URL.Path == httpinterno.RutaPreparacionesInformeJuridico ||
		r.URL.Path == rutaBolsasRRHHDesarrollo || rutaBolsasCandidatosRRHHDesarrollo(r.URL.Path) || rutaBolsasOperacionesRRHHDesarrollo(r.URL.Path) ||
		r.URL.Path == rutaEstadisticasBolsaRRHHDesarrollo ||
		r.URL.Path == rutaAvisosBolsaRRHHDesarrollo ||
		r.URL.Path == bolsahttp.RutaPlazoRespuestaLlamamiento ||
		r.URL.Path == rutaReglasSituacionBolsaDesarrollo ||
		r.URL.Path == rutaCatalogosAltaContratacionTemporalDesarrollo ||
		r.URL.Path == rutaOrganizacionContratacionTemporalDesarrollo ||
		r.URL.Path == rutaCambiosOrganizacionContratacionTemporalDesarrollo ||
		r.URL.Path == rutaConfiguracionAnalisisContratacionTemporalDesarrollo ||
		r.URL.Path == rutaCircuitoFirmaContratacionTemporalDesarrollo ||
		rutaCalendariosDesarrollo(r.URL.Path) || rutaDocumentacionFormalizacionDesarrollo(r.URL.Path) ||
		rutaReglasVigentesDesarrollo(r.URL.Path)
}
