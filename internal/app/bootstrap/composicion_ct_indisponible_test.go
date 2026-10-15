package bootstrap

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestArranqueSinCTConservaSesionPortalYSalud(t *testing.T) {
	cfg, rutas := generarMaterialDesarrolloPrueba(t)
	var registro bytes.Buffer
	servidor, composicion, err := NewHTTPServerDesarrolloWithConfig(cfg, &registro)
	if err != nil || servidor == nil || composicion == nil {
		t.Fatalf("arranque sin PostgreSQL de CT: servidor=%v error=%v", servidor, err)
	}
	prueba := httptest.NewUnstartedServer(servidor.Handler)
	prueba.TLS = servidor.TLSConfig.Clone()
	prueba.StartTLS()
	t.Cleanup(prueba.Close)

	certificado, err := tls.LoadX509KeyPair(rutas.ClientCertificate, rutas.ClientPrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := os.ReadFile(rutas.CACertificate)
	if err != nil {
		t.Fatal(err)
	}
	raices := x509.NewCertPool()
	if !raices.AppendCertsFromPEM(ca) {
		t.Fatal("CA de prueba no válida")
	}
	cliente := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		Certificates: []tls.Certificate{certificado}, RootCAs: raices,
		ServerName: "localhost", MinVersion: tls.VersionTLS13,
	}}}
	sinCertificado := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: raices, ServerName: "localhost", MinVersion: tls.VersionTLS13,
	}}}
	if respuesta, err := sinCertificado.Get(prueba.URL + "/api/vec/session"); err == nil {
		respuesta.Body.Close()
		t.Fatal("sesión aceptada sin certificado mTLS")
	}
	for _, caso := range []struct {
		ruta   string
		estado int
		marca  string
	}{
		{"/livez", http.StatusOK, `"status":"ok"`},
		{"/healthz", http.StatusServiceUnavailable, `"status":"unavailable"`},
		{"/portal-empleado/", http.StatusOK, ""},
		{"/api/vec/session", http.StatusOK, `"auth_assurance":"alto"`},
		{"/api/publico/bolsa/convocatorias", http.StatusOK, `"esquema":"vec.bolsa.publico.convocatorias.v1"`},
		{"/api/vec/contratacion-temporal/expedientes/consultas", http.StatusServiceUnavailable, `"codigo":"servicio_no_disponible"`},
		{"/api/vec/bolsa/bolsas", http.StatusServiceUnavailable, `"codigo":"servicio_no_disponible"`},
		{"/api/vec/dietas/comisiones", http.StatusServiceUnavailable, `"codigo":"servicio_no_disponible"`},
	} {
		respuesta, err := cliente.Get(prueba.URL + caso.ruta)
		if err != nil {
			t.Fatalf("GET %s: %v", caso.ruta, err)
		}
		cuerpo, err := io.ReadAll(respuesta.Body)
		respuesta.Body.Close()
		if err != nil || respuesta.StatusCode != caso.estado || !strings.Contains(string(cuerpo), caso.marca) {
			t.Fatalf("GET %s: estado=%d cuerpo=%q error=%v", caso.ruta, respuesta.StatusCode, cuerpo, err)
		}
	}
	if strings.Contains(registro.String(), "postgres://") || strings.Contains(registro.String(), cfg.DevelopmentMaterialDir) {
		t.Fatal("registro de arranque contiene configuración privada")
	}
}

func TestCierreCTDelimitaPrefijos(t *testing.T) {
	for _, ruta := range []string{
		"/api/vec/contratacion-temporal", "/api/vec/contratacion-temporal/expedientes",
		"/api/vec/bolsa/bolsas", "/api/vec/dietas/comisiones/123",
		"/api/vec/personal/relaciones-dietas", "/api/vec/usuarios/mis-preferencias",
	} {
		if !rutaDependienteContratacionTemporal(ruta) {
			t.Fatalf("ruta dependiente abierta: %s", ruta)
		}
	}
	for _, ruta := range []string{
		"/api/vec/session", "/api/vec/modules", "/api/vec/personal/categories",
		"/api/vec/dietas/road-route", "/api/vec/contratacion-temporal-ajena",
	} {
		if rutaDependienteContratacionTemporal(ruta) {
			t.Fatalf("ruta independiente cerrada: %s", ruta)
		}
	}
}
