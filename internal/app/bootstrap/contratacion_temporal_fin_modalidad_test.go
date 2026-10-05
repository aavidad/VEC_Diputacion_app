package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

type relojFinModalidadPrueba struct{}

func (relojFinModalidadPrueba) Ahora() time.Time {
	return time.Date(2026, 10, 2, 15, 0, 0, 0, time.UTC)
}

func TestCatalogoReglasCTVersionTresPublicaFinPorCausa(t *testing.T) {
	ruta := "../../../data/demo/reglas/ct_reglas.ejemplo.demo.v3.json"
	resolver, err := nuevoResolutorReglasEjemplo(ruta, reglas.CatalogoContratacionTemporal,
		reglas.ModuloContratacionTemporal, nil, relojFinModalidadPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	opciones, err := nuevasOpcionesAnalisisCT(context.Background(), resolver)
	if err != nil {
		t.Fatal(err)
	}
	for clave, causa := range map[domain.ClaveCatalogo]domain.ClaveCatalogo{
		"sustitucion": "reincorporacion_titular",
		"vacante":     "cobertura_reglamentaria",
	} {
		modalidad, ok := opciones.modalidad(clave)
		if !ok || modalidad.FechaFin != fechaFinNoAplicaCT || modalidad.CausaFin != causa ||
			modalidad.CatalogoVersion != 3 || modalidad.ReglaRef == "" || modalidad.CatalogoHuellaSHA256 == "" {
			t.Fatalf("regla %s: %+v, existe=%v", clave, modalidad, ok)
		}
	}
}

func TestModalidadFinSinFechaFijaReglaYRechazaFechaInventada(t *testing.T) {
	inicio := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	m := modalidadAnalisisCT{
		Clave: "sustitucion", FechaFin: fechaFinNoAplicaCT, CausaFin: "reincorporacion_titular",
		ReglaRef: "catalogo:3:c12.modalidad.sustitucion", CatalogoVersion: 3,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
	}
	periodo, err := m.periodoConPolitica(domain.PeriodoPrevisto{Inicio: inicio, CausaFin: "reincorporacion_titular"})
	if err != nil || periodo.Fin != (time.Time{}) || periodo.PoliticaFin.CatalogoVersion != 3 ||
		periodo.PoliticaFin.CausaFin != periodo.CausaFin {
		t.Fatalf("fin abierto sin regla conservada: %+v, %v", periodo, err)
	}
	if _, err := m.periodoConPolitica(domain.PeriodoPrevisto{Inicio: inicio, Fin: inicio.AddDate(0, 1, 0)}); err == nil {
		t.Fatal("fecha arbitraria aceptada para no_aplica")
	}
	if _, err := m.periodoConPolitica(domain.PeriodoPrevisto{Inicio: inicio, CausaFin: "cobertura_reglamentaria"}); err == nil {
		t.Fatal("causa ajena al catálogo aceptada")
	}
}
