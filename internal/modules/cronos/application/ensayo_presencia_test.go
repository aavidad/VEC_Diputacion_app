package application

import (
	"context"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type lectorPresenciaTest struct {
	s        ports.SnapshotEnsayoPresencia
	llamadas *int
}

func (l lectorPresenciaTest) LeerSnapshotEnsayoPresencia(context.Context) (ports.SnapshotEnsayoPresencia, error) {
	if l.llamadas != nil {
		*l.llamadas++
	}
	return l.s, nil
}
func snapshotPresenciaTest() ports.SnapshotEnsayoPresencia {
	completo := true
	return ports.SnapshotEnsayoPresencia{VersionEsquema: 1, Demostracion: true, Referencia: "demo:snapshot:1", Version: 1, FuenteRef: "demo:fuente:1", FuenteVersion: 1, OrganizacionRef: "demo:organizacion:1", Fecha: "2026-10-01", ZonaHoraria: "Europe/Madrid", InstanteCorteUTC: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC), PersonasRef: []string{"demo:persona:1", "demo:persona:2"}, Personas: []ports.PersonaEnsayoPresencia{{PersonaRef: "demo:persona:1", CompletaHastaCorte: &completo, Marcajes: []ports.MarcajeEnsayoPresencia{{Referencia: "demo:marca:1", Version: 1, FuenteRef: "demo:fuente:1", Movimiento: domain.PunchEntry, InstanteUTC: time.Date(2026, 10, 1, 6, 0, 0, 0, time.UTC)}}}, {PersonaRef: "demo:persona:2", CompletaHastaCorte: &completo, Marcajes: []ports.MarcajeEnsayoPresencia{}}}, SHA256: strings.Repeat("a", 64)}
}
func TestEnsayoPresenciaSeleccionExactaYAgregadoCoherente(t *testing.T) {
	s := snapshotPresenciaTest()
	s.Personas[0], s.Personas[1] = s.Personas[1], s.Personas[0]
	llamadas := 0
	r, err := EnsayarPresencia(context.Background(), lectorPresenciaTest{s, &llamadas})
	if err != nil || llamadas != 1 || r.Personas[0].PersonaRef != s.PersonasRef[0] || r.Personas[0].Estado != domain.PresenciaRegistrada || r.Personas[1].Estado != domain.PresenciaIndeterminada || r.Personas[0].Motivo != "" || r.Personas[1].Motivo != domain.SinMarcajes || r.Agregado.Total != 2 || r.Agregado.EntradasRegistradas != 1 || r.Agregado.Indeterminado != 1 || r.SnapshotSHA256 != s.SHA256 || r.FuenteVersion != s.FuenteVersion {
		t.Fatalf("%+v %v", r, err)
	}
}
func TestEnsayoPresenciaFallaCerrado(t *testing.T) {
	for _, tc := range []struct {
		nombre string
		mutar  func(*ports.SnapshotEnsayoPresencia)
	}{
		{"seleccion vacia", func(s *ports.SnapshotEnsayoPresencia) { s.PersonasRef = nil; s.Personas = nil }},
		{"duplicada", func(s *ports.SnapshotEnsayoPresencia) { s.PersonasRef[1] = s.PersonasRef[0] }},
		{"persona fuera", func(s *ports.SnapshotEnsayoPresencia) { s.Personas[0].PersonaRef = "demo:otra" }},
		{"sujeto repetido", func(s *ports.SnapshotEnsayoPresencia) { s.Personas[1] = s.Personas[0] }},
		{"comodin", func(s *ports.SnapshotEnsayoPresencia) { s.PersonasRef[0] = "*" }},
		{"no demo", func(s *ports.SnapshotEnsayoPresencia) { s.Demostracion = false }},
		{"ref real", func(s *ports.SnapshotEnsayoPresencia) { s.OrganizacionRef = "organizacion:real" }},
		{"cobertura ausente", func(s *ports.SnapshotEnsayoPresencia) { s.Personas[0].CompletaHastaCorte = nil }},
		{"marcas ausentes", func(s *ports.SnapshotEnsayoPresencia) { s.Personas[1].Marcajes = nil }},
		{"fuente otra", func(s *ports.SnapshotEnsayoPresencia) { s.Personas[0].Marcajes[0].FuenteRef = "demo:otra" }},
		{"version ausente", func(s *ports.SnapshotEnsayoPresencia) { s.Personas[0].Marcajes[0].Version = 0 }},
		{"marca repetida", func(s *ports.SnapshotEnsayoPresencia) {
			s.Personas[0].Marcajes = append(s.Personas[0].Marcajes, s.Personas[0].Marcajes[0])
		}},
		{"posterior corte", func(s *ports.SnapshotEnsayoPresencia) {
			s.Personas[0].Marcajes[0].InstanteUTC = s.InstanteCorteUTC.Add(time.Microsecond)
		}},
		{"fecha distinta", func(s *ports.SnapshotEnsayoPresencia) { s.Fecha = "2026-10-02" }},
		{"fecha no canonica", func(s *ports.SnapshotEnsayoPresencia) { s.Fecha = "2026-1-1" }},
		{"zona local ambiente", func(s *ports.SnapshotEnsayoPresencia) { s.ZonaHoraria = "Local" }},
		{"zona distinta", func(s *ports.SnapshotEnsayoPresencia) { s.ZonaHoraria = "Zona/Inexistente" }},
		{"sha invalido", func(s *ports.SnapshotEnsayoPresencia) { s.SHA256 = "" }},
		{"exceso seleccion", func(s *ports.SnapshotEnsayoPresencia) { s.PersonasRef = make([]string, 101) }},
		{"exceso marcas", func(s *ports.SnapshotEnsayoPresencia) {
			s.Personas[0].Marcajes = make([]ports.MarcajeEnsayoPresencia, 101)
		}},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			s := snapshotPresenciaTest()
			tc.mutar(&s)
			r, err := EnsayarPresencia(context.Background(), lectorPresenciaTest{s, nil})
			if err == nil || r.Demostracion || len(r.Personas) != 0 {
				t.Fatal("invalid snapshot produced output")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	llamadas := 0
	if _, err := EnsayarPresencia(ctx, lectorPresenciaTest{snapshotPresenciaTest(), &llamadas}); err == nil || llamadas != 0 {
		t.Fatal("read cancelled context")
	}
}
func TestEnsayoCoberturaParcialNuncaAfirmaEstado(t *testing.T) {
	s := snapshotPresenciaTest()
	parcial := false
	s.Personas[0].CompletaHastaCorte = &parcial
	r, err := EnsayarPresencia(context.Background(), lectorPresenciaTest{s, nil})
	if err != nil || r.Personas[0].Estado != domain.PresenciaIndeterminada || r.Personas[0].Motivo != domain.CoberturaIncompleta || r.Personas[0].CoberturaCompleta || r.Agregado.Indeterminado != 2 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestEnsayoRechazaLectorNuloTipado(t *testing.T) {
	var lector *lectorPresenciaTest
	if _, err := EnsayarPresencia(context.Background(), lector); err == nil {
		t.Fatal("typed nil accepted")
	}
}
