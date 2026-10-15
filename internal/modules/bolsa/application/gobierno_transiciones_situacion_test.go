package application

import (
	"context"
	"slices"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type repositorioGobiernoTransicionesPrueba struct {
	valor ports.GobiernoTransicionesSituacion
}

func (r repositorioGobiernoTransicionesPrueba) Leer(context.Context, int, string) (ports.GobiernoTransicionesSituacion, error) {
	return r.valor, nil
}

type repositorioReplayNormaCambiadaPrueba struct {
	actual   ports.GobiernoTransicionesSituacion
	original ports.GobiernoTransicionesSituacion
	llamadas int
}

func (r *repositorioReplayNormaCambiadaPrueba) Leer(context.Context, int, string) (ports.GobiernoTransicionesSituacion, error) {
	return r.actual, nil
}
func (r *repositorioReplayNormaCambiadaPrueba) Publicar(_ context.Context, c ports.ComandoGobiernoTransicionesSituacion) (ports.GobiernoTransicionesSituacion, error) {
	r.llamadas++
	if !slices.Equal(c.Transiciones, r.original.TransicionesGuardadas) || c.VersionEsperada != 1 || c.ClaveIdempotencia != "clave-original" {
		return ports.GobiernoTransicionesSituacion{}, ports.ErrGobiernoTransicionesConflicto
	}
	return r.original, nil
}
func (r repositorioGobiernoTransicionesPrueba) Publicar(context.Context, ports.ComandoGobiernoTransicionesSituacion) (ports.GobiernoTransicionesSituacion, error) {
	return r.valor, nil
}

type reglaGobiernoTransicionesPrueba struct{}

func (reglaGobiernoTransicionesPrueba) DestinosSituacion(_ context.Context, origen string) ([]string, bool, error) {
	if origen == domain.SituacionRenuncia {
		return []string{domain.SituacionExcluido}, true, nil
	}
	return nil, false, nil
}

func TestGobiernoTransicionesMuestraLaRestriccionOperativaReal(t *testing.T) {
	situacion, repo, _ := servicioConPoliticaPrueba(t)
	politica := politicaReglamentoPrueba(t)
	repo.vigente = ports.PoliticaTransicionesVigente{Version: 2, Politica: politica}
	situacion.EstablecerReglasTransiciones(reglaGobiernoTransicionesPrueba{})
	sha := strings.Repeat("a", 64)
	gobierno, err := NuevoServicioGobiernoTransicionesSituacion(repositorioGobiernoTransicionesPrueba{
		valor: ports.GobiernoTransicionesSituacion{Version: 2, TransicionesGuardadas: politica.Pares(),
			ReciboRef: "recibo:previo", Reutilizada: true,
			Limites: ports.LimitesTransicionesSituacion{CatalogoRef: "bolsa.transiciones_situacion:1",
				CatalogoVersion: 1, CatalogoSHA256: sha, Transiciones: politica.Pares()}},
	}, situacion, 1, sha)
	if err != nil {
		t.Fatal(err)
	}
	v, err := gobierno.Consultar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(v.TransicionesGuardadas, "renuncia>no_disponible") ||
		slices.Contains(v.TransicionesEfectivas, "renuncia>no_disponible") ||
		!slices.Contains(v.TransicionesEfectivas, "renuncia>excluido") || v.Version != 2 ||
		v.ReciboRef != "" || v.Reutilizada {
		t.Fatalf("guardadas/efectivas no reflejan el servicio: %+v", v)
	}
}

func TestReplayNoSeFiltraPorLimitesNuevosAntesDelRepositorio(t *testing.T) {
	situacion, repo, _ := servicioConPoliticaPrueba(t)
	politicaA := politicaReglamentoPrueba(t)
	repo.vigente = ports.PoliticaTransicionesVigente{Version: 2, Politica: politicaA}
	shaA, shaB := strings.Repeat("a", 64), strings.Repeat("b", 64)
	politicaB := politicaA.Restringir(domain.SituacionRenuncia, []string{domain.SituacionExcluido})
	r := &repositorioReplayNormaCambiadaPrueba{
		actual: ports.GobiernoTransicionesSituacion{Version: 2, TransicionesGuardadas: politicaA.Pares(),
			Limites: ports.LimitesTransicionesSituacion{CatalogoVersion: 2, CatalogoSHA256: shaB, Transiciones: politicaB.Pares()}},
		original: ports.GobiernoTransicionesSituacion{Version: 2, TransicionesGuardadas: politicaA.Pares(),
			Limites:   ports.LimitesTransicionesSituacion{CatalogoVersion: 1, CatalogoSHA256: shaA, Transiciones: politicaA.Pares()},
			ReciboRef: "recibo:original", Reutilizada: true},
	}
	servicio, err := NuevoServicioGobiernoTransicionesSituacion(r, situacion, 2, shaB)
	if err != nil {
		t.Fatal(err)
	}
	v, err := servicio.Publicar(context.Background(), ports.ComandoGobiernoTransicionesSituacion{
		VersionEsperada: 1, Transiciones: politicaA.Pares(), Motivo: "mismo motivo", ClaveIdempotencia: "clave-original"})
	if err != nil || r.llamadas != 1 || !v.Reutilizada || v.ReciboRef != "recibo:original" || v.Limites.CatalogoSHA256 != shaA {
		t.Fatalf("el replay original se bloqueó por catálogo B: %+v %v llamadas=%d", v, err, r.llamadas)
	}
	divergente := politicaB.Pares()
	_, err = servicio.Publicar(context.Background(), ports.ComandoGobiernoTransicionesSituacion{
		VersionEsperada: 1, Transiciones: divergente, Motivo: "mismo motivo", ClaveIdempotencia: "clave-original"})
	if err != ports.ErrGobiernoTransicionesConflicto || r.llamadas != 2 {
		t.Fatalf("cuerpo divergente no conservó conflicto: %v llamadas=%d", err, r.llamadas)
	}
}
