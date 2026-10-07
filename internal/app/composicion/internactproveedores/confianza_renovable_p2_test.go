package internactproveedores

import (
	"context"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/composicion/gobiernov3lector"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestEmisorMaterialRenovablePermiteLecturasConcurrentes(t *testing.T) {
	dia := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
	raiz, _ := raizB2(t, 7, dia)
	publicacion := publicacionB2(t, raiz, dia, 20260925)
	reloj := &relojB2{ahora: dia.Add(time.Hour)}
	entro := make(chan struct{}, 2)
	liberar := make(chan struct{})
	lector, err := gobiernov3lector.Nuevo(publicacion, raiz, reloj, func(context.Context, gobiernov3lector.Publicacion) (gobiernov3lector.Publicacion, error) {
		entro <- struct{}{}
		<-liberar
		return publicacion, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	emisor := &emisorMaterialRenovable{lector: lector}
	var grupo sync.WaitGroup
	for range 2 {
		grupo.Go(func() {
			// Tras la instantánea, las dependencias nulas fuerzan denegación.
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
