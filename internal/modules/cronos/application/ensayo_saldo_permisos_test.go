package application_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type lectorEnsayo struct {
	s        ports.SnapshotEnsayoSaldoPermisos
	llamadas int
}

func (l *lectorEnsayo) LeerSnapshotSaldoPermisos(context.Context) (ports.SnapshotEnsayoSaldoPermisos, error) {
	l.llamadas++
	return l.s, nil
}

func TestEnsayoExigeSnapshotUnicoYPoliticaExacta(t *testing.T) {
	b, err := os.ReadFile("../../../../data/demo/cronos/escenarios-saldo.json")
	if err != nil {
		t.Fatal(err)
	}
	var s ports.SnapshotEnsayoSaldoPermisos
	if err = json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	s.SHA256 = hex.EncodeToString(h[:])
	b, err = os.ReadFile("../../../../data/demo/reglas/cronos-efectos-permisos.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := catalogoefectos.Cargar(b, s.PoliticaSHA256)
	if err != nil {
		t.Fatal(err)
	}
	l := &lectorEnsayo{s: s}
	r, err := application.EnsayarSaldoPermisos(context.Background(), l, c.Politica)
	if err != nil || l.llamadas != 1 || !r.Demostracion || r.EscenariosSHA256 != s.SHA256 || r.PoliticaSHA256 != c.Politica.SHA256 {
		t.Fatalf("snapshot: %+v %v", r, err)
	}
	for nombre, mutar := range map[string]func(*ports.SnapshotEnsayoSaldoPermisos){
		"version":             func(s *ports.SnapshotEnsayoSaldoPermisos) { s.PoliticaVersion++ },
		"politica":            func(s *ports.SnapshotEnsayoSaldoPermisos) { s.PoliticaRef += ":otra" },
		"huella":              func(s *ports.SnapshotEnsayoSaldoPermisos) { s.PoliticaSHA256 = s.SHA256 },
		"sin_demo":            func(s *ports.SnapshotEnsayoSaldoPermisos) { s.Demostracion = false },
		"escenario_duplicado": func(s *ports.SnapshotEnsayoSaldoPermisos) { s.Escenarios = append(s.Escenarios, s.Escenarios[0]) },
		"fecha_duplicada": func(s *ports.SnapshotEnsayoSaldoPermisos) {
			s.Escenarios = append([]ports.EscenarioSaldoPermisos{}, s.Escenarios...)
			s.Escenarios[0].Dias = append(s.Escenarios[0].Dias, s.Escenarios[0].Dias[0])
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			copia := s
			mutar(&copia)
			l := &lectorEnsayo{s: copia}
			r, err := application.EnsayarSaldoPermisos(context.Background(), l, c.Politica)
			if err == nil || len(r.Escenarios) != 0 || l.llamadas != 1 {
				t.Fatal("ensayo incompatible publicado")
			}
		})
	}
}
