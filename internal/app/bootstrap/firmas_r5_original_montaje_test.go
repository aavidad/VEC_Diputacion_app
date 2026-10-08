package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestMontajeOriginalFirmaR5DeniegaDependenciasAusentes(t *testing.T) {
	if _, err := nuevoMontajeOriginalFirmaR5Desarrollo(nil, nil, nil); !errors.Is(err, vecports.ErrOriginalFirmableCTNoDisponible) {
		t.Fatalf("fuente y autoridades ausentes: %v", err)
	}
	if _, err := nuevoMontajeOriginalFirmaR5Desarrollo(nil, &autoridadDocumentosDesarrollo{}, nil); !errors.Is(err, vecports.ErrOriginalFirmableCTNoDisponible) {
		t.Fatalf("Documentos incompleto: %v", err)
	}
}

func TestPoliticasConservacionR5MantienenVersionesExactas(t *testing.T) {
	reloj := relojRutasDietas{}
	v1, resolverApagado, err := catalogosDocumentosR5Desarrollo(reloj, false)
	if err != nil {
		t.Fatal(err)
	}
	if resolverApagado != v1 {
		t.Fatal("selector apagado sustituyó el catálogo v1")
	}
	v2, resolutor, err := catalogosDocumentosR5Desarrollo(reloj, true)
	if err != nil {
		t.Fatal(err)
	}
	resolver, ok := resolutor.(politicasConservacionR5Desarrollo)
	if !ok || resolver.actual != v2 || resolver.historico == nil {
		t.Fatal("selector activo no montó v2 y la lectura histórica v1")
	}
	for _, tc := range []struct {
		catalogo *conservacion.Catalogo
		tipo     string
	}{
		{v1, "contratacion_temporal.borrador.v1"},
		{v1, "dietas.justificante.v1"},
		{v2, "contratacion_temporal.borrador.resolucion.v1"},
		{v2, "contratacion_temporal.resolucion_firmada.v1"},
	} {
		solicitud, err := tc.catalogo.SolicitudPara(tc.tipo, "ref:"+strings.Repeat("ab", 32))
		if err != nil {
			t.Fatalf("solicitud %s: %v", tc.tipo, err)
		}
		politicas, err := resolver.BuscarPoliticasConservacionDocumental(context.Background(), solicitud)
		if err != nil || len(politicas) != 1 || !reflect.DeepEqual(politicas[0].Solicitud(), solicitud) {
			t.Fatalf("política %s: coincidencias=%d err=%v", tc.tipo, len(politicas), err)
		}
	}
	if _, err := resolver.BuscarPoliticasConservacionDocumental(context.Background(), vecports.SolicitudPoliticaConservacionDocumental{}); !errors.Is(err, vecports.ErrPoliticaConservacionDocumentalNoResuelta) {
		t.Fatalf("solicitud vacía: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := resolver.BuscarPoliticasConservacionDocumental(ctx, vecports.SolicitudPoliticaConservacionDocumental{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("contexto cancelado oculto: %v", err)
	}
}
