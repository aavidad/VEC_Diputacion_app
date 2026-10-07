package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestCandidatosRRHHConResumenConjuntoEvitaLecturasPorParticipacion(t *testing.T) {
	const bolsa = "bolsa:prueba:02"
	anterior, _ := fuenteVariasBolsasRRHHPrueba(t, 2, 200, true)
	esperado, err := anterior.cargarBolsa(context.Background(), bolsa)
	if err != nil {
		t.Fatal(err)
	}
	f, contadores := fuenteVariasBolsasRRHHPrueba(t, 2, 200, true)
	var lecturas atomic.Int64
	f.resumenConjunto = lectorResumenPrueba{repo: f.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &lecturas}
	obtenido, err := f.cargarBolsa(context.Background(), bolsa)
	if err != nil {
		t.Fatal(err)
	}
	if lecturas.Load() != 1 || contadores.situacionLote.Load() != 0 || contadores.ceseLote.Load() != 0 ||
		contadores.situacionIndividual.Load() != 0 || contadores.ceseIndividual.Load() != 0 ||
		contadores.orden.Load() != 1 || contadores.recuperar.Load() != 1 {
		t.Fatalf("lecturas: conjunto=%d situacion=%d/%d cese=%d/%d orden=%d acta=%d", lecturas.Load(),
			contadores.situacionLote.Load(), contadores.situacionIndividual.Load(),
			contadores.ceseLote.Load(), contadores.ceseIndividual.Load(),
			contadores.orden.Load(), contadores.recuperar.Load())
	}
	a, _ := json.Marshal(esperado)
	b, _ := json.Marshal(obtenido)
	if string(a) != string(b) {
		t.Fatal("la lectura de conjunto cambió la lista de candidatos")
	}
}

func TestCandidatosRRHHConResumenConjuntoConservaBolsaNoEncontrada(t *testing.T) {
	f, _ := fuenteVariasBolsasRRHHPrueba(t, 1, 3, true)
	var lecturas atomic.Int64
	f.resumenConjunto = lectorResumenPrueba{repo: f.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &lecturas}
	datos, err := f.cargarBolsa(context.Background(), "bolsa:ausente")
	if err != nil || len(datos.Bolsas) != 0 || len(datos.Candidaturas) != 0 {
		t.Fatalf("bolsa ausente: bolsas=%d candidaturas=%d error=%v", len(datos.Bolsas), len(datos.Candidaturas), err)
	}
}

type lectorResumenBolsaAlterado struct {
	lectorResumenPrueba
	alterar func([]ports.SituacionResumenParticipacion) []ports.SituacionResumenParticipacion
}

func (l lectorResumenBolsaAlterado) LeerResumen(ctx context.Context, corte time.Time) ([]ports.SituacionResumenParticipacion, map[string]bolsadominio.PoliticaOrdenBolsa, error) {
	filas, politicas, err := l.lectorResumenPrueba.LeerResumen(ctx, corte)
	if err != nil {
		return nil, nil, err
	}
	return l.alterar(filas), politicas, nil
}

func TestCandidatosRRHHConResumenConjuntoFallaCerradoAnteInstantaneaIncoherente(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		alterar func([]ports.SituacionResumenParticipacion) []ports.SituacionResumenParticipacion
	}{
		{"fila_ausente", func(filas []ports.SituacionResumenParticipacion) []ports.SituacionResumenParticipacion {
			return filas[1:]
		}},
		{"orden_cambiado", func(filas []ports.SituacionResumenParticipacion) []ports.SituacionResumenParticipacion {
			filas[0].Orden++
			return filas
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			f, _ := fuenteVariasBolsasRRHHPrueba(t, 1, 3, true)
			var lecturas atomic.Int64
			f.resumenConjunto = lectorResumenBolsaAlterado{lectorResumenPrueba: lectorResumenPrueba{repo: f.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &lecturas}, alterar: caso.alterar}
			if _, err := f.cargarBolsa(context.Background(), "bolsa:prueba:01"); !errors.Is(err, ErrComposicionDesarrolloIncompleta) {
				t.Fatalf("se sirvió instantánea incoherente: %v", err)
			}
		})
	}
}
