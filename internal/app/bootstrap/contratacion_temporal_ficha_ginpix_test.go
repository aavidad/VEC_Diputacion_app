package bootstrap

import (
	"context"
	"reflect"
	"testing"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestFichaGINPIXV2ContextoConservaScopeNominal(t *testing.T) {
	alta, _, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	for _, ruta := range []string{httpinterno.RutaFichaGINPIXV2, httpinterno.RutaIncorporacionEjercicioV2} {
		ctx := contextoRutaCoberturaDesarrolloPrueba(alta.soporte, principal, ruta)
		antes := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		hijo, err := contextoDetalleIncorporacionV2Desarrollo(ctx, alta.soporte)
		if err != nil {
			t.Fatal(err)
		}
		despues := hijo.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		if !reflect.DeepEqual(antes.principal, despues.principal) || antes.sello != despues.sello || despues.ruta != httpinterno.RutaConsultaDetalleRRHH || hijo.Value(claveIncorporacionV2Desarrollo{}) != alta.soporte.sello {
			t.Fatal("contexto nominal alterado")
		}
	}
	otro := contextoRutaCoberturaDesarrolloPrueba(alta.soporte, principal, httpinterno.RutaAltaSolicitudes)
	if _, err := contextoDetalleIncorporacionV2Desarrollo(otro, alta.soporte); err == nil {
		t.Fatal("ruta ajena aceptada")
	}
}
func TestFichaGINPIXV2MapeoEjercicioInmutable(t *testing.T) {
	a, err := nuevoMapeoFichaGINPIXDesarrollo()
	if err != nil {
		t.Fatal(err)
	}
	b, err := nuevoMapeoFichaGINPIXDesarrollo()
	if err != nil {
		t.Fatal(err)
	}
	if a.m.Publicacion().HuellaSHA256 != b.m.Publicacion().HuellaSHA256 || len(a.m.Publicacion().Reglas) != 6 {
		t.Fatal("mapeo no determinista")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := a.ResolverMapeoFichaGINPIXV2(ctx, ct.ReciboIncorporacionAplicacionV2{}); err == nil {
		t.Fatal("cancelacion ignorada")
	}
}
