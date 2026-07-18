package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var instanteHTTPPropuestaPrueba = time.Date(2026, time.July, 15, 10, 30, 0, 123_456_000, time.UTC)

type preparadorPropuestaLlamamientoDoble struct {
	solicitud puertosbolsa.SolicitudProponerLlamamiento
	err       error
	llamadas  int
	entrada   EntradaPreparacionPropuestaLlamamientoInterno
}

func (p *preparadorPropuestaLlamamientoDoble) PrepararSolicitudPropuestaLlamamientoInterno(
	_ context.Context,
	entrada EntradaPreparacionPropuestaLlamamientoInterno,
) (puertosbolsa.SolicitudProponerLlamamiento, error) {
	p.llamadas++
	p.entrada = entrada
	return p.solicitud, p.err
}

type proponenteLlamamientoDoble struct {
	propuesta dominiobolsa.PropuestaLlamamiento
	err       error
	llamadas  int
	solicitud puertosbolsa.SolicitudProponerLlamamiento
}

func (p *proponenteLlamamientoDoble) ProponerPrimerLlamamiento(
	_ context.Context,
	solicitud puertosbolsa.SolicitudProponerLlamamiento,
) (dominiobolsa.PropuestaLlamamiento, error) {
	p.llamadas++
	p.solicitud = solicitud
	return p.propuesta, p.err
}

func TestNuevoHandlerPropuestasLlamamientoExigeDependenciasReales(t *testing.T) {
	preparador := &preparadorPropuestaLlamamientoDoble{}
	proponente := &proponenteLlamamientoDoble{}
	var preparadorNulo *preparadorPropuestaLlamamientoDoble
	var proponenteNulo *proponenteLlamamientoDoble

	for nombre, construir := range map[string]func() (http.Handler, error){
		"preparador nil": func() (http.Handler, error) {
			return NuevoHandlerPropuestasLlamamiento(nil, proponente)
		},
		"preparador typed nil": func() (http.Handler, error) {
			return NuevoHandlerPropuestasLlamamiento(preparadorNulo, proponente)
		},
		"proponente nil": func() (http.Handler, error) {
			return NuevoHandlerPropuestasLlamamiento(preparador, nil)
		},
		"proponente typed nil": func() (http.Handler, error) {
			return NuevoHandlerPropuestasLlamamiento(preparador, proponenteNulo)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			handler, err := construir()
			if handler != nil || !errors.Is(err, ErrHandlerPropuestasLlamamientoInvalido) {
				t.Fatalf("dependencia nula admitida: handler=%T err=%v", handler, err)
			}
		})
	}
}

func TestHandlerPropuestasLlamamientoProyectaConfirmacionRealCompactaYMinimizada(t *testing.T) {
	propuesta := propuestaHTTPPrueba(t)
	solicitud := solicitudHTTPPrueba(t, propuesta.NecesidadRef)
	preparador := &preparadorPropuestaLlamamientoDoble{solicitud: solicitud}
	proponente := &proponenteLlamamientoDoble{propuesta: propuesta}
	handler := handlerPropuestaHTTPPrueba(t, preparador, proponente)
	peticion := peticionPropuestaHTTPPrueba(cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef))
	peticion.Header.Set("Authorization", "Bearer secreto-que-el-handler-no-debe-interpretar")
	respuesta := httptest.NewRecorder()

	handler.ServeHTTP(respuesta, peticion)

	if respuesta.Code != http.StatusCreated {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if preparador.llamadas != 1 || proponente.llamadas != 1 ||
		preparador.entrada.NecesidadRef != propuesta.NecesidadRef ||
		proponente.solicitud.NecesidadRef != propuesta.NecesidadRef {
		t.Fatalf("flujo inesperado: preparador=%+v proponente=%+v", preparador, proponente)
	}
	if respuesta.Header().Get("Location") != "" {
		t.Fatalf("se anuncio una lectura inexistente: %q", respuesta.Header().Get("Location"))
	}
	etagEsperado := `"vec-propuesta-llamamiento-v1.sha256-` + propuesta.HuellaContenidoSHA256 + `"`
	if respuesta.Header().Get("ETag") != etagEsperado {
		t.Fatalf("etag=%q esperado=%q", respuesta.Header().Get("ETag"), etagEsperado)
	}
	if respuesta.Header().Get("Cache-Control") != "no-store" ||
		respuesta.Header().Get("Content-Security-Policy") == "" ||
		respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		respuesta.Header().Get("Content-Length") != strconv.Itoa(respuesta.Body.Len()) {
		t.Fatalf("cabeceras de seguridad incompletas: %v", respuesta.Header())
	}

	contenido := respuesta.Body.String()
	for _, prohibido := range []string{
		"puntuacion", "recibo", "demostracion", `"evaluaciones":`, "motivos",
		"entrada_evaluacion", "resultado_evaluacion", "sujeto_ref", "participacion_ref",
		"secreto-que-el-handler-no-debe-interpretar",
		propuesta.SujetoSeleccionadoRef, propuesta.ParticipacionSeleccionadaRef,
	} {
		if strings.Contains(contenido, prohibido) {
			t.Fatalf("la salida contiene %q: %s", prohibido, contenido)
		}
	}
	var envelope struct {
		Data struct {
			Esquema                         string                       `json:"esquema"`
			PropuestaRef                    string                       `json:"propuesta_ref"`
			HuellaPropuestaSHA256           string                       `json:"huella_propuesta_sha256"`
			Bolsa                           versionHuellaLlamamientoJSON `json:"bolsa"`
			Necesidad                       versionHuellaLlamamientoJSON `json:"necesidad"`
			Instantanea                     versionHuellaLlamamientoJSON `json:"instantanea"`
			Politica                        versionHuellaLlamamientoJSON `json:"politica"`
			OrdenSeleccionado               string                       `json:"orden_seleccionado"`
			TotalParticipacionesInstantanea string                       `json:"total_participaciones_instantanea"`
			TotalEvaluaciones               string                       `json:"total_evaluaciones"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("json: %v", err)
	}
	if envelope.Data.Esquema != esquemaConfirmacionPropuestaLlamamiento ||
		envelope.Data.PropuestaRef != propuesta.PropuestaRef ||
		envelope.Data.HuellaPropuestaSHA256 != propuesta.HuellaContenidoSHA256 ||
		envelope.Data.Bolsa.Referencia != propuesta.BolsaRef ||
		envelope.Data.Bolsa.Version != "3" ||
		envelope.Data.Bolsa.HuellaSHA256 != propuesta.HuellaBolsaSHA256 ||
		envelope.Data.Necesidad.Referencia != propuesta.NecesidadRef ||
		envelope.Data.Instantanea.Referencia != propuesta.InstantaneaRef ||
		envelope.Data.Politica.Referencia != propuesta.PoliticaRef ||
		envelope.Data.OrdenSeleccionado != "1" ||
		envelope.Data.TotalParticipacionesInstantanea != "1" ||
		envelope.Data.TotalEvaluaciones != "1" {
		t.Fatalf("confirmacion incompleta: %+v", envelope.Data)
	}
}

func TestConfirmacionPropuestaLlamamientoTieneTamanoMaximoDeterminista(t *testing.T) {
	referenciaMaxima := strings.Repeat("r", 512)
	huellaMaxima := strings.Repeat("f", 64)
	versionMaxima := "18446744073709551615"
	version := versionHuellaLlamamientoJSON{
		Referencia: referenciaMaxima, Version: versionMaxima, HuellaSHA256: huellaMaxima,
	}
	confirmacionMaxima := envelopePropuestaLlamamientoJSON{Data: propuestaLlamamientoJSON{
		Esquema:      esquemaConfirmacionPropuestaLlamamiento,
		PropuestaRef: referenciaMaxima, HuellaPropuestaSHA256: huellaMaxima,
		Bolsa: version, Necesidad: version, Instantanea: version, Politica: version,
		InstanteReferencia:              "9999-12-31T23:59:59.999999Z",
		InstantaneaGeneradaEn:           "9999-12-31T23:59:59.999999Z",
		TotalParticipacionesInstantanea: "250000", TotalEvaluaciones: "250000",
		OrdenSeleccionado: "250000", GeneradaEn: "9999-12-31T23:59:59.999999Z",
	}}
	contenido, err := json.Marshal(confirmacionMaxima)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(contenido) >= maximoCuerpoRespuestaLlamamiento {
		t.Fatalf("confirmacion maxima=%d limite=%d", len(contenido), maximoCuerpoRespuestaLlamamiento)
	}
	for _, prohibido := range []string{`"evaluaciones":`, "participacion_ref", "sujeto_ref"} {
		if strings.Contains(string(contenido), prohibido) {
			t.Fatalf("campo no acotado o identificativo %q: %s", prohibido, contenido)
		}
	}
}

func TestHandlerPropuestasLlamamientoSoloAdmiteRutaYMetodoCanonicos(t *testing.T) {
	preparador, proponente, handler, propuesta := dependenciasHandlerPropuestaHTTPPrueba(t)
	casosRuta := []struct {
		nombre string
		mutar  func(*http.Request)
	}{
		{"barra final", func(r *http.Request) { r.URL.Path += "/"; r.RequestURI += "/" }},
		{"query", func(r *http.Request) { r.URL.RawQuery = "x=1"; r.RequestURI += "?x=1" }},
		{"force query", func(r *http.Request) { r.URL.ForceQuery = true; r.RequestURI += "?" }},
		{"raw path", func(r *http.Request) {
			r.URL.RawPath = "/api/vec/bolsa/%70ropuestas-llamamiento"
			r.RequestURI = r.URL.RawPath
		}},
		{"forma absoluta", func(r *http.Request) { r.URL.Scheme = "https"; r.URL.Host = "interno.example" }},
		{"request uri ausente", func(r *http.Request) { r.RequestURI = "" }},
	}
	for _, caso := range casosRuta {
		t.Run(caso.nombre, func(t *testing.T) {
			antesPreparador, antesProponente := preparador.llamadas, proponente.llamadas
			peticion := peticionPropuestaHTTPPrueba(cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef))
			caso.mutar(peticion)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusNotFound || codigoErrorLlamamientoPrueba(t, respuesta) != "recurso_no_encontrado" {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
			}
			if preparador.llamadas != antesPreparador || proponente.llamadas != antesProponente {
				t.Fatal("una ruta no canonica alcanzo dependencias")
			}
		})
	}

	for _, metodo := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(metodo, func(t *testing.T) {
			peticion := peticionPropuestaHTTPPrueba(cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef))
			peticion.Method = metodo
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusMethodNotAllowed || respuesta.Header().Get("Allow") != http.MethodPost ||
				codigoErrorLlamamientoPrueba(t, respuesta) != "metodo_no_permitido" {
				t.Fatalf("metodo=%s estado=%d allow=%q cuerpo=%s", metodo, respuesta.Code, respuesta.Header().Get("Allow"), respuesta.Body.String())
			}
		})
	}
}

func TestHandlerPropuestasLlamamientoRechazaCabecerasNoConfiablesYTrailers(t *testing.T) {
	preparador, proponente, handler, propuesta := dependenciasHandlerPropuestaHTTPPrueba(t)
	casos := []struct {
		nombre string
		mutar  func(*http.Request)
		codigo string
	}{
		{"idempotency key", func(r *http.Request) { r.Header["Idempotency-Key"] = []string{"018fb754-5ad7-7dac-b0af-7b368f1a63c1"} }, "idempotencia_http_no_soportada"},
		{"cookie vacia", func(r *http.Request) { r.Header["Cookie"] = []string{""} }, "peticion_no_permitida"},
		{"proxy authorization", func(r *http.Request) { r.Header.Set("Proxy-Authorization", "Basic secreto") }, "peticion_no_permitida"},
		{"forwarded", func(r *http.Request) { r.Header.Set("Forwarded", "for=203.0.113.1") }, "peticion_no_permitida"},
		{"identidad vec", func(r *http.Request) { r.Header.Set("X-Vec-Actor", "admin") }, "peticion_no_permitida"},
		{"identidad auth", func(r *http.Request) { r.Header.Set("X-Auth-Roles", "admin") }, "peticion_no_permitida"},
		{"x forwarded", func(r *http.Request) { r.Header.Set("X-Forwarded-User", "admin") }, "peticion_no_permitida"},
		{"real ip", func(r *http.Request) { r.Header.Set("X-Real-IP", "203.0.113.1") }, "peticion_no_permitida"},
		{"envoy", func(r *http.Request) { r.Header.Set("X-Envoy-External-Address", "203.0.113.1") }, "peticion_no_permitida"},
		{"trailer declarado", func(r *http.Request) { r.Header.Set("Trailer", "X-Firma") }, "peticion_no_permitida"},
		{"trailer materializado", func(r *http.Request) { r.Trailer = http.Header{"X-Firma": []string{"valor"}} }, "peticion_no_permitida"},
		{"transfer encoding", func(r *http.Request) { r.TransferEncoding = []string{"chunked"} }, "peticion_no_permitida"},
		{"content encoding", func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, "peticion_no_permitida"},
		{"if match", func(r *http.Request) { r.Header.Set("If-Match", `"estado"`) }, "peticion_no_permitida"},
		{"accept ausente", func(r *http.Request) { r.Header.Del("Accept") }, "peticion_no_permitida"},
		{"accept duplicado", func(r *http.Request) { r.Header["Accept"] = []string{"application/json", "application/json"} }, "peticion_no_permitida"},
		{"accept no canonico", func(r *http.Request) { r.Header.Set("Accept", "application/*") }, "peticion_no_permitida"},
		{"content type no canonico", func(r *http.Request) { r.Header.Set("Content-Type", "Application/JSON") }, "peticion_no_permitida"},
		{"longitud desconocida", func(r *http.Request) { r.ContentLength = -1 }, "peticion_no_permitida"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			antesPreparador, antesProponente := preparador.llamadas, proponente.llamadas
			peticion := peticionPropuestaHTTPPrueba(cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef))
			caso.mutar(peticion)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest || codigoErrorLlamamientoPrueba(t, respuesta) != caso.codigo {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
			}
			if preparador.llamadas != antesPreparador || proponente.llamadas != antesProponente {
				t.Fatal("metadatos no confiables alcanzaron dependencias")
			}
		})
	}
}

func TestHandlerPropuestasLlamamientoRechazaJSONAmbiguoIdentidadYReferenciasPersonales(t *testing.T) {
	preparador, proponente, handler, propuesta := dependenciasHandlerPropuestaHTTPPrueba(t)
	casos := []struct {
		nombre string
		cuerpo string
	}{
		{"vacio", ""},
		{"mal formado", `{"data":`},
		{"dos documentos", cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef) + `{}`},
		{"data mayuscula", `{"Data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v1","necesidad_id":"necesidad:01K0VS7P"}}`},
		{"data duplicada", `{"data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v1","necesidad_id":"necesidad:01K0VS7P"},"data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v1","necesidad_id":"necesidad:01K0VS7P"}}`},
		{"campo desconocido", `{"data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v1","necesidad_id":"necesidad:01K0VS7P","forzar":true}}`},
		{"actor en cuerpo", `{"data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v1","necesidad_id":"necesidad:01K0VS7P","actor":{"rol":"admin"}}}`},
		{"perfil en cuerpo", `{"data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v1","necesidad_id":"necesidad:01K0VS7P","perfil_activo_ref":"prf_admin"}}`},
		{"esquema desconocido", `{"data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v2","necesidad_id":"necesidad:01K0VS7P"}}`},
		{"dni", `{"data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v1","necesidad_id":"necesidad:dni:12345678Z"}}`},
		{"espacios", `{"data":{"esquema":"vec.bolsa.propuesta-llamamiento.solicitud.v1","necesidad_id":" necesidad:01K0VS7P"}}`},
		{"unicode no nfc", "{\"data\":{\"esquema\":\"vec.bolsa.propuesta-llamamiento.solicitud.v1\",\"necesidad_id\":\"necesidad:cafe\\u0301\"}}"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			antesPreparador, antesProponente := preparador.llamadas, proponente.llamadas
			peticion := peticionPropuestaHTTPPrueba(caso.cuerpo)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
			}
			if preparador.llamadas != antesPreparador || proponente.llamadas != antesProponente {
				t.Fatal("json invalido alcanzo dependencias")
			}
		})
	}

	cuerpoGrande := strings.Repeat("x", maximoCuerpoSolicitudLlamamientoBytes+1)
	peticion := peticionPropuestaHTTPPrueba(cuerpoGrande)
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusRequestEntityTooLarge ||
		codigoErrorLlamamientoPrueba(t, respuesta) != "peticion_demasiado_grande" {
		t.Fatalf("cuerpo grande: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}

func TestHandlerPropuestasLlamamientoFallaCerradoSiPreparadorNoLigaNecesidad(t *testing.T) {
	propuesta := propuestaHTTPPrueba(t)
	casos := []struct {
		nombre    string
		solicitud puertosbolsa.SolicitudProponerLlamamiento
	}{
		{"solicitud invalida", puertosbolsa.SolicitudProponerLlamamiento{}},
		{"necesidad distinta", solicitudHTTPPrueba(t, "necesidad:otra-referencia-opaca")},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			preparador := &preparadorPropuestaLlamamientoDoble{solicitud: caso.solicitud}
			proponente := &proponenteLlamamientoDoble{propuesta: propuesta}
			handler := handlerPropuestaHTTPPrueba(t, preparador, proponente)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticionPropuestaHTTPPrueba(cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef)))
			if respuesta.Code != http.StatusInternalServerError ||
				codigoErrorLlamamientoPrueba(t, respuesta) != "error_interno" || proponente.llamadas != 0 {
				t.Fatalf("estado=%d llamadas=%d cuerpo=%s", respuesta.Code, proponente.llamadas, respuesta.Body.String())
			}
		})
	}
}

func TestHandlerPropuestasLlamamientoClasificaErroresSinFiltrarCausas(t *testing.T) {
	propuesta := propuestaHTTPPrueba(t)
	secreto := errors.New("dsn=postgres://usuario:clave@servidor/base")
	casos := []struct {
		nombre string
		err    error
		estado int
		codigo string
	}{
		{"autenticacion", errors.Join(ErrAutenticacionInternaAusente, secreto), http.StatusUnauthorized, "autenticacion_requerida"},
		{"autorizacion", errors.Join(dominiovec.ErrAutorizacionDenegada, secreto), http.StatusForbidden, "acceso_denegado"},
		{"sin elegible", errors.Join(dominiobolsa.ErrSinParticipacionElegible, secreto), http.StatusForbidden, "acceso_denegado"},
		{"necesidad ausente", errors.Join(puertosbolsa.ErrRecursoNecesidadNoEncontrado, secreto), http.StatusNotFound, "necesidad_no_disponible"},
		{"conflicto", errors.Join(puertosbolsa.ErrPersistenciaPropuestaNoDisponible, puertosbolsa.ErrNecesidadLlamamientoYaPropuesta, secreto), http.StatusConflict, "propuesta_en_conflicto"},
		{"dependencia", errors.Join(puertosvec.ErrFuenteAutorizacionNoDisponible, secreto), http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"persistencia", errors.Join(puertosbolsa.ErrPersistenciaPropuestaNoDisponible, secreto), http.StatusServiceUnavailable, "servicio_no_disponible"},
		{"desconocido", secreto, http.StatusInternalServerError, "error_interno"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre+" preparador", func(t *testing.T) {
			preparador := &preparadorPropuestaLlamamientoDoble{err: caso.err}
			proponente := &proponenteLlamamientoDoble{propuesta: propuesta}
			handler := handlerPropuestaHTTPPrueba(t, preparador, proponente)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticionPropuestaHTTPPrueba(cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef)))
			verificarErrorHTTPPropuesta(t, respuesta, caso.estado, caso.codigo, secreto.Error())
			if proponente.llamadas != 0 {
				t.Fatal("error del preparador alcanzo el caso de uso")
			}
		})
		t.Run(caso.nombre+" caso de uso", func(t *testing.T) {
			preparador := &preparadorPropuestaLlamamientoDoble{solicitud: solicitudHTTPPrueba(t, propuesta.NecesidadRef)}
			proponente := &proponenteLlamamientoDoble{err: caso.err}
			handler := handlerPropuestaHTTPPrueba(t, preparador, proponente)
			respuesta := httptest.NewRecorder()
			handler.ServeHTTP(respuesta, peticionPropuestaHTTPPrueba(cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef)))
			verificarErrorHTTPPropuesta(t, respuesta, caso.estado, caso.codigo, secreto.Error())
		})
	}
}

func TestHandlerPropuestasLlamamientoRechazaSalidaManipulada(t *testing.T) {
	propuesta := propuestaHTTPPrueba(t)
	propuesta.HuellaContenidoSHA256 = strings.Repeat("0", 64)
	preparador := &preparadorPropuestaLlamamientoDoble{solicitud: solicitudHTTPPrueba(t, propuesta.NecesidadRef)}
	proponente := &proponenteLlamamientoDoble{propuesta: propuesta}
	handler := handlerPropuestaHTTPPrueba(t, preparador, proponente)
	respuesta := httptest.NewRecorder()
	handler.ServeHTTP(respuesta, peticionPropuestaHTTPPrueba(cuerpoSolicitudPropuestaHTTP(propuesta.NecesidadRef)))

	if respuesta.Code != http.StatusInternalServerError || respuesta.Header().Get("ETag") != "" ||
		codigoErrorLlamamientoPrueba(t, respuesta) != "error_interno" {
		t.Fatalf("salida manipulada admitida: estado=%d etag=%q cuerpo=%s", respuesta.Code, respuesta.Header().Get("ETag"), respuesta.Body.String())
	}
}

func dependenciasHandlerPropuestaHTTPPrueba(
	t *testing.T,
) (*preparadorPropuestaLlamamientoDoble, *proponenteLlamamientoDoble, http.Handler, dominiobolsa.PropuestaLlamamiento) {
	t.Helper()
	propuesta := propuestaHTTPPrueba(t)
	preparador := &preparadorPropuestaLlamamientoDoble{solicitud: solicitudHTTPPrueba(t, propuesta.NecesidadRef)}
	proponente := &proponenteLlamamientoDoble{propuesta: propuesta}
	return preparador, proponente, handlerPropuestaHTTPPrueba(t, preparador, proponente), propuesta
}

func handlerPropuestaHTTPPrueba(
	t *testing.T,
	preparador PreparadorSolicitudPropuestaLlamamientoInterno,
	proponente ProponentePrimerLlamamiento,
) http.Handler {
	t.Helper()
	handler, err := NuevoHandlerPropuestasLlamamiento(preparador, proponente)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	return handler
}

func peticionPropuestaHTTPPrueba(cuerpo string) *http.Request {
	peticion := httptest.NewRequest(http.MethodPost, RutaPropuestasLlamamiento, strings.NewReader(cuerpo))
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Content-Type", "application/json")
	return peticion
}

func cuerpoSolicitudPropuestaHTTP(necesidadRef string) string {
	contenido, _ := json.Marshal(envelopeSolicitudPropuestaLlamamientoJSON{Data: &solicitudPropuestaLlamamientoJSON{
		Esquema: "vec.bolsa.propuesta-llamamiento.solicitud.v1", NecesidadID: necesidadRef,
	}})
	return string(contenido)
}

func codigoErrorLlamamientoPrueba(t *testing.T, respuesta *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope envelopeErrorLlamamiento
	if err := json.Unmarshal(respuesta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error no json: %v cuerpo=%q", err, respuesta.Body.String())
	}
	if envelope.Error.CorrelacionRef == "" {
		t.Fatalf("error sin correlacion: %s", respuesta.Body.String())
	}
	return envelope.Error.Codigo
}

func verificarErrorHTTPPropuesta(
	t *testing.T,
	respuesta *httptest.ResponseRecorder,
	estado int,
	codigo, secreto string,
) {
	t.Helper()
	if respuesta.Code != estado || codigoErrorLlamamientoPrueba(t, respuesta) != codigo {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if strings.Contains(respuesta.Body.String(), secreto) {
		t.Fatalf("causa filtrada: %s", respuesta.Body.String())
	}
}

func solicitudHTTPPrueba(t *testing.T, necesidadRef string) puertosbolsa.SolicitudProponerLlamamiento {
	t.Helper()
	actor := actorHTTPPropuestaPrueba(t)
	solicitud := puertosbolsa.SolicitudProponerLlamamiento{
		Actor: actor, PerfilActivoRef: actor.PerfilActivoRef,
		AutenticacionRef: "aut_" + strings.Repeat("a", 22),
		SesionRef:        "ses_" + strings.Repeat("s", 22),
		NecesidadRef:     necesidadRef,
		CorrelacionRef:   "corr_" + strings.Repeat("c", 22),
	}
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("solicitud: %v", err)
	}
	return solicitud
}

func actorHTTPPropuestaPrueba(t *testing.T) dominiovec.ContextoActor {
	t.Helper()
	token := func(caracter string) string { return strings.Repeat(caracter, 22) }
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_" + token("c"), Metodo: dominiovec.AuthMethodCertificate,
		Garantia: dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef: "vca_" + token("v"), VinculoVersion: 1, CuentaRef: cuenta.CuentaRef,
		PersonaRef: "per_" + token("p"), PersonaVersion: 1,
		PerfilActivoRef: "prf_" + token("r"), PerfilVersion: 1,
		Estado:       dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde: instanteHTTPPropuestaPrueba.Add(-24 * time.Hour),
		VigenteHasta: instanteHTTPPropuestaPrueba.Add(24 * time.Hour),
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, instanteHTTPPropuestaPrueba.Add(-time.Minute))
	if err != nil {
		t.Fatalf("actor: %v", err)
	}
	return actor
}

func propuestaHTTPPrueba(t *testing.T) dominiobolsa.PropuestaLlamamiento {
	t.Helper()
	huella := func(caracter string) string { return strings.Repeat(caracter, 64) }
	bolsa, err := dominiobolsa.NuevaBolsaConstituida(dominiobolsa.AltaBolsaConstituida{
		BolsaRef: "bolsa:01K0VRZZ", Version: 3, ProcesoRef: "proceso:01K0VRZY",
		CategoriaRef: "categoria:auxiliar", ListadoDefinitivoRef: "listado:01K0VRZX",
		VersionListado: 7, HuellaListadoSHA256: huella("a"),
		ResolucionConstitucionRef: "resolucion:01K0VRZW", HuellaResolucionSHA256: huella("b"),
		ConstituidaEn: instanteHTTPPropuestaPrueba.Add(-48 * time.Hour),
		VigenteDesde:  instanteHTTPPropuestaPrueba.Add(-24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("bolsa: %v", err)
	}
	huellaBolsa, err := bolsa.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatalf("huella bolsa: %v", err)
	}
	necesidad, err := dominiobolsa.NuevaNecesidadCobertura(dominiobolsa.AltaNecesidadCobertura{
		NecesidadRef: "necesidad:01K0VS7P", Version: 2,
		BolsaRef: bolsa.BolsaRef, VersionBolsa: bolsa.Version, HuellaBolsaSHA256: huellaBolsa,
		CategoriaRef: bolsa.CategoriaRef, PuestoRef: "puesto:01K0VT10", UnidadRef: "unidad:01K0VT11",
		TipoCoberturaRef: "tipo_cobertura:01K0VT12", NumeroPuestos: 1,
		InicioPrevisto: instanteHTTPPropuestaPrueba.Add(24 * time.Hour),
		FinPrevisto:    instanteHTTPPropuestaPrueba.Add(60 * 24 * time.Hour),
		CreadaEn:       instanteHTTPPropuestaPrueba.Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("necesidad: %v", err)
	}
	politica, err := dominiobolsa.NuevaReferenciaPoliticaLlamamiento(dominiobolsa.ReferenciaPoliticaLlamamiento{
		PoliticaRef: "politica:01K0VS6N", Clave: "llamamiento.reglamento_publicado",
		Version: 9, HuellaSHA256: huella("c"),
		PublicadaEn:  instanteHTTPPropuestaPrueba.Add(-72 * time.Hour),
		VigenteDesde: instanteHTTPPropuestaPrueba.Add(-48 * time.Hour),
	})
	if err != nil {
		t.Fatalf("politica: %v", err)
	}
	altaParticipacion := instanteHTTPPropuestaPrueba.Add(-24 * time.Hour)
	participacion, err := dominiobolsa.NuevaParticipacionBolsa(dominiobolsa.AltaParticipacionBolsa{
		ParticipacionRef: "participacion:01K0VSA", BolsaRef: bolsa.BolsaRef,
		SujetoRef: "sujeto:01K0VTA", Version: 2, AltaEn: altaParticipacion,
		Situaciones: []dominiobolsa.SituacionParticipacionBolsa{{
			Secuencia: 1, EstadoClave: "estado_operativo", EstadoVersion: 4,
			HuellaEstadoSHA256: huella("d"), CausaClave: "constitucion_bolsa", CausaVersion: 1,
			HuellaCausaSHA256: huella("e"), DecisionRef: "decision:situacion:01A",
			HuellaDecisionSHA256: huella("f"), Desde: altaParticipacion,
		}},
	})
	if err != nil {
		t.Fatalf("participacion: %v", err)
	}
	instantanea, err := dominiobolsa.NuevaInstantaneaOrdenBolsa(dominiobolsa.AltaInstantaneaOrdenBolsa{
		InstantaneaRef: "instantanea:01K0VS8Q", Version: 4, Bolsa: bolsa,
		ReferidaEn: instanteHTTPPropuestaPrueba,
		GeneradaEn: instanteHTTPPropuestaPrueba.Add(time.Minute),
		Entradas:   []dominiobolsa.EntradaOrdenBolsa{{Orden: 1, Participacion: participacion}},
	})
	if err != nil {
		t.Fatalf("instantanea: %v", err)
	}
	huellaNecesidad, err := necesidad.HuellaCanonicaSHA256()
	if err != nil {
		t.Fatalf("huella necesidad: %v", err)
	}
	evaluacion := dominiobolsa.EvaluacionParticipacionLlamamiento{
		ParticipacionRef: participacion.ParticipacionRef, SujetoRef: participacion.SujetoRef,
		Orden: 1, SituacionSecuencia: 1, EstadoClave: "estado_operativo", EstadoVersion: 4,
		HuellaEstadoSHA256: huella("d"),
		NecesidadRef:       necesidad.NecesidadRef, VersionNecesidad: necesidad.Version,
		HuellaNecesidadSHA256: huellaNecesidad,
		InstantaneaRef:        instantanea.InstantaneaRef, VersionInstantanea: instantanea.Version,
		HuellaInstantaneaSHA256: instantanea.HuellaContenidoSHA256,
		PoliticaRef:             politica.PoliticaRef, VersionPolitica: politica.Version,
		HuellaPoliticaSHA256: politica.HuellaSHA256,
		Resultado:            dominiobolsa.ResultadoElegible,
		Motivos: []dominiobolsa.MotivoEvaluacionLlamamiento{{
			Clave: "resultado_final", ReglaRef: "regla:seleccion:A", VersionRegla: 3,
			HuellaReglaSHA256: huella("1"),
		}},
		EntradaEvaluacionRef: "entrada:evaluacion:A", HuellaEntradaSHA256: huella("2"),
		ResultadoEvaluacionRef: "resultado:evaluacion:A", HuellaResultadoSHA256: huella("3"),
		EvaluadaEn: instantanea.GeneradaEn,
	}
	propuesta, err := dominiobolsa.ProponerPrimerLlamamiento(dominiobolsa.OrdenProponerPrimerLlamamiento{
		PropuestaRef: "propuesta:01K0VS9A7Q", Bolsa: bolsa, Necesidad: necesidad,
		Instantanea: instantanea, Politica: politica,
		Evaluaciones: []dominiobolsa.EvaluacionParticipacionLlamamiento{evaluacion},
		GeneradaEn:   instanteHTTPPropuestaPrueba.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("propuesta: %v", err)
	}
	return propuesta
}
