package seguridad

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestGeneradorReferenciasFuentesAutoridadSeparaEspaciosYUsa128Bits(t *testing.T) {
	generador := GeneradorReferenciasCriptograficas{}
	ctx := context.Background()

	solicitud, err := generador.NuevaReferenciaSolicitud(ctx)
	if err != nil {
		t.Fatalf("generar referencia de solicitud: %v", err)
	}
	operacion, err := generador.NuevaReferenciaOperacion(ctx)
	if err != nil {
		t.Fatalf("generar referencia de operacion: %v", err)
	}
	valorSolicitud, err := solicitud.Referencia()
	if err != nil {
		t.Fatalf("leer referencia de solicitud: %v", err)
	}
	valorOperacion, err := operacion.Referencia()
	if err != nil {
		t.Fatalf("leer referencia de operacion: %v", err)
	}

	comprobarReferenciaCriptograficaAutoridad(
		t,
		valorSolicitud,
		ports.PrefijoReferenciaSolicitudFuenteAutoridad,
	)
	comprobarReferenciaCriptograficaAutoridad(
		t,
		valorOperacion,
		ports.PrefijoReferenciaOperacionFuenteAutoridad,
	)
	if valorSolicitud == valorOperacion {
		t.Fatal("solicitud y operacion comparten referencia")
	}
}

func TestGeneradorReferenciasFuentesAutoridadFallaCerrado(t *testing.T) {
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()

	casos := []struct {
		nombre  string
		ctx     context.Context
		lector  io.Reader
		esperar error
	}{
		{nombre: "contexto nulo", lector: bytes.NewReader(make([]byte, 16)), esperar: ports.ErrGeneracionReferenciaFuenteAutoridad},
		{nombre: "lector nulo", ctx: context.Background(), esperar: ports.ErrGeneracionReferenciaFuenteAutoridad},
		{nombre: "cancelacion", ctx: ctxCancelado, lector: bytes.NewReader(make([]byte, 16)), esperar: context.Canceled},
		{nombre: "entropia insuficiente", ctx: context.Background(), lector: bytes.NewReader(make([]byte, 15)), esperar: io.ErrUnexpectedEOF},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			valor, err := nuevaReferenciaFuenteAutoridad(
				caso.ctx,
				caso.lector,
				ports.PrefijoReferenciaSolicitudFuenteAutoridad,
			)
			if valor != "" || !errors.Is(err, ports.ErrGeneracionReferenciaFuenteAutoridad) ||
				!errors.Is(err, caso.esperar) {
				t.Fatalf("resultado = %q, %v", valor, err)
			}
		})
	}
}

func TestGeneradorReferenciasFuentesAutoridadVectorDeterminista(t *testing.T) {
	valor, err := nuevaReferenciaFuenteAutoridad(
		context.Background(),
		bytes.NewReader([]byte{
			0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
			0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
		}),
		ports.PrefijoReferenciaOperacionFuenteAutoridad,
	)
	const esperado = ports.PrefijoReferenciaOperacionFuenteAutoridad +
		"00112233445566778899aabbccddeeff"
	if err != nil || valor != esperado {
		t.Fatalf("vector = %q, %v; esperado %q", valor, err, esperado)
	}
}

func TestGeneradorReferenciasAutorizacionV2CreaPerfilesSeparados(t *testing.T) {
	generador := GeneradorReferenciasCriptograficas{}
	ctx := context.Background()

	correlacion, err := generador.NuevaReferenciaCorrelacionAutorizacionV2(ctx)
	if err != nil {
		t.Fatalf("generar correlacion: %v", err)
	}
	claveMotivo, err := generador.NuevaClaveMotivoAutorizacionV2(ctx)
	if err != nil {
		t.Fatalf("generar clave de motivo: %v", err)
	}
	if !strings.HasPrefix(correlacion, "correlacion_") || len(correlacion) != len("correlacion_")+32 {
		t.Fatalf("correlacion fuera de perfil: %q", correlacion)
	}
	if !domain.ClaveMotivoAutorizacionV2Valida(claveMotivo) ||
		!strings.HasPrefix(claveMotivo, "motivo_") || len(claveMotivo) != len("motivo_")+32 {
		t.Fatalf("clave de motivo fuera de perfil: %q", claveMotivo)
	}
}

func TestGeneradorReferenciasAutorizacionV2FallaCerrado(t *testing.T) {
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()

	for _, caso := range []struct {
		nombre  string
		ctx     context.Context
		lector  io.Reader
		esperar error
	}{
		{nombre: "contexto nulo", lector: bytes.NewReader(make([]byte, 16)), esperar: ports.ErrGeneracionReferenciaAutorizacionV2},
		{nombre: "lector nulo", ctx: context.Background(), esperar: ports.ErrGeneracionReferenciaAutorizacionV2},
		{nombre: "cancelacion", ctx: ctxCancelado, lector: bytes.NewReader(make([]byte, 16)), esperar: context.Canceled},
		{nombre: "entropia insuficiente", ctx: context.Background(), lector: bytes.NewReader(make([]byte, 15)), esperar: io.ErrUnexpectedEOF},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			valor, err := nuevaReferenciaOpacaHex128(
				caso.ctx,
				caso.lector,
				"correlacion_",
				ports.ErrGeneracionReferenciaAutorizacionV2,
			)
			if valor != "" || !errors.Is(err, ports.ErrGeneracionReferenciaAutorizacionV2) ||
				!errors.Is(err, caso.esperar) {
				t.Fatalf("resultado = %q, %v", valor, err)
			}
		})
	}
}

func comprobarReferenciaCriptograficaAutoridad(t *testing.T, valor, prefijo string) {
	t.Helper()
	sufijo := strings.TrimPrefix(valor, prefijo)
	if !strings.HasPrefix(valor, prefijo) || len(sufijo) != 32 {
		t.Fatalf("referencia sin 128 bits hexadecimales: %q", valor)
	}
	for _, caracter := range sufijo {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			t.Fatalf("sufijo no hexadecimal minusculo: %q", sufijo)
		}
	}
}
