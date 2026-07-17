package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type preparadorDoble struct {
	orden    aplicacionbolsa.OrdenConsultaPanelInterno
	err      error
	llamadas int
	peticion *http.Request
}

func (p *preparadorDoble) PrepararOrdenConsultaPanelInterno(
	peticion *http.Request,
) (aplicacionbolsa.OrdenConsultaPanelInterno, error) {
	p.llamadas++
	p.peticion = peticion
	return p.orden, p.err
}

type consultorDoble struct {
	resultado puertosbolsa.InstantaneaPanelInterno
	err       error
	llamadas  int
	contexto  context.Context
	orden     aplicacionbolsa.OrdenConsultaPanelInterno
}

func (c *consultorDoble) Consultar(
	ctx context.Context,
	orden aplicacionbolsa.OrdenConsultaPanelInterno,
) (puertosbolsa.InstantaneaPanelInterno, error) {
	c.llamadas++
	c.contexto = ctx
	c.orden = orden
	return c.resultado, c.err
}

func instantaneaHTTPPrueba() puertosbolsa.InstantaneaPanelInterno {
	actualizada := time.Date(2026, 7, 18, 7, 30, 0, 123000000, time.UTC)
	confirmada := actualizada.Add(time.Second)
	limite := actualizada.Add(48 * time.Hour)
	return puertosbolsa.InstantaneaPanelInterno{
		Esquema: puertosbolsa.EsquemaPanelInternoBolsaV1,
		Selector: puertosbolsa.SelectorPanelInterno{
			Clase: puertosbolsa.AmbitoPanelOrganizacion, OrganizacionRef: "org_0123456789abcdef",
		},
		Origen: puertosbolsa.OrigenPanelInterno{
			Revision: "rev_0123456789abcdef", ActualizadaEn: actualizada, Demostracion: false,
		},
		PruebaLectura: puertosbolsa.PruebaLecturaPanelInterno{
			LecturaRef: "lec_0123456789abcdef", AuditoriaRef: "aud_0123456789abcdef",
			AuditoriaSecuencia: 8, DecisionRef: "decision-opaca",
			HuellaDecisionSHA256: strings.Repeat("a", 64), CorrelacionRef: "correlacion-opaca",
			ConfirmadaEn: confirmada,
		},
		Indicadores: puertosbolsa.IndicadoresPanelInterno{
			ConvocatoriasBorrador: 2, ConvocatoriasRevision: 1,
			ConvocatoriasPendientesFirma: 3, ConvocatoriasPublicadas: 5,
			BolsasActivas: 7, LlamamientosPendientes: 11, IncidenciasAbiertas: 1,
		},
		Convocatorias: []puertosbolsa.ResumenConvocatoriaPanelInterno{
			{
				ConvocatoriaRef: "cnv_0123456789abcdef", CategoriaClave: "auxiliar.administrativo",
				EstadoClave: "publicada", NumeroSolicitudes: 20, NumeroPendientes: 4,
			},
			{
				ConvocatoriaRef: "cnv_fedcba9876543210", CategoriaClave: "trabajo.social",
				EstadoClave: "revision", PlazoCierraEn: limite, NumeroSolicitudes: 9, NumeroPendientes: 2,
			},
		},
		ActuacionesPendientes: []puertosbolsa.ActuacionPendientePanelInterno{
			{
				ActuacionRef: "act_0123456789abcdef", RecursoRef: "cnv_0123456789abcdef",
				TipoClave: "firma", EstadoClave: "pendiente", PrioridadClave: "alta", NumeroElementos: 2,
			},
			{
				ActuacionRef: "act_fedcba9876543210", RecursoRef: "bol_0123456789abcdef",
				TipoClave: "revision", EstadoClave: "pendiente", PrioridadClave: "normal",
				FechaLimite: limite, NumeroElementos: 4,
			},
		},
	}
}

func nuevoHandlerPrueba(
	t *testing.T,
	preparador *preparadorDoble,
	consultor *consultorDoble,
) http.Handler {
	t.Helper()
	handler, err := NuevoHandler(preparador, consultor)
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestHandlerPanelInternoGETEnvelopeCerradoYOrdenPreparada(t *testing.T) {
	orden := aplicacionbolsa.OrdenConsultaPanelInterno{
		Selector: puertosbolsa.SelectorPanelInterno{
			Clase: puertosbolsa.AmbitoPanelUnidad, OrganizacionRef: "org_0123456789abcdef",
			UnidadGestionRef: "uni_0123456789abcdef",
		},
	}
	preparador := &preparadorDoble{orden: orden}
	consultor := &consultorDoble{resultado: instantaneaHTTPPrueba()}
	handler := nuevoHandlerPrueba(t, preparador, consultor)
	peticion := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Authorization", "Negotiate valor-no-interpretado-por-el-handler")
	respuesta := httptest.NewRecorder()

	handler.ServeHTTP(respuesta, peticion)

	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado = %d: %s", respuesta.Code, respuesta.Body.String())
	}
	if preparador.llamadas != 1 || consultor.llamadas != 1 || preparador.peticion != peticion ||
		consultor.contexto != peticion.Context() || !reflect.DeepEqual(consultor.orden, orden) {
		t.Fatalf("flujo incorrecto: preparador=%d consultor=%d", preparador.llamadas, consultor.llamadas)
	}
	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(respuesta.Body.Bytes(), &raiz); err != nil {
		t.Fatal(err)
	}
	if len(raiz) != 1 || raiz["data"] == nil {
		t.Fatalf("envelope no canónico: %s", respuesta.Body.String())
	}
	for _, prohibido := range []string{
		"valor-no-interpretado",
		`"candidatos"`, `"dni"`, `"correo"`, `"sesion"`, `"demostracion":true`, "0001-01-01",
	} {
		if strings.Contains(strings.ToLower(respuesta.Body.String()), strings.ToLower(prohibido)) {
			t.Fatalf("la respuesta contiene %q: %s", prohibido, respuesta.Body.String())
		}
	}
	if !strings.Contains(respuesta.Body.String(), `"convocatorias":[`) ||
		!strings.Contains(respuesta.Body.String(), `"actuaciones_pendientes":[`) {
		t.Fatalf("listas canónicas ausentes: %s", respuesta.Body.String())
	}
	if strings.Contains(respuesta.Body.String(), `"convocatoria_ref":"cnv_0123456789abcdef","categoria_clave":"auxiliar.administrativo","estado_clave":"publicada","plazo_cierra_en"`) {
		t.Fatalf("la fecha cero no se omitió: %s", respuesta.Body.String())
	}
	if !strings.Contains(respuesta.Body.String(), `"plazo_cierra_en":"2026-07-20T07:30:00.123Z"`) ||
		!strings.Contains(respuesta.Body.String(), `"fecha_limite":"2026-07-20T07:30:00.123Z"`) {
		t.Fatalf("las fechas presentes no se conservaron: %s", respuesta.Body.String())
	}
	comprobarCabecerasEstrictas(t, respuesta)
}

func TestHandlerPanelInternoHEADConservaSemanticaSinCuerpo(t *testing.T) {
	preparador := &preparadorDoble{}
	consultor := &consultorDoble{resultado: instantaneaHTTPPrueba()}
	handler := nuevoHandlerPrueba(t, preparador, consultor)
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest(http.MethodGet, RutaPanel, nil))
	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, RutaPanel, nil))

	if get.Code != http.StatusOK || head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("GET/HEAD = %d/%d, cuerpo HEAD=%q", get.Code, head.Code, head.Body.String())
	}
	if head.Header().Get("Content-Length") != strconv.Itoa(get.Body.Len()) {
		t.Fatalf("Content-Length HEAD=%q; GET=%d", head.Header().Get("Content-Length"), get.Body.Len())
	}
	if preparador.llamadas != 2 || consultor.llamadas != 2 {
		t.Fatalf("HEAD no siguió el caso de uso: %d/%d", preparador.llamadas, consultor.llamadas)
	}
	comprobarCabecerasEstrictas(t, head)
}

func TestHandlerPanelInternoPermiteAuthorizationSinInterpretarla(t *testing.T) {
	consultar := func(autorizacion string) (*httptest.ResponseRecorder, *preparadorDoble, *consultorDoble) {
		preparador := &preparadorDoble{}
		consultor := &consultorDoble{resultado: instantaneaHTTPPrueba()}
		peticion := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
		peticion.Header["aUtHoRiZaTiOn"] = []string{autorizacion}
		respuesta := httptest.NewRecorder()
		nuevoHandlerPrueba(t, preparador, consultor).ServeHTTP(respuesta, peticion)
		return respuesta, preparador, consultor
	}
	primera, preparadorPrimero, consultorPrimero := consultar("Negotiate token-uno-no-publicable")
	segunda, preparadorSegundo, consultorSegundo := consultar("VEC-Native token-dos-no-publicable")
	if primera.Code != http.StatusOK || segunda.Code != http.StatusOK ||
		primera.Body.String() != segunda.Body.String() {
		t.Fatalf("Authorization alteró el resultado: %d/%d", primera.Code, segunda.Code)
	}
	if preparadorPrimero.llamadas != 1 || preparadorSegundo.llamadas != 1 ||
		consultorPrimero.llamadas != 1 || consultorSegundo.llamadas != 1 {
		t.Fatalf("flujo incompleto: %d/%d/%d/%d", preparadorPrimero.llamadas, preparadorSegundo.llamadas, consultorPrimero.llamadas, consultorSegundo.llamadas)
	}
	for _, token := range []string{"token-uno-no-publicable", "token-dos-no-publicable"} {
		if strings.Contains(primera.Body.String(), token) || strings.Contains(segunda.Body.String(), token) {
			t.Fatalf("Authorization se filtró en la respuesta")
		}
	}
}

func TestHandlerPanelInternoRechazaTodaEntradaDelCliente(t *testing.T) {
	casos := []struct {
		nombre   string
		peticion func() *http.Request
		esperado int
	}{
		{"query selector", func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaPanel+"?selector=global", nil) }, http.StatusBadRequest},
		{"query rol", func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaPanel+"?rol=administrador", nil) }, http.StatusBadRequest},
		{"query motivo", func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaPanel+"?motivo=urgente", nil) }, http.StatusBadRequest},
		{"query vacía explícita", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.URL.ForceQuery = true
			return p
		}, http.StatusBadRequest},
		{"cuerpo", func() *http.Request {
			return httptest.NewRequest(http.MethodGet, RutaPanel, strings.NewReader(`{"rol":"administrador"}`))
		}, http.StatusBadRequest},
		{"cuerpo desconocido", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Body = io.NopCloser(bytes.NewBuffer(nil))
			return p
		}, http.StatusBadRequest},
		{"transfer encoding", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.TransferEncoding = []string{"chunked"}
			return p
		}, http.StatusBadRequest},
		{"cookie", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.AddCookie(&http.Cookie{Name: "sesion", Value: "secreto"})
			return p
		}, http.StatusBadRequest},
		{"cookie vacía", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["Cookie"] = []string{""}
			return p
		}, http.StatusBadRequest},
		{"cookie no canónica", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["cOoKiE"] = []string{"sesion=secreto"}
			return p
		}, http.StatusBadRequest},
		{"proxy authorization", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header.Set("Proxy-Authorization", "Basic secreto")
			return p
		}, http.StatusBadRequest},
		{"proxy authorization no canónica", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["pRoXy-AuThOrIzAtIoN"] = []string{"Basic secreto"}
			return p
		}, http.StatusBadRequest},
		{"X-VEC identidad", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["x-vEc-sUbJeCt"] = []string{"secreto"}
			return p
		}, http.StatusBadRequest},
		{"X-VEC rol vacío", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["X-VEC-Roles"] = []string{""}
			return p
		}, http.StatusBadRequest},
		{"X-Auth identidad", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["x-AuTh-ReQuEsT-UsEr"] = []string{"secreto"}
			return p
		}, http.StatusBadRequest},
		{"X-Remote-User", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["x-rEmOtE-uSeR"] = []string{"secreto"}
			return p
		}, http.StatusBadRequest},
		{"Remote-User vacío", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["rEmOtE-uSeR"] = []string{""}
			return p
		}, http.StatusBadRequest},
		{"X-Forwarded identidad", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.Header["x-FoRwArDeD-fOr"] = []string{"secreto"}
			return p
		}, http.StatusBadRequest},
		{"barra final", func() *http.Request { return httptest.NewRequest(http.MethodGet, RutaPanel+"/", nil) }, http.StatusNotFound},
		{"ruta ajena", func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/panel-ajeno", nil) }, http.StatusNotFound},
		{"ruta codificada", func() *http.Request {
			p := httptest.NewRequest(http.MethodGet, RutaPanel, nil)
			p.URL.RawPath = "/api/vec/bolsa/%70anel"
			return p
		}, http.StatusNotFound},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			preparador := &preparadorDoble{}
			consultor := &consultorDoble{resultado: instantaneaHTTPPrueba()}
			handler := nuevoHandlerPrueba(t, preparador, consultor)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, caso.peticion())
			if respuesta.Code != caso.esperado || preparador.llamadas != 0 || consultor.llamadas != 0 {
				t.Fatalf("estado=%d, llamadas=%d/%d: %s", respuesta.Code, preparador.llamadas, consultor.llamadas, respuesta.Body.String())
			}
			if strings.Contains(respuesta.Body.String(), "secreto") || respuesta.Header().Get("Set-Cookie") != "" {
				t.Fatalf("se filtró entrada o cookie: %s", respuesta.Body.String())
			}
			comprobarCabecerasEstrictas(t, respuesta)
		})
	}
}

func TestHandlerPanelInternoSoloGETYHEAD(t *testing.T) {
	for _, metodo := range []string{
		http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
		http.MethodOptions, http.MethodConnect, http.MethodTrace,
	} {
		t.Run(metodo, func(t *testing.T) {
			preparador := &preparadorDoble{}
			consultor := &consultorDoble{}
			respuesta := httptest.NewRecorder()
			nuevoHandlerPrueba(t, preparador, consultor).ServeHTTP(
				respuesta, httptest.NewRequest(metodo, RutaPanel, nil),
			)
			if respuesta.Code != http.StatusMethodNotAllowed || respuesta.Header().Get("Allow") != "GET, HEAD" ||
				preparador.llamadas != 0 || consultor.llamadas != 0 {
				t.Fatalf("%s = %d Allow=%q llamadas=%d/%d", metodo, respuesta.Code, respuesta.Header().Get("Allow"), preparador.llamadas, consultor.llamadas)
			}
		})
	}
}

func TestHandlerPanelInternoClasificaErroresSinFiltrarCausas(t *testing.T) {
	secreto := errors.New("clave_supersecreta_host_interno")
	casos := []struct {
		nombre        string
		errorPrep     error
		errorConsulta error
		estado        int
		codigo        string
	}{
		{"autenticación ausente", errors.Join(ErrAutenticacionInternaAusente, secreto), nil, http.StatusUnauthorized, "autenticacion_requerida"},
		{"denegación en preparador", errors.Join(dominiovec.ErrAutorizacionDenegada, secreto), nil, http.StatusForbidden, "acceso_denegado"},
		{"denegación en servicio", nil, errors.Join(dominiovec.ErrPermissionDenied, secreto), http.StatusForbidden, "acceso_denegado"},
		{"dependencia frontera", errors.Join(ErrDependenciaPanelInternoNoDisponible, secreto), nil, http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"dependencia prevalece sobre denegación cerrada", nil, errors.Join(dominiovec.ErrAutorizacionDenegada, puertosvec.ErrFuenteAutorizacionNoDisponible, secreto), http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"fuente de contexto", errors.Join(puertosvec.ErrFuenteContextoActorNoDisponible, secreto), nil, http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"servicio inválido", nil, errors.Join(aplicacionbolsa.ErrServicioPanelInternoInvalido, secreto), http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"fallo inesperado preparador", secreto, nil, http.StatusInternalServerError, "error_interno"},
		{"fallo inesperado servicio", nil, secreto, http.StatusInternalServerError, "error_interno"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			preparador := &preparadorDoble{err: caso.errorPrep}
			consultor := &consultorDoble{resultado: instantaneaHTTPPrueba(), err: caso.errorConsulta}
			respuesta := httptest.NewRecorder()
			nuevoHandlerPrueba(t, preparador, consultor).ServeHTTP(
				respuesta, httptest.NewRequest(http.MethodGet, RutaPanel, nil),
			)
			if respuesta.Code != caso.estado || !strings.Contains(respuesta.Body.String(), `"codigo":"`+caso.codigo+`"`) {
				t.Fatalf("estado=%d: %s", respuesta.Code, respuesta.Body.String())
			}
			if strings.Contains(respuesta.Body.String(), secreto.Error()) || strings.Contains(respuesta.Body.String(), "host_interno") {
				t.Fatalf("causa filtrada: %s", respuesta.Body.String())
			}
			esperadasConsulta := 1
			if caso.errorPrep != nil {
				esperadasConsulta = 0
			}
			if preparador.llamadas != 1 || consultor.llamadas != esperadasConsulta {
				t.Fatalf("llamadas=%d/%d", preparador.llamadas, consultor.llamadas)
			}
			comprobarCabecerasEstrictas(t, respuesta)
		})
	}
}

func TestHandlerPanelInternoHEADOcultaTambienErrores(t *testing.T) {
	preparador := &preparadorDoble{err: ErrAutenticacionInternaAusente}
	respuesta := httptest.NewRecorder()
	nuevoHandlerPrueba(t, preparador, &consultorDoble{}).ServeHTTP(
		respuesta, httptest.NewRequest(http.MethodHead, RutaPanel, nil),
	)
	if respuesta.Code != http.StatusUnauthorized || respuesta.Body.Len() != 0 ||
		respuesta.Header().Get("Content-Length") == "0" || respuesta.Header().Get("Content-Length") == "" {
		t.Fatalf("HEAD error=%d cuerpo=%q length=%q", respuesta.Code, respuesta.Body.String(), respuesta.Header().Get("Content-Length"))
	}
}

func TestNuevoHandlerFallaCerradoConDependenciasNulas(t *testing.T) {
	var preparadorNulo *preparadorDoble
	var consultorNulo *consultorDoble
	casos := []struct {
		preparador PreparadorOrdenConsultaPanelInterno
		consultor  ConsultorPanelInterno
	}{
		{nil, &consultorDoble{}}, {&preparadorDoble{}, nil},
		{preparadorNulo, &consultorDoble{}}, {&preparadorDoble{}, consultorNulo},
	}
	for indice, caso := range casos {
		if handler, err := NuevoHandler(caso.preparador, caso.consultor); handler != nil ||
			!errors.Is(err, ErrHandlerPanelInternoInvalido) {
			t.Fatalf("caso %d: handler=%v error=%v", indice, handler, err)
		}
	}
	var handlerNulo *Handler
	respuesta := httptest.NewRecorder()
	handlerNulo.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, RutaPanel, nil))
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("handler nil = %d", respuesta.Code)
	}
	handlerValido := &Handler{}
	respuesta = httptest.NewRecorder()
	handlerValido.ServeHTTP(respuesta, nil)
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("petición nil = %d", respuesta.Code)
	}
}

func TestHandlerPanelInternoCanonizaListasNulasComoVacias(t *testing.T) {
	instantanea := instantaneaHTTPPrueba()
	instantanea.Convocatorias = nil
	instantanea.ActuacionesPendientes = nil
	respuesta := httptest.NewRecorder()
	nuevoHandlerPrueba(t, &preparadorDoble{}, &consultorDoble{resultado: instantanea}).ServeHTTP(
		respuesta, httptest.NewRequest(http.MethodGet, RutaPanel, nil),
	)
	if respuesta.Code != http.StatusOK || !strings.Contains(respuesta.Body.String(), `"convocatorias":[]`) ||
		!strings.Contains(respuesta.Body.String(), `"actuaciones_pendientes":[]`) || strings.Contains(respuesta.Body.String(), ":null") {
		t.Fatalf("listas no canónicas: %s", respuesta.Body.String())
	}
}

func comprobarCabecerasEstrictas(t *testing.T, respuesta *httptest.ResponseRecorder) {
	t.Helper()
	esperadas := map[string]string{
		"Content-Type": "application/json", "Cache-Control": "no-store", "Pragma": "no-cache",
		"Expires": "0", "X-Content-Type-Options": "nosniff", "Content-Security-Policy": "default-src 'none'",
		"Referrer-Policy": "no-referrer", "Permissions-Policy": "geolocation=()",
		"Cross-Origin-Resource-Policy": "same-origin", "X-Frame-Options": "DENY",
	}
	for nombre, fragmento := range esperadas {
		if !strings.Contains(respuesta.Header().Get(nombre), fragmento) {
			t.Errorf("%s=%q, no contiene %q", nombre, respuesta.Header().Get(nombre), fragmento)
		}
	}
	for _, prohibida := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials"} {
		if respuesta.Header().Get(prohibida) != "" {
			t.Errorf("cabecera prohibida %s=%q", prohibida, respuesta.Header().Get(prohibida))
		}
	}
}
