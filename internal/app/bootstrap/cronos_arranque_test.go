package bootstrap

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	appserver "vec-diputacion-granada/internal/app/server"
)

func TestPlazoCronosNoHeredaDeadlineAgotadoDeCT(t *testing.T) {
	ctxCT, cancelarCT := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelarCT()
	if !errors.Is(ctxCT.Err(), context.DeadlineExceeded) {
		t.Fatal("el plazo de CT debía estar agotado")
	}
	ctxCronos, cancelarCronos := contextoPublicacionCronos(ctxCT)
	if ctxCronos.Err() != nil {
		t.Fatalf("Cronos heredó la cancelación de CT: %v", ctxCronos.Err())
	}
	limite, existe := ctxCronos.Deadline()
	if !existe || time.Until(limite) <= 0 {
		t.Fatal("Cronos no tiene plazo propio vigente")
	}
	cancelarCronos()
	if !errors.Is(ctxCronos.Err(), context.Canceled) {
		t.Fatal("el plazo propio de Cronos no se pudo cancelar")
	}
}

func TestPreparacionCronosApagadoNoLeeMaterialNiSondea(t *testing.T) {
	cfg := config.Config{DevelopmentMaterialDir: filepath.Join(t.TempDir(), "ausente")}
	if solicitado, err := prepararCronosAntesDeCT(cfg, nil); solicitado || err != nil {
		t.Fatalf("Cronos apagado: solicitado=%v error=%v", solicitado, err)
	}
}

func TestPreparacionCronosAislaSelectoresYMaterialPrivado(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	identidad, ok := composicion.identidad.(*resolvedorIdentidadDesarrollo)
	if !ok {
		t.Fatal("identidad de desarrollo ausente")
	}
	casos := []struct {
		nombre, empleado, resolucion, avisos, componente string
	}{
		{"selector principal", "si", "", "", "cronos_selector"},
		{"selector subordinado", "false", "true", "", "cronos_selector"},
		{"selector de avisos", "true", "", "si", "cronos_selector"},
		{"fichero ausente", "true", "", "", "cronos_material_privado"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			actual := cfg
			actual.CronosEmpleadoEnabled = caso.empleado
			actual.CronosResolucionEnabled = caso.resolucion
			actual.CronosNotificacionesEnabled = caso.avisos
			solicitado, fallo := prepararCronosAntesDeCT(actual, identidad)
			if !solicitado || fallo == nil || ComponenteFalloArranque(fallo) != caso.componente {
				t.Fatalf("aislamiento = (%v, %v), componente %s", solicitado, fallo, ComponenteFalloArranque(fallo))
			}
		})
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "cronos-empleado.json")
	if err := os.WriteFile(ruta, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg.CronosEmpleadoEnabled = "true"
	solicitado, fallo := prepararCronosAntesDeCT(cfg, identidad)
	if !solicitado || ComponenteFalloArranque(fallo) != "cronos_material_privado" {
		t.Fatalf("fichero incompleto = (%v, %v)", solicitado, fallo)
	}
	seleccion, err := seleccionMaterialCTDesarrolloDesdeConfig(configuracionCTSinCronos(cfg))
	if err != nil || seleccion.cronos || seleccion.cronosResolucion || seleccion.cronosAvisos {
		t.Fatalf("CT conserva material Cronos fallido: %+v, %v", seleccion, err)
	}
	for _, descriptor := range descriptoresMaterialSeleccionadosCTDesarrollo(seleccion) {
		if strings.HasPrefix(descriptor.Dominio, "vec.cronos.") {
			t.Fatalf("CT intenta publicar audiencia Cronos fallida: %s", descriptor.Audiencia)
		}
	}
}

func TestRaizCronosNoDisponibleConservaOtrosCaminos(t *testing.T) {
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/vec/session":
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	})
	servidor, err := appserver.NewHTTPServer(config.Config{Address: "127.0.0.1:0", APIBasePath: "/api"}, componerRaizConCronosIndisponible(base))
	if err != nil {
		t.Fatal(err)
	}
	raiz := servidor.Handler
	for _, ruta := range []string{"/api/interna/cronos", "/api/interna/cronos/saldo/propio", "/api/interna/cronos/desconocida"} {
		respuesta := httptest.NewRecorder()
		peticion := httptest.NewRequest(http.MethodGet, ruta, nil)
		peticion.RemoteAddr = "127.0.0.1:1234"
		raiz.ServeHTTP(respuesta, peticion)
		if respuesta.Code != http.StatusServiceUnavailable || !strings.Contains(respuesta.Body.String(), `"error":"no_disponible"`) || respuesta.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("ruta Cronos %s: estado=%d cuerpo=%q", ruta, respuesta.Code, respuesta.Body.String())
		}
	}
	for _, caso := range []struct {
		ruta   string
		estado int
	}{{"/livez", http.StatusOK}, {"/portal-empleado/", http.StatusOK}, {"/api/vec/session", http.StatusNoContent}} {
		respuesta := httptest.NewRecorder()
		peticion := httptest.NewRequest(http.MethodGet, caso.ruta, nil)
		peticion.RemoteAddr = "127.0.0.1:1234"
		raiz.ServeHTTP(respuesta, peticion)
		if respuesta.Code != caso.estado {
			t.Fatalf("ruta ajena %s: estado=%d", caso.ruta, respuesta.Code)
		}
	}
}

func TestServidorDesarrolloCronosIncompletoNoDetieneLaRaiz(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	cfg.CronosEmpleadoEnabled = "true"
	servidor, _, err := NewHTTPServerDesarrolloWithConfig(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := servidor.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Error(err)
		}
	})
	for _, caso := range []struct {
		ruta   string
		estado int
	}{
		{"/livez", http.StatusOK},
		{"/portal-empleado/", http.StatusOK},
		{"/api/interna/cronos/saldo/propio", http.StatusServiceUnavailable},
	} {
		respuesta := httptest.NewRecorder()
		peticion := httptest.NewRequest(http.MethodGet, caso.ruta, nil)
		peticion.RemoteAddr = "127.0.0.1:1234"
		servidor.Handler.ServeHTTP(respuesta, peticion)
		if respuesta.Code != caso.estado {
			t.Fatalf("ruta %s: estado=%d, esperado=%d", caso.ruta, respuesta.Code, caso.estado)
		}
	}
}
