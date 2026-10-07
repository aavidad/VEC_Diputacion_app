package administracion

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMaterialConexionPanicDeniegaSinMaterialNiDetalles(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	defer slog.SetDefault(anterior)
	huella, err := materialConexionPerfiles(tls.ConnectionState{})
	if huella != [32]byte{} || !errors.Is(err, errAccesoDenegado) {
		t.Fatal("panic concedio material o perdió denegación", err)
	}
	if err.Error() != errAccesoDenegado.Error() || !strings.Contains(registro.String(), errAccesoDenegado.Error()) {
		t.Fatal("fallo no propagado y registrado de forma genérica")
	}
	if strings.Contains(registro.String(), "runtime") || strings.Contains(registro.String(), "certificate") || strings.Contains(registro.String(), "VEC-ADMIN") {
		t.Fatal("registro expuso detalle del canal")
	}
}

func TestMaterialConexionErrorExporterSePropagaYDeniegaSesion(t *testing.T) {
	servidor := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer servidor.Close()
	config := servidor.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	config.InsecureSkipVerify = false
	config.MinVersion = tls.VersionTLS13
	config.RootCAs.AddCert(servidor.Certificate())
	// tls rechaza ExportKeyingMaterial si la conexión permite renegociación.
	config.Renegotiation = tls.RenegotiateFreelyAsClient
	cliente, err := tls.Dial("tcp", servidor.Listener.Addr().String(), config)
	if err != nil {
		t.Fatal(err)
	}
	defer cliente.Close()
	estado := cliente.ConnectionState()
	_, causa := estado.ExportKeyingMaterial("VEC-ADMIN-perfiles-conexion-v1", nil, 32)
	if causa == nil {
		t.Fatal("el escenario no rechazó el exporter")
	}
	huella, err := materialConexionPerfiles(estado)
	if huella != [32]byte{} || !errors.Is(err, errAccesoDenegado) {
		t.Fatal("error del exporter produjo material")
	}
	var fallo falloMaterialConexionPerfiles
	if !errors.As(err, &fallo) || fallo.causa == nil || fallo.causa.Error() != causa.Error() || err.Error() != errAccesoDenegado.Error() {
		t.Fatal("causa no preservada o expuesta en error")
	}
	ctx := context.WithValue(context.Background(), claveConexionPerfiles{}, conexionPerfiles{aceptadaEn: time.Now().UTC(), conexion: cliente})
	instante, err := autenticacionConexionPerfiles(ctx, &http.Request{TLS: &estado})
	if !instante.IsZero() || !errors.As(err, &fallo) || !errors.Is(err, errAccesoDenegado) {
		t.Fatal("fallo de material no cerró la sesión")
	}
}
