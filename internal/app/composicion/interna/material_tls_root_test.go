package interna

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const entornoDirectorioTLSRootPrueba = "VEC_PRUEBA_TLS_ROOT_DIR"

func TestPrepararMaterialTLSRootParaRuntimeNoPrivilegiado(t *testing.T) {
	directorioRaiz := os.Getenv(entornoDirectorioTLSRootPrueba)
	if directorioRaiz == "" {
		t.Skip("solo lo ejecuta el runner root/nonroot")
	}
	if os.Geteuid() != 0 {
		t.Fatal("preparacion TLS requiere UID provisionador 0")
	}
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	directorio := filepath.Join(directorioRaiz, "secretos")
	if err := os.Mkdir(directorio, 0o750); err != nil {
		t.Fatal(err)
	}
	copiarMaterialTLSPrueba(t, material.cfg.CertificadoServidorTLS, filepath.Join(directorio, "servidor.crt"), 0o440)
	copiarMaterialTLSPrueba(t, material.cfg.ClaveServidorTLS, filepath.Join(directorio, "servidor.key"), 0o440)
	copiarMaterialTLSPrueba(t, material.cfg.AutoridadClientesTLS, filepath.Join(directorio, "clientes-ca.crt"), 0o440)
	if err := os.Chmod(directorio, 0o550); err != nil {
		t.Fatal(err)
	}
}

func TestConstruirServidorInternoCargaRealComoRuntimeNoPrivilegiado(t *testing.T) {
	directorioRaiz := os.Getenv(entornoDirectorioTLSRootPrueba)
	if directorioRaiz == "" {
		t.Skip("solo lo ejecuta el runner root/nonroot")
	}
	cfg := configuracionInternaValidaPrueba()
	cfg.DireccionEscucha = "127.0.0.1:8443"
	cfg.RedesPermitidas = []string{"127.0.0.0/8"}
	cfg.NombreServidorTLS = "servidor.interna.test"
	cfg.CertificadoServidorTLS = filepath.Join(directorioRaiz, "secretos", "servidor.crt")
	cfg.ClaveServidorTLS = filepath.Join(directorioRaiz, "secretos", "servidor.key")
	cfg.AutoridadClientesTLS = filepath.Join(directorioRaiz, "secretos", "clientes-ca.crt")
	servidor, err := construirServidorInterno(cfg, http.NotFoundHandler())
	if os.Geteuid() == 0 {
		if servidor != nil || !errors.Is(err, ErrTLSMutuoNoVerificado) {
			t.Fatalf("runtime root aceptado = (%v, %v)", servidor, err)
		}
		return
	}
	if os.Geteuid() != 10001 || os.Getegid() != 10001 {
		t.Fatalf("identidad de prueba inesperada: %d:%d", os.Geteuid(), os.Getegid())
	}
	if err != nil || servidor == nil {
		t.Fatalf("carga root:grupo-runtime 0440 = (%v, %v)", servidor, err)
	}
	if err := ValidarServidorParaEscucha(servidor); err != nil {
		t.Fatalf("servidor cargado no conserva sello: %v", err)
	}
}

func copiarMaterialTLSPrueba(t *testing.T, origen, destino string, modo os.FileMode) {
	t.Helper()
	contenido, err := os.ReadFile(origen)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destino, contenido, modo); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(destino, modo); err != nil {
		t.Fatal(err)
	}
}
