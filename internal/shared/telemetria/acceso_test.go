package telemetria

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestAccesoEscribeRutaEstadoYDuracionSinValores(t *testing.T) {
	var b bytes.Buffer
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/vec/bolsa/{bolsa}/participaciones", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hola"))
	})
	h := Middleware(opciones(&b), mux)
	r := httptest.NewRequest(http.MethodGet, "/api/vec/bolsa/9f1c2a7e-0000/participaciones?dni=12345678Z", strings.NewReader("secreto"))
	r.Header.Set("Authorization", "Bearer secreto")
	h.ServeHTTP(httptest.NewRecorder(), r)

	l := lineas(t, &b)[0]
	for k, v := range map[string]any{
		"level": "INFO", "msg": "peticion", "servicio": "vec-server", "superficie": "interno", "entorno": "desarrollo",
		"metodo": "GET", "ruta": "/api/vec/bolsa/{bolsa}/participaciones", "estado": float64(200), "bytes": float64(4),
	} {
		if l[k] != v {
			t.Errorf("%s = %v, se esperaba %v", k, l[k], v)
		}
	}
	if _, ok := l["duracion_ms"].(float64); !ok || len(l["correlacion"].(string)) != 32 || l["version"] == "" {
		t.Errorf("linea = %v", l)
	}
	for _, prohibido := range []string{"9f1c2a7e", "12345678Z", "dni", "secreto", "Bearer"} {
		if strings.Contains(b.String(), prohibido) {
			t.Errorf("el registro contiene %q", prohibido)
		}
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
	if l[0]["ruta"] != "/api/v2/expedientes/{valor}/{valor}" || l[0]["level"] != "ERROR" {
		t.Errorf("5xx = %v %v", l[0]["ruta"], l[0]["level"])
	}
	if l[1]["ruta"] != "/static/{valor}" {
		t.Errorf("estatico = %v", l[1]["ruta"])
	}
	if l[2]["ruta"] != "{sin_plantilla}" || strings.Contains(b.String(), "juan") {
		t.Errorf("4xx = %v", l[2]["ruta"])
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
	if l[0]["level"] != "WARN" || l[0]["lenta"] != true || l[0]["correlacion"] != esperada || vista != esperada {
		t.Errorf("lenta = %v", l[0])
	}
	if l[1]["level"] != "ERROR" || l[1]["estado"] != float64(500) || l[1]["interrumpida"] != true || strings.Contains(b.String(), "Juan") {
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
