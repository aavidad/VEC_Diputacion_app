package bootstrap

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	ajusteshttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/ajustesreglas"
	plantillashttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

var ErrCoberturaRutasCTDesarrollo = errors.New("bootstrap: cobertura de rutas CT incompleta")

// Una guardia vacía exige frontera del PDP común. Las demás entradas nombran
// la autoridad local que debe revisar el cambio de ruta en una revisión
// sensible; este inventario no concede permisos ni sustituye esa autoridad.
type metodoRutaCTDesarrollo struct {
	metodo  string
	guardia string
}

func pdpCT(metodo string) metodoRutaCTDesarrollo { return metodoRutaCTDesarrollo{metodo: metodo} }
func nominalCT(metodo, guardia string) metodoRutaCTDesarrollo {
	return metodoRutaCTDesarrollo{metodo: metodo, guardia: guardia}
}

// inventarioRutasCTDesarrollo es deliberadamente cerrado. Las rutas opcionales
// sólo se comprueban si la composición realmente las registra. No incluye
// rutas de Bolsa, Calendarios, Usuarios ni Documentos aunque compartan slice.
func inventarioRutasCTDesarrollo() map[string][]metodoRutaCTDesarrollo {
	const (
		centro      = "contratacion_temporal_peticion_centro_http_desarrollo.go:manejadorPeticionCentroDesarrollo"
		continuidad = "contratacion_temporal_continuidad_nominal.go:autoridadContinuidadNominal"
		firmasR5V2  = "contratacion_temporal_firmas_r5_v2_desarrollo.go:fuenteNominalFirmasR5V2CTDesarrollo"
	)
	return map[string][]metodoRutaCTDesarrollo{
		// PDP común: 21 pares base, POST de entrega, reincorporación y plantillas opcionales.
		httpinterno.RutaRegistroAnalisisRRHH:            {pdpCT(http.MethodPost)},
		httpinterno.RutaRectificacionAnalisisRRHH:       {pdpCT(http.MethodPost)},
		httpinterno.RutaAltaSolicitudes:                 {pdpCT(http.MethodPost)},
		httpinterno.RutaAsignaciones:                    {pdpCT(http.MethodPost)},
		httpinterno.RutaPreparacionesInformeJuridico:    {pdpCT(http.MethodPost)},
		httpinterno.RutaSubsanacionReparos:              {pdpCT(http.MethodPost)},
		httpinterno.RutaDecisionCobertura:               {pdpCT(http.MethodPost)},
		httpinterno.RutaRectificacionCobertura:          {pdpCT(http.MethodPost)},
		httpinterno.RutaConsultaCuadroRRHH:              {pdpCT(http.MethodPost)},
		httpinterno.RutaConsultaDetalleRRHH:             {pdpCT(http.MethodPost)},
		httpinterno.RutaConsultaCircuitoRRHH:            {pdpCT(http.MethodPost)},
		httpinterno.RutaPropuestaCobertura:              {pdpCT(http.MethodPost)},
		httpinterno.RutaResultadoCobertura:              {pdpCT(http.MethodPost)},
		httpinterno.RutaCesesNombramiento:               {pdpCT(http.MethodPost)},
		httpinterno.RutaCierresExpediente:               {pdpCT(http.MethodPost)},
		httpinterno.RutaModificacionesNombramiento:      {pdpCT(http.MethodPost)},
		httpinterno.RutaSeguimientoCese:                 {pdpCT(http.MethodPost)},
		httpinterno.RutaConfirmacionesGINPIX:            {pdpCT(http.MethodPost)},
		httpinterno.RutaNoIncorporaciones:               {pdpCT(http.MethodPost)},
		httpinterno.RutaCancelacionesExpediente:         {pdpCT(http.MethodPost)},
		httpinterno.RutaCancelacionExpediente:           {pdpCT(http.MethodPost)},
		httpinterno.RutaVinculosEmisionBolsa:            {pdpCT(http.MethodPost)},
		rutaEntregaPeticionCentro:                       {pdpCT(http.MethodGet), pdpCT(http.MethodPost)},
		httpinterno.RutaReincorporacionesTitular:        {pdpCT(http.MethodPost)},
		httpinterno.RutaCapacidadReincorporacionTitular: {pdpCT(http.MethodPost)},
		plantillashttp.RutaCatalogo:                     {pdpCT(http.MethodGet)},
		ajusteshttp.Ruta:                                {pdpCT(http.MethodGet), pdpCT(http.MethodPost)},
		plantillashttp.RutaEntradas:                     {pdpCT(http.MethodPost)},
		plantillashttp.RutaPublicar:                     {pdpCT(http.MethodPost)},
		plantillashttp.RutaBorradoresDisponibles:        {pdpCT(http.MethodPost)},
		plantillashttp.RutaBorradores:                   {pdpCT(http.MethodPost)},
		// Llamamiento: su autoridad decide en el PDP común (frontera por ruta
		// y método en contratacion_temporal_llamamiento_pdp_comun_desarrollo.go).
		httpinterno.RutaSeleccionLlamamiento:              {pdpCT(http.MethodPost)},
		httpinterno.RutaRegistroComunicacionLlamamiento:   {pdpCT(http.MethodPost)},
		httpinterno.RutaConsultaComunicacionesExpediente:  {pdpCT(http.MethodGet)},
		httpinterno.RutaRegistroRespuestaRecibida:         {pdpCT(http.MethodPost)},
		httpinterno.RutaConsultaReciboRespuesta:           {pdpCT(http.MethodGet)},
		httpinterno.RutaResolucionComunicacionLlamamiento: {pdpCT(http.MethodPost)},
		httpinterno.RutaContinuacionLlamamiento:           {pdpCT(http.MethodPost)},
		httpinterno.RutaEventoPlazoLlamamiento:            {pdpCT(http.MethodPost)},
		httpinterno.RutaPropuestaFormalizacion:            {pdpCT(http.MethodPost)},

		// Autoridades nominales existentes. Una lectura sin acción V3 propia
		// conserva su guarda de identidad/ruta; no se inventa una acción V3.
		httpinterno.RutaReasignaciones:                          {nominalCT(http.MethodPost, "contratacion_temporal_asignacion_desarrollo.go:nuevasDependenciasAsignacionContratacionTemporalDesarrollo")},
		httpinterno.RutaResultadosFiscalizacion:                 {nominalCT(http.MethodPost, "contratacion_temporal_fiscalizacion_desarrollo.go:soporteFiscalizacionContratacionTemporalDesarrollo")},
		httpinterno.RutaIncorporacionEjercicioV2:                {nominalCT(http.MethodGet, "contratacion_temporal_incorporacion_v2.go:ligarContextoIncorporacionV2Desarrollo"), nominalCT(http.MethodPost, "contratacion_temporal_incorporacion_v2.go:ligarContextoIncorporacionV2Desarrollo")},
		httpinterno.RutaFichaGINPIXV2:                           {nominalCT(http.MethodGet, "contratacion_temporal_incorporacion_v2.go:ligarContextoIncorporacionV2Desarrollo")},
		httpinterno.RutaConsultaSeguimientoV2:                   {nominalCT(http.MethodGet, "contratacion_temporal_incorporacion_v2.go:ligarContextoIncorporacionV2Desarrollo")},
		httpinterno.RutaAnotacionesAdministrativas:              {nominalCT(http.MethodPost, continuidad)},
		httpinterno.RutaRecuperacionAnotacionesAdministrativas:  {nominalCT(http.MethodGet, continuidad)},
		httpinterno.RutaCerrarAdministrativamenteSinCese:        {nominalCT(http.MethodPost, continuidad)},
		httpinterno.RutaPreparacionCierreSinCese:                {nominalCT(http.MethodGet, continuidad)},
		httpinterno.RutaResolucionFormalizacion:                 {nominalCT(http.MethodGet, "contratacion_temporal_resolucion_formalizacion_desarrollo.go:ejecutorResolucionFormalizacionDesarrollo"), pdpCT(http.MethodPost)},
		rutaOrganizacionContratacionTemporalDesarrollo:          {nominalCT(http.MethodGet, "contratacion_temporal_organizacion_desarrollo.go:manejadorOrganizacionContratacionTemporalDesarrollo"), nominalCT(http.MethodHead, "contratacion_temporal_organizacion_desarrollo.go:manejadorOrganizacionContratacionTemporalDesarrollo")},
		rutaCambiosOrganizacionContratacionTemporalDesarrollo:   {nominalCT(http.MethodPost, "contratacion_temporal_organizacion_edicion_desarrollo.go:proveedorOrganizacionDesarrollo")},
		httpinterno.RutaEstadisticasRRHH:                        {nominalCT(http.MethodGet, "contratacion_temporal_estadisticas_desarrollo.go:resolutorAlcanceEstadisticasRRHHDesarrollo"), nominalCT(http.MethodHead, "contratacion_temporal_estadisticas_desarrollo.go:resolutorAlcanceEstadisticasRRHHDesarrollo")},
		rutaOperacionesPeticionCentro:                           {nominalCT(http.MethodPost, centro)},
		rutaBandejaPeticionCentro:                               {nominalCT(http.MethodGet, centro)},
		rutaContextoPeticionCentro:                              {nominalCT(http.MethodGet, centro)},
		rutaBandejaIncorporacionCentro:                          {nominalCT(http.MethodGet, "contratacion_temporal_incorporacion_centro_desarrollo.go:incorporacionCentroDesarrollo")},
		rutaConfirmacionIncorporacionCentro:                     {nominalCT(http.MethodPost, "contratacion_temporal_incorporacion_centro_desarrollo.go:incorporacionCentroDesarrollo")},
		rutaCancelacionesCentro:                                 {nominalCT(http.MethodPost, "contratacion_temporal_cancelacion_centro_desarrollo.go:cancelacionCentroDesarrollo")},
		rutaCancelacionCentro:                                   {nominalCT(http.MethodPost, "contratacion_temporal_cancelacion_centro_desarrollo.go:cancelacionCentroDesarrollo")},
		httpinterno.RutaConsultaFirmasR5V2:                      {nominalCT(http.MethodPost, firmasR5V2)},
		httpinterno.RutaRecuperacionFirmasR5V2:                  {nominalCT(http.MethodPost, firmasR5V2)},
		rutaCatalogosAltaContratacionTemporalDesarrollo:         {nominalCT(http.MethodGet, "contratacion_temporal_catalogos_alta_desarrollo.go:manejadorCatalogosAltaContratacionTemporalDesarrollo"), nominalCT(http.MethodHead, "contratacion_temporal_catalogos_alta_desarrollo.go:manejadorCatalogosAltaContratacionTemporalDesarrollo")},
		rutaConfiguracionAnalisisContratacionTemporalDesarrollo: {nominalCT(http.MethodGet, "contratacion_temporal_analisis_desarrollo.go:manejadorConfiguracionAnalisisContratacionTemporalDesarrollo"), nominalCT(http.MethodHead, "contratacion_temporal_analisis_desarrollo.go:manejadorConfiguracionAnalisisContratacionTemporalDesarrollo")},
		rutaCircuitoFirmaContratacionTemporalDesarrollo:         {nominalCT(http.MethodGet, "contratacion_temporal_circuito_firma_desarrollo.go:manejadorCircuitoFirmaContratacionTemporalDesarrollo"), nominalCT(http.MethodHead, "contratacion_temporal_circuito_firma_desarrollo.go:manejadorCircuitoFirmaContratacionTemporalDesarrollo")},
		httpinterno.RutaDocumentacionFormalizacion:              {nominalCT(http.MethodGet, "contratacion_temporal_documentacion_formalizacion_desarrollo.go:nuevaRutaDocumentacionFormalizacionDesarrollo")},
		httpinterno.RutaFirmaDocumento:                          {pdpCT(http.MethodPost)},
		httpinterno.RutaConsultaFirmaDocumento:                  {pdpCT(http.MethodPost)},
		httpinterno.RutaPlanB2:                                  {pdpCT(http.MethodGet), pdpCT(http.MethodPost)},
		httpinterno.RutaConfirmacionB2:                          {pdpCT(http.MethodPost)},
		httpinterno.RutaVinculoCategoriaRPTB2:                   {pdpCT(http.MethodGet), pdpCT(http.MethodPost)},
		httpinterno.RutaCategoriasRPTB2:                         {pdpCT(http.MethodGet)},
		rutaCatalogosRegistroEmpleadoB2:                         {pdpCT(http.MethodPost)},
	}
}

// validarCoberturaRutasCTDesarrollo se ejecuta sobre el slice final, antes del
// dispatcher. Cada ruta CT necesita un contrato de métodos cerrado y cada par
// una frontera PDP resuelta o una autoridad nominal identificada. RutaExacta
// no declara verbos: cambios internos de un handler deben revisarse aparte.
func validarCoberturaRutasCTDesarrollo(rutas []vechttp.RutaExacta, fronteras catalogoFronterasComunDesarrollo) error {
	inventario := inventarioRutasCTDesarrollo()
	vistas := make(map[string]struct{})
	for _, ruta := range rutas {
		if !esPrefijoRutaCTDesarrollo(ruta.Ruta) {
			continue
		}
		if _, duplicada := vistas[ruta.Ruta]; duplicada || ruta.Manejador == nil {
			return fmt.Errorf("%w: ruta CT duplicada o sin manejador %q", ErrCoberturaRutasCTDesarrollo, ruta.Ruta)
		}
		vistas[ruta.Ruta] = struct{}{}
		metodos, conocida := inventario[ruta.Ruta]
		if !conocida || len(metodos) == 0 {
			return fmt.Errorf("%w: ruta CT no inventariada %q", ErrCoberturaRutasCTDesarrollo, ruta.Ruta)
		}
		for _, metodo := range metodos {
			if err := validarMetodoRutaCTDesarrollo(ruta.Ruta, metodo, fronteras); err != nil {
				return err
			}
		}
	}
	// La comprobación de composición también limita el transporte efectivo:
	// un manejador sustituto no puede atender otro verbo fuera del inventario.
	for i, ruta := range rutas {
		if esPrefijoRutaCTDesarrollo(ruta.Ruta) {
			rutas[i].Manejador = manejadorMetodosInventariadosCTDesarrollo{ruta.Ruta, inventario[ruta.Ruta], ruta.Manejador}
		}
	}
	return nil
}

type manejadorMetodosInventariadosCTDesarrollo struct {
	ruta     string
	metodos  []metodoRutaCTDesarrollo
	delegado http.Handler
}

func (m manejadorMetodosInventariadosCTDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != m.ruta || m.delegado == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	permitidos := make([]string, 0, len(m.metodos))
	for _, metodo := range m.metodos {
		permitidos = append(permitidos, metodo.metodo)
		if r.Method == metodo.metodo {
			m.delegado.ServeHTTP(w, r)
			return
		}
	}
	w.Header().Set("Allow", strings.Join(permitidos, ", "))
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func esPrefijoRutaCTDesarrollo(ruta string) bool {
	return ruta == "/api/vec/contratacion-temporal" || strings.HasPrefix(ruta, "/api/vec/contratacion-temporal/") || strings.HasPrefix(ruta, "/api/interno/contratacion-temporal/")
}

func validarMetodoRutaCTDesarrollo(ruta string, metodo metodoRutaCTDesarrollo, fronteras catalogoFronterasComunDesarrollo) error {
	esperados, conocida := inventarioRutasCTDesarrollo()[ruta]
	declarado := false
	for _, esperado := range esperados {
		if esperado == metodo {
			declarado = true
			break
		}
	}
	if !conocida || !declarado || metodo.metodo == "" {
		return fmt.Errorf("%w: metodo CT no inventariado %s %q", ErrCoberturaRutasCTDesarrollo, metodo.metodo, ruta)
	}
	frontera, resuelta := fronteras.resolver(metodo.metodo, ruta)
	if metodo.guardia == "" {
		if !resuelta || frontera.Metodo != metodo.metodo || frontera.Ruta != ruta || frontera.Superficie != superficieInternaSeguridadComunDesarrollo || frontera.ClaveCapacidad == "" {
			return fmt.Errorf("%w: frontera PDP ausente %s %q", ErrCoberturaRutasCTDesarrollo, metodo.metodo, ruta)
		}
		return nil
	}
	if resuelta {
		return fmt.Errorf("%w: frontera PDP no inventariada %s %q", ErrCoberturaRutasCTDesarrollo, metodo.metodo, ruta)
	}
	return nil
}

// esRutaContratacionTemporalDesarrollo enumera las rutas del perfil de
// desarrollo que el revalidador protege con la identidad mTLS.
func esRutaContratacionTemporalDesarrollo(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if _, noCompuesta := rutasCapacidadNoCompuestaContratacionTemporal[r.URL.Path]; noCompuesta {
		return true
	}
	return rutaContinuidadNominal(r.URL.Path) || r.URL.Path == httpinterno.RutaPlanB2 || r.URL.Path == httpinterno.RutaConfirmacionB2 || r.URL.Path == httpinterno.RutaVinculoCategoriaRPTB2 || r.URL.Path == httpinterno.RutaCategoriasRPTB2 || r.URL.Path == rutaCatalogosRegistroEmpleadoB2 || r.URL.Path == httpinterno.RutaConsultaSeguimientoV2 || r.URL.Path == httpinterno.RutaFichaGINPIXV2 || r.URL.Path == httpinterno.RutaIncorporacionEjercicioV2 || r.URL.Path == httpinterno.RutaResolucionFormalizacion || r.URL.Path == rutaEntregaPeticionCentro || rutaPeticionCentroDesarrollo(r.URL.Path) || rutaAnalisisContratacionTemporalDesarrollo(r.URL.Path) ||
		rutaPlantillasCatalogoCTDesarrollo(r.URL.Path) ||
		r.URL.Path == ajusteshttp.Ruta ||
		rutaPlantillasDocumentalCTDesarrollo(r.URL.Path) ||
		r.URL.Path == httpinterno.RutaResolucionComunicacionLlamamiento ||
		r.URL.Path == httpinterno.RutaContinuacionLlamamiento ||
		r.URL.Path == httpinterno.RutaRegistroRespuestaRecibida ||
		r.URL.Path == httpinterno.RutaConsultaReciboRespuesta ||
		r.URL.Path == httpinterno.RutaConsultaComunicacionesExpediente ||
		r.URL.Path == httpinterno.RutaConsultaCircuitoRRHH ||
		r.URL.Path == httpinterno.RutaVinculosEmisionBolsa ||
		rutaFirmasR5V2CTDesarrollo(r.URL.Path) ||
		r.URL.Path == httpinterno.RutaEventoPlazoLlamamiento ||
		r.URL.Path == httpinterno.RutaRegistroComunicacionLlamamiento ||
		r.URL.Path == httpinterno.RutaResultadosFiscalizacion ||
		r.URL.Path == httpinterno.RutaSubsanacionReparos ||
		rutaSeguimientoCeseDesarrollo(r.URL.Path) || rutaCancelacionCTDesarrollo(r.URL.Path) ||
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
		rutaFirmaDocumentoCTDesarrollo(r.URL.Path) ||
		rutaCalendariosDesarrollo(r.URL.Path) || rutaDocumentacionFormalizacionDesarrollo(r.URL.Path) ||
		rutaReglasVigentesDesarrollo(r.URL.Path)
}
