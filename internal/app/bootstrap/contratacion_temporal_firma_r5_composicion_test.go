package bootstrap

import (
	"errors"
	"testing"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

func TestComposicionFirmasR5DeniegaDependenciasIncompletas(t *testing.T) {
	for _, f := range []*firmaDocumentoCTDesarrollo{nil, {}, {servicio: &ctapplication.ServicioFirmaDocumento{}}} {
		if err := f.componerFirmasR5(dependenciasFirmaR5Desarrollo{}); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) {
			t.Fatalf("composición incompleta: %v", err)
		}
		if f != nil && (f.firmaExterna != nil || f.firmaVec != nil) {
			t.Fatal("una dependencia ausente publicó una vía R5")
		}
	}
}
