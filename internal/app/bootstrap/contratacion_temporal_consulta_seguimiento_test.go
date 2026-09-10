package bootstrap

import (
	"net/http/httptest"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

func TestConsultaSeguimientoV2MantieneIdentidadNominal(t *testing.T) {
	alta, _, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	ctx := contextoRutaCoberturaDesarrolloPrueba(alta.soporte, principal, httpinterno.RutaConsultaSeguimientoV2)
	antes := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	hijo, err := contextoDetalleIncorporacionV2Desarrollo(ctx, alta.soporte)
	if err != nil {
		t.Fatal(err)
	}
	despues := hijo.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !reflect.DeepEqual(antes.principal, despues.principal) || antes.sello != despues.sello || despues.ruta != httpinterno.RutaConsultaDetalleRRHH || hijo.Value(claveIncorporacionV2Desarrollo{}) != alta.soporte.sello {
		t.Fatal("la consulta histórica cambió la identidad o no conservó el sello nominal")
	}
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest("GET", httpinterno.RutaConsultaSeguimientoV2, nil)) {
		t.Fatal("la consulta histórica quedó fuera de la superficie privada")
	}
}
