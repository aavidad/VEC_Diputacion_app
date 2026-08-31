package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

const claveSeleccionLlamamientoHTTPPrueba = "" +
	"4d36e96e-e325-4f9b-bebc-291d91d6f732"

type ejecutorSeleccionLlamamientoHTTPPrueba struct {
	mu       sync.Mutex
	recibo   application.DatosReciboSeleccionLlamamientoParaAdaptador
	err      error
	durante  func(context.Context)
	entradas []application.SolicitudSeleccionLlamamiento
}

func (e *ejecutorSeleccionLlamamientoHTTPPrueba) SeleccionarYLlamarParaAdaptador(
	ctx context.Context,
	solicitud application.SolicitudSeleccionLlamamiento,
) (application.DatosReciboSeleccionLlamamientoParaAdaptador, error) {
	e.mu.Lock()
	e.entradas = append(e.entradas, solicitud)
	e.mu.Unlock()
	if e.durante != nil {
		e.durante(ctx)
	}
	return e.recibo, e.err
}

func (e *ejecutorSeleccionLlamamientoHTTPPrueba) total() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.entradas)
}

func (e *ejecutorSeleccionLlamamientoHTTPPrueba) ultima() (
	application.SolicitudSeleccionLlamamiento,
	bool,
) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.entradas) == 0 {
		return application.SolicitudSeleccionLlamamiento{}, false
	}
	return e.entradas[len(e.entradas)-1], true
}

func reciboSeleccionLlamamientoHTTPPrueba() application.DatosReciboSeleccionLlamamientoParaAdaptador {
	return application.DatosReciboSeleccionLlamamientoParaAdaptador{
		ReciboRef: "recibo:llamamiento:http:001",
		ConfirmadaEn: time.Date(
			2026, 8, 31, 10, 0, 0, 123000000, time.UTC,
		),
	}
}

func nuevaPeticionSeleccionLlamamientoHTTPPrueba() *http.Request {
	cuerpo := `{"clave_idempotencia":"` +
		claveSeleccionLlamamientoHTTPPrueba + `"}`
	peticion := httptest.NewRequest(
		http.MethodPost,
		RutaSeleccionLlamamiento,
		strings.NewReader(cuerpo),
	)
	peticion.Header.Set("Content-Type", "application/json; charset=utf-8")
	peticion.Header.Set("Accept", "application/json")
	return peticion
}

func nuevoManejadorSeleccionLlamamientoHTTPPrueba(
	t *testing.T,
	ejecutor EjecutorSeleccionLlamamiento,
) http.Handler {
	t.Helper()
	manejador, err := NuevoManejadorSeleccionLlamamiento(ejecutor)
	if err != nil {
		t.Fatalf("construir manejador: %v", err)
	}
	return manejador
}

func TestManejadorSeleccionLlamamientoPublicaReciboMinimoAutenticado(
	t *testing.T,
) {
	t.Parallel()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		recibo: reciboSeleccionLlamamientoHTTPPrueba(),
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionSeleccionLlamamientoHTTPPrueba())
	if respuesta.Code != http.StatusOK {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
	esperado := `{"data":{"esquema":"` +
		EsquemaReciboSeleccionLlamamiento +
		`","estado":"confirmado","recibo_ref":` +
		`"recibo:llamamiento:http:001",` +
		`"confirmada_en":"2026-08-31T10:00:00.123Z"}}`
	if respuesta.Body.String() != esperado {
		t.Fatalf("respuesta inesperada:\n%s\n!=\n%s", respuesta.Body, esperado)
	}
	entrada, existe := ejecutor.ultima()
	if !existe || entrada.ClaveIdempotencia != claveSeleccionLlamamientoHTTPPrueba ||
		ejecutor.total() != 1 {
		t.Fatalf("delegacion no exacta: entrada=%#v total=%d", entrada, ejecutor.total())
	}
	for _, privado := range []string{
		"organizacion", "expediente", "correlacion", "llamamiento:privado",
		"auditoria", "evento", "evidencia", "hmac", "orden_seleccionado",
	} {
		if strings.Contains(strings.ToLower(respuesta.Body.String()), privado) {
			t.Fatalf("dato privado %q serializado: %s", privado, respuesta.Body)
		}
	}
}

func TestManejadorSeleccionLlamamientoReplayTerminalConservaSalidaPublica(
	t *testing.T,
) {
	t.Parallel()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		recibo: reciboSeleccionLlamamientoHTTPPrueba(),
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	var cuerpos [2]string
	for intento := range cuerpos {
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(
			respuesta,
			nuevaPeticionSeleccionLlamamientoHTTPPrueba(),
		)
		if respuesta.Code != http.StatusOK {
			t.Fatalf("intento %d: estado=%d", intento, respuesta.Code)
		}
		cuerpos[intento] = respuesta.Body.String()
	}
	if cuerpos[0] != cuerpos[1] || ejecutor.total() != 2 {
		t.Fatalf("replay público inestable: %#v llamadas=%d", cuerpos, ejecutor.total())
	}
}

func TestManejadorSeleccionLlamamientoClasificaEstadosNoReintentables(
	t *testing.T,
) {
	t.Parallel()
	casos := []error{
		application.ErrClaveSeleccionLlamamientoEnColision,
		application.ErrEjecucionSeleccionLlamamientoConcurrente,
		application.ErrEjecucionSeleccionLlamamientoIndeterminada,
		errors.Join(
			application.ErrEjecucionSeleccionLlamamientoIndeterminada,
			context.DeadlineExceeded,
		),
	}
	for indice, fallo := range casos {
		ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{err: fallo}
		manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
		respuesta := httptest.NewRecorder()
		manejador.ServeHTTP(
			respuesta,
			nuevaPeticionSeleccionLlamamientoHTTPPrueba(),
		)
		if respuesta.Code != http.StatusConflict ||
			!bytes.Contains(
				respuesta.Body.Bytes(),
				[]byte(`"codigo":"conflicto_no_reintentable"`),
			) ||
			respuesta.Header().Get("Retry-After") != "" || ejecutor.total() != 1 {
			t.Fatalf(
				"caso %d: estado=%d cuerpo=%s retry=%q llamadas=%d",
				indice,
				respuesta.Code,
				respuesta.Body,
				respuesta.Header().Get("Retry-After"),
				ejecutor.total(),
			)
		}
	}
}

func TestManejadorSeleccionLlamamientoNoFiltraErroresInternos(t *testing.T) {
	t.Parallel()
	ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{
		err: errors.New("postgres://persona:secreto@privado/rrhh"),
	}
	manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionSeleccionLlamamientoHTTPPrueba())
	if respuesta.Code != http.StatusServiceUnavailable {
		t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
	}
	for _, privado := range []string{"postgres", "persona", "secreto", "privado"} {
		if bytes.Contains(respuesta.Body.Bytes(), []byte(privado)) {
			t.Fatalf("error privado filtrado: %s", respuesta.Body)
		}
	}
	var salida struct {
		Error struct {
			Codigo    string `json:"codigo"`
			ClaveI18n string `json:"clave_i18n"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respuesta.Body.Bytes(), &salida); err != nil ||
		salida.Error.Codigo != "servicio_no_disponible" ||
		!strings.HasPrefix(
			salida.Error.ClaveI18n,
			"api.contratacion_temporal.seleccion_llamamiento.error.",
		) {
		t.Fatalf("error público inválido: %#v err=%v", salida, err)
	}
}

func TestManejadorSeleccionLlamamientoRechazaProyeccionNoPublicable(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre string
		mutar  func(*application.DatosReciboSeleccionLlamamientoParaAdaptador)
	}{
		{"salida cero", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			*r = application.DatosReciboSeleccionLlamamientoParaAdaptador{}
		}},
		{"recibo privado invalido", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.ReciboRef = "persona@example.invalid"
		}},
		{"instante no canonico", func(r *application.DatosReciboSeleccionLlamamientoParaAdaptador) {
			r.ConfirmadaEn = r.ConfirmadaEn.In(time.FixedZone("privada", 3600))
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			recibo := reciboSeleccionLlamamientoHTTPPrueba()
			caso.mutar(&recibo)
			ejecutor := &ejecutorSeleccionLlamamientoHTTPPrueba{recibo: recibo}
			manejador := nuevoManejadorSeleccionLlamamientoHTTPPrueba(t, ejecutor)
			respuesta := httptest.NewRecorder()
			manejador.ServeHTTP(
				respuesta,
				nuevaPeticionSeleccionLlamamientoHTTPPrueba(),
			)
			if respuesta.Code != http.StatusBadGateway ||
				!bytes.Contains(
					respuesta.Body.Bytes(),
					[]byte(`"codigo":"resultado_no_confiable"`),
				) {
				t.Fatalf("estado=%d cuerpo=%s", respuesta.Code, respuesta.Body)
			}
		})
	}
}

func TestManejadorSeleccionLlamamientoRechazaDependenciasNulas(t *testing.T) {
	t.Parallel()
	var nulo *ejecutorSeleccionLlamamientoHTTPPrueba
	for nombre, dependencia := range map[string]EjecutorSeleccionLlamamiento{
		"nulo":        nil,
		"nulo tipado": nulo,
	} {
		if manejador, err := NuevoManejadorSeleccionLlamamiento(dependencia); manejador != nil ||
			!errors.Is(err, ErrManejadorSeleccionLlamamientoInvalido) {
			t.Fatalf("%s aceptado: manejador=%#v err=%v", nombre, manejador, err)
		}
	}
	tipo := reflect.TypeOf(application.SolicitudSeleccionLlamamiento{})
	if tipo.NumField() != 1 || tipo.Field(0).Name != "ClaveIdempotencia" {
		t.Fatalf("entrada permite autoridad: %v", tipo)
	}
	metodo, existe := reflect.TypeOf((*EjecutorSeleccionLlamamiento)(nil)).Elem().MethodByName(
		"SeleccionarYLlamarParaAdaptador",
	)
	tipoProyeccion := reflect.TypeOf(application.DatosReciboSeleccionLlamamientoParaAdaptador{})
	if !existe || metodo.Type.NumOut() != 2 || metodo.Type.Out(0) != tipoProyeccion ||
		tipoProyeccion.NumField() != 2 || tipoProyeccion.Field(0).Name != "ReciboRef" ||
		tipoProyeccion.Field(1).Name != "ConfirmadaEn" {
		t.Fatalf("HTTP no exige la proyeccion de application: %v", metodo.Type)
	}
}
