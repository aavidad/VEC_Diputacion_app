package observabilidad

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestColaConservaCorrelacionSinLeerAleatoriedad(t *testing.T) {
	destino := &destinoSeguro{}
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Aleatorio: aleatorioFallido{}})
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	otra, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	e.EmitirConContexto(ctx, solicitudValida())
	e.EmitirConContexto(ctx, solicitudValida())
	e.EmitirConContexto(otra, solicitudValida())
	cerrar(t, e)
	lineas := destino.lineas(t)
	a, _ := ports.CorrelacionIncidenciasPeticion(ctx)
	b, _ := ports.CorrelacionIncidenciasPeticion(otra)
	if len(lineas) != 3 {
		t.Fatalf("líneas = %d", len(lineas))
	}
	if lineas[0]["correlacion"] != a || lineas[1]["correlacion"] != a || lineas[2]["correlacion"] != b {
		t.Fatal("worker regeneró correlaciones")
	}
	if m := e.MetricasEmision(); m.FallosCorrelacion != 0 || m.Escritas != 3 {
		t.Fatalf("métricas: %+v", m)
	}
	e.EmitirConContexto(ctx, solicitudValida())
	if m := e.MetricasEmision(); m.Descartadas != 1 {
		t.Fatalf("aceptó después de cierre: %+v", m)
	}
}
