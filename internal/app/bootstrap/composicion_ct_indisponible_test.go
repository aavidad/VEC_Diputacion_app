package bootstrap

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestReglasCTInexistentesNoDetienenPortal(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	cfg.ReglasEjemplo.CTSourcePath = filepath.Join(directorioTemporalFueraDeGitPrueba(t), "ct-inexistente.json")
	comprobarArranqueSinCapacidadPrevia(t, cfg, "reglas_ejemplo")
}

func TestPreferenciasInvalidasNoDetienenPortal(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	t.Setenv(envUsuariosPreferenciasDesarrollo, "true")
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", nombreConfiguracionPreferencias(core.SuperficieAutenticacionInternaCorporativaV1))
	if err := os.WriteFile(ruta, []byte(`{"version":`), 0o600); err != nil {
		t.Fatal(err)
	}
	comprobarArranqueSinCapacidadPrevia(t, cfg, "identidades_preferencias")
}

func comprobarArranqueSinCapacidadPrevia(t *testing.T, cfg config.Config, componente string) {
	t.Helper()
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	defer slog.SetDefault(anterior)
	servidor, composicion, err := NewHTTPServerDesarrolloWithConfig(cfg, io.Discard)
	if err != nil || servidor == nil || composicion == nil || servidor.Handler == nil ||
		servidor.TLSConfig == nil || servidor.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatalf("configuración de %s detuvo el portal o rebajó mTLS: servidor=%v error=%v", componente, servidor, err)
	}
	if !strings.Contains(registro.String(), "componente="+componente) ||
		strings.Contains(registro.String(), cfg.DevelopmentMaterialDir) ||
		strings.Contains(registro.String(), cfg.ReglasEjemplo.CTSourcePath) && cfg.ReglasEjemplo.CTSourcePath != "" {
		t.Fatalf("causa de %s ausente o insegura: %q", componente, registro.String())
	}
}

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
		{"/api/vec/auditoria/consultas", http.StatusServiceUnavailable, `"codigo":"servicio_no_disponible"`},
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
		"/api/vec/auditoria/consultas", "/api/vec/auditoria/opciones",
		"/api/vec/calendarios/centros", "/api/vec/reglas/vigentes",
		"/api/interna/personal/mi-ficha/servicios/historia",
		"/api/vec/personal/relaciones-dietas", "/api/vec/usuarios/mis-preferencias",
	} {
		if !rutaDependienteContratacionTemporal(ruta) {
			t.Fatalf("ruta dependiente abierta: %s", ruta)
		}
	}
	for _, ruta := range []string{
		"/api/vec/session", "/api/vec/modules", "/api/vec/personal/categories",
		"/api/publico/bolsa/convocatorias", "/api/vec/dietas/road-route",
		"/api/vec/auditoria-ajena", "/api/vec/contratacion-temporal-ajena",
	} {
		if rutaDependienteContratacionTemporal(ruta) {
			t.Fatalf("ruta independiente cerrada: %s", ruta)
		}
	}
}

type escritorFallidoContratacionTemporalPrueba struct{ cabecera http.Header }

func (e *escritorFallidoContratacionTemporalPrueba) Header() http.Header { return e.cabecera }
func (*escritorFallidoContratacionTemporalPrueba) WriteHeader(int)       {}
func (*escritorFallidoContratacionTemporalPrueba) Write([]byte) (int, error) {
	return 0, errors.New("postgres://usuario:clave@host/base")
}

func TestRespuestaCTInterrumpidaRegistraSoloCausaCerrada(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	defer slog.SetDefault(anterior)
	escritor := &escritorFallidoContratacionTemporalPrueba{cabecera: make(http.Header)}
	responderContratacionTemporalNoDisponible(escritor, httptest.NewRequest(http.MethodGet, "/api/vec/auditoria/consultas", nil))
	if !strings.Contains(registro.String(), "escritura_respuesta_no_disponible") ||
		strings.Contains(registro.String(), "postgres://") ||
		escritor.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("registro de escritura inseguro o ausente: %q", registro.String())
	}
}
