package seguridad

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestGeneradorReferenciasAltaUsaEntropiaIndependiente(t *testing.T) {
	entropia := make([]byte, bytesAleatoriosReferenciaAlta*4)
	for indice := range entropia {
		entropia[indice] = byte(indice + 1)
	}
	generador := &GeneradorReferenciasAltaCriptografico{
		lector: bytes.NewReader(entropia),
		ahora: func() time.Time {
			return time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
		},
	}
	referencias, err := generador.GenerarReferenciasAlta(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	reserva, err := generador.NuevaReferenciaReservaAlta(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if referencias.ExpedienteRef == referencias.ReciboRef ||
		referencias.ExpedienteRef == reserva ||
		referencias.ReciboRef == reserva ||
		referencias.NumeroVisible[:8] != "2026/CT-" {
		t.Fatalf("referencias no separadas: %#v %q", referencias, reserva)
	}
}

func TestGeneradorReferenciasAltaFallaCerrado(t *testing.T) {
	casos := map[string]*GeneradorReferenciasAltaCriptografico{
		"valor cero": {},
		"sin reloj": {
			lector: bytes.NewReader(make([]byte, 128)),
		},
		"entropia insuficiente": {
			lector: bytes.NewReader([]byte{1, 2, 3}),
			ahora:  func() time.Time { return time.Now().UTC() },
		},
	}
	for nombre, generador := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := generador.GenerarReferenciasAlta(
				context.Background(),
			); !errors.Is(err, ErrGeneracionReferenciaAlta) {
				t.Fatalf("error inesperado: %v", err)
			}
		})
	}
}

func TestGeneradorReferenciasAltaPropagaCancelacion(t *testing.T) {
	generador := NuevoGeneradorReferenciasAltaCriptografico()
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := generador.GenerarReferenciasAlta(ctx); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("cancelación perdida: %v", err)
	}
}

func TestGeneradorReferenciasAsignacionGeneraSeisReferenciasSeparadas(t *testing.T) {
	entropia := make([]byte, bytesAleatoriosReferenciaAlta*6)
	for indice := range entropia {
		entropia[indice] = byte(indice + 1)
	}
	generador := &GeneradorReferenciasAltaCriptografico{
		lector: bytes.NewReader(entropia),
		ahora:  func() time.Time { return time.Now().UTC() },
	}
	referencias, err := generador.GenerarReferenciasAsignacion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if referencias.Validar() != nil {
		t.Fatalf("referencias inválidas: %#v", referencias)
	}
	vistas := []string{
		referencias.ReservaRef,
		referencias.ReciboRef,
		referencias.NotificacionRef,
		referencias.BandejaRef,
		referencias.AuditoriaRef,
		referencias.EventoRef,
	}
	unicas := make(map[string]struct{}, len(vistas))
	for _, referencia := range vistas {
		unicas[referencia] = struct{}{}
	}
	if len(unicas) != len(vistas) {
		t.Fatalf("referencias de efecto repetidas: %#v", referencias)
	}
}

func TestGeneradorReferenciasAsignacionFallaCerrado(t *testing.T) {
	generador := &GeneradorReferenciasAltaCriptografico{
		lector: bytes.NewReader(make([]byte, bytesAleatoriosReferenciaAlta*5)),
		ahora:  func() time.Time { return time.Now().UTC() },
	}
	if referencias, err := generador.GenerarReferenciasAsignacion(
		context.Background(),
	); !errors.Is(err, ErrGeneracionReferenciaAlta) ||
		referencias != (ports.ReferenciasEfectoAsignacion{}) {
		t.Fatalf("entropía insuficiente aceptada: %#v / %v", referencias, err)
	}
}
