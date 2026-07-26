package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var instantePrueba = time.Date(2026, 7, 23, 12, 30, 0, 123000000, time.UTC)

const claveIdempotenciaPrueba = "4d36e96e-e325-4f9b-bebc-291d91d6f732"

type autoridadPrueba struct {
	mu       sync.Mutex
	contexto application.SolicitudRegistrarExpediente
	err      error
	llamadas int
	alLlamar func()
}

func (a *autoridadPrueba) ResolverContextoCanalAlta(
	context.Context,
) (application.SolicitudRegistrarExpediente, error) {
	a.mu.Lock()
	a.llamadas++
	alLlamar := a.alLlamar
	contexto, err := a.contexto, a.err
	a.mu.Unlock()
	if alLlamar != nil {
		alLlamar()
	}
	return contexto, err
}

func (a *autoridadPrueba) numeroLlamadas() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.llamadas
}

type ejecutorPrueba struct {
	mu       sync.Mutex
	recibo   ports.ReciboAlta
	err      error
	llamadas int
	comandos []application.SolicitudRegistrarExpediente
}

func (e *ejecutorPrueba) Registrar(
	_ context.Context,
	comando application.SolicitudRegistrarExpediente,
) (ports.ReciboAlta, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.llamadas++
	clon, _ := comando.Solicitud.Clonar()
	comando.Solicitud = clon
	e.comandos = append(e.comandos, comando)
	return e.recibo, e.err
}

func (e *ejecutorPrueba) instantanea() (int, []application.SolicitudRegistrarExpediente) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.llamadas, append([]application.SolicitudRegistrarExpediente(nil), e.comandos...)
}

type relojPrueba struct{ instante time.Time }

func (r relojPrueba) Ahora() time.Time { return r.instante }

func contextoCanalValidoPrueba() application.SolicitudRegistrarExpediente {
	return application.SolicitudRegistrarExpediente{
		AutenticacionRef: "aut_0123456789abcdefghijkl",
		SesionRef:        "ses_0123456789abcdefghijkl",
		PerfilRef:        "prf_0123456789abcdefghijkl",
		OrganizacionRef:  "organizacion:rrhh:001",
	}
}

func reciboValidoPrueba() ports.ReciboAlta {
	return ports.ReciboAlta{
		ExpedienteRef: "expediente:ct:0001",
		NumeroVisible: "2026/CT-0001",
		Version:       1,
		ReciboRef:     "recibo:ct:0001",
		AuditoriaRef:  "auditoria:ct:0001",
		EventoRef:     "evento:ct:0001",
		ConfirmadaEn:  instantePrueba,
	}
}

func cuerpoValidoPrueba() []byte {
	return []byte(`{
		"clave_idempotencia":"4d36e96e-e325-4f9b-bebc-291d91d6f732",
		"solicitud":{
			"centro_ref":"centro:solicitante:001",
			"contacto_ref":"contacto:opaco:001",
			"categoria_ref":"categoria:tecnica:001",
			"grupo_subgrupo":"A1",
			"motivo_clave":"necesidad_temporal",
			"detalle":"Cobertura temporal de una necesidad catalogada.",
			"periodo":{"inicio":"2026-08-01T00:00:00Z","fin":"2026-12-31T00:00:00Z"},
			"rc":{"existe":false},
			"documentos_adjuntos":["documento:opaco:001"],
			"observaciones":"Tramitación ordinaria."
		}
	}`)
}

func nuevaPeticionPrueba(t *testing.T, cuerpo []byte) *http.Request {
	t.Helper()
	peticion := httptest.NewRequest(http.MethodPost, RutaAltaSolicitudes, bytes.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func nuevoEscenarioPrueba(
	t *testing.T,
) (http.Handler, *autoridadPrueba, *ejecutorPrueba) {
	t.Helper()
	autoridad := &autoridadPrueba{contexto: contextoCanalValidoPrueba()}
	ejecutor := &ejecutorPrueba{recibo: reciboValidoPrueba()}
	manejador, err := NuevoManejadorAlta(
		autoridad,
		ejecutor,
		relojPrueba{instante: instantePrueba},
	)
	if err != nil {
		t.Fatalf("NuevoManejadorAlta() error = %v", err)
	}
	return manejador, autoridad, ejecutor
}

func ejecutarPeticionPrueba(
	t *testing.T,
	manejador http.Handler,
	peticion *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, peticion)
	return respuesta
}

func codigoErrorPrueba(t *testing.T, respuesta *httptest.ResponseRecorder) string {
	t.Helper()
	var envoltorio envoltorioErrorAlta
	if err := json.Unmarshal(respuesta.Body.Bytes(), &envoltorio); err != nil {
		t.Fatalf("error no es JSON: %v; cuerpo=%q", err, respuesta.Body.String())
	}
	claveEsperada := "api.contratacion_temporal.alta.error." + envoltorio.Error.Codigo
	if envoltorio.Error.ClaveI18n != claveEsperada || envoltorio.Error.CorrelacionRef == "" {
		t.Fatalf("error público incompleto: %+v", envoltorio.Error)
	}
	return envoltorio.Error.Codigo
}

func TestNuevoManejadorAltaExigeDependencias(t *testing.T) {
	autoridad := &autoridadPrueba{}
	ejecutor := &ejecutorPrueba{}
	reloj := relojPrueba{instante: instantePrueba}
	casos := []struct {
		nombre    string
		autoridad AutoridadContextoCanal
		ejecutor  EjecutorAlta
		reloj     ports.Reloj
	}{
		{"sin autoridad", nil, ejecutor, reloj},
		{"sin ejecutor", autoridad, nil, reloj},
		{"sin reloj", autoridad, ejecutor, nil},
		{"autoridad tipada nula", (*autoridadPrueba)(nil), ejecutor, reloj},
		{"ejecutor tipado nulo", autoridad, (*ejecutorPrueba)(nil), reloj},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := NuevoManejadorAlta(caso.autoridad, caso.ejecutor, caso.reloj); !errors.Is(
				err,
				ErrManejadorAltaInvalido,
			) {
				t.Fatalf("error = %v; esperado ErrManejadorAltaInvalido", err)
			}
		})
	}
}

func TestManejadorAltaConfirmaYMinimizaRecibo(t *testing.T) {
	manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
	respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
	if respuesta.Code != http.StatusCreated {
		t.Fatalf("estado = %d; cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	var envoltorio map[string]map[string]any
	if err := json.Unmarshal(respuesta.Body.Bytes(), &envoltorio); err != nil {
		t.Fatal(err)
	}
	datos := envoltorio["data"]
	claves := make([]string, 0, len(datos))
	for clave := range datos {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	esperadas := []string{"confirmada_en", "expediente_ref", "numero_visible", "recibo_ref", "version"}
	if !reflect.DeepEqual(claves, esperadas) {
		t.Fatalf("campos de salida = %v; esperados %v", claves, esperadas)
	}
	prohibidos := []string{
		"auditoria", "evento", "correlacion", "identidad", "actor", "perfil",
		"organizacion", "idempotencia", claveIdempotenciaPrueba,
	}
	for _, prohibido := range prohibidos {
		if strings.Contains(strings.ToLower(respuesta.Body.String()), strings.ToLower(prohibido)) {
			t.Fatalf("la salida contiene dato prohibido %q: %s", prohibido, respuesta.Body.String())
		}
	}
	if autoridad.numeroLlamadas() != 1 {
		t.Fatalf("llamadas a autoridad = %d", autoridad.numeroLlamadas())
	}
	llamadas, comandos := ejecutor.instantanea()
	if llamadas != 1 || len(comandos) != 1 ||
		comandos[0].ClaveIdempotencia != claveIdempotenciaPrueba ||
		comandos[0].OrganizacionRef != contextoCanalValidoPrueba().OrganizacionRef ||
		comandos[0].Solicitud.Validar() != nil {
		t.Fatalf("comando no ligado: llamadas=%d comandos=%+v", llamadas, comandos)
	}
	comprobarCabecerasSegurasPrueba(t, respuesta)
}

func TestManejadorAltaConservaExitoConfirmadoTrasCancelacion(t *testing.T) {
	manejador, _, ejecutor := nuevoEscenarioPrueba(t)
	ejecutor.err = context.Canceled
	respuesta := ejecutarPeticionPrueba(t, manejador, nuevaPeticionPrueba(t, cuerpoValidoPrueba()))
	if respuesta.Code != http.StatusCreated {
		t.Fatalf("degradó recibo confirmado: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
}

func TestManejadorAltaCierraRutaMetodoYNegociacion(t *testing.T) {
	casos := []struct {
		nombre     string
		modificar  func(*http.Request)
		estado     int
		codigo     string
		autoridad0 bool
	}{
		{"ruta distinta", func(r *http.Request) { r.URL.Path += "/" }, 404, "recurso_no_encontrado", true},
		{"ruta interna histórica", func(r *http.Request) {
			r.URL.Path = "/api/interno/v1/contratacion-temporal/solicitudes"
		}, 404, "recurso_no_encontrado", true},
		{"ruta escapada", func(r *http.Request) { r.URL.RawPath = RutaAltaSolicitudes + "%2f" }, 404, "recurso_no_encontrado", true},
		{"query", func(r *http.Request) { r.URL.RawQuery = "actor=inyectado" }, 400, "peticion_no_permitida", true},
		{"force query", func(r *http.Request) { r.URL.ForceQuery = true }, 400, "peticion_no_permitida", true},
		{"get", func(r *http.Request) { r.Method = http.MethodGet }, 405, "metodo_no_permitido", true},
		{"options", func(r *http.Request) { r.Method = http.MethodOptions }, 405, "metodo_no_permitido", true},
		{"content type texto", func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, 415, "tipo_contenido_no_admitido", true},
		{"charset no permitido", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; charset=iso-8859-1") }, 415, "tipo_contenido_no_admitido", true},
		{"boundary no permitido", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; boundary=valor") }, 415, "tipo_contenido_no_admitido", true},
		{"profile no permitido", func(r *http.Request) { r.Header.Set("Content-Type", "application/json; profile=otro") }, 415, "tipo_contenido_no_admitido", true},
		{"accept incompatible", func(r *http.Request) { r.Header.Set("Accept", "text/html") }, 406, "representacion_no_aceptable", true},
		{"accept q cero", func(r *http.Request) { r.Header.Set("Accept", "application/json;q=0") }, 406, "representacion_no_aceptable", true},
		{"accept exacto excluido prevalece", func(r *http.Request) { r.Header.Set("Accept", "application/json;q=0, */*;q=1") }, 406, "representacion_no_aceptable", true},
		{"accept subtipo excluido prevalece", func(r *http.Request) { r.Header.Set("Accept", "application/*;q=0, */*;q=1") }, 406, "representacion_no_aceptable", true},
		{"accept profile no coincide", func(r *http.Request) { r.Header.Set("Accept", "application/json;profile=otro") }, 406, "representacion_no_aceptable", true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			manejador, autoridad, ejecutor := nuevoEscenarioPrueba(t)
			peticion := nuevaPeticionPrueba(t, cuerpoValidoPrueba())
			caso.modificar(peticion)
			respuesta := ejecutarPeticionPrueba(t, manejador, peticion)
			if respuesta.Code != caso.estado || codigoErrorPrueba(t, respuesta) != caso.codigo {
				t.Fatalf("estado/código = %d/%s; cuerpo=%s", respuesta.Code, codigoErrorPrueba(t, respuesta), respuesta.Body.String())
			}
			if caso.autoridad0 && (autoridad.numeroLlamadas() != 0) {
				t.Fatalf("la autoridad recibió entrada rechazada")
			}
			if llamadas, _ := ejecutor.instantanea(); llamadas != 0 {
				t.Fatalf("el ejecutor recibió entrada rechazada")
			}
			if caso.estado == http.StatusMethodNotAllowed && respuesta.Header().Get("Allow") != http.MethodPost {
				t.Fatalf("Allow = %q", respuesta.Header().Get("Allow"))
			}
		})
	}
}

func TestManejadorAltaAdmiteAcceptCompatible(t *testing.T) {
	for _, accept := range []string{"", "application/json", "application/*", "*/*", "text/html;q=0, application/json;q=0.5"} {
		t.Run(strings.ReplaceAll(accept, "/", "_"), func(t *testing.T) {
			manejador, _, _ := nuevoEscenarioPrueba(t)
			peticion := nuevaPeticionPrueba(t, cuerpoValidoPrueba())
			if accept == "" {
				peticion.Header.Del("Accept")
			} else {
				peticion.Header.Set("Accept", accept)
			}
			if respuesta := ejecutarPeticionPrueba(t, manejador, peticion); respuesta.Code != http.StatusCreated {
				t.Fatalf("Accept %q rechazado: %d %s", accept, respuesta.Code, respuesta.Body.String())
			}
		})
	}
}

func TestContratoOpenAPISeCorrespondeConDTO(t *testing.T) {
	_, fichero, _, _ := runtime.Caller(0)
	ruta := filepath.Clean(filepath.Join(
		filepath.Dir(fichero),
		"..", "..", "..", "..", "..",
		"docs", "api", "contratacion_temporal_alta_interna_v1.yaml",
	))
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var documento map[string]any
	if err := yaml.Unmarshal(contenido, &documento); err != nil {
		t.Fatalf("OpenAPI no parsea: %v", err)
	}
	if documento["openapi"] != "3.1.0" {
		t.Fatalf("versión OpenAPI = %v", documento["openapi"])
	}
	if _, existe := documento["servers"]; existe {
		t.Fatal("OpenAPI no debe declarar servidores")
	}
	componentes := mapaPrueba(t, documento, "components")
	esquemas := mapaPrueba(t, componentes, "schemas")
	operacion := mapaPrueba(
		t,
		mapaPrueba(
			t,
			mapaPrueba(t, documento, "paths"),
			RutaAltaSolicitudes,
		),
		"post",
	)
	esquemaEntrada := mapaPrueba(
		t,
		mapaPrueba(
			t,
			mapaPrueba(t, operacion, "requestBody"),
			"content",
		),
		"application/json",
	)
	if referencia, correcta := mapaPrueba(t, esquemaEntrada, "schema")["$ref"].(string); !correcta || referencia != "#/components/schemas/SolicitudAltaV1" {
		t.Fatalf("requestBody no usa el sobre cerrado: %v", referencia)
	}
	if numeroPrueba(t, operacion, "x-maximo-cuerpo-bytes") != MaximoCuerpoAltaBytes ||
		numeroPrueba(t, operacion, "x-maximo-profundidad-json") != profundidadMaximaJSONAlta ||
		numeroPrueba(t, operacion, "x-maximo-tokens-json") != tokensMaximosJSONAlta {
		t.Fatal("límites técnicos Go/OpenAPI desalineados")
	}
	sobre := mapaPrueba(t, esquemas, "SolicitudAltaV1")
	if sobre["additionalProperties"] != false {
		t.Fatal("SolicitudAltaV1 no está cerrada")
	}
	propiedadesSobre := mapaPrueba(t, sobre, "properties")
	compararClavesDTOPrueba(t, reflect.TypeOf(solicitudAltaJSON{}), propiedadesSobre)
	solicitud := mapaPrueba(t, esquemas, "SolicitudCentroAltaV1")
	if solicitud["additionalProperties"] != false {
		t.Fatal("SolicitudCentroAltaV1 no está cerrada")
	}
	propiedades := mapaPrueba(t, solicitud, "properties")
	compararClavesDTOPrueba(t, reflect.TypeOf(solicitudCentroJSON{}), propiedades)
	if numeroPrueba(t, mapaPrueba(t, propiedades, "documentos_adjuntos"), "maxItems") != MaximoDocumentosAdjuntos ||
		numeroPrueba(t, mapaPrueba(t, esquemas, "TextoDetalleV1"), "maxLength") != MaximoCaracteresDetalle ||
		numeroPrueba(t, mapaPrueba(t, mapaPrueba(t, esquemas, "ImporteRCEURV1"), "properties"), "centimos", "maximum") != int(MaximoCentimosJSON) {
		t.Fatal("límites Go/OpenAPI desalineados")
	}
	for _, nombre := range []string{
		"SolicitudAltaV1", "SolicitudCentroAltaV1", "PeriodoPrevistoAltaV1", "ImporteRCEURV1",
		"DeclaracionRCSinCreditoV1", "DeclaracionRCConCreditoV1",
		"ReciboAltaMinimoV1", "EnvelopeAltaConfirmadaV1", "ErrorAltaV1", "EnvelopeErrorAltaV1",
	} {
		if mapaPrueba(t, esquemas, nombre)["additionalProperties"] != false {
			t.Fatalf("%s no está cerrado", nombre)
		}
	}
	compararCatalogoErroresOpenAPIPrueba(t, operacion, componentes, esquemas)
}

func compararCatalogoErroresOpenAPIPrueba(
	t *testing.T,
	operacion map[string]any,
	componentes map[string]any,
	esquemas map[string]any,
) {
	t.Helper()
	esperados := map[string][]errorPublicoAlta{
		"400": {errorPeticionNoPermitida, errorPeticionNoValida},
		"401": {errorAutenticacionRequerida},
		"403": {errorAccesoDenegado},
		"404": {errorRecursoNoEncontrado},
		"405": {errorMetodoNoPermitido},
		"406": {errorRepresentacionNoAceptable},
		"408": {errorPeticionCancelada},
		"409": {errorClaveIdempotenciaReutilizada},
		"413": {errorPeticionDemasiadoGrande},
		"415": {errorTipoContenidoNoAdmitido},
		"422": {errorContenidoNoValido},
		"500": {errorInterno},
		"502": {errorResultadoNoConfiable},
		"503": {errorOperacionPendiente, errorServicioNoDisponible},
		"504": {errorPlazoAgotado},
	}
	respuestas := mapaPrueba(t, operacion, "responses")
	respuestasComponentes := mapaPrueba(t, componentes, "responses")
	union := make([]string, 0)
	parejasEsperadas := make(map[string]string)
	for estado, problemasEsperados := range esperados {
		codigosEsperados := make([]string, 0, len(problemasEsperados))
		for _, problema := range problemasEsperados {
			codigosEsperados = append(codigosEsperados, problema.codigo)
			parejasEsperadas[problema.codigo] = problema.claveI18n
		}
		respuesta := mapaPrueba(t, respuestas, estado)
		if referencia, existe := respuesta["$ref"].(string); existe {
			const prefijo = "#/components/responses/"
			respuesta = mapaPrueba(
				t,
				respuestasComponentes,
				strings.TrimPrefix(referencia, prefijo),
			)
		}
		contenido := mapaPrueba(t, respuesta, "content")
		jsonHTTP := mapaPrueba(t, contenido, "application/json")
		esquemaRespuesta := mapaPrueba(t, jsonHTTP, "schema")
		referencia, correcta := esquemaRespuesta["$ref"].(string)
		if !correcta {
			t.Fatalf("respuesta %s sin esquema referenciado", estado)
		}
		const prefijo = "#/components/schemas/"
		esquema := mapaPrueba(t, esquemas, strings.TrimPrefix(referencia, prefijo))
		codigos := codigosRestringidosPrueba(t, esquema)
		sort.Strings(codigos)
		sort.Strings(codigosEsperados)
		if !reflect.DeepEqual(codigos, codigosEsperados) {
			t.Fatalf("estado %s: códigos %v != %v", estado, codigos, codigosEsperados)
		}
		union = append(union, codigos...)
	}
	sort.Strings(union)
	union = unicosPrueba(union)
	catalogo := mapaPrueba(t, mapaPrueba(t, esquemas, "ErrorAltaV1"), "properties")
	codigo := mapaPrueba(t, catalogo, "codigo")
	var declarados []string
	for _, valor := range codigo["enum"].([]any) {
		declarados = append(declarados, valor.(string))
	}
	sort.Strings(declarados)
	if !reflect.DeepEqual(union, declarados) {
		t.Fatalf("catálogo estado-código incompleto: %v != %v", union, declarados)
	}
	parejasDeclaradas := parejasI18nOpenAPIPrueba(
		t,
		mapaPrueba(t, esquemas, "ErrorAltaV1"),
	)
	if !reflect.DeepEqual(parejasDeclaradas, parejasEsperadas) {
		t.Fatalf(
			"catálogo código-clave i18n desalineado: %v != %v",
			parejasDeclaradas,
			parejasEsperadas,
		)
	}
}

func parejasI18nOpenAPIPrueba(t *testing.T, esquema map[string]any) map[string]string {
	t.Helper()
	opciones, correcto := esquema["oneOf"].([]any)
	if !correcto {
		t.Fatalf("ErrorAltaV1 no liga códigos y claves i18n: %v", esquema["oneOf"])
	}
	resultado := make(map[string]string, len(opciones))
	for _, opcion := range opciones {
		restriccion, correcta := opcion.(map[string]any)
		if !correcta {
			t.Fatalf("restricción código-clave inválida: %T", opcion)
		}
		propiedades := mapaPrueba(t, restriccion, "properties")
		codigo, codigoCorrecto := mapaPrueba(t, propiedades, "codigo")["const"].(string)
		clave, claveCorrecta := mapaPrueba(t, propiedades, "clave_i18n")["const"].(string)
		if !codigoCorrecto || !claveCorrecta || codigo == "" || clave == "" {
			t.Fatalf("pareja código-clave inválida: %v", propiedades)
		}
		if _, duplicado := resultado[codigo]; duplicado {
			t.Fatalf("código i18n duplicado: %s", codigo)
		}
		resultado[codigo] = clave
	}
	return resultado
}

func codigosRestringidosPrueba(t *testing.T, esquema map[string]any) []string {
	t.Helper()
	errorProp := mapaPrueba(t, mapaPrueba(t, esquema, "properties"), "error")
	todos, correcto := errorProp["allOf"].([]any)
	if !correcto || len(todos) != 2 {
		t.Fatalf("restricción de error inválida: %v", errorProp)
	}
	restriccion := todos[1].(map[string]any)
	codigo := mapaPrueba(t, mapaPrueba(t, restriccion, "properties"), "codigo")
	if constante, existe := codigo["const"].(string); existe {
		return []string{constante}
	}
	lista, correcto := codigo["enum"].([]any)
	if !correcto {
		t.Fatalf("código sin const/enum: %v", codigo)
	}
	resultado := make([]string, 0, len(lista))
	for _, valor := range lista {
		resultado = append(resultado, valor.(string))
	}
	return resultado
}

func unicosPrueba(valores []string) []string {
	if len(valores) == 0 {
		return nil
	}
	resultado := []string{valores[0]}
	for _, valor := range valores[1:] {
		if valor != resultado[len(resultado)-1] {
			resultado = append(resultado, valor)
		}
	}
	return resultado
}

func compararClavesDTOPrueba(t *testing.T, tipo reflect.Type, propiedades map[string]any) {
	t.Helper()
	esperadas := make([]string, 0, tipo.NumField())
	for indice := 0; indice < tipo.NumField(); indice++ {
		esperadas = append(esperadas, strings.Split(tipo.Field(indice).Tag.Get("json"), ",")[0])
	}
	recibidas := make([]string, 0, len(propiedades))
	for clave := range propiedades {
		recibidas = append(recibidas, clave)
	}
	sort.Strings(esperadas)
	sort.Strings(recibidas)
	if !reflect.DeepEqual(esperadas, recibidas) {
		t.Fatalf("DTO/OpenAPI: %v != %v", esperadas, recibidas)
	}
}

func mapaPrueba(t *testing.T, origen map[string]any, claves ...string) map[string]any {
	t.Helper()
	actual := origen
	for _, clave := range claves {
		siguiente, correcto := actual[clave].(map[string]any)
		if !correcto {
			t.Fatalf("%s no es mapa: %T", clave, actual[clave])
		}
		actual = siguiente
	}
	return actual
}

func numeroPrueba(t *testing.T, origen map[string]any, claves ...string) int {
	t.Helper()
	actual := any(origen)
	for _, clave := range claves {
		mapa, correcto := actual.(map[string]any)
		if !correcto {
			t.Fatalf("ruta numérica inválida en %s", clave)
		}
		actual = mapa[clave]
	}
	switch valor := actual.(type) {
	case int:
		return valor
	case int64:
		return int(valor)
	default:
		t.Fatalf("valor no entero: %T %v", actual, actual)
		return 0
	}
}

func comprobarCabecerasSegurasPrueba(t *testing.T, respuesta *httptest.ResponseRecorder) {
	t.Helper()
	if respuesta.Header().Get("Content-Type") != "application/json; charset=utf-8" ||
		respuesta.Header().Get("Cache-Control") != "no-store, no-transform" ||
		respuesta.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("cabeceras defensivas incompletas: %v", respuesta.Header())
	}
	for _, prohibida := range []string{
		"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials",
		"Content-Encoding", "Retry-After",
	} {
		if valor := respuesta.Header().Get(prohibida); valor != "" {
			t.Fatalf("cabecera prohibida %s=%q", prohibida, valor)
		}
	}
}
