package autorizacion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type emisorIngestaNoConcede struct{ acciones []string }

func (e *emisorIngestaNoConcede) AutorizarOperacionIngestaMayan(_ context.Context, accion string, _ []byte, _, _ string) (docports.AutorizacionV3, error) {
	e.acciones = append(e.acciones, accion)
	return docports.AutorizacionV3{}, errors.New("PDP denegó")
}

func TestIngestaMayanV3EnviaTresAccionesReservaAlPDP(t *testing.T) {
	emisor := &emisorIngestaNoConcede{}
	a, err := NuevaAutoridadIngestaMayanV3(emisor, relojFijo{time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	ref := "ref:" + strings.Repeat("a", 64)
	acciones := []string{docports.AccionReservarRecursoMayan, docports.AccionConsultarReservaMayan, docports.AccionAbandonarRecursoMayan}
	for _, accion := range acciones {
		if _, err := a.AutorizarIngestaMayan(context.Background(), accion, []byte(`{"dato":{}}`), ref, ref); !errors.Is(err, vecports.ErrAutorizacionAlmacenInvalida) {
			t.Fatalf("%s: denegación inesperada: %v", accion, err)
		}
	}
	if len(emisor.acciones) != len(acciones) {
		t.Fatalf("se omitieron acciones de reserva: %+v", emisor.acciones)
	}
	for i, accion := range acciones {
		if emisor.acciones[i] != accion {
			t.Fatalf("PDP recibió %s, esperado %s", emisor.acciones[i], accion)
		}
	}
	_, _ = a.AutorizarIngestaMayan(context.Background(), "documentos.ingesta_mayan.no_publicada", []byte(`{}`), ref, ref)
	if len(emisor.acciones) != len(acciones) {
		t.Fatal("acción ajena llegó al PDP")
	}
}
