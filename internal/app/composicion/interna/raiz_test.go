package interna

import (
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
)

func TestNuevoServidorPermaneceCerradoConInventarioCompleto(t *testing.T) {
	servidor, err := NuevoServidor(configuracionInternaValidaPrueba())
	if servidor != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) {
		t.Fatalf("raiz C4 = (%v, %v)", servidor, err)
	}
	var faltantes *ErrorDependenciasFaltantes
	if !errors.As(err, &faltantes) {
		t.Fatalf("error no conserva el inventario tipado: %T", err)
	}
	esperadas := append([]Dependencia(nil), dependenciasC4[:]...)
	if !reflect.DeepEqual(faltantes.Faltantes(), esperadas) {
		t.Fatalf("faltantes = %v; se esperaba %v", faltantes.Faltantes(), esperadas)
	}
	for _, dependencia := range esperadas {
		if !faltantes.Falta(dependencia) {
			t.Errorf("no se declaro la dependencia %s", dependencia)
		}
	}
	if strings.Contains(err.Error(), "/run/secrets") || strings.Contains(err.Error(), "10.7.15.40") {
		t.Fatalf("el error revelo configuracion: %q", err)
	}
}

func TestNuevoServidorNoConstruyeHealthcheckNiAbreSocket(t *testing.T) {
	reserva, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	direccion := reserva.Addr().String()
	if err := reserva.Close(); err != nil {
		t.Fatal(err)
	}

	cfg := configuracionInternaValidaPrueba()
	cfg.DireccionEscucha = direccion
	cfg.RedesPermitidas = []string{"127.0.0.0/8"}
	servidor, err := NuevoServidor(cfg)
	if servidor != nil || !errors.Is(err, ErrDependenciasProductivasNoDisponibles) {
		t.Fatalf("raiz C4 = (%v, %v)", servidor, err)
	}

	comprobacion, err := net.Listen("tcp", direccion)
	if err != nil {
		t.Fatalf("la raiz abrio el socket antes de completar C5/C6: %v", err)
	}
	if err := comprobacion.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNuevoServidorValidaConfiguracionAntesDelInventario(t *testing.T) {
	cfg := configuracionInternaValidaPrueba()
	cfg.CertificadoServidorTLS = "MARCADOR_PRIVADO_NO_REFLEJAR"
	servidor, err := NuevoServidor(cfg)
	if servidor != nil || !errors.Is(err, ErrConfiguracionTLSIncompleta) {
		t.Fatalf("configuracion insegura = (%v, %v)", servidor, err)
	}
	if strings.Contains(err.Error(), "MARCADOR") {
		t.Fatalf("el error reflejo el valor invalido: %q", err)
	}
}
