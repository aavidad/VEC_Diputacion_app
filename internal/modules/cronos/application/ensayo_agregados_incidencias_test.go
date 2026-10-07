package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type lectorAgregadosTest struct {
	snapshot ports.SnapshotEnsayoAgregadosIncidencias
	err      error
	cancelar context.CancelFunc
	lecturas int
}

func (l *lectorAgregadosTest) LeerSnapshotEnsayoAgregadosIncidencias(context.Context) (ports.SnapshotEnsayoAgregadosIncidencias, error) {
	l.lecturas++
	if l.cancelar != nil {
		l.cancelar()
	}
	return l.snapshot, l.err
}
func snapshotAgregadosTest(t *testing.T) ports.SnapshotEnsayoAgregadosIncidencias {
	t.Helper()
	b, err := os.ReadFile("../../../../data/demo/cronos/agregados-incidencias-periodo.json")
	if err != nil {
		t.Fatal(err)
	}
	var s ports.SnapshotEnsayoAgregadosIncidencias
	if err = json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	s.SHA256 = hex.EncodeToString(h[:])
	return s
}

func TestEnsayoAgregadosCoberturaYMinimizacion(t *testing.T) {
	s := snapshotAgregadosTest(t)
	l := &lectorAgregadosTest{snapshot: s}
	r, err := EnsayarAgregadosIncidencias(context.Background(), l)
	if err != nil || l.lecturas != 1 || r.Agregado.Estado != "incompleto" || r.Agregado.TotalIncidencias != nil || r.Agregado.IncidenciasObservadas != 3 || r.Agregado.PersonasDiasCompletos != 5 || r.Agregado.PersonasDiasIncompletos != 1 || r.Agregado.PersonasAfectadasObservadas != 2 || r.Agregado.DiasAfectadosObservados != 2 || r.Agregado.PersonasDiasAfectadosObservados != 2 {
		t.Fatalf("result: %+v %v", r, err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range append(append([]string{}, s.PersonasRef...), s.Incidencias[0].Referencia) {
		if strings.Contains(string(b), ref) {
			t.Fatal("individual reference escaped")
		}
	}
	if r.SnapshotSHA256 != s.SHA256 || r.CatalogoSHA256 == "" || r.Fuentes[0].SHA256 != s.Fuentes[0].SHA256 {
		t.Fatal("provenance lost")
	}
	s.Cobertura = s.Cobertura[:len(s.Cobertura)-1]
	s.Incidencias = []ports.HechoEnsayoAgregadosIncidencias{}
	r, err = EnsayarAgregadosIncidencias(context.Background(), &lectorAgregadosTest{snapshot: s})
	if err != nil || r.Agregado.TotalIncidencias != nil || r.Agregado.PersonasDiasDesconocidos != 1 {
		t.Fatalf("missing coverage: %+v %v", r, err)
	}
	s.Fuentes[0].Estado = "no_disponible"
	r, err = EnsayarAgregadosIncidencias(context.Background(), &lectorAgregadosTest{snapshot: s})
	if err != nil || r.Agregado.Estado != "desconocido" || r.Agregado.TotalIncidencias != nil || r.Agregado.PersonasDiasDesconocidos != 6 {
		t.Fatalf("source down: %+v %v", r, err)
	}
	s = snapshotAgregadosTest(t)
	s.Cobertura[len(s.Cobertura)-1].Estado = domain.CoberturaIncidenciasCompleta
	s.Incidencias = []ports.HechoEnsayoAgregadosIncidencias{}
	r, err = EnsayarAgregadosIncidencias(context.Background(), &lectorAgregadosTest{snapshot: s})
	if err != nil || r.Agregado.TotalIncidencias == nil || *r.Agregado.TotalIncidencias != 0 {
		t.Fatalf("known zero: %+v %v", r, err)
	}
}

func TestEnsayoAgregadosFuentesAcumuladas(t *testing.T) {
	s := snapshotAgregadosTest(t)
	f := s.Fuentes[0]
	f.Referencia = "demo:fuente:segunda"
	s.Fuentes = append(s.Fuentes, f)
	r, err := EnsayarAgregadosIncidencias(context.Background(), &lectorAgregadosTest{snapshot: s})
	if err != nil || r.Agregado.PersonasDiasDesconocidos != 6 || r.Agregado.TotalIncidencias != nil {
		t.Fatalf("second source missing: %+v %v", r, err)
	}
	for _, c := range append([]ports.CoberturaEnsayoAgregadosIncidencias{}, s.Cobertura...) {
		c.FuenteRef = f.Referencia
		s.Cobertura = append(s.Cobertura, c)
	}
	r, err = EnsayarAgregadosIncidencias(context.Background(), &lectorAgregadosTest{snapshot: s})
	if err != nil || r.Agregado.PersonasDiasCompletos != 5 || r.Agregado.PersonasDiasIncompletos != 1 || r.Agregado.IncidenciasObservadas != 3 {
		t.Fatalf("coverage is person-day not source rows: %+v %v", r, err)
	}
}

func TestEnsayoAgregadosRechazaEntradas(t *testing.T) {
	casos := map[string]func(*ports.SnapshotEnsayoAgregadosIncidencias){
		"empty":            func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.PersonasRef = nil },
		"wildcard":         func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.PersonasRef[0] = "*" },
		"duplicate_person": func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.PersonasRef[1] = s.PersonasRef[0] },
		"real_ref":         func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.PersonasRef[0] = "persona:1" },
		"not_demo":         func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Demo = false },
		"source_empty":     func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Fuentes = nil },
		"source_duplicate": func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Fuentes = append(s.Fuentes, s.Fuentes[0]) },
		"event_duplicate": func(s *ports.SnapshotEnsayoAgregadosIncidencias) {
			s.Incidencias = append(s.Incidencias, s.Incidencias[0])
		},
		"coverage_duplicate": func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Cobertura = append(s.Cobertura, s.Cobertura[0]) },
		"coverage_null":      func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Cobertura = nil },
		"events_null":        func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Incidencias = nil },
		"unknown_source":     func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Incidencias[0].FuenteRef = "demo:fuente:otra" },
		"source_version":     func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Incidencias[0].FuenteVersion++ },
		"coverage_version":   func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Cobertura[0].FuenteVersion++ },
		"outside_selection":  func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Incidencias[0].PersonaRef = "demo:persona:otra" },
		"end_excluded":       func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Incidencias[0].Fecha = s.HastaExclusiva },
		"before_start":       func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Incidencias[0].Fecha = "2026-09-27" },
		"after_cutoff": func(s *ports.SnapshotEnsayoAgregadosIncidencias) {
			s.Incidencias[0].RegistradaUTC = s.InstanteCorteUTC.Add(time.Microsecond)
		},
		"before_day": func(s *ports.SnapshotEnsayoAgregadosIncidencias) {
			s.Incidencias[0].RegistradaUTC = s.Incidencias[0].RegistradaUTC.Add(-24 * time.Hour)
		},
		"future_period": func(s *ports.SnapshotEnsayoAgregadosIncidencias) {
			s.InstanteCorteUTC = s.InstanteCorteUTC.Add(-24 * time.Hour)
		},
		"unknown_code":  func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Incidencias[0].Codigo = "absentismo" },
		"unknown_state": func(s *ports.SnapshotEnsayoAgregadosIncidencias) { s.Incidencias[0].Estado = "aprobada" },
		"catalogue_open": func(s *ports.SnapshotEnsayoAgregadosIncidencias) {
			s.Catalogo.Codigos = append(s.Catalogo.Codigos, "salud")
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			s := snapshotAgregadosTest(t)
			mutar(&s)
			r, err := EnsayarAgregadosIncidencias(context.Background(), &lectorAgregadosTest{snapshot: s})
			if err == nil || r.Demo {
				t.Fatalf("accepted invalid input: %+v", r)
			}
		})
	}
}

func TestEnsayoAgregadosCancelacionYErrores(t *testing.T) {
	s := snapshotAgregadosTest(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	l := &lectorAgregadosTest{snapshot: s}
	if _, err := EnsayarAgregadosIncidencias(ctx, l); !errors.Is(err, context.Canceled) || l.lecturas != 0 {
		t.Fatal("cancelled read")
	}
	ctx, cancelar = context.WithCancel(context.Background())
	defer cancelar()
	l = &lectorAgregadosTest{snapshot: s, cancelar: cancelar}
	if _, err := EnsayarAgregadosIncidencias(ctx, l); !errors.Is(err, context.Canceled) {
		t.Fatal("cancel after source read")
	}
	for _, l := range []ports.LectorSnapshotEnsayoAgregadosIncidencias{nil, (*lectorAgregadosTest)(nil), &lectorAgregadosTest{err: os.ErrPermission}} {
		r, err := EnsayarAgregadosIncidencias(context.Background(), l)
		if err == nil || r.Demo {
			t.Fatal("source error must not produce zero")
		}
	}
	s.Incidencias[0].RegistradaUTC = s.InstanteCorteUTC
	if _, err := EnsayarAgregadosIncidencias(context.Background(), &lectorAgregadosTest{snapshot: s}); err != nil {
		t.Fatal("cutoff is inclusive", err)
	}
}
