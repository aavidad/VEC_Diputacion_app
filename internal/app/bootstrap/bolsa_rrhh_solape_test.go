package bootstrap

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	importaciondominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type recuperadorSolapadoRRHHPrueba struct {
	base      recuperadorContadoRRHHPrueba
	entro     chan struct{}
	liberar   <-chan struct{}
	cancelado chan struct{}
}

func (r recuperadorSolapadoRRHHPrueba) RecuperarLote(ctx context.Context, huella, categoria string) (importaciondominio.LoteValidado, importacionapp.EstadoImportacion, bool, error) {
	close(r.entro)
	select {
	case <-r.liberar:
		return r.base.RecuperarLote(ctx, huella, categoria)
	case <-ctx.Done():
		if r.cancelado != nil {
			close(r.cancelado)
		}
		return importaciondominio.LoteValidado{}, importacionapp.EstadoImportacion{}, false, ctx.Err()
	}
}

type lectorSolapadoRRHHPrueba struct {
	base    lectorBolsaConjuntoPrueba
	entro   chan struct{}
	liberar <-chan struct{}
	fallo   error
}

func (l lectorSolapadoRRHHPrueba) LeerBolsa(ctx context.Context, bolsa string, corte time.Time) (dominiobolsa.OrdenVigenteBolsa, []ports.SituacionBolsaRRHH, int, error) {
	close(l.entro)
	if l.fallo != nil {
		return dominiobolsa.OrdenVigenteBolsa{}, nil, 0, l.fallo
	}
	select {
	case <-l.liberar:
		return l.base.LeerBolsa(ctx, bolsa, corte)
	case <-ctx.Done():
		return dominiobolsa.OrdenVigenteBolsa{}, nil, 0, ctx.Err()
	}
}

func esperarInicioSolapeRRHHPrueba(t *testing.T, inicio <-chan struct{}) {
	t.Helper()
	select {
	case <-inicio:
	case <-time.After(5 * time.Second):
		t.Fatal("la segunda lectura no comenzó mientras la primera esperaba")
	}
}

func TestBolsaRRHHRecuperacionProtegidaYConjuntoSeSolapan(t *testing.T) {
	f, c := fuenteVariasBolsasRRHHPrueba(t, 2, 50, true)
	repo := f.repositorio.(repositorioVariasBolsasRRHHPrueba)
	liberar := make(chan struct{})
	inicioActa, inicioConjunto := make(chan struct{}), make(chan struct{})
	f.recuperador = recuperadorSolapadoRRHHPrueba{base: recuperadorContadoRRHHPrueba{c: c}, entro: inicioActa, liberar: liberar}
	f.bolsaConjunto = lectorSolapadoRRHHPrueba{base: lectorBolsaConjuntoPrueba{repo: repo, lecturas: &atomic.Int64{}}, entro: inicioConjunto, liberar: liberar}
	resultado := make(chan error, 1)
	go func() {
		_, err := f.cargarBolsaRRHH(context.Background(), repo.vigentes[0].Bolsa.BolsaRef)
		resultado <- err
	}()
	esperarInicioSolapeRRHHPrueba(t, inicioActa)
	esperarInicioSolapeRRHHPrueba(t, inicioConjunto)
	close(liberar)
	select {
	case err := <-resultado:
		if err != nil || c.vigentesGlobales.Load() != 0 || c.vigentesFiltradas.Load() != 1 || c.recuperar.Load() != 1 {
			t.Fatalf("lectura solapada: error=%v globales=%d filtradas=%d actas=%d", err, c.vigentesGlobales.Load(), c.vigentesFiltradas.Load(), c.recuperar.Load())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("la lectura solapada no terminó")
	}
}

func TestBolsaRRHHFalloDelConjuntoCancelaYEsperaElActa(t *testing.T) {
	f, c := fuenteVariasBolsasRRHHPrueba(t, 1, 2, true)
	repo := f.repositorio.(repositorioVariasBolsasRRHHPrueba)
	inicioActa, inicioConjunto, cancelado := make(chan struct{}), make(chan struct{}), make(chan struct{})
	f.recuperador = recuperadorSolapadoRRHHPrueba{base: recuperadorContadoRRHHPrueba{c: c}, entro: inicioActa, cancelado: cancelado}
	fallo := errors.New("fallo fijo de lector de prueba")
	f.bolsaConjunto = lectorSolapadoRRHHPrueba{entro: inicioConjunto, fallo: fallo}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	_, err := f.cargarBolsaRRHH(ctx, repo.vigentes[0].Bolsa.BolsaRef)
	if !errors.Is(err, fallo) {
		t.Fatalf("error del lector no propagado: %v", err)
	}
	esperarInicioSolapeRRHHPrueba(t, inicioActa)
	esperarInicioSolapeRRHHPrueba(t, inicioConjunto)
	esperarInicioSolapeRRHHPrueba(t, cancelado)
}
