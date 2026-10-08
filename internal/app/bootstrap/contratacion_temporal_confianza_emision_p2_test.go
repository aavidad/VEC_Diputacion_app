package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
)

// El lector sintético mantiene la primera lectura abierta. La segunda debe
// entrar antes de que se libere la primera: la emisión no comparte su cerrojo.
func TestEmisorRenovableCTPermiteLecturasConcurrentes(t *testing.T) {
	m := materialRenovableCTPrueba(t, time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC))
	r := &relojRenovableCTPrueba{}
	r.fijar(m.publicadaEn.Add(time.Hour))
	f := fuenteRenovableCTPrueba(t, m, r)
	entro := make(chan struct{}, 2)
	liberar := make(chan struct{})
	f.leer = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		entro <- struct{}{}
		<-liberar
		return m, nil
	}
	f.renovar = func(context.Context, materialAtestacionContratacionTemporalDesarrollo, time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
		t.Error("renovación inesperada")
		return materialAtestacionContratacionTemporalDesarrollo{}, nil
	}
	emisor := &emisorMaterialRenovableCTDesarrollo{proveedor: &proveedorMaterialAltaContratacionTemporalDesarrollo{
		confianza: f.actual, fuenteConfianza: f,
	}}
	var grupo sync.WaitGroup
	for range 2 {
		grupo.Go(func() {
			// Las dependencias nominales son intencionadamente nulas: solo se
			// mide la lectura previa y la emisión termina denegada.
			_, _, _, _ = emisor.EmitirMaterialAutorizacionAtestadaV3(context.Background(), core.SolicitudAutorizacionLigadaV3{}, core.ResultadoContextoActorRegistradoV2{})
		})
	}
	<-entro
	segundo := false
	select {
	case <-entro:
		segundo = true
	case <-time.After(time.Second):
	}
	close(liberar)
	grupo.Wait()
	if !segundo {
		t.Fatal("lecturas simultáneas antes de liberar: 1/2")
	}
	t.Log("lecturas simultáneas antes de liberar: 2/2")
}
