package httpinterno

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const claveCierreAdministrativoHTTPPrueba = "12345678-1234-4567-8abc-123456789abc"

type autoridadCierreAdministrativoHTTPPrueba struct {
	organizacionRef string
	err             error
	durante         func(context.Context)
	llamadas        int
	orden           *[]string
}

func (a *autoridadCierreAdministrativoHTTPPrueba) ResolverOrganizacionCierreAdministrativo(
	ctx context.Context,
) (string, error) {
	a.llamadas++
	if a.orden != nil {
		*a.orden = append(*a.orden, "autoridad")
	}
	if a.durante != nil {
		a.durante(ctx)
	}
	return a.organizacionRef, a.err
}

type ejecutorCierreAdministrativoHTTPPrueba struct {
	err               error
	estado            ports.EstadoResultadoCierreAdministrativo
	resultadoInvalido bool
	resultadoAjeno    bool
	resultadoConError bool
	durante           func(context.Context)
	llamadasCerrar    int
	llamadasReabrir   int
	ultimaCerrar      application.SolicitudCerrarAdministrativamente
	ultimaReabrir     application.SolicitudReabrirExcepcionalmente
	orden             *[]string
}

func (e *ejecutorCierreAdministrativoHTTPPrueba) Cerrar(
	ctx context.Context,
	solicitud application.SolicitudCerrarAdministrativamente,
) (ports.ResultadoCierreAdministrativo, error) {
	e.llamadasCerrar++
	e.ultimaCerrar = solicitud
	if e.orden != nil {
		*e.orden = append(*e.orden, "cerrar")
	}
	return e.responder(ctx, solicitudPuertoDesdeCierreHTTP(solicitud))
}

func (e *ejecutorCierreAdministrativoHTTPPrueba) ReabrirExcepcionalmente(
	ctx context.Context,
	solicitud application.SolicitudReabrirExcepcionalmente,
) (ports.ResultadoCierreAdministrativo, error) {
	e.llamadasReabrir++
	e.ultimaReabrir = solicitud
	if e.orden != nil {
		*e.orden = append(*e.orden, "reabrir")
	}
	return e.responder(ctx, solicitudPuertoDesdeReaperturaHTTP(solicitud))
}

func (e *ejecutorCierreAdministrativoHTTPPrueba) responder(
	ctx context.Context,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
) (ports.ResultadoCierreAdministrativo, error) {
	if e.durante != nil {
		e.durante(ctx)
	}
	if e.resultadoInvalido {
		return ports.ResultadoCierreAdministrativo{}, e.err
	}
	if e.err != nil && !e.resultadoConError {
		return ports.ResultadoCierreAdministrativo{}, e.err
	}
	if e.resultadoAjeno {
		solicitud.SeguimientoRef = referenciaCierreAdministrativoHTTPPrueba(
			"seguimiento_ajeno",
		)
	}
	estado := e.estado
	if estado == "" {
		estado = ports.EstadoResultadoCierreAdministrativoConfirmado
	}
	resultado, err := ports.NuevoResultadoCierreAdministrativo(
		ports.DatosResultadoCierreAdministrativo{
			Solicitud:         solicitud,
			VersionResultante: solicitud.VersionEsperada + 1,
			ActuacionRef: referenciaCierreAdministrativoHTTPPrueba(
				"actuacion_resultado",
			),
			ReciboRef: referenciaCierreAdministrativoHTTPPrueba(
				"recibo_resultado",
			),
			ActorRef: referenciaCierreAdministrativoHTTPPrueba(
				"actor_resultado",
			),
			CorrelacionRef: referenciaCierreAdministrativoHTTPPrueba(
				"correlacion_resultado",
			),
			Estado: estado,
		},
	)
	if err != nil {
		return ports.ResultadoCierreAdministrativo{}, err
	}
	return resultado, e.err
}

func TestManejadorCierreAdministrativoExponeDosRutasMinimizadas(
	t *testing.T,
) {
	casos := []struct {
		nombre          string
		ruta            string
		estadoResultado ports.EstadoResultadoCierreAdministrativo
		estadoHTTP      int
		paso            string
	}{
		{
			"cerrar",
			RutaCerrarAdministrativamente,
			ports.EstadoResultadoCierreAdministrativoConfirmado,
			http.StatusCreated,
			"cerrar",
		},
		{
			"reabrir excepcionalmente",
			RutaReabrirExcepcionalmente,
			ports.EstadoResultadoCierreAdministrativoReplayConfirmado,
			http.StatusOK,
			"reabrir",
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			orden := []string{}
			autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
			autoridad.orden = &orden
			ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{
				estado: caso.estadoResultado,
				orden:  &orden,
			}
			manejador := nuevoManejadorCierreAdministrativoHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				peticionCierreAdministrativoHTTPPrueba(t, caso.ruta),
			)

			if respuesta.Code != caso.estadoHTTP ||
				autoridad.llamadas != 1 ||
				ejecutor.llamadasCerrar+ejecutor.llamadasReabrir != 1 {
				t.Fatalf(
					"estado=%d llamadas=%d/%d/%d cuerpo=%s",
					respuesta.Code,
					autoridad.llamadas,
					ejecutor.llamadasCerrar,
					ejecutor.llamadasReabrir,
					respuesta.Body,
				)
			}
			if len(orden) != 2 || orden[0] != "autoridad" ||
				orden[1] != caso.paso {
				t.Fatalf("orden de fronteras = %#v", orden)
			}
			comprobarSalidaCierreAdministrativoHTTPPrueba(t, respuesta)
			comprobarSolicitudCierreAdministrativoHTTPPrueba(
				t,
				autoridad.organizacionRef,
				ejecutor,
				caso.ruta,
			)
		})
	}
}

func TestManejadorCierreAdministrativoClasificaErroresSinDetalles(
	t *testing.T,
) {
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"cancelacion", context.Canceled, http.StatusRequestTimeout, "peticion_cancelada"},
		{"plazo", context.DeadlineExceeded, http.StatusGatewayTimeout, "plazo_agotado"},
		{"entrada", application.ErrSolicitudCierreAdministrativoInvalida, http.StatusUnprocessableEntity, "contenido_no_valido"},
		{"denegacion", application.ErrCierreAdministrativoNoPermitido, http.StatusForbidden, "acceso_denegado"},
		{"denegacion puerto", ports.ErrCierreAdministrativoDenegado, http.StatusForbidden, "acceso_denegado"},
		{"version", application.ErrVersionCierreAdministrativoEnConflicto, http.StatusConflict, "version_en_conflicto"},
		{"colision", ports.ErrClaveIdempotenciaCierreAdministrativoUsada, http.StatusConflict, "clave_idempotencia_reutilizada"},
		{"resultado", application.ErrResultadoCierreAdministrativoInvalido, http.StatusBadGateway, "resultado_no_confiable"},
		{"no disponible", errors.New("detalle privado irrepetible"), http.StatusServiceUnavailable, "servicio_no_disponible"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			autoridad := autoridadCierreAdministrativoHTTPValidaPrueba()
			ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{err: caso.err}
			respuesta := httptest.NewRecorder()
			nuevoManejadorCierreAdministrativoHTTPPrueba(
				t,
				autoridad,
				ejecutor,
			).ServeHTTP(
				respuesta,
				peticionCierreAdministrativoHTTPPrueba(
					t,
					RutaCerrarAdministrativamente,
				),
			)
			if respuesta.Code != caso.estado ||
				!strings.Contains(respuesta.Body.String(), `"codigo":"`+caso.codigo+`"`) ||
				strings.Contains(respuesta.Body.String(), "detalle privado") {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestManejadorCierreAdministrativoNoPublicaResultadoAmbiguoOAjeno(
	t *testing.T,
) {
	casos := []struct {
		nombre   string
		ejecutor *ejecutorCierreAdministrativoHTTPPrueba
	}{
		{
			"resultado cero",
			&ejecutorCierreAdministrativoHTTPPrueba{resultadoInvalido: true},
		},
		{
			"resultado de otra solicitud",
			&ejecutorCierreAdministrativoHTTPPrueba{resultadoAjeno: true},
		},
		{
			"resultado junto con error",
			&ejecutorCierreAdministrativoHTTPPrueba{
				err:               application.ErrCierreAdministrativoNoPermitido,
				resultadoConError: true,
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			nuevoManejadorCierreAdministrativoHTTPPrueba(
				t,
				autoridadCierreAdministrativoHTTPValidaPrueba(),
				caso.ejecutor,
			).ServeHTTP(
				respuesta,
				peticionCierreAdministrativoHTTPPrueba(
					t,
					RutaCerrarAdministrativamente,
				),
			)
			if respuesta.Code != http.StatusBadGateway ||
				!strings.Contains(respuesta.Body.String(),
					`"codigo":"resultado_no_confiable"`) ||
				strings.Contains(respuesta.Body.String(), "recibo_resultado") {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestNuevoManejadorCierreAdministrativoRechazaNulosTipados(t *testing.T) {
	var autoridadNula *autoridadCierreAdministrativoHTTPPrueba
	var ejecutorNulo *ejecutorCierreAdministrativoHTTPPrueba
	ejecutor := &ejecutorCierreAdministrativoHTTPPrueba{}
	if _, err := NuevoManejadorCierreAdministrativo(
		autoridadNula,
		ejecutor,
	); !errors.Is(err, ErrManejadorCierreAdministrativoInvalido) {
		t.Fatalf("autoridad nula tipada: %v", err)
	}
	if _, err := NuevoManejadorCierreAdministrativo(
		autoridadCierreAdministrativoHTTPValidaPrueba(),
		ejecutorNulo,
	); !errors.Is(err, ErrManejadorCierreAdministrativoInvalido) {
		t.Fatalf("ejecutor nulo tipado: %v", err)
	}
}

func autoridadCierreAdministrativoHTTPValidaPrueba() *autoridadCierreAdministrativoHTTPPrueba {
	return &autoridadCierreAdministrativoHTTPPrueba{
		organizacionRef: referenciaCierreAdministrativoHTTPPrueba(
			"organizacion_servidor",
		),
	}
}

func nuevoManejadorCierreAdministrativoHTTPPrueba(
	t *testing.T,
	autoridad AutoridadServidorCierreAdministrativo,
	ejecutor EjecutorCierreAdministrativo,
) http.Handler {
	t.Helper()
	manejador, err := NuevoManejadorCierreAdministrativo(autoridad, ejecutor)
	if err != nil {
		t.Fatalf("crear manejador: %v", err)
	}
	return manejador
}

func entradaCierreAdministrativoHTTPPrueba() cierreAdministrativoEntradaJSON {
	return cierreAdministrativoEntradaJSON{
		ExpedienteRef: referenciaCierreAdministrativoHTTPPrueba(
			"expediente_http",
		),
		SeguimientoRef: referenciaCierreAdministrativoHTTPPrueba(
			"seguimiento_http",
		),
		VersionEsperada:   7,
		ClaveIdempotencia: claveCierreAdministrativoHTTPPrueba,
		TransicionClave:   "cierre_administrativo",
		MotivoClave:       "fin_relacion_confirmado",
	}
}

func cuerpoCierreAdministrativoHTTPPrueba(t *testing.T) string {
	t.Helper()
	contenido, err := json.Marshal(entradaCierreAdministrativoHTTPPrueba())
	if err != nil {
		t.Fatalf("codificar entrada: %v", err)
	}
	return string(contenido)
}

func peticionCierreAdministrativoHTTPPrueba(
	t *testing.T,
	ruta string,
) *http.Request {
	t.Helper()
	peticion := httptest.NewRequest(
		http.MethodPost,
		ruta,
		strings.NewReader(cuerpoCierreAdministrativoHTTPPrueba(t)),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func referenciaCierreAdministrativoHTTPPrueba(etiqueta string) string {
	huella := sha256.Sum256([]byte(etiqueta))
	return "ref:" + hex.EncodeToString(huella[:])
}

func solicitudPuertoDesdeCierreHTTP(
	s application.SolicitudCerrarAdministrativamente,
) ports.SolicitudTransaccionCierreAdministrativo {
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:         ports.OperacionCerrarAdministrativamente,
		OrganizacionRef:   s.OrganizacionRef,
		ExpedienteRef:     s.ExpedienteRef,
		SeguimientoRef:    s.SeguimientoRef,
		VersionEsperada:   s.VersionEsperada,
		ClaveIdempotencia: s.ClaveIdempotencia,
		TransicionClave:   s.TransicionClave,
		MotivoClave:       s.MotivoClave,
	}
}

func solicitudPuertoDesdeReaperturaHTTP(
	s application.SolicitudReabrirExcepcionalmente,
) ports.SolicitudTransaccionCierreAdministrativo {
	return ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:         ports.OperacionReabrirExcepcionalmente,
		OrganizacionRef:   s.OrganizacionRef,
		ExpedienteRef:     s.ExpedienteRef,
		SeguimientoRef:    s.SeguimientoRef,
		VersionEsperada:   s.VersionEsperada,
		ClaveIdempotencia: s.ClaveIdempotencia,
		TransicionClave:   s.TransicionClave,
		MotivoClave:       s.MotivoClave,
	}
}

func comprobarSalidaCierreAdministrativoHTTPPrueba(
	t *testing.T,
	respuesta *httptest.ResponseRecorder,
) {
	t.Helper()
	var salida map[string]map[string]any
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil {
		t.Fatalf("respuesta JSON: %v", err)
	}
	datos, existe := salida["data"]
	if len(salida) != 1 || !existe || len(datos) != 2 ||
		datos["recibo_ref"] != referenciaCierreAdministrativoHTTPPrueba(
			"recibo_resultado",
		) || datos["version_seguimiento"] != float64(8) {
		t.Fatalf("salida no minimizada: %#v", salida)
	}
	for _, privado := range []string{
		"expediente", "organizacion", "actor", "perfil",
		"unidad", "autorizacion", "correlacion",
	} {
		if strings.Contains(respuesta.Body.String(), privado) {
			t.Fatalf("salida filtra %q: %s", privado, respuesta.Body)
		}
	}
}

func comprobarSolicitudCierreAdministrativoHTTPPrueba(
	t *testing.T,
	organizacionRef string,
	ejecutor *ejecutorCierreAdministrativoHTTPPrueba,
	ruta string,
) {
	t.Helper()
	entrada := entradaCierreAdministrativoHTTPPrueba()
	if ruta == RutaCerrarAdministrativamente {
		s := ejecutor.ultimaCerrar
		if s.OrganizacionRef != organizacionRef ||
			s.ExpedienteRef != entrada.ExpedienteRef ||
			s.SeguimientoRef != entrada.SeguimientoRef ||
			s.VersionEsperada != entrada.VersionEsperada ||
			s.ClaveIdempotencia != entrada.ClaveIdempotencia ||
			s.TransicionClave != domain.ClaveCatalogo(entrada.TransicionClave) ||
			s.MotivoClave != domain.ClaveCatalogo(entrada.MotivoClave) {
			t.Fatalf("solicitud de cierre alterada: %#v", s)
		}
		return
	}
	s := ejecutor.ultimaReabrir
	if s.OrganizacionRef != organizacionRef ||
		s.ExpedienteRef != entrada.ExpedienteRef ||
		s.SeguimientoRef != entrada.SeguimientoRef ||
		s.VersionEsperada != entrada.VersionEsperada ||
		s.ClaveIdempotencia != entrada.ClaveIdempotencia ||
		s.TransicionClave != domain.ClaveCatalogo(entrada.TransicionClave) ||
		s.MotivoClave != domain.ClaveCatalogo(entrada.MotivoClave) {
		t.Fatalf("solicitud de reapertura alterada: %#v", s)
	}
}
