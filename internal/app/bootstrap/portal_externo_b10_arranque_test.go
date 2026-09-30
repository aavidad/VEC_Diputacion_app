package bootstrap

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/app/separacionportales"
)

func TestProcesoExternoComponeConsultasSinActivarB10(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	m := generarMaterialPortalExternoPrueba(t)
	m.cfg.PersonalCatalogPath = "memory"
	var registro bytes.Buffer
	servidor, _, err := NewHTTPServerDesarrolloWithConfig(m.cfg, &registro)
	if err != nil {
		t.Fatalf("arranque externo: %v", err)
	}
	if servidor == nil {
		t.Fatal("servidor externo ausente")
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)
	candidato := m.cliente(t, "candidato")
	for _, caso := range []struct {
		ruta   string
		estado int
	}{
		{"/api/publico/bolsa/convocatorias", http.StatusOK},
		{"/api/publico/bolsa/bolsas", http.StatusNotFound},
		{"/api/vec/contratacion-temporal/expedientes", http.StatusNotFound},
	} {
		respuesta, err := candidato.Get(prueba.URL + caso.ruta)
		if err != nil {
			t.Fatal(err)
		}
		_, err = io.Copy(io.Discard, respuesta.Body)
		respuesta.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if respuesta.StatusCode != caso.estado {
			t.Fatalf("ruta %s = %d; esperado %d", caso.ruta, respuesta.StatusCode, caso.estado)
		}
	}
}

func TestProcesoExternoDeniegaAntesDeLeerMaterialConCredencialesImplicitas(t *testing.T) {
	vaciarConexionesDelEntorno(t)
	m := generarMaterialPortalExternoPrueba(t)
	// El fichero ni siquiera existe: la frontera debe rechazar la variable
	// antes de que PostgreSQL intente resolverla o cargar material.
	t.Setenv("PGPASSFILE", "credenciales-sinteticas-no-existentes")
	m.cfg.DevelopmentMaterialDir = t.TempDir()
	var registro bytes.Buffer
	servidor, seguridad, err := NewHTTPServerDesarrolloWithConfig(m.cfg, &registro)
	if !errors.Is(err, separacionportales.ErrSeparacionPortales) || servidor != nil || seguridad != nil {
		t.Fatalf("credencial implicita no denegada primero: servidor=%v seguridad=%v error=%v", servidor, seguridad, err)
	}
	if registro.Len() != 0 {
		t.Fatal("el proceso registro un arranque tras la denegacion")
	}
}
