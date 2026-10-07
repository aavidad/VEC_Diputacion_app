package cotejopromocionjson

import (
	"context"
	"testing"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

func TestLectorDevuelveCopiasEntreConsultas(t *testing.T) {
	d := ports.DictamenPromocionSintetico{Comprobaciones: []ports.ComprobacionPromocionSintetica{{HechosReferencias: []string{"hecho"}}}}
	l := Lector{Dictamen: &d}
	a, err := l.ConsultarCotejoPromocionSintetico(context.Background(), ports.ConsultaPromocionSintetica{})
	if err != nil {
		t.Fatal(err)
	}
	a.Comprobaciones[0].HechosReferencias[0] = "otra"
	a.Comprobaciones[0].EstadoAportado = "otro"
	b, err := l.ConsultarCotejoPromocionSintetico(context.Background(), ports.ConsultaPromocionSintetica{})
	if err != nil || b.Comprobaciones[0].HechosReferencias[0] != "hecho" || b.Comprobaciones[0].EstadoAportado != "" {
		t.Fatal("devuelve datos mutables del proveedor")
	}
}
