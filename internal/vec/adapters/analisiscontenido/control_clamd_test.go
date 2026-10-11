package analisiscontenido

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidarPoliticaClamdAlertExceedsMax(t *testing.T) {
	const dbDir = "/tmp/vec-analizador-clamd-prueba-db"
	const socket = "/tmp/vec-analizador-clamd-prueba-socket/clamd.sock"
	politica := strings.Join([]string{
		"AlertExceedsMax yes",
		"DatabaseDirectory " + dbDir,
		"LocalSocket " + socket,
		"LocalSocketMode 600",
		"ScanPDF yes",
		"StreamMaxLength 40M",
		"MaxFileSize 40M",
		"MaxScanSize 128M",
	}, "\n") + "\n"
	for _, caso := range []struct {
		nombre, texto string
		valida        bool
	}{
		{"activada", politica, true},
		{"desactivada", strings.Replace(politica, "AlertExceedsMax yes", "AlertExceedsMax no", 1), false},
		{"tcp_prohibido", politica + "TCPSocket 3310\n", false},
		{"duplicada", politica + "AlertExceedsMax yes\n", false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			err := validarPoliticaClamd([]byte(caso.texto), dbDir, socket)
			if caso.valida && err != nil {
				t.Fatalf("política válida rechazada: %v", err)
			}
			if !caso.valida && !errors.Is(err, ErrConfiguracion) {
				t.Fatalf("política inválida aceptada: %v", err)
			}
		})
	}
}

func TestIniciarClamdControladoPoliticaYSalidaSinSocket(t *testing.T) {
	for _, caso := range []struct {
		nombre, alert string
		error         error
	}{
		{"alerta_desactivada", "no", ErrConfiguracion},
		{"daemon_termina_sin_socket", "yes", ErrRemoto},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			base, err := os.MkdirTemp("/tmp", "vec-analizador-clamd-prueba-")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(base) })
			dbDir := filepath.Join(base, "db")
			socketDir := filepath.Join(base, "socket")
			for _, dir := range []string{dbDir, socketDir} {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			socket := filepath.Join(socketDir, "clamd.sock")
			politica := strings.Join([]string{
				"AlertExceedsMax " + caso.alert,
				"DatabaseDirectory " + dbDir,
				"LocalSocket " + socket,
				"LocalSocketMode 600",
				"ScanPDF yes",
				"StreamMaxLength 40M",
				"MaxFileSize 40M",
				"MaxScanSize 128M",
			}, "\n") + "\n"
			rutaConfig := filepath.Join(base, "clamd.conf")
			if err := os.WriteFile(rutaConfig, []byte(politica), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelar()
			control, err := IniciarClamdControlado(ctx, ConfiguracionClamdControlado{
				Bwrap: buscarHerramienta(t, "bwrap"), ClamdBin: "/usr/bin/gnutrue", Prlimit: buscarHerramienta(t, "prlimit"),
				RutaConfiguracion: rutaConfig, DBDir: dbDir, SocketDir: socketDir, MaxEdadFirmas: time.Hour,
			})
			if control != nil || !errors.Is(err, caso.error) || ctx.Err() != nil {
				t.Fatalf("control=%v, error=%v, contexto=%v; esperado=%v", control, err, ctx.Err(), caso.error)
			}
			if _, err := os.Lstat(socket); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("el daemon falso dejó un socket: %v", err)
			}
		})
	}
}

func TestCapacidadesSaturadasNoConsultaMotores(t *testing.T) {
	identidad, err := url.Parse("spiffe://vec.test/analizador-cliente")
	if err != nil {
		t.Fatal(err)
	}
	// Los motores quedan intencionadamente ausentes: cualquier consulta antes
	// de aplicar el cupo produciría un fallo en vez del 503 esperado.
	servicio := &Servicio{
		identidades: map[string]struct{}{identidad.String(): {}},
		ocupacion:   make(chan struct{}, 2),
	}
	servicio.ocupacion <- struct{}{}
	servicio.ocupacion <- struct{}{}
	cert := &x509.Certificate{URIs: []*url.URL{identidad}}
	peticion := httptest.NewRequest(http.MethodGet, "https://vec.test/v1/capacidades", nil)
	peticion.TLS = &tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{cert},
		VerifiedChains:   [][]*x509.Certificate{{cert}},
	}
	respuesta := httptest.NewRecorder()
	servicio.Handler().ServeHTTP(respuesta, peticion)
	if respuesta.Code != http.StatusServiceUnavailable || len(servicio.ocupacion) != 2 {
		t.Fatalf("cupo ignorado: HTTP %d, ocupación %d", respuesta.Code, len(servicio.ocupacion))
	}
}
