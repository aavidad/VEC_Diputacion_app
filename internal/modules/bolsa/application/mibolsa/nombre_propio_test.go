package mibolsa

import (
	"context"
	"errors"
	"slices"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteNombrePrueba struct {
	nombre, apellidos string
	encontrado        bool
	err               error
	candidatos        []string
	bolsas            [][]string
}

func (f *fuenteNombrePrueba) NombrePropio(_ context.Context, candidato string, bolsas []string) (string, string, bool, error) {
	f.candidatos = append(f.candidatos, candidato)
	f.bolsas = append(f.bolsas, slices.Clone(bolsas))
	return f.nombre, f.apellidos, f.encontrado, f.err
}

func entornoConNombre(t *testing.T, fuente *fuenteNombrePrueba) *entornoMiBolsa {
	t.Helper()
	e := nuevoEntorno(t)
	var err error
	e.servicio, err = e.servicio.ConNombrePropio(fuente)
	exigir(t, err)
	return e
}

func TestNombrePropioSoloParaElCandidatoAutenticado(t *testing.T) {
	fuente := &fuenteNombrePrueba{nombre: "  Lucía ", apellidos: "Ortega   Pérez", encontrado: true}
	e := entornoConNombre(t, fuente)
	r, err := e.servicio.Consultar(context.Background(), e.orden)
	exigir(t, err)
	if len(fuente.candidatos) != 1 || fuente.candidatos[0] != e.repositorio.solicitud.CandidatoRef {
		t.Fatalf("el nombre se pidió para otro candidato: %v", fuente.candidatos)
	}
	var bolsas []string
	for _, p := range r.Participaciones {
		bolsas = append(bolsas, p.Bolsa)
	}
	if !slices.Equal(fuente.bolsas[0], bolsas) {
		t.Fatalf("el nombre se buscó fuera de sus bolsas: %v", fuente.bolsas[0])
	}
	if r.NombrePropio == nil || r.NombrePropio.Visible != "Lucía Ortega Pérez" || r.NombrePropio.Iniciales != "LO" {
		t.Fatalf("nombre inexacto: %+v", r.NombrePropio)
	}
}

func TestNombrePropioAusenteNoTumbaLaConsulta(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		fuente *fuenteNombrePrueba
	}{
		{"fallo técnico", &fuenteNombrePrueba{err: errors.New("descifrado")}},
		{"no atribuible", &fuenteNombrePrueba{}},
		{"sin nombre", &fuenteNombrePrueba{apellidos: "Ortega", encontrado: true}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := entornoConNombre(t, caso.fuente)
			r, err := e.servicio.Consultar(context.Background(), e.orden)
			if err != nil || len(r.Participaciones) != 1 || r.NombrePropio != nil {
				t.Fatalf("consulta alterada: %v %+v", err, r.NombrePropio)
			}
		})
	}
}

func TestNombrePropioNoSePideSiLaConsultaNoProspera(t *testing.T) {
	fuente := &fuenteNombrePrueba{nombre: "Lucía", encontrado: true}
	e := entornoConNombre(t, fuente)
	e.repositorio.err = ports.ErrFuenteAutorizacionNoDisponible
	if _, err := e.servicio.Consultar(context.Background(), e.orden); err == nil {
		t.Fatal("consulta fallida aceptada")
	}
	vacia := entornoConNombre(t, fuente)
	vacia.repositorio.mutar = func(r *bolsa.InstantaneaMiBolsa) { r.Participaciones = nil }
	r, err := vacia.servicio.Consultar(context.Background(), vacia.orden)
	if err != nil || r.NombrePropio != nil || len(fuente.candidatos) != 0 {
		t.Fatalf("se pidió el nombre sin consulta propia: %v %d", err, len(fuente.candidatos))
	}
	if _, err := nuevoEntorno(t).servicio.ConNombrePropio(nil); !errors.Is(err, ErrServicioMiBolsaInvalido) {
		t.Fatal("fuente nula aceptada")
	}
}
