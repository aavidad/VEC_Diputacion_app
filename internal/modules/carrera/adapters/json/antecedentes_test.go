package json

import (
	"context"
	"os"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

func TestLectorAntecedentesConservaInstantaneaEntreConsultas(t *testing.T) {
	f, err := os.Open("../../../../../cmd/vec-carrera-preparar/testdata/antecedentes_expediente.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	l, err := LeerAntecedentes(f)
	if err != nil {
		t.Fatal(err)
	}
	q := ports.ConsultaAntecedentesSinteticos{CasoRef: "expediente-grado-1", VersionEsperada: "carrera-preparacion-1"}
	a, err := l.ConsultarAntecedentesSinteticos(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	a.Fuentes[0].Version = "otra"
	a.Grado.Valor = 99
	*a.Ocupaciones[0].Nivel = 99
	a.Servicios[0].Periodo.Inicio = "otra"
	a.Ocupaciones[0].Referencia = "otra"
	b, err := l.ConsultarAntecedentesSinteticos(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if b.Fuentes[0].Version != "ensayo-1" || b.Grado.Valor != 20 || *b.Ocupaciones[0].Nivel != 24 || b.Servicios[0].Periodo.Inicio != "2020-01-01" || b.Ocupaciones[0].Referencia != "ocupacion-sintetica" {
		t.Fatal("consulta modifica instantánea privada")
	}
}

func TestLectorRechazaInstantaneasDuplicadasYAusentes(t *testing.T) {
	i := `{"alcance":"preparacion_sintetica","caso_ref":"caso","version":"v1"}`
	for _, in := range []string{`{"instantaneas":[` + i + `,` + i + `]}`, `{"instantaneas":[]}`, `{"instantaneas":[{"alcance":"preparacion_sintetica","caso_ref":"caso","version":"v1","Version":"otra"}]}`} {
		if _, err := LeerAntecedentes(strings.NewReader(in)); err == nil {
			t.Fatal("admite instantánea ambigua")
		}
	}
}

func TestLectorGradoAusenteONuloNoFabricaCero(t *testing.T) {
	for _, grado := range []string{`{}`, `{"valor":null}`} {
		in := `{"instantaneas":[{"alcance":"preparacion_sintetica","caso_ref":"caso","version":"v1","grado":` + grado + `}]}`
		if _, err := LeerAntecedentes(strings.NewReader(in)); err == nil {
			t.Fatal("grado sin valor admitido como cero")
		}
	}
	in := `{"instantaneas":[{"alcance":"preparacion_sintetica","caso_ref":"caso","version":"v1","grado":{"valor":0}}]}`
	l, err := LeerAntecedentes(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	a, err := l.ConsultarAntecedentesSinteticos(context.Background(), ports.ConsultaAntecedentesSinteticos{CasoRef: "caso", VersionEsperada: "v1"})
	if err != nil || a.Grado == nil || a.Grado.Valor != 0 {
		t.Fatal("rechaza cero explícito")
	}
}
