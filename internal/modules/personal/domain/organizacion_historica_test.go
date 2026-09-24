package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func selectorHistoricoPrueba() SelectorOrganizacionHistorica {
	fecha, _ := NuevaFechaCivil("2024-12-31")
	return SelectorOrganizacionHistorica{
		OrganismoRef: "organismo:dipgra", UnidadClave: "centro:uno", VigenteEn: fecha,
		ConocidoEn: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), Limite: 20,
	}
}

func TestSelectorHistoricoExigeDosCortesYLimite(t *testing.T) {
	base := selectorHistoricoPrueba()
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	for nombre, cambiar := range map[string]func(*SelectorOrganizacionHistorica){
		"fecha civil":        func(s *SelectorOrganizacionHistorica) { s.VigenteEn = "2024-02-30" },
		"conocimiento local": func(s *SelectorOrganizacionHistorica) { s.ConocidoEn = s.ConocidoEn.In(time.FixedZone("local", 3600)) },
		"submicrosegundo":    func(s *SelectorOrganizacionHistorica) { s.ConocidoEn = s.ConocidoEn.Add(time.Nanosecond) },
		"organismo":          func(s *SelectorOrganizacionHistorica) { s.OrganismoRef = "" },
		"unidad":             func(s *SelectorOrganizacionHistorica) { s.UnidadClave = "*" },
		"limite":             func(s *SelectorOrganizacionHistorica) { s.Limite = 101 },
		"cursor":             func(s *SelectorOrganizacionHistorica) { s.Cursor = "a/b" },
	} {
		t.Run(nombre, func(t *testing.T) {
			s := base
			cambiar(&s)
			if !errors.Is(s.Validar(), ErrConsultaOrganizacionHistoricaInvalida) {
				t.Fatalf("acepto %+v", s)
			}
		})
	}
}

func TestTrazaHistoricaDistingueEfectosDeConocimiento(t *testing.T) {
	s := selectorHistoricoPrueba()
	desde, _ := NuevaFechaCivil("2024-01-01")
	hasta, _ := NuevaFechaCivil("2025-01-01")
	base := TrazaOrganizacionHistorica{ID: "plaza:uno", Version: 2, FuenteRef: "fuente:rpt", ActoRef: "acto:dos", HuellaSHA256: strings.Repeat("a", 64), EfectosDesde: desde, EfectosHasta: hasta, ConocidoDesde: time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)}
	if err := base.ValidarEn(s); err != nil {
		t.Fatal(err)
	}
	previo := s
	previo.ConocidoEn = time.Date(2025, 1, 9, 0, 0, 0, 0, time.UTC)
	if !errors.Is(base.ValidarEn(previo), ErrOrganizacionHistoricaNoDisponible) {
		t.Fatal("rectificacion visible antes de conocerse")
	}
	despues := s
	despues.VigenteEn = "2025-01-01"
	if !errors.Is(base.ValidarEn(despues), ErrOrganizacionHistoricaNoDisponible) {
		t.Fatal("fin de efectos inclusivo")
	}
	base.ConocidoHasta = s.ConocidoEn
	if !errors.Is(base.ValidarEn(s), ErrOrganizacionHistoricaNoDisponible) {
		t.Fatal("fin de conocimiento inclusivo")
	}
}
