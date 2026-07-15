package seguridad

import (
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestSelladorHMACEsDeterministaYSeparaClaves(t *testing.T) {
	primero, err := NuevoSelladorHMAC("clave-2026-01", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("NuevoSelladorHMAC() error = %v", err)
	}
	segundo, err := NuevoSelladorHMAC("clave-2026-02", []byte("abcdef0123456789abcdef0123456789"))
	if err != nil {
		t.Fatalf("segundo sellador: %v", err)
	}
	a, err := primero.SellarDatos(context.Background(), []byte("dato sensible"))
	if err != nil {
		t.Fatalf("SellarDatos() error = %v", err)
	}
	b, _ := primero.SellarDatos(context.Background(), []byte("dato sensible"))
	c, _ := segundo.SellarDatos(context.Background(), []byte("dato sensible"))
	if a != b || a == c || !strings.HasPrefix(a, "hmac-sha256:clave-2026-01:") {
		t.Fatalf("sellos inesperados: a=%q b=%q c=%q", a, b, c)
	}
}

func TestSelladorHMACRechazaClaveCorta(t *testing.T) {
	_, err := NuevoSelladorHMAC("clave", []byte("demasiado-corta"))
	if !errors.Is(err, ErrConfiguracionSeguridadInvalida) {
		t.Fatalf("NuevoSelladorHMAC() error = %v", err)
	}
}

func TestSelladorHMACSeudonimizaPorAmbitoConClaveExclusiva(t *testing.T) {
	seudonimizador, err := NuevoSelladorHMAC(
		"seudonimo-almacen-v1", []byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudSeudonimizarSujetoAlmacen(
		"persona:0123456789abcdef", "documento:hmac-solicitud-uno",
	)
	if err != nil {
		t.Fatal(err)
	}
	primero, err := seudonimizador.SeudonimizarSujetoAlmacen(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	segundo, _ := seudonimizador.SeudonimizarSujetoAlmacen(context.Background(), solicitud)
	otroAmbito, _ := ports.NuevaSolicitudSeudonimizarSujetoAlmacen(
		"persona:0123456789abcdef", "documento:hmac-solicitud-dos",
	)
	tercero, _ := seudonimizador.SeudonimizarSujetoAlmacen(context.Background(), otroAmbito)
	if primero != segundo || primero == tercero ||
		!strings.HasPrefix(primero, "hmac-sha256:seudonimo-almacen-v1:") ||
		strings.Contains(primero, "persona:0123456789abcdef") {
		t.Fatalf("seudonimos incorrectos: %q / %q / %q", primero, segundo, tercero)
	}
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := seudonimizador.SeudonimizarSujetoAlmacen(ctxCancelado, solicitud); !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado aceptado: %v", err)
	}
	if _, err := seudonimizador.SeudonimizarSujetoAlmacen(
		context.Background(), ports.SolicitudSeudonimizarSujetoAlmacen{},
	); !errors.Is(err, ports.ErrSeudonimizacionAlmacenNoDisponible) {
		t.Fatalf("solicitud cero aceptada: %v", err)
	}
}

func TestGeneradorIDCreaIdentificadoresDistintos(t *testing.T) {
	a, err := (GeneradorID{}).NuevoIDDocumento()
	if err != nil {
		t.Fatalf("primer ID: %v", err)
	}
	b, err := (GeneradorID{}).NuevoIDDocumento()
	if err != nil {
		t.Fatalf("segundo ID: %v", err)
	}
	if a == b || !strings.HasPrefix(a, "documento-") || !strings.HasPrefix(b, "documento-") {
		t.Fatalf("identificadores inesperados: %q %q", a, b)
	}
}
