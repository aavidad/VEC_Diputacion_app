package bootstrap

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	calendarioshttp "vec-diputacion-granada/internal/modules/calendarios/adapters/httpinterno"
)

func TestCalendariosQuedanDentroDelPerimetroMTLS(t *testing.T) {
	for _, ruta := range calendarioshttp.Rutas() {
		if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, ruta+"?anio=2026", nil)) || !rutaCalendariosDesarrollo(ruta) {
			t.Fatalf("%s fuera del perímetro mTLS", ruta)
		}
	}
	if rutaCalendariosDesarrollo("/api/vec/calendarios/escritura") {
		t.Fatal("solo se protegen las rutas declaradas")
	}
}

func TestCalendariosSinConexionRespondenNoDisponible(t *testing.T) {
	rutas, cerrar, err := nuevasRutasCalendariosDesarrollo(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer cerrar()
	if len(rutas) != 3 {
		t.Fatalf("rutas: %d", len(rutas))
	}
	for _, r := range rutas {
		w := httptest.NewRecorder()
		r.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, r.Ruta+"?anio=2026", nil))
		if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "servicio_no_disponible") {
			t.Fatalf("%s: %d %s", r.Ruta, w.Code, w.Body.String())
		}
	}
}

func TestCalendariosConConexionExigenDobleLlaveYDSNValido(t *testing.T) {
	t.Setenv(config.EnvCalendariosDatabaseURL, "postgres://lector@127.0.0.1:1/vec?sslmode=disable")
	if _, _, err := nuevasRutasCalendariosDesarrollo(config.Load()); !errors.Is(err, errCalendariosDesarrolloNoDisponible) {
		t.Fatalf("sin perfil de desarrollo con doble llave debe fallar: %v", err)
	}
	if _, err := abrirPoolCalendariosDesarrollo(t.Context(), "postgres://lector@db.ejemplo.invalid:5432/vec?sslmode=disable"); err == nil {
		t.Fatal("un destino remoto sin TLS verificado no se admite")
	}
}

// Con el PostgreSQL de Contratación configurado, compone el servidor real y
// comprueba que sin certificado de cliente no se sirve el calendario.
func TestCalendariosExigenIdentidadEnServidorReal(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)
	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	respuesta, err := cliente.Get(prueba.URL + calendarioshttp.RutaCentros + "?anio=2026")
	if err != nil {
		t.Fatal(err)
	}
	contenido, _ := io.ReadAll(respuesta.Body)
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusOK && respuesta.StatusCode != http.StatusServiceUnavailable && respuesta.StatusCode != http.StatusForbidden {
		t.Fatalf("consulta mTLS: %d %s", respuesta.StatusCode, contenido)
	}
	peticion, _ := http.NewRequest(http.MethodGet, prueba.URL+calendarioshttp.RutaCentros+"?anio=2026", nil)
	peticion.Header.Set("Cookie", "sesion=sintetica")
	if respuesta, err = cliente.Do(peticion); err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	if respuesta.StatusCode == http.StatusOK || respuesta.Header.Get("Set-Cookie") != "" {
		t.Fatalf("una cookie no puede acompañar la consulta: %d", respuesta.StatusCode)
	}
}
