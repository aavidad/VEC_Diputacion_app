package telemetria

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/ports"
)

func lineas(t *testing.T, b *bytes.Buffer) []map[string]any {
	t.Helper()
	var salida []map[string]any
	for _, l := range strings.Split(strings.TrimSpace(b.String()), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("linea no JSON: %q", l)
		}
		salida = append(salida, m)
	}
	return salida
}

func opciones(b *bytes.Buffer) Opciones {
	return Opciones{Destino: b, Servicio: "vec-server", Superficie: "interno", Entorno: "desarrollo", Lenta: time.Hour}
}

func TestAccesoConservaFasesDeExitoYFalloSinErroresNiDatos(t *testing.T) {
	var b bytes.Buffer
	h := Middleware(opciones(&b), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RegistrarFase(r.Context(), FaseIdentidad, 2*time.Millisecond, nil)
		RegistrarFase(r.Context(), FaseSesion, 3*time.Millisecond, errors.New("persona privada 12345678Z"))
		RegistrarFase(r.Context(), FaseContexto, 4*time.Millisecond, context.Canceled)
		RegistrarFase(r.Context(), Fase("persona privada"), time.Second, errors.New("dato privado"))
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/vec/usuarios/mis-preferencias", nil))
	l := lineas(t, &b)[0]
	fases, ok := l["vec.fases"].([]any)
	if !ok || len(fases) != 3 {
		t.Fatalf("fases registradas = %v", l["vec.fases"])
	}
	for i, nombre := range []string{"identidad", "sesion", "contexto"} {
		fase := fases[i].(map[string]any)
		if fase["nombre"] != nombre || fase["n"] != float64(1) || fase["total"].(float64) <= 0 {
			t.Fatalf("fase %d = %v", i, fase)
		}
	}
	if fases[1].(map[string]any)["errores"] != float64(1) || fases[2].(map[string]any)["canceladas"] != float64(1) ||
		strings.Contains(b.String(), "privada") || strings.Contains(b.String(), "12345678Z") {
		t.Fatalf("fases o privacidad incorrectas: %s", b.String())
	}
}

func TestAccesoEscribeRutaEstadoYDuracionSinValores(t *testing.T) {
	var b bytes.Buffer
	h := Middleware(opciones(&b), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hola"))
	}))
	r := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/9f1c2a7e-0000/participaciones?dni=12345678Z", strings.NewReader("secreto"))
	r.Header.Set("Authorization", "Bearer secreto")
	h.ServeHTTP(httptest.NewRecorder(), r)

	l := lineas(t, &b)[0]
	for k, v := range map[string]any{
		"level": "INFO", "msg": "http.server.request", "service.name": "vec-server", "vec.superficie": "interno",
		"deployment.environment.name": "desarrollo", "http.request.method": "GET",
		"url.path": "/api/vec/bolsa/{valor}/participaciones", "http.response.status_code": float64(200), "http.response.body.size": float64(4),
	} {
		if l[k] != v {
			t.Errorf("%s = %v, se esperaba %v", k, l[k], v)
		}
	}
	if _, ok := l["http.server.request.duration"].(float64); !ok || len(l["vec.correlacion"].(string)) != 32 || l["service.version"] == "" || l["http.route"] != nil {
		t.Errorf("linea = %v", l)
	}
	for _, prohibido := range []string{"9f1c2a7e", "12345678Z", "dni", "secreto", "Bearer"} {
		if strings.Contains(b.String(), prohibido) {
			t.Errorf("el registro contiene %q", prohibido)
		}
	}
}

func TestAccesoNoEmiteResumenDeConsultasEnPeticionRapidaCorrecta(t *testing.T) {
	var b bytes.Buffer
	h := Middleware(opciones(&b), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr := trazador{}
		q := tr.TraceQueryStart(r.Context(), nil, pgx.TraceQueryStartData{SQL: "SELECT vec_prueba.consultar($1)", Args: []any{"dato privado"}})
		tr.TraceQueryEnd(q, nil, pgx.TraceQueryEndData{})
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	l := lineas(t, &b)[0]
	if l["vec.bd.consultas"] != float64(1) || l["vec.bd.operaciones"] != nil || strings.Contains(b.String(), "dato privado") {
		t.Errorf("petición rápida = %v", l)
	}
}

func TestAccesoEmiteResumenEnErrorCuatrocientos(t *testing.T) {
	var b bytes.Buffer
	h := Middleware(opciones(&b), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr := trazador{}
		q := tr.TraceQueryStart(r.Context(), nil, pgx.TraceQueryStartData{SQL: "SELECT vec_persona_juan.perez($1)"})
		tr.TraceQueryEnd(q, nil, pgx.TraceQueryEndData{})
		w.WriteHeader(http.StatusForbidden)
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x/identidad-opaca", nil))
	l := lineas(t, &b)[0]
	resumen, ok := l["vec.bd.operaciones"].([]any)
	if !ok || len(resumen) != 1 || l["url.path"] != "{oculto}" || l["vec.lenta"] != nil ||
		l["vec.bd.operaciones_desconocidas"] != float64(1) || strings.Contains(b.String(), "juan") {
		t.Errorf("error cuatrocientos = %v", l)
	}
}

func TestAccesoNormalizaCaminoYOcultaLos4xxSinPlantilla(t *testing.T) {
	var b bytes.Buffer
	h := Middleware(opciones(&b), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/"):
			w.WriteHeader(http.StatusServiceUnavailable)
		case strings.HasPrefix(r.URL.Path, "/static/"):
		default:
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	for _, camino := range []string{"/api/v2/expedientes/recibo:408fda57/Juan", "/static/app.v123.js", "/portal-empleado/juanperez"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, camino, nil))
	}
	l := lineas(t, &b)
	if l[0]["url.path"] != "/api/v2/expedientes/{valor}/{valor}" || l[0]["level"] != "ERROR" || l[0]["error.type"] != "503" {
		t.Errorf("5xx = %v", l[0])
	}
	if l[1]["url.path"] != "/static/{valor}" { // app.v123.js
		t.Errorf("estatico = %v", l[1]["url.path"])
	}
	if l[2]["url.path"] != "{oculto}" || strings.Contains(b.String(), "juan") {
		t.Errorf("4xx = %v", l[2]["url.path"])
	}
}

func TestAccesoLentaPanicoYCorrelacionDeLaSupervision(t *testing.T) {
	var b bytes.Buffer
	o := opciones(&b)
	o.Lenta = time.Nanosecond
	var vista string
	h := Middleware(o, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vista, _ = ports.CorrelacionIncidenciasPeticion(r.Context())
		time.Sleep(time.Millisecond)
	}))
	ctx, _ := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	esperada, _ := ports.CorrelacionIncidenciasPeticion(ctx)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(ctx))

	panico := Middleware(o, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("datos de Juan") }))
	func() {
		defer func() {
			if recover() == nil {
				t.Error("el panico no siguio hacia la supervision")
			}
		}()
		panico.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))
	}()
	l := lineas(t, &b)
	if l[0]["level"] != "WARN" || l[0]["vec.lenta"] != true || l[0]["vec.correlacion"] != esperada || vista != esperada {
		t.Errorf("lenta = %v", l[0])
	}
	if l[1]["level"] != "ERROR" || l[1]["http.response.status_code"] != float64(500) || l[1]["vec.interrumpida"] != true || strings.Contains(b.String(), "Juan") {
		t.Errorf("panico = %v", l[1])
	}
}

func TestEntornoVersionYUmbral(t *testing.T) {
	for _, c := range []struct{ declarado, perfil, want string }{
		{"presentacion", "desarrollo", "presentacion"}, {"10.1.2.3", "", "desconocido"},
		{"", "desarrollo", "desarrollo"}, {"", "presentacion_rrhh", "presentacion"}, {"", "", "desconocido"},
	} {
		if got := Entorno(c.declarado, c.perfil); got != c.want {
			t.Errorf("Entorno(%q, %q) = %q", c.declarado, c.perfil, got)
		}
	}
	anterior := Revision
	t.Cleanup(func() { Revision = anterior })
	Revision = "aae093f6f0aa"
	if Version() != "aae093f6f0aa" {
		t.Errorf("Version() = %q", Version())
	}
	entorno := map[string]string{"VEC_TELEMETRIA_LENTA_MS": "500"}
	if UmbralLenta(func(k string) string { return entorno[k] }) != 500*time.Millisecond {
		t.Error("umbral no leido")
	}
	if UmbralLenta(func(string) string { return "-1" }) != 300*time.Millisecond {
		t.Error("umbral no valido aceptado")
	}
}

func TestTramosFijos(t *testing.T) {
	for tramo, fijo := range map[string]bool{
		"portal-empleado": true, "mis_datos": true, "v2": true, "bolsa": true,
		"app.js": false, "index.html": false, "Juan": false, "12345678Z": false, "recibo:1": false, "peña": false,
	} {
		if esTramoFijo(tramo) != fijo {
			t.Errorf("esTramoFijo(%q) = %t", tramo, !fijo)
		}
	}
}
