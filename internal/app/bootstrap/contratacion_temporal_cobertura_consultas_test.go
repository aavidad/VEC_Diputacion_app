package bootstrap

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La propuesta de cobertura solo necesita recuentos por estado de la bolsa de
// su categoría. Antes cargaba todas las bolsas con todo su detalle (orden,
// llamamientos, entradas, acta descifrada, situaciones y ceses de cada una);
// ahora usa el resumen del cuadro de Bolsa: con B82/B85, una lectura de
// conjunto, sin descifrar el acta, y la misma situación para cada categoría.
func TestCoberturaCTLeeBolsaConUnaLecturaDeConjunto(t *testing.T) {
	const bolsas, participaciones = 20, 40
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	completa, antes := fuenteCoberturaPrueba(t, bolsas, participaciones)
	f, despues := fuenteCoberturaPrueba(t, bolsas, participaciones)
	var conjunto, emisiones atomic.Int64
	f.resumenConjunto = lectorResumenPrueba{repo: f.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &conjunto}
	f.emisiones = emisionesContadasResumenPrueba{llamadas: &emisiones}
	f.repositorio = nil
	nueva := situacionBolsaCoberturaDesarrollo{fuente: f, ahora: func() time.Time { return ahora }}
	for b := 1; b <= bolsas; b++ {
		categoria := fmt.Sprintf("categoria:rpt:prueba-%02d", b)
		// Lectura anterior: la carga completa de todas las bolsas.
		datos, err := completa.cargar(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		esperada, errEsperada := resumirSituacionBolsaCobertura(datos, categoria, ahora)
		obtenida, errObtenida := nueva.SituacionBolsaCobertura(context.Background(), categoria)
		if errEsperada != nil || errObtenida != nil || !reflect.DeepEqual(esperada, obtenida) || !obtenida.Existe || obtenida.Integrantes != participaciones {
			t.Fatalf("%s: antes %+v %v; ahora %+v %v", categoria, esperada, errEsperada, obtenida, errObtenida)
		}
	}
	if antes.orden.Load() != bolsas*bolsas || antes.recuperar.Load() != bolsas*bolsas || antes.situacionLote.Load() != bolsas*bolsas {
		t.Fatalf("la carga anterior no leyó cada bolsa: orden=%d actas=%d situaciones=%d", antes.orden.Load(), antes.recuperar.Load(), antes.situacionLote.Load())
	}
	// Una lectura de conjunto por propuesta y ninguna lectura por bolsa.
	if conjunto.Load() != bolsas || emisiones.Load() != 0 || despues.orden.Load() != 0 || despues.recuperar.Load() != 0 ||
		despues.situacionLote.Load() != 0 || despues.ceseLote.Load() != 0 || despues.situacionIndividual.Load() != 0 || despues.ceseIndividual.Load() != 0 {
		t.Fatalf("lecturas por propuesta: conjunto=%d/%d llamamientos=%d orden=%d actas=%d situaciones=%d ceses=%d", conjunto.Load(), bolsas,
			emisiones.Load(), despues.orden.Load(), despues.recuperar.Load(), despues.situacionLote.Load(), despues.ceseLote.Load())
	}
}

// Sin B82/B85 la cobertura sigue sin descifrar el acta y da la misma
// situación.
func TestCoberturaCTSinResumenConjuntoNoDescifraActa(t *testing.T) {
	const bolsas, participaciones = 5, 30
	ahora := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	completa, _ := fuenteCoberturaPrueba(t, bolsas, participaciones)
	datos, err := completa.cargar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f, c := fuenteCoberturaPrueba(t, bolsas, participaciones)
	esperada, _ := resumirSituacionBolsaCobertura(datos, "categoria:rpt:prueba-03", ahora)
	obtenida, err := situacionBolsaCoberturaDesarrollo{fuente: f, ahora: func() time.Time { return ahora }}.SituacionBolsaCobertura(context.Background(), "categoria:rpt:prueba-03")
	if err != nil || !reflect.DeepEqual(esperada, obtenida) {
		t.Fatalf("antes %+v; ahora %+v %v", esperada, obtenida, err)
	}
	if c.recuperar.Load() != 0 {
		t.Fatalf("la cobertura descifró %d actas", c.recuperar.Load())
	}
}

// Quien espera la lectura de otra petición de la misma categoría deja de
// esperar si su petición se cancela.
func TestCoberturaCTEsperaCancelable(t *testing.T) {
	bloqueo := make(chan struct{})
	consulta := &situacionBolsaCoberturaBloqueadaPrueba{bloqueo: bloqueo, entrada: make(chan struct{})}
	fijable := &situacionBolsaCoberturaFijable{}
	fijable.fijar(consulta)
	hecho := make(chan struct{})
	go func() {
		defer close(hecho)
		_, _ = fijable.SituacionBolsaCobertura(context.Background(), "categoria:rpt:prueba-01")
	}()
	<-consulta.entrada
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, ok := fijable.situacion(cancelado, "categoria:rpt:prueba-01"); ok {
		t.Fatal("una petición cancelada obtuvo situación")
	}
	close(bloqueo)
	<-hecho
}

type situacionBolsaCoberturaBloqueadaPrueba struct {
	bloqueo <-chan struct{}
	entrada chan struct{}
}

func (s *situacionBolsaCoberturaBloqueadaPrueba) SituacionBolsaCobertura(context.Context, string) (ctports.SituacionBolsaCobertura, error) {
	close(s.entrada)
	<-s.bloqueo
	return ctports.SituacionBolsaCobertura{}, ctports.ErrSituacionBolsaCoberturaNoDisponible
}

// fuenteCoberturaPrueba fija la fecha de constitución, que la cobertura exige.
func fuenteCoberturaPrueba(t *testing.T, bolsas, participaciones int) (*fuenteConstituidaRRHHDesarrollo, *contadoresBolsasRRHHPrueba) {
	t.Helper()
	f, c := fuenteVariasBolsasRRHHPrueba(t, bolsas, participaciones, true)
	repo := f.repositorio.(repositorioVariasBolsasRRHHPrueba)
	for i := range repo.vigentes {
		repo.vigentes[i].ConfirmadaEn = time.Date(2026, 3, 1+i, 9, 0, 0, 0, time.UTC)
	}
	return f, c
}
