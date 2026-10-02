package bootstrap

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

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
