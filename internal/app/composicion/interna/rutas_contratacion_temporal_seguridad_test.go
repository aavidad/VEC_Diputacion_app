package interna

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

type negocioContratacionNoInvocablePrueba struct {
	altas           atomic.Int64
	propuestas      atomic.Int64
	decisiones      atomic.Int64
	rectificaciones atomic.Int64
	consultas       atomic.Int64
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

func (n *negocioContratacionNoInvocablePrueba) total() int64 {
	return n.altas.Load() + n.propuestas.Load() +
		n.decisiones.Load() + n.rectificaciones.Load() + n.consultas.Load()
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
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			negocio := &negocioContratacionNoInvocablePrueba{}
			handler := nuevoHandlerContratacionErrorPrueba(
				t,
				caso.err,
				negocio,
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
			if llamadas := negocio.total(); llamadas != 0 {
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
	handler := nuevoHandlerContratacionErrorPrueba(
		t,
		httpinterno.ErrContextoCanalNoDisponible,
		negocio,
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

func nuevoHandlerContratacionErrorPrueba(
	t *testing.T,
	errAutoridad error,
	negocio *negocioContratacionNoInvocablePrueba,
	autoridad httpapi.AutoridadRutasExactas,
) *httpapi.Handler {
	t.Helper()
	rutas, err := nuevasRutasContratacionTemporal(
		dependenciasRutasContratacionTemporal{
			autoridadAlta: autoridadAltaErrorComposicionPrueba{
				err: errAutoridad,
			},
			ejecutorAlta: negocio,
			reloj:        relojComposicionPrueba{},
			autoridadCobertura: autoridadCoberturaErrorComposicionPrueba{
				err: errAutoridad,
			},
			presentador:        negocio,
			decisor:            negocio,
			consultorResultado: negocio,
		},
	)
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
