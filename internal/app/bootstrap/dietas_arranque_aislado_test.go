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
	"strings"
	"testing"

	"vec-diputacion-granada/config"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestDietasIncompletaConservaShellYDeniegaRutas(t *testing.T) {
	base, rutas := generarMaterialDesarrolloPrueba(t)
	seguridad, err := NuevaComposicionSeguridadDesarrollo(base, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer seguridad.derivadorIdempotencia.borrar()
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
		t.Fatal("CA local inválida")
	}
	for _, caso := range []struct {
		nombre     string
		configurar func(*config.Config)
		causa      string
	}{
		{"sin configuración", func(*config.Config) {}, ""},
		{"selector incompleto", func(c *config.Config) { c.DietasBorradoresEnabled = "true" }, "comisiones"},
		{"OSRM parcial", func(c *config.Config) { c.OSRMBaseURL = "http://127.0.0.1:5000" }, "cartografia"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			cfg := base
			caso.configurar(&cfg)
			var registro bytes.Buffer
			anterior := slog.Default()
			slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
			defer slog.SetDefault(anterior)
			montaje := componerDietasAisladasDesarrollo(cfg, seguridad.identidad, seguridad.derivadorIdempotencia, materialDietasDesdeCTDesarrollo{})
			defer montaje.cerrarRutas()
			if montaje.rutas != nil || montaje.comisiones != nil ||
				(caso.causa != "" && !strings.Contains(registro.String(), "componente="+caso.causa)) {
				t.Fatalf("capacidad abierta o fallo sin diagnóstico: %s", registro.String())
			}
			if caso.causa == "" && registro.Len() != 0 {
				t.Fatalf("Dietas apagada registró fallo: %s", registro.String())
			}
			if caso.causa == "cartografia" && (montaje.cfgShell.OSRMBaseURL != "" || cfg.OSRMBaseURL == "") {
				t.Fatal("la configuración original de OSRM se alteró o el shell la conservó incompleta")
			}
			shell, err := newVECShellAPICompuestaConIdentidadYRutas(
				montaje.cfgShell, vecports.EmisorIncidenciasTecnicasNulo{}, seguridad.identidad, nil,
				nil, nil, nil, montaje.rutas,
			)
			if err != nil {
				t.Fatalf("shell común no compuesto: %v", err)
			}
			shell = denegarDietasNoDisponibles(shell)
			servidor := httptest.NewUnstartedServer(shell)
			servidor.TLS = seguridad.tls.Clone()
			servidor.StartTLS()
			defer servidor.Close()
			transporte := &http.Transport{TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{certificado}, RootCAs: raices,
				ServerName: "localhost", MinVersion: tls.VersionTLS13,
			}}
			defer transporte.CloseIdleConnections()
			cliente := &http.Client{Transport: transporte}
			for _, peticion := range []struct {
				metodo, ruta string
				estado       int
			}{
				{http.MethodGet, "/api/vec/session", http.StatusOK},
				{http.MethodGet, "/api/vec/dietas/route-catalog", http.StatusServiceUnavailable},
				{http.MethodPost, "/api/vec/dietas/road-route", http.StatusServiceUnavailable},
				{http.MethodGet, "/api/vec/dietas/comisiones", http.StatusServiceUnavailable},
				{http.MethodGet, "/api/vec/personal/relaciones-dietas", http.StatusServiceUnavailable},
			} {
				req, err := http.NewRequest(peticion.metodo, servidor.URL+peticion.ruta, nil)
				if err != nil {
					t.Fatal(err)
				}
				resp, err := cliente.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode != peticion.estado {
					t.Fatalf("%s %s: %d, esperado %d", peticion.metodo, peticion.ruta, resp.StatusCode, peticion.estado)
				}
			}
		})
	}
}

func TestFalloDietasNoRegistraCausaLibre(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	defer slog.SetDefault(anterior)
	registrarFalloDietasArranque("material", errors.New("postgres://login:secreto@local/privado"))
	if strings.Contains(registro.String(), "secreto") || strings.Contains(registro.String(), "postgres://") ||
		!strings.Contains(registro.String(), "componente=material") || !strings.Contains(registro.String(), "causa=dependencia_no_disponible") {
		t.Fatalf("diagnóstico no acotado: %s", registro.String())
	}
}
