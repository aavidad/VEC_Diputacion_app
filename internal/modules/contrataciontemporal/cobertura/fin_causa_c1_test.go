package cobertura_test

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestPreparacionC1ConservaCausaYReglaDelAnalisis(t *testing.T) {
	base := time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)
	periodo := domain.PeriodoPrevisto{
		Inicio:   time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC),
		CausaFin: "reincorporacion_titular",
		PoliticaFin: domain.PoliticaFin{
			ReglaRef: "regla:modalidad:sustitucion:v2", CatalogoVersion: 2,
			CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			FechaFin:             "no_aplica", CausaFin: "reincorporacion_titular",
		},
	}
	expediente := domain.Expediente{
		Referencia:      "expediente_temporal_causa_c1_01",
		OrganizacionRef: organizacionOrdenC3,
		Version:         2,
		Solicitud: domain.SolicitudCentro{
			CategoriaRef: "categoria_sintetica_causa_c1", Periodo: periodo,
		},
	}
	catalogo, politica := catalogoYPoliticaOrdenC3(t, base)
	preparacion := prepararC1OrdenC3(
		t, expediente, "analisis_causa_c1_01",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		catalogo, politica, base,
	)
	ordenes, err := preparacion.OrdenesPendientesEn(base)
	if err != nil || len(ordenes) != 1 {
		t.Fatalf("la fuente no preparó C1 con causa: %v", err)
	}
	resumen, err := ordenes[0].ResumenPendienteEn(base)
	if err != nil || resumen.Periodo != periodo {
		t.Fatalf("el resumen no conserva la instantánea original: %v", err)
	}
	propuesta, err := preparacion.DatosCrearPropuestaEn(base)
	if err != nil || propuesta.Periodo != periodo {
		t.Fatalf("la propuesta no conserva la instantánea original: %v", err)
	}
}
