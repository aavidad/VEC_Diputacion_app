package bootstrap

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type lectorBolsaConjuntoPrueba struct {
	repo     repositorioVariasBolsasRRHHPrueba
	lecturas *atomic.Int64
}

func (l lectorBolsaConjuntoPrueba) LeerBolsa(ctx context.Context, bolsa string, corte time.Time) (dominiobolsa.OrdenVigenteBolsa, []ports.SituacionResumenParticipacion, int, error) {
	l.lecturas.Add(1)
	orden, err := ordenContadoRRHHPrueba{c: &contadoresBolsasRRHHPrueba{}, entradas: l.repo.entradas}.ConsultarOrdenVigente(ctx, bolsa)
	if err != nil {
		return orden, nil, 0, err
	}
	resumen, err := (lectorResumenPrueba{repo: l.repo, lecturas: &atomic.Int64{}}).situaciones()
	if err != nil {
		return orden, nil, 0, err
	}
	filas := make([]ports.SituacionResumenParticipacion, 0, len(orden.Posiciones))
	for _, fila := range resumen {
		if fila.BolsaRef == bolsa {
			filas = append(filas, fila)
		}
	}
	return orden, filas, 0, nil
}

func TestFuenteConstituidaRRHHBolsaConjuntoConservaListaSinConsultasPorParticipacion(t *testing.T) {
	const bolsas, personasPorBolsa = 2, 200
	legado, _ := fuenteVariasBolsasRRHHPrueba(t, bolsas, personasPorBolsa, true)
	ref := "bolsa:prueba:01"
	esperado, err := legado.cargarBolsa(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	actual, contadores := fuenteVariasBolsasRRHHPrueba(t, bolsas, personasPorBolsa, true)
	var lecturas atomic.Int64
	actual.bolsaConjunto = lectorBolsaConjuntoPrueba{repo: actual.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &lecturas}
	obtenido, err := actual.cargarBolsa(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if lecturas.Load() != 1 || contadores.orden.Load() != 0 || contadores.situacionLote.Load() != 0 ||
		contadores.ceseLote.Load() != 0 || contadores.situacionIndividual.Load() != 0 || contadores.ceseIndividual.Load() != 0 {
		t.Fatalf("lecturas conjunto=%d orden=%d situaciones=%d/%d ceses=%d/%d", lecturas.Load(), contadores.orden.Load(),
			contadores.situacionLote.Load(), contadores.situacionIndividual.Load(), contadores.ceseLote.Load(), contadores.ceseIndividual.Load())
	}
	if fmt.Sprint(obtenido.Bolsas) != fmt.Sprint(esperado.Bolsas) || fmt.Sprint(obtenido.Candidaturas) != fmt.Sprint(esperado.Candidaturas) {
		t.Fatal("la lectura de conjunto cambió bolsas o candidaturas")
	}
}

func TestFuenteConstituidaRRHHBolsaConjuntoRechazaInstantaneaDivergente(t *testing.T) {
	f, _ := fuenteVariasBolsasRRHHPrueba(t, 1, 2, true)
	repo := f.repositorio.(repositorioVariasBolsasRRHHPrueba)
	lecturas := &atomic.Int64{}
	lector := lectorBolsaConjuntoPrueba{repo: repo, lecturas: lecturas}
	_, filas, _, err := lector.LeerBolsa(context.Background(), repo.vigentes[0].Bolsa.BolsaRef, f.ahora())
	if err != nil {
		t.Fatal(err)
	}
	filas[0].VersionInstantanea++
	if _, _, err := mapearSituacionesConjuntoBolsa(repo.vigentes[0], repo.entradas[repo.vigentes[0].Instantanea.InstantaneaRef], filas); err == nil {
		t.Fatal("se aceptó una situación de otra instantánea")
	}
}
