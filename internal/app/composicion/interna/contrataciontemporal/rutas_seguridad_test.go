package contrataciontemporal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	contratacionapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
)

type autoridadDespachoContratacionPrueba struct{}

func (autoridadDespachoContratacionPrueba) AutorizarRutaExacta(
	context.Context,
	string,
) error {
	return nil
}

type autoridadDespachoContratacionDenegadaPrueba struct{}

func (autoridadDespachoContratacionDenegadaPrueba) AutorizarRutaExacta(
	context.Context,
	string,
) error {
	return httpapi.ErrAccesoRutaExactaDenegado
}

type autoridadDespachoContratacionEspiaPrueba struct {
	llamadas atomic.Int64
	ruta     atomic.Value
}

func (a *autoridadDespachoContratacionEspiaPrueba) AutorizarRutaExacta(
	_ context.Context,
	ruta string,
) error {
	a.ruta.Store(ruta)
	a.llamadas.Add(1)
	return nil
}

func (a *autoridadDespachoContratacionEspiaPrueba) estado() (int64, string) {
	ruta, _ := a.ruta.Load().(string)
	return a.llamadas.Load(), ruta
}

type autoridadAltaErrorComposicionPrueba struct {
	err error
}

func (a autoridadAltaErrorComposicionPrueba) ResolverContextoCanalAlta(
	context.Context,
) (contratacionapp.SolicitudRegistrarExpediente, error) {
	return contratacionapp.SolicitudRegistrarExpediente{}, a.err
}

type autoridadCoberturaErrorComposicionPrueba struct {
	err error
}

func (a autoridadCoberturaErrorComposicionPrueba) ResolverContextoCanalCobertura(
	context.Context,
) (httpinterno.ContextoCanalCobertura, error) {
	return httpinterno.ContextoCanalCobertura{}, a.err
}

type autoridadAnalisisErrorComposicionPrueba struct {
	err error
}

func (a autoridadAnalisisErrorComposicionPrueba) ResolverContextoCanalAnalisisRRHH(
	context.Context,
) (httpinterno.ContextoCanalAnalisisRRHH, error) {
	return httpinterno.ContextoCanalAnalisisRRHH{}, a.err
}

type negocioContratacionNoInvocablePrueba struct {
	altas           atomic.Int64
	propuestas      atomic.Int64
	decisiones      atomic.Int64
	rectificaciones atomic.Int64
	consultas       atomic.Int64
	llamamientos    atomic.Int64
}

func (n *negocioContratacionNoInvocablePrueba) Registrar(
	context.Context,
	contratacionapp.SolicitudRegistrarExpediente,
) (ports.ReciboAlta, error) {
	n.altas.Add(1)
	return ports.ReciboAlta{}, nil
}

func (n *negocioContratacionNoInvocablePrueba) ProponerParaAdaptador(
	context.Context,
	contratacionapp.SolicitudProponerCobertura,
) (contratacionapp.ResultadoPropuestaCoberturaParaAdaptador, error) {
	n.propuestas.Add(1)
	return contratacionapp.ResultadoPropuestaCoberturaParaAdaptador{}, nil
}

func (n *negocioContratacionNoInvocablePrueba) DecidirParaAdaptador(
	context.Context,
	contratacionapp.SolicitudDecidirCobertura,
) (contratacionapp.ResultadoDecisionCoberturaParaAdaptador, error) {
	n.decisiones.Add(1)
	return contratacionapp.ResultadoDecisionCoberturaParaAdaptador{}, nil
}

func (n *negocioContratacionNoInvocablePrueba) RectificarParaAdaptador(
	context.Context,
	contratacionapp.SolicitudRectificarCobertura,
) (contratacionapp.ResultadoDecisionCoberturaParaAdaptador, error) {
	n.rectificaciones.Add(1)
	return contratacionapp.ResultadoDecisionCoberturaParaAdaptador{}, nil
}

func (n *negocioContratacionNoInvocablePrueba) ConsultarParaAdaptador(
	context.Context,
	contratacionapp.SolicitudConsultaResultadoCobertura,
) (contratacionapp.DatosConsultaResultadoCoberturaParaAdaptador, error) {
	n.consultas.Add(1)
	return contratacionapp.DatosConsultaResultadoCoberturaParaAdaptador{}, nil
}

func (n *negocioContratacionNoInvocablePrueba) SeleccionarYLlamarParaAdaptador(
	context.Context,
	contratacionapp.SolicitudSeleccionLlamamiento,
) (contratacionapp.DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	n.llamamientos.Add(1)
	return contratacionapp.DatosReciboSeleccionLlamamientoParaAdaptador{}, nil
}

func (n *negocioContratacionNoInvocablePrueba) total() int64 {
	return n.altas.Load() + n.propuestas.Load() +
		n.decisiones.Load() + n.rectificaciones.Load() + n.consultas.Load() +
		n.llamamientos.Load()
}

type negocioAnalisisNoInvocablePrueba struct {
	registros       atomic.Int64
	rectificaciones atomic.Int64
}

func (n *negocioAnalisisNoInvocablePrueba) Registrar(
	context.Context,
	contratacionapp.SolicitudRegistrarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	n.registros.Add(1)
	return ports.ReciboOperacionAnalisis{}, nil
}

func (n *negocioAnalisisNoInvocablePrueba) Rectificar(
	context.Context,
	contratacionapp.SolicitudRectificarAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	n.rectificaciones.Add(1)
	return ports.ReciboOperacionAnalisis{}, nil
}

func (n *negocioAnalisisNoInvocablePrueba) total() int64 {
	return n.registros.Load() + n.rectificaciones.Load()
}

func TestRutasContratacionTemporalDenieganSinInvocarNegocio(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{
			"canal ausente",
			httpinterno.ErrContextoCanalAusente,
			http.StatusUnauthorized,
			"autenticacion_requerida",
		},
		{
			"canal caducado",
			httpinterno.ErrContextoCanalCaducado,
			http.StatusUnauthorized,
			"autenticacion_requerida",
		},
		{
			"organizacion denegada",
			httpinterno.ErrContextoCanalOrganizacionDenegada,
			http.StatusForbidden,
			"acceso_denegado",
		},
		{
			"autoridad no disponible",
			httpinterno.ErrContextoCanalNoDisponible,
			http.StatusServiceUnavailable,
			"servicio_no_disponible",
		},
	}
	rutas := []string{
		httpinterno.RutaAltaSolicitudes,
		httpinterno.RutaPropuestaCobertura,
		httpinterno.RutaDecisionCobertura,
		httpinterno.RutaRectificacionCobertura,
		httpinterno.RutaRegistroAnalisisRRHH,
		httpinterno.RutaRectificacionAnalisisRRHH,
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			negocio := &negocioContratacionNoInvocablePrueba{}
			analisis := &negocioAnalisisNoInvocablePrueba{}
			handler := nuevoHandlerContratacionErrorPrueba(
				t,
				caso.err,
				negocio,
				analisis,
				autoridadDespachoContratacionPrueba{},
			)
			for _, ruta := range rutas {
				respuesta := httptest.NewRecorder()
				handler.ServeHTTP(
					respuesta,
					nuevaPeticionContratacionErrorPrueba(ruta),
				)
				if respuesta.Code != caso.estado ||
					!strings.Contains(
						respuesta.Body.String(),
						`"codigo":"`+caso.codigo+`"`,
					) {
					t.Fatalf(
						"%s: estado=%d cuerpo=%s",
						ruta,
						respuesta.Code,
						respuesta.Body.String(),
					)
				}
			}
			if llamadas := negocio.total() + analisis.total(); llamadas != 0 {
				t.Fatalf("el negocio recibio %d llamadas", llamadas)
			}
		})
	}
}

func TestRutaResultadoCoberturaExigeAutoridadExteriorAntesDelConsultor(
	t *testing.T,
) {
	t.Parallel()
	negocio := &negocioContratacionNoInvocablePrueba{}
	analisis := &negocioAnalisisNoInvocablePrueba{}
	handler := nuevoHandlerContratacionErrorPrueba(
		t,
		httpinterno.ErrContextoCanalNoDisponible,
		negocio,
		analisis,
		autoridadDespachoContratacionDenegadaPrueba{},
	)
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(
		respuesta,
		nuevaPeticionContratacionErrorPrueba(
			httpinterno.RutaResultadoCobertura,
		),
	)
	if respuesta.Code != http.StatusForbidden ||
		!strings.Contains(
			respuesta.Body.String(),
			`"codigo":"acceso_denegado"`,
		) ||
		negocio.total() != 0 {
		t.Fatalf(
			"autoridad exterior omitida: estado=%d cuerpo=%s negocio=%d",
			respuesta.Code,
			respuesta.Body.String(),
			negocio.total(),
		)
	}
}

func TestRutaSeleccionLlamamientoExigeAutoridadExteriorAntesDelNegocio(
	t *testing.T,
) {
	t.Parallel()
	negocio := &negocioContratacionNoInvocablePrueba{}
	handler := nuevoHandlerContratacionErrorPrueba(
		t,
		httpinterno.ErrContextoCanalNoDisponible,
		negocio,
		&negocioAnalisisNoInvocablePrueba{},
		autoridadDespachoContratacionDenegadaPrueba{},
	)
	peticion := httptest.NewRequest(
		http.MethodPost,
		httpinterno.RutaSeleccionLlamamiento,
		strings.NewReader(
			`{"clave_idempotencia":"4d36e96e-e325-4f9b-bebc-291d91d6f732"}`,
		),
	)
	peticion.Header.Set("Content-Type", "application/json")
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusForbidden ||
		!strings.Contains(
			respuesta.Body.String(),
			`"codigo":"acceso_denegado"`,
		) ||
		negocio.total() != 0 {
		t.Fatalf(
			"autoridad exterior omitida: estado=%d cuerpo=%s negocio=%d",
			respuesta.Code,
			respuesta.Body.String(),
			negocio.total(),
		)
	}
}

func TestRutaPropuestaFormalizacionDelegaRutaExactaBajoAutoridadExterior(
	t *testing.T,
) {
	t.Parallel()
	dependencias := dependenciasRutasPrueba()
	autoridadInterior := dependencias.AutoridadPropuestaFormalizacion.(*autoridadPropuestaFormalizacionComposicionPrueba)
	ejecutor := dependencias.EjecutorPropuestaFormalizacion.(*ejecutorPropuestaFormalizacionComposicionPrueba)
	autoridadExterior := &autoridadDespachoContratacionEspiaPrueba{}
	handler := nuevoHandlerContratacionConDependenciasPrueba(
		t,
		dependencias,
		autoridadExterior,
	)
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(
		respuesta,
		nuevaPeticionPropuestaFormalizacionComposicionPrueba(),
	)
	llamadas, ruta := autoridadExterior.estado()
	if respuesta.Code != http.StatusCreated || llamadas != 1 ||
		ruta != httpinterno.RutaPropuestaFormalizacion ||
		autoridadInterior.resoluciones != 1 || ejecutor.ejecuciones != 1 {
		t.Fatalf(
			"estado=%d exterior=%d/%q interior=%d ejecutor=%d cuerpo=%s",
			respuesta.Code,
			llamadas,
			ruta,
			autoridadInterior.resoluciones,
			ejecutor.ejecuciones,
			respuesta.Body.String(),
		)
	}
}

func TestRutaPropuestaFormalizacionDenegadaNoInvocaManejador(
	t *testing.T,
) {
	t.Parallel()
	dependencias := dependenciasRutasPrueba()
	autoridadInterior := dependencias.AutoridadPropuestaFormalizacion.(*autoridadPropuestaFormalizacionComposicionPrueba)
	ejecutor := dependencias.EjecutorPropuestaFormalizacion.(*ejecutorPropuestaFormalizacionComposicionPrueba)
	handler := nuevoHandlerContratacionConDependenciasPrueba(
		t,
		dependencias,
		autoridadDespachoContratacionDenegadaPrueba{},
	)
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(
		respuesta,
		nuevaPeticionPropuestaFormalizacionComposicionPrueba(),
	)
	if respuesta.Code != http.StatusForbidden ||
		!strings.Contains(respuesta.Body.String(), `"codigo":"acceso_denegado"`) ||
		autoridadInterior.resoluciones != 0 || ejecutor.ejecuciones != 0 {
		t.Fatalf(
			"estado=%d interior=%d ejecutor=%d cuerpo=%s",
			respuesta.Code,
			autoridadInterior.resoluciones,
			ejecutor.ejecuciones,
			respuesta.Body.String(),
		)
	}
}

func TestRutasCierreAdministrativoDespachanBajoAutoridadExterior(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre      string
		ruta        string
		cierres     int
		reaperturas int
	}{
		{
			"cerrar",
			httpinterno.RutaCerrarAdministrativamente,
			1,
			0,
		},
		{
			"reabrir excepcionalmente",
			httpinterno.RutaReabrirExcepcionalmente,
			0,
			1,
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			dependencias := dependenciasRutasPrueba()
			autoridadInterior := dependencias.AutoridadCierreAdministrativo.(*autoridadCierreAdministrativoComposicionPrueba)
			ejecutor := dependencias.EjecutorCierreAdministrativo.(*ejecutorCierreAdministrativoComposicionPrueba)
			autoridadExterior := &autoridadDespachoContratacionEspiaPrueba{}
			handler := nuevoHandlerContratacionConDependenciasPrueba(
				t,
				dependencias,
				autoridadExterior,
			)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(
				respuesta,
				nuevaPeticionCierreAdministrativoComposicionPrueba(caso.ruta),
			)
			llamadas, ruta := autoridadExterior.estado()
			if respuesta.Code != http.StatusCreated || llamadas != 1 ||
				ruta != caso.ruta || autoridadInterior.resoluciones != 1 ||
				ejecutor.cierres != caso.cierres ||
				ejecutor.reaperturas != caso.reaperturas {
				t.Fatalf(
					"estado=%d exterior=%d/%q interior=%d ejecutor=%d/%d cuerpo=%s",
					respuesta.Code,
					llamadas,
					ruta,
					autoridadInterior.resoluciones,
					ejecutor.cierres,
					ejecutor.reaperturas,
					respuesta.Body.String(),
				)
			}
		})
	}
}

func TestRutasCierreAdministrativoRechazanOrganizacionDesdeHTTP(
	t *testing.T,
) {
	t.Parallel()
	dependencias := dependenciasRutasPrueba()
	autoridadInterior := dependencias.AutoridadCierreAdministrativo.(*autoridadCierreAdministrativoComposicionPrueba)
	ejecutor := dependencias.EjecutorCierreAdministrativo.(*ejecutorCierreAdministrativoComposicionPrueba)
	handler := nuevoHandlerContratacionConDependenciasPrueba(
		t,
		dependencias,
		autoridadDespachoContratacionPrueba{},
	)
	cuerpo := `{"organizacion_ref":"` +
		referenciaCierreAdministrativoComposicionPrueba("2") + `",` +
		`"expediente_ref":"` +
		referenciaCierreAdministrativoComposicionPrueba("b") + `",` +
		`"seguimiento_ref":"` +
		referenciaCierreAdministrativoComposicionPrueba("c") + `",` +
		`"version_esperada":7,` +
		`"clave_idempotencia":"12345678-1234-4567-8abc-123456789abc",` +
		`"transicion_clave":"cierre_administrativo",` +
		`"motivo_clave":"fin_relacion_confirmado"}`
	peticion := httptest.NewRequest(
		http.MethodPost,
		httpinterno.RutaCerrarAdministrativamente,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusBadRequest ||
		!strings.Contains(
			respuesta.Body.String(),
			`"codigo":"peticion_no_valida"`,
		) || autoridadInterior.resoluciones != 0 ||
		ejecutor.cierres != 0 || ejecutor.reaperturas != 0 {
		t.Fatalf(
			"estado=%d interior=%d ejecutor=%d/%d cuerpo=%s",
			respuesta.Code,
			autoridadInterior.resoluciones,
			ejecutor.cierres,
			ejecutor.reaperturas,
			respuesta.Body.String(),
		)
	}
}

func nuevoHandlerContratacionErrorPrueba(
	t *testing.T,
	errAutoridad error,
	negocio *negocioContratacionNoInvocablePrueba,
	analisis *negocioAnalisisNoInvocablePrueba,
	autoridad httpapi.AutoridadRutasExactas,
) *httpapi.Handler {
	t.Helper()
	dependencias := dependenciasRutasPrueba()
	dependencias.AutoridadAlta = autoridadAltaErrorComposicionPrueba{
		err: errAutoridad,
	}
	dependencias.EjecutorAlta = negocio
	dependencias.AutoridadAnalisis = autoridadAnalisisErrorComposicionPrueba{
		err: errAutoridad,
	}
	dependencias.EjecutorAnalisis = analisis
	dependencias.AutoridadCobertura = autoridadCoberturaErrorComposicionPrueba{
		err: errAutoridad,
	}
	dependencias.Presentador = negocio
	dependencias.Decisor = negocio
	dependencias.ConsultorResultado = negocio
	dependencias.EjecutorSeleccion = negocio
	return nuevoHandlerContratacionConDependenciasPrueba(
		t,
		dependencias,
		autoridad,
	)
}

func nuevoHandlerContratacionConDependenciasPrueba(
	t *testing.T,
	dependencias DependenciasRutas,
	autoridad httpapi.AutoridadRutasExactas,
) *httpapi.Handler {
	t.Helper()
	rutas, err := NuevasRutas(dependencias)
	if err != nil {
		t.Fatalf("construir rutas: %v", err)
	}
	almacen := memory.NewStore()
	servicio, err := vecapp.NewService(almacen, almacen, almacen)
	if err != nil {
		t.Fatalf("construir servicio VEC: %v", err)
	}
	handler, err := httpapi.NewHandlerWithOptions(
		servicio,
		httpapi.HandlerOptions{
			RutasExactas:          rutas,
			AutoridadRutasExactas: autoridad,
		},
	)
	if err != nil {
		t.Fatalf("construir handler VEC: %v", err)
	}
	return handler
}

func nuevaPeticionContratacionErrorPrueba(ruta string) *http.Request {
	cuerpo := `{}`
	if ruta == httpinterno.RutaAltaSolicitudes {
		cuerpo = `{
			"clave_idempotencia":"4d36e96e-e325-4f9b-bebc-291d91d6f732",
			"solicitud":{
				"centro_ref":"centro:solicitante:001",
				"contacto_ref":"contacto:opaco:001",
				"categoria_ref":"categoria:tecnica:001",
				"grupo_subgrupo":"A1",
				"motivo_clave":"necesidad_temporal",
				"detalle":"Cobertura temporal de una necesidad catalogada.",
				"periodo":{
					"inicio":"2026-08-01T00:00:00Z",
					"fin":"2026-12-31T00:00:00Z"
				},
				"rc":{"existe":false},
				"documentos_adjuntos":["documento:opaco:001"],
				"observaciones":"Tramitación ordinaria."
			}
		}`
	} else if ruta == httpinterno.RutaResultadoCobertura {
		cuerpo = `{
			"expediente_ref":"expediente:ct:0001",
			"clave_idempotencia":"4d36e96e-e325-4f9b-bebc-291d91d6f732"
		}`
	} else if ruta == httpinterno.RutaRegistroAnalisisRRHH ||
		ruta == httpinterno.RutaRectificacionAnalisisRRHH {
		motivo := ""
		version := "1"
		if ruta == httpinterno.RutaRectificacionAnalisisRRHH {
			motivo = `"motivo_rectificacion_clave":"ajuste_jornada",`
			version = "2"
		}
		cuerpo = `{
			"expediente_ref":"expediente:analisis:http:001",
			"version_esperada":` + version + `,
			"clave_idempotencia":"11111111-2222-4333-8444-555555555555",
			"artefacto_ref":"artefacto:analisis:http:001",` + motivo + `
			"analisis":{
				"modalidad_clave":"interinidad",
				"categoria_ref":"categoria:tecnico:001",
				"grupo_subgrupo":"A2",
				"causa_clave":"sustitucion",
				"periodo":{
					"inicio":"2026-09-01T00:00:00Z",
					"fin":"2027-02-28T00:00:00Z"
				},
				"porcentaje_jornada":7500,
				"entrada_rc":{
					"referencia":"entrada:rc:http:001",
					"huella_sha256":"` + strings.Repeat("9", 64) + `"
				}
			}
		}`
	}
	peticion := httptest.NewRequest(
		http.MethodPost,
		ruta,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}
