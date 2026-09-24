package bootstrap

import (
	"context"
	"testing"

	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type resolutorContextoActorAlcanceDietasPrueba struct{}

func (resolutorContextoActorAlcanceDietasPrueba) ResolverYRegistrarContextoActorV2(
	context.Context, vp.SolicitudResolucionRegistroContextoActorV2,
) (vp.ConfirmacionRegistroContextoActorV2, error) {
	return vp.ConfirmacionRegistroContextoActorV2{}, vp.ErrResolutorRegistroContextoActorNoDisponible
}

// Dietas declara {empleado}; la composición heredada (CT y resto) conserva el
// alcance vacío y, con él, los bytes del contexto y del manifiesto.
func TestComposicionContextoActorDietasDeclaraEmpleado(t *testing.T) {
	servicio, err := servicioContextoActorDietas(resolutorContextoActorAlcanceDietasPrueba{}, relojRutasDietas{})
	if err != nil || !servicio.Alcance().IncluyeEmpleado() {
		t.Fatalf("Dietas no pide el empleado canónico: %v", err)
	}
	heredado, err := vecapp.NuevoServicioContextoActorProductivoV2(
		resolutorContextoActorAlcanceDietasPrueba{},
		contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), relojRutasDietas{},
	)
	if err != nil || !heredado.Alcance().Vacio() {
		t.Fatalf("la composición heredada amplió el alcance: %v", err)
	}
	var nulo *vecapp.ServicioContextoActor
	if !nulo.Alcance().Vacio() {
		t.Fatal("un servicio nulo declara proyecciones")
	}
}
