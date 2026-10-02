package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type lectorIncidenciasTest struct {
	snapshot ports.SnapshotEnsayoIncidencias
	err      error
	llamados int
	cancelar context.CancelFunc
}

func (l *lectorIncidenciasTest) LeerSnapshotEnsayoIncidencias(context.Context) (ports.SnapshotEnsayoIncidencias, error) {
	l.llamados++
	if l.cancelar != nil {
		l.cancelar()
	}
	return l.snapshot, l.err
}
func snapshotIncidenciasTest() ports.SnapshotEnsayoIncidencias {
	return ports.SnapshotEnsayoIncidencias{VersionEsquema: 1, Demostracion: true, Ref: "demo:snapshot:uno", Version: 1, PersonaRef: "demo:persona:uno", SHA256: strings.Repeat("a", 64), Fuente: ports.FuenteEnsayoIncidencias{Referencia: "demo:fuente:uno", Version: 1, SHA256: strings.Repeat("b", 64)}, Consulta: domain.ConsultaIncidenciasPeriodo{Desde: "2026-10-01", Hasta: "2026-10-02", Zona: "UTC", CorteUTC: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), AntecedentesCompletos: true, Cobertura: []domain.CoberturaRegistro{{Fecha: "2026-10-01", Estado: domain.CoberturaRegistroCompleta}}}}
}

func TestEnsayoIncidenciasFuenteCaidaPropagaErrorSinResultado(t *testing.T) {
	causa := errors.New("synthetic read failure")
	r, err := EnsayarIncidencias(context.Background(), &lectorIncidenciasTest{err: causa})
	if !errors.Is(err, causa) || r.Demostracion || r.Dias != nil {
		t.Fatalf("error became empty healthy result: %+v %v", r, err)
	}
}
func TestEnsayoIncidenciasCancelacionAntesYDuranteFuente(t *testing.T) {
	for _, durante := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		l := &lectorIncidenciasTest{snapshot: snapshotIncidenciasTest()}
		if durante {
			l.cancelar = cancel
		} else {
			cancel()
		}
		r, err := EnsayarIncidencias(ctx, l)
		cancel()
		if !errors.Is(err, context.Canceled) || r.Demostracion {
			t.Fatalf("cancellation ignored: %+v %v", r, err)
		}
		if !durante && l.llamados != 0 {
			t.Fatal("source read after cancellation")
		}
	}
	var l *lectorIncidenciasTest
	if _, err := EnsayarIncidencias(context.Background(), l); err == nil {
		t.Fatal("accepted typed nil")
	}
}
func TestEnsayoIncidenciasConservaFuenteYNoClasificaFaltaDeDatos(t *testing.T) {
	s := snapshotIncidenciasTest()
	s.Consulta.Cobertura = nil
	r, err := EnsayarIncidencias(context.Background(), &lectorIncidenciasTest{snapshot: s})
	if err != nil {
		t.Fatal(err)
	}
	if r.Fuente != s.Fuente || r.SnapshotSHA256 != s.SHA256 || r.SnapshotVersion != s.Version || r.PersonaRef != s.PersonaRef || r.Dias[0].Estado != "indeterminado" || r.Dias[0].HechosRegistrados != nil {
		t.Fatalf("lost provenance or invented count: %+v", r)
	}
}
func TestEnsayoIncidenciasSoloSnapshotDemoVersionado(t *testing.T) {
	cases := map[string]func(*ports.SnapshotEnsayoIncidencias){
		"not synthetic":      func(s *ports.SnapshotEnsayoIncidencias) { s.Demostracion = false },
		"schema":             func(s *ports.SnapshotEnsayoIncidencias) { s.VersionEsquema = 2 },
		"version":            func(s *ports.SnapshotEnsayoIncidencias) { s.Version = 0 },
		"source version":     func(s *ports.SnapshotEnsayoIncidencias) { s.Fuente.Version = 0 },
		"source hash":        func(s *ports.SnapshotEnsayoIncidencias) { s.Fuente.SHA256 = "missing" },
		"snapshot hash":      func(s *ports.SnapshotEnsayoIncidencias) { s.SHA256 = "missing" },
		"snapshot reference": func(s *ports.SnapshotEnsayoIncidencias) { s.Ref = "snapshot:real" },
		"person reference":   func(s *ports.SnapshotEnsayoIncidencias) { s.PersonaRef = "persona:real" },
		"source reference":   func(s *ports.SnapshotEnsayoIncidencias) { s.Fuente.Referencia = "fuente:real" },
		"fact reference": func(s *ports.SnapshotEnsayoIncidencias) {
			s.Consulta.Hechos = []domain.HechoRegistroIncidencia{{Ref: "registro:real", Version: 1, Movimiento: domain.PunchEntry, InstanteUTC: s.Consulta.CorteUTC.Add(-time.Hour)}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := snapshotIncidenciasTest()
			mutate(&s)
			r, err := EnsayarIncidencias(context.Background(), &lectorIncidenciasTest{snapshot: s})
			if err == nil || r.Demostracion {
				t.Fatalf("accepted %s", name)
			}
		})
	}
}
