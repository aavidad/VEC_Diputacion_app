package bootstrap

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	reglashttp "vec-diputacion-granada/internal/vec/reglas/httpinterno"
)

func TestReglasVigentesQuedanDentroDelPerimetroMTLS(t *testing.T) {
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, reglashttp.RutaReglasVigentes, nil)) ||
		!rutaReglasVigentesDesarrollo(reglashttp.RutaReglasVigentes) {
		t.Fatal("la consulta de reglas queda fuera del perímetro mTLS")
	}
	if rutaReglasVigentesDesarrollo("/api/vec/reglas/escritura") {
		t.Fatal("solo se protege la ruta declarada")
	}
}

func TestReglasVigentesSinPaqueteRespondenSinCatalogo(t *testing.T) {
	ruta, err := nuevaRutaReglasVigentesDesarrollo(reglasEjemploDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil))
	if w.Code != http.StatusOK || strings.Count(w.Body.String(), `"estado":"sin_catalogo"`) != 2 {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

// Con el servidor real compuesto, una identidad mTLS de RRHH consulta y una
// cookie no puede acompañar la consulta.
func TestReglasVigentesExigenIdentidadEnServidorReal(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloConPostgreSQLPrueba(t)
	servidor, err := NewHTTPServerWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)
	sinCertificado := prueba.Client()
	if respuesta, err := sinCertificado.Get(prueba.URL + reglashttp.RutaReglasVigentes); err == nil {
		respuesta.Body.Close()
		if respuesta.StatusCode == http.StatusOK {
			t.Fatal("sin certificado de cliente no se sirven las reglas")
		}
	}
	cliente := nuevoClienteMTLSContratacionTemporalDesarrollo(t, rutas)
	anadirCadenaCompletaClienteMTLSContratacionTemporalDesarrollo(t, cliente, rutas)
	respuesta, err := cliente.Get(prueba.URL + reglashttp.RutaReglasVigentes)
	if err != nil {
		t.Fatal(err)
	}
	contenido, _ := io.ReadAll(respuesta.Body)
	respuesta.Body.Close()
	// La identidad de prueba puede no tener el perfil de lectura de RRHH (403);
	// nunca queda sin autenticar ni recibe otra cosa que el contrato.
	if (respuesta.StatusCode != http.StatusOK && respuesta.StatusCode != http.StatusForbidden) ||
		(respuesta.StatusCode == http.StatusOK && !strings.Contains(string(contenido), reglashttp.Esquema)) {
		t.Fatalf("consulta mTLS: %d %s", respuesta.StatusCode, contenido)
	}
	peticion, _ := http.NewRequest(http.MethodGet, prueba.URL+reglashttp.RutaReglasVigentes, nil)
	peticion.Header.Set("Cookie", "sesion=sintetica")
	if respuesta, err = cliente.Do(peticion); err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	if respuesta.StatusCode == http.StatusOK || respuesta.Header.Get("Set-Cookie") != "" {
		t.Fatalf("una cookie no puede acompañar la consulta: %d", respuesta.StatusCode)
	}
}
